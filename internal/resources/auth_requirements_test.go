package resources

import (
	"testing"
)

func TestExtractAuthRequirements_GeminiNanoBanana2_RequiresComfyOrg(t *testing.T) {
	prompt := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "GeminiNanoBanana2",
			"inputs": map[string]interface{}{
				"prompt": "a beautiful landscape",
			},
		},
	}

	reqs, err := ExtractAuthRequirements(prompt)
	if err != nil {
		t.Fatalf("ExtractAuthRequirements failed: %v", err)
	}

	if len(reqs) != 1 {
		t.Fatalf("expected 1 auth requirement, got %d", len(reqs))
	}

	req := reqs[0]
	if req.Family != "comfy_org" {
		t.Errorf("expected family 'comfy_org', got %q", req.Family)
	}

	expectedFields := []string{"auth_token_comfy_org", "api_key_comfy_org"}
	if len(req.RequiredFields) != len(expectedFields) {
		t.Errorf("expected %d required fields, got %d", len(expectedFields), len(req.RequiredFields))
	}
	for _, field := range expectedFields {
		if !stringInSlice(req.RequiredFields, field) {
			t.Errorf("expected required field %q not found", field)
		}
	}

	if len(req.TriggeringNodes) != 1 {
		t.Errorf("expected 1 triggering node, got %d", len(req.TriggeringNodes))
	}
	if req.TriggeringNodes[0] != "GeminiNanoBanana2" {
		t.Errorf("expected triggering node 'GeminiNanoBanana2', got %q", req.TriggeringNodes[0])
	}
}

func TestExtractAuthRequirements_WanImageToVideoApi_RequiresComfyOrg(t *testing.T) {
	prompt := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "WanImageToVideoApi",
			"inputs": map[string]interface{}{
				"prompt": "a moving landscape",
			},
		},
	}

	reqs, err := ExtractAuthRequirements(prompt)
	if err != nil {
		t.Fatalf("ExtractAuthRequirements failed: %v", err)
	}

	if len(reqs) != 1 {
		t.Fatalf("expected 1 auth requirement, got %d", len(reqs))
	}

	req := reqs[0]
	if req.Family != "comfy_org" {
		t.Errorf("expected family 'comfy_org', got %q", req.Family)
	}

	expectedFields := []string{"auth_token_comfy_org", "api_key_comfy_org"}
	if len(req.RequiredFields) != len(expectedFields) {
		t.Errorf("expected %d required fields, got %d", len(expectedFields), len(req.RequiredFields))
	}
	for _, field := range expectedFields {
		if !stringInSlice(req.RequiredFields, field) {
			t.Errorf("expected required field %q not found", field)
		}
	}

	if len(req.TriggeringNodes) != 1 {
		t.Errorf("expected 1 triggering node, got %d", len(req.TriggeringNodes))
	}
	if req.TriggeringNodes[0] != "WanImageToVideoApi" {
		t.Errorf("expected triggering node 'WanImageToVideoApi', got %q", req.TriggeringNodes[0])
	}
}

func TestExtractAuthRequirements_NonPartnerNode_NoRequirements(t *testing.T) {
	prompt := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "KSampler",
			"inputs": map[string]interface{}{
				"seed": 42,
			},
		},
	}

	reqs, err := ExtractAuthRequirements(prompt)
	if err != nil {
		t.Fatalf("ExtractAuthRequirements failed: %v", err)
	}

	if len(reqs) != 0 {
		t.Errorf("expected 0 auth requirements for non-partner node, got %d", len(reqs))
	}
}

func TestExtractAuthRequirements_MultiplePartnerNodes_DeduplicatesFamily(t *testing.T) {
	prompt := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "GeminiNanoBanana2",
			"inputs": map[string]interface{}{
				"prompt": "a beautiful landscape",
			},
		},
		"2": map[string]interface{}{
			"class_type": "WanImageToVideoApi",
			"inputs": map[string]interface{}{
				"prompt": "a moving landscape",
			},
		},
	}

	reqs, err := ExtractAuthRequirements(prompt)
	if err != nil {
		t.Fatalf("ExtractAuthRequirements failed: %v", err)
	}

	if len(reqs) != 1 {
		t.Fatalf("expected 1 deduplicated auth requirement, got %d", len(reqs))
	}

	req := reqs[0]
	if req.Family != "comfy_org" {
		t.Errorf("expected family 'comfy_org', got %q", req.Family)
	}

	if len(req.TriggeringNodes) != 2 {
		t.Errorf("expected 2 triggering nodes, got %d", len(req.TriggeringNodes))
	}

	expectedNodes := []string{"GeminiNanoBanana2", "WanImageToVideoApi"}
	for _, node := range expectedNodes {
		if !stringInSlice(req.TriggeringNodes, node) {
			t.Errorf("expected triggering node %q not found", node)
		}
	}
}

