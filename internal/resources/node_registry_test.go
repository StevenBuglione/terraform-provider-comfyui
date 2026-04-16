package resources

import (
	"strings"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/nodeschema"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestFindNestedDynamicComboInput_SanitizedNameMatchesRawName verifies that
// findNestedDynamicComboInput locates a child whose raw schema name contains spaces
// when the caller passes the sanitized Terraform attribute name.
func TestFindNestedDynamicComboInput_SanitizedNameMatchesRawName(t *testing.T) {
	parent := nodeschema.GeneratedNodeSchemaInput{
		Name: "parent",
		Type: "COMFY_DYNAMICCOMBO_V3",
		DynamicComboOptions: []nodeschema.GeneratedDynamicComboOption{
			{
				Key: "opt",
				Inputs: []nodeschema.GeneratedNodeSchemaInput{
					{Name: "my nested combo", Type: "COMFY_DYNAMICCOMBO_V3"},
				},
			},
		},
	}

	// The Terraform attribute name is the sanitized form of "my nested combo".
	got, ok := findNestedDynamicComboInput(parent, "my_nested_combo")
	if !ok {
		t.Fatal("expected findNestedDynamicComboInput to match via sanitized name; got false")
	}
	if got.Name != "my nested combo" {
		t.Errorf("got.Name = %q, want %q", got.Name, "my nested combo")
	}
}

// TestFindNestedDynamicComboInput_RawNameDoesNotMatchSanitized confirms that passing the
// raw (unsanitized) name does NOT match when the raw name differs from its sanitized form,
// ensuring the lookup enforces the Terraform-name contract.
func TestFindNestedDynamicComboInput_RawNameDoesNotMatchSanitized(t *testing.T) {
	parent := nodeschema.GeneratedNodeSchemaInput{
		Name: "parent",
		Type: "COMFY_DYNAMICCOMBO_V3",
		DynamicComboOptions: []nodeschema.GeneratedDynamicComboOption{
			{
				Key: "opt",
				Inputs: []nodeschema.GeneratedNodeSchemaInput{
					{Name: "my nested combo", Type: "COMFY_DYNAMICCOMBO_V3"},
				},
			},
		},
	}

	// The raw name "my nested combo" must NOT match because callers always pass sanitized names.
	_, ok := findNestedDynamicComboInput(parent, "my nested combo")
	if ok {
		t.Error("expected findNestedDynamicComboInput to NOT match the raw (unsanitized) name")
	}
}

// TestLookupDynamicComboInput_SanitizedNameMatchesRawName verifies that
// lookupDynamicComboInput finds a DynamicCombo input whose raw schema name
// contains punctuation, using the sanitized Terraform attribute name.
func TestLookupDynamicComboInput_SanitizedNameMatchesRawName(t *testing.T) {
	const testNodeType = "__test_sanitize_lookup__"
	nodeschema.RegisterForTest(testNodeType, nodeschema.GeneratedNodeSchema{
		NodeType: testNodeType,
		RequiredInputs: []nodeschema.GeneratedNodeSchemaInput{
			{Name: "my combo input", Type: "COMFY_DYNAMICCOMBO_V3"},
		},
	})

	// "my combo input" sanitizes to "my_combo_input".
	got, ok := lookupDynamicComboInput(testNodeType, "my_combo_input")
	if !ok {
		t.Fatal("expected lookupDynamicComboInput to match via sanitized name; got false")
	}
	if got.Name != "my combo input" {
		t.Errorf("got.Name = %q, want %q", got.Name, "my combo input")
	}
}

// TestCollectDynamicComboInputs_KeyedBySanitizedName verifies that
// collectDynamicComboInputs keys the returned map by the Terraform-sanitized name,
// not the raw schema name, so dotted-key lookups in the assembler work correctly.
func TestCollectDynamicComboInputs_KeyedBySanitizedName(t *testing.T) {
	const testNodeType = "__test_sanitize_collect__"
	nodeschema.RegisterForTest(testNodeType, nodeschema.GeneratedNodeSchema{
		NodeType: testNodeType,
		RequiredInputs: []nodeschema.GeneratedNodeSchemaInput{
			{Name: "My Combo", Type: "COMFY_DYNAMICCOMBO_V3"},
		},
		OptionalInputs: []nodeschema.GeneratedNodeSchemaInput{
			{Name: "1st optional combo", Type: "COMFY_DYNAMICCOMBO_V3"},
		},
	})

	dcMap := collectDynamicComboInputs(testNodeType)

	// Raw name "My Combo" sanitizes to "my_combo".
	if inp, ok := dcMap["my_combo"]; !ok {
		t.Error("expected map to have key \"my_combo\" (sanitized from \"My Combo\")")
	} else if inp.Name != "My Combo" {
		t.Errorf("inp.Name = %q, want %q", inp.Name, "My Combo")
	}

	// Leading digit: "1st optional combo" sanitizes to "_1st_optional_combo".
	if inp, ok := dcMap["_1st_optional_combo"]; !ok {
		t.Error("expected map to have key \"_1st_optional_combo\" (sanitized from \"1st optional combo\")")
	} else if inp.Name != "1st optional combo" {
		t.Errorf("inp.Name = %q, want %q", inp.Name, "1st optional combo")
	}

	// Raw names must NOT be present as keys.
	if _, ok := dcMap["My Combo"]; ok {
		t.Error("raw name \"My Combo\" must not be a key in the collected map; expected sanitized key only")
	}
}

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

// dcTestNodeComboAttrTypes matches the schema of DCTestNode.combo: the union of all
// option children (string, integer, image, subcombo) plus the selection key.
var dcTestNodeComboAttrTypes = map[string]attr.Type{
	"selection": types.StringType,
	"string":    types.StringType,
	"integer":   types.Int64Type,
	"image":     types.StringType,
	"subcombo": types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"selection": types.StringType,
			"float_x":   types.Float64Type,
			"float_y":   types.Float64Type,
			"mask1":     types.StringType,
		},
	},
}

