package resources

import (
	"encoding/json"
	"fmt"
	"testing"
)

// --- isConnectionRef tests ---

func TestIsConnectionRef_ValidRefs(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"slot 0", "550e8400-e29b-41d4-a716-446655440000:0"},
		{"slot 1", "550e8400-e29b-41d4-a716-446655440000:1"},
		{"slot 10", "abcdef01-2345-6789-abcd-ef0123456789:10"},
		{"all zeros", "00000000-0000-0000-0000-000000000000:0"},
		{"all f's", "ffffffff-ffff-ffff-ffff-ffffffffffff:99"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !isConnectionRef(tt.input) {
				t.Errorf("expected %q to be a valid connection ref", tt.input)
			}
		})
	}
}

func TestIsConnectionRef_InvalidStrings(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"plain string", "hello"},
		{"number only", "42"},
		{"uuid without slot", "550e8400-e29b-41d4-a716-446655440000"},
		{"missing uuid", ":0"},
		{"short uuid", "550e8400-e29b-41d4-a716:0"},
		{"uppercase hex", "550E8400-E29B-41D4-A716-446655440000:0"},
		{"colon only", ":"},
		{"path-like", "/some/path:0"},
		{"negative slot", "550e8400-e29b-41d4-a716-446655440000:-1"},
		{"float slot", "550e8400-e29b-41d4-a716-446655440000:0.5"},
		{"double colon", "550e8400-e29b-41d4-a716-446655440000::0"},
		{"trailing colon", "550e8400-e29b-41d4-a716-446655440000:0:"},
		{"extra segment", "550e8400-e29b-41d4-a716-446655440000-extra:0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isConnectionRef(tt.input) {
				t.Errorf("expected %q to NOT be a valid connection ref", tt.input)
			}
		})
	}
}

// --- parseConnectionRef tests ---

func TestParseConnectionRef_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantUUID string
		wantSlot int
	}{
		{"slot 0", "550e8400-e29b-41d4-a716-446655440000:0", "550e8400-e29b-41d4-a716-446655440000", 0},
		{"slot 5", "abcdef01-2345-6789-abcd-ef0123456789:5", "abcdef01-2345-6789-abcd-ef0123456789", 5},
		{"slot 42", "00000000-0000-0000-0000-000000000000:42", "00000000-0000-0000-0000-000000000000", 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uuid, slot, err := parseConnectionRef(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if uuid != tt.wantUUID {
				t.Errorf("UUID = %q, want %q", uuid, tt.wantUUID)
			}
			if slot != tt.wantSlot {
				t.Errorf("slot = %d, want %d", slot, tt.wantSlot)
			}
		})
	}
}

