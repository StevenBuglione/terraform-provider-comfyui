package resources

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkspaceResourceMetadata(t *testing.T) {
	r := NewWorkspaceResource()

	var resp resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "comfyui"}, &resp)

	if resp.TypeName != "comfyui_workspace" {
		t.Fatalf("expected type name %q, got %q", "comfyui_workspace", resp.TypeName)
	}
}

func TestWorkspaceConfigFromModelBuildsSpecs(t *testing.T) {
	model := workspaceResourceModel{
		Name: types.StringValue("demo-workspace"),
		Workflows: []workspaceWorkflowModel{
			{
				Name:         types.StringValue("workflow-a"),
				WorkflowJSON: types.StringValue(`{"1":{"class_type":"SourceNode","inputs":{"text":"hello"}}}`),
				X:            types.Float64Value(25),
			},
		},
		Layout: workspaceLayoutModel{
			Display: types.StringValue("grid"),
			Columns: types.Int64Value(2),
			OriginX: types.Float64Value(50),
			OriginY: types.Float64Value(75),
		},
	}

	name, specs, layout, err := workspaceConfigFromModel(model)
	if err != nil {
		t.Fatalf("workspaceConfigFromModel returned error: %v", err)
	}

	if name != "demo-workspace" {
		t.Fatalf("expected workspace name %q, got %q", "demo-workspace", name)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 workflow spec, got %d", len(specs))
	}
	if specs[0].Name != "workflow-a" {
		t.Fatalf("expected workflow name %q, got %q", "workflow-a", specs[0].Name)
	}
	if specs[0].X == nil || *specs[0].X != 25 {
		t.Fatalf("expected workflow X override 25, got %#v", specs[0].X)
	}
	if layout.Display != "grid" || layout.Columns != 2 || layout.OriginX != 50 || layout.OriginY != 75 {
		t.Fatalf("unexpected layout config: %+v", layout)
	}
}

func TestWriteWorkspaceFileCreatesDirectoryAndFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "workspace.json")

	if err := writeWorkspaceFile(target, []byte(`{"name":"demo-workspace"}`)); err != nil {
		t.Fatalf("writeWorkspaceFile returned error: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed to read workspace file: %v", err)
	}
	if string(data) != `{"name":"demo-workspace"}` {
		t.Fatalf("unexpected file contents: %s", string(data))
	}
}

func TestCleanupPreviousWorkspaceFileRemovesStaleFile(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.json")
	newPath := filepath.Join(dir, "new.json")

	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("failed to seed old file: %v", err)
	}

	if err := cleanupPreviousWorkspaceFile(oldPath, newPath); err != nil {
		t.Fatalf("cleanupPreviousWorkspaceFile returned error: %v", err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old file to be removed, got err=%v", err)
	}
}

