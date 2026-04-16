package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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

// wan2I2VModelAttrTypes mirrors the Wan2ImageToVideoAPIModel.Model attribute types
// as defined by the generated resource schema.
var wan2I2VModelAttrTypes = map[string]attr.Type{
	"selection":       types.StringType,
	"prompt":          types.StringType,
	"negative_prompt": types.StringType,
	"resolution":      types.StringType,
	"duration":        types.Int64Type,
}

// TestRegisterNodeStateFromModel_Wan2DynamicComboModelFlattensToPromptKeys
// documents the regression: for a WAN2 DynamicCombo "model" input, the registered
// node state must store flattened prompt keys (model, model.prompt, model.resolution,
// model.duration, model.negative_prompt) that match the ComfyUI wire format, NOT a
// nested map under the "model" key.
//
// ComfyUI reconstructs the DynamicCombo widget values from dotted-key prompt entries:
//
//	model                = "wan2.7-i2v"   (the selection / model name)
//	model.prompt         = "..."
//	model.negative_prompt= "..."
//	model.resolution     = "720P"
//	model.duration       = 5
//
// This test FAILS before the fix: the registry currently converts types.Object to a
// nested map[string]interface{} instead of expanding the DynamicCombo to flat keys.
func TestRegisterNodeStateFromModel_Wan2DynamicComboModelFlattensToPromptKeys(t *testing.T) {
	resetNodeStateRegistry()

	const nodeID = "cccccccc-1234-5678-abcd-cccccccccccc"
	const imageRef = "dddddddd-0000-0000-0000-000000000000:0"

	model := struct {
		ID           types.String `tfsdk:"id"`
		NodeID       types.String `tfsdk:"node_id"`
		Model        types.Object `tfsdk:"model"`
		FirstFrame   types.String `tfsdk:"first_frame"`
		LastFrame    types.String `tfsdk:"last_frame"`
		Audio        types.String `tfsdk:"audio"`
		Seed         types.Int64  `tfsdk:"seed"`
		PromptExtend types.Bool   `tfsdk:"prompt_extend"`
		Watermark    types.Bool   `tfsdk:"watermark"`
		VideoOutput  types.String `tfsdk:"video_output"`
	}{
		ID:     types.StringValue(nodeID),
		NodeID: types.StringValue("Wan2ImageToVideoApi"),
		Model: types.ObjectValueMust(wan2I2VModelAttrTypes, map[string]attr.Value{
			"selection":       types.StringValue("wan2.7-i2v"),
			"prompt":          types.StringValue("a cat running in slow motion"),
			"negative_prompt": types.StringValue(""),
			"resolution":      types.StringValue("720P"),
			"duration":        types.Int64Value(5),
		}),
		FirstFrame:   types.StringValue(imageRef),
		LastFrame:    types.StringNull(),
		Audio:        types.StringNull(),
		Seed:         types.Int64Value(42),
		PromptExtend: types.BoolValue(true),
		Watermark:    types.BoolValue(false),
		VideoOutput:  types.StringValue(nodeID + ":0"),
	}

	if err := RegisterNodeStateFromModel(nodeID, "Wan2ImageToVideoApi", model); err != nil {
		t.Fatalf("RegisterNodeStateFromModel returned error: %v", err)
	}

	got, ok := LookupNodeState(nodeID)
	if !ok {
		t.Fatal("expected registered node state to be present")
	}

	// "model" key must be the selection string, not a nested map.
	if got.Inputs["model"] != "wan2.7-i2v" {
		t.Errorf("inputs[model] = %#v, want \"wan2.7-i2v\"", got.Inputs["model"])
	}
	if _, isMap := got.Inputs["model"].(map[string]interface{}); isMap {
		t.Error("inputs[model] must not be a nested map; DynamicCombo must be flattened to dotted keys")
	}

	// Child fields must appear as top-level dotted-key entries.
	if got.Inputs["model.prompt"] != "a cat running in slow motion" {
		t.Errorf("inputs[model.prompt] = %#v, want \"a cat running in slow motion\"", got.Inputs["model.prompt"])
	}
	if got.Inputs["model.resolution"] != "720P" {
		t.Errorf("inputs[model.resolution] = %#v, want \"720P\"", got.Inputs["model.resolution"])
	}
	if got.Inputs["model.negative_prompt"] != "" {
		t.Errorf("inputs[model.negative_prompt] = %#v, want \"\"", got.Inputs["model.negative_prompt"])
	}
	if got.Inputs["model.duration"] != int64(5) {
		t.Errorf("inputs[model.duration] = %#v, want int64(5)", got.Inputs["model.duration"])
	}
}
