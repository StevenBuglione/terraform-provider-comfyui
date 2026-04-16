package resources

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/nodeschema"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var nodeStateRegistry = struct {
	mu    sync.RWMutex
	nodes map[string]NodeState
}{
	nodes: make(map[string]NodeState),
}

func RegisterNodeState(node NodeState) {
	nodeStateRegistry.mu.Lock()
	defer nodeStateRegistry.mu.Unlock()
	nodeStateRegistry.nodes[node.ID] = node
}

func RegisterNodeStateFromModel(id, classType string, model any) error {
	if id == "" {
		return fmt.Errorf("cannot register node state with empty id")
	}
	if classType == "" {
		return fmt.Errorf("cannot register node state with empty class type")
	}

	inputs, err := extractInputsFromModel(classType, model)
	if err != nil {
		return err
	}

	RegisterNodeState(NodeState{
		ID:        id,
		ClassType: classType,
		Inputs:    inputs,
	})

	return nil
}

func LookupNodeState(id string) (NodeState, bool) {
	nodeStateRegistry.mu.RLock()
	defer nodeStateRegistry.mu.RUnlock()

	node, ok := nodeStateRegistry.nodes[id]
	return node, ok
}

func DeleteNodeState(id string) {
	nodeStateRegistry.mu.Lock()
	defer nodeStateRegistry.mu.Unlock()
	delete(nodeStateRegistry.nodes, id)
}

func resetNodeStateRegistry() {
	nodeStateRegistry.mu.Lock()
	defer nodeStateRegistry.mu.Unlock()
	nodeStateRegistry.nodes = make(map[string]NodeState)
}

func AssembleWorkflowFromNodeIDs(ids []string) (*AssembledWorkflow, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("cannot assemble workflow: no node_ids provided")
	}

	nodes := make([]NodeState, 0, len(ids))
	for _, id := range ids {
		node, ok := LookupNodeState(id)
		if !ok {
			return nil, fmt.Errorf("node %q is not registered; node resources must be created before comfyui_workflow", id)
		}
		nodes = append(nodes, node)
	}

	if errs := ValidateWorkflow(nodes); len(errs) > 0 {
		msgs := make([]string, 0, len(errs))
		for _, err := range errs {
			msgs = append(msgs, err.Error())
		}
		return nil, fmt.Errorf("workflow validation failed: %s", strings.Join(msgs, "; "))
	}

	return AssembleWorkflow(nodes)
}

func extractInputsFromModel(classType string, model any) (map[string]interface{}, error) {
	value := reflect.ValueOf(model)
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct model, got %T", model)
	}

	modelType := value.Type()
	inputs := make(map[string]interface{})

	for i := 0; i < value.NumField(); i++ {
		fieldType := modelType.Field(i)
		attrName := fieldType.Tag.Get("tfsdk")
		if attrName == "" || attrName == "id" || attrName == "node_id" || strings.HasSuffix(attrName, "_output") {
			continue
		}

		converted, ok, err := terraformValueToNative(value.Field(i).Interface())
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", attrName, err)
		}
		if ok {
			if m, isMap := converted.(map[string]interface{}); isMap {
				if parentInput, isDC := lookupDynamicComboInput(classType, attrName); isDC {
					flattenDynamicComboInto(attrName, m, parentInput, inputs)
					continue
				}
			}
			inputs[attrName] = converted
		}
	}

	return inputs, nil
}

func terraformValueToNative(value any) (interface{}, bool, error) {
	switch v := value.(type) {
	case basetypes.StringValue:
		if v.IsNull() || v.IsUnknown() {
			return nil, false, nil
		}
		return v.ValueString(), true, nil
	case basetypes.Int64Value:
		if v.IsNull() || v.IsUnknown() {
			return nil, false, nil
		}
		return v.ValueInt64(), true, nil
	case basetypes.Float64Value:
		if v.IsNull() || v.IsUnknown() {
			return nil, false, nil
		}
		return v.ValueFloat64(), true, nil
	case basetypes.BoolValue:
		if v.IsNull() || v.IsUnknown() {
			return nil, false, nil
		}
		return v.ValueBool(), true, nil
	case basetypes.ListValue:
		if v.IsNull() || v.IsUnknown() {
			return nil, false, nil
		}
		out := make([]interface{}, 0, len(v.Elements()))
		for _, elem := range v.Elements() {
			converted, ok, err := attrValueToNative(elem)
			if err != nil {
				return nil, false, err
			}
			if ok {
				out = append(out, converted)
			}
		}
		return out, true, nil
	case basetypes.MapValue:
		if v.IsNull() || v.IsUnknown() {
			return nil, false, nil
		}
		out := make(map[string]interface{}, len(v.Elements()))
		for key, elem := range v.Elements() {
			converted, ok, err := attrValueToNative(elem)
			if err != nil {
				return nil, false, err
			}
			if ok {
				out[key] = converted
			}
		}
		return out, true, nil
	case basetypes.ObjectValue:
		if v.IsNull() || v.IsUnknown() {
			return nil, false, nil
		}
		out := make(map[string]interface{}, len(v.Attributes()))
		for key, elem := range v.Attributes() {
			converted, ok, err := attrValueToNative(elem)
			if err != nil {
				return nil, false, err
			}
			if ok {
				out[key] = converted
			}
		}
		return out, true, nil
	default:
		return value, true, nil
	}
}

