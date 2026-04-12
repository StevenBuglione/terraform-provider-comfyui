package resources

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

const (
	defaultWorkflowGap       = 320.0
	defaultNodeYSpacing      = 140.0
	defaultNodeWidth         = 240.0
	defaultNodeHeight        = 120.0
	defaultNodeEdgePadding   = 40.0
	nodeHeightBase           = 62.0
	nodeHeightPerRow         = 20.0
	defaultGroupHeaderHeight = 40.0
	defaultGroupBodyTopPad   = 40.0
	defaultNodeColumnGap     = defaultNodeEdgePadding
	defaultNodeRowGap        = defaultNodeEdgePadding
	defaultGroupFontSize     = 24
)

type workspaceWorkflowSpec struct {
	Name         string
	WorkflowJSON string
	X            *float64
	Y            *float64
	Style        workspaceWorkflowStyleConfig
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

func buildWorkspaceSubgraph(name string, workflows []workspaceWorkflowSpec, layout workspaceLayoutConfig, nodeLayout workspaceNodeLayoutConfig, nodeInfo map[string]client.NodeInfo) (*workspaceSubgraph, error) {
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
			// Build nodes at (0,0) temporarily; layoutWorkflowNodesLeftToRight will position them
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

		// Apply DAG-based left-to-right layout with configurable spacing
		if err := layoutWorkflowNodesLeftToRight(workflowNodes, nodeIndexByID, workflowLinks, nodeLayout); err != nil {
			return nil, fmt.Errorf("workflow %q: %w", workflow.Name, err)
		}

		var group *workspaceGroup
		if workflow.Name != "" && len(workflowNodeIDs) > 0 {
			builtGroup := buildWorkspaceGroup(workflowIndex+1, workflow.Name, workflowNodeIDs, nodeIndexByID, workflowNodes, workflow.Style)
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

// layoutWorkflowNodesLeftToRight positions nodes in a left-to-right DAG layout.
// Uses longest-upstream-path for column assignment and average-parent-row for vertical placement.
func layoutWorkflowNodesLeftToRight(nodes []workspaceNode, nodeIndexByID map[int]int, links []workspaceLink, nodeLayout workspaceNodeLayoutConfig) error {
	if len(nodes) == 0 {
		return nil
	}

	// Build dependency graph
	upstreamLinks := make(map[int][]int) // node ID -> list of upstream node IDs
	for _, link := range links {
		upstreamLinks[link.TargetID] = append(upstreamLinks[link.TargetID], link.OriginID)
	}

	// Compute longest upstream path (column level) for each node
	// Use cycle detection to prevent stack overflow
	levels := make(map[int]int)
	visiting := make(map[int]bool)
	var computeLevel func(nodeID int) (int, error)
	computeLevel = func(nodeID int) (int, error) {
		if level, ok := levels[nodeID]; ok {
			return level, nil
		}
		if visiting[nodeID] {
			return 0, fmt.Errorf("cycle detected in workflow graph at node %d", nodeID)
		}
		visiting[nodeID] = true
		defer func() { visiting[nodeID] = false }()

		maxUpstreamLevel := -1
		for _, upstreamID := range upstreamLinks[nodeID] {
			upstreamLevel, err := computeLevel(upstreamID)
			if err != nil {
				return 0, err
			}
			if upstreamLevel > maxUpstreamLevel {
				maxUpstreamLevel = upstreamLevel
			}
		}
		levels[nodeID] = maxUpstreamLevel + 1
		return levels[nodeID], nil
	}

	for _, node := range nodes {
		if _, err := computeLevel(node.ID); err != nil {
			return err
		}
	}

	// Group nodes by column (level)
	columnNodes := make(map[int][]int)
	for nodeID, level := range levels {
		columnNodes[level] = append(columnNodes[level], nodeID)
	}

	// Find max level
	maxLevel := 0
	for level := range columnNodes {
		if level > maxLevel {
			maxLevel = level
		}
	}

	// Assign row positions within columns
	// Process columns left to right
	nodeRows := make(map[int]float64)
	preferredRows := make(map[int]float64)
	columnMaxRow := make(map[int]float64)

	for level := 0; level <= maxLevel; level++ {
		nodeIDs, ok := columnNodes[level]
		if !ok {
			continue
		}

		// Compute preferred rows for this column
		for _, nodeID := range nodeIDs {
			parents := upstreamLinks[nodeID]
			if len(parents) > 0 {
				// Average parent row, rounded
				totalRow := 0.0
				for _, parentID := range parents {
					totalRow += nodeRows[parentID]
				}
				preferredRows[nodeID] = math.Round(totalRow / float64(len(parents)))
			} else {
				// Use original node index as preferred row for source nodes
				preferredRows[nodeID] = float64(nodeIndexByID[nodeID])
			}
		}

		// Sort nodes within this column by preferred row, then original prompt order
		sort.SliceStable(nodeIDs, func(i, j int) bool {
			prefI := preferredRows[nodeIDs[i]]
			prefJ := preferredRows[nodeIDs[j]]
			if prefI != prefJ {
				return prefI < prefJ
			}
			// Tie-breaker: original node index (prompt order)
			indexI := nodeIndexByID[nodeIDs[i]]
			indexJ := nodeIndexByID[nodeIDs[j]]
			return indexI < indexJ
		})
		columnNodes[level] = nodeIDs

		// Assign final row positions with collision resolution
		usedRows := make(map[float64]bool)
		for _, nodeID := range nodeIDs {
			// Use precomputed preferred row
			preferredRow := preferredRows[nodeID]

			// Find first available row at or below preferred row
			finalRow := preferredRow
			for usedRows[finalRow] {
				finalRow++
			}
			nodeRows[nodeID] = finalRow
			usedRows[finalRow] = true

			if finalRow > columnMaxRow[level] {
				columnMaxRow[level] = finalRow
			}
		}
	}

	// Apply spacing configuration
	columnGap := nodeLayout.ColumnGap
	if columnGap <= 0 {
		columnGap = defaultNodeColumnGap
	}
	rowGap := nodeLayout.RowGap
	if rowGap <= 0 {
		rowGap = defaultNodeRowGap
	}

	rowHeights := make(map[int]float64)
	columnWidths := make(map[int]float64)
	maxRow := 0
	maxLevelSeen := 0
	for _, node := range nodes {
		row := int(nodeRows[node.ID])
		if node.Size[1] > rowHeights[row] {
			rowHeights[row] = node.Size[1]
		}
		level := levels[node.ID]
		if node.Size[0] > columnWidths[level] {
			columnWidths[level] = node.Size[0]
		}
		if row > maxRow {
			maxRow = row
		}
		if level > maxLevelSeen {
			maxLevelSeen = level
		}
	}

	rowOffsets := make(map[int]float64, maxRow+1)
	currentY := 0.0
	for row := 0; row <= maxRow; row++ {
		rowOffsets[row] = currentY
		height := rowHeights[row]
		if height == 0 {
			height = defaultNodeHeight
		}
		currentY += height + rowGap
	}

	columnOffsets := make(map[int]float64, maxLevelSeen+1)
	currentX := 0.0
	for level := 0; level <= maxLevelSeen; level++ {
		columnOffsets[level] = currentX
		width := columnWidths[level]
		if width == 0 {
			width = defaultNodeWidth
		}
		currentX += width + columnGap
	}

	// Position nodes
	for _, node := range nodes {
		level := levels[node.ID]
		row := int(nodeRows[node.ID])

		x := columnOffsets[level]
		y := rowOffsets[row]

		nodes[nodeIndexByID[node.ID]].Pos = []float64{x, y}
	}

	return nil
}

func buildWorkspaceNode(nodeID int, classType string, info client.NodeInfo, orderedInputs []string, rawInputs map[string]interface{}, baseX, baseY float64, nodeIndex int, order int) (workspaceNode, []interface{}) {
	width, height := estimateWorkspaceNodeSize(classType, info, orderedInputs)
	node := workspaceNode{
		ID:         nodeID,
		Type:       classType,
		Pos:        []float64{baseX, baseY},
		Size:       []float64{width, height},
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

func estimateWorkspaceNodeSize(classType string, info client.NodeInfo, orderedInputs []string) (float64, float64) {
	width := defaultNodeWidth
	height := estimateWorkspaceNodeHeight(len(orderedInputs), len(info.Output))

	if hint, ok := generatedNodeUIHints[classType]; ok {
		if hint.MinWidth > width {
			width = hint.MinWidth
		}
		if hint.MinHeight > height {
			height = hint.MinHeight
		}
	}

	for _, inputName := range orderedInputs {
		if inputType := lookupNodeInputType(info, inputName); !isWidgetInputType(inputType) {
			continue
		}
		minWidth, minHeight := widgetMinimumNodeSize(classType, inputName)
		if minWidth > width {
			width = minWidth
		}
		if minHeight > height {
			height = minHeight
		}
	}

	return width, height
}

func estimateWorkspaceNodeHeight(inputCount, outputCount int) float64 {
	rowCount := inputCount
	if outputCount > rowCount {
		rowCount = outputCount
	}

	height := nodeHeightBase + float64(rowCount)*nodeHeightPerRow
	if height < defaultNodeHeight {
		return defaultNodeHeight
	}

	return height
}

func widgetMinimumNodeSize(classType string, inputName string) (float64, float64) {
	hint, ok := generatedNodeUIHints[classType]
	if !ok {
		return 0, 0
	}
	widgetHint, ok := hint.Widgets[inputName]
	if !ok {
		return 0, 0
	}

	return widgetHint.MinNodeWidth, widgetHint.MinNodeHeight
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

func buildWorkspaceGroup(groupID int, title string, workflowNodeIDs []int, nodeIndexByID map[int]int, nodes []workspaceNode, style workspaceWorkflowStyleConfig) workspaceGroup {
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

	headerHeight := defaultGroupHeaderHeight
	if style.TitleFontSize > 0 {
		computedHeaderHeight := float64(style.TitleFontSize) * 1.5
		if computedHeaderHeight > headerHeight {
			headerHeight = computedHeaderHeight
		}
	}
	bodyTopPad := defaultGroupBodyTopPad
	sidePad := 40.0
	bottomPad := 40.0

	// Group bounding box starts above the first node by header + body padding
	groupTop := minY - headerHeight - bodyTopPad
	groupLeft := minX - sidePad
	groupWidth := (maxX - minX) + sidePad*2
	groupHeight := (maxY - groupTop) + bottomPad

	group := workspaceGroup{
		ID:       groupID,
		Title:    title,
		Bounding: []float64{groupLeft, groupTop, groupWidth, groupHeight},
		Flags:    map[string]interface{}{},
	}

	// Apply style fields
	if style.GroupColor != "" {
		group.Color = style.GroupColor
	}
	if style.TitleFontSize > 0 {
		group.FontSize = style.TitleFontSize
	} else if style.TitleFontSize == 0 {
		// Use default font size
		group.FontSize = defaultGroupFontSize
	}

	return group
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
