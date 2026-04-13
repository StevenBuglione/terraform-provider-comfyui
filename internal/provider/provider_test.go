package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func TestProviderRegistersSubgraphSurface(t *testing.T) {
	p := &ComfyUIProvider{version: "test"}

	resourceFactories := p.Resources(context.Background())
	resourceTypes := make([]string, 0, len(resourceFactories))
	for _, factory := range resourceFactories {
		r := factory()
		var resp resource.MetadataResponse
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "comfyui"}, &resp)
		resourceTypes = append(resourceTypes, resp.TypeName)
	}

	dataSourceFactories := p.DataSources(context.Background())
	dataSourceTypes := make([]string, 0, len(dataSourceFactories))
	for _, factory := range dataSourceFactories {
		ds := factory()
		var resp datasource.MetadataResponse
		ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "comfyui"}, &resp)
		dataSourceTypes = append(dataSourceTypes, resp.TypeName)
	}

	requiredResources := map[string]bool{
		"comfyui_subgraph": false,
	}
	for _, typeName := range resourceTypes {
		if _, ok := requiredResources[typeName]; ok {
			requiredResources[typeName] = true
		}
	}
	for typeName, found := range requiredResources {
		if !found {
			t.Fatalf("expected provider to register %s, got %v", typeName, resourceTypes)
		}
	}

	requiredDataSources := map[string]bool{
		"comfyui_subgraph_catalog":    false,
		"comfyui_subgraph_definition": false,
	}
	for _, typeName := range dataSourceTypes {
		if _, ok := requiredDataSources[typeName]; ok {
			requiredDataSources[typeName] = true
		}
	}
	for typeName, found := range requiredDataSources {
		if !found {
			t.Fatalf("expected provider to register %s, got %v", typeName, dataSourceTypes)
		}
	}
}

func TestResolveProviderSettings_UsesConfigAndParsesExtraData(t *testing.T) {
	settings, err := resolveProviderSettings(ComfyUIProviderModel{
		Host:                         types.StringValue("http://127.0.0.1:8188"),
		Port:                         types.Int64Value(8188),
		APIKey:                       types.StringValue("api-key"),
		ComfyOrgAuthToken:            types.StringValue("auth-token"),
		ComfyOrgAPIKey:               types.StringValue("partner-key"),
		DefaultWorkflowExtraDataJSON: types.StringValue(`{"tenant":"dev","extra_pnginfo":{"workflow":{"id":"wf-1"}}}`),
	}, func(string) string {
		return ""
	})
	if err != nil {
		t.Fatalf("resolveProviderSettings returned error: %v", err)
	}

	if settings.ComfyOrgAuthToken != "auth-token" || settings.ComfyOrgAPIKey != "partner-key" {
		t.Fatalf("unexpected comfy_org credentials: %#v", settings)
	}

	expected := map[string]interface{}{
		"tenant": "dev",
		"extra_pnginfo": map[string]interface{}{
			"workflow": map[string]interface{}{
				"id": "wf-1",
			},
		},
	}
	if !reflect.DeepEqual(settings.DefaultWorkflowExtraData, expected) {
		t.Fatalf("unexpected default workflow extra data: %#v", settings.DefaultWorkflowExtraData)
	}
}

func TestResolveProviderSettings_UsesEnvFallbacksAndValidatesMode(t *testing.T) {
	getenv := func(key string) string {
		switch key {
		case "COMFYUI_COMFY_ORG_AUTH_TOKEN":
			return "env-auth-token"
		case "COMFYUI_COMFY_ORG_API_KEY":
			return "env-partner-key"
		case "COMFYUI_UNSUPPORTED_DYNAMIC_VALIDATION_MODE":
			return "warning"
		default:
			return ""
		}
	}

	settings, err := resolveProviderSettings(ComfyUIProviderModel{}, getenv)
	if err != nil {
		t.Fatalf("resolveProviderSettings returned error: %v", err)
	}
	if settings.ComfyOrgAuthToken != "env-auth-token" || settings.ComfyOrgAPIKey != "env-partner-key" {
		t.Fatalf("unexpected env credentials: %#v", settings)
	}
	if settings.UnsupportedDynamicValidationMode != "warning" {
		t.Fatalf("expected warning mode, got %q", settings.UnsupportedDynamicValidationMode)
	}
}

func TestResolveProviderSettings_RejectsInvalidValidationMode(t *testing.T) {
	_, err := resolveProviderSettings(ComfyUIProviderModel{
		UnsupportedDynamicValidationMode: types.StringValue("explode"),
	}, func(string) string {
		return ""
	})
	if err == nil {
		t.Fatal("expected invalid validation mode to fail")
	}
}
