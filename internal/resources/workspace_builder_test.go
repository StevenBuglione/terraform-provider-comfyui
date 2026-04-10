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
