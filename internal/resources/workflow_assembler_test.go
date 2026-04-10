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
