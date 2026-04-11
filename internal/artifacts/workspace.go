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
	Config      map[string]interface{} `json:"config,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
	Version     interface{}            `json:"version,omitempty"`
	rawFields   map[string]json.RawMessage
}

type WorkspaceNode struct {
	ID            int                    `json:"id"`
	Type          string                 `json:"type"`
	Pos           []float64              `json:"pos,omitempty"`
	Size          []float64              `json:"size,omitempty"`
	Flags         map[string]interface{} `json:"flags,omitempty"`
	Order         int                    `json:"order,omitempty"`
	Mode          int                    `json:"mode,omitempty"`
	Inputs        []WorkspaceNodeInput   `json:"inputs,omitempty"`
	Outputs       []WorkspaceNodeOutput  `json:"outputs,omitempty"`
	Properties    map[string]interface{} `json:"properties,omitempty"`
	WidgetsValues []interface{}          `json:"widgets_values,omitempty"`
	Title         string                 `json:"title,omitempty"`
	rawFields     map[string]json.RawMessage
}

type WorkspaceNodeInput struct {
	Label         string               `json:"label,omitempty"`
	LocalizedName string               `json:"localized_name,omitempty"`
	Name          string               `json:"name"`
	Type          string               `json:"type,omitempty"`
	Shape         int                  `json:"shape,omitempty"`
	Widget        *WorkspaceNodeWidget `json:"widget,omitempty"`
	Link          interface{}          `json:"link"`
	rawFields     map[string]json.RawMessage
}

type WorkspaceNodeOutput struct {
	Label         string `json:"label,omitempty"`
	LocalizedName string `json:"localized_name,omitempty"`
	Name          string `json:"name"`
	Type          string `json:"type,omitempty"`
	SlotIndex     int    `json:"slot_index,omitempty"`
	Links         []int  `json:"links,omitempty"`
	rawFields     map[string]json.RawMessage
}

type WorkspaceNodeWidget struct {
	Name string `json:"name"`
}

type WorkspaceGroup struct {
	ID        int                    `json:"id"`
	Title     string                 `json:"title"`
	Bounding  []float64              `json:"bounding,omitempty"`
	Color     string                 `json:"color,omitempty"`
	FontSize  int                    `json:"font_size,omitempty"`
	Flags     map[string]interface{} `json:"flags,omitempty"`
	rawFields map[string]json.RawMessage
}

type WorkspaceDefinitions struct {
	Subgraphs []WorkspaceSubgraphDefinition `json:"subgraphs,omitempty"`
	rawFields map[string]json.RawMessage
}

type WorkspaceSubgraphDefinition struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name,omitempty"`
	Version    int                      `json:"version,omitempty"`
	State      map[string]interface{}   `json:"state,omitempty"`
	Revision   int                      `json:"revision,omitempty"`
	Config     map[string]interface{}   `json:"config,omitempty"`
	InputNode  map[string]interface{}   `json:"inputNode,omitempty"`
	OutputNode map[string]interface{}   `json:"outputNode,omitempty"`
	Inputs     []map[string]interface{} `json:"inputs,omitempty"`
	Outputs    []map[string]interface{} `json:"outputs,omitempty"`
	Widgets    []interface{}            `json:"widgets,omitempty"`
	Nodes      []WorkspaceNode          `json:"nodes,omitempty"`
	Links      []WorkspaceLink          `json:"links,omitempty"`
	Groups     []WorkspaceGroup         `json:"groups,omitempty"`
	Extra      map[string]interface{}   `json:"extra,omitempty"`
	Category   string                   `json:"category,omitempty"`
	rawFields  map[string]json.RawMessage
}

type WorkspaceLink struct {
	ID          int    `json:"id"`
	OriginID    int    `json:"origin_id"`
	OriginSlot  int    `json:"origin_slot"`
	TargetID    int    `json:"target_id"`
	TargetSlot  int    `json:"target_slot"`
	Type        string `json:"type"`
	objectStyle bool
	rawFields   map[string]json.RawMessage
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
		l.objectStyle = false
		l.rawFields = nil
		return nil
	}

	type alias WorkspaceLink
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*l = WorkspaceLink(decoded)
	l.objectStyle = true
	raw, err := unmarshalRawFields(data)
	if err != nil {
		return err
	}
	l.rawFields = raw
	return nil
}

