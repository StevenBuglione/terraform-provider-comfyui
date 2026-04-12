package resources

import (
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

func testWorkspaceNodeLayout() workspaceNodeLayoutConfig {
	return workspaceNodeLayoutConfig{
		Mode:      "dag",
		Direction: "left_to_right",
	}
}

func TestBuildWorkspaceSubgraphSingleWorkflow(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"demo-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "hello"}},
					"2": {"class_type": "TargetNode", "inputs": {"source": ["1", 0], "strength": 0.5}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	if subgraph.Name != "demo-workspace" {
		t.Fatalf("expected workspace name %q, got %q", "demo-workspace", subgraph.Name)
	}
	if len(subgraph.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(subgraph.Nodes))
	}
	if len(subgraph.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(subgraph.Links))
	}
	if subgraph.State.LastNodeID != 2 {
		t.Fatalf("expected last node id 2, got %d", subgraph.State.LastNodeID)
	}
	if subgraph.State.LastLinkID != 1 {
		t.Fatalf("expected last link id 1, got %d", subgraph.State.LastLinkID)
	}
	if subgraph.Nodes[0].Type != "SourceNode" {
		t.Fatalf("expected first node type %q, got %q", "SourceNode", subgraph.Nodes[0].Type)
	}
	if len(subgraph.Nodes[0].WidgetsValues) != 1 {
		t.Fatalf("expected only one widget-backed value, got %#v", subgraph.Nodes[0].WidgetsValues)
	}
	if subgraph.Nodes[0].WidgetsValues[0] != "hello" {
		t.Fatalf("expected first node widget value %q, got %#v", "hello", subgraph.Nodes[0].WidgetsValues[0])
	}
	if subgraph.Nodes[1].Inputs[0].Link != 1 {
		t.Fatalf("expected input link id 1, got %v", subgraph.Nodes[1].Inputs[0].Link)
	}
	if subgraph.Links[0].OriginID != 1 || subgraph.Links[0].TargetID != 2 {
		t.Fatalf("expected link to connect node 1 to node 2, got %+v", subgraph.Links[0])
	}
	if len(subgraph.Nodes[0].Outputs[0].Links) != 1 || subgraph.Nodes[0].Outputs[0].Links[0] != 1 {
		t.Fatalf("expected source node output to reference link 1, got %+v", subgraph.Nodes[0].Outputs[0].Links)
	}
	if len(subgraph.Groups) != 1 || subgraph.Groups[0].Title != "workflow-a" {
		t.Fatalf("expected workflow group title %q, got %+v", "workflow-a", subgraph.Groups)
	}
}

