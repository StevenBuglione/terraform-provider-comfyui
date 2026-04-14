package resources

import (
	"context"
	"encoding/json"
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
	node, err := nodeStateFromModel(id, classType, model)
	if err != nil {
		return err
	}

	RegisterNodeState(node)
	return nil
}

func RegisterNodeStateAndDefinitionFromModel(id, classType string, model any) (string, error) {
	node, err := nodeStateFromModel(id, classType, model)
	if err != nil {
		return "", err
	}

	RegisterNodeState(node)
	return marshalNodeDefinitionJSON(node)
}

func NodeDefinitionJSONFromModel(id, classType string, model any) (string, error) {
	node, err := nodeStateFromModel(id, classType, model)
	if err != nil {
		return "", err
	}

	return marshalNodeDefinitionJSON(node)
}

func ParseNodeDefinitionJSON(raw string) (NodeState, error) {
	if strings.TrimSpace(raw) == "" {
		return NodeState{}, fmt.Errorf("node definition JSON cannot be empty")
	}

	var node NodeState
	if err := json.Unmarshal([]byte(raw), &node); err != nil {
		return NodeState{}, err
	}
	if node.ID == "" {
		return NodeState{}, fmt.Errorf("node definition JSON must include id")
	}
	if node.ClassType == "" {
		return NodeState{}, fmt.Errorf("node definition JSON must include class_type")
	}
	if node.Inputs == nil {
		node.Inputs = make(map[string]interface{})
	}

	return node, nil
}

func nodeStateFromModel(id, classType string, model any) (NodeState, error) {
	if id == "" {
		return NodeState{}, fmt.Errorf("cannot register node state with empty id")
	}
	if classType == "" {
		return NodeState{}, fmt.Errorf("cannot register node state with empty class type")
	}

	inputs, err := extractInputsFromModel(model)
	if err != nil {
		return NodeState{}, err
	}

	return NodeState{
		ID:        id,
		ClassType: classType,
		Inputs:    inputs,
	}, nil
}

func marshalNodeDefinitionJSON(node NodeState) (string, error) {
	raw, err := json.Marshal(node)
	if err != nil {
		return "", fmt.Errorf("failed to marshal node definition JSON: %w", err)
	}
	return string(raw), nil
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
	return AssembleWorkflowFromNodeIDsWithDefinitions(ids, nil)
}

func AssembleWorkflowFromNodeIDsWithDefinitions(ids []string, definitionJSONs []string) (*AssembledWorkflow, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("cannot assemble workflow: no node_ids provided")
	}
	if len(definitionJSONs) > 0 && len(definitionJSONs) != len(ids) {
		return nil, fmt.Errorf("node_definition_jsons must have the same length as node_ids")
	}

	nodes := make([]NodeState, 0, len(ids))
	for i, id := range ids {
		var fallbackNode NodeState
		hasFallbackNode := false
		if len(definitionJSONs) > 0 {
			var err error
			fallbackNode, err = ParseNodeDefinitionJSON(definitionJSONs[i])
			if err != nil {
				return nil, fmt.Errorf("node_definition_jsons[%d] for node %q must be valid JSON: %w", i, id, err)
			}
			if fallbackNode.ID != id {
				return nil, fmt.Errorf("node_definition_jsons[%d] id %q does not match node_ids[%d] %q", i, fallbackNode.ID, i, id)
			}
			hasFallbackNode = true
		}

		node, ok := LookupNodeState(id)
		if ok {
			nodes = append(nodes, node)
			continue
		}
		if !hasFallbackNode {
			return nil, fmt.Errorf("node %q is not registered; node resources must be created before comfyui_workflow", id)
		}

		nodes = append(nodes, fallbackNode)
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
		if attrName == "" || attrName == "id" || attrName == "node_id" || attrName == "node_definition_json" || strings.HasSuffix(attrName, "_output") {
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
