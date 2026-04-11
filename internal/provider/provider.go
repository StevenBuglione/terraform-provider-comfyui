package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/datasources"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/resources"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/resources/generated"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = &ComfyUIProvider{}

type ComfyUIProvider struct {
	version string
}

type ComfyUIProviderModel struct {
	Host   types.String `tfsdk:"host"`
	Port   types.Int64  `tfsdk:"port"`
	APIKey types.String `tfsdk:"api_key"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ComfyUIProvider{
			version: version,
		}
	}
}

func (p *ComfyUIProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "comfyui"
	resp.Version = p.version
}

func (p *ComfyUIProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf(
			"Terraform provider for ComfyUI - a node-based Stable Diffusion GUI. "+
				"Generated from ComfyUI %s with %d node resources.",
			generated.ComfyUIVersion, generated.NodeCount,
		),
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "ComfyUI server host. Can also be set with COMFYUI_HOST environment variable. Defaults to localhost.",
				Optional:    true,
			},
			"port": schema.Int64Attribute{
				Description: "ComfyUI server port. Can also be set with COMFYUI_PORT environment variable. Defaults to 8188.",
				Optional:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "API key for ComfyUI authentication (if enabled). Can also be set with COMFYUI_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *ComfyUIProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring ComfyUI provider",
		map[string]interface{}{
			"provider_version": p.version,
			"comfyui_version":  generated.ComfyUIVersion,
			"node_count":       generated.NodeCount,
		},
	)

	var config ComfyUIProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := "localhost"
	if !config.Host.IsNull() && !config.Host.IsUnknown() {
		host = config.Host.ValueString()
	} else if v := os.Getenv("COMFYUI_HOST"); v != "" {
		host = v
	}

	port := int64(8188)
	if !config.Port.IsNull() && !config.Port.IsUnknown() {
		port = config.Port.ValueInt64()
	} else if v := os.Getenv("COMFYUI_PORT"); v != "" {
		var p int64
		_, err := fmt.Sscanf(v, "%d", &p)
		if err == nil {
			port = p
		}
	}

	apiKey := ""
	if !config.APIKey.IsNull() && !config.APIKey.IsUnknown() {
		apiKey = config.APIKey.ValueString()
	} else if v := os.Getenv("COMFYUI_API_KEY"); v != "" {
		apiKey = v
	}

	c := client.NewClient(host, port, apiKey)

	tflog.Debug(ctx, "ComfyUI client configured", map[string]interface{}{
		"host": host,
		"port": port,
	})

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *ComfyUIProvider) Resources(_ context.Context) []func() resource.Resource {
	all := generated.AllResources()
	all = append(all, resources.NewWorkflowResource)
	all = append(all, resources.NewWorkflowCollectionResource)
	all = append(all, resources.NewWorkspaceResource)
	all = append(all, resources.NewPromptArtifactResource)
	all = append(all, resources.NewWorkspaceArtifactResource)
	all = append(all, resources.NewSubgraphResource)
	all = append(all, resources.NewUploadedImageResource)
	all = append(all, resources.NewUploadedMaskResource)
	all = append(all, resources.NewOutputArtifactResource)
	return all
}

func (p *ComfyUIProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewSystemStatsDataSource,
		datasources.NewQueueDataSource,
		datasources.NewNodeInfoDataSource,
		datasources.NewWorkflowHistoryDataSource,
		datasources.NewOutputDataSource,
		datasources.NewPromptJSONDataSource,
		datasources.NewPromptValidationDataSource,
		datasources.NewPromptToWorkspaceDataSource,
		datasources.NewWorkspaceJSONDataSource,
		datasources.NewWorkspaceValidationDataSource,
		datasources.NewWorkspaceToPromptDataSource,
		datasources.NewSubgraphCatalogDataSource,
		datasources.NewSubgraphDefinitionDataSource,
		datasources.NewProviderInfoDataSource(p.version),
	}
}
