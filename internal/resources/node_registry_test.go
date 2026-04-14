package resources

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRegisterNodeStateFromModel_StoresInputsOnly(t *testing.T) {
	resetNodeStateRegistry()

	model := struct {
		ID          types.String `tfsdk:"id"`
		NodeID      types.String `tfsdk:"node_id"`
		CkptName    types.String `tfsdk:"ckpt_name"`
		ModelOutput types.String `tfsdk:"model_output"`
		CLIPOutput  types.String `tfsdk:"clip_output"`
		VAEOutput   types.String `tfsdk:"vae_output"`
	}{
		ID:          types.StringValue(uuidCheckpoint),
		NodeID:      types.StringValue("CheckpointLoaderSimple"),
		CkptName:    types.StringValue("v1-5-pruned-emaonly.safetensors"),
		ModelOutput: types.StringValue(uuidCheckpoint + ":0"),
		CLIPOutput:  types.StringValue(uuidCheckpoint + ":1"),
		VAEOutput:   types.StringValue(uuidCheckpoint + ":2"),
	}

	if err := RegisterNodeStateFromModel(model.ID.ValueString(), model.NodeID.ValueString(), model); err != nil {
		t.Fatalf("RegisterNodeStateFromModel returned error: %v", err)
	}

	got, ok := LookupNodeState(model.ID.ValueString())
	if !ok {
		t.Fatal("expected registered node state to be present")
	}

	if got.ClassType != "CheckpointLoaderSimple" {
		t.Fatalf("ClassType = %q, want %q", got.ClassType, "CheckpointLoaderSimple")
	}

	if got.Inputs["ckpt_name"] != "v1-5-pruned-emaonly.safetensors" {
		t.Fatalf("ckpt_name = %#v, want model filename", got.Inputs["ckpt_name"])
	}

	if _, exists := got.Inputs["model_output"]; exists {
		t.Fatal("model_output should not be stored as an input")
	}
}

func TestAssembleWorkflowFromNodeIDs_UsesRegisteredNodeState(t *testing.T) {
	resetNodeStateRegistry()

	nodes := fullPipeline()
	for _, node := range nodes {
		RegisterNodeState(node)
	}

	assembled, err := AssembleWorkflowFromNodeIDs([]string{
		uuidCheckpoint,
		uuidCLIP,
		uuidKSampler,
		uuidVAEDecode,
		uuidSaveImage,
	})
	if err != nil {
		t.Fatalf("AssembleWorkflowFromNodeIDs returned error: %v", err)
	}

	if len(assembled.Prompt) != 5 {
		t.Fatalf("assembled prompt has %d nodes, want 5", len(assembled.Prompt))
	}

	node3, ok := assembled.Prompt["3"].(map[string]interface{})
	if !ok {
		t.Fatal("expected node 3 in assembled prompt")
	}
	inputs, ok := node3["inputs"].(map[string]interface{})
	if !ok {
		t.Fatal("expected node 3 inputs map")
	}
	modelRef, ok := inputs["model"].([]interface{})
	if !ok {
		t.Fatalf("model ref type = %T, want []interface{}", inputs["model"])
	}
	if modelRef[0] != "1" || modelRef[1] != 0 {
		t.Fatalf("model ref = %#v, want [\"1\", 0]", modelRef)
	}
}