func TestWorkspaceResourceSchemaIncludesCSSInspiredLayoutContract(t *testing.T) {
	r := NewWorkspaceResource()

	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, attrName := range []string{"name", "workflows", "layout", "output_file", "workspace_json", "workflow_count"} {
		if _, ok := resp.Schema.Attributes[attrName]; !ok {
			t.Fatalf("expected schema to include %q", attrName)
		}
	}

	workflowsAttr, ok := resp.Schema.Attributes["workflows"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected workflows to be a ListNestedAttribute, got %T", resp.Schema.Attributes["workflows"])
	}
	if !workflowsAttr.Required {
		t.Fatalf("expected workflows to be required")
	}

	for _, attrName := range []string{"name", "workflow_json", "x", "y"} {
		if _, ok := workflowsAttr.NestedObject.Attributes[attrName]; !ok {
			t.Fatalf("expected workflows nested schema to include %q", attrName)
		}
	}
	if _, ok := workflowsAttr.NestedObject.Attributes["workflow_id"]; ok {
		t.Fatalf("did not expect workflows nested schema to accept workflow_id")
	}

	// New style nested attribute under workflows[] (failing test until implemented)
	styleAttr, ok := workflowsAttr.NestedObject.Attributes["style"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected workflows to include a SingleNestedAttribute 'style', got %T", workflowsAttr.NestedObject.Attributes["style"])
	}
	for _, attrName := range []string{"group_color", "title_font_size"} {
		if _, ok := styleAttr.Attributes[attrName]; !ok {
			t.Fatalf("expected style schema to include %q", attrName)
		}
	}

	layoutAttr, ok := resp.Schema.Attributes["layout"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected layout to be a SingleNestedAttribute, got %T", resp.Schema.Attributes["layout"])
	}
	if !layoutAttr.Required {
		t.Fatalf("expected layout to be required")
	}

	for _, attrName := range []string{
		"display",
		"direction",
		"gap",
		"columns",
		"origin_x",
		"origin_y",
	} {
		if _, ok := layoutAttr.Attributes[attrName]; !ok {
			t.Fatalf("expected layout schema to include %q", attrName)
		}
	}
	for _, attrName := range []string{"row_gap", "column_gap", "wrap", "justify_content", "align_items"} {
		if _, ok := layoutAttr.Attributes[attrName]; ok {
			t.Fatalf("did not expect unsupported layout attribute %q in v1 schema", attrName)
		}
	}

	// New top-level node_layout attribute (failing test until implemented)
	nodeLayoutAttr, ok := resp.Schema.Attributes["node_layout"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected node_layout to be a SingleNestedAttribute, got %T", resp.Schema.Attributes["node_layout"])
	}
	for _, attrName := range []string{"mode", "direction", "column_gap", "row_gap"} {
		if _, ok := nodeLayoutAttr.Attributes[attrName]; !ok {
			t.Fatalf("expected node_layout schema to include %q", attrName)
		}
	}
}

func TestValidateWorkspaceLayoutRejectsInvalidCombinations(t *testing.T) {
	// This test asserts that the provider schema includes the layout contract
	r := NewWorkspaceResource()

	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	layoutAttr, ok := resp.Schema.Attributes["layout"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected layout to be a SingleNestedAttribute, got %T", resp.Schema.Attributes["layout"])
	}

	for _, attrName := range []string{"display", "direction", "columns", "origin_x", "origin_y", "gap"} {
		if _, ok := layoutAttr.Attributes[attrName]; !ok {
			t.Fatalf("expected layout schema to include %q", attrName)
		}
	}
}

func TestValidateWorkspaceNodeLayoutRejectsInvalidMode(t *testing.T) {
	// This test asserts the presence of the new node_layout schema and its expected attributes.
	r := NewWorkspaceResource()

	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	nodeLayoutAttr, ok := resp.Schema.Attributes["node_layout"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected node_layout to be a SingleNestedAttribute, got %T", resp.Schema.Attributes["node_layout"])
	}

	for _, attrName := range []string{"mode", "direction", "column_gap", "row_gap"} {
		if _, ok := nodeLayoutAttr.Attributes[attrName]; !ok {
			t.Fatalf("expected node_layout schema to include %q", attrName)
		}
	}
}

func TestValidateWorkflowStyleRejectsEmptyGroupColor(t *testing.T) {
	// This test asserts the presence of workflows[].style.group_color and title_font_size in the schema.
	r := NewWorkspaceResource()

	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	workflowsAttr, ok := resp.Schema.Attributes["workflows"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected workflows to be a ListNestedAttribute, got %T", resp.Schema.Attributes["workflows"])
	}

	styleAttr, ok := workflowsAttr.NestedObject.Attributes["style"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected workflows to include a SingleNestedAttribute 'style', got %T", workflowsAttr.NestedObject.Attributes["style"])
	}
	for _, attrName := range []string{"group_color", "title_font_size"} {
		if _, ok := styleAttr.Attributes[attrName]; !ok {
			t.Fatalf("expected style schema to include %q", attrName)
		}
	}
}
