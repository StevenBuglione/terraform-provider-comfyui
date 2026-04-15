package resources

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/inventory"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/nodeschema"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/validation"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func ValidateDynamicInputs(ctx context.Context, c *client.Client, nodeType string, model any, diags *diag.Diagnostics) {
	schema, ok := nodeschema.LookupGeneratedNodeSchema(nodeType)
	if !ok {
		return
	}

	fields := modelFieldsByTag(model)
	service := inventory.NewService(c)
	validateInputGroup(ctx, c, service, fields, schema.RequiredInputs, diags)
	validateInputGroup(ctx, c, service, fields, schema.OptionalInputs, diags)
}

func ValidateDynamicInputsForTest(ctx context.Context, c *client.Client, nodeType string, model any) diag.Diagnostics {
	var diags diag.Diagnostics
	ValidateDynamicInputs(ctx, c, nodeType, model, &diags)
	return diags
}

func validateInputGroup(ctx context.Context, c *client.Client, service *inventory.Service, fields map[string]reflect.Value, inputs []nodeschema.GeneratedNodeSchemaInput, diags *diag.Diagnostics) {
	mode := unsupportedDynamicValidationMode(c)
	for _, input := range inputs {
		tfName := generatedInputTerraformName(input)
		field, ok := fields[tfName]
		if !ok {
			continue
		}

		value, ok := attrValueFromField(field)
		if !ok {
			continue
		}

		validateGeneratedInput(ctx, service, mode, input, value, path.Root(tfName), diags)
	}
}

func unsupportedDynamicValidationMode(c *client.Client) string {
	if c == nil || strings.TrimSpace(c.UnsupportedDynamicValidationMode) == "" {
		return "error"
	}
	return strings.ToLower(strings.TrimSpace(c.UnsupportedDynamicValidationMode))
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func validateGeneratedInput(ctx context.Context, service *inventory.Service, mode string, input nodeschema.GeneratedNodeSchemaInput, value attr.Value, attributePath path.Path, diags *diag.Diagnostics) {
	if len(input.DynamicComboOptions) > 0 && validateDynamicComboValue(ctx, service, mode, input, value, attributePath, diags) {
		return
	}

	validateInputByKind(ctx, service, mode, input, value, attributePath, diags)
}

func validateDynamicComboValue(ctx context.Context, service *inventory.Service, mode string, input nodeschema.GeneratedNodeSchemaInput, value attr.Value, attributePath path.Path, diags *diag.Diagnostics) bool {
	switch typed := value.(type) {
	case basetypes.ObjectValue:
		if typed.IsNull() || typed.IsUnknown() {
			return true
		}
		validateDynamicComboObject(ctx, service, mode, input, typed, attributePath, diags)
		return true
	case basetypes.ListValue:
		if typed.IsNull() || typed.IsUnknown() {
			return true
		}
		for idx, elem := range typed.Elements() {
			elemPath := attributePath.AtListIndex(idx)
			if !validateDynamicComboValue(ctx, service, mode, input, elem, elemPath, diags) {
				validateInputByKind(ctx, service, mode, input, elem, elemPath, diags)
			}
		}
		return true
	default:
		return false
	}
}

func validateDynamicComboObject(ctx context.Context, service *inventory.Service, mode string, input nodeschema.GeneratedNodeSchemaInput, value basetypes.ObjectValue, attributePath path.Path, diags *diag.Diagnostics) {
	attributes := value.Attributes()
	selectionValue, ok := attributes["selection"]
	if !ok {
		return
	}

	selection, shouldValidate, known, ok := stringValueFromAttr(selectionValue)
	if !ok || !shouldValidate || !known {
		return
	}

	selectedOption, found := dynamicComboOption(input.DynamicComboOptions, selection)
	if !found {
		diags.AddAttributeError(
			attributePath.AtName("selection"),
			"Invalid DynamicCombo Selection",
			fmt.Sprintf("Selection %q is not valid for DynamicCombo input %q. Valid options: %s.", selection, input.Name, strings.Join(dynamicComboOptionKeys(input.DynamicComboOptions), ", ")),
		)
		return
	}

	allowedFields := dynamicComboAllowedFieldNames(selectedOption.Inputs)
	for fieldName, fieldValue := range attributes {
		if fieldName == dynamicComboSelectionKey || allowedFields[fieldName] || attrValueIsNull(fieldValue) {
			continue
		}

		diags.AddAttributeError(
			attributePath.AtName(fieldName),
			"Invalid DynamicCombo Field",
			fmt.Sprintf(
				"DynamicCombo selection %q for input %q does not accept child field %q. Valid child fields for this selection: %s.",
				selection,
				input.Name,
				fieldName,
				strings.Join(dynamicComboAllowedFieldList(selectedOption.Inputs), ", "),
			),
		)
	}

	for _, childInput := range selectedOption.Inputs {
		childName := generatedInputTerraformName(childInput)
		childPath := attributePath.AtName(childName)
		childValue, exists := attributes[childName]

		if !exists || attrValueIsNull(childValue) {
			if childInput.Required {
				diags.AddAttributeError(
					childPath,
					"Missing DynamicCombo Field",
					fmt.Sprintf("DynamicCombo selection %q for input %q requires child field %q.", selection, input.Name, childInput.Name),
				)
			}
			continue
		}

		validateGeneratedInput(ctx, service, mode, childInput, childValue, childPath, diags)
	}
}

const dynamicComboSelectionKey = "selection"

func validateInputByKind(ctx context.Context, service *inventory.Service, mode string, input nodeschema.GeneratedNodeSchemaInput, value attr.Value, attributePath path.Path, diags *diag.Diagnostics) {
	switch input.ValidationKind {
	case validation.InputValidationKindDynamicExpression:
		addUnsupportedDynamicValidationDiagnostic(mode, input, attributePath, diags)
	case validation.InputValidationKindDynamicInventory:
		validateDynamicInventoryInput(ctx, service, input, value, attributePath, diags)
	}
}

func addUnsupportedDynamicValidationDiagnostic(mode string, input nodeschema.GeneratedNodeSchemaInput, attributePath path.Path, diags *diag.Diagnostics) {
	switch mode {
	case "ignore":
		return
	case "warning":
		diags.AddAttributeWarning(
			attributePath,
			"Unsupported Dynamic Plan Validation",
			fmt.Sprintf("Input %q on node %q uses dynamic ComfyUI options that this provider cannot strictly validate during terraform plan. Execution may still succeed at runtime.", input.Name, input.Type),
		)
	default:
		diags.AddAttributeError(
			attributePath,
			"Unsupported Dynamic Plan Validation",
			fmt.Sprintf("Input %q on node %q uses dynamic ComfyUI options that this provider cannot strictly validate during terraform plan.", input.Name, input.Type),
		)
	}
}

func validateDynamicInventoryInput(ctx context.Context, service *inventory.Service, input nodeschema.GeneratedNodeSchemaInput, value attr.Value, attributePath path.Path, diags *diag.Diagnostics) {
	stringValue, shouldValidate, known, ok := stringValueFromAttr(value)
	if !ok || !shouldValidate {
		return
	}
	if !known {
		diags.AddAttributeError(
			attributePath,
			"Unknown Dynamic Inventory Value",
			fmt.Sprintf("Input %q must be known during terraform plan so the provider can validate it against the live ComfyUI %s inventory.", input.Name, input.InventoryKind),
		)
		return
	}

	kind, kindOK := inventory.ParseKind(input.InventoryKind)
	if !kindOK {
		diags.AddAttributeError(
			attributePath,
			"Unsupported Inventory Kind",
			fmt.Sprintf("Input %q uses unsupported inventory kind %q.", input.Name, input.InventoryKind),
		)
		return
	}

	validValues, err := service.GetInventory(ctx, kind)
	if err != nil {
		diags.AddAttributeError(
			attributePath,
			"Dynamic Inventory Validation Failed",
			err.Error(),
		)
		return
	}

	if !containsString(validValues, stringValue) {
		diags.AddAttributeError(
			attributePath,
			"Invalid Dynamic Inventory Value",
			fmt.Sprintf("Value %q is not available in the live ComfyUI %s inventory for input %q.", stringValue, input.InventoryKind, input.Name),
		)
	}
}

func dynamicComboOption(options []nodeschema.GeneratedDynamicComboOption, selection string) (nodeschema.GeneratedDynamicComboOption, bool) {
	for _, option := range options {
		if option.Key == selection {
			return option, true
		}
	}
	return nodeschema.GeneratedDynamicComboOption{}, false
}

func dynamicComboOptionKeys(options []nodeschema.GeneratedDynamicComboOption) []string {
	keys := make([]string, 0, len(options))
	for _, option := range options {
		keys = append(keys, option.Key)
	}
	slices.Sort(keys)
	return keys
}

func dynamicComboAllowedFieldNames(inputs []nodeschema.GeneratedNodeSchemaInput) map[string]bool {
	allowed := make(map[string]bool, len(inputs)+1)
	allowed[dynamicComboSelectionKey] = true
	for _, input := range inputs {
		allowed[generatedInputTerraformName(input)] = true
	}
	return allowed
}

func dynamicComboAllowedFieldList(inputs []nodeschema.GeneratedNodeSchemaInput) []string {
	fields := make([]string, 0, len(inputs)+1)
	fields = append(fields, dynamicComboSelectionKey)
	for _, input := range inputs {
		fields = append(fields, generatedInputTerraformName(input))
	}
	slices.Sort(fields)
	return fields
}

func attrValueFromField(field reflect.Value) (attr.Value, bool) {
	if !field.IsValid() {
		return nil, false
	}
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return nil, false
		}
		field = field.Elem()
	}
	value, ok := field.Interface().(attr.Value)
	return value, ok
}

