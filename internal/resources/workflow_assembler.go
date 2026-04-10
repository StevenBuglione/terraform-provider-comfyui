package resources

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// connectionRefPattern matches a UUID:slot_index connection reference.
var connectionRefPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}:\d+$`)

// NodeState represents a single virtual node resource's state.
type NodeState struct {
	ID        string                 // Terraform resource UUID
	ClassType string                 // ComfyUI node class (e.g., "KSampler")
	Inputs    map[string]interface{} // Input values — strings, ints, floats, bools, or connection refs
}

// AssembledWorkflow is the output of assembly.
type AssembledWorkflow struct {
	Prompt  map[string]interface{} // ComfyUI API format prompt
	NodeMap map[string]string      // UUID → assigned numeric ID ("1", "2", ...)
	JSON    string                 // Serialized JSON string
}

// AssembleWorkflow takes a list of NodeState and produces ComfyUI API format JSON.
func AssembleWorkflow(nodes []NodeState) (*AssembledWorkflow, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("cannot assemble workflow: no nodes provided")
	}

	// Step 1: Assign sequential numeric IDs.
	nodeMap := make(map[string]string, len(nodes))
	for i, node := range nodes {
		if node.ID == "" {
			return nil, fmt.Errorf("node at index %d has empty ID", i)
		}
		if node.ClassType == "" {
			return nil, fmt.Errorf("node %q has empty ClassType", node.ID)
		}
		nodeMap[node.ID] = strconv.Itoa(i + 1)
	}

	// Step 2: Build the prompt map.
	prompt := make(map[string]interface{}, len(nodes))
	for _, node := range nodes {
		numericID := nodeMap[node.ID]

		processedInputs := make(map[string]interface{})
		for key, value := range node.Inputs {
			resolved, err := resolveInputValue(value, nodeMap)
			if err != nil {
				return nil, fmt.Errorf("node %q input %q: %w", node.ID, key, err)
			}
			if resolved == nil {
				continue // skip nil/empty
			}
			processedInputs[key] = resolved
		}

		prompt[numericID] = map[string]interface{}{
			"class_type": node.ClassType,
			"inputs":     processedInputs,
		}
	}

	// Step 3: Marshal to indented JSON.
	jsonBytes, err := json.MarshalIndent(prompt, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workflow JSON: %w", err)
	}

	return &AssembledWorkflow{
		Prompt:  prompt,
		NodeMap: nodeMap,
		JSON:    string(jsonBytes),
	}, nil
}

// isConnectionRef checks if a string value is a node output reference.
func isConnectionRef(value string) bool {
	return connectionRefPattern.MatchString(value)
}

// parseConnectionRef extracts UUID and slot index from a reference string.
func parseConnectionRef(value string) (uuid string, slotIndex int, err error) {
	if !isConnectionRef(value) {
		return "", 0, fmt.Errorf("invalid connection reference: %q", value)
	}
	lastColon := strings.LastIndex(value, ":")
	uuid = value[:lastColon]
	slotIndex, err = strconv.Atoi(value[lastColon+1:])
	if err != nil {
		return "", 0, fmt.Errorf("invalid slot index in reference %q: %w", value, err)
	}
	return uuid, slotIndex, nil
}

// resolveInputValue converts a raw input value, resolving connection refs to [id, slot] arrays.
// Returns nil if the value should be skipped (empty string or nil).
func resolveInputValue(value interface{}, nodeMap map[string]string) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case string:
		if v == "" {
			return nil, nil
		}
		if isConnectionRef(v) {
			uuid, slotIndex, err := parseConnectionRef(v)
			if err != nil {
				return nil, err
			}
			numericID, ok := nodeMap[uuid]
			if !ok {
				return nil, fmt.Errorf("connection reference to unknown node UUID %q", uuid)
			}
			return []interface{}{numericID, slotIndex}, nil
		}
		return v, nil

	case int, int64, float64, bool:
		return v, nil

	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i, nil
		}
		if f, err := v.Float64(); err == nil {
			return f, nil
		}
		return v.String(), nil

	default:
		return v, nil
	}
}