func (l WorkspaceLink) MarshalJSON() ([]byte, error) {
	if !l.objectStyle {
		return json.Marshal([]interface{}{l.ID, l.OriginID, l.OriginSlot, l.TargetID, l.TargetSlot, l.Type})
	}

	raw := cloneRawFields(l.rawFields)
	if err := marshalField(raw, "id", l.ID, l.ID != 0 || hasRawField(l.rawFields, "id")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "origin_id", l.OriginID, l.OriginID != 0 || hasRawField(l.rawFields, "origin_id")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "origin_slot", l.OriginSlot, l.OriginSlot != 0 || hasRawField(l.rawFields, "origin_slot")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "target_id", l.TargetID, l.TargetID != 0 || hasRawField(l.rawFields, "target_id")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "target_slot", l.TargetSlot, l.TargetSlot != 0 || hasRawField(l.rawFields, "target_slot")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "type", l.Type, l.Type != "" || hasRawField(l.rawFields, "type")); err != nil {
		return nil, err
	}
	return json.Marshal(raw)
}

func (w *Workspace) UnmarshalJSON(data []byte) error {
	type alias Workspace
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*w = Workspace(decoded)
	raw, err := unmarshalRawFields(data)
	if err != nil {
		return err
	}
	w.rawFields = raw
	return nil
}

func (w Workspace) MarshalJSON() ([]byte, error) {
	raw := cloneRawFields(w.rawFields)
	if err := marshalField(raw, "id", w.ID, w.ID != "" || hasRawField(w.rawFields, "id")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "name", w.Name, w.Name != "" || hasRawField(w.rawFields, "name")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "revision", w.Revision, w.Revision != 0 || hasRawField(w.rawFields, "revision")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "last_node_id", w.LastNodeID, w.LastNodeID != 0 || hasRawField(w.rawFields, "last_node_id")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "last_link_id", w.LastLinkID, w.LastLinkID != 0 || hasRawField(w.rawFields, "last_link_id")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "nodes", w.Nodes, len(w.Nodes) > 0 || hasRawField(w.rawFields, "nodes")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "links", w.Links, len(w.Links) > 0 || hasRawField(w.rawFields, "links")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "groups", w.Groups, len(w.Groups) > 0 || hasRawField(w.rawFields, "groups")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "definitions", w.Definitions, len(w.Definitions.Subgraphs) > 0 || hasRawField(w.rawFields, "definitions")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "config", w.Config, len(w.Config) > 0 || hasRawField(w.rawFields, "config")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "extra", w.Extra, len(w.Extra) > 0 || hasRawField(w.rawFields, "extra")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "version", w.Version, w.Version != nil || hasRawField(w.rawFields, "version")); err != nil {
		return nil, err
	}
	return json.Marshal(raw)
}

func (n *WorkspaceNode) UnmarshalJSON(data []byte) error {
	type alias WorkspaceNode
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*n = WorkspaceNode(decoded)
	raw, err := unmarshalRawFields(data)
	if err != nil {
		return err
	}
	n.rawFields = raw
	return nil
}