func TestParseConnectionRef_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"plain string", "not-a-ref"},
		{"uuid only", "550e8400-e29b-41d4-a716-446655440000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseConnectionRef(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}

// --- AssembleWorkflow tests ---

func TestAssembleWorkflow_EmptyNodes(t *testing.T) {
	_, err := AssembleWorkflow(nil)
	if err == nil {
		t.Fatal("expected error for nil nodes")
	}

	_, err = AssembleWorkflow([]NodeState{})
	if err == nil {
		t.Fatal("expected error for empty nodes")
	}
}

func TestAssembleWorkflow_EmptyID(t *testing.T) {
	_, err := AssembleWorkflow([]NodeState{
		{ID: "", ClassType: "KSampler", Inputs: nil},
	})
	if err == nil {
		t.Fatal("expected error for empty node ID")
	}
}

func TestAssembleWorkflow_EmptyClassType(t *testing.T) {
	_, err := AssembleWorkflow([]NodeState{
		{ID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ClassType: "", Inputs: nil},
	})
	if err == nil {
		t.Fatal("expected error for empty ClassType")
	}
}

func TestAssembleWorkflow_SingleNode(t *testing.T) {
	nodes := []NodeState{
		{
			ID:        "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			ClassType: "CheckpointLoaderSimple",
			Inputs: map[string]interface{}{
				"ckpt_name": "model.safetensors",
			},
		},
	}

	result, err := AssembleWorkflow(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NodeMap["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"] != "1" {
		t.Errorf("expected node mapped to '1', got %q", result.NodeMap["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"])
	}

	node1 := result.Prompt["1"].(map[string]interface{})
	if node1["class_type"] != "CheckpointLoaderSimple" {
		t.Errorf("class_type = %v, want CheckpointLoaderSimple", node1["class_type"])
	}

	inputs := node1["inputs"].(map[string]interface{})
	if inputs["ckpt_name"] != "model.safetensors" {
		t.Errorf("ckpt_name = %v, want model.safetensors", inputs["ckpt_name"])
	}

	if result.JSON == "" {
		t.Error("JSON output should not be empty")
	}
}

func TestAssembleWorkflow_TwoNodeChain(t *testing.T) {
	loaderID := "11111111-1111-1111-1111-111111111111"
	samplerID := "22222222-2222-2222-2222-222222222222"

	nodes := []NodeState{
		{
			ID:        loaderID,
			ClassType: "CheckpointLoaderSimple",
			Inputs: map[string]interface{}{
				"ckpt_name": "model.safetensors",
			},
		},
		{
			ID:        samplerID,
			ClassType: "KSampler",
			Inputs: map[string]interface{}{
				"model": loaderID + ":0",
				"seed":  42,
				"steps": 20,
			},
		},
	}

	result, err := AssembleWorkflow(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify node map
	if result.NodeMap[loaderID] != "1" {
		t.Errorf("loader mapped to %q, want '1'", result.NodeMap[loaderID])
	}
	if result.NodeMap[samplerID] != "2" {
		t.Errorf("sampler mapped to %q, want '2'", result.NodeMap[samplerID])
	}

	// Verify sampler's model input is a connection reference
	sampler := result.Prompt["2"].(map[string]interface{})
	inputs := sampler["inputs"].(map[string]interface{})

	modelRef, ok := inputs["model"].([]interface{})
	if !ok {
		t.Fatalf("model input should be []interface{}, got %T", inputs["model"])
	}
	if modelRef[0] != "1" || modelRef[1] != 0 {
		t.Errorf("model ref = %v, want [\"1\", 0]", modelRef)
	}

	// Verify literal values preserved
	if inputs["seed"] != 42 {
		t.Errorf("seed = %v, want 42", inputs["seed"])
	}
	if inputs["steps"] != 20 {
		t.Errorf("steps = %v, want 20", inputs["steps"])
	}

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result.JSON), &parsed); err != nil {
		t.Fatalf("output JSON is invalid: %v", err)
	}
}

func TestAssembleWorkflow_ComplexSevenNodeWorkflow(t *testing.T) {
	ids := []string{
		"aaaaaaaa-0001-0001-0001-000000000001", // CheckpointLoaderSimple
		"aaaaaaaa-0002-0002-0002-000000000002", // CLIPTextEncode (positive)
		"aaaaaaaa-0003-0003-0003-000000000003", // CLIPTextEncode (negative)
		"aaaaaaaa-0004-0004-0004-000000000004", // EmptyLatentImage
		"aaaaaaaa-0005-0005-0005-000000000005", // KSampler
		"aaaaaaaa-0006-0006-0006-000000000006", // VAEDecode
		"aaaaaaaa-0007-0007-0007-000000000007", // SaveImage
	}

	nodes := []NodeState{
		{ID: ids[0], ClassType: "CheckpointLoaderSimple", Inputs: map[string]interface{}{
			"ckpt_name": "v1-5-pruned.safetensors",
		}},
		{ID: ids[1], ClassType: "CLIPTextEncode", Inputs: map[string]interface{}{
			"text": "a beautiful landscape",
			"clip": ids[0] + ":1", // CLIP output from loader
		}},
		{ID: ids[2], ClassType: "CLIPTextEncode", Inputs: map[string]interface{}{
			"text": "ugly, blurry",
			"clip": ids[0] + ":1", // shared CLIP connection
		}},
		{ID: ids[3], ClassType: "EmptyLatentImage", Inputs: map[string]interface{}{
			"width":      512,
			"height":     512,
			"batch_size": 1,
		}},
		{ID: ids[4], ClassType: "KSampler", Inputs: map[string]interface{}{
			"model":        ids[0] + ":0",
			"positive":     ids[1] + ":0",
			"negative":     ids[2] + ":0",
			"latent_image": ids[3] + ":0",
			"seed":         8566257,
			"steps":        20,
			"cfg":          8.0,
			"sampler_name": "euler",
			"scheduler":    "normal",
			"denoise":      1.0,
		}},
		{ID: ids[5], ClassType: "VAEDecode", Inputs: map[string]interface{}{
			"samples": ids[4] + ":0",
			"vae":     ids[0] + ":2",
		}},
		{ID: ids[6], ClassType: "SaveImage", Inputs: map[string]interface{}{
			"images":          ids[5] + ":0",
			"filename_prefix": "ComfyUI",
		}},
	}

	result, err := AssembleWorkflow(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all 7 nodes assigned
	if len(result.Prompt) != 7 {
		t.Errorf("prompt has %d nodes, want 7", len(result.Prompt))
	}

	// Verify shared connection: both CLIP encoders reference node "1" slot 1
	pos := result.Prompt["2"].(map[string]interface{})["inputs"].(map[string]interface{})
	neg := result.Prompt["3"].(map[string]interface{})["inputs"].(map[string]interface{})

	posClip := pos["clip"].([]interface{})
	negClip := neg["clip"].([]interface{})
	if posClip[0] != "1" || posClip[1] != 1 {
		t.Errorf("positive clip ref = %v, want [\"1\", 1]", posClip)
	}
	if negClip[0] != "1" || negClip[1] != 1 {
		t.Errorf("negative clip ref = %v, want [\"1\", 1]", negClip)
	}

	// Verify KSampler has multiple connection refs
	ks := result.Prompt["5"].(map[string]interface{})["inputs"].(map[string]interface{})
	modelRef := ks["model"].([]interface{})
	if modelRef[0] != "1" || modelRef[1] != 0 {
		t.Errorf("KSampler model = %v, want [\"1\", 0]", modelRef)
	}

	// Verify float values preserved
	if ks["cfg"] != 8.0 {
		t.Errorf("cfg = %v, want 8.0", ks["cfg"])
	}

	// Verify valid JSON output
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result.JSON), &parsed); err != nil {
		t.Fatalf("output JSON is invalid: %v", err)
	}
	if len(parsed) != 7 {
		t.Errorf("parsed JSON has %d keys, want 7", len(parsed))
	}
}