func TestBuildWorkspaceSubgraphRenumbersMultipleWorkflows(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"demo-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "alpha"}},
					"2": {"class_type": "TargetNode", "inputs": {"source": ["1", 0], "strength": 0.1}}
				}`,
			},
			{
				Name: "workflow-b",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "beta"}},
					"2": {"class_type": "TargetNode", "inputs": {"source": ["1", 0], "strength": 0.2}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
			Gap:       200,
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	if len(subgraph.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(subgraph.Nodes))
	}
	if len(subgraph.Links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(subgraph.Links))
	}
	if subgraph.State.LastNodeID != 4 {
		t.Fatalf("expected last node id 4, got %d", subgraph.State.LastNodeID)
	}
	if subgraph.State.LastLinkID != 2 {
		t.Fatalf("expected last link id 2, got %d", subgraph.State.LastLinkID)
	}
	if subgraph.Links[1].OriginID != 3 || subgraph.Links[1].TargetID != 4 {
		t.Fatalf("expected second workflow link to connect nodes 3 -> 4, got %+v", subgraph.Links[1])
	}
	if len(subgraph.Nodes[2].Outputs[0].Links) != 1 || subgraph.Nodes[2].Outputs[0].Links[0] != 2 {
		t.Fatalf("expected second workflow source node output to reference link 2, got %+v", subgraph.Nodes[2].Outputs[0].Links)
	}
	for i, node := range subgraph.Nodes {
		if node.Order != i {
			t.Fatalf("expected global node order %d, got %d for node %+v", i, node.Order, node)
		}
	}
	if subgraph.Nodes[2].Pos[0] <= subgraph.Nodes[0].Pos[0] {
		t.Fatalf("expected second workflow to be offset on the X axis, got first=%v second=%v", subgraph.Nodes[0].Pos, subgraph.Nodes[2].Pos)
	}
}

func TestBuildWorkspaceSubgraphGridLayoutUsesColumnsAndOrigin(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"grid-workspace",
		[]workspaceWorkflowSpec{
			{Name: "workflow-a", WorkflowJSON: `{"1": {"class_type": "SourceNode", "inputs": {"text": "alpha"}}}`},
			{Name: "workflow-b", WorkflowJSON: `{"1": {"class_type": "SourceNode", "inputs": {"text": "beta"}}}`},
			{Name: "workflow-c", WorkflowJSON: `{"1": {"class_type": "SourceNode", "inputs": {"text": "gamma"}}}`},
		},
		workspaceLayoutConfig{
			Display: "grid",
			Columns: 2,
			Gap:     180,
			OriginX: 50,
			OriginY: 25,
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	first := subgraph.Nodes[0].Pos
	second := subgraph.Nodes[1].Pos
	third := subgraph.Nodes[2].Pos

	if first[0] != 50 || first[1] != 25 {
		t.Fatalf("expected first workflow at origin [50 25], got %v", first)
	}
	if second[0] <= first[0] || second[1] != first[1] {
		t.Fatalf("expected second workflow on same row to the right of first, got first=%v second=%v", first, second)
	}
	if third[0] != first[0] || third[1] <= first[1] {
		t.Fatalf("expected third workflow to wrap to next row, got first=%v third=%v", first, third)
	}
}

func TestBuildWorkspaceSubgraphFlexColumnLayoutStacksVertically(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"column-workspace",
		[]workspaceWorkflowSpec{
			{Name: "workflow-a", WorkflowJSON: `{"1": {"class_type": "SourceNode", "inputs": {"text": "alpha"}}}`},
			{Name: "workflow-b", WorkflowJSON: `{"1": {"class_type": "SourceNode", "inputs": {"text": "beta"}}}`},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "column",
			Gap:       180,
			OriginX:   10,
			OriginY:   20,
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	first := subgraph.Nodes[0].Pos
	second := subgraph.Nodes[1].Pos

	if first[0] != 10 || first[1] != 20 {
		t.Fatalf("expected first workflow at [10 20], got %v", first)
	}
	if second[0] != first[0] || second[1] <= first[1] {
		t.Fatalf("expected second workflow to stack below first, got first=%v second=%v", first, second)
	}
}

func TestBuildWorkspaceSubgraphGridLayoutUsesWorkflowBoundsBetweenRows(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"grid-bounds-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "a"}},
					"2": {"class_type": "SourceNode", "inputs": {"text": "b"}},
					"3": {"class_type": "SourceNode", "inputs": {"text": "c"}},
					"4": {"class_type": "SourceNode", "inputs": {"text": "d"}},
					"5": {"class_type": "SourceNode", "inputs": {"text": "e"}},
					"6": {"class_type": "SourceNode", "inputs": {"text": "f"}}
				}`,
			},
			{
				Name:         "workflow-b",
				WorkflowJSON: `{"1": {"class_type": "SourceNode", "inputs": {"text": "beta"}}}`,
			},
			{
				Name:         "workflow-c",
				WorkflowJSON: `{"1": {"class_type": "SourceNode", "inputs": {"text": "gamma"}}}`,
			},
		},
		workspaceLayoutConfig{
			Display: "grid",
			Columns: 2,
			Gap:     180,
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	firstGroup := subgraph.Groups[0]
	thirdGroup := subgraph.Groups[2]
	firstGroupBottom := firstGroup.Bounding[1] + firstGroup.Bounding[3]
	expectedMinY := firstGroupBottom + 180

	if thirdGroup.Bounding[1] < expectedMinY {
		t.Fatalf("expected second grid row group to start at or below %v, got %v", expectedMinY, thirdGroup.Bounding[1])
	}
}

