package datasources

import (
	"context"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/resources/generated"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ProviderInfoDataSource{}

type ProviderInfoDataSource struct {
	providerVersion string
	client          *client.Client
}

type ProviderInfoModel struct {
	ID                      types.String `tfsdk:"id"`
	ProviderVersion         types.String `tfsdk:"provider_version"`
	ComfyUIVersion          types.String `tfsdk:"comfyui_version"`
	NodeCount               types.Int64  `tfsdk:"node_count"`
	ExtractedAt             types.String `tfsdk:"extracted_at"`
	ServiceAPIKeyConfigured types.Bool   `tfsdk:"service_api_key_configured"`
	PartnerAuthConfigured   types.Bool   `tfsdk:"partner_auth_configured"`
	ConfiguredAuthFamilies  types.List   `tfsdk:"configured_auth_families"`
}

type providerAuthPosture struct {
	ServiceAPIKeyConfigured bool
	PartnerAuthConfigured   bool
	ConfiguredAuthFamilies  []string
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
			"service_api_key_configured": schema.BoolAttribute{
				Description: "Whether the provider currently has a ComfyUI service API key configured.",
				Computed:    true,
			},
			"partner_auth_configured": schema.BoolAttribute{
				Description: "Whether the provider currently has explicit partner execution auth configured.",
				Computed:    true,
			},
			"configured_auth_families": schema.ListAttribute{
				Description: "Auth families that are explicitly configured on the provider for workflow execution.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *ProviderInfoDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if clientData, ok := req.ProviderData.(*client.Client); ok {
		d.client = clientData
	}
}

func (d *ProviderInfoDataSource) Read(_ context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	posture := authPosture(d.client)
	families := make([]attr.Value, 0, len(posture.ConfiguredAuthFamilies))
	for _, family := range posture.ConfiguredAuthFamilies {
		families = append(families, types.StringValue(family))
	}

	state := ProviderInfoModel{
		ID:                      types.StringValue("provider-info"),
		ProviderVersion:         types.StringValue(d.providerVersion),
		ComfyUIVersion:          types.StringValue(generated.ComfyUIVersion),
		NodeCount:               types.Int64Value(int64(generated.NodeCount)),
		ExtractedAt:             types.StringValue(generated.ExtractedAt),
		ServiceAPIKeyConfigured: types.BoolValue(posture.ServiceAPIKeyConfigured),
		PartnerAuthConfigured:   types.BoolValue(posture.PartnerAuthConfigured),
		ConfiguredAuthFamilies:  types.ListValueMust(types.StringType, families),
	}
	resp.Diagnostics.Append(resp.State.Set(context.Background(), &state)...)
}

func authPosture(clientData *client.Client) providerAuthPosture {
	posture := providerAuthPosture{}
	if clientData == nil {
		return posture
	}
	posture.ServiceAPIKeyConfigured = clientData.APIKey != ""
	if clientData.ComfyOrgAPIKey != "" || clientData.ComfyOrgAuthToken != "" {
		posture.PartnerAuthConfigured = true
		posture.ConfiguredAuthFamilies = append(posture.ConfiguredAuthFamilies, "comfy_org")
	}
	return posture
}
