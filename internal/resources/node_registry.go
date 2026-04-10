package resources

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

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

	inputs, err := extractInputsFromModel(model)
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

func extractInputsFromModel(model any) (map[string]interface{}, error) {
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
