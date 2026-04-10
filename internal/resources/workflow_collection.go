package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

var (
	_ resource.Resource              = &WorkflowCollectionResource{}
	_ resource.ResourceWithConfigure = &WorkflowCollectionResource{}
)

type WorkflowCollectionResource struct {
	client *client.Client
}

type WorkflowCollectionModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	OutputDir     types.String `tfsdk:"output_dir"`
	Workflows     types.List   `tfsdk:"workflows"`
	IndexJSON     types.String `tfsdk:"index_json"`
	WorkflowCount types.Int64  `tfsdk:"workflow_count"`
}

type indexManifest struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	WorkflowCount int             `json:"workflow_count"`
	Workflows     []indexWorkflow `json:"workflows"`
}

type indexWorkflow struct {
	ID    string `json:"id"`
	Index int    `json:"index"`
}

func NewWorkflowCollectionResource() resource.Resource {
	return &WorkflowCollectionResource{}
}

func (r *WorkflowCollectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow_collection"
}

func (r *WorkflowCollectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Organizes and labels multiple comfyui_workflow resources into a collection with metadata and an optional file-based index manifest.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for this workflow collection.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the workflow collection",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of this collection",
			},
			"output_dir": schema.StringAttribute{
				Optional:    true,
				Description: "Directory path to write workflow files and index manifest",
			},
			"workflows": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "List of comfyui_workflow resource IDs to include in this collection",
			},
			"index_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON manifest indexing all workflows in this collection",
			},
			"workflow_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of workflows in this collection",
			},
		},
	}
}

func (r *WorkflowCollectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *WorkflowCollectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkflowCollectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(uuid.New().String())

	workflowIDs, diags := r.extractWorkflowIDs(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexJSON := buildIndexJSON(data.Name.ValueString(), descriptionValue(data.Description), workflowIDs)
	data.IndexJSON = types.StringValue(indexJSON)
	data.WorkflowCount = types.Int64Value(int64(len(workflowIDs)))

	if !data.OutputDir.IsNull() && !data.OutputDir.IsUnknown() {
		if err := writeIndexFile(data.OutputDir.ValueString(), indexJSON); err != nil {
			resp.Diagnostics.AddError("Failed to write index manifest", err.Error())
			return
		}
		tflog.Info(ctx, "Wrote index.json to output directory", map[string]interface{}{
			"output_dir": data.OutputDir.ValueString(),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowCollectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkflowCollectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If output_dir is set, verify it still exists
	if !data.OutputDir.IsNull() && !data.OutputDir.IsUnknown() {
		dir := data.OutputDir.ValueString()
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			tflog.Warn(ctx, "Output directory no longer exists", map[string]interface{}{
				"output_dir": dir,
			})
		}
	}

	// Refresh the index from current state
	workflowIDs, diags := r.extractWorkflowIDs(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexJSON := buildIndexJSON(data.Name.ValueString(), descriptionValue(data.Description), workflowIDs)
	data.IndexJSON = types.StringValue(indexJSON)
	data.WorkflowCount = types.Int64Value(int64(len(workflowIDs)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowCollectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WorkflowCollectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workflowIDs, diags := r.extractWorkflowIDs(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	indexJSON := buildIndexJSON(data.Name.ValueString(), descriptionValue(data.Description), workflowIDs)
	data.IndexJSON = types.StringValue(indexJSON)
	data.WorkflowCount = types.Int64Value(int64(len(workflowIDs)))

	if !data.OutputDir.IsNull() && !data.OutputDir.IsUnknown() {
		if err := writeIndexFile(data.OutputDir.ValueString(), indexJSON); err != nil {
			resp.Diagnostics.AddError("Failed to write index manifest", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowCollectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkflowCollectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Clean up the index.json file if output_dir was set
	if !data.OutputDir.IsNull() && !data.OutputDir.IsUnknown() {
		indexPath := filepath.Join(data.OutputDir.ValueString(), "index.json")
		if err := os.Remove(indexPath); err != nil && !os.IsNotExist(err) {
			tflog.Warn(ctx, "Failed to remove index.json during delete", map[string]interface{}{
				"path":  indexPath,
				"error": err.Error(),
			})
		}
	}
}

// extractWorkflowIDs converts the Workflows list attribute to a string slice.
func (r *WorkflowCollectionResource) extractWorkflowIDs(ctx context.Context, data WorkflowCollectionModel) ([]string, diag.Diagnostics) {
	var ids []string

	diags := data.Workflows.ElementsAs(ctx, &ids, false)
	if ids == nil {
		ids = []string{}
	}
	return ids, diags
}

// buildIndexJSON creates the JSON manifest string from collection metadata.
func buildIndexJSON(name, description string, workflowIDs []string) string {
	manifest := indexManifest{
		Name:          name,
		Description:   description,
		WorkflowCount: len(workflowIDs),
		Workflows:     make([]indexWorkflow, len(workflowIDs)),
	}
	for i, id := range workflowIDs {
		manifest.Workflows[i] = indexWorkflow{ID: id, Index: i}
	}

	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

// writeIndexFile creates the output directory and writes index.json.
func writeIndexFile(dir, indexJSON string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	indexPath := filepath.Join(dir, "index.json")
	if err := os.WriteFile(indexPath, []byte(indexJSON), 0o644); err != nil {
		return fmt.Errorf("writing index.json: %w", err)
	}
	return nil
}

// descriptionValue returns the string value or empty string for null/unknown.
func descriptionValue(d types.String) string {
	if d.IsNull() || d.IsUnknown() {
		return ""
	}
	return d.ValueString()
}
