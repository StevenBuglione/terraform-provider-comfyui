package resources

import (
	"context"
	"fmt"
	"os"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &OutputArtifactResource{}
var _ resource.ResourceWithConfigure = &OutputArtifactResource{}

type OutputArtifactResource struct {
	client *client.Client
}

type OutputArtifactModel struct {
	ID            types.String `tfsdk:"id"`
	Filename      types.String `tfsdk:"filename"`
	Subfolder     types.String `tfsdk:"subfolder"`
	Type          types.String `tfsdk:"type"`
	Path          types.String `tfsdk:"path"`
	SHA256        types.String `tfsdk:"sha256"`
	ContentLength types.Int64  `tfsdk:"content_length"`
	URL           types.String `tfsdk:"url"`
}

func NewOutputArtifactResource() resource.Resource {
	return &OutputArtifactResource{}
}

func (r *OutputArtifactResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_output_artifact"
}

func (r *OutputArtifactResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OutputArtifactResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Downloads a remote ComfyUI file via /view and manages the local artifact on disk.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Absolute local path of the managed artifact.",
			},
			"filename": schema.StringAttribute{
				Required:    true,
				Description: "Remote ComfyUI filename to download.",
			},
			"subfolder": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Optional remote subfolder for the file.",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("output"),
				Description: "Remote ComfyUI storage type for the file: input, output, or temp.",
			},
			"path": schema.StringAttribute{
				Required:    true,
				Description: "Local destination path for the downloaded artifact.",
			},
			"sha256": schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the downloaded local file.",
			},
			"content_length": schema.Int64Attribute{
				Computed:    true,
				Description: "Downloaded file size in bytes.",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "ComfyUI /view URL for the remote file.",
			},
		},
	}
}

func (r *OutputArtifactResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OutputArtifactModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.download(ctx, &data, ""); err != nil {
		resp.Diagnostics.AddError("Unable to download output artifact", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OutputArtifactResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OutputArtifactModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.refresh(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Unable to refresh output artifact", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OutputArtifactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OutputArtifactModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OutputArtifactModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.download(ctx, &data, state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to update output artifact", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OutputArtifactResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OutputArtifactModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := os.Remove(data.Path.ValueString()); err != nil && !os.IsNotExist(err) {
		resp.Diagnostics.AddError("Unable to delete output artifact", err.Error())
	}
}

func (r *OutputArtifactResource) download(_ context.Context, data *OutputArtifactModel, previousPath string) error {
	if r.client == nil {
		return fmt.Errorf("client not configured")
	}

	resp, err := r.client.DownloadView(stringValue(data.Filename), stringValue(data.Subfolder), stringValue(data.Type))
	if err != nil {
		return err
	}

	absPath, sha, length, err := writeManagedBinaryFile(data.Path.ValueString(), resp.Content)
	if err != nil {
		return err
	}
	if err := cleanupPreviousArtifactFile(previousPath, data.Path.ValueString()); err != nil {
		return err
	}

	data.ID = types.StringValue(absPath)
	data.Subfolder = types.StringValue(stringValue(data.Subfolder))
	data.Type = types.StringValue(stringValue(data.Type))
	data.SHA256 = types.StringValue(sha)
	data.ContentLength = types.Int64Value(length)
	data.URL = types.StringValue(r.client.GetViewURL(stringValue(data.Filename), stringValue(data.Subfolder), stringValue(data.Type)))
	return nil
}

func (r *OutputArtifactResource) refresh(ctx context.Context, data *OutputArtifactModel) (bool, error) {
	if r.client == nil {
		return false, fmt.Errorf("client not configured")
	}

	exists, err := r.client.CheckOutputExists(stringValue(data.Filename), stringValue(data.Subfolder), stringValue(data.Type))
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	if _, err := os.Stat(data.Path.ValueString()); os.IsNotExist(err) {
		return true, r.download(ctx, data, "")
	} else if err != nil {
		return true, err
	}

	_, sha, length, err := readManagedBinaryFile(data.Path.ValueString())
	if err != nil {
		return true, err
	}

	data.ID = types.StringValue(stringValue(data.ID))
	data.SHA256 = types.StringValue(sha)
	data.ContentLength = types.Int64Value(length)
	data.URL = types.StringValue(r.client.GetViewURL(stringValue(data.Filename), stringValue(data.Subfolder), stringValue(data.Type)))
	return true, nil
}
