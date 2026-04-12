package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestBuildQueuePromptRequest_PreservesExplicitMetadataAndAddsPromptPNGInfo(t *testing.T) {
	prompt := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "KSampler",
			"inputs":     map[string]interface{}{},
		},
	}

	req, err := buildQueuePromptRequest(prompt, workflowExecutionRequestConfig{
		PromptID:                "prompt-123",
		ClientID:                "client-456",
		ExtraDataJSON:           `{"tenant":"dev","extra_pnginfo":{"workflow":{"id":"workspace-1"}}}`,
		PartialExecutionTargets: []string{"3", "7"},
	})
	if err != nil {
		t.Fatalf("buildQueuePromptRequest returned error: %v", err)
	}

	if req.PromptID != "prompt-123" {
		t.Fatalf("expected prompt_id to be preserved, got %q", req.PromptID)
	}
	if req.ClientID != "client-456" {
		t.Fatalf("expected client_id to be preserved, got %q", req.ClientID)
	}
	if len(req.PartialExecutionTargets) != 2 {
		t.Fatalf("expected partial execution targets to be preserved, got %#v", req.PartialExecutionTargets)
	}
	if req.ExtraData["tenant"] != "dev" {
		t.Fatalf("expected extra_data.tenant to be preserved, got %#v", req.ExtraData["tenant"])
	}

	extraPNGInfo, ok := req.ExtraData["extra_pnginfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected extra_pnginfo object, got %#v", req.ExtraData["extra_pnginfo"])
	}
	if _, ok := extraPNGInfo["prompt"]; !ok {
		t.Fatal("expected extra_pnginfo.prompt to be added automatically")
	}
	workflow, ok := extraPNGInfo["workflow"].(map[string]interface{})
	if !ok || workflow["id"] != "workspace-1" {
		t.Fatalf("expected explicit workflow metadata to be preserved, got %#v", extraPNGInfo["workflow"])
	}
}

func TestWorkflowAttrValueFromAny_PreservesLargeInt64Precision(t *testing.T) {
	attrVal, err := workflowAttrValueFromAny(int64(9007199254740993))
	if err != nil {
		t.Fatalf("workflowAttrValueFromAny failed: %v", err)
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

func TestBuildQueuePromptRequest_PreservesExplicitPromptPNGInfo(t *testing.T) {
	prompt := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "SaveImage",
			"inputs":     map[string]interface{}{},
		},
	}

	req, err := buildQueuePromptRequest(prompt, workflowExecutionRequestConfig{
		ExtraDataJSON: `{"extra_pnginfo":{"prompt":{"sentinel":"keep-me"}}}`,
	})
	if err != nil {
		t.Fatalf("buildQueuePromptRequest returned error: %v", err)
	}

	extraPNGInfo, ok := req.ExtraData["extra_pnginfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected extra_pnginfo object, got %#v", req.ExtraData["extra_pnginfo"])
	}
	explicitPrompt, ok := extraPNGInfo["prompt"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected explicit prompt metadata to remain an object, got %#v", extraPNGInfo["prompt"])
	}
	if explicitPrompt["sentinel"] != "keep-me" {
		t.Fatalf("expected explicit prompt metadata to be preserved, got %#v", explicitPrompt)
	}
	if _, ok := extraPNGInfo["workflow"]; ok {
		t.Fatalf("expected workflow metadata to remain absent when none was provided, got %#v", extraPNGInfo["workflow"])
	}
}

func TestBuildQueuePromptRequest_InvalidExtraDataJSON(t *testing.T) {
	_, err := buildQueuePromptRequest(map[string]interface{}{}, workflowExecutionRequestConfig{
		ExtraDataJSON: "{not-json}",
	})
	if err == nil {
		t.Fatal("expected invalid extra_data_json to return an error")
	}
}

func TestBuildQueuePromptRequest_ReturnsClientRequest(t *testing.T) {
	req, err := buildQueuePromptRequest(map[string]interface{}{}, workflowExecutionRequestConfig{})
	if err != nil {
		t.Fatalf("buildQueuePromptRequest returned error: %v", err)
	}
	if req.PromptID != "" || req.ClientID != "" {
		t.Fatalf("expected empty config to leave prompt_id/client_id unset, got %#v", req)
	}
	if len(req.PartialExecutionTargets) != 0 {
		t.Fatalf("expected empty config to leave partial targets empty, got %#v", req.PartialExecutionTargets)
	}
	extraPNGInfo, ok := req.ExtraData["extra_pnginfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected empty config to still add extra_pnginfo.prompt, got %#v", req.ExtraData)
	}
	if _, ok := extraPNGInfo["prompt"]; !ok {
		t.Fatalf("expected empty config to still add extra_pnginfo.prompt, got %#v", extraPNGInfo)
	}
	if !reflect.DeepEqual(req.Prompt, map[string]interface{}{}) {
		t.Fatalf("expected empty config to preserve prompt map, got %#v", req.Prompt)
	}
}

func TestWorkflowSchema_PromptIDDoesNotReusePriorState(t *testing.T) {
	r := NewWorkflowResource().(*WorkflowResource)
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	promptIDAttr, ok := resp.Schema.Attributes["prompt_id"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatalf("expected prompt_id to be a string attribute, got %#v", resp.Schema.Attributes["prompt_id"])
	}
	if len(promptIDAttr.PlanModifiers) != 0 {
		t.Fatalf("expected prompt_id to avoid plan modifiers that reuse prior state, got %d", len(promptIDAttr.PlanModifiers))
	}
}

