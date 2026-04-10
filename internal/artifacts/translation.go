package artifacts

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

type TranslationReport struct {
	PreservedFields   []string `json:"preserved_fields"`
	SynthesizedFields []string `json:"synthesized_fields"`
	UnsupportedFields []string `json:"unsupported_fields"`
	Notes             []string `json:"notes"`
}

func NewTranslationReport() *TranslationReport {
	return &TranslationReport{
		PreservedFields:   []string{},
		SynthesizedFields: []string{},
		UnsupportedFields: []string{},
		Notes:             []string{},
	}
}

func (r *TranslationReport) AddPreservedField(path string) {
	r.PreservedFields = append(r.PreservedFields, path)
}

func (r *TranslationReport) AddSynthesizedField(path string) {
	r.SynthesizedFields = append(r.SynthesizedFields, path)
}

func (r *TranslationReport) AddUnsupportedField(path string) {
	r.UnsupportedFields = append(r.UnsupportedFields, path)
}

func (r *TranslationReport) AddNote(note string) {
	r.Notes = append(r.Notes, note)
}

func (r *TranslationReport) Fidelity() string {
	if r == nil {
		return "lossless"
	}
	if len(r.UnsupportedFields) > 0 {
		return "lossy"
	}
	if len(r.SynthesizedFields) > 0 {
		return "synthetic"
	}
	return "lossless"
}

func TranslateWorkspaceToPrompt(workspace *Workspace) (*Prompt, *TranslationReport, error) {
	if workspace == nil {
		return nil, nil, fmt.Errorf("workspace is nil")
	}

	nodes := workspace.Nodes
	links := workspace.Links
	if len(nodes) == 0 && len(workspace.Definitions.Subgraphs) > 0 {
		nodes = workspace.Definitions.Subgraphs[0].Nodes
		links = workspace.Definitions.Subgraphs[0].Links
	}
	if len(nodes) == 0 {
		return nil, nil, fmt.Errorf("workspace must contain at least one node")
	}

	linkByID := make(map[int]WorkspaceLink, len(links))
	for _, link := range links {
		linkByID[link.ID] = link
	}

	report := NewTranslationReport()
	prompt := &Prompt{Nodes: make(map[string]PromptNode, len(nodes))}
	for _, node := range nodes {
		inputs := map[string]interface{}{}
		widgetIndex := 0
		for _, input := range node.Inputs {
			if linkID, ok := linkValueToInt(input.Link); ok {
				link, found := linkByID[linkID]
				if found {
					inputs[input.Name] = []interface{}{strconv.Itoa(link.OriginID), link.OriginSlot}
				}
				continue
			}
			if input.Widget != nil && widgetIndex < len(node.WidgetsValues) {
				inputs[input.Name] = node.WidgetsValues[widgetIndex]
				widgetIndex++
			}
		}

		prompt.Nodes[strconv.Itoa(node.ID)] = PromptNode{
			ClassType: node.Type,
			Inputs:    inputs,
			Meta:      map[string]interface{}{},
		}
		report.AddPreservedField(fmt.Sprintf("nodes[%d].type", node.ID))
		if len(node.Pos) > 0 {
			report.AddUnsupportedField(fmt.Sprintf("nodes[%d].pos", node.ID))
		}
		if len(node.Size) > 0 {
			report.AddUnsupportedField(fmt.Sprintf("nodes[%d].size", node.ID))
		}
	}
	for _, group := range workspace.Groups {
		report.AddUnsupportedField(fmt.Sprintf("groups[%d]", group.ID))
	}
	if len(workspace.Definitions.Subgraphs) > 0 {
		report.AddUnsupportedField("definitions.subgraphs")
	}

	return prompt, report, nil
}

