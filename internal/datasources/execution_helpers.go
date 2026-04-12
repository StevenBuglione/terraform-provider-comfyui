package datasources

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dynamicFromInterface(value interface{}) (types.Dynamic, error) {
	if value == nil {
		return types.DynamicNull(), nil
	}

	attrValue, err := interfaceToAttrValue(value)
	if err != nil {
		return types.DynamicNull(), err
	}

	return types.DynamicValue(attrValue), nil
}

func jsonStringFromValue(value interface{}) (types.String, error) {
	if value == nil {
		return types.StringNull(), nil
	}

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return types.StringNull(), err
	}

	return types.StringValue(string(jsonBytes)), nil
}

func normalizeJSONValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var normalized interface{}
	if err := json.Unmarshal(jsonBytes, &normalized); err != nil {
		return nil, err
	}

	return normalized, nil
}

func mapStringAny(value interface{}) map[string]interface{} {
	if value == nil {
		return nil
	}

	mapped, _ := value.(map[string]interface{})
	return mapped
}

func stringFromAny(value interface{}) string {
	s, _ := value.(string)
	return s
}

func int64FromAny(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func int64ValueFromAny(value interface{}) types.Int64 {
	if n, ok := int64FromAny(value); ok {
		return types.Int64Value(n)
	}
	return types.Int64Null()
}

func extractExecutionMetadata(extraData map[string]interface{}) (types.Int64, types.String) {
	if extraData == nil {
		return types.Int64Null(), types.StringNull()
	}

	createTime := int64ValueFromAny(extraData["create_time"])
	workflowID := types.StringNull()
	if extraPNGInfo, ok := extraData["extra_pnginfo"].(map[string]interface{}); ok {
		if workflow, ok := extraPNGInfo["workflow"].(map[string]interface{}); ok {
			if workflowIDValue := stringFromAny(workflow["id"]); workflowIDValue != "" {
				workflowID = types.StringValue(workflowIDValue)
			}
		}
	}

	return createTime, workflowID
}

func extractExecutionEvents(messages [][]interface{}) (types.Int64, types.Int64, map[string]interface{}) {
	startTime := types.Int64Null()
	endTime := types.Int64Null()
	var executionError map[string]interface{}

	for _, message := range messages {
		if len(message) < 2 {
			continue
		}

		eventName, _ := message[0].(string)
		eventData, _ := message[1].(map[string]interface{})
		if eventData == nil {
			continue
		}

		switch eventName {
		case "execution_start":
			startTime = int64ValueFromAny(eventData["timestamp"])
		case "execution_success", "execution_error", "execution_interrupted":
			endTime = int64ValueFromAny(eventData["timestamp"])
			if eventName == "execution_error" {
				executionError = eventData
			}
		}
	}

	return startTime, endTime, executionError
}

func countHistoryOutputs(outputs interface{}) (int64, error) {
	if outputs == nil {
		return 0, nil
	}

	raw, err := json.Marshal(outputs)
	if err != nil {
		return 0, err
	}

	var decoded map[string]map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return 0, err
	}

	var count int64
	for _, nodeOutputs := range decoded {
		for mediaType, items := range nodeOutputs {
			if mediaType == "animated" {
				continue
			}
			list, ok := items.([]interface{})
			if !ok {
				continue
			}
			count += int64(len(list))
		}
	}

	return count, nil
}

func buildPromptTupleFields(promptTuple []interface{}) (types.String, types.Dynamic, types.String, types.Dynamic, types.String, types.Dynamic, types.Int64, types.String, error) {
	if len(promptTuple) < 4 {
		return types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.Int64Null(), types.StringNull(), fmt.Errorf("expected prompt tuple with at least 4 elements, got %d", len(promptTuple))
	}

	promptJSON, err := jsonStringFromValue(promptTuple[2])
	if err != nil {
		return types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.Int64Null(), types.StringNull(), fmt.Errorf("marshal prompt: %w", err)
	}
	promptValue, err := dynamicFromInterface(promptTuple[2])
	if err != nil {
		return types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.Int64Null(), types.StringNull(), fmt.Errorf("convert prompt: %w", err)
	}

	extraData := mapStringAny(promptTuple[3])
	extraDataJSON, err := jsonStringFromValue(extraData)
	if err != nil {
		return types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.Int64Null(), types.StringNull(), fmt.Errorf("marshal extra_data: %w", err)
	}
	extraDataValue, err := dynamicFromInterface(extraData)
	if err != nil {
		return types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.Int64Null(), types.StringNull(), fmt.Errorf("convert extra_data: %w", err)
	}

	outputsToExecuteJSON := types.StringNull()
	outputsToExecuteValue := types.DynamicNull()
	if len(promptTuple) >= 5 {
		outputsToExecuteJSON, err = jsonStringFromValue(promptTuple[4])
		if err != nil {
			return types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.Int64Null(), types.StringNull(), fmt.Errorf("marshal outputs_to_execute: %w", err)
		}
		outputsToExecuteValue, err = dynamicFromInterface(promptTuple[4])
		if err != nil {
			return types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.StringNull(), types.DynamicNull(), types.Int64Null(), types.StringNull(), fmt.Errorf("convert outputs_to_execute: %w", err)
		}
	}

	createTime, workflowID := extractExecutionMetadata(extraData)
	return promptJSON, promptValue, extraDataJSON, extraDataValue, outputsToExecuteJSON, outputsToExecuteValue, createTime, workflowID, nil
}
