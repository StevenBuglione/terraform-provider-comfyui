package artifacts

import (
	"encoding/json"
	"fmt"
)

type Workspace struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Revision    int                    `json:"revision,omitempty"`
	LastNodeID  int                    `json:"last_node_id,omitempty"`
	LastLinkID  int                    `json:"last_link_id,omitempty"`
	Nodes       []WorkspaceNode        `json:"nodes,omitempty"`
	Links       []WorkspaceLink        `json:"links,omitempty"`
	Groups      []WorkspaceGroup       `json:"groups,omitempty"`
	Definitions WorkspaceDefinitions   `json:"definitions,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
	Version     interface{}            `json:"version,omitempty"`
}

type WorkspaceNode struct {
	ID            int                    `json:"id"`
	Type          string                 `json:"type"`
	Pos           []float64              `json:"pos,omitempty"`
	Size          []float64              `json:"size,omitempty"`
	Inputs        []WorkspaceNodeInput   `json:"inputs,omitempty"`
	Outputs       []WorkspaceNodeOutput  `json:"outputs,omitempty"`
	Properties    map[string]interface{} `json:"properties,omitempty"`
	WidgetsValues []interface{}          `json:"widgets_values,omitempty"`
}

type WorkspaceNodeInput struct {
	Name   string               `json:"name"`
	Type   string               `json:"type,omitempty"`
	Widget *WorkspaceNodeWidget `json:"widget,omitempty"`
	Link   interface{}          `json:"link"`
}

type WorkspaceNodeOutput struct {
	Name  string `json:"name"`
	Type  string `json:"type,omitempty"`
	Links []int  `json:"links,omitempty"`
}

type WorkspaceNodeWidget struct {
	Name string `json:"name"`
}

type WorkspaceGroup struct {
	ID       int       `json:"id"`
	Title    string    `json:"title"`
	Bounding []float64 `json:"bounding,omitempty"`
	Color    string    `json:"color,omitempty"`
	FontSize int       `json:"font_size,omitempty"`
}

type WorkspaceDefinitions struct {
	Subgraphs []WorkspaceSubgraphDefinition `json:"subgraphs,omitempty"`
}

type WorkspaceSubgraphDefinition struct {
	ID     string           `json:"id"`
	Name   string           `json:"name,omitempty"`
	Nodes  []WorkspaceNode  `json:"nodes,omitempty"`
	Links  []WorkspaceLink  `json:"links,omitempty"`
	Groups []WorkspaceGroup `json:"groups,omitempty"`
}

type WorkspaceLink struct {
	ID         int    `json:"id"`
	OriginID   int    `json:"origin_id"`
	OriginSlot int    `json:"origin_slot"`
	TargetID   int    `json:"target_id"`
	TargetSlot int    `json:"target_slot"`
	Type       string `json:"type"`
}

func (l *WorkspaceLink) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	if data[0] == '[' {
		var values []interface{}
		if err := json.Unmarshal(data, &values); err != nil {
			return err
		}
		if len(values) < 6 {
			return fmt.Errorf("workspace link array must have at least 6 items")
		}
		l.ID = interfaceToInt(values[0])
		l.OriginID = interfaceToInt(values[1])
		l.OriginSlot = interfaceToInt(values[2])
		l.TargetID = interfaceToInt(values[3])
		l.TargetSlot = interfaceToInt(values[4])
		if s, ok := values[5].(string); ok {
			l.Type = s
		}
		return nil
	}

	type alias WorkspaceLink
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*l = WorkspaceLink(decoded)
	return nil
}

func (l WorkspaceLink) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{l.ID, l.OriginID, l.OriginSlot, l.TargetID, l.TargetSlot, l.Type})
}

func (w Workspace) MarshalJSON() ([]byte, error) {
	type workspaceJSON struct {
		ID          string                 `json:"id,omitempty"`
		Name        string                 `json:"name,omitempty"`
		Revision    int                    `json:"revision,omitempty"`
		LastNodeID  int                    `json:"last_node_id,omitempty"`
		LastLinkID  int                    `json:"last_link_id,omitempty"`
		Nodes       []WorkspaceNode        `json:"nodes,omitempty"`
		Links       []WorkspaceLink        `json:"links,omitempty"`
		Groups      []WorkspaceGroup       `json:"groups,omitempty"`
		Definitions *WorkspaceDefinitions  `json:"definitions,omitempty"`
		Extra       map[string]interface{} `json:"extra,omitempty"`
		Version     interface{}            `json:"version,omitempty"`
	}

	payload := workspaceJSON{
		ID:         w.ID,
		Name:       w.Name,
		Revision:   w.Revision,
		LastNodeID: w.LastNodeID,
		LastLinkID: w.LastLinkID,
		Nodes:      w.Nodes,
		Links:      w.Links,
		Groups:     w.Groups,
		Extra:      w.Extra,
		Version:    w.Version,
	}
	if len(w.Definitions.Subgraphs) > 0 {
		payload.Definitions = &w.Definitions
	}

	return json.Marshal(payload)
}

func ParseWorkspaceJSON(raw string) (*Workspace, error) {
	var workspace Workspace
	if err := json.Unmarshal([]byte(raw), &workspace); err != nil {
		return nil, fmt.Errorf("parse workspace JSON: %w", err)
	}

	if workspace.Nodes == nil {
		workspace.Nodes = []WorkspaceNode{}
	}
	if workspace.Links == nil {
		workspace.Links = []WorkspaceLink{}
	}
	if workspace.Groups == nil {
		workspace.Groups = []WorkspaceGroup{}
	}
	for i := range workspace.Definitions.Subgraphs {
		if workspace.Definitions.Subgraphs[i].Nodes == nil {
			workspace.Definitions.Subgraphs[i].Nodes = []WorkspaceNode{}
		}
		if workspace.Definitions.Subgraphs[i].Links == nil {
			workspace.Definitions.Subgraphs[i].Links = []WorkspaceLink{}
		}
		if workspace.Definitions.Subgraphs[i].Groups == nil {
			workspace.Definitions.Subgraphs[i].Groups = []WorkspaceGroup{}
		}
	}
	if workspace.Extra == nil {
		workspace.Extra = map[string]interface{}{}
	}

	return &workspace, nil
}

func (w *Workspace) JSON() (string, error) {
	if w == nil {
		return "", fmt.Errorf("workspace is nil")
	}

	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal workspace JSON: %w", err)
	}

	return string(data), nil
}

func interfaceToInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}
