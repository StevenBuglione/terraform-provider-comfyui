package resources

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

const (
	defaultWorkflowGap  = 320.0
	defaultNodeYSpacing = 140.0
)

type workspaceWorkflowSpec struct {
	Name         string
	WorkflowJSON string
	X            *float64
	Y            *float64
}

type renderedWorkspaceWorkflow struct {
	spec   workspaceWorkflowSpec
	nodes  []workspaceNode
	links  []workspaceLink
	group  *workspaceGroup
	width  float64
	height float64
}

type workspaceSubgraph struct {
	Name     string                 `json:"name"`
	Version  int                    `json:"version"`
	Revision int                    `json:"revision"`
	Config   map[string]interface{} `json:"config"`
	State    workspaceSubgraphState `json:"state"`
	Nodes    []workspaceNode        `json:"nodes"`
	Links    []workspaceLink        `json:"links"`
	Groups   []workspaceGroup       `json:"groups"`
	Inputs   []interface{}          `json:"inputs"`
	Outputs  []interface{}          `json:"outputs"`
	Widgets  []interface{}          `json:"widgets"`
}

type workspaceSubgraphState struct {
	LastGroupID   int `json:"lastGroupId"`
	LastNodeID    int `json:"lastNodeId"`
	LastLinkID    int `json:"lastLinkId"`
	LastRerouteID int `json:"lastRerouteId"`
}

type workspaceNode struct {
	ID            int                    `json:"id"`
	Type          string                 `json:"type"`
	Pos           []float64              `json:"pos"`
	Size          []float64              `json:"size"`
	Flags         map[string]interface{} `json:"flags"`
	Order         int                    `json:"order"`
	Mode          int                    `json:"mode"`
	Inputs        []workspaceNodeInput   `json:"inputs"`
	Outputs       []workspaceNodeOutput  `json:"outputs"`
	Properties    map[string]interface{} `json:"properties"`
	WidgetsValues []interface{}          `json:"widgets_values"`
}

type workspaceNodeInput struct {
	LocalizedName string               `json:"localized_name,omitempty"`
	Name          string               `json:"name"`
	Type          string               `json:"type"`
	Widget        *workspaceNodeWidget `json:"widget,omitempty"`
	Link          interface{}          `json:"link"`
}

