package resources_test

import (
	"context"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/resources"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/resources/generated"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var wan2TextToVideoModelAttributeTypes = map[string]attr.Type{
	"selection":       types.StringType,
	"prompt":          types.StringType,
	"negative_prompt": types.StringType,
	"resolution":      types.StringType,
	"ratio":           types.StringType,
	"duration":        types.Int64Type,
}

func TestWan2TextToVideoAPIResourceSchema_ModelUsesNestedDynamicCombo(t *testing.T) {
	r := generated.NewWan2TextToVideoAPIResource()

	var resp frameworkresource.SchemaResponse
	r.Schema(context.Background(), frameworkresource.SchemaRequest{}, &resp)

	modelAttr, ok := resp.Schema.Attributes["model"].(resourceschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected model to be a SingleNestedAttribute, got %T", resp.Schema.Attributes["model"])
	}
	if !modelAttr.Required {
		t.Fatalf("expected model to be required")
	}

	selectionAttr, ok := modelAttr.Attributes["selection"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatalf("expected model.selection to be a string attribute, got %T", modelAttr.Attributes["selection"])
	}
	if !selectionAttr.Required {
		t.Fatalf("expected model.selection to be required")
	}

	for _, attrName := range []string{"prompt", "negative_prompt", "resolution", "ratio", "duration"} {
		if _, ok := modelAttr.Attributes[attrName]; !ok {
			t.Fatalf("expected model nested schema to include %q", attrName)
		}
	}
}

func TestWan2TextToVideoAPIValidation_RequiresSelectedModelChildren(t *testing.T) {
	model := generated.Wan2TextToVideoAPIModel{
		Model: wan2TextToVideoModelObject(map[string]attr.Value{
			"selection": types.StringValue("wan2.7-t2v"),
		}),
		Seed:         types.Int64Value(42),
		PromptExtend: types.BoolValue(true),
		Watermark:    types.BoolValue(false),
	}

	diags := resources.ValidateDynamicInputsForTest(context.Background(), &client.Client{}, "Wan2TextToVideoApi", model)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for missing WAN2 DynamicCombo child fields")
	}

	assertDiagnosticPathPresent(t, diags, path.Root("model").AtName("prompt"))
	assertDiagnosticPathPresent(t, diags, path.Root("model").AtName("negative_prompt"))
	assertDiagnosticPathPresent(t, diags, path.Root("model").AtName("resolution"))
	assertDiagnosticPathPresent(t, diags, path.Root("model").AtName("ratio"))
	assertDiagnosticPathPresent(t, diags, path.Root("model").AtName("duration"))
}

func TestWan2TextToVideoAPIValidation_AcceptsCompleteModelSelection(t *testing.T) {
	model := generated.Wan2TextToVideoAPIModel{
		Model: wan2TextToVideoModelObject(map[string]attr.Value{
			"selection":       types.StringValue("wan2.7-t2v"),
			"prompt":          types.StringValue("cinematic storm over the ocean"),
			"negative_prompt": types.StringValue(""),
			"resolution":      types.StringValue("720P"),
			"ratio":           types.StringValue("16:9"),
			"duration":        types.Int64Value(5),
		}),
		Seed:         types.Int64Value(42),
		PromptExtend: types.BoolValue(true),
		Watermark:    types.BoolValue(false),
	}

	diags := resources.ValidateDynamicInputsForTest(context.Background(), &client.Client{}, "Wan2TextToVideoApi", model)
	if diags.HasError() || diags.WarningsCount() != 0 {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
}

func wan2TextToVideoModelObject(overrides map[string]attr.Value) types.Object {
	attributes := map[string]attr.Value{
		"selection":       types.StringNull(),
		"prompt":          types.StringNull(),
		"negative_prompt": types.StringNull(),
		"resolution":      types.StringNull(),
		"ratio":           types.StringNull(),
		"duration":        types.Int64Null(),
	}
	for key, value := range overrides {
		attributes[key] = value
	}
	return types.ObjectValueMust(wan2TextToVideoModelAttributeTypes, attributes)
}

func assertDiagnosticPathPresent(t *testing.T, diags diag.Diagnostics, want path.Path) {
	t.Helper()

	for _, diagnostic := range diags {
		withPath, ok := diagnostic.(diag.DiagnosticWithPath)
		if !ok {
			continue
		}
		if withPath.Path().Equal(want) {
			return
		}
	}

	t.Fatalf("expected diagnostic for path %s, got %v", want, diags)
}