func TestNodeDefinitionJSONFromModel_SerializesNodeState(t *testing.T) {
	resetNodeStateRegistry()

	model := struct {
		ID                   types.String `tfsdk:"id"`
		NodeID               types.String `tfsdk:"node_id"`
		CkptName             types.String `tfsdk:"ckpt_name"`
		ControlAfterGenerate types.String `tfsdk:"control_after_generate"`
		ModelOutput          types.String `tfsdk:"model_output"`
	}{
		ID:                   types.StringValue(uuidCheckpoint),
		NodeID:               types.StringValue("CheckpointLoaderSimple"),
		CkptName:             types.StringValue("v1-5-pruned-emaonly.safetensors"),
		ControlAfterGenerate: types.StringValue("fixed"),
		ModelOutput:          types.StringValue(uuidCheckpoint + ":0"),
	}

	raw, err := NodeDefinitionJSONFromModel(model.ID.ValueString(), model.NodeID.ValueString(), model)
	if err != nil {
		t.Fatalf("NodeDefinitionJSONFromModel returned error: %v", err)
	}

	var got NodeState
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("node definition JSON should be valid JSON: %v", err)
	}

	if got.ID != uuidCheckpoint {
		t.Fatalf("ID = %q, want %q", got.ID, uuidCheckpoint)
	}
	if got.ClassType != "CheckpointLoaderSimple" {
		t.Fatalf("ClassType = %q, want %q", got.ClassType, "CheckpointLoaderSimple")
	}
	if got.Inputs["ckpt_name"] != "v1-5-pruned-emaonly.safetensors" {
		t.Fatalf("ckpt_name = %#v, want model filename", got.Inputs["ckpt_name"])
	}
	if got.Inputs["control_after_generate"] != "fixed" {
		t.Fatalf("control_after_generate = %#v, want %q", got.Inputs["control_after_generate"], "fixed")
	}
	if _, exists := got.Inputs["model_output"]; exists {
		t.Fatal("model_output should not be serialized as an input")
	}
}

func TestNodeDefinitionJSONFromModel_ExcludesNodeDefinitionJSONInput(t *testing.T) {
	model := struct {
		ID                 types.String `tfsdk:"id"`
		NodeID             types.String `tfsdk:"node_id"`
		CkptName           types.String `tfsdk:"ckpt_name"`
		NodeDefinitionJSON types.String `tfsdk:"node_definition_json"`
	}{
		ID:                 types.StringValue(uuidCheckpoint),
		NodeID:             types.StringValue("CheckpointLoaderSimple"),
		CkptName:           types.StringValue("v1-5-pruned-emaonly.safetensors"),
		NodeDefinitionJSON: types.StringValue(`{"id":"old"}`),
	}

	raw, err := NodeDefinitionJSONFromModel(model.ID.ValueString(), model.NodeID.ValueString(), model)
	if err != nil {
		t.Fatalf("NodeDefinitionJSONFromModel returned error: %v", err)
	}

	var got NodeState
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("node definition JSON should be valid JSON: %v", err)
	}
	if _, exists := got.Inputs["node_definition_json"]; exists {
		t.Fatal("node_definition_json should not be serialized as an input")
	}
}

func TestAssembleWorkflowFromNodeIDsWithDefinitions_UsesFallbackWhenRegistryEmpty(t *testing.T) {
	resetNodeStateRegistry()

	nodes := fullPipeline()
	ids := []string{
		uuidCheckpoint,
		uuidCLIP,
		uuidKSampler,
		uuidVAEDecode,
		uuidSaveImage,
	}
	definitions := make([]string, 0, len(nodes))
	for _, node := range nodes {
		raw, err := json.Marshal(node)
		if err != nil {
			t.Fatalf("failed to marshal node definition: %v", err)
		}
		definitions = append(definitions, string(raw))
	}

	assembled, err := AssembleWorkflowFromNodeIDsWithDefinitions(ids, definitions)
	if err != nil {
		t.Fatalf("AssembleWorkflowFromNodeIDsWithDefinitions returned error: %v", err)
	}

	if len(assembled.Prompt) != 5 {
		t.Fatalf("assembled prompt has %d nodes, want 5", len(assembled.Prompt))
	}

	node3, ok := assembled.Prompt["3"].(map[string]interface{})
	if !ok {
		t.Fatal("expected node 3 in assembled prompt")
	}
	inputs, ok := node3["inputs"].(map[string]interface{})
	if !ok {
		t.Fatal("expected node 3 inputs map")
	}
	modelRef, ok := inputs["model"].([]interface{})
	if !ok {
		t.Fatalf("model ref type = %T, want []interface{}", inputs["model"])
	}
	if modelRef[0] != "1" || modelRef[1] != 0 {
		t.Fatalf("model ref = %#v, want [\"1\", 0]", modelRef)
	}
}

