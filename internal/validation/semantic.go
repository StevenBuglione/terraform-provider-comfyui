package validation

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/artifacts"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

type Options struct {
	RequireOutputNode bool
}

func ValidatePrompt(prompt *artifacts.Prompt, nodeInfo map[string]client.NodeInfo, opts Options) Report {
	if prompt == nil {
		report := NewReport(0)
		report.AddError("prompt is nil")
		return report
	}

	report := NewReport(len(prompt.Nodes))
	if len(prompt.Nodes) == 0 {
		report.AddError("prompt must contain at least one node")
		return report
	}

	hasOutputNode := false
	nodeIDs := sortedNodeIDs(prompt.Nodes)
	for _, nodeID := range nodeIDs {
		node := prompt.Nodes[nodeID]
		info, ok := nodeInfo[node.ClassType]
		if !ok {
			report.AddError(fmt.Sprintf(`node %q uses unknown class_type %q`, nodeID, node.ClassType))
			continue
		}
		if info.OutputNode {
			hasOutputNode = true
		}

		allowedInputs := allowedInputNames(info)
		for inputName := range node.Inputs {
			if !allowedInputs[inputName] {
				report.AddError(fmt.Sprintf(`node %q (%s) uses unexpected input %q`, nodeID, node.ClassType, inputName))
			}
		}

		for _, requiredInput := range requiredInputNames(info) {
			if _, ok := node.Inputs[requiredInput]; !ok {
				report.AddError(fmt.Sprintf(`node %q (%s) is missing required input %q`, nodeID, node.ClassType, requiredInput))
			}
		}

		for inputName, value := range node.Inputs {
			sourceNodeID, sourceSlot, linked := promptLinkValue(value)
			if !linked {
				continue
			}

			sourceNode, ok := prompt.Nodes[sourceNodeID]
			if !ok {
				report.AddError(fmt.Sprintf(`node %q (%s) input %q references missing source node %q`, nodeID, node.ClassType, inputName, sourceNodeID))
				continue
			}

			sourceInfo, ok := nodeInfo[sourceNode.ClassType]
			if !ok {
				report.AddError(fmt.Sprintf(`node %q (%s) input %q references source node %q with unknown class_type %q`, nodeID, node.ClassType, inputName, sourceNodeID, sourceNode.ClassType))
				continue
			}

			if sourceSlot < 0 || sourceSlot >= len(sourceInfo.Output) {
				report.AddError(fmt.Sprintf(`node %q (%s) input %q references output slot %d on node %q (%s), but only %d outputs exist`, nodeID, node.ClassType, inputName, sourceSlot, sourceNodeID, sourceNode.ClassType, len(sourceInfo.Output)))
				continue
			}

			targetTypes := inputTypes(info, inputName)
			if len(targetTypes) == 0 {
				continue
			}

			sourceType := sourceInfo.Output[sourceSlot]
			if !typesCompatible(sourceType, targetTypes) {
				report.AddError(fmt.Sprintf(`node %q (%s) input %q expects type %q but linked output is %q`, nodeID, node.ClassType, inputName, targetTypes[0], sourceType))
			}
		}
	}

	if opts.RequireOutputNode && !hasOutputNode {
		report.AddError("prompt does not include any node marked output_node=true")
	}
	return report
}

func sortedNodeIDs(nodes map[string]artifacts.PromptNode) []string {
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		left, leftErr := strconv.Atoi(ids[i])
		right, rightErr := strconv.Atoi(ids[j])
		if leftErr == nil && rightErr == nil {
			return left < right
		}
		return ids[i] < ids[j]
	})
	return ids
}

func allowedInputNames(info client.NodeInfo) map[string]bool {
	allowed := map[string]bool{}
	for _, section := range []map[string]interface{}{info.Input.Required, info.Input.Optional, info.Input.Hidden} {
		for name := range section {
			allowed[name] = true
		}
	}
	return allowed
}

func requiredInputNames(info client.NodeInfo) []string {
	required := make([]string, 0, len(info.Input.Required))
	for name := range info.Input.Required {
		if _, hidden := info.Input.Hidden[name]; hidden || isServerInjectedHiddenInput(name) {
			continue
		}
		required = append(required, name)
	}
	sort.Strings(required)
	return required
}

func isServerInjectedHiddenInput(name string) bool {
	return slices.Contains([]string{"prompt", "extra_pnginfo", "unique_id"}, name)
}

func promptLinkValue(value interface{}) (string, int, bool) {
	values, ok := value.([]interface{})
	if !ok || len(values) != 2 {
		return "", 0, false
	}

	nodeID, ok := values[0].(string)
	if !ok || nodeID == "" {
		return "", 0, false
	}

	slot, ok := interfaceToInt(values[1])
	if !ok {
		return "", 0, false
	}

	return nodeID, slot, true
}

func interfaceToInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		if v != float64(int(v)) {
			return 0, false
		}
		return int(v), true
	case json.Number:
		parsed, err := strconv.Atoi(v.String())
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func inputTypes(info client.NodeInfo, inputName string) []string {
	for _, section := range []map[string]interface{}{info.Input.Required, info.Input.Optional} {
		raw, ok := section[inputName]
		if !ok {
			continue
		}
		if values := definitionTypes(raw); len(values) > 0 {
			return values
		}
	}
	return nil
}

func definitionTypes(raw interface{}) []string {
	switch value := raw.(type) {
	case string:
		return []string{value}
	case []string:
		return append([]string(nil), value...)
	case []interface{}:
		if len(value) == 0 {
			return nil
		}
		switch first := value[0].(type) {
		case string:
			return []string{first}
		case []string:
			return append([]string(nil), first...)
		case []interface{}:
			types := make([]string, 0, len(first))
			for _, item := range first {
				if typeName, ok := item.(string); ok {
					types = append(types, typeName)
				}
			}
			return types
		}
	}
	return nil
}

func typesCompatible(sourceType string, targetTypes []string) bool {
	if sourceType == "*" {
		return true
	}
	for _, targetType := range targetTypes {
		if targetType == "*" || targetType == sourceType {
			return true
		}
	}
	return false
}