type workspaceNodeOutput struct {
	LocalizedName string `json:"localized_name,omitempty"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Links         []int  `json:"links"`
}

type workspaceNodeWidget struct {
	Name string `json:"name"`
}

type workspaceLink struct {
	ID         int    `json:"id"`
	OriginID   int    `json:"origin_id"`
	OriginSlot int    `json:"origin_slot"`
	TargetID   int    `json:"target_id"`
	TargetSlot int    `json:"target_slot"`
	Type       string `json:"type"`
}

type workspaceGroup struct {
	ID       int                    `json:"id"`
	Title    string                 `json:"title"`
	Bounding []float64              `json:"bounding"`
	Color    string                 `json:"color,omitempty"`
	FontSize int                    `json:"font_size,omitempty"`
	Flags    map[string]interface{} `json:"flags"`
}

type promptNode struct {
	ClassType string                 `json:"class_type"`
	Inputs    map[string]interface{} `json:"inputs"`
}

func buildWorkspaceSubgraph(name string, workflows []workspaceWorkflowSpec, layout workspaceLayoutConfig, nodeInfo map[string]client.NodeInfo) (*workspaceSubgraph, error) {
	if err := validateWorkspaceLayout(layout); err != nil {
		return nil, err
	}
	if len(workflows) == 0 {
		return nil, fmt.Errorf("at least one workflow is required")
	}

	gap := layout.Gap
	if gap <= 0 {
		gap = defaultWorkflowGap
	}

	subgraph := &workspaceSubgraph{
		Name:     name,
		Version:  1,
		Revision: 0,
		Config:   map[string]interface{}{},
		Nodes:    []workspaceNode{},
		Links:    []workspaceLink{},
		Groups:   []workspaceGroup{},
		Inputs:   []interface{}{},
		Outputs:  []interface{}{},
		Widgets:  []interface{}{},
	}

	nextNodeID := 1
	nextLinkID := 1
	globalOrder := 0
	renderedWorkflows := make([]renderedWorkspaceWorkflow, 0, len(workflows))

	for workflowIndex, workflow := range workflows {
		var prompt map[string]promptNode
		if err := json.Unmarshal([]byte(workflow.WorkflowJSON), &prompt); err != nil {
			return nil, fmt.Errorf("workflow %q: invalid workflow_json: %w", workflow.Name, err)
		}

		originalIDs, err := sortedPromptNodeIDs(prompt)
		if err != nil {
			return nil, fmt.Errorf("workflow %q: %w", workflow.Name, err)
		}

		idMap := make(map[string]int, len(originalIDs))
		for _, originalID := range originalIDs {
			idMap[originalID] = nextNodeID
			nextNodeID++
		}

		nodeIndexByID := make(map[int]int, len(originalIDs))
		inputOrders := make(map[int][]string, len(originalIDs))
		workflowNodeIDs := make([]int, 0, len(originalIDs))
		workflowNodes := make([]workspaceNode, 0, len(originalIDs))
		workflowLinks := make([]workspaceLink, 0, len(originalIDs))

		for nodeIndex, originalID := range originalIDs {
			nodeData := prompt[originalID]
			info, ok := nodeInfo[nodeData.ClassType]
			if !ok {
				return nil, fmt.Errorf("workflow %q: node type %q not found in object info", workflow.Name, nodeData.ClassType)
			}

			orderedInputs, err := orderedNodeInputs(info)
			if err != nil {
				return nil, fmt.Errorf("workflow %q: node type %q: %w", workflow.Name, nodeData.ClassType, err)
			}
			node, widgetValues := buildWorkspaceNode(idMap[originalID], nodeData.ClassType, info, orderedInputs, nodeData.Inputs, 0, 0, nodeIndex, globalOrder)
			node.WidgetsValues = widgetValues
			workflowNodes = append(workflowNodes, node)
			nodeIndexByID[node.ID] = len(workflowNodes) - 1
			inputOrders[node.ID] = orderedInputs
			workflowNodeIDs = append(workflowNodeIDs, node.ID)
			globalOrder++
		}

		for _, originalID := range originalIDs {
			nodeData := prompt[originalID]
			targetNodeID := idMap[originalID]
			targetNode := &workflowNodes[nodeIndexByID[targetNodeID]]
			orderedInputs := inputOrders[targetNodeID]

			for inputIndex, inputName := range orderedInputs {
				value, exists := nodeData.Inputs[inputName]
				if !exists {
					continue
				}

				originOriginalID, originSlot, ok := parsePromptConnection(value)
				if !ok {
					continue
				}

				originNodeID, exists := idMap[originOriginalID]
				if !exists {
					return nil, fmt.Errorf("workflow %q: unknown origin node %q", workflow.Name, originOriginalID)
				}

				linkType := targetNode.Inputs[inputIndex].Type
				link := workspaceLink{
					ID:         nextLinkID,
					OriginID:   originNodeID,
					OriginSlot: originSlot,
					TargetID:   targetNodeID,
					TargetSlot: inputIndex,
					Type:       linkType,
				}
				workflowLinks = append(workflowLinks, link)
				targetNode.Inputs[inputIndex].Link = nextLinkID
				originNode := &workflowNodes[nodeIndexByID[originNodeID]]
				if originSlot < 0 || originSlot >= len(originNode.Outputs) {
					return nil, fmt.Errorf("workflow %q: node %d output slot %d is out of range", workflow.Name, originNodeID, originSlot)
				}
				originNode.Outputs[originSlot].Links = append(originNode.Outputs[originSlot].Links, nextLinkID)
				nextLinkID++
			}
		}

		var group *workspaceGroup
		if workflow.Name != "" && len(workflowNodeIDs) > 0 {
			builtGroup := buildWorkspaceGroup(workflowIndex+1, workflow.Name, workflowNodeIDs, nodeIndexByID, workflowNodes)
			group = &builtGroup
		}

		rendered := renderedWorkspaceWorkflow{
			spec:  workflow,
			nodes: workflowNodes,
			links: workflowLinks,
			group: group,
		}
		if group != nil {
			rendered.width = group.Bounding[2]
			rendered.height = group.Bounding[3]
		} else {
			rendered.width, rendered.height = workflowNodeBounds(workflowNodes)
		}
		renderedWorkflows = append(renderedWorkflows, rendered)
	}

	autoPositions := workflowBasePositions(renderedWorkflows, layout, gap)
	for workflowIndex, rendered := range renderedWorkflows {
		baseX, baseY := autoPositions[workflowIndex][0], autoPositions[workflowIndex][1]
		if rendered.spec.X != nil {
			baseX = *rendered.spec.X
		}
		if rendered.spec.Y != nil {
			baseY = *rendered.spec.Y
		}

		translatedNodes := translateWorkflowNodes(rendered.nodes, baseX, baseY)
		subgraph.Nodes = append(subgraph.Nodes, translatedNodes...)
		subgraph.Links = append(subgraph.Links, rendered.links...)
		if rendered.group != nil {
			subgraph.Groups = append(subgraph.Groups, translateWorkflowGroup(*rendered.group, baseX, baseY))
		}
	}

	subgraph.State = workspaceSubgraphState{
		LastGroupID: len(subgraph.Groups),
		LastNodeID:  nextNodeID - 1,
		LastLinkID:  nextLinkID - 1,
	}

	return subgraph, nil
}

func sortedPromptNodeIDs(prompt map[string]promptNode) ([]string, error) {
	type nodeID struct {
		raw string
		num int
	}

	ids := make([]nodeID, 0, len(prompt))
	for id := range prompt {
		n, err := strconv.Atoi(id)
		if err != nil {
			return nil, fmt.Errorf("node id %q is not numeric", id)
		}
		ids = append(ids, nodeID{raw: id, num: n})
	}

	sort.Slice(ids, func(i, j int) bool {
		return ids[i].num < ids[j].num
	})

	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = id.raw
	}
	return result, nil
}

func orderedNodeInputs(info client.NodeInfo) ([]string, error) {
	definedInputs := len(info.Input.Required) + len(info.Input.Optional)
	if definedInputs == 0 {
		return []string{}, nil
	}

	ordered := make([]string, 0, len(info.Input.Required)+len(info.Input.Optional))
	ordered = append(ordered, info.InputOrder["required"]...)
	ordered = append(ordered, info.InputOrder["optional"]...)
	if len(ordered) == 0 {
		return nil, fmt.Errorf("missing input_order metadata")
	}

	seen := make(map[string]struct{}, len(ordered))
	for _, inputName := range ordered {
		if _, ok := seen[inputName]; ok {
			return nil, fmt.Errorf("input_order contains duplicate input %q", inputName)
		}
		if !inputExists(info.Input.Required, inputName) && !inputExists(info.Input.Optional, inputName) {
			return nil, fmt.Errorf("input_order contains unknown input %q", inputName)
		}
		seen[inputName] = struct{}{}
	}
	for inputName := range info.Input.Required {
		if _, ok := seen[inputName]; !ok {
			return nil, fmt.Errorf("input_order is missing required input %q", inputName)
		}
	}
	for inputName := range info.Input.Optional {
		if _, ok := seen[inputName]; !ok {
			return nil, fmt.Errorf("input_order is missing optional input %q", inputName)
		}
	}

	return ordered, nil
}

func buildWorkspaceNode(nodeID int, classType string, info client.NodeInfo, orderedInputs []string, rawInputs map[string]interface{}, baseX, baseY float64, nodeIndex int, order int) (workspaceNode, []interface{}) {
	node := workspaceNode{
		ID:         nodeID,
		Type:       classType,
		Pos:        []float64{baseX, baseY + float64(nodeIndex)*defaultNodeYSpacing},
		Size:       []float64{240, 120},
		Flags:      map[string]interface{}{},
		Order:      order,
		Mode:       0,
		Inputs:     make([]workspaceNodeInput, 0, len(orderedInputs)),
		Outputs:    make([]workspaceNodeOutput, 0, len(info.Output)),
		Properties: map[string]interface{}{"Node name for S&R": classType},
	}

	widgetValues := make([]interface{}, 0, len(orderedInputs))

	for _, inputName := range orderedInputs {
		inputType := lookupNodeInputType(info, inputName)
		value, exists := rawInputs[inputName]
		_, _, isLink := parsePromptConnection(value)
		widgetBacked := isWidgetInputType(inputType)

		input := workspaceNodeInput{
			LocalizedName: inputName,
			Name:          inputName,
			Type:          inputType,
			Link:          nil,
		}
		if widgetBacked {
			input.Widget = &workspaceNodeWidget{Name: inputName}
		}
		node.Inputs = append(node.Inputs, input)

		if exists && !isLink && widgetBacked {
			widgetValues = append(widgetValues, value)
		} else if !exists && widgetBacked {
			widgetValues = append(widgetValues, nil)
		}
	}

	for outputIndex, outputType := range info.Output {
		outputName := outputType
		if outputIndex < len(info.OutputName) && info.OutputName[outputIndex] != "" {
			outputName = info.OutputName[outputIndex]
		}
		node.Outputs = append(node.Outputs, workspaceNodeOutput{
			LocalizedName: outputName,
			Name:          outputName,
			Type:          outputType,
			Links:         []int{},
		})
	}

	return node, widgetValues
}

func lookupNodeInputType(info client.NodeInfo, inputName string) string {
	if inputType, ok := extractInputType(info.Input.Required[inputName]); ok {
		return inputType
	}
	if inputType, ok := extractInputType(info.Input.Optional[inputName]); ok {
		return inputType
	}
	return "UNKNOWN"
}

func extractInputType(raw interface{}) (string, bool) {
	values, ok := raw.([]interface{})
	if !ok || len(values) == 0 {
		return "", false
	}
	if _, ok := values[0].([]interface{}); ok {
		return "COMBO", true
	}
	value, ok := values[0].(string)
	if !ok {
		return "", false
	}
	return value, true
}

func parsePromptConnection(value interface{}) (string, int, bool) {
	values, ok := value.([]interface{})
	if !ok || len(values) != 2 {
		return "", 0, false
	}

	originID, ok := values[0].(string)
	if !ok {
		return "", 0, false
	}

	switch slot := values[1].(type) {
	case float64:
		return originID, int(slot), true
	case int:
		return originID, slot, true
	default:
		return "", 0, false
	}
}

func isWidgetInputType(inputType string) bool {
	switch inputType {
	case "STRING", "INT", "FLOAT", "BOOLEAN", "BOOL", "COMBO", "NUMBER":
		return true
	default:
		return false
	}
}

func inputExists(inputs map[string]interface{}, inputName string) bool {
	if inputs == nil {
		return false
	}
	_, ok := inputs[inputName]
	return ok
}

func buildWorkspaceGroup(groupID int, title string, workflowNodeIDs []int, nodeIndexByID map[int]int, nodes []workspaceNode) workspaceGroup {
	minX, minY := 0.0, 0.0
	maxX, maxY := 0.0, 0.0

	for index, nodeID := range workflowNodeIDs {
		node := nodes[nodeIndexByID[nodeID]]
		x, y := node.Pos[0], node.Pos[1]
		width, height := node.Size[0], node.Size[1]
		if index == 0 {
			minX, minY = x, y
			maxX, maxY = x+width, y+height
			continue
		}
		if x < minX {
			minX = x
		}
		if y < minY {
			minY = y
		}
		if x+width > maxX {
			maxX = x + width
		}
		if y+height > maxY {
			maxY = y + height
		}
	}

	padding := 40.0

	return workspaceGroup{
		ID:       groupID,
		Title:    title,
		Bounding: []float64{minX - padding, minY - padding, (maxX - minX) + padding*2, (maxY - minY) + padding*2},
		Flags:    map[string]interface{}{},
	}
}

func workflowBasePosition(workflowIndex int, layout workspaceLayoutConfig, gap float64) (float64, float64) {
	switch layout.Display {
	case "grid":
		columns := int(layout.Columns)
		if columns <= 0 {
			columns = 1
		}
		column := workflowIndex % columns
		row := workflowIndex / columns
		return layout.OriginX + float64(column)*gap, layout.OriginY + float64(row)*gap
	default:
		if layout.Direction == "column" {
			return layout.OriginX, layout.OriginY + float64(workflowIndex)*gap
		}
		return layout.OriginX + float64(workflowIndex)*gap, layout.OriginY
	}
}

func workflowBasePositions(workflows []renderedWorkspaceWorkflow, layout workspaceLayoutConfig, gap float64) [][2]float64 {
	positions := make([][2]float64, len(workflows))

	switch layout.Display {
	case "grid":
		columns := int(layout.Columns)
		if columns <= 0 {
			columns = 1
		}

		columnWidths := make([]float64, columns)
		rowHeights := make([]float64, 0, (len(workflows)+columns-1)/columns)
		for index, workflow := range workflows {
			column := index % columns
			row := index / columns
			if row >= len(rowHeights) {
				rowHeights = append(rowHeights, 0)
			}
			if workflow.width > columnWidths[column] {
				columnWidths[column] = workflow.width
			}
			if workflow.height > rowHeights[row] {
				rowHeights[row] = workflow.height
			}
		}

		for index := range workflows {
			column := index % columns
			row := index / columns
			x := layout.OriginX
			for prevColumn := 0; prevColumn < column; prevColumn++ {
				x += columnWidths[prevColumn] + gap
			}
			y := layout.OriginY
			for prevRow := 0; prevRow < row; prevRow++ {
				y += rowHeights[prevRow] + gap
			}
			positions[index] = [2]float64{x, y}
		}
	default:
		nextX := layout.OriginX
		nextY := layout.OriginY
		for index, workflow := range workflows {
			positions[index] = [2]float64{nextX, nextY}
			if layout.Direction == "column" {
				nextY += workflow.height + gap
				continue
			}
			nextX += workflow.width + gap
		}
	}

	return positions
}

func workflowNodeBounds(nodes []workspaceNode) (float64, float64) {
	if len(nodes) == 0 {
		return 0, 0
	}

	minX, minY := nodes[0].Pos[0], nodes[0].Pos[1]
	maxX, maxY := nodes[0].Pos[0]+nodes[0].Size[0], nodes[0].Pos[1]+nodes[0].Size[1]
	for _, node := range nodes[1:] {
		if node.Pos[0] < minX {
			minX = node.Pos[0]
		}
		if node.Pos[1] < minY {
			minY = node.Pos[1]
		}
		if node.Pos[0]+node.Size[0] > maxX {
			maxX = node.Pos[0] + node.Size[0]
		}
		if node.Pos[1]+node.Size[1] > maxY {
			maxY = node.Pos[1] + node.Size[1]
		}
	}

	return maxX - minX, maxY - minY
}

func translateWorkflowNodes(nodes []workspaceNode, xOffset, yOffset float64) []workspaceNode {
	translated := make([]workspaceNode, len(nodes))
	copy(translated, nodes)
	for index := range translated {
		translated[index].Pos = []float64{translated[index].Pos[0] + xOffset, translated[index].Pos[1] + yOffset}
	}
	return translated
}

func translateWorkflowGroup(group workspaceGroup, xOffset, yOffset float64) workspaceGroup {
	translated := group
	translated.Bounding = []float64{
		group.Bounding[0] + xOffset,
		group.Bounding[1] + yOffset,
		group.Bounding[2],
		group.Bounding[3],
	}
	return translated
}
