package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
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

func TestProviderRegistersValidationDataSources(t *testing.T) {
	p := &ComfyUIProvider{version: "test"}

	dataSourceFactories := p.DataSources(context.Background())
	typeNames := make([]string, 0, len(dataSourceFactories))
	for _, factory := range dataSourceFactories {
		ds := factory()
		var resp datasource.MetadataResponse
		ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "comfyui"}, &resp)
		typeNames = append(typeNames, resp.TypeName)
	}

	hasPromptValidation := false
	hasWorkspaceValidation := false
	for _, typeName := range typeNames {
		if typeName == "comfyui_prompt_validation" {
			hasPromptValidation = true
		}
		if typeName == "comfyui_workspace_validation" {
			hasWorkspaceValidation = true
		}
	}

	if !hasPromptValidation || !hasWorkspaceValidation {
		t.Fatalf("expected provider to register validation data sources, got %v", typeNames)
	}
}

func TestProviderRegistersFileLifecycleResources(t *testing.T) {
	p := &ComfyUIProvider{}

	resourceFactories := p.Resources(context.Background())
	typeNames := make([]string, 0, len(resourceFactories))
	for _, factory := range resourceFactories {
		r := factory()
		var resp resource.MetadataResponse
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "comfyui"}, &resp)
		typeNames = append(typeNames, resp.TypeName)
	}

	required := map[string]bool{
		"comfyui_uploaded_image":  false,
		"comfyui_uploaded_mask":   false,
		"comfyui_output_artifact": false,
	}
	for _, typeName := range typeNames {
		if _, ok := required[typeName]; ok {
			required[typeName] = true
		}
	}
	for typeName, found := range required {
		if !found {
			t.Fatalf("expected provider to register %s, got %v", typeName, typeNames)
		}
	}
}
