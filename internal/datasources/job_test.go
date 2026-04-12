package datasources

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestBuildJobModel_BasicFields(t *testing.T) {
	job := &client.Job{
		ID:       "job-123",
		Status:   "completed",
		Priority: 5,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	if model.ID.ValueString() != "job-123" {
		t.Errorf("expected ID 'job-123', got %q", model.ID.ValueString())
	}

	if model.Status.ValueString() != "completed" {
		t.Errorf("expected Status 'completed', got %q", model.Status.ValueString())
	}

	if model.Priority.ValueInt64() != 5 {
		t.Errorf("expected Priority 5, got %d", model.Priority.ValueInt64())
	}
}

func TestInterfaceToAttrValue_PreservesLargeInt64Precision(t *testing.T) {
	attrVal, err := interfaceToAttrValue(int64(9007199254740993))
	if err != nil {
		t.Fatalf("interfaceToAttrValue failed: %v", err)
	}

	numVal, ok := attrVal.(basetypes.NumberValue)
	if !ok {
		t.Fatalf("expected number value, got %T", attrVal)
	}

	expected := new(big.Float).SetPrec(64).SetInt64(9007199254740993)
	if numVal.ValueBigFloat().Cmp(expected) != 0 {
		t.Fatalf("expected exact %s, got %s", expected.Text('f', 0), numVal.ValueBigFloat().Text('f', 0))
	}
}

func TestBuildJobModel_TimestampFields(t *testing.T) {
	job := &client.Job{
		ID:                 "job-456",
		Status:             "running",
		Priority:           0,
		CreateTime:         int64Ptr(1700000000),
		ExecutionStartTime: int64Ptr(1700000100),
		ExecutionEndTime:   int64Ptr(0), // not yet finished
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	if model.CreateTime.ValueInt64() != 1700000000 {
		t.Errorf("expected CreateTime 1700000000, got %d", model.CreateTime.ValueInt64())
	}

	if model.ExecutionStartTime.ValueInt64() != 1700000100 {
		t.Errorf("expected ExecutionStartTime 1700000100, got %d", model.ExecutionStartTime.ValueInt64())
	}

	// ExecutionEndTime is 0, should be represented as 0
	if model.ExecutionEndTime.ValueInt64() != 0 {
		t.Errorf("expected ExecutionEndTime 0, got %d", model.ExecutionEndTime.ValueInt64())
	}
}

func TestBuildJobModel_OutputsCount(t *testing.T) {
	job := &client.Job{
		ID:           "job-789",
		Status:       "completed",
		Priority:     0,
		OutputsCount: intPtr(3),
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	if model.OutputsCount.ValueInt64() != 3 {
		t.Errorf("expected OutputsCount 3, got %d", model.OutputsCount.ValueInt64())
	}
}

func TestBuildJobModel_NullNumericFieldsWhenOmitted(t *testing.T) {
	job := &client.Job{
		ID:     "job-null-numerics",
		Status: "pending",
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	if !model.CreateTime.IsNull() {
		t.Fatalf("expected create_time to be null when omitted, got %d", model.CreateTime.ValueInt64())
	}
	if !model.ExecutionStartTime.IsNull() {
		t.Fatalf("expected execution_start_time to be null when omitted, got %d", model.ExecutionStartTime.ValueInt64())
	}
	if !model.ExecutionEndTime.IsNull() {
		t.Fatalf("expected execution_end_time to be null when omitted, got %d", model.ExecutionEndTime.ValueInt64())
	}
	if !model.OutputsCount.IsNull() {
		t.Fatalf("expected outputs_count to be null when omitted, got %d", model.OutputsCount.ValueInt64())
	}
}

func TestBuildJobModel_WorkflowID(t *testing.T) {
	job := &client.Job{
		ID:         "job-111",
		Status:     "pending",
		Priority:   1,
		WorkflowID: "workflow-abc",
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	if model.WorkflowID.ValueString() != "workflow-abc" {
		t.Errorf("expected WorkflowID 'workflow-abc', got %q", model.WorkflowID.ValueString())
	}
}

func TestBuildJobModel_NullWorkflowID(t *testing.T) {
	job := &client.Job{
		ID:         "job-222",
		Status:     "pending",
		Priority:   0,
		WorkflowID: "", // empty
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	if !model.WorkflowID.IsNull() {
		t.Errorf("expected WorkflowID to be null, got %q", model.WorkflowID.ValueString())
	}
}

func TestBuildJobModel_PreviewOutputJSON(t *testing.T) {
	previewOutput := map[string]interface{}{
		"node_id": "42",
		"images":  []interface{}{"preview.png"},
	}

	job := &client.Job{
		ID:            "job-333",
		Status:        "completed",
		Priority:      0,
		PreviewOutput: previewOutput,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	if model.PreviewOutputJSON.IsNull() {
		t.Fatal("expected PreviewOutputJSON to be set")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(model.PreviewOutputJSON.ValueString()), &decoded); err != nil {
		t.Fatalf("PreviewOutputJSON is not valid JSON: %v", err)
	}

	if decoded["node_id"] != "42" {
		t.Errorf("expected node_id '42', got %v", decoded["node_id"])
	}
}

func TestBuildJobModel_NullPreviewOutput(t *testing.T) {
	job := &client.Job{
		ID:            "job-444",
		Status:        "pending",
		Priority:      0,
		PreviewOutput: nil,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	if !model.PreviewOutputJSON.IsNull() {
		t.Errorf("expected PreviewOutputJSON to be null, got %q", model.PreviewOutputJSON.ValueString())
	}
}

func TestBuildJobModel_OutputsJSON(t *testing.T) {
	outputs := map[string]interface{}{
		"9": map[string]interface{}{
			"images": []interface{}{
				map[string]interface{}{
					"filename":  "output.png",
					"subfolder": "",
					"type":      "output",
				},
			},
		},
	}

	job := &client.Job{
		ID:       "job-555",
		Status:   "completed",
		Priority: 0,
		Outputs:  outputs,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	if model.OutputsJSON.IsNull() {
		t.Fatal("expected OutputsJSON to be set")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(model.OutputsJSON.ValueString()), &decoded); err != nil {
		t.Fatalf("OutputsJSON is not valid JSON: %v", err)
	}

	if _, ok := decoded["9"]; !ok {
		t.Error("expected outputs to contain node '9'")
	}
}

func TestBuildJobModel_ExecutionStatusJSON(t *testing.T) {
	executionStatus := map[string]interface{}{
		"status_str": "success",
		"completed":  true,
	}

	job := &client.Job{
		ID:              "job-666",
		Status:          "completed",
		Priority:        0,
		ExecutionStatus: executionStatus,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	if model.ExecutionStatusJSON.IsNull() {
		t.Fatal("expected ExecutionStatusJSON to be set")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(model.ExecutionStatusJSON.ValueString()), &decoded); err != nil {
		t.Fatalf("ExecutionStatusJSON is not valid JSON: %v", err)
	}

	if decoded["status_str"] != "success" {
		t.Errorf("expected status_str 'success', got %v", decoded["status_str"])
	}
}

func TestBuildJobModel_ExecutionErrorJSON(t *testing.T) {
	executionError := map[string]interface{}{
		"exception_message": "Node validation failed",
		"node_id":           "7",
	}

	job := &client.Job{
		ID:             "job-777",
		Status:         "error",
		Priority:       0,
		ExecutionError: executionError,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	if model.ExecutionErrorJSON.IsNull() {
		t.Fatal("expected ExecutionErrorJSON to be set")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(model.ExecutionErrorJSON.ValueString()), &decoded); err != nil {
		t.Fatalf("ExecutionErrorJSON is not valid JSON: %v", err)
	}

	if decoded["node_id"] != "7" {
		t.Errorf("expected node_id '7', got %v", decoded["node_id"])
	}
}

func TestBuildJobModel_WorkflowJSON(t *testing.T) {
	workflow := &client.JobWorkflow{
		Prompt: map[string]interface{}{
			"3": map[string]interface{}{
				"class_type": "KSampler",
			},
		},
		ExtraData: map[string]interface{}{
			"client_id": "test-client",
		},
	}

	job := &client.Job{
		ID:       "job-888",
		Status:   "completed",
		Priority: 0,
		Workflow: workflow,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	if model.WorkflowJSON.IsNull() {
		t.Fatal("expected WorkflowJSON to be set")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(model.WorkflowJSON.ValueString()), &decoded); err != nil {
		t.Fatalf("WorkflowJSON is not valid JSON: %v", err)
	}

	if _, ok := decoded["prompt"]; !ok {
		t.Error("expected workflow to contain 'prompt' field")
	}

	if _, ok := decoded["extra_data"]; !ok {
		t.Error("expected workflow to contain 'extra_data' field")
	}
}

func TestBuildJobModel_NullWorkflow(t *testing.T) {
	job := &client.Job{
		ID:       "job-999",
		Status:   "pending",
		Priority: 0,
		Workflow: nil,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	if !model.WorkflowJSON.IsNull() {
		t.Errorf("expected WorkflowJSON to be null, got %q", model.WorkflowJSON.ValueString())
	}
}

func TestJobDataSource_Factory(t *testing.T) {
	ds := NewJobDataSource()
	if ds == nil {
		t.Fatal("factory returned nil data source")
	}

	jobDS, ok := ds.(*JobDataSource)
	if !ok {
		t.Fatal("factory returned wrong type")
	}

	if jobDS.client != nil {
		t.Error("expected client to be nil before Configure")
	}
}

func TestBuildJobModel_JSONMarshalFailure(t *testing.T) {
	// Create a job with an un-marshalable field (e.g., a channel)
	job := &client.Job{
		ID:       "job-unmarshalable",
		Status:   "completed",
		Priority: 0,
		Outputs: map[string]interface{}{
			"broken": make(chan int), // channels cannot be marshaled to JSON
		},
	}

	_, err := buildJobModel(job)
	if err == nil {
		t.Fatal("expected error from buildJobModel with un-marshalable Outputs, got nil")
	}

	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

// TDD: Structured field tests - these should fail until we add the structured attributes

func TestBuildJobModel_StructuredPreviewOutput(t *testing.T) {
	previewOutput := map[string]interface{}{
		"node_id": "42",
		"images":  []interface{}{"preview.png"},
	}

	job := &client.Job{
		ID:            "job-structured-preview",
		Status:        "completed",
		Priority:      0,
		PreviewOutput: previewOutput,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	// Verify structured field is set (not null)
	if model.PreviewOutput.IsNull() {
		t.Fatal("expected PreviewOutput (structured) to be set")
	}

	// Verify it's a dynamic type that can be unmarshaled back
	if model.PreviewOutput.IsUnknown() {
		t.Fatal("expected PreviewOutput to not be unknown")
	}

	// Verify JSON field still works for backward compatibility
	if model.PreviewOutputJSON.IsNull() {
		t.Fatal("expected PreviewOutputJSON to still be set for backward compatibility")
	}
}

func TestBuildJobModel_StructuredOutputs(t *testing.T) {
	outputs := map[string]interface{}{
		"9": map[string]interface{}{
			"images": []interface{}{
				map[string]interface{}{
					"filename":  "output.png",
					"subfolder": "",
					"type":      "output",
				},
			},
		},
	}

	job := &client.Job{
		ID:       "job-structured-outputs",
		Status:   "completed",
		Priority: 0,
		Outputs:  outputs,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	// Verify structured field is set
	if model.Outputs.IsNull() {
		t.Fatal("expected Outputs (structured) to be set")
	}

	if model.Outputs.IsUnknown() {
		t.Fatal("expected Outputs to not be unknown")
	}

	// Verify JSON field still works
	if model.OutputsJSON.IsNull() {
		t.Fatal("expected OutputsJSON to still be set for backward compatibility")
	}
}

func TestBuildJobModel_StructuredExecutionStatus(t *testing.T) {
	executionStatus := map[string]interface{}{
		"status_str": "success",
		"completed":  true,
		"messages":   []interface{}{"Processing complete"},
	}

	job := &client.Job{
		ID:              "job-structured-status",
		Status:          "completed",
		Priority:        0,
		ExecutionStatus: executionStatus,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	// Verify structured field is set
	if model.ExecutionStatus.IsNull() {
		t.Fatal("expected ExecutionStatus (structured) to be set")
	}

	if model.ExecutionStatus.IsUnknown() {
		t.Fatal("expected ExecutionStatus to not be unknown")
	}

	// Verify JSON field still works
	if model.ExecutionStatusJSON.IsNull() {
		t.Fatal("expected ExecutionStatusJSON to still be set for backward compatibility")
	}
}

func TestBuildJobModel_StructuredExecutionError(t *testing.T) {
	executionError := map[string]interface{}{
		"exception_message": "Node validation failed",
		"node_id":           "7",
		"traceback":         []interface{}{"line 1", "line 2"},
	}

	job := &client.Job{
		ID:             "job-structured-error",
		Status:         "error",
		Priority:       0,
		ExecutionError: executionError,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	// Verify structured field is set
	if model.ExecutionError.IsNull() {
		t.Fatal("expected ExecutionError (structured) to be set")
	}

	if model.ExecutionError.IsUnknown() {
		t.Fatal("expected ExecutionError to not be unknown")
	}

	// Verify JSON field still works
	if model.ExecutionErrorJSON.IsNull() {
		t.Fatal("expected ExecutionErrorJSON to still be set for backward compatibility")
	}
}

func TestBuildJobModel_StructuredWorkflow(t *testing.T) {
	workflow := &client.JobWorkflow{
		Prompt: map[string]interface{}{
			"3": map[string]interface{}{
				"class_type": "KSampler",
				"inputs": map[string]interface{}{
					"seed": 42,
				},
			},
		},
		ExtraData: map[string]interface{}{
			"client_id": "test-client",
		},
	}

	job := &client.Job{
		ID:       "job-structured-workflow",
		Status:   "completed",
		Priority: 0,
		Workflow: workflow,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	// Verify structured field is set
	if model.Workflow.IsNull() {
		t.Fatal("expected Workflow (structured) to be set")
	}

	if model.Workflow.IsUnknown() {
		t.Fatal("expected Workflow to not be unknown")
	}

	// Verify JSON field still works
	if model.WorkflowJSON.IsNull() {
		t.Fatal("expected WorkflowJSON to still be set for backward compatibility")
	}
}

func TestBuildJobModel_StructuredNullValues(t *testing.T) {
	job := &client.Job{
		ID:              "job-null-structured",
		Status:          "pending",
		Priority:        0,
		PreviewOutput:   nil,
		Outputs:         nil,
		ExecutionStatus: nil,
		ExecutionError:  nil,
		Workflow:        nil,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	// Verify all structured fields are null when source is nil
	if !model.PreviewOutput.IsNull() {
		t.Error("expected PreviewOutput (structured) to be null")
	}

	if !model.Outputs.IsNull() {
		t.Error("expected Outputs (structured) to be null")
	}

	if !model.ExecutionStatus.IsNull() {
		t.Error("expected ExecutionStatus (structured) to be null")
	}

	if !model.ExecutionError.IsNull() {
		t.Error("expected ExecutionError (structured) to be null")
	}

	if !model.Workflow.IsNull() {
		t.Error("expected Workflow (structured) to be null")
	}

	// Verify JSON fields are also null for backward compatibility
	if !model.PreviewOutputJSON.IsNull() {
		t.Error("expected PreviewOutputJSON to be null")
	}

	if !model.OutputsJSON.IsNull() {
		t.Error("expected OutputsJSON to be null")
	}

	if !model.ExecutionStatusJSON.IsNull() {
		t.Error("expected ExecutionStatusJSON to be null")
	}

	if !model.ExecutionErrorJSON.IsNull() {
		t.Error("expected ExecutionErrorJSON to be null")
	}

	if !model.WorkflowJSON.IsNull() {
		t.Error("expected WorkflowJSON to be null")
	}
}

// TDD: Test for null propagation in nested structured data
func TestBuildJobModel_NestedNullPropagation(t *testing.T) {
	// Test that nil values in nested structures preserve dynamic null semantics
	outputs := map[string]interface{}{
		"node_1": map[string]interface{}{
			"result":   "success",
			"metadata": nil, // null value in nested object
			"optional": nil,
		},
		"node_2": map[string]interface{}{
			"images": []interface{}{
				map[string]interface{}{
					"filename": "test.png",
					"tags":     nil, // null in array element
				},
			},
		},
	}

	job := &client.Job{
		ID:       "job-nested-nulls",
		Status:   "completed",
		Priority: 0,
		Outputs:  outputs,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	// Verify structured field is set
	if model.Outputs.IsNull() {
		t.Fatal("expected Outputs (structured) to be set")
	}

	// Verify we can access the underlying value
	underlyingValue := model.Outputs.UnderlyingValue()
	if underlyingValue == nil {
		t.Fatal("expected non-nil underlying value")
	}

	// The conversion should succeed without error - nulls should be preserved as dynamic nulls
	// not converted to string nulls
}

// TDD: Test for heterogeneous array handling in structured fields
func TestBuildJobModel_HeterogeneousArrays(t *testing.T) {
	// Test that arrays with mixed types are handled correctly
	// Real-world case: API responses may have arrays like [1, "string", true, null, {"key": "value"}]
	executionStatus := map[string]interface{}{
		"messages": []interface{}{
			"Processing started", // string
			42,                   // number
			true,                 // bool
			nil,                  // null
			map[string]interface{}{ // object
				"timestamp": "2024-01-01",
				"level":     "info",
			},
			[]interface{}{"nested", "array"}, // nested array
		},
		"metadata": []interface{}{
			map[string]interface{}{"id": 1},
			map[string]interface{}{"id": 2},
			nil, // null in middle of objects
			map[string]interface{}{"id": 3},
		},
	}

	job := &client.Job{
		ID:              "job-hetero-arrays",
		Status:          "running",
		Priority:        0,
		ExecutionStatus: executionStatus,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	// Verify structured field is set
	if model.ExecutionStatus.IsNull() {
		t.Fatal("expected ExecutionStatus (structured) to be set")
	}

	// Verify we can access the underlying value
	underlyingValue := model.ExecutionStatus.UnderlyingValue()
	if underlyingValue == nil {
		t.Fatal("expected non-nil underlying value")
	}

	// The conversion should succeed - heterogeneous arrays should be supported
	// using tuple types instead of homogeneous list types
}

func intPtr(v int) *int {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

// TDD: Test for deeply nested structures with mixed types and nulls
func TestBuildJobModel_DeeplyNestedMixedTypes(t *testing.T) {
	outputs := map[string]interface{}{
		"complex_node": map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": map[string]interface{}{
						"mixed_array": []interface{}{
							1,
							"two",
							nil,
							map[string]interface{}{
								"nested": true,
							},
						},
						"null_field": nil,
					},
					"another_null": nil,
				},
			},
			"parallel_branch": []interface{}{
				nil,
				map[string]interface{}{
					"id":   1,
					"data": nil,
				},
				[]interface{}{
					"nested",
					nil,
					42,
				},
			},
		},
	}

	job := &client.Job{
		ID:       "job-deeply-nested",
		Status:   "completed",
		Priority: 0,
		Outputs:  outputs,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	// Verify structured field is set
	if model.Outputs.IsNull() {
		t.Fatal("expected Outputs (structured) to be set")
	}

	// Verify we can access the underlying value
	underlyingValue := model.Outputs.UnderlyingValue()
	if underlyingValue == nil {
		t.Fatal("expected non-nil underlying value")
	}

	// Should handle deep nesting with mixed types and nulls correctly
}

// TDD: Test empty arrays and objects
func TestBuildJobModel_EmptyCollections(t *testing.T) {
	executionStatus := map[string]interface{}{
		"empty_array":  []interface{}{},
		"empty_object": map[string]interface{}{},
		"nested_empty": map[string]interface{}{
			"arrays": []interface{}{
				[]interface{}{},
				[]interface{}{},
			},
			"objects": []interface{}{
				map[string]interface{}{},
				map[string]interface{}{},
			},
		},
	}

	job := &client.Job{
		ID:              "job-empty-collections",
		Status:          "completed",
		Priority:        0,
		ExecutionStatus: executionStatus,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	// Verify structured field is set
	if model.ExecutionStatus.IsNull() {
		t.Fatal("expected ExecutionStatus (structured) to be set")
	}

	// Empty collections should be preserved correctly
	underlyingValue := model.ExecutionStatus.UnderlyingValue()
	if underlyingValue == nil {
		t.Fatal("expected non-nil underlying value")
	}
}

// TDD: Test array with all nulls
func TestBuildJobModel_AllNullArray(t *testing.T) {
	outputs := map[string]interface{}{
		"node_1": map[string]interface{}{
			"all_nulls": []interface{}{nil, nil, nil},
			"mixed":     []interface{}{nil, "value", nil},
		},
	}

	job := &client.Job{
		ID:       "job-all-null-array",
		Status:   "completed",
		Priority: 0,
		Outputs:  outputs,
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	// Verify structured field is set
	if model.Outputs.IsNull() {
		t.Fatal("expected Outputs (structured) to be set")
	}

	// Arrays with all nulls should be handled correctly
	underlyingValue := model.Outputs.UnderlyingValue()
	if underlyingValue == nil {
		t.Fatal("expected non-nil underlying value")
	}
}

// Regression test for Fix 1: Preserve typed-nil nested maps as nulls
func TestBuildJobModel_WorkflowTypedNilMapsPreserveNull(t *testing.T) {
	job := &client.Job{
		ID:       "job-typed-nil-workflow",
		Status:   "completed",
		Priority: 0,
		Workflow: &client.JobWorkflow{
			Prompt:    nil, // typed-nil map
			ExtraData: nil, // typed-nil map
		},
	}

	model, err := buildJobModel(job)
	if err != nil {
		t.Fatalf("buildJobModel failed: %v", err)
	}

	// Verify workflow_json is set (backward compatibility)
	if model.WorkflowJSON.IsNull() {
		t.Error("expected WorkflowJSON to be set for non-nil workflow")
	}

	// Verify the JSON representation shows null fields properly
	var workflowJSON map[string]interface{}
	if err := json.Unmarshal([]byte(model.WorkflowJSON.ValueString()), &workflowJSON); err != nil {
		t.Fatalf("WorkflowJSON is not valid JSON: %v", err)
	}

	// nil maps should marshal to null in JSON
	promptVal, hasPrompt := workflowJSON["prompt"]
	if !hasPrompt {
		t.Error("expected 'prompt' field in workflow JSON")
	}
	if promptVal != nil {
		t.Errorf("expected 'prompt' to be null in JSON, got %v (type %T)", promptVal, promptVal)
	}

	extraDataVal, hasExtraData := workflowJSON["extra_data"]
	if !hasExtraData {
		t.Error("expected 'extra_data' field in workflow JSON")
	}
	if extraDataVal != nil {
		t.Errorf("expected 'extra_data' to be null in JSON, got %v (type %T)", extraDataVal, extraDataVal)
	}

	// Verify structured workflow field preserves null semantics
	if model.Workflow.IsNull() {
		t.Error("expected Workflow (structured) to be non-null when workflow object exists")
	}

	// The structured field should have null values for the nested maps
	underlyingValue := model.Workflow.UnderlyingValue()
	if underlyingValue == nil {
		t.Fatal("expected non-nil underlying value for workflow")
	}

	// Extract the object and verify nested fields are null
	objVal, ok := underlyingValue.(basetypes.ObjectValue)
	if !ok {
		t.Fatalf("expected ObjectValue, got %T", underlyingValue)
	}

	attrs := objVal.Attributes()
	promptAttr, hasPrompt := attrs["prompt"]
	if !hasPrompt {
		t.Error("expected 'prompt' attribute in workflow object")
	}
	if !promptAttr.IsNull() {
		t.Errorf("expected 'prompt' attribute to be null, got: %v", promptAttr)
	}

	extraDataAttr, hasExtraData := attrs["extra_data"]
	if !hasExtraData {
		t.Error("expected 'extra_data' attribute in workflow object")
	}
	if !extraDataAttr.IsNull() {
		t.Errorf("expected 'extra_data' attribute to be null, got: %v", extraDataAttr)
	}
}