// dcTestNodeSubcomboAttrTypes are the attribute types for the subcombo nested object.
var dcTestNodeSubcomboAttrTypes = map[string]attr.Type{
	"selection": types.StringType,
	"float_x":   types.Float64Type,
	"float_y":   types.Float64Type,
	"mask1":     types.StringType,
}

// TestFlattenDynamicComboInto_NestedDynamicComboFlattensRecursively proves that
// flattenDynamicComboInto recursively expands a nested DynamicCombo child
// (combo.subcombo) into dotted prompt keys (combo.subcombo, combo.subcombo.float_x, …)
// rather than leaving the child value as a nested map.
func TestFlattenDynamicComboInto_NestedDynamicComboFlattensRecursively(t *testing.T) {
	comboInput, ok := nodeschema.LookupGeneratedNodeSchema("DCTestNode")
	if !ok {
		t.Fatal("expected generated schema for DCTestNode")
	}
	var parentInput nodeschema.GeneratedNodeSchemaInput
	for _, inp := range comboInput.RequiredInputs {
		if inp.Name == "combo" {
			parentInput = inp
			break
		}
	}
	if parentInput.Name == "" {
		t.Fatal("could not find 'combo' input in DCTestNode schema")
	}

	value := map[string]interface{}{
		"selection": "option4",
		"subcombo": map[string]interface{}{
			"selection": "opt1",
			"float_x":   float64(1.5),
			"float_y":   float64(2.5),
		},
	}

	target := make(map[string]interface{})
	childKeys := flattenDynamicComboInto("combo", value, parentInput, target)

	// "combo" must be the selection, not a nested map.
	if target["combo"] != "option4" {
		t.Errorf("target[combo] = %#v, want \"option4\"", target["combo"])
	}
	if _, isMap := target["combo"].(map[string]interface{}); isMap {
		t.Error("target[combo] must not be a nested map")
	}

	// "combo.subcombo" must be the sub-selection, not a nested map.
	if target["combo.subcombo"] != "opt1" {
		t.Errorf("target[combo.subcombo] = %#v, want \"opt1\"", target["combo.subcombo"])
	}
	if _, isMap := target["combo.subcombo"].(map[string]interface{}); isMap {
		t.Error("target[combo.subcombo] must not be a nested map; subcombo must also be flattened")
	}

	// Leaf fields of the nested DynamicCombo must be dotted keys.
	if target["combo.subcombo.float_x"] != float64(1.5) {
		t.Errorf("target[combo.subcombo.float_x] = %#v, want 1.5", target["combo.subcombo.float_x"])
	}
	if target["combo.subcombo.float_y"] != float64(2.5) {
		t.Errorf("target[combo.subcombo.float_y] = %#v, want 2.5", target["combo.subcombo.float_y"])
	}

	// childKeys must include the nested leaf keys (not the nested map itself).
	wantKeys := map[string]bool{
		"combo.subcombo":         true,
		"combo.subcombo.float_x": true,
		"combo.subcombo.float_y": true,
	}
	for _, k := range childKeys {
		if k == "combo.subcombo" && target["combo.subcombo"] != "opt1" {
			t.Errorf("childKey combo.subcombo is present but target value is not the selection")
		}
	}
	for want := range wantKeys {
		found := false
		for _, k := range childKeys {
			if k == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("childKeys missing expected key %q; got %v", want, childKeys)
		}
	}
}

