package resources

import (
	"testing"
)

// helper UUIDs for test nodes
const (
	uuidCheckpoint = "00000000-0000-0000-0000-000000000001"
	uuidCLIP       = "00000000-0000-0000-0000-000000000002"
	uuidKSampler   = "00000000-0000-0000-0000-000000000003"
	uuidVAEDecode  = "00000000-0000-0000-0000-000000000004"
	uuidSaveImage  = "00000000-0000-0000-0000-000000000005"
	uuidMissing    = "ffffffff-ffff-ffff-ffff-ffffffffffff"
)

// fullPipeline returns a valid checkpoint→CLIP→KSampler→VAEDecode→SaveImage graph.
func fullPipeline() []NodeState {
	return []NodeState{
		{ID: uuidCheckpoint, ClassType: "CheckpointLoaderSimple", Inputs: map[string]interface{}{
			"ckpt_name": "v1-5-pruned.safetensors",
		}},
		{ID: uuidCLIP, ClassType: "CLIPTextEncode", Inputs: map[string]interface{}{
			"text": "a photo of a cat",
			"clip": uuidCheckpoint + ":1",
		}},
		{ID: uuidKSampler, ClassType: "KSampler", Inputs: map[string]interface{}{
			"model":    uuidCheckpoint + ":0",
			"positive": uuidCLIP + ":0",
			"seed":     42,
			"steps":    20,
		}},
		{ID: uuidVAEDecode, ClassType: "VAEDecode", Inputs: map[string]interface{}{
			"samples": uuidKSampler + ":0",
			"vae":     uuidCheckpoint + ":2",
		}},
		{ID: uuidSaveImage, ClassType: "SaveImage", Inputs: map[string]interface{}{
			"images":          uuidVAEDecode + ":0",
			"filename_prefix": "ComfyUI",
		}},
	}
}

func TestValidateNodeConnections_ValidCompleteGraph(t *testing.T) {
	errs := ValidateNodeConnections(fullPipeline())
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors for valid graph, got %d: %v", len(errs), errs)
	}
}

func TestValidateNodeConnections_MissingDependency(t *testing.T) {
	nodes := []NodeState{
		{ID: uuidKSampler, ClassType: "KSampler", Inputs: map[string]interface{}{
			"model": uuidMissing + ":0",
			"seed":  42,
		}},
		{ID: uuidSaveImage, ClassType: "SaveImage", Inputs: map[string]interface{}{
			"images":          uuidKSampler + ":0",
			"filename_prefix": "ComfyUI",
		}},
	}

	errs := ValidateNodeConnections(nodes)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	e := errs[0]
	if e.SourceNodeID != uuidKSampler {
		t.Errorf("SourceNodeID = %q, want %q", e.SourceNodeID, uuidKSampler)
	}
	if e.SourceClassType != "KSampler" {
		t.Errorf("SourceClassType = %q, want %q", e.SourceClassType, "KSampler")
	}
	if e.InputName != "model" {
		t.Errorf("InputName = %q, want %q", e.InputName, "model")
	}
	if e.ReferencedUUID != uuidMissing {
		t.Errorf("ReferencedUUID = %q, want %q", e.ReferencedUUID, uuidMissing)
	}
	if e.SlotIndex != 0 {
		t.Errorf("SlotIndex = %d, want 0", e.SlotIndex)
	}
}

func TestValidateNodeConnections_MultipleBrokenRefs(t *testing.T) {
	missingA := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	missingB := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	nodes := []NodeState{
		{ID: uuidKSampler, ClassType: "KSampler", Inputs: map[string]interface{}{
			"model":    missingA + ":0",
			"positive": missingB + ":0",
		}},
	}

	errs := ValidateNodeConnections(nodes)
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d: %v", len(errs), errs)
	}

	refSet := map[string]bool{}
	for _, e := range errs {
		refSet[e.ReferencedUUID] = true
	}
	if !refSet[missingA] {
		t.Errorf("expected error referencing %q", missingA)
	}
	if !refSet[missingB] {
		t.Errorf("expected error referencing %q", missingB)
	}
}

func TestValidateNodeConnections_NoConnections(t *testing.T) {
	nodes := []NodeState{
		{ID: uuidCheckpoint, ClassType: "CheckpointLoaderSimple", Inputs: map[string]interface{}{
			"ckpt_name": "v1-5-pruned.safetensors",
		}},
	}

	errs := ValidateNodeConnections(nodes)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors for standalone node, got %d: %v", len(errs), errs)
	}
}