func TestBuildWorkspaceSubgraphFlexColumnUsesWorkflowBoundsBetweenItems(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"column-bounds-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "a"}},
					"2": {"class_type": "SourceNode", "inputs": {"text": "b"}},
					"3": {"class_type": "SourceNode", "inputs": {"text": "c"}},
					"4": {"class_type": "SourceNode", "inputs": {"text": "d"}},
					"5": {"class_type": "SourceNode", "inputs": {"text": "e"}}
				}`,
			},
			{
				Name:         "workflow-b",
				WorkflowJSON: `{"1": {"class_type": "SourceNode", "inputs": {"text": "beta"}}}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "column",
			Gap:       150,
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	firstGroup := subgraph.Groups[0]
	secondGroup := subgraph.Groups[1]
	firstGroupBottom := firstGroup.Bounding[1] + firstGroup.Bounding[3]
	expectedMinY := firstGroupBottom + 150

	if secondGroup.Bounding[1] < expectedMinY {
		t.Fatalf("expected second flex item group to start at or below %v, got %v", expectedMinY, secondGroup.Bounding[1])
	}
}

func testWorkspaceNodeInfo() map[string]client.NodeInfo {
	return map[string]client.NodeInfo{
		"SourceNode": {
			Input: client.NodeInputInfo{
				Required: map[string]interface{}{
					"text": []interface{}{"STRING", map[string]interface{}{}},
				},
				Optional: map[string]interface{}{
					"upstream": []interface{}{"LATENT", map[string]interface{}{}},
				},
			},
			InputOrder:   map[string][]string{"required": {"text"}, "optional": {"upstream"}},
			Output:       []string{"TEXT"},
			OutputName:   []string{"TEXT"},
			OutputIsList: []bool{false},
			Name:         "SourceNode",
			DisplayName:  "Source Node",
		},
		"TargetNode": {
			Input: client.NodeInputInfo{
				Required: map[string]interface{}{
					"source":   []interface{}{"TEXT", map[string]interface{}{}},
					"strength": []interface{}{"FLOAT", map[string]interface{}{}},
				},
			},
			InputOrder:   map[string][]string{"required": {"source", "strength"}},
			Output:       []string{"IMAGE"},
			OutputName:   []string{"IMAGE"},
			OutputIsList: []bool{false},
			Name:         "TargetNode",
			DisplayName:  "Target Node",
		},
		"MergeNode": {
			Input: client.NodeInputInfo{
				Required: map[string]interface{}{
					"source_a": []interface{}{"IMAGE", map[string]interface{}{}},
					"source_b": []interface{}{"IMAGE", map[string]interface{}{}},
					"strength": []interface{}{"FLOAT", map[string]interface{}{}},
				},
			},
			InputOrder:   map[string][]string{"required": {"source_a", "source_b", "strength"}},
			Output:       []string{"IMAGE"},
			OutputName:   []string{"IMAGE"},
			OutputIsList: []bool{false},
			Name:         "MergeNode",
			DisplayName:  "Merge Node",
		},
		"ComboNode": {
			Input: client.NodeInputInfo{
				Required: map[string]interface{}{
					"choice": []interface{}{
						[]interface{}{"alpha", "beta"},
						map[string]interface{}{},
					},
				},
			},
			InputOrder:   map[string][]string{"required": {"choice"}},
			Output:       []string{"STRING"},
			OutputName:   []string{"STRING"},
			OutputIsList: []bool{false},
			Name:         "ComboNode",
			DisplayName:  "Combo Node",
		},
		"NoOrderTargetNode": {
			Input: client.NodeInputInfo{
				Required: map[string]interface{}{
					"source":   []interface{}{"TEXT", map[string]interface{}{}},
					"strength": []interface{}{"FLOAT", map[string]interface{}{}},
				},
			},
			Output:       []string{"IMAGE"},
			OutputName:   []string{"IMAGE"},
			OutputIsList: []bool{false},
			Name:         "NoOrderTargetNode",
			DisplayName:  "No Order Target Node",
		},
		"BadOrderTargetNode": {
			Input: client.NodeInputInfo{
				Required: map[string]interface{}{
					"source":   []interface{}{"STRING", map[string]interface{}{}},
					"strength": []interface{}{"FLOAT", map[string]interface{}{}},
				},
			},
			InputOrder:   map[string][]string{"required": {"source", "source", "ghost", "strength"}},
			Output:       []string{"IMAGE"},
			OutputName:   []string{"IMAGE"},
			OutputIsList: []bool{false},
			Name:         "BadOrderTargetNode",
			DisplayName:  "Bad Order Target Node",
		},
		"MultiWidgetNode": {
			Input: client.NodeInputInfo{
				Required: map[string]interface{}{
					"prompt":   []interface{}{"STRING", map[string]interface{}{}},
					"strength": []interface{}{"FLOAT", map[string]interface{}{}},
				},
			},
			InputOrder:   map[string][]string{"required": {"prompt", "strength"}},
			Output:       []string{"STRING"},
			OutputName:   []string{"STRING"},
			OutputIsList: []bool{false},
			Name:         "MultiWidgetNode",
			DisplayName:  "Multi Widget Node",
		},
		"SamplerNode": {
			Input: client.NodeInputInfo{
				Required: map[string]interface{}{
					"model":        []interface{}{"MODEL", map[string]interface{}{}},
					"seed":         []interface{}{"INT", map[string]interface{}{}},
					"steps":        []interface{}{"INT", map[string]interface{}{}},
					"cfg":          []interface{}{"FLOAT", map[string]interface{}{}},
					"sampler_name": []interface{}{[]interface{}{"euler", "heun"}, map[string]interface{}{}},
					"scheduler":    []interface{}{[]interface{}{"normal", "karras"}, map[string]interface{}{}},
					"positive":     []interface{}{"CONDITIONING", map[string]interface{}{}},
					"negative":     []interface{}{"CONDITIONING", map[string]interface{}{}},
					"latent_image": []interface{}{"LATENT", map[string]interface{}{}},
					"denoise":      []interface{}{"FLOAT", map[string]interface{}{}},
				},
			},
			InputOrder:   map[string][]string{"required": {"model", "seed", "steps", "cfg", "sampler_name", "scheduler", "positive", "negative", "latent_image", "denoise"}},
			Output:       []string{"LATENT"},
			OutputName:   []string{"LATENT"},
			OutputIsList: []bool{false},
			Name:         "SamplerNode",
			DisplayName:  "Sampler Node",
		},
	}
}