// TestRegisterNodeStateFromModel_DCTestNodeNestedDynamicComboFlattens exercises the
// full registration path for DCTestNode with option4 selected (which contains a nested
// COMFY_DYNAMICCOMBO_V3 child "subcombo").  All nested values must be fully flattened
// into dotted prompt keys.
func TestRegisterNodeStateFromModel_DCTestNodeNestedDynamicComboFlattens(t *testing.T) {
	resetNodeStateRegistry()

	const nodeID = "eeeeeeee-0000-0000-0000-000000000000"

	subcomboObj := types.ObjectValueMust(dcTestNodeSubcomboAttrTypes, map[string]attr.Value{
		"selection": types.StringValue("opt1"),
		"float_x":   types.Float64Value(1.5),
		"float_y":   types.Float64Value(2.5),
		"mask1":     types.StringNull(),
	})

	comboObj := types.ObjectValueMust(dcTestNodeComboAttrTypes, map[string]attr.Value{
		"selection": types.StringValue("option4"),
		"string":    types.StringNull(),
		"integer":   types.Int64Null(),
		"image":     types.StringNull(),
		"subcombo":  subcomboObj,
	})

	model := struct {
		ID            types.String `tfsdk:"id"`
		NodeID        types.String `tfsdk:"node_id"`
		Combo         types.Object `tfsdk:"combo"`
		UnnamedOutput types.String `tfsdk:"unnamed_output"`
	}{
		ID:            types.StringValue(nodeID),
		NodeID:        types.StringValue("DCTestNode"),
		Combo:         comboObj,
		UnnamedOutput: types.StringValue(nodeID + ":0"),
	}

	if err := RegisterNodeStateFromModel(nodeID, "DCTestNode", model); err != nil {
		t.Fatalf("RegisterNodeStateFromModel returned error: %v", err)
	}

	got, ok := LookupNodeState(nodeID)
	if !ok {
		t.Fatal("expected registered node state")
	}

	if got.Inputs["combo"] != "option4" {
		t.Errorf("inputs[combo] = %#v, want \"option4\"", got.Inputs["combo"])
	}
	if _, isMap := got.Inputs["combo.subcombo"].(map[string]interface{}); isMap {
		t.Error("inputs[combo.subcombo] must not be a nested map; nested DynamicCombo must be flattened")
	}
	if got.Inputs["combo.subcombo"] != "opt1" {
		t.Errorf("inputs[combo.subcombo] = %#v, want \"opt1\"", got.Inputs["combo.subcombo"])
	}
	if got.Inputs["combo.subcombo.float_x"] != float64(1.5) {
		t.Errorf("inputs[combo.subcombo.float_x] = %#v, want 1.5", got.Inputs["combo.subcombo.float_x"])
	}
	if got.Inputs["combo.subcombo.float_y"] != float64(2.5) {
		t.Errorf("inputs[combo.subcombo.float_y] = %#v, want 2.5", got.Inputs["combo.subcombo.float_y"])
	}
}

