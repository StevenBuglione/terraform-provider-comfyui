package datasources

import (
	"context"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/resources/generated"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ProviderInfoDataSource{}

type ProviderInfoDataSource struct {
	providerVersion string
}

type ProviderInfoModel struct {
	ID              types.String `tfsdk:"id"`
	ProviderVersion types.String `tfsdk:"provider_version"`
	ComfyUIVersion  types.String `tfsdk:"comfyui_version"`
	NodeCount       types.Int64  `tfsdk:"node_count"`
	ExtractedAt     types.String `tfsdk:"extracted_at"`
}

func NewProviderInfoDataSource(providerVersion string) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &ProviderInfoDataSource{providerVersion: providerVersion}
	}
}

func (d *ProviderInfoDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_provider_info"
}

func (d *ProviderInfoDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides version and compatibility information about the ComfyUI Terraform provider.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this data source.",
				Computed:    true,
			},
			"provider_version": schema.StringAttribute{
				Description: "The version of the Terraform provider.",
				Computed:    true,
			},
			"comfyui_version": schema.StringAttribute{
				Description: "The version of ComfyUI that node resources were generated from.",
				Computed:    true,
			},
			"node_count": schema.Int64Attribute{
				Description: "The total number of ComfyUI node resources available.",
				Computed:    true,
			},
			"extracted_at": schema.StringAttribute{
				Description: "The timestamp when node specifications were extracted from ComfyUI.",
				Computed:    true,
			},
		},
	}
}

func (d *ProviderInfoDataSource) Configure(_ context.Context, _ datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
}

func (d *ProviderInfoDataSource) Read(_ context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	state := ProviderInfoModel{
		ID:              types.StringValue("provider-info"),
		ProviderVersion: types.StringValue(d.providerVersion),
		ComfyUIVersion:  types.StringValue(generated.ComfyUIVersion),
		NodeCount:       types.Int64Value(int64(generated.NodeCount)),
		ExtractedAt:     types.StringValue(generated.ExtractedAt),
	}
	resp.Diagnostics.Append(resp.State.Set(context.Background(), &state)...)
}