func (n WorkspaceNode) MarshalJSON() ([]byte, error) {
	raw := cloneRawFields(n.rawFields)
	if err := marshalField(raw, "id", n.ID, n.ID != 0 || hasRawField(n.rawFields, "id")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "type", n.Type, n.Type != "" || hasRawField(n.rawFields, "type")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "pos", n.Pos, len(n.Pos) > 0 || hasRawField(n.rawFields, "pos")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "size", n.Size, len(n.Size) > 0 || hasRawField(n.rawFields, "size")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "flags", n.Flags, len(n.Flags) > 0 || hasRawField(n.rawFields, "flags")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "order", n.Order, n.Order != 0 || hasRawField(n.rawFields, "order")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "mode", n.Mode, n.Mode != 0 || hasRawField(n.rawFields, "mode")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "inputs", n.Inputs, len(n.Inputs) > 0 || hasRawField(n.rawFields, "inputs")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "outputs", n.Outputs, len(n.Outputs) > 0 || hasRawField(n.rawFields, "outputs")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "properties", n.Properties, len(n.Properties) > 0 || hasRawField(n.rawFields, "properties")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "widgets_values", n.WidgetsValues, len(n.WidgetsValues) > 0 || hasRawField(n.rawFields, "widgets_values")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "title", n.Title, n.Title != "" || hasRawField(n.rawFields, "title")); err != nil {
		return nil, err
	}
	return json.Marshal(raw)
}

func (i *WorkspaceNodeInput) UnmarshalJSON(data []byte) error {
	type alias WorkspaceNodeInput
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*i = WorkspaceNodeInput(decoded)
	raw, err := unmarshalRawFields(data)
	if err != nil {
		return err
	}
	i.rawFields = raw
	return nil
}

func (i WorkspaceNodeInput) MarshalJSON() ([]byte, error) {
	raw := cloneRawFields(i.rawFields)
	if err := marshalField(raw, "label", i.Label, i.Label != "" || hasRawField(i.rawFields, "label")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "localized_name", i.LocalizedName, i.LocalizedName != "" || hasRawField(i.rawFields, "localized_name")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "name", i.Name, i.Name != "" || hasRawField(i.rawFields, "name")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "type", i.Type, i.Type != "" || hasRawField(i.rawFields, "type")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "shape", i.Shape, i.Shape != 0 || hasRawField(i.rawFields, "shape")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "widget", i.Widget, i.Widget != nil || hasRawField(i.rawFields, "widget")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "link", i.Link, true); err != nil {
		return nil, err
	}
	return json.Marshal(raw)
}

func (o *WorkspaceNodeOutput) UnmarshalJSON(data []byte) error {
	type alias WorkspaceNodeOutput
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*o = WorkspaceNodeOutput(decoded)
	raw, err := unmarshalRawFields(data)
	if err != nil {
		return err
	}
	o.rawFields = raw
	return nil
}

func (o WorkspaceNodeOutput) MarshalJSON() ([]byte, error) {
	raw := cloneRawFields(o.rawFields)
	if err := marshalField(raw, "label", o.Label, o.Label != "" || hasRawField(o.rawFields, "label")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "localized_name", o.LocalizedName, o.LocalizedName != "" || hasRawField(o.rawFields, "localized_name")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "name", o.Name, o.Name != "" || hasRawField(o.rawFields, "name")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "type", o.Type, o.Type != "" || hasRawField(o.rawFields, "type")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "slot_index", o.SlotIndex, o.SlotIndex != 0 || hasRawField(o.rawFields, "slot_index")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "links", o.Links, len(o.Links) > 0 || hasRawField(o.rawFields, "links")); err != nil {
		return nil, err
	}
	return json.Marshal(raw)
}

func (g *WorkspaceGroup) UnmarshalJSON(data []byte) error {
	type alias WorkspaceGroup
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*g = WorkspaceGroup(decoded)
	raw, err := unmarshalRawFields(data)
	if err != nil {
		return err
	}
	g.rawFields = raw
	return nil
}

func (g WorkspaceGroup) MarshalJSON() ([]byte, error) {
	raw := cloneRawFields(g.rawFields)
	if err := marshalField(raw, "id", g.ID, g.ID != 0 || hasRawField(g.rawFields, "id")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "title", g.Title, g.Title != "" || hasRawField(g.rawFields, "title")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "bounding", g.Bounding, len(g.Bounding) > 0 || hasRawField(g.rawFields, "bounding")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "color", g.Color, g.Color != "" || hasRawField(g.rawFields, "color")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "font_size", g.FontSize, g.FontSize != 0 || hasRawField(g.rawFields, "font_size")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "flags", g.Flags, len(g.Flags) > 0 || hasRawField(g.rawFields, "flags")); err != nil {
		return nil, err
	}
	return json.Marshal(raw)
}

