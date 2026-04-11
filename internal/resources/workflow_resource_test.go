package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	if data.Status.ValueString() != "queued" {
		t.Fatalf("expected queued status, got %q", data.Status.ValueString())
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
