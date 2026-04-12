package datasources

import (
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

func TestBuildWorkflowHistoryModel_EnrichesExecutionMetadata(t *testing.T) {
	entry := &client.HistoryEntry{
		Prompt: []interface{}{
			float64(7),
			"prompt-123",
			map[string]interface{}{
				"1": map[string]interface{}{"class_type": "KSampler"},
			},
			map[string]interface{}{
				"create_time": float64(1712345600),
				"extra_pnginfo": map[string]interface{}{
					"workflow": map[string]interface{}{"id": "wf-history"},
				},
			},
			[]interface{}{"5"},
		},
		Outputs: map[string]interface{}{
			"5": map[string]interface{}{
				"images": []interface{}{
					map[string]interface{}{"filename": "image.png", "type": "output"},
				},
			},
		},
		Status: client.ExecutionStatus{
			StatusStr: "error",
			Completed: true,
			Messages: [][]interface{}{
				{"execution_start", map[string]interface{}{"timestamp": float64(100)}},
				{"execution_error", map[string]interface{}{
					"timestamp":         float64(105),
					"exception_type":    "ValueError",
					"exception_message": "broken graph",
				}},
			},
		},
	}

	model, err := buildWorkflowHistoryModel("prompt-123", entry)
	if err != nil {
		t.Fatalf("buildWorkflowHistoryModel failed: %v", err)
	}

	if model.PromptID.ValueString() != "prompt-123" {
		t.Fatalf("expected prompt_id prompt-123, got %q", model.PromptID.ValueString())
	}
	if model.Status.ValueString() != "error" {
		t.Fatalf("expected status error, got %q", model.Status.ValueString())
	}
	if !model.Completed.ValueBool() {
		t.Fatal("expected completed=true")
	}
	if model.CreateTime.ValueInt64() != 1712345600 {
		t.Fatalf("expected create_time 1712345600, got %d", model.CreateTime.ValueInt64())
	}
	if model.ExecutionStartTime.ValueInt64() != 100 {
		t.Fatalf("expected execution_start_time 100, got %d", model.ExecutionStartTime.ValueInt64())
	}
	if model.ExecutionEndTime.ValueInt64() != 105 {
		t.Fatalf("expected execution_end_time 105, got %d", model.ExecutionEndTime.ValueInt64())
	}
	if model.WorkflowID.ValueString() != "wf-history" {
		t.Fatalf("expected workflow_id wf-history, got %q", model.WorkflowID.ValueString())
	}
	if model.OutputsCount.ValueInt64() != 1 {
		t.Fatalf("expected outputs_count 1, got %d", model.OutputsCount.ValueInt64())
	}
	if model.PromptJSON.IsNull() || model.PromptJSON.ValueString() == "" {
		t.Fatal("expected prompt_json to be populated")
	}
	if model.Prompt.IsNull() {
		t.Fatal("expected structured prompt to be populated")
	}
	if model.ExtraDataJSON.IsNull() || model.ExtraDataJSON.ValueString() == "" {
		t.Fatal("expected extra_data_json to be populated")
	}
	if model.ExtraData.IsNull() {
		t.Fatal("expected structured extra_data to be populated")
	}
	if model.OutputsToExecuteJSON.IsNull() || model.OutputsToExecuteJSON.ValueString() == "" {
		t.Fatal("expected outputs_to_execute_json to be populated")
	}
	if model.OutputsToExecute.IsNull() {
		t.Fatal("expected structured outputs_to_execute to be populated")
	}
	if model.ExecutionStatusJSON.IsNull() || model.ExecutionStatusJSON.ValueString() == "" {
		t.Fatal("expected execution_status_json to be populated")
	}
	if model.ExecutionStatus.IsNull() {
		t.Fatal("expected structured execution_status to be populated")
	}
	if model.ExecutionErrorJSON.IsNull() || model.ExecutionErrorJSON.ValueString() == "" {
		t.Fatal("expected execution_error_json to be populated")
	}
	if model.ExecutionError.IsNull() {
		t.Fatal("expected structured execution_error to be populated")
	}
	if model.Outputs.IsNull() || model.Outputs.ValueString() == "" {
		t.Fatal("expected outputs to be populated")
	}
	if model.OutputsStructured.IsNull() {
		t.Fatal("expected structured outputs to be populated")
	}
}

func TestBuildWorkflowHistoryModel_AllowsMissingPromptTuple(t *testing.T) {
	entry := &client.HistoryEntry{
		Outputs: map[string]interface{}{
			"5": map[string]interface{}{
				"images": []interface{}{
					map[string]interface{}{"filename": "image.png", "type": "output"},
				},
			},
		},
		Status: client.ExecutionStatus{
			StatusStr: "success",
			Completed: true,
		},
	}

	model, err := buildWorkflowHistoryModel("prompt-older-shape", entry)
	if err != nil {
		t.Fatalf("expected missing prompt tuple to keep working, got error: %v", err)
	}

	if model.PromptID.ValueString() != "prompt-older-shape" {
		t.Fatalf("expected prompt_id prompt-older-shape, got %q", model.PromptID.ValueString())
	}
	if model.Status.ValueString() != "success" {
		t.Fatalf("expected status success, got %q", model.Status.ValueString())
	}
	if model.Outputs.IsNull() || model.Outputs.ValueString() == "" {
		t.Fatal("expected legacy outputs to remain populated")
	}
	if !model.PromptJSON.IsNull() {
		t.Fatalf("expected prompt_json to stay null when prompt tuple is absent, got %q", model.PromptJSON.ValueString())
	}
	if !model.Prompt.IsNull() {
		t.Fatal("expected prompt to stay null when prompt tuple is absent")
	}
}

func TestBuildWorkflowHistoryModel_AllowsLegacyFourElementPromptTuple(t *testing.T) {
	entry := &client.HistoryEntry{
		Prompt: []interface{}{
			float64(7),
			"prompt-legacy-four",
			map[string]interface{}{
				"1": map[string]interface{}{"class_type": "KSampler"},
			},
			map[string]interface{}{
				"create_time": float64(200),
				"extra_pnginfo": map[string]interface{}{
					"workflow": map[string]interface{}{"id": "wf-legacy-four"},
				},
			},
		},
		Outputs: map[string]interface{}{},
		Status: client.ExecutionStatus{
			StatusStr: "success",
			Completed: true,
		},
	}

	model, err := buildWorkflowHistoryModel("prompt-legacy-four", entry)
	if err != nil {
		t.Fatalf("expected 4-element prompt tuple to remain supported, got error: %v", err)
	}

	if model.CreateTime.ValueInt64() != 200 {
		t.Fatalf("expected create_time 200, got %d", model.CreateTime.ValueInt64())
	}
	if model.WorkflowID.ValueString() != "wf-legacy-four" {
		t.Fatalf("expected workflow_id wf-legacy-four, got %q", model.WorkflowID.ValueString())
	}
	if model.PromptJSON.IsNull() || model.PromptJSON.ValueString() == "" {
		t.Fatal("expected prompt_json to be populated for 4-element prompt tuple")
	}
}

func TestBuildWorkflowHistoryModel_PreservesNonImageOutputs(t *testing.T) {
	entry := &client.HistoryEntry{
		Outputs: map[string]interface{}{
			"12": map[string]interface{}{
				"text": []interface{}{"hello world"},
			},
		},
		Status: client.ExecutionStatus{
			StatusStr: "success",
			Completed: true,
		},
	}

	model, err := buildWorkflowHistoryModel("prompt-text", entry)
	if err != nil {
		t.Fatalf("buildWorkflowHistoryModel failed: %v", err)
	}

	if model.Outputs.ValueString() != `{"12":{"text":["hello world"]}}` {
		t.Fatalf("expected raw text outputs to survive, got %q", model.Outputs.ValueString())
	}
	if model.OutputsStructured.IsNull() {
		t.Fatal("expected structured outputs to preserve non-image payloads")
	}
	if model.OutputsCount.ValueInt64() != 1 {
		t.Fatalf("expected outputs_count 1 for text output, got %d", model.OutputsCount.ValueInt64())
	}
}

func TestBuildWorkflowHistoryModel_NormalizesNilOutputsToEmptyObject(t *testing.T) {
	var outputs map[string]interface{}

	entry := &client.HistoryEntry{
		Outputs: outputs,
		Status: client.ExecutionStatus{
			StatusStr: "success",
			Completed: true,
		},
	}

	model, err := buildWorkflowHistoryModel("prompt-empty", entry)
	if err != nil {
		t.Fatalf("buildWorkflowHistoryModel failed: %v", err)
	}

	if model.Outputs.ValueString() != "{}" {
		t.Fatalf("expected nil outputs to normalize to {}, got %q", model.Outputs.ValueString())
	}
	if model.OutputsStructured.IsNull() {
		t.Fatal("expected structured outputs to stay object-shaped for nil outputs")
	}
}