func stringValueFromAttr(value attr.Value) (string, bool, bool, bool) {
	stringValue, ok := value.(basetypes.StringValue)
	if !ok {
		return "", false, false, false
	}
	if stringValue.IsNull() {
		return "", false, true, true
	}
	if stringValue.IsUnknown() {
		return "", true, false, true
	}
	return stringValue.ValueString(), true, true, true
}

func attrValueIsNull(value attr.Value) bool {
	return value == nil || value.IsNull()
}

func modelFieldsByTag(model any) map[string]reflect.Value {
	result := map[string]reflect.Value{}
	value := reflect.ValueOf(model)
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return result
	}

	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("tfsdk")
		if tag == "" {
			continue
		}
		result[tag] = value.Field(i)
	}
	return result
}

func generatedInputTerraformName(input nodeschema.GeneratedNodeSchemaInput) string {
	return sanitizeGeneratedName(input.Name)
}

var generatedNonAlphanumRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func sanitizeGeneratedName(name string) string {
	s := generatedNonAlphanumRe.ReplaceAllString(name, "_")
	s = strings.Trim(s, "_")
	s = strings.ToLower(s)
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		s = "_" + s
	}
	if s == "" {
		s = "unnamed"
	}
	if generatedTerraformReservedRootNames[s] {
		s += "_value"
	}
	return s
}

var generatedTerraformReservedRootNames = map[string]bool{
	"count":       true,
	"for_each":    true,
	"depends_on":  true,
	"lifecycle":   true,
	"provider":    true,
	"provisioner": true,
	"connection":  true,
}