func (d *WorkspaceDefinitions) UnmarshalJSON(data []byte) error {
	type alias WorkspaceDefinitions
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*d = WorkspaceDefinitions(decoded)
	raw, err := unmarshalRawFields(data)
	if err != nil {
		return err
	}
	d.rawFields = raw
	return nil
}

func (d WorkspaceDefinitions) MarshalJSON() ([]byte, error) {
	raw := cloneRawFields(d.rawFields)
	if err := marshalField(raw, "subgraphs", d.Subgraphs, len(d.Subgraphs) > 0 || hasRawField(d.rawFields, "subgraphs")); err != nil {
		return nil, err
	}
	return json.Marshal(raw)
}

func (s *WorkspaceSubgraphDefinition) UnmarshalJSON(data []byte) error {
	type alias WorkspaceSubgraphDefinition
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*s = WorkspaceSubgraphDefinition(decoded)
	raw, err := unmarshalRawFields(data)
	if err != nil {
		return err
	}
	s.rawFields = raw
	return nil
}

func (s WorkspaceSubgraphDefinition) MarshalJSON() ([]byte, error) {
	raw := cloneRawFields(s.rawFields)
	if err := marshalField(raw, "id", s.ID, s.ID != "" || hasRawField(s.rawFields, "id")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "version", s.Version, s.Version != 0 || hasRawField(s.rawFields, "version")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "state", s.State, len(s.State) > 0 || hasRawField(s.rawFields, "state")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "revision", s.Revision, s.Revision != 0 || hasRawField(s.rawFields, "revision")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "config", s.Config, len(s.Config) > 0 || hasRawField(s.rawFields, "config")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "name", s.Name, s.Name != "" || hasRawField(s.rawFields, "name")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "inputNode", s.InputNode, len(s.InputNode) > 0 || hasRawField(s.rawFields, "inputNode")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "outputNode", s.OutputNode, len(s.OutputNode) > 0 || hasRawField(s.rawFields, "outputNode")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "inputs", s.Inputs, len(s.Inputs) > 0 || hasRawField(s.rawFields, "inputs")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "outputs", s.Outputs, len(s.Outputs) > 0 || hasRawField(s.rawFields, "outputs")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "widgets", s.Widgets, len(s.Widgets) > 0 || hasRawField(s.rawFields, "widgets")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "nodes", s.Nodes, len(s.Nodes) > 0 || hasRawField(s.rawFields, "nodes")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "groups", s.Groups, len(s.Groups) > 0 || hasRawField(s.rawFields, "groups")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "links", s.Links, len(s.Links) > 0 || hasRawField(s.rawFields, "links")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "extra", s.Extra, len(s.Extra) > 0 || hasRawField(s.rawFields, "extra")); err != nil {
		return nil, err
	}
	if err := marshalField(raw, "category", s.Category, s.Category != "" || hasRawField(s.rawFields, "category")); err != nil {
		return nil, err
	}
	return json.Marshal(raw)
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
	if workspace.Config == nil && hasRawField(workspace.rawFields, "config") {
		workspace.Config = map[string]interface{}{}
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

func unmarshalRawFields(data []byte) (map[string]json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func cloneRawFields(src map[string]json.RawMessage) map[string]json.RawMessage {
	if len(src) == 0 {
		return map[string]json.RawMessage{}
	}
	cloned := make(map[string]json.RawMessage, len(src))
	for key, value := range src {
		cloned[key] = append(json.RawMessage(nil), value...)
	}
	return cloned
}

func hasRawField(raw map[string]json.RawMessage, key string) bool {
	if raw == nil {
		return false
	}
	_, ok := raw[key]
	return ok
}

func marshalField(raw map[string]json.RawMessage, key string, value interface{}, include bool) error {
	if !include {
		delete(raw, key)
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal field %s: %w", key, err)
	}
	raw[key] = data
	return nil
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
