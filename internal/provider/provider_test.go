package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestProviderRegistersWorkspaceResource(t *testing.T) {
	p := &ComfyUIProvider{}

	resourceFactories := p.Resources(context.Background())
	typeNames := make([]string, 0, len(resourceFactories))
	for _, factory := range resourceFactories {
		r := factory()
		var resp resource.MetadataResponse
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "comfyui"}, &resp)
		typeNames = append(typeNames, resp.TypeName)
	}

	for _, typeName := range typeNames {
		if typeName == "comfyui_workspace" {
			return
		}
	}

	t.Fatalf("expected provider to register comfyui_workspace, got %v", typeNames)
}