func TestValidateHasOutputNode_WithSaveImage(t *testing.T) {
	nodes := fullPipeline()
	if !ValidateHasOutputNode(nodes) {
		t.Fatal("expected output node detection for pipeline with SaveImage")
	}
}

func TestValidateHasOutputNode_MissingOutputNode(t *testing.T) {
	nodes := []NodeState{
		{ID: uuidCheckpoint, ClassType: "CheckpointLoaderSimple", Inputs: map[string]interface{}{
			"ckpt_name": "v1-5-pruned.safetensors",
		}},
		{ID: uuidCLIP, ClassType: "CLIPTextEncode", Inputs: map[string]interface{}{
			"text": "a photo of a cat",
			"clip": uuidCheckpoint + ":1",
		}},
	}

	if ValidateHasOutputNode(nodes) {
		t.Fatal("expected no output node detected for processing-only workflow")
	}
}

func TestValidateHasOutputNode_CustomSaveNode(t *testing.T) {
	nodes := []NodeState{
		{ID: uuidCheckpoint, ClassType: "CustomSaveNode", Inputs: map[string]interface{}{}},
	}
	if !ValidateHasOutputNode(nodes) {
		t.Fatal("expected output node detection for class type containing 'Save'")
	}
}

func TestValidateHasOutputNode_CustomPreviewNode(t *testing.T) {
	nodes := []NodeState{
		{ID: uuidCheckpoint, ClassType: "MyPreviewWidget", Inputs: map[string]interface{}{}},
	}
	if !ValidateHasOutputNode(nodes) {
		t.Fatal("expected output node detection for class type containing 'Preview'")
	}
}

func TestValidateNodeConnections_MixedValidAndBroken(t *testing.T) {
	// Checkpoint exists; CLIP refs checkpoint (valid); KSampler refs missing (broken)
	nodes := []NodeState{
		{ID: uuidCheckpoint, ClassType: "CheckpointLoaderSimple", Inputs: map[string]interface{}{
			"ckpt_name": "v1-5-pruned.safetensors",
		}},
		{ID: uuidCLIP, ClassType: "CLIPTextEncode", Inputs: map[string]interface{}{
			"text": "a photo of a cat",
			"clip": uuidCheckpoint + ":1",
		}},
		{ID: uuidKSampler, ClassType: "KSampler", Inputs: map[string]interface{}{
			"model":    uuidMissing + ":0",
			"positive": uuidCLIP + ":0",
		}},
	}

	errs := ValidateNodeConnections(nodes)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].ReferencedUUID != uuidMissing {
		t.Errorf("expected broken ref to %q, got %q", uuidMissing, errs[0].ReferencedUUID)
	}
}

func TestValidateWorkflow_CombinesDiagnostics(t *testing.T) {
	// No output node + broken ref → at least 2 errors
	nodes := []NodeState{
		{ID: uuidKSampler, ClassType: "KSampler", Inputs: map[string]interface{}{
			"model": uuidMissing + ":0",
		}},
	}

	errs := ValidateWorkflow(nodes)
	if len(errs) < 2 {
		t.Fatalf("expected at least 2 errors (broken ref + missing output), got %d: %v", len(errs), errs)
	}

	hasBrokenRef := false
	hasMissingOutput := false
	for _, e := range errs {
		if e.ReferencedUUID == uuidMissing {
			hasBrokenRef = true
		}
		if e.SourceNodeID == "workflow" && e.InputName == "output" {
			hasMissingOutput = true
		}
	}
	if !hasBrokenRef {
		t.Error("expected a broken-ref error")
	}
	if !hasMissingOutput {
		t.Error("expected a missing-output-node error")
	}
}

func TestValidateWorkflow_ValidPipeline(t *testing.T) {
	errs := ValidateWorkflow(fullPipeline())
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors for valid pipeline, got %d: %v", len(errs), errs)
	}
}

func TestValidationError_ErrorString(t *testing.T) {
	e := ValidationError{
		SourceNodeID:    uuidKSampler,
		SourceClassType: "KSampler",
		InputName:       "model",
		ReferencedUUID:  uuidMissing,
		SlotIndex:       0,
	}
	s := e.Error()
	if s == "" {
		t.Fatal("Error() returned empty string")
	}
	// Verify key parts are present.
	for _, want := range []string{"KSampler", "model", uuidMissing} {
		if !contains(s, want) {
			t.Errorf("Error() = %q, missing %q", s, want)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
