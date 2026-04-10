package resources

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
