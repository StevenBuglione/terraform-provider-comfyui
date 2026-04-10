package resources

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &WorkspaceArtifactResource{}
	_ resource.ResourceWithImportState = &WorkspaceArtifactResource{}
)

type WorkspaceArtifactResource struct{}

type WorkspaceArtifactModel struct {
	ID          types.String `tfsdk:"id"`
	Path        types.String `tfsdk:"path"`
	ContentJSON types.String `tfsdk:"content_json"`
	SHA256      types.String `tfsdk:"sha256"`
}

func NewWorkspaceArtifactResource() resource.Resource {
	return &WorkspaceArtifactResource{}
}

func (r *WorkspaceArtifactResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_artifact"
}

func (r *WorkspaceArtifactResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Materializes Terraform-owned ComfyUI workspace or subgraph JSON to a file on disk.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Managed artifact identifier, equal to the absolute file path.",
			},
			"path": schema.StringAttribute{
				Required:    true,
				Description: "File path where the workspace JSON should be written.",
			},
			"content_json": schema.StringAttribute{
				Required:    true,
				Description: "Exact ComfyUI workspace or subgraph JSON to persist.",
			},
			"sha256": schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the persisted workspace JSON.",
			},
		},
	}
}

func (r *WorkspaceArtifactResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkspaceArtifactModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sha, err := writeManagedArtifactFile(data.Path.ValueString(), data.ContentJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to write workspace artifact", err.Error())
		return
	}

	data.ID = types.StringValue(data.Path.ValueString())
	data.SHA256 = types.StringValue(sha)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkspaceArtifactResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkspaceArtifactModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, sha, err := readManagedArtifactFile(data.Path.ValueString())
	if err != nil {
		if os.IsNotExist(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read workspace artifact", err.Error())
		return
	}

	data.ContentJSON = types.StringValue(content)
	data.SHA256 = types.StringValue(sha)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkspaceArtifactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WorkspaceArtifactModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WorkspaceArtifactModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sha, err := writeManagedArtifactFile(data.Path.ValueString(), data.ContentJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to update workspace artifact", err.Error())
		return
	}
	if err := cleanupPreviousArtifactFile(state.Path.ValueString(), data.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to clean up previous workspace artifact", err.Error())
		return
	}

	data.ID = types.StringValue(data.Path.ValueString())
	data.SHA256 = types.StringValue(sha)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkspaceArtifactResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkspaceArtifactModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := os.Remove(data.Path.ValueString()); err != nil && !os.IsNotExist(err) {
		resp.Diagnostics.AddError("Unable to delete workspace artifact", err.Error())
	}
}

func (r *WorkspaceArtifactResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import path", "Import path must not be empty.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)

	content, sha, err := readManagedArtifactFile(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to import workspace artifact", fmt.Sprintf("Could not read %q: %s", req.ID, err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_json"), content)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("sha256"), sha)...)
}
