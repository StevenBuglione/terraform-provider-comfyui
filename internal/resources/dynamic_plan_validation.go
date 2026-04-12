package resources

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/inventory"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/nodeschema"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/validation"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ValidateDynamicInputs(ctx context.Context, c *client.Client, nodeType string, model any, diags *diag.Diagnostics) {
	schema, ok := nodeschema.LookupGeneratedNodeSchema(nodeType)
	if !ok {
		return
	}

	fields := modelFieldsByTag(model)
	service := inventory.NewService(c)
	validateInputGroup(ctx, service, fields, schema.RequiredInputs, diags)
	validateInputGroup(ctx, service, fields, schema.OptionalInputs, diags)
}

func ValidateDynamicInputsForTest(ctx context.Context, c *client.Client, nodeType string, model any) diag.Diagnostics {
	var diags diag.Diagnostics
	ValidateDynamicInputs(ctx, c, nodeType, model, &diags)
	return diags
}

func validateInputGroup(ctx context.Context, service *inventory.Service, fields map[string]reflect.Value, inputs []nodeschema.GeneratedNodeSchemaInput, diags *diag.Diagnostics) {
	for _, input := range inputs {
		tfName := generatedInputTerraformName(input)
		field, ok := fields[tfName]
		if !ok {
			continue
		}

		value, shouldValidate, known, ok := stringValueFromField(field)
		if !ok || !shouldValidate {
			continue
		}

		switch input.ValidationKind {
		case validation.InputValidationKindDynamicExpression:
			diags.AddAttributeError(
				path.Root(tfName),
				"Unsupported Dynamic Plan Validation",
				fmt.Sprintf("Input %q on node %q uses dynamic ComfyUI options that this provider cannot strictly validate during terraform plan.", input.Name, input.Type),
			)
		case validation.InputValidationKindDynamicInventory:
			if !known {
				diags.AddAttributeError(
					path.Root(tfName),
					"Unknown Dynamic Inventory Value",
					fmt.Sprintf("Input %q must be known during terraform plan so the provider can validate it against the live ComfyUI %s inventory.", input.Name, input.InventoryKind),
				)
				continue
			}
			kind, kindOK := inventory.ParseKind(input.InventoryKind)
			if !kindOK {
				diags.AddAttributeError(
					path.Root(tfName),
					"Unsupported Inventory Kind",
					fmt.Sprintf("Input %q uses unsupported inventory kind %q.", input.Name, input.InventoryKind),
				)
				continue
			}
			validValues, err := service.GetInventory(ctx, kind)
			if err != nil {
				diags.AddAttributeError(
					path.Root(tfName),
					"Dynamic Inventory Validation Failed",
					err.Error(),
				)
				continue
			}
			if !containsString(validValues, value) {
				diags.AddAttributeError(
					path.Root(tfName),
					"Invalid Dynamic Inventory Value",
					fmt.Sprintf("Value %q is not available in the live ComfyUI %s inventory for input %q.", value, input.InventoryKind, input.Name),
				)
			}
		}
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringValueFromField(field reflect.Value) (string, bool, bool, bool) {
	if !field.IsValid() {
		return "", false, false, false
	}
	if field.Type() != reflect.TypeOf(types.String{}) {
		return "", false, false, false
	}
	value := field.Interface().(types.String)
	if value.IsNull() {
		return "", false, true, true
	}
	if value.IsUnknown() {
		return "", true, false, true
	}
	return value.ValueString(), true, true, true
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