func TestWorkflowSchema_ValidationPreflightAttributes(t *testing.T) {
	r := NewWorkflowResource().(*WorkflowResource)
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	validateAttr, ok := resp.Schema.Attributes["validate_before_execute"].(resourceschema.BoolAttribute)
	if !ok {
		t.Fatalf("expected validate_before_execute to be a bool attribute, got %#v", resp.Schema.Attributes["validate_before_execute"])
	}
	if !validateAttr.Optional || !validateAttr.Computed || validateAttr.Default == nil {
		t.Fatalf("expected validate_before_execute to be optional, computed, and defaulted, got %#v", validateAttr)
	}

	summaryAttr, ok := resp.Schema.Attributes["validation_summary_json"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatalf("expected validation_summary_json to be a string attribute, got %#v", resp.Schema.Attributes["validation_summary_json"])
	}
	if !summaryAttr.Computed {
		t.Fatalf("expected validation_summary_json to be computed, got %#v", summaryAttr)
	}
	if len(summaryAttr.PlanModifiers) != 1 {
		t.Fatalf("expected validation_summary_json to keep prior state when unknown, got %#v", summaryAttr.PlanModifiers)
	}
}

func newWorkflowTestClient(server *httptest.Server) *client.Client {
	return &client.Client{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
	}
}

func mustEncodeResourceJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("failed to encode JSON response: %v", err)
	}
}

