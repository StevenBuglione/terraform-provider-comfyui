package datasources

import (
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/resources/generated"
)

func TestProviderInfoDataSource_VersionConstants(t *testing.T) {
	if generated.ComfyUIVersion == "" {
		t.Fatal("generated.ComfyUIVersion must not be empty")
	}

	if generated.NodeCount <= 0 {
		t.Fatalf("generated.NodeCount must be positive, got %d", generated.NodeCount)
	}

	if generated.ExtractedAt == "" {
		t.Fatal("generated.ExtractedAt must not be empty")
	}

	t.Logf("ComfyUI Version: %s", generated.ComfyUIVersion)
	t.Logf("Node Count: %d", generated.NodeCount)
	t.Logf("Extracted At: %s", generated.ExtractedAt)
}

func TestProviderInfoDataSource_Factory(t *testing.T) {
	factory := NewProviderInfoDataSource("1.0.0-test")
	ds := factory()
	if ds == nil {
		t.Fatal("factory returned nil data source")
	}

	info, ok := ds.(*ProviderInfoDataSource)
	if !ok {
		t.Fatal("factory returned wrong type")
	}

	if info.providerVersion != "1.0.0-test" {
		t.Errorf("expected provider version '1.0.0-test', got %q", info.providerVersion)
	}
}

func TestAuthPosture_WithComfyOrgAndServiceAPIKey(t *testing.T) {
	posture := authPosture(&client.Client{
		APIKey:            "service-key",
		ComfyOrgAPIKey:    "partner-key",
		ComfyOrgAuthToken: "frontend-token",
	})

	if !posture.ServiceAPIKeyConfigured {
		t.Fatal("expected service API key to be reported as configured")
	}
	if !posture.PartnerAuthConfigured {
		t.Fatal("expected partner auth to be reported as configured")
	}
	if len(posture.ConfiguredAuthFamilies) != 1 || posture.ConfiguredAuthFamilies[0] != "comfy_org" {
		t.Fatalf("expected configured comfy_org family, got %#v", posture.ConfiguredAuthFamilies)
	}
}

func TestAuthPosture_WithoutPartnerAuth(t *testing.T) {
	posture := authPosture(&client.Client{
		APIKey: "service-key",
	})

	if !posture.ServiceAPIKeyConfigured {
		t.Fatal("expected service API key to be reported as configured")
	}
	if posture.PartnerAuthConfigured {
		t.Fatal("expected partner auth to be reported as not configured")
	}
	if len(posture.ConfiguredAuthFamilies) != 0 {
		t.Fatalf("expected no configured auth families, got %#v", posture.ConfiguredAuthFamilies)
	}
}
