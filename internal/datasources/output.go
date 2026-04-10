package datasources

import (
	"context"
	"fmt"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &OutputDataSource{}
var _ datasource.DataSourceWithConfigure = &OutputDataSource{}

type OutputDataSource struct {
	client *client.Client
}

type OutputModel struct {
	Filename  types.String `tfsdk:"filename"`
	Subfolder types.String `tfsdk:"subfolder"`
	Type      types.String `tfsdk:"type"`
	URL       types.String `tfsdk:"url"`
	Exists    types.Bool   `tfsdk:"exists"`
}

func NewOutputDataSource() datasource.DataSource {
	return &OutputDataSource{}
}

func (d *OutputDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_output"
}

func (d *OutputDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *OutputDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a ComfyUI output file.",
		Attributes: map[string]schema.Attribute{
			"filename": schema.StringAttribute{
				Description: "The output filename to look up.",
				Required:    true,
			},
			"subfolder": schema.StringAttribute{
				Description: "Subfolder within the output directory.",
				Optional:    true,
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Output type (e.g., output, input, temp).",
				Optional:    true,
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "Full URL to download the output file.",
				Computed:    true,
			},
			"exists": schema.BoolAttribute{
				Description: "Whether the output file exists on the server.",
				Computed:    true,
			},
		},
	}
}

func (d *OutputDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config OutputModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filename := config.Filename.ValueString()

	subfolder := ""
	if !config.Subfolder.IsNull() && !config.Subfolder.IsUnknown() {
		subfolder = config.Subfolder.ValueString()
	}

	outputType := "output"
	if !config.Type.IsNull() && !config.Type.IsUnknown() {
		outputType = config.Type.ValueString()
	}

	viewURL := d.client.GetViewURL(filename, subfolder, outputType)

	exists, err := d.client.CheckOutputExists(filename, subfolder, outputType)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Unable to check output existence",
			fmt.Sprintf("Could not verify if output %q exists: %s", filename, err.Error()),
		)
		exists = false
	}

	state := OutputModel{
		Filename:  types.StringValue(filename),
		Subfolder: types.StringValue(subfolder),
		Type:      types.StringValue(outputType),
		URL:       types.StringValue(viewURL),
		Exists:    types.BoolValue(exists),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
