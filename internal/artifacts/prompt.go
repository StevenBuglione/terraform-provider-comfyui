package artifacts

import (
	"encoding/json"
	"fmt"
)

type Prompt struct {
	Nodes map[string]PromptNode
}

type PromptNode struct {
	ClassType string                 `json:"class_type"`
	Inputs    map[string]interface{} `json:"inputs"`
	Meta      map[string]interface{} `json:"_meta,omitempty"`
}

func ParsePromptJSON(raw string) (*Prompt, error) {
	var nodes map[string]PromptNode
	if err := json.Unmarshal([]byte(raw), &nodes); err != nil {
		return nil, fmt.Errorf("parse prompt JSON: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("prompt JSON must contain at least one node")
	}

	for nodeID, node := range nodes {
		if node.ClassType == "" {
			return nil, fmt.Errorf("prompt node %q missing class_type", nodeID)
		}
		if node.Inputs == nil {
			node.Inputs = map[string]interface{}{}
		}
		if node.Meta == nil {
			node.Meta = map[string]interface{}{}
		}
		nodes[nodeID] = node
	}

	return &Prompt{Nodes: nodes}, nil
}

func (p *Prompt) JSON() (string, error) {
	if p == nil {
		return "", fmt.Errorf("prompt is nil")
	}

	data, err := json.MarshalIndent(p.Nodes, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal prompt JSON: %w", err)
	}

	return string(data), nil
}