func attrValueToNative(value attr.Value) (interface{}, bool, error) {
	switch v := value.(type) {
	case basetypes.StringValue:
		return terraformValueToNative(v)
	case basetypes.Int64Value:
		return terraformValueToNative(v)
	case basetypes.Float64Value:
		return terraformValueToNative(v)
	case basetypes.BoolValue:
		return terraformValueToNative(v)
	case basetypes.ListValue:
		return terraformValueToNative(v)
	case basetypes.MapValue:
		return terraformValueToNative(v)
	case basetypes.ObjectValue:
		return terraformValueToNative(v)
	default:
		return nil, false, fmt.Errorf("unsupported attribute value type %T", value)
	}
}

func listValueToStrings(ctx context.Context, list basetypes.ListValue) ([]string, diag.Diagnostics) {
	var ids []string
	diags := list.ElementsAs(ctx, &ids, false)
	return ids, diags
}

// lookupDynamicComboInput returns the GeneratedNodeSchemaInput for the named input of
// classType if it is a COMFY_DYNAMICCOMBO_V3 input; returns false otherwise.
// inputName is compared against the Terraform-sanitized form of each schema input name
// so that raw names with spaces or punctuation match their sanitized attribute names.
func lookupDynamicComboInput(classType, inputName string) (nodeschema.GeneratedNodeSchemaInput, bool) {
	schema, ok := nodeschema.LookupGeneratedNodeSchema(classType)
	if !ok {
		return nodeschema.GeneratedNodeSchemaInput{}, false
	}
	for _, inp := range schema.RequiredInputs {
		if sanitizeGeneratedName(inp.Name) == inputName && inp.Type == "COMFY_DYNAMICCOMBO_V3" {
			return inp, true
		}
	}
	for _, inp := range schema.OptionalInputs {
		if sanitizeGeneratedName(inp.Name) == inputName && inp.Type == "COMFY_DYNAMICCOMBO_V3" {
			return inp, true
		}
	}
	return nodeschema.GeneratedNodeSchemaInput{}, false
}

// collectDynamicComboInputs returns a map of Terraform-sanitized input name → GeneratedNodeSchemaInput for all
// COMFY_DYNAMICCOMBO_V3 inputs of classType. Used to avoid repeated schema lookups in hot paths.
// Keys are the sanitized Terraform attribute names so they match dotted keys in node.Inputs.
func collectDynamicComboInputs(classType string) map[string]nodeschema.GeneratedNodeSchemaInput {
	schema, ok := nodeschema.LookupGeneratedNodeSchema(classType)
	if !ok {
		return nil
	}
	result := make(map[string]nodeschema.GeneratedNodeSchemaInput)
	for _, inp := range schema.RequiredInputs {
		if inp.Type == "COMFY_DYNAMICCOMBO_V3" {
			result[sanitizeGeneratedName(inp.Name)] = inp
		}
	}
	for _, inp := range schema.OptionalInputs {
		if inp.Type == "COMFY_DYNAMICCOMBO_V3" {
			result[sanitizeGeneratedName(inp.Name)] = inp
		}
	}
	return result
}

// findNestedDynamicComboInput searches the DynamicComboOptions of parent for a child input
// named childName whose type is COMFY_DYNAMICCOMBO_V3. Returns the child input and true if found.
// childName is compared against the Terraform-sanitized form of each child's raw schema name
// so that raw names with spaces or punctuation match their sanitized attribute names.
func findNestedDynamicComboInput(parent nodeschema.GeneratedNodeSchemaInput, childName string) (nodeschema.GeneratedNodeSchemaInput, bool) {
	for _, option := range parent.DynamicComboOptions {
		for _, inp := range option.Inputs {
			if sanitizeGeneratedName(inp.Name) == childName && inp.Type == "COMFY_DYNAMICCOMBO_V3" {
				return inp, true
			}
		}
	}
	return nodeschema.GeneratedNodeSchemaInput{}, false
}

// flattenDynamicComboInto expands a DynamicCombo map into target using dotted keys.
// The "selection" key becomes target[inputName]; all other keys become target[inputName.childKey].
// Nested DynamicCombo children (identified via parentInput's DynamicComboOptions) are
// recursively flattened so every leaf maps to a fully-dotted prompt key.
// Returns the list of all child keys (excluding the selection) that were added.
func flattenDynamicComboInto(inputName string, value map[string]interface{}, parentInput nodeschema.GeneratedNodeSchemaInput, target map[string]interface{}) []string {
	var childKeys []string
	if selection, ok := value["selection"]; ok {
		target[inputName] = selection
	}
	for k, v := range value {
		if k == "selection" {
			continue
		}
		dotKey := inputName + "." + k
		if childMap, isMap := v.(map[string]interface{}); isMap {
			if childInput, isDC := findNestedDynamicComboInput(parentInput, k); isDC {
				// Include the nested DC's own key (its selection is stored at dotKey)
				// so callers can track it as a DynamicCombo child.
				childKeys = append(childKeys, dotKey)
				nested := flattenDynamicComboInto(dotKey, childMap, childInput, target)
				childKeys = append(childKeys, nested...)
				continue
			}
		}
		target[dotKey] = v
		childKeys = append(childKeys, dotKey)
	}
	return childKeys
}