func TestBuildWorkspaceSubgraphRejectsOutOfRangeOutputSlot(t *testing.T) {
	_, err := buildWorkspaceSubgraph(
		"broken-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "hello"}},
					"2": {"class_type": "TargetNode", "inputs": {"source": ["1", 5], "strength": 0.5}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err == nil {
		t.Fatalf("expected out-of-range output slot to return an error")
	}
}

func TestBuildWorkspaceSubgraphTreatsListMetadataInputsAsCombo(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"combo-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "ComboNode", "inputs": {"choice": "beta"}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	if subgraph.Nodes[0].Inputs[0].Type != "COMBO" {
		t.Fatalf("expected COMBO input type, got %q", subgraph.Nodes[0].Inputs[0].Type)
	}
}

func TestBuildWorkspaceSubgraphRejectsMissingInputOrder(t *testing.T) {
	_, err := buildWorkspaceSubgraph(
		"missing-order-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "hello"}},
					"2": {"class_type": "NoOrderTargetNode", "inputs": {"source": ["1", 0], "strength": 0.5}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err == nil {
		t.Fatalf("expected missing input_order to return an error")
	}
}

func TestBuildWorkspaceSubgraphRejectsInvalidInputOrderEntries(t *testing.T) {
	_, err := buildWorkspaceSubgraph(
		"bad-order-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "BadOrderTargetNode", "inputs": {"source": "hello", "strength": 0.5}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err == nil {
		t.Fatalf("expected invalid input_order entries to return an error")
	}
}

func TestBuildWorkspaceSubgraphPreservesWidgetAlignmentWhenRequiredWidgetMissing(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"widget-alignment-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "MultiWidgetNode", "inputs": {"strength": 0.5}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	values := subgraph.Nodes[0].WidgetsValues
	if len(values) != 2 {
		t.Fatalf("expected widget_values placeholders for missing required widgets, got %#v", values)
	}
	if values[0] != nil || values[1] != 0.5 {
		t.Fatalf("expected widget alignment [nil, 0.5], got %#v", values)
	}
}

func TestWorkspaceBuilderRespectsHeaderClearance(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"header-clearance-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "alpha"}},
					"2": {"class_type": "TargetNode", "inputs": {"source": ["1", 0], "strength": 0.5}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	group := subgraph.Groups[0]
	firstNode := subgraph.Nodes[0]

	minNodeY := group.Bounding[1] + 80
	if firstNode.Pos[1] < minNodeY {
		t.Fatalf("expected first node Y to be at least %v (group top %v + 80px header), got %v", minNodeY, group.Bounding[1], firstNode.Pos[1])
	}
}

func TestWorkspaceBuilderEnforcesLeftToRightDAGOrdering(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"dag-ordering-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "source"}},
					"2": {"class_type": "TargetNode", "inputs": {"source": ["1", 0], "strength": 0.3}},
					"3": {"class_type": "TargetNode", "inputs": {"source": ["1", 0], "strength": 0.7}},
					"4": {"class_type": "MergeNode", "inputs": {"source_a": ["2", 0], "source_b": ["3", 0], "strength": 0.5}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	// Node 1 is the source
	source := subgraph.Nodes[0]
	// Nodes 2 and 3 are both branches from source
	branch1 := subgraph.Nodes[1]
	branch2 := subgraph.Nodes[2]
	// Node 4 is the merge node (depends on both branches)
	merge := subgraph.Nodes[3]

	// Both branches should be to the right of the source
	if branch1.Pos[0] <= source.Pos[0] {
		t.Fatalf("expected branch1 (node 2) X position %v to be > source (node 1) X position %v", branch1.Pos[0], source.Pos[0])
	}
	if branch2.Pos[0] <= source.Pos[0] {
		t.Fatalf("expected branch2 (node 3) X position %v to be > source (node 1) X position %v", branch2.Pos[0], source.Pos[0])
	}

	// Merge node should be to the right of both branches
	if merge.Pos[0] <= branch1.Pos[0] {
		t.Fatalf("expected merge (node 4) X position %v to be > branch1 (node 2) X position %v", merge.Pos[0], branch1.Pos[0])
	}
	if merge.Pos[0] <= branch2.Pos[0] {
		t.Fatalf("expected merge (node 4) X position %v to be > branch2 (node 3) X position %v", merge.Pos[0], branch2.Pos[0])
	}
}

func TestWorkspaceBuilderEnforcesNodeContainment(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"containment-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "alpha"}},
					"2": {"class_type": "SourceNode", "inputs": {"text": "beta"}},
					"3": {"class_type": "SourceNode", "inputs": {"text": "gamma"}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	group := subgraph.Groups[0]
	minY := group.Bounding[1] + 80

	for i, node := range subgraph.Nodes {
		if node.Pos[1] < minY {
			t.Fatalf("node %d intrudes into header area: Y position %v is below minimum %v (group top %v + 80)", i, node.Pos[1], minY, group.Bounding[1])
		}
	}
}

func TestWorkspaceBuilderSerializesWorkflowStyle(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"style-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "styled"}}
				}`,
				Style: workspaceWorkflowStyleConfig{
					GroupColor:    "#ff00ff",
					TitleFontSize: 28,
				},
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	group := subgraph.Groups[0]
	if group.Color != "#ff00ff" {
		t.Fatalf("expected group color %q, got %q", "#ff00ff", group.Color)
	}
	if group.FontSize != 28 {
		t.Fatalf("expected group font size %d, got %d", 28, group.FontSize)
	}
}

func TestWorkspaceBuilderExpandsHeaderForLargeFontSize(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"large-font-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "styled"}}
				}`,
				Style: workspaceWorkflowStyleConfig{
					TitleFontSize: 60,
				},
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	group := subgraph.Groups[0]
	firstNode := subgraph.Nodes[0]
	expectedClearance := 130.0 // max(40, 60*1.5) + 40 body padding
	if firstNode.Pos[1]-group.Bounding[1] < expectedClearance {
		t.Fatalf("expected at least %vpx of clearance for large title font, got %v", expectedClearance, firstNode.Pos[1]-group.Bounding[1])
	}
}

func TestWorkspaceBuilderDetectsCycles(t *testing.T) {
	_, err := buildWorkspaceSubgraph(
		"cyclic-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "hello", "upstream": ["2", 0]}},
					"2": {"class_type": "TargetNode", "inputs": {"source": ["1", 0], "strength": 0.5}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err == nil {
		t.Fatalf("expected cycle detection to return an error")
	}
}

func TestWorkspaceBuilderRoundsMergeRowAnchor(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"merge-row-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "top"}},
					"2": {"class_type": "SourceNode", "inputs": {"text": "bottom"}},
					"3": {"class_type": "MergeNode", "inputs": {"source_a": ["1", 0], "source_b": ["2", 0], "strength": 0.5}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	// Nodes 1 and 2 are sources in column 0, rows 0 and 1.
	// Their average row anchor is 0.5, which must round to row 1.
	node2 := subgraph.Nodes[1]
	merge := subgraph.Nodes[2]

	if merge.Pos[1] != node2.Pos[1] {
		t.Fatalf("expected merge node at rounded parent row %v, got %v", node2.Pos[1], merge.Pos[1])
	}
}

func TestWorkspaceBuilderHonorsPreferredRowInColumnSorting(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"preferred-row-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "parent-a"}},
					"2": {"class_type": "SourceNode", "inputs": {"text": "parent-b"}},
					"3": {"class_type": "TargetNode", "inputs": {"source": ["2", 0], "strength": 0.2}},
					"4": {"class_type": "TargetNode", "inputs": {"source": ["1", 0], "strength": 0.1}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	// Node 1 is parent-a (row 0), Node 2 is parent-b (row 1)
	// Node 3 (prompt order 2) depends on node 2 -> preferred row 1
	// Node 4 (prompt order 3) depends on node 1 -> preferred row 0
	// Even though node 3 comes first in prompt order, node 4 should be placed first (row 0)
	// because preferred row takes precedence over prompt order
	node1 := subgraph.Nodes[0] // parent-a, row 0
	node2 := subgraph.Nodes[1] // parent-b, row 1
	node4 := subgraph.Nodes[3] // depends on node 1
	node3 := subgraph.Nodes[2] // depends on node 2

	// Node 4 should be above node 3 (smaller Y) because its preferred row is 0 < 1
	if node4.Pos[1] >= node3.Pos[1] {
		t.Fatalf("expected node 4 (preferred row 0) to be above node 3 (preferred row 1), got Y positions: node4=%v, node3=%v", node4.Pos[1], node3.Pos[1])
	}

	// Verify node 4 is at parent-a's row
	if node4.Pos[1] != node1.Pos[1] {
		t.Fatalf("expected node 4 at same Y as parent (node 1): %v, got %v", node1.Pos[1], node4.Pos[1])
	}

	// Verify node 3 is at parent-b's row
	if node3.Pos[1] != node2.Pos[1] {
		t.Fatalf("expected node 3 at same Y as parent (node 2): %v, got %v", node2.Pos[1], node3.Pos[1])
	}
}

func TestBuildWorkspaceNodeExpandsHeightForDenseControlNodes(t *testing.T) {
	info := testWorkspaceNodeInfo()["SamplerNode"]
	orderedInputs, err := orderedNodeInputs(info)
	if err != nil {
		t.Fatalf("orderedNodeInputs returned error: %v", err)
	}

	node, _ := buildWorkspaceNode(
		1,
		"SamplerNode",
		info,
		orderedInputs,
		map[string]interface{}{
			"seed":         7,
			"steps":        20,
			"cfg":          7.0,
			"sampler_name": "euler",
			"scheduler":    "normal",
			"denoise":      1.0,
		},
		0,
		0,
		0,
		0,
	)

	if node.Size[1] != 262 {
		t.Fatalf("expected dense control node height 262, got %v", node.Size[1])
	}
}

func TestBuildWorkspaceNodeHonorsGeneratedUIHints(t *testing.T) {
	info := client.NodeInfo{
		Input: client.NodeInputInfo{
			Required: map[string]interface{}{
				"text": []interface{}{"STRING", map[string]interface{}{
					"multiline": true,
				}},
				"clip": []interface{}{"CLIP", map[string]interface{}{}},
			},
		},
		InputOrder:   map[string][]string{"required": {"text", "clip"}},
		Output:       []string{"CONDITIONING"},
		OutputName:   []string{"CONDITIONING"},
		OutputIsList: []bool{false},
		Name:         "CLIPTextEncode",
		DisplayName:  "CLIP Text Encode (Prompt)",
	}

	orderedInputs, err := orderedNodeInputs(info)
	if err != nil {
		t.Fatalf("orderedNodeInputs returned error: %v", err)
	}

	node, _ := buildWorkspaceNode(
		1,
		"CLIPTextEncode",
		info,
		orderedInputs,
		map[string]interface{}{
			"text": "hero product shot, reflective chrome, dramatic studio rim light",
			"clip": []interface{}{"2", 0},
		},
		0,
		0,
		0,
		0,
	)

	if node.Size[0] != 400 {
		t.Fatalf("expected multiline widget min width 400, got %v", node.Size[0])
	}
	if node.Size[1] != 200 {
		t.Fatalf("expected multiline widget min height 200, got %v", node.Size[1])
	}
}

func TestBuildWorkspaceNodeFallsBackWithoutGeneratedUIHints(t *testing.T) {
	info := client.NodeInfo{
		Input: client.NodeInputInfo{
			Required: map[string]interface{}{
				"text": []interface{}{"STRING", map[string]interface{}{
					"multiline": true,
				}},
			},
		},
		InputOrder:   map[string][]string{"required": {"text"}},
		Output:       []string{"STRING"},
		OutputName:   []string{"STRING"},
		OutputIsList: []bool{false},
		Name:         "UnknownCustomNode",
		DisplayName:  "Unknown Custom Node",
	}

	orderedInputs, err := orderedNodeInputs(info)
	if err != nil {
		t.Fatalf("orderedNodeInputs returned error: %v", err)
	}

	node, _ := buildWorkspaceNode(1, "UnknownCustomNode", info, orderedInputs, map[string]interface{}{"text": "hello"}, 0, 0, 0, 0)

	if node.Size[0] != defaultNodeWidth {
		t.Fatalf("expected fallback width %v, got %v", defaultNodeWidth, node.Size[0])
	}
	expectedHeight := estimateWorkspaceNodeHeight(len(orderedInputs), len(info.Output))
	if node.Size[1] != expectedHeight {
		t.Fatalf("expected fallback height %v, got %v", expectedHeight, node.Size[1])
	}
}

func TestWorkspaceBuilderUsesActualNodeHeightsForVerticalSpacing(t *testing.T) {
	nodes := []workspaceNode{
		{ID: 1, Type: "SamplerNode", Pos: []float64{0, 0}, Size: []float64{240, 262}},
		{ID: 2, Type: "VAEDecode", Pos: []float64{0, 0}, Size: []float64{240, 120}},
		{ID: 3, Type: "SaveImage", Pos: []float64{0, 0}, Size: []float64{240, 120}},
	}
	nodeIndexByID := map[int]int{1: 0, 2: 1, 3: 2}

	if err := layoutWorkflowNodesLeftToRight(nodes, nodeIndexByID, nil, testWorkspaceNodeLayout()); err != nil {
		t.Fatalf("layoutWorkflowNodesLeftToRight returned error: %v", err)
	}

	const expectedNodePadding = 40.0
	if nodes[1].Pos[1] < nodes[0].Pos[1]+nodes[0].Size[1]+expectedNodePadding {
		t.Fatalf("expected second node below first node bottom + padding, got first=%v second=%v", nodes[0], nodes[1])
	}
	if nodes[2].Pos[1] < nodes[1].Pos[1]+nodes[1].Size[1]+expectedNodePadding {
		t.Fatalf("expected third node below second node bottom + padding, got second=%v third=%v", nodes[1], nodes[2])
	}
}

func TestWorkspaceBuilderTreatsConfiguredRowGapAsEmptySpace(t *testing.T) {
	nodes := []workspaceNode{
		{ID: 1, Type: "SourceNode", Pos: []float64{0, 0}, Size: []float64{240, 120}},
		{ID: 2, Type: "SourceNode", Pos: []float64{0, 0}, Size: []float64{240, 120}},
	}
	nodeIndexByID := map[int]int{1: 0, 2: 1}

	if err := layoutWorkflowNodesLeftToRight(nodes, nodeIndexByID, nil, workspaceNodeLayoutConfig{
		Mode:      "dag",
		Direction: "left_to_right",
		RowGap:    120,
	}); err != nil {
		t.Fatalf("layoutWorkflowNodesLeftToRight returned error: %v", err)
	}

	if gap := nodes[1].Pos[1] - (nodes[0].Pos[1] + nodes[0].Size[1]); gap != 120 {
		t.Fatalf("expected configured row_gap to create 120px empty space, got %v", gap)
	}
}

func TestWorkspaceBuilderUsesActualNodeWidthsForHorizontalSpacing(t *testing.T) {
	nodes := []workspaceNode{
		{ID: 1, Type: "WideNode", Pos: []float64{0, 0}, Size: []float64{400, 200}},
		{ID: 2, Type: "TargetNode", Pos: []float64{0, 0}, Size: []float64{240, 120}},
	}
	nodeIndexByID := map[int]int{1: 0, 2: 1}
	links := []workspaceLink{
		{ID: 1, OriginID: 1, OriginSlot: 0, TargetID: 2, TargetSlot: 0, Type: "TEXT"},
	}

	if err := layoutWorkflowNodesLeftToRight(nodes, nodeIndexByID, links, workspaceNodeLayoutConfig{
		Mode:      "dag",
		Direction: "left_to_right",
		ColumnGap: 40,
	}); err != nil {
		t.Fatalf("layoutWorkflowNodesLeftToRight returned error: %v", err)
	}

	if gap := nodes[1].Pos[0] - (nodes[0].Pos[0] + nodes[0].Size[0]); gap != 40 {
		t.Fatalf("expected configured column_gap to create 40px empty space after actual node width, got %v", gap)
	}
}

func TestWorkspaceBuilderDefaultsToReadableColumnGap(t *testing.T) {
	subgraph, err := buildWorkspaceSubgraph(
		"column-gap-workspace",
		[]workspaceWorkflowSpec{
			{
				Name: "workflow-a",
				WorkflowJSON: `{
					"1": {"class_type": "SourceNode", "inputs": {"text": "hello"}},
					"2": {"class_type": "TargetNode", "inputs": {"source": ["1", 0], "strength": 0.5}}
				}`,
			},
		},
		workspaceLayoutConfig{
			Display:   "flex",
			Direction: "row",
		},
		testWorkspaceNodeLayout(),
		testWorkspaceNodeInfo(),
	)
	if err != nil {
		t.Fatalf("buildWorkspaceSubgraph returned error: %v", err)
	}

	if gap := subgraph.Nodes[1].Pos[0] - subgraph.Nodes[0].Pos[0]; gap < 280 {
		t.Fatalf("expected default column gap to be at least 280, got %v", gap)
	}
}

func TestWorkspaceBuilderDefaultsToReadableRowGap(t *testing.T) {
	nodes := []workspaceNode{
		{ID: 1, Type: "SourceNode", Pos: []float64{0, 0}, Size: []float64{240, 120}},
		{ID: 2, Type: "SourceNode", Pos: []float64{0, 0}, Size: []float64{240, 120}},
	}
	nodeIndexByID := map[int]int{1: 0, 2: 1}

	if err := layoutWorkflowNodesLeftToRight(nodes, nodeIndexByID, nil, testWorkspaceNodeLayout()); err != nil {
		t.Fatalf("layoutWorkflowNodesLeftToRight returned error: %v", err)
	}

	if gap := nodes[1].Pos[1] - (nodes[0].Pos[1] + nodes[0].Size[1]); gap < 40 {
		t.Fatalf("expected default row gap to leave at least 40px edge padding, got %v", gap)
	}
}
