package resources

import "testing"

func TestResolveAuthRequirements_ComfyOrgPrefersAPIKey(t *testing.T) {
	requirements := []AuthRequirement{
		{
			Family:          "comfy_org",
			RequiredFields:  []string{"auth_token_comfy_org", "api_key_comfy_org"},
			TriggeringNodes: []string{"GeminiNanoBanana2"},
		},
	}

	resolution := ResolveAuthRequirements(requirements, AuthResolverConfig{
		ComfyOrgAPIKey:    "partner-api-key",
		ComfyOrgAuthToken: "frontend-token",
	})

	if len(resolution.Unsatisfied) != 0 {
		t.Fatalf("expected no unsatisfied requirements, got %#v", resolution.Unsatisfied)
	}

	if got := resolution.ExtraData["api_key_comfy_org"]; got != "partner-api-key" {
		t.Fatalf("expected api_key_comfy_org to be resolved, got %#v", got)
	}

	if _, ok := resolution.ExtraData["auth_token_comfy_org"]; ok {
		t.Fatalf("expected auth token to be omitted when API key is available, got %#v", resolution.ExtraData)
	}

	if len(resolution.Resolved) != 1 {
		t.Fatalf("expected one resolved requirement, got %d", len(resolution.Resolved))
	}

	if resolution.Resolved[0].Mode != "api_key" {
		t.Fatalf("expected api_key mode, got %q", resolution.Resolved[0].Mode)
	}
}

func TestResolveAuthRequirements_ComfyOrgFallsBackToAuthToken(t *testing.T) {
	requirements := []AuthRequirement{
		{
			Family:          "comfy_org",
			RequiredFields:  []string{"auth_token_comfy_org", "api_key_comfy_org"},
			TriggeringNodes: []string{"WanImageToVideoApi"},
		},
	}

	resolution := ResolveAuthRequirements(requirements, AuthResolverConfig{
		ComfyOrgAuthToken: "frontend-token",
	})

	if len(resolution.Unsatisfied) != 0 {
		t.Fatalf("expected no unsatisfied requirements, got %#v", resolution.Unsatisfied)
	}

	if got := resolution.ExtraData["auth_token_comfy_org"]; got != "frontend-token" {
		t.Fatalf("expected auth_token_comfy_org to be resolved, got %#v", got)
	}

	if _, ok := resolution.ExtraData["api_key_comfy_org"]; ok {
		t.Fatalf("expected api key to be omitted when only token is available, got %#v", resolution.ExtraData)
	}

	if resolution.Resolved[0].Mode != "auth_token" {
		t.Fatalf("expected auth_token mode, got %q", resolution.Resolved[0].Mode)
	}
}

func TestResolveAuthRequirements_MissingComfyOrgLeavesRequirementUnsatisfied(t *testing.T) {
	requirements := []AuthRequirement{
		{
			Family:          "comfy_org",
			RequiredFields:  []string{"auth_token_comfy_org", "api_key_comfy_org"},
			TriggeringNodes: []string{"GeminiNanoBanana2"},
		},
	}

	resolution := ResolveAuthRequirements(requirements, AuthResolverConfig{})

	if len(resolution.Resolved) != 0 {
		t.Fatalf("expected no resolved requirements, got %#v", resolution.Resolved)
	}

	if len(resolution.Unsatisfied) != 1 {
		t.Fatalf("expected one unsatisfied requirement, got %#v", resolution.Unsatisfied)
	}

	if resolution.Unsatisfied[0].Family != "comfy_org" {
		t.Fatalf("expected comfy_org to remain unsatisfied, got %#v", resolution.Unsatisfied[0])
	}

	if len(resolution.ExtraData) != 0 {
		t.Fatalf("expected no auth payload when resolution fails, got %#v", resolution.ExtraData)
	}
}