func TestExecuteWorkflow_ValidationFailureBlocksQueueing(t *testing.T) {
	objectInfoHits := 0
	promptHits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/object_info":
			objectInfoHits++
			mustEncodeResourceJSON(t, w, map[string]client.NodeInfo{
				"SaveImage": {
					Input: client.NodeInputInfo{
						Required: map[string]interface{}{
							"images": []interface{}{"IMAGE"},
						},
						Hidden: map[string]interface{}{
							"prompt": "PROMPT",
						},
					},
					InputOrder: map[string][]string{
						"required": {"images"},
					},
					OutputNode: true,
				},
			})
		case "/prompt":
			promptHits++
			mustEncodeResourceJSON(t, w, client.QueueResponse{
				PromptID:   "should-not-run",
				Number:     1,
				NodeErrors: map[string]interface{}{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	r := &WorkflowResource{client: newWorkflowTestClient(server)}
	data := WorkflowModel{
		Execute:               types.BoolValue(true),
		WaitForCompletion:     types.BoolValue(false),
		TimeoutSeconds:        types.Int64Value(30),
		ValidateBeforeExecute: types.BoolValue(true),
	}
	prompt := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "SaveImage",
			"inputs": map[string]interface{}{
				"filename_prefix": "ComfyUI",
			},
		},
	}

	var diags diag.Diagnostics
	r.executeWorkflow(context.Background(), prompt, &data, &diags)
	if !diags.HasError() {
		t.Fatal("expected semantic validation failure to add diagnostics")
	}
	if objectInfoHits != 1 {
		t.Fatalf("expected /object_info to be called once, got %d", objectInfoHits)
	}
	if promptHits != 0 {
		t.Fatalf("expected /prompt not to be called after validation failure, got %d", promptHits)
	}
	if data.ValidationSummaryJSON.IsNull() || data.ValidationSummaryJSON.ValueString() == "" {
		t.Fatal("expected validation_summary_json to be populated")
	}

	var summary map[string]interface{}
	if err := json.Unmarshal([]byte(data.ValidationSummaryJSON.ValueString()), &summary); err != nil {
		t.Fatalf("validation_summary_json should be valid JSON: %v", err)
	}
	if summary["valid"] != false {
		t.Fatalf("expected validation summary to mark prompt invalid, got %#v", summary["valid"])
	}
}

func TestExecuteWorkflow_DisabledValidationSkipsPreflight(t *testing.T) {
	objectInfoHits := 0
	promptHits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/object_info":
			objectInfoHits++
			http.Error(w, "should not validate", http.StatusInternalServerError)
		case "/prompt":
			promptHits++
			mustEncodeResourceJSON(t, w, client.QueueResponse{
				PromptID:   "queued-123",
				Number:     1,
				NodeErrors: map[string]interface{}{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	r := &WorkflowResource{client: newWorkflowTestClient(server)}
	data := WorkflowModel{
		Execute:               types.BoolValue(true),
		WaitForCompletion:     types.BoolValue(false),
		TimeoutSeconds:        types.Int64Value(30),
		ValidateBeforeExecute: types.BoolValue(false),
	}
	prompt := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "SaveImage",
			"inputs": map[string]interface{}{
				"filename_prefix": "ComfyUI",
			},
		},
	}

	var diags diag.Diagnostics
	r.executeWorkflow(context.Background(), prompt, &data, &diags)
	if diags.HasError() {
		t.Fatalf("expected validation to be skipped, got diagnostics %v", diags)
	}
	if objectInfoHits != 0 {
		t.Fatalf("expected /object_info not to be called when validation is disabled, got %d", objectInfoHits)
	}
	if promptHits != 1 {
		t.Fatalf("expected /prompt to be called once, got %d", promptHits)
	}
	if data.PromptID.ValueString() != "queued-123" {
		t.Fatalf("expected queued prompt_id, got %q", data.PromptID.ValueString())
	}
}

func TestExecuteWorkflow_ObjectInfoFailureReturnsDiagnostic(t *testing.T) {
	objectInfoHits := 0
	promptHits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/object_info":
			objectInfoHits++
			http.Error(w, "metadata unavailable", http.StatusInternalServerError)
		case "/prompt":
			promptHits++
			mustEncodeResourceJSON(t, w, client.QueueResponse{
				PromptID:   "should-not-run",
				Number:     1,
				NodeErrors: map[string]interface{}{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	r := &WorkflowResource{client: newWorkflowTestClient(server)}
	data := WorkflowModel{
		Execute:               types.BoolValue(true),
		WaitForCompletion:     types.BoolValue(false),
		TimeoutSeconds:        types.Int64Value(30),
		ValidateBeforeExecute: types.BoolValue(true),
	}
	prompt := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "SaveImage",
			"inputs": map[string]interface{}{
				"filename_prefix": "ComfyUI",
			},
		},
	}

	var diags diag.Diagnostics
	r.executeWorkflow(context.Background(), prompt, &data, &diags)
	if !diags.HasError() {
		t.Fatal("expected object_info fetch failure to add diagnostics")
	}
	if objectInfoHits != 1 {
		t.Fatalf("expected /object_info to be called once, got %d", objectInfoHits)
	}
	if promptHits != 0 {
		t.Fatalf("expected /prompt not to be called after metadata fetch failure, got %d", promptHits)
	}
}

func TestExecuteWorkflow_InvalidExtraDataAddsDiagnostic(t *testing.T) {
	r := &WorkflowResource{client: &client.Client{}}
	data := WorkflowModel{
		Execute:               types.BoolValue(true),
		WaitForCompletion:     types.BoolValue(false),
		TimeoutSeconds:        types.Int64Value(30),
		ValidateBeforeExecute: types.BoolValue(false),
		ExtraDataJSON:         types.StringValue("{not-json}"),
	}

	var diags diag.Diagnostics
	r.executeWorkflow(context.Background(), map[string]interface{}{}, &data, &diags)
	if !diags.HasError() {
		t.Fatal("expected invalid extra_data_json to add diagnostics")
	}
}

func TestExecuteWorkflow_QueueFailureAddsDiagnostic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/prompt" {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "queue unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	r := &WorkflowResource{client: newWorkflowTestClient(server)}
	data := WorkflowModel{
		Execute:               types.BoolValue(true),
		WaitForCompletion:     types.BoolValue(false),
		TimeoutSeconds:        types.Int64Value(30),
		ValidateBeforeExecute: types.BoolValue(false),
	}

	var diags diag.Diagnostics
	r.executeWorkflow(context.Background(), map[string]interface{}{}, &data, &diags)
	if !diags.HasError() {
		t.Fatal("expected queue failure to add diagnostics")
	}
	if data.PromptID.ValueString() != "" {
		t.Fatalf("expected prompt_id to stay empty on queue failure, got %q", data.PromptID.ValueString())
	}
}

func TestExecuteWorkflow_WaitForCompletionEnrichesFromJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/prompt":
			mustEncodeResourceJSON(t, w, client.QueueResponse{
				PromptID:   "prompt-123",
				Number:     1,
				NodeErrors: map[string]interface{}{},
			})
		case "/history/prompt-123":
			mustEncodeResourceJSON(t, w, client.HistoryResponse{
				"prompt-123": {
					Outputs: map[string]interface{}{
						"5": map[string]interface{}{
							"images": []interface{}{
								map[string]interface{}{"filename": "image.png", "type": "output"},
							},
						},
					},
					Status: client.ExecutionStatus{StatusStr: "completed", Completed: true},
				},
			})
		case "/api/jobs/prompt-123":
			mustEncodeResourceJSON(t, w, client.Job{
				ID:                 "prompt-123",
				Status:             "completed",
				CreateTime:         int64Ptr(100),
				ExecutionStartTime: int64Ptr(110),
				ExecutionEndTime:   int64Ptr(120),
				OutputsCount:       intPtr(1),
				WorkflowID:         "workflow-123",
				PreviewOutput: map[string]interface{}{
					"node_id": "5",
				},
				Outputs: map[string]interface{}{
					"5": map[string]interface{}{
						"images": []interface{}{
							map[string]interface{}{"filename": "image.png", "type": "output"},
						},
					},
				},
				ExecutionStatus: map[string]interface{}{
					"status_str": "completed",
					"completed":  true,
				},
				Workflow: &client.JobWorkflow{
					Prompt: map[string]interface{}{
						"5": map[string]interface{}{"class_type": "SaveImage"},
					},
					ExtraData: map[string]interface{}{
						"client_id": "client-1",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	r := &WorkflowResource{client: newWorkflowTestClient(server)}
	data := WorkflowModel{
		Execute:               types.BoolValue(true),
		WaitForCompletion:     types.BoolValue(true),
		TimeoutSeconds:        types.Int64Value(3),
		ValidateBeforeExecute: types.BoolValue(false),
	}
	prompt := map[string]interface{}{
		"5": map[string]interface{}{
			"class_type": "SaveImage",
			"inputs":     map[string]interface{}{},
		},
	}

	var diags diag.Diagnostics
	r.executeWorkflow(context.Background(), prompt, &data, &diags)
	if diags.HasError() {
		t.Fatalf("expected completion path to succeed, got diagnostics %v", diags)
	}
	if data.OutputsJSON.IsNull() || data.OutputsJSON.ValueString() == "" {
		t.Fatal("expected outputs_json to be populated")
	}
	if data.OutputsStructured.IsNull() {
		t.Fatal("expected outputs_structured to be populated")
	}
	if data.ExecutionWorkflowJSON.IsNull() || data.ExecutionWorkflowJSON.ValueString() == "" {
		t.Fatal("expected execution_workflow_json to be populated from /api/jobs")
	}
	if data.ExecutionWorkflow.IsNull() {
		t.Fatal("expected execution_workflow to be populated from /api/jobs")
	}
	if data.WorkflowID.IsNull() || data.WorkflowID.ValueString() != "workflow-123" {
		t.Fatalf("expected workflow_id from /api/jobs enrichment, got %q", data.WorkflowID.ValueString())
	}
}

func TestExecuteWorkflow_WaitFailureAddsDiagnostic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/prompt":
			mustEncodeResourceJSON(t, w, client.QueueResponse{
				PromptID:   "prompt-timeout",
				Number:     1,
				NodeErrors: map[string]interface{}{},
			})
		case "/history/prompt-timeout":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	r := &WorkflowResource{client: newWorkflowTestClient(server)}
	data := WorkflowModel{
		Execute:               types.BoolValue(true),
		WaitForCompletion:     types.BoolValue(true),
		TimeoutSeconds:        types.Int64Value(1),
		ValidateBeforeExecute: types.BoolValue(false),
	}

	var diags diag.Diagnostics
	r.executeWorkflow(context.Background(), map[string]interface{}{}, &data, &diags)
	if !diags.HasError() {
		t.Fatal("expected wait failure to add diagnostics")
	}
}

func TestExecuteWorkflow_WaitForCompletionKeepsHistoryStateWhenJobsLookupFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/prompt":
			mustEncodeResourceJSON(t, w, client.QueueResponse{
				PromptID:   "prompt-456",
				Number:     1,
				NodeErrors: map[string]interface{}{},
			})
		case "/history/prompt-456":
			mustEncodeResourceJSON(t, w, client.HistoryResponse{
				"prompt-456": {
					Outputs: map[string]interface{}{
						"7": map[string]interface{}{
							"images": []interface{}{
								map[string]interface{}{"filename": "fallback.png", "type": "output"},
							},
						},
					},
					Status: client.ExecutionStatus{StatusStr: "completed", Completed: true},
				},
			})
		case "/api/jobs/prompt-456":
			http.Error(w, "jobs unavailable", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	r := &WorkflowResource{client: newWorkflowTestClient(server)}
	data := WorkflowModel{
		Execute:               types.BoolValue(true),
		WaitForCompletion:     types.BoolValue(true),
		TimeoutSeconds:        types.Int64Value(3),
		ValidateBeforeExecute: types.BoolValue(false),
	}
	prompt := map[string]interface{}{
		"7": map[string]interface{}{
			"class_type": "SaveImage",
			"inputs":     map[string]interface{}{},
		},
	}

	var diags diag.Diagnostics
	r.executeWorkflow(context.Background(), prompt, &data, &diags)
	if diags.HasError() {
		t.Fatalf("expected history fallback to stay non-fatal when /api/jobs fails, got diagnostics %v", diags)
	}
	if data.OutputsJSON.IsNull() || data.OutputsJSON.ValueString() == "" {
		t.Fatal("expected outputs_json from history fallback")
	}
	if data.OutputsStructured.IsNull() {
		t.Fatal("expected outputs_structured from history fallback")
	}
	if data.ExecutionStatusJSON.IsNull() || data.ExecutionStatusJSON.ValueString() == "" {
		t.Fatal("expected execution_status_json from history fallback")
	}
	if !data.ExecutionWorkflowJSON.IsNull() {
		t.Fatal("expected execution_workflow_json to remain null when /api/jobs enrichment fails")
	}
}

func TestRefreshWorkflowExecutionState_PreservesHistoryWhenJobIsSparse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/history/prompt-789":
			mustEncodeResourceJSON(t, w, client.HistoryResponse{
				"prompt-789": {
					Prompt: []interface{}{
						float64(7),
						"prompt-789",
						map[string]interface{}{
							"1": map[string]interface{}{"class_type": "KSampler"},
						},
						map[string]interface{}{
							"create_time": float64(1712345600),
							"extra_pnginfo": map[string]interface{}{
								"workflow": map[string]interface{}{"id": "wf-sparse"},
							},
						},
						[]interface{}{"5"},
					},
					Outputs: map[string]interface{}{
						"5": map[string]interface{}{
							"text": []interface{}{"hello world"},
						},
					},
					Status: client.ExecutionStatus{
						StatusStr: "completed",
						Completed: true,
						Messages: [][]interface{}{
							{"execution_start", map[string]interface{}{"timestamp": float64(100)}},
							{"execution_success", map[string]interface{}{"timestamp": float64(105)}},
						},
					},
				},
			})
		case "/api/jobs/prompt-789":
			mustEncodeResourceJSON(t, w, client.Job{
				ID:     "prompt-789",
				Status: "completed",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	r := &WorkflowResource{client: newWorkflowTestClient(server)}
	data := WorkflowModel{
		PromptID: types.StringValue("prompt-789"),
	}

	if err := r.refreshWorkflowExecutionState(context.Background(), &data); err != nil {
		t.Fatalf("refreshWorkflowExecutionState failed: %v", err)
	}

	if data.CreateTime.IsNull() || data.CreateTime.ValueInt64() != 1712345600 {
		t.Fatalf("expected create_time from history to survive sparse job overlay, got %v", data.CreateTime)
	}
	if data.WorkflowID.IsNull() || data.WorkflowID.ValueString() != "wf-sparse" {
		t.Fatalf("expected workflow_id from history to survive sparse job overlay, got %q", data.WorkflowID.ValueString())
	}
	if data.OutputsStructured.IsNull() {
		t.Fatal("expected outputs_structured from history to survive sparse job overlay")
	}
	if data.ExecutionWorkflowJSON.IsNull() || data.ExecutionWorkflowJSON.ValueString() == "" {
		t.Fatal("expected execution_workflow_json from history to survive sparse job overlay")
	}
}

// Chunk 3 Tests: Workflow structured outputs + cancel_on_delete

func TestWorkflowSchema_Chunk3FieldsPresent(t *testing.T) {
	r := NewWorkflowResource().(*WorkflowResource)
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	// Test cancel_on_delete
	cancelAttr, ok := resp.Schema.Attributes["cancel_on_delete"].(resourceschema.BoolAttribute)
	if !ok {
		t.Fatalf("expected cancel_on_delete to be a bool attribute, got %#v", resp.Schema.Attributes["cancel_on_delete"])
	}
	if !cancelAttr.Optional || !cancelAttr.Computed {
		t.Fatalf("expected cancel_on_delete to be optional+computed, got optional=%v computed=%v", cancelAttr.Optional, cancelAttr.Computed)
	}

	legacyFields := []string{"status", "outputs", "error"}
	for _, field := range legacyFields {
		if _, ok := resp.Schema.Attributes[field]; ok {
			t.Fatalf("expected legacy field %s to be absent, got %#v", field, resp.Schema.Attributes[field])
		}
	}

	expectedComputedStrings := []string{
		"workflow_id",
		"preview_output_json",
		"outputs_json",
		"execution_status_json",
		"execution_error_json",
		"execution_workflow_json",
	}
	for _, field := range expectedComputedStrings {
		attr, ok := resp.Schema.Attributes[field].(resourceschema.StringAttribute)
		if !ok {
			t.Errorf("expected %s to be a string attribute, got %#v", field, resp.Schema.Attributes[field])
		}
		if !attr.Computed {
			t.Errorf("expected %s to be computed, got %#v", field, attr)
		}
	}

	expectedComputedInts := []string{
		"create_time",
		"execution_start_time",
		"execution_end_time",
		"outputs_count",
	}
	for _, field := range expectedComputedInts {
		attr, ok := resp.Schema.Attributes[field].(resourceschema.Int64Attribute)
		if !ok {
			t.Errorf("expected %s to be an int64 attribute, got %#v", field, resp.Schema.Attributes[field])
		}
		if !attr.Computed {
			t.Errorf("expected %s to be computed, got %#v", field, attr)
		}
	}

	// Test structured outputs (collision-free name)
	expectedListFields := []string{
		"preview_output",
		"outputs_structured",
		"execution_status",
		"execution_error",
		"execution_workflow",
	}
	for _, field := range expectedListFields {
		attr, ok := resp.Schema.Attributes[field].(resourceschema.DynamicAttribute)
		if !ok {
			t.Errorf("expected %s to be a dynamic attribute, got %#v", field, resp.Schema.Attributes[field])
			continue
		}
		if !attr.Computed {
			t.Errorf("expected %s to be computed, got %#v", field, attr)
		}
	}
}

func TestApplyJobStateToWorkflowModel_PopulatesRichExecutionFields(t *testing.T) {
	job := &client.Job{
		ID:                 "test-prompt-123",
		Status:             "completed",
		CreateTime:         int64Ptr(1704067200),
		ExecutionStartTime: int64Ptr(1704067230),
		ExecutionEndTime:   int64Ptr(1704067260),
		OutputsCount:       intPtr(3),
		WorkflowID:         "wf-456",
		Outputs: map[string]interface{}{
			"5": map[string]interface{}{
				"images": []interface{}{
					map[string]interface{}{
						"filename":  "output_00001.png",
						"subfolder": "",
						"type":      "output",
					},
				},
			},
		},
		ExecutionStatus: map[string]interface{}{
			"status_str": "success",
			"completed":  true,
		},
		ExecutionError: map[string]interface{}{
			"exception_type": "",
		},
	}

	data := WorkflowModel{
		PromptID: types.StringValue("test-prompt-123"),
	}

	err := applyJobStateToWorkflowModel(&data, job)
	if err != nil {
		t.Fatalf("applyJobStateToWorkflowModel failed: %v", err)
	}

	if data.WorkflowID.ValueString() != "wf-456" {
		t.Errorf("expected workflow_id 'wf-456', got %q", data.WorkflowID.ValueString())
	}

	if data.OutputsCount.ValueInt64() != 3 {
		t.Errorf("expected outputs_count 3, got %d", data.OutputsCount.ValueInt64())
	}
	if data.ExecutionStatus.IsNull() {
		t.Error("expected execution_status to be populated")
	}
	if data.OutputsStructured.IsNull() {
		t.Error("expected outputs_structured to be populated")
	}
	if data.OutputsJSON.IsNull() || data.OutputsJSON.ValueString() == "" {
		t.Error("expected outputs_json to be populated")
	}
}

func TestApplyJobStateToWorkflowModel_ErrorCase(t *testing.T) {
	job := &client.Job{
		ID:     "test-prompt-error",
		Status: "error",
		ExecutionError: map[string]interface{}{
			"exception_type":    "ValueError",
			"exception_message": "Invalid input dimension",
			"node_id":           "3",
			"node_type":         "KSampler",
		},
	}

	data := WorkflowModel{
		PromptID: types.StringValue("test-prompt-error"),
	}

	err := applyJobStateToWorkflowModel(&data, job)
	if err != nil {
		t.Fatalf("applyJobStateToWorkflowModel failed: %v", err)
	}

	if data.ExecutionErrorJSON.IsNull() || data.ExecutionErrorJSON.ValueString() == "" {
		t.Error("expected execution_error_json to be populated")
	}
	if data.ExecutionError.IsNull() {
		t.Error("expected execution_error to be populated")
	}
}

func TestApplyJobStateToWorkflowModel_PreservesFailedStatus(t *testing.T) {
	job := &client.Job{
		ID:     "test-prompt-failed",
		Status: "failed",
		ExecutionError: map[string]interface{}{
			"exception_message": "Backend rejected request",
		},
	}

	data := WorkflowModel{
		PromptID: types.StringValue("test-prompt-failed"),
	}

	if err := applyJobStateToWorkflowModel(&data, job); err != nil {
		t.Fatalf("applyJobStateToWorkflowModel failed: %v", err)
	}

	if data.ExecutionErrorJSON.IsNull() || data.ExecutionErrorJSON.ValueString() == "" {
		t.Fatal("expected failed job to preserve execution_error_json")
	}
}

func TestApplyJobStateToWorkflowModel_PreservesRichOutputsWhenJobOutputsMissing(t *testing.T) {
	var jobOutputs map[string]interface{}

	data := WorkflowModel{
		OutputsJSON:       types.StringValue(`{"5":{"images":[{"filename":"history.png"}]}}`),
		OutputsStructured: types.DynamicValue(types.StringValue("keep-outputs")),
	}

	job := &client.Job{
		ID:      "prompt-keep",
		Status:  "completed",
		Outputs: jobOutputs,
	}

	if err := applyJobStateToWorkflowModel(&data, job); err != nil {
		t.Fatalf("applyJobStateToWorkflowModel failed: %v", err)
	}

	if data.OutputsJSON.ValueString() != `{"5":{"images":[{"filename":"history.png"}]}}` {
		t.Fatalf("expected outputs_json to remain preserved, got %q", data.OutputsJSON.ValueString())
	}
	if data.OutputsStructured.IsNull() {
		t.Fatal("expected outputs_structured to remain preserved")
	}
}

func TestApplyJobStateToWorkflowModel_PreservesHistoryMetadataWhenJobFieldsMissing(t *testing.T) {
	data := WorkflowModel{
		CreateTime:            types.Int64Value(10),
		ExecutionStartTime:    types.Int64Value(20),
		ExecutionEndTime:      types.Int64Value(30),
		OutputsCount:          types.Int64Value(2),
		WorkflowID:            types.StringValue("wf-history"),
		ExecutionStatusJSON:   types.StringValue(`{"status_str":"success","messages":[["execution_start",{"timestamp":20}]]}`),
		ExecutionStatus:       types.DynamicValue(types.StringValue("keep-status")),
		ExecutionErrorJSON:    types.StringValue(`{"exception_message":"keep-error"}`),
		ExecutionError:        types.DynamicValue(types.StringValue("keep-error")),
		ExecutionWorkflowJSON: types.StringValue(`{"prompt":{"1":{"class_type":"KSampler"}},"extra_data":{"create_time":10}}`),
		ExecutionWorkflow:     types.DynamicValue(types.StringValue("keep-workflow")),
	}

	job := &client.Job{
		ID:     "prompt-sparse",
		Status: "completed",
	}

	if err := applyJobStateToWorkflowModel(&data, job); err != nil {
		t.Fatalf("applyJobStateToWorkflowModel failed: %v", err)
	}

	if data.CreateTime.ValueInt64() != 10 {
		t.Fatalf("expected create_time to remain preserved, got %d", data.CreateTime.ValueInt64())
	}
	if data.ExecutionStartTime.ValueInt64() != 20 {
		t.Fatalf("expected execution_start_time to remain preserved, got %d", data.ExecutionStartTime.ValueInt64())
	}
	if data.ExecutionEndTime.ValueInt64() != 30 {
		t.Fatalf("expected execution_end_time to remain preserved, got %d", data.ExecutionEndTime.ValueInt64())
	}
	if data.OutputsCount.ValueInt64() != 2 {
		t.Fatalf("expected outputs_count to remain preserved, got %d", data.OutputsCount.ValueInt64())
	}
	if data.WorkflowID.ValueString() != "wf-history" {
		t.Fatalf("expected workflow_id to remain preserved, got %q", data.WorkflowID.ValueString())
	}
	if data.ExecutionStatusJSON.ValueString() == "" {
		t.Fatal("expected execution_status_json to remain preserved")
	}
	if data.ExecutionStatus.IsNull() {
		t.Fatal("expected execution_status to remain preserved")
	}
	if data.ExecutionErrorJSON.ValueString() == "" {
		t.Fatal("expected execution_error_json to remain preserved")
	}
	if data.ExecutionError.IsNull() {
		t.Fatal("expected execution_error to remain preserved")
	}
	if data.ExecutionWorkflowJSON.ValueString() == "" {
		t.Fatal("expected execution_workflow_json to remain preserved")
	}
	if data.ExecutionWorkflow.IsNull() {
		t.Fatal("expected execution_workflow to remain preserved")
	}
}

func TestUpdateFromHistoryEntry_PreservesExistingRichJobFields(t *testing.T) {
	data := WorkflowModel{
		WorkflowID:         types.StringValue("wf-existing"),
		CreateTime:         types.Int64Value(10),
		ExecutionStartTime: types.Int64Value(20),
		ExecutionEndTime:   types.Int64Value(30),
		OutputsCount:       types.Int64Value(4),
		PreviewOutputJSON:  types.StringValue(`{"node_id":"9"}`),
		PreviewOutput:      types.DynamicValue(types.StringValue("keep-preview")),
		ExecutionErrorJSON: types.StringValue(`{"message":"keep"}`),
		ExecutionError:     types.DynamicValue(types.StringValue("keep-error")),
	}

	entry := &client.HistoryEntry{
		Outputs: map[string]interface{}{
			"5": map[string]interface{}{
				"images": []interface{}{
					map[string]interface{}{"filename": "image.png", "type": "output"},
				},
			},
		},
		Status: client.ExecutionStatus{StatusStr: "completed", Completed: true},
	}

	r := &WorkflowResource{}
	r.updateFromHistoryEntry(&data, entry)

	if data.WorkflowID.ValueString() != "wf-existing" {
		t.Fatalf("expected workflow_id to remain unchanged, got %q", data.WorkflowID.ValueString())
	}
	if data.PreviewOutputJSON.ValueString() != `{"node_id":"9"}` {
		t.Fatalf("expected preview_output_json to be preserved, got %q", data.PreviewOutputJSON.ValueString())
	}
	if data.ExecutionErrorJSON.ValueString() != `{"message":"keep"}` {
		t.Fatalf("expected execution_error_json to be preserved, got %q", data.ExecutionErrorJSON.ValueString())
	}
	if data.OutputsJSON.IsNull() || data.OutputsJSON.ValueString() == "" {
		t.Fatal("expected history fallback to refresh outputs_json")
	}
	if data.OutputsStructured.IsNull() {
		t.Fatal("expected history fallback to refresh outputs_structured")
	}
	if data.ExecutionStatusJSON.IsNull() || data.ExecutionStatusJSON.ValueString() == "" {
		t.Fatal("expected history fallback to refresh execution_status_json")
	}
}

func TestUpdateFromHistoryEntry_TypedNilOutputsLeaveRichOutputsUnset(t *testing.T) {
	var outputs map[string]interface{}

	data := WorkflowModel{}
	entry := &client.HistoryEntry{
		Outputs: outputs,
		Status:  client.ExecutionStatus{StatusStr: "completed", Completed: true},
	}

	r := &WorkflowResource{}
	r.updateFromHistoryEntry(&data, entry)

	if !data.OutputsJSON.IsNull() {
		t.Fatalf("expected outputs_json to remain null when history outputs are absent, got %q", data.OutputsJSON.ValueString())
	}
}

func TestUpdateFromHistoryEntry_TypedNilOutputsPreserveExistingRichOutputs(t *testing.T) {
	var outputs map[string]interface{}

	data := WorkflowModel{
		OutputsJSON:       types.StringValue(`{"5":{"text":["hello"]}}`),
		OutputsStructured: types.DynamicValue(types.StringValue("keep-outputs")),
		OutputsCount:      types.Int64Value(1),
	}
	entry := &client.HistoryEntry{
		Outputs: outputs,
		Status:  client.ExecutionStatus{StatusStr: "completed", Completed: true},
	}

	r := &WorkflowResource{}
	r.updateFromHistoryEntry(&data, entry)

	if data.OutputsJSON.ValueString() != `{"5":{"text":["hello"]}}` {
		t.Fatalf("expected outputs_json to remain preserved, got %q", data.OutputsJSON.ValueString())
	}
	if data.OutputsStructured.IsNull() {
		t.Fatal("expected outputs_structured to remain preserved")
	}
	if data.OutputsCount.ValueInt64() != 1 {
		t.Fatalf("expected outputs_count to remain preserved, got %d", data.OutputsCount.ValueInt64())
	}
}

func TestUpdateFromHistoryEntry_EnrichesExecutionMetadataFromHistory(t *testing.T) {
	data := WorkflowModel{}
	entry := &client.HistoryEntry{
		Prompt: []interface{}{
			float64(7),
			"prompt-history-rich",
			map[string]interface{}{
				"1": map[string]interface{}{"class_type": "KSampler"},
			},
			map[string]interface{}{
				"create_time": float64(1712345600),
				"extra_pnginfo": map[string]interface{}{
					"workflow": map[string]interface{}{"id": "wf-history-rich"},
				},
			},
			[]interface{}{"5"},
		},
		Outputs: map[string]interface{}{
			"5": map[string]interface{}{
				"text": []interface{}{"hello world"},
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

	r := &WorkflowResource{}
	r.updateFromHistoryEntry(&data, entry)

	if data.CreateTime.IsNull() || data.CreateTime.ValueInt64() != 1712345600 {
		t.Fatalf("expected create_time from history metadata, got %v", data.CreateTime)
	}
	if data.WorkflowID.IsNull() || data.WorkflowID.ValueString() != "wf-history-rich" {
		t.Fatalf("expected workflow_id from history metadata, got %q", data.WorkflowID.ValueString())
	}
	if data.ExecutionStartTime.IsNull() || data.ExecutionStartTime.ValueInt64() != 100 {
		t.Fatalf("expected execution_start_time 100, got %v", data.ExecutionStartTime)
	}
	if data.ExecutionEndTime.IsNull() || data.ExecutionEndTime.ValueInt64() != 105 {
		t.Fatalf("expected execution_end_time 105, got %v", data.ExecutionEndTime)
	}
	if data.OutputsCount.ValueInt64() != 1 {
		t.Fatalf("expected outputs_count 1, got %d", data.OutputsCount.ValueInt64())
	}
	if data.ExecutionErrorJSON.IsNull() || data.ExecutionErrorJSON.ValueString() == "" {
		t.Fatal("expected execution_error_json to be populated from history messages")
	}
	if data.ExecutionError.IsNull() {
		t.Fatal("expected execution_error to be populated from history messages")
	}
	if data.ExecutionWorkflowJSON.IsNull() || data.ExecutionWorkflowJSON.ValueString() == "" {
		t.Fatal("expected execution_workflow_json to be populated from history prompt tuple")
	}
	if data.ExecutionWorkflow.IsNull() {
		t.Fatal("expected execution_workflow to be populated from history prompt tuple")
	}
	if !strings.Contains(data.ExecutionStatusJSON.ValueString(), "messages") {
		t.Fatalf("expected full execution_status_json to preserve messages, got %q", data.ExecutionStatusJSON.ValueString())
	}
}

func TestUpdateFromHistoryEntry_FailedStatusPreservesExecutionErrorData(t *testing.T) {
	data := WorkflowModel{}
	entry := &client.HistoryEntry{
		Outputs: map[string]interface{}{},
		Status: client.ExecutionStatus{
			StatusStr: "failed",
			Completed: true,
			Messages: [][]interface{}{
				{"execution_error", map[string]interface{}{
					"timestamp":         float64(105),
					"exception_message": "Backend rejected request",
				}},
			},
		},
	}

	r := &WorkflowResource{}
	r.updateFromHistoryEntry(&data, entry)

	if data.ExecutionErrorJSON.IsNull() || !strings.Contains(data.ExecutionErrorJSON.ValueString(), "Backend rejected request") {
		t.Fatalf("expected execution_error_json to be populated, got %q", data.ExecutionErrorJSON.ValueString())
	}
}

func TestUpdateFromHistoryEntry_CancelledStatusLeavesExecutionErrorUnset(t *testing.T) {
	data := WorkflowModel{}
	entry := &client.HistoryEntry{
		Outputs: map[string]interface{}{},
		Status: client.ExecutionStatus{
			StatusStr: "cancelled",
			Completed: true,
		},
	}

	r := &WorkflowResource{}
	r.updateFromHistoryEntry(&data, entry)

	if !data.ExecutionErrorJSON.IsNull() {
		t.Fatalf("expected cancelled history fallback to leave execution_error_json unset, got %q", data.ExecutionErrorJSON.ValueString())
	}
}

func TestDetermineDeleteAction_CancelOnDeleteFalse(t *testing.T) {
	data := WorkflowModel{
		CancelOnDelete: types.BoolValue(false),
		PromptID:       types.StringValue("test-123"),
	}

	action := determineDeleteAction(&data, &client.Job{Status: "running"})
	if action != deleteActionNoop {
		t.Errorf("expected noop when cancel_on_delete=false, got %v", action)
	}
}

func TestDetermineDeleteAction_NoPromptID(t *testing.T) {
	data := WorkflowModel{
		CancelOnDelete: types.BoolValue(true),
		PromptID:       types.StringValue(""),
	}

	action := determineDeleteAction(&data, nil)
	if action != deleteActionNoop {
		t.Errorf("expected noop when prompt_id is empty, got %v", action)
	}
}

func TestDetermineDeleteAction_FallsBackToExecutionStatusWhenJobLookupFails(t *testing.T) {
	queued := WorkflowModel{
		CancelOnDelete:      types.BoolValue(true),
		PromptID:            types.StringValue("queued-123"),
		ExecutionStatusJSON: types.StringValue(`{"status_str":"queued","completed":false}`),
	}
	if action := determineDeleteAction(&queued, nil); action != deleteActionRemoveFromQueue {
		t.Fatalf("expected queued fallback action, got %v", action)
	}

	running := WorkflowModel{
		CancelOnDelete:      types.BoolValue(true),
		PromptID:            types.StringValue("running-123"),
		ExecutionStatusJSON: types.StringValue(`{"status_str":"running","completed":false}`),
	}
	if action := determineDeleteAction(&running, nil); action != deleteActionInterrupt {
		t.Fatalf("expected running fallback action, got %v", action)
	}
}

func TestDetermineDeleteAction_QueuedStatus(t *testing.T) {
	data := WorkflowModel{
		CancelOnDelete: types.BoolValue(true),
		PromptID:       types.StringValue("test-123"),
	}

	action := determineDeleteAction(&data, &client.Job{Status: "pending"})
	if action != deleteActionRemoveFromQueue {
		t.Errorf("expected deleteActionRemoveFromQueue for pending job, got %v", action)
	}

	action = determineDeleteAction(&data, &client.Job{Status: "queued"})
	if action != deleteActionRemoveFromQueue {
		t.Errorf("expected deleteActionRemoveFromQueue for queued job, got %v", action)
	}
}

func TestDetermineDeleteAction_RunningStatus(t *testing.T) {
	data := WorkflowModel{
		CancelOnDelete: types.BoolValue(true),
		PromptID:       types.StringValue("test-123"),
	}

	action := determineDeleteAction(&data, &client.Job{Status: "running"})
	if action != deleteActionInterrupt {
		t.Errorf("expected deleteActionInterrupt for running job, got %v", action)
	}

	action = determineDeleteAction(&data, &client.Job{Status: "executing"})
	if action != deleteActionInterrupt {
		t.Errorf("expected deleteActionInterrupt for executing job, got %v", action)
	}
}

func TestDetermineDeleteAction_CompletedStatus(t *testing.T) {
	data := WorkflowModel{
		CancelOnDelete: types.BoolValue(true),
		PromptID:       types.StringValue("test-123"),
	}

	action := determineDeleteAction(&data, &client.Job{Status: "completed"})
	if action != deleteActionNoop {
		t.Errorf("expected noop for completed job, got %v", action)
	}

	action = determineDeleteAction(&data, &client.Job{Status: "error"})
	if action != deleteActionNoop {
		t.Errorf("expected noop for error job, got %v", action)
	}

	action = determineDeleteAction(&data, &client.Job{Status: "cancelled"})
	if action != deleteActionNoop {
		t.Errorf("expected noop for cancelled job, got %v", action)
	}

	action = determineDeleteAction(&data, &client.Job{Status: "failed"})
	if action != deleteActionNoop {
		t.Errorf("expected noop for failed job, got %v", action)
	}
}

func TestDetermineDeleteAction_JobNotFound(t *testing.T) {
	data := WorkflowModel{
		CancelOnDelete: types.BoolValue(true),
		PromptID:       types.StringValue("test-123"),
	}

	// nil job means not found
	action := determineDeleteAction(&data, nil)
	if action != deleteActionNoop {
		t.Errorf("expected noop when job not found, got %v", action)
	}
}

func TestShouldUseExecutionStatusFallbackOnDelete(t *testing.T) {
	if !shouldUseExecutionStatusFallbackOnDelete(fmt.Errorf("unexpected status 500: boom")) {
		t.Fatal("expected non-404 lookup failures to fall back to stored status")
	}

	if !shouldUseExecutionStatusFallbackOnDelete(fmt.Errorf("unexpected status 404: 404 page not found")) {
		t.Fatal("expected endpoint-missing 404 to fall back to stored status")
	}

	if shouldUseExecutionStatusFallbackOnDelete(fmt.Errorf("unexpected status 404: job not found")) {
		t.Fatal("expected missing-job 404 to remain a no-op")
	}
}

func TestCancelWorkflowOnDelete_ReturnsErrorWhenInterruptFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/jobs/prompt-123":
			mustEncodeResourceJSON(t, w, client.Job{ID: "prompt-123", Status: "running"})
		case "/interrupt":
			http.Error(w, "interrupt failed", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	r := &WorkflowResource{client: newWorkflowTestClient(server)}
	data := WorkflowModel{
		CancelOnDelete: types.BoolValue(true),
		PromptID:       types.StringValue("prompt-123"),
	}

	err := r.cancelWorkflowOnDelete(context.Background(), &data)
	if err == nil {
		t.Fatal("expected interrupt failure to be returned")
	}
	if !strings.Contains(err.Error(), "interrupt workflow") {
		t.Fatalf("expected wrapped interrupt error, got %v", err)
	}
}

func intPtr(v int) *int {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}
