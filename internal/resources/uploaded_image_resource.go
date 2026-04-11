package resources

import (
	"context"
	"fmt"
	"os"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &UploadedImageResource{}
var _ resource.ResourceWithConfigure = &UploadedImageResource{}

type UploadedImageResource struct {
	client *client.Client
}

type UploadedImageModel struct {
	ID        types.String `tfsdk:"id"`
	FilePath  types.String `tfsdk:"file_path"`
	Filename  types.String `tfsdk:"filename"`
	Subfolder types.String `tfsdk:"subfolder"`
	Type      types.String `tfsdk:"type"`
	Overwrite types.Bool   `tfsdk:"overwrite"`
	SHA256    types.String `tfsdk:"sha256"`
	URL       types.String `tfsdk:"url"`
}

func NewUploadedImageResource() resource.Resource {
	return &UploadedImageResource{}
}

func (r *UploadedImageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uploaded_image"
}

func (r *UploadedImageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UploadedImageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Uploads a local image into ComfyUI and tracks the resulting remote file path.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Remote ComfyUI file identifier in {type}/{subfolder}/{filename} form.",
			},
			"file_path": schema.StringAttribute{
				Required:    true,
				Description: "Local source file path to upload.",
			},
			"filename": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Requested filename, updated to the actual remote filename returned by ComfyUI.",
			},
			"subfolder": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Optional ComfyUI subfolder for the uploaded file.",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("input"),
				Description: "ComfyUI storage type for the uploaded file: input, output, or temp.",
			},
			"overwrite": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the upload should overwrite an existing file when ComfyUI allows it.",
			},
			"sha256": schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the local source file at upload time.",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "ComfyUI /view URL for the uploaded file.",
			},
		},
	}
}

func (r *UploadedImageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UploadedImageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.upload(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Unable to upload image", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UploadedImageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UploadedImageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.refresh(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Unable to refresh uploaded image", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UploadedImageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UploadedImageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state UploadedImageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.upload(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Unable to update uploaded image", err.Error())
		return
	}
	if stringValue(state.ID) != "" && stringValue(state.ID) != stringValue(data.ID) {
		addRemoteDeleteWarning(&resp.Diagnostics, state.ID.ValueString())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UploadedImageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UploadedImageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if id := stringValue(data.ID); id != "" {
		addRemoteDeleteWarning(&resp.Diagnostics, id)
	}
}

func (r *UploadedImageResource) upload(_ context.Context, data *UploadedImageModel) error {
	if r.client == nil {
		return fmt.Errorf("client not configured")
	}

	sha, err := localFileSHA256(data.FilePath.ValueString())
	if err != nil {
		return err
	}

	uploadType := stringValue(data.Type)
	overwrite := boolValueOrDefault(data.Overwrite, true)
	resp, err := r.client.UploadImage(
		data.FilePath.ValueString(),
		stringValue(data.Filename),
		stringValue(data.Subfolder),
		uploadType,
		overwrite,
	)
	if err != nil {
		return err
	}

	data.ID = types.StringValue(remoteFileID(resp.Type, resp.Subfolder, resp.Name))
	data.Filename = types.StringValue(resp.Name)
	data.Subfolder = types.StringValue(resp.Subfolder)
	data.Type = types.StringValue(resp.Type)
	data.Overwrite = types.BoolValue(overwrite)
	data.SHA256 = types.StringValue(sha)
	data.URL = types.StringValue(r.client.GetViewURL(resp.Name, resp.Subfolder, resp.Type))
	return nil
}

func (r *UploadedImageResource) refresh(_ context.Context, data *UploadedImageModel) (bool, error) {
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

	sha := stringValue(data.SHA256)
	if currentSHA, err := localFileSHA256(data.FilePath.ValueString()); err == nil {
		sha = currentSHA
	} else if !os.IsNotExist(err) {
		return true, err
	}

	data.ID = types.StringValue(remoteFileID(stringValue(data.Type), stringValue(data.Subfolder), stringValue(data.Filename)))
	data.URL = types.StringValue(r.client.GetViewURL(stringValue(data.Filename), stringValue(data.Subfolder), stringValue(data.Type)))
	data.SHA256 = types.StringValue(sha)
	return true, nil
}