func TranslatePromptToWorkspace(name string, prompt *Prompt, nodeInfo map[string]client.NodeInfo) (*Workspace, *TranslationReport, error) {
	if prompt == nil {
		return nil, nil, fmt.Errorf("prompt is nil")
	}
	if len(prompt.Nodes) == 0 {
		return nil, nil, fmt.Errorf("prompt must contain at least one node")
	}

	sortedIDs := sortedPromptNodeIDs(prompt.Nodes)
	linkID := 1
	outgoingLinks := make(map[string]map[int][]int, len(prompt.Nodes))
	workspaceLinks := []WorkspaceLink{}
	for _, nodeID := range sortedIDs {
		node := prompt.Nodes[nodeID]
		info := nodeInfo[node.ClassType]
		orderedInputs := orderedPromptInputs(node, info)
		for _, input := range orderedInputs {
			inputName := input.Name
			value, exists := node.Inputs[inputName]
			if !exists {
				continue
			}
			originID, originSlot, ok := promptLinkValue(value)
			if !ok {
				continue
			}
			targetID, _ := strconv.Atoi(nodeID)
			originNumericID, _ := strconv.Atoi(originID)
			linkType := "UNKNOWN"
			if info, ok := nodeInfo[prompt.Nodes[originID].ClassType]; ok && originSlot < len(info.Output) {
				linkType = info.Output[originSlot]
			}
			workspaceLinks = append(workspaceLinks, WorkspaceLink{
				ID:         linkID,
				OriginID:   originNumericID,
				OriginSlot: originSlot,
				TargetID:   targetID,
				TargetSlot: input.Slot,
				Type:       linkType,
			})
			if outgoingLinks[originID] == nil {
				outgoingLinks[originID] = map[int][]int{}
			}
			outgoingLinks[originID][originSlot] = append(outgoingLinks[originID][originSlot], linkID)
			linkID++
		}
	}

	workspace := &Workspace{
		Name:    name,
		Nodes:   make([]WorkspaceNode, 0, len(sortedIDs)),
		Links:   workspaceLinks,
		Groups:  []WorkspaceGroup{},
		Extra:   map[string]interface{}{},
		Version: 0.4,
	}

	report := NewTranslationReport()
	report.AddSynthesizedField("nodes[].pos")
	report.AddSynthesizedField("nodes[].size")
	report.AddNote("Workspace node positions and sizes were synthesized from prompt graph structure.")

	for index, nodeID := range sortedIDs {
		node := prompt.Nodes[nodeID]
		numericID, _ := strconv.Atoi(nodeID)
		info := nodeInfo[node.ClassType]
		orderedInputs := orderedPromptInputs(node, info)
		widgetValues := []interface{}{}
		workspaceInputs := make([]WorkspaceNodeInput, 0, len(orderedInputs))
		for _, input := range orderedInputs {
			inputName := input.Name
			value, exists := node.Inputs[inputName]
			if !exists {
				continue
			}
			inputType := inputTypeForNode(info, inputName, value)
			if linkRef, _, ok := promptLinkValue(value); ok {
				linkRefID, found := findLinkID(workspaceLinks, linkRef, nodeID, input.Slot)
				if !found {
					return nil, nil, fmt.Errorf("link not found for prompt node %s input %s", nodeID, inputName)
				}
				workspaceInputs = append(workspaceInputs, WorkspaceNodeInput{
					Name: inputName,
					Type: inputType,
					Link: linkRefID,
				})
				continue
			}
			workspaceInputs = append(workspaceInputs, WorkspaceNodeInput{
				Name:   inputName,
				Type:   inputType,
				Widget: &WorkspaceNodeWidget{Name: inputName},
				Link:   nil,
			})
			widgetValues = append(widgetValues, value)
		}

		workspaceOutputs := buildWorkspaceOutputs(nodeID, node, info, outgoingLinks[nodeID])
		workspace.Nodes = append(workspace.Nodes, WorkspaceNode{
			ID:            numericID,
			Type:          node.ClassType,
			Pos:           []float64{float64(index) * 280, 0},
			Size:          []float64{240, 120},
			Inputs:        workspaceInputs,
			Outputs:       workspaceOutputs,
			Properties:    map[string]interface{}{},
			WidgetsValues: widgetValues,
		})
		report.AddPreservedField(fmt.Sprintf("prompt.%s.class_type", nodeID))
	}

	return workspace, report, nil
}

func sortedPromptNodeIDs(nodes map[string]PromptNode) []string {
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

func linkValueToInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func promptLinkValue(value interface{}) (string, int, bool) {
	ref, ok := value.([]interface{})
	if !ok || len(ref) != 2 {
		return "", 0, false
	}
	originID, ok := ref[0].(string)
	if !ok {
		return "", 0, false
	}
	return originID, interfaceToInt(ref[1]), true
}

type orderedPromptInput struct {
	Name string
	Slot int
}

func orderedPromptInputs(node PromptNode, info client.NodeInfo) []orderedPromptInput {
	ordered := []orderedPromptInput{}
	added := map[string]bool{}
	extras := []string{}
	slot := 0
	for _, section := range []string{"required", "optional"} {
		for _, name := range info.InputOrder[section] {
			if _, ok := node.Inputs[name]; ok {
				ordered = append(ordered, orderedPromptInput{Name: name, Slot: slot})
				added[name] = true
			}
			slot++
		}
	}
	for name := range node.Inputs {
		if !added[name] {
			extras = append(extras, name)
		}
	}
	sort.Strings(extras)
	for _, name := range extras {
		ordered = append(ordered, orderedPromptInput{Name: name, Slot: slot})
		slot++
	}
	return ordered
}

func inputTypeForNode(info client.NodeInfo, inputName string, value interface{}) string {
	for _, section := range []map[string]interface{}{info.Input.Required, info.Input.Optional} {
		if raw, ok := section[inputName]; ok {
			if values, ok := raw.([]interface{}); ok && len(values) > 0 {
				if inputType, ok := values[0].(string); ok {
					return inputType
				}
			}
		}
	}
	if _, _, ok := promptLinkValue(value); ok {
		return "LINK"
	}
	return "UNKNOWN"
}

func buildWorkspaceOutputs(nodeID string, node PromptNode, info client.NodeInfo, outgoing map[int][]int) []WorkspaceNodeOutput {
	count := len(info.OutputName)
	if count == 0 {
		count = len(info.Output)
	}
	outputs := make([]WorkspaceNodeOutput, 0, count)
	for slot := 0; slot < count; slot++ {
		name := fmt.Sprintf("OUTPUT_%d", slot)
		if slot < len(info.OutputName) && info.OutputName[slot] != "" {
			name = info.OutputName[slot]
		}
		outputType := "UNKNOWN"
		if slot < len(info.Output) && info.Output[slot] != "" {
			outputType = info.Output[slot]
		}
		outputs = append(outputs, WorkspaceNodeOutput{
			Name:  name,
			Type:  outputType,
			Links: append([]int(nil), outgoing[slot]...),
		})
	}
	return outputs
}

func findLinkID(links []WorkspaceLink, originID string, targetID string, targetSlot int) (int, bool) {
	originNumericID, _ := strconv.Atoi(originID)
	targetNumericID, _ := strconv.Atoi(targetID)
	for _, link := range links {
		if link.OriginID == originNumericID && link.TargetID == targetNumericID && link.TargetSlot == targetSlot {
			return link.ID, true
		}
	}
	return 0, false
}