func TestAssembleWorkflowFromNodeIDsWithDefinitions_RejectsMismatchedPairingEvenWhenRegistryWarm(t *testing.T) {
	resetNodeStateRegistry()

	nodes := fullPipeline()
	for _, node := range nodes {
		RegisterNodeState(node)
	}

	swappedDefinitions := make([]string, len(nodes))
	for i, node := range nodes {
		target := node
		switch i {
		case 0:
			target = nodes[1]
		case 1:
			target = nodes[0]
		}

		raw, err := json.Marshal(target)
		if err != nil {
			t.Fatalf("failed to marshal node definition: %v", err)
		}
		swappedDefinitions[i] = string(raw)
	}

	_, err := AssembleWorkflowFromNodeIDsWithDefinitions([]string{
		uuidCheckpoint,
		uuidCLIP,
		uuidKSampler,
		uuidVAEDecode,
		uuidSaveImage,
	}, swappedDefinitions)
	if err == nil {
		t.Fatal("expected mismatched paired node definition JSON to fail")
	}
	if !strings.Contains(err.Error(), `node_definition_jsons[0] id "`) {
		t.Fatalf("expected pairing mismatch error, got %v", err)
	}
}

func TestAssembleWorkflowFromNodeIDsWithDefinitions_UsesMixedRegistryAndFallback(t *testing.T) {
	resetNodeStateRegistry()

	nodes := fullPipeline()
	RegisterNodeState(nodes[0])
	RegisterNodeState(nodes[1])

	definitions := make([]string, 0, len(nodes))
	for _, node := range nodes {
		raw, err := json.Marshal(node)
		if err != nil {
			t.Fatalf("failed to marshal node definition: %v", err)
		}
		definitions = append(definitions, string(raw))
	}

	assembled, err := AssembleWorkflowFromNodeIDsWithDefinitions([]string{
		uuidCheckpoint,
		uuidCLIP,
		uuidKSampler,
		uuidVAEDecode,
		uuidSaveImage,
	}, definitions)
	if err != nil {
		t.Fatalf("AssembleWorkflowFromNodeIDsWithDefinitions returned error: %v", err)
	}
	if len(assembled.Prompt) != 5 {
		t.Fatalf("assembled prompt has %d nodes, want 5", len(assembled.Prompt))
	}
}

func TestAssembleWorkflowFromNodeIDs_MissingNodeFails(t *testing.T) {
	resetNodeStateRegistry()

	RegisterNodeState(NodeState{
		ID:        uuidCheckpoint,
		ClassType: "CheckpointLoaderSimple",
		Inputs: map[string]interface{}{
			"ckpt_name": "v1-5-pruned-emaonly.safetensors",
		},
	})

	_, err := AssembleWorkflowFromNodeIDs([]string{uuidCheckpoint, uuidKSampler})
	if err == nil {
		t.Fatal("expected error when a requested node ID is not registered")
	}
}

func TestAssembleWorkflowFromNodeIDs_RequiresInProcessRegistryState(t *testing.T) {
	resetNodeStateRegistry()

	// This documents the current compatibility limitation: node_ids assembly only works
	// when the process-local registry has already been hydrated in this provider process.
	nodeIDs := []string{
		uuidCheckpoint,
		uuidCLIP,
		uuidKSampler,
		uuidVAEDecode,
		uuidSaveImage,
	}
	for _, node := range fullPipeline() {
		RegisterNodeState(node)
	}

	if _, err := AssembleWorkflowFromNodeIDs(nodeIDs); err != nil {
		t.Fatalf("expected initial in-process assembly to succeed, got %v", err)
	}

	resetNodeStateRegistry()

	_, err := AssembleWorkflowFromNodeIDs(nodeIDs)
	if err == nil {
		t.Fatal("expected assembly to fail after process-local registry state is cleared")
	}
	if !strings.Contains(err.Error(), "must be created before comfyui_workflow") {
		t.Fatalf("unexpected error after registry reset: %v", err)
	}
}