func TestAssembleWorkflow_ReferenceToUnknownNode(t *testing.T) {
	nodes := []NodeState{
		{
			ID:        "11111111-1111-1111-1111-111111111111",
			ClassType: "KSampler",
			Inputs: map[string]interface{}{
				"model": "99999999-9999-9999-9999-999999999999:0",
			},
		},
	}

	_, err := AssembleWorkflow(nodes)
	if err == nil {
		t.Fatal("expected error for reference to unknown node")
	}
}

func TestAssembleWorkflow_SkipsEmptyAndNilInputs(t *testing.T) {
	nodes := []NodeState{
		{
			ID:        "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			ClassType: "TestNode",
			Inputs: map[string]interface{}{
				"keep_this":  "value",
				"skip_empty": "",
				"skip_nil":   nil,
				"keep_int":   100,
				"keep_bool":  true,
			},
		},
	}

	result, err := AssembleWorkflow(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inputs := result.Prompt["1"].(map[string]interface{})["inputs"].(map[string]interface{})

	if _, ok := inputs["skip_empty"]; ok {
		t.Error("empty string input should be skipped")
	}
	if _, ok := inputs["skip_nil"]; ok {
		t.Error("nil input should be skipped")
	}
	if inputs["keep_this"] != "value" {
		t.Error("string input should be preserved")
	}
	if inputs["keep_int"] != 100 {
		t.Error("int input should be preserved")
	}
	if inputs["keep_bool"] != true {
		t.Error("bool input should be preserved")
	}
}

func TestAssembleWorkflow_JSONMatchesAPIFormat(t *testing.T) {
	loaderID := "11111111-1111-1111-1111-111111111111"
	samplerID := "22222222-2222-2222-2222-222222222222"

	nodes := []NodeState{
		{
			ID:        loaderID,
			ClassType: "CheckpointLoaderSimple",
			Inputs: map[string]interface{}{
				"ckpt_name": "model.safetensors",
			},
		},
		{
			ID:        samplerID,
			ClassType: "KSampler",
			Inputs: map[string]interface{}{
				"model": loaderID + ":0",
				"seed":  42,
				"steps": 20,
			},
		},
	}

	result, err := AssembleWorkflow(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Re-parse the JSON to verify structure
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(result.JSON), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Verify keys are string numbers
	for _, key := range []string{"1", "2"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("missing key %q in JSON output", key)
		}
	}

	// Parse node "2" and verify connection reference format
	var node2 struct {
		ClassType string                     `json:"class_type"`
		Inputs    map[string]json.RawMessage `json:"inputs"`
	}
	if err := json.Unmarshal(parsed["2"], &node2); err != nil {
		t.Fatalf("failed to parse node 2: %v", err)
	}
	if node2.ClassType != "KSampler" {
		t.Errorf("class_type = %q, want KSampler", node2.ClassType)
	}

	// model should be ["1", 0]
	var modelArr []interface{}
	if err := json.Unmarshal(node2.Inputs["model"], &modelArr); err != nil {
		t.Fatalf("model is not an array: %v", err)
	}
	if len(modelArr) != 2 {
		t.Fatalf("model array length = %d, want 2", len(modelArr))
	}
	if modelArr[0] != "1" {
		t.Errorf("model[0] = %v, want \"1\"", modelArr[0])
	}
	// JSON numbers decode as float64
	if modelArr[1] != float64(0) {
		t.Errorf("model[1] = %v, want 0", modelArr[1])
	}
}

func TestAssembleWorkflow_ResolvesNestedObjectsAndLists(t *testing.T) {
	// This test covers nodes that genuinely accept nested JSON payloads as inputs
	// (e.g., a node whose "payload" field takes an arbitrary nested object).
	// This is distinct from generated DynamicCombo widget inputs, which ComfyUI
	// represents as flattened dotted-key prompt entries rather than nested objects.
	imageID := "11111111-1111-1111-1111-111111111111"
	maskID := "22222222-2222-2222-2222-222222222222"
	consumerID := "33333333-3333-3333-3333-333333333333"

	nodes := []NodeState{
		{
			ID:        imageID,
			ClassType: "LoadImage",
			Inputs: map[string]interface{}{
				"image": "image.png",
			},
		},
		{
			ID:        maskID,
			ClassType: "LoadMask",
			Inputs: map[string]interface{}{
				"mask": "mask.png",
			},
		},
		{
			ID:        consumerID,
			ClassType: "DynamicComboConsumer",
			Inputs: map[string]interface{}{
				"payload": map[string]interface{}{
					"primary": imageID + ":0",
					"options": []interface{}{
						"literal-option",
						true,
						map[string]interface{}{
							"mask":   maskID + ":0",
							"weight": 0.75,
						},
					},
					"settings": map[string]interface{}{
						"enabled": true,
						"label":   "keep-me",
						"count":   3,
					},
				},
			},
		},
	}

	result, err := AssembleWorkflow(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inputs := result.Prompt["3"].(map[string]interface{})["inputs"].(map[string]interface{})
	payload := inputs["payload"].(map[string]interface{})

	primaryRef, ok := payload["primary"].([]interface{})
	if !ok {
		t.Fatalf("payload.primary type = %T, want []interface{}", payload["primary"])
	}
	if primaryRef[0] != "1" || primaryRef[1] != 0 {
		t.Fatalf("payload.primary = %#v, want [\"1\", 0]", primaryRef)
	}

	options, ok := payload["options"].([]interface{})
	if !ok {
		t.Fatalf("payload.options type = %T, want []interface{}", payload["options"])
	}
	if options[0] != "literal-option" {
		t.Fatalf("payload.options[0] = %#v, want literal-option", options[0])
	}
	if options[1] != true {
		t.Fatalf("payload.options[1] = %#v, want true", options[1])
	}

	nestedOption, ok := options[2].(map[string]interface{})
	if !ok {
		t.Fatalf("payload.options[2] type = %T, want map[string]interface{}", options[2])
	}
	maskRef, ok := nestedOption["mask"].([]interface{})
	if !ok {
		t.Fatalf("nested mask type = %T, want []interface{}", nestedOption["mask"])
	}
	if maskRef[0] != "2" || maskRef[1] != 0 {
		t.Fatalf("nested mask = %#v, want [\"2\", 0]", maskRef)
	}
	if nestedOption["weight"] != 0.75 {
		t.Fatalf("nested weight = %#v, want 0.75", nestedOption["weight"])
	}

	settings, ok := payload["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("payload.settings type = %T, want map[string]interface{}", payload["settings"])
	}
	if settings["enabled"] != true || settings["label"] != "keep-me" || settings["count"] != 3 {
		t.Fatalf("payload.settings = %#v, want preserved scalars", settings)
	}
}

func TestAssembleWorkflow_JSONSerializesNestedDynamicComboRefs(t *testing.T) {
	sourceID := "44444444-4444-4444-4444-444444444444"
	consumerID := "55555555-5555-5555-5555-555555555555"

	nodes := []NodeState{
		{
			ID:        sourceID,
			ClassType: "LoadImage",
			Inputs: map[string]interface{}{
				"image": "image.png",
			},
		},
		{
			ID:        consumerID,
			ClassType: "DynamicComboConsumer",
			Inputs: map[string]interface{}{
				"payload": map[string]interface{}{
					"nested_list": []interface{}{
						map[string]interface{}{
							"image": sourceID + ":0",
						},
					},
				},
			},
		},
	}

	result, err := AssembleWorkflow(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]struct {
		Inputs map[string]json.RawMessage `json:"inputs"`
	}
	if err := json.Unmarshal([]byte(result.JSON), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	var payload struct {
		NestedList []struct {
			Image []interface{} `json:"image"`
		} `json:"nested_list"`
	}
	if err := json.Unmarshal(parsed["2"].Inputs["payload"], &payload); err != nil {
		t.Fatalf("failed to parse payload JSON: %v", err)
	}

	if len(payload.NestedList) != 1 {
		t.Fatalf("nested_list length = %d, want 1", len(payload.NestedList))
	}
	if len(payload.NestedList[0].Image) != 2 {
		t.Fatalf("image ref length = %d, want 2", len(payload.NestedList[0].Image))
	}
	if payload.NestedList[0].Image[0] != "1" || payload.NestedList[0].Image[1] != float64(0) {
		t.Fatalf("image ref = %#v, want [\"1\", 0]", payload.NestedList[0].Image)
	}
}

func TestResolveInputValue_Types(t *testing.T) {
	nodeMap := map[string]string{
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee": "1",
	}

	tests := []struct {
		name     string
		input    interface{}
		wantNil  bool
		wantType string
	}{
		{"nil", nil, true, ""},
		{"empty string", "", true, ""},
		{"plain string", "hello", false, "string"},
		{"int", 42, false, "int"},
		{"int64", int64(42), false, "int64"},
		{"float64", 3.14, false, "float64"},
		{"bool", true, false, "bool"},
		{"connection ref", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:0", false, "[]interface {}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveInputValue(tt.input, nodeMap)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			gotType := fmt.Sprintf("%T", result)
			if gotType != tt.wantType {
				t.Errorf("type = %s, want %s", gotType, tt.wantType)
			}
		})
	}
}

// TestAssembleWorkflow_Wan2DynamicComboInputsAreFlattened documents the regression:
// a WAN2 DynamicCombo "model" input registered as a nested map by the current
// RegisterNodeStateFromModel must be expanded into flat dotted-key prompt entries
// in the assembled JSON.
//
// ComfyUI rebuilds the DynamicCombo widget from these flat entries during execution:
//
//	"model"                → "wan2.7-i2v"      (the selection/model name)
//	"model.prompt"         → "a cat ..."
//	"model.negative_prompt"→ ""
//	"model.resolution"     → "720P"
//	"model.duration"       → 5
//
// This test FAILS before the fix: the assembler currently passes the nested
// map through as-is, producing "model": {"selection": ..., "prompt": ...}
// in the prompt JSON instead of the required flat keys.
func TestAssembleWorkflow_Wan2DynamicComboInputsAreFlattened(t *testing.T) {
	imageLoaderID := "eeeeeeee-0000-0000-0000-000000000001"
	wan2NodeID := "eeeeeeee-0000-0000-0000-000000000002"

	// This is the nested map that RegisterNodeStateFromModel currently stores for
	// a types.Object DynamicCombo field — i.e., the current (broken) registry output.
	nodes := []NodeState{
		{
			ID:        imageLoaderID,
			ClassType: "LoadImage",
			Inputs: map[string]interface{}{
				"image": "cat.jpg",
			},
		},
		{
			ID:        wan2NodeID,
			ClassType: "Wan2ImageToVideoApi",
			Inputs: map[string]interface{}{
				// Nested map — current (wrong) output of RegisterNodeStateFromModel
				// for a types.Object DynamicCombo field.
				"model": map[string]interface{}{
					"selection":       "wan2.7-i2v",
					"prompt":          "a cat running in slow motion",
					"negative_prompt": "",
					"resolution":      "720P",
					"duration":        int64(5),
				},
				"first_frame":   imageLoaderID + ":0",
				"seed":          int64(42),
				"prompt_extend": true,
				"watermark":     false,
			},
		},
	}

	result, err := AssembleWorkflow(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the Wan2ImageToVideoApi node in the assembled prompt.
	var wan2Inputs map[string]interface{}
	for _, entry := range result.Prompt {
		node, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if node["class_type"] == "Wan2ImageToVideoApi" {
			wan2Inputs, ok = node["inputs"].(map[string]interface{})
			if !ok {
				t.Fatal("expected wan2 inputs to be a map")
			}
			break
		}
	}
	if wan2Inputs == nil {
		t.Fatal("expected Wan2ImageToVideoApi node in assembled prompt")
	}

	// "model" must be the bare selection string, not a nested object.
	if wan2Inputs["model"] != "wan2.7-i2v" {
		t.Errorf("inputs[model] = %#v, want \"wan2.7-i2v\"", wan2Inputs["model"])
	}
	if _, isMap := wan2Inputs["model"].(map[string]interface{}); isMap {
		t.Error("inputs[model] must not be a nested map; DynamicCombo must be flattened")
	}

	// Child fields must be top-level dotted-key entries in the assembled JSON.
	if wan2Inputs["model.prompt"] != "a cat running in slow motion" {
		t.Errorf("inputs[model.prompt] = %#v, want \"a cat running in slow motion\"", wan2Inputs["model.prompt"])
	}
	if wan2Inputs["model.negative_prompt"] != "" {
		t.Errorf("inputs[model.negative_prompt] = %#v, want \"\"", wan2Inputs["model.negative_prompt"])
	}
	if wan2Inputs["model.resolution"] != "720P" {
		t.Errorf("inputs[model.resolution] = %#v, want \"720P\"", wan2Inputs["model.resolution"])
	}
	// duration is int64 in native Go; JSON round-trips to float64 but we check the in-memory map.
	if wan2Inputs["model.duration"] != int64(5) {
		t.Errorf("inputs[model.duration] = %#v, want int64(5)", wan2Inputs["model.duration"])
	}
}

// TestAssembleWorkflow_PreFlattenedDynamicComboChildEmptyStringPreserved covers the
// production path where node_registry.go has already flattened a DynamicCombo input
// into dotted keys before AssembleWorkflow is called.  Empty-string child values
// (e.g. model.negative_prompt = "") must survive assembly unchanged; they must NOT be
// dropped by the top-level empty-string filter.
//
// This is distinct from TestAssembleWorkflow_Wan2DynamicComboInputsAreFlattened which
// exercises the assembler receiving a still-nested map[string]interface{} DynamicCombo
// value — the legacy / fallback path.
func TestAssembleWorkflow_PreFlattenedDynamicComboChildEmptyStringPreserved(t *testing.T) {
	imageLoaderID := "ffff0000-0000-0000-0000-000000000001"
	wan2NodeID := "ffff0000-0000-0000-0000-000000000002"

	// Inputs as produced by RegisterNodeStateFromModel after registry-time flattening:
	// the DynamicCombo "model" input has already been expanded to dotted keys.
	nodes := []NodeState{
		{
			ID:        imageLoaderID,
			ClassType: "LoadImage",
			Inputs: map[string]interface{}{
				"image": "cat.jpg",
			},
		},
		{
			ID:        wan2NodeID,
			ClassType: "Wan2ImageToVideoApi",
			Inputs: map[string]interface{}{
				// Pre-flattened DynamicCombo keys — output of node_registry.go
				"model":                 "wan2.7-i2v",
				"model.prompt":          "a cat running in slow motion",
				"model.negative_prompt": "", // empty string must be preserved
				"model.resolution":      "720P",
				"model.duration":        int64(5),
				// Connection ref to image loader
				"first_frame": imageLoaderID + ":0",
				"seed":        int64(42),
			},
		},
	}

	result, err := AssembleWorkflow(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var wan2Inputs map[string]interface{}
	for _, entry := range result.Prompt {
		node, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if node["class_type"] == "Wan2ImageToVideoApi" {
			wan2Inputs, ok = node["inputs"].(map[string]interface{})
			if !ok {
				t.Fatal("expected wan2 inputs to be a map")
			}
			break
		}
	}
	if wan2Inputs == nil {
		t.Fatal("expected Wan2ImageToVideoApi node in assembled prompt")
	}

	// Selection key must be the bare string.
	if wan2Inputs["model"] != "wan2.7-i2v" {
		t.Errorf("inputs[model] = %#v, want \"wan2.7-i2v\"", wan2Inputs["model"])
	}

	// Child fields must be present as dotted-key entries.
	if wan2Inputs["model.prompt"] != "a cat running in slow motion" {
		t.Errorf("inputs[model.prompt] = %#v, want \"a cat running in slow motion\"", wan2Inputs["model.prompt"])
	}

	// Empty-string child value must NOT be dropped.
	negPrompt, exists := wan2Inputs["model.negative_prompt"]
	if !exists {
		t.Error("inputs[model.negative_prompt] must be present (empty string must be preserved for DynamicCombo child)")
	} else if negPrompt != "" {
		t.Errorf("inputs[model.negative_prompt] = %#v, want \"\"", negPrompt)
	}

	if wan2Inputs["model.resolution"] != "720P" {
		t.Errorf("inputs[model.resolution] = %#v, want \"720P\"", wan2Inputs["model.resolution"])
	}
	if wan2Inputs["model.duration"] != int64(5) {
		t.Errorf("inputs[model.duration] = %#v, want int64(5)", wan2Inputs["model.duration"])
	}

	// Connection ref must resolve to [numericID, slotIndex].
	firstFrame, ok := wan2Inputs["first_frame"].([]interface{})
	if !ok || len(firstFrame) != 2 {
		t.Fatalf("inputs[first_frame] = %#v, want [numericID, slotIndex]", wan2Inputs["first_frame"])
	}
}