func TestExtractAuthRequirements_RepeatedPartnerNode_DeduplicatesTriggeringNodeClass(t *testing.T) {
	prompt := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "GeminiNanoBanana2",
			"inputs": map[string]interface{}{
				"prompt": "first prompt",
			},
		},
		"2": map[string]interface{}{
			"class_type": "GeminiNanoBanana2",
			"inputs": map[string]interface{}{
				"prompt": "second prompt",
			},
		},
	}

	reqs, err := ExtractAuthRequirements(prompt)
	if err != nil {
		t.Fatalf("ExtractAuthRequirements failed: %v", err)
	}

	if len(reqs) != 1 {
		t.Fatalf("expected 1 auth requirement, got %d", len(reqs))
	}

	req := reqs[0]
	if req.Family != "comfy_org" {
		t.Errorf("expected family 'comfy_org', got %q", req.Family)
	}

	if len(req.TriggeringNodes) != 1 {
		t.Fatalf("expected repeated node class to be deduplicated, got %v", req.TriggeringNodes)
	}

	if req.TriggeringNodes[0] != "GeminiNanoBanana2" {
		t.Errorf("expected triggering node 'GeminiNanoBanana2', got %q", req.TriggeringNodes[0])
	}
}

func TestExtractAuthRequirements_MixedNodes_OnlyReturnsPartnerRequirements(t *testing.T) {
	prompt := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "KSampler",
			"inputs":     map[string]interface{}{},
		},
		"2": map[string]interface{}{
			"class_type": "GeminiNanoBanana2",
			"inputs": map[string]interface{}{
				"prompt": "test",
			},
		},
		"3": map[string]interface{}{
			"class_type": "SaveImage",
			"inputs":     map[string]interface{}{},
		},
	}

	reqs, err := ExtractAuthRequirements(prompt)
	if err != nil {
		t.Fatalf("ExtractAuthRequirements failed: %v", err)
	}

	if len(reqs) != 1 {
		t.Fatalf("expected 1 auth requirement, got %d", len(reqs))
	}

	req := reqs[0]
	if req.Family != "comfy_org" {
		t.Errorf("expected family 'comfy_org', got %q", req.Family)
	}

	if len(req.TriggeringNodes) != 1 {
		t.Errorf("expected 1 triggering node, got %d", len(req.TriggeringNodes))
	}
}

func TestExtractAuthRequirements_EmptyPrompt_NoRequirements(t *testing.T) {
	prompt := map[string]interface{}{}

	reqs, err := ExtractAuthRequirements(prompt)
	if err != nil {
		t.Fatalf("ExtractAuthRequirements failed: %v", err)
	}

	if len(reqs) != 0 {
		t.Errorf("expected 0 auth requirements for empty prompt, got %d", len(reqs))
	}
}

func TestExtractAuthRequirements_MalformedNode_SkipsGracefully(t *testing.T) {
	prompt := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "GeminiNanoBanana2",
			"inputs":     map[string]interface{}{},
		},
		"2": "not a valid node",
		"3": map[string]interface{}{
			// Missing class_type
			"inputs": map[string]interface{}{},
		},
	}

	reqs, err := ExtractAuthRequirements(prompt)
	if err != nil {
		t.Fatalf("ExtractAuthRequirements failed: %v", err)
	}

	if len(reqs) != 1 {
		t.Errorf("expected 1 auth requirement (malformed nodes should be skipped), got %d", len(reqs))
	}

	if reqs[0].Family != "comfy_org" {
		t.Errorf("expected family 'comfy_org', got %q", reqs[0].Family)
	}
}