// TestFlattenDynamicComboInto_NonDynamicComboMapChildNotFlattened verifies that a
// map-typed child that is NOT a DynamicCombo input (no schema entry) is stored as-is,
// preserving existing behavior for non-DynamicCombo nested payloads.
func TestFlattenDynamicComboInto_NonDynamicComboMapChildNotFlattened(t *testing.T) {
	// Use a parent input with no DynamicComboOptions children (e.g. option1 which has STRING).
	comboSchema, ok := nodeschema.LookupGeneratedNodeSchema("DCTestNode")
	if !ok {
		t.Fatal("expected generated schema for DCTestNode")
	}
	var parentInput nodeschema.GeneratedNodeSchemaInput
	for _, inp := range comboSchema.RequiredInputs {
		if inp.Name == "combo" {
			parentInput = inp
			break
		}
	}

	// The value contains a plain map under key "extra_data" — not a DynamicCombo child.
	// flattenDynamicComboInto must store it as-is, not attempt to recurse.
	value := map[string]interface{}{
		"selection":  "option1",
		"extra_data": map[string]interface{}{"foo": "bar"},
	}

	target := make(map[string]interface{})
	flattenDynamicComboInto("combo", value, parentInput, target)

	if _, isMap := target["combo.extra_data"].(map[string]interface{}); !isMap {
		t.Errorf("non-DynamicCombo map child must remain as nested map, got %T", target["combo.extra_data"])
	}
}

// TestFlattenDynamicComboInto_NullSelectionOmitsParentKey documents the invariant that
// when the "selection" key is absent from the DynamicCombo map (because the user left
// it null and terraformValueToNative skipped it), flattenDynamicComboInto must NOT add
// any entry for the parent input key.  Terraform schema guarantees selection is required
// for strict nodes; for non-strict nodes the entire object will typically be null and
// extractInputsFromModel skips it entirely before flattenDynamicComboInto is called.
//
// This test documents the current invariant so that future refactors don't silently
// regress to writing an empty-string or nil entry for the parent key.
func TestFlattenDynamicComboInto_NullSelectionOmitsParentKey(t *testing.T) {
	comboSchema, ok := nodeschema.LookupGeneratedNodeSchema("DCTestNode")
	if !ok {
		t.Fatal("expected generated schema for DCTestNode")
	}
	var parentInput nodeschema.GeneratedNodeSchemaInput
	for _, inp := range comboSchema.RequiredInputs {
		if inp.Name == "combo" {
			parentInput = inp
			break
		}
	}

	// terraformValueToNative skips null attributes, so "selection" is absent.
	value := map[string]interface{}{
		"string": "hello", // child present but no selection
	}

	target := make(map[string]interface{})
	flattenDynamicComboInto("combo", value, parentInput, target)

	if _, exists := target["combo"]; exists {
		t.Errorf("parent key \"combo\" must not be written when selection is absent, got %#v", target["combo"])
	}
	// The child key is still written with a dotted path.
	if target["combo.string"] != "hello" {
		t.Errorf("combo.string = %#v, want \"hello\"", target["combo.string"])
	}
}

// TestRegisterNodeStateFromModel_Wan2DynamicCombo_NullObjectSkipped documents that when
// the entire DynamicCombo Object field is null (not just individual children), the
// extractInputsFromModel path omits it entirely — the inputs map must not contain the
// parent key or any dotted child key for that field.
func TestRegisterNodeStateFromModel_Wan2DynamicCombo_NullObjectSkipped(t *testing.T) {
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
		Model: types.ObjectNull(map[string]attr.Type{
			"selection":       types.StringType,
			"prompt":          types.StringType,
			"negative_prompt": types.StringType,
			"resolution":      types.StringType,
			"duration":        types.Int64Type,
		}),
		FirstFrame:   types.StringValue("bbbbbbbb-0000-0000-0000-000000000000:0"),
		Seed:         types.Int64Value(42),
		PromptExtend: types.BoolValue(false),
		Watermark:    types.BoolValue(false),
	}

	inputs, err := extractInputsFromModel("Wan2ImageToVideoApi", model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for key := range inputs {
		if key == "model" || strings.HasPrefix(key, "model.") {
			t.Errorf("null DynamicCombo object must not produce any model/* keys in inputs, found %q", key)
		}
	}

	if inputs["first_frame"] != "bbbbbbbb-0000-0000-0000-000000000000:0" {
		t.Errorf("first_frame = %#v, want connection ref string", inputs["first_frame"])
	}
}
