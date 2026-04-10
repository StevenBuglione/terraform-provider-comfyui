package datasources

import (
	"testing"

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
