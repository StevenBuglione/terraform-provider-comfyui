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
	_ resource.Resource                = &PromptArtifactResource{}
	_ resource.ResourceWithImportState = &PromptArtifactResource{}
)

type PromptArtifactResource struct{}

type PromptArtifactModel struct {
	ID          types.String `tfsdk:"id"`
	Path        types.String `tfsdk:"path"`
	ContentJSON types.String `tfsdk:"content_json"`
	SHA256      types.String `tfsdk:"sha256"`
}

func NewPromptArtifactResource() resource.Resource {
	return &PromptArtifactResource{}
}

func (r *PromptArtifactResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt_artifact"
}

func (r *PromptArtifactResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Materializes Terraform-owned ComfyUI prompt JSON to a file on disk.",
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
				Description: "File path where the prompt JSON should be written.",
			},
			"content_json": schema.StringAttribute{
				Required:    true,
				Description: "Exact ComfyUI prompt JSON to persist.",
			},
			"sha256": schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the persisted prompt JSON.",
			},
		},
	}
}

func (r *PromptArtifactResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PromptArtifactModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sha, err := writeManagedArtifactFile(data.Path.ValueString(), data.ContentJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to write prompt artifact", err.Error())
		return
	}

	data.ID = types.StringValue(data.Path.ValueString())
	data.SHA256 = types.StringValue(sha)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PromptArtifactResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PromptArtifactModel
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
		resp.Diagnostics.AddError("Unable to read prompt artifact", err.Error())
		return
	}

	data.ContentJSON = types.StringValue(content)
	data.SHA256 = types.StringValue(sha)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PromptArtifactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PromptArtifactModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state PromptArtifactModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sha, err := writeManagedArtifactFile(data.Path.ValueString(), data.ContentJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to update prompt artifact", err.Error())
		return
	}
	if err := cleanupPreviousArtifactFile(state.Path.ValueString(), data.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to clean up previous prompt artifact", err.Error())
		return
	}

	data.ID = types.StringValue(data.Path.ValueString())
	data.SHA256 = types.StringValue(sha)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PromptArtifactResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PromptArtifactModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := os.Remove(data.Path.ValueString()); err != nil && !os.IsNotExist(err) {
		resp.Diagnostics.AddError("Unable to delete prompt artifact", err.Error())
	}
}

func (r *PromptArtifactResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import path", "Import path must not be empty.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)

	content, sha, err := readManagedArtifactFile(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to import prompt artifact", fmt.Sprintf("Could not read %q: %s", req.ID, err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_json"), content)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("sha256"), sha)...)
}
