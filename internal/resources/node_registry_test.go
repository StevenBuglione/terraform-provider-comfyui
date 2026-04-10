package resources

import (
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
