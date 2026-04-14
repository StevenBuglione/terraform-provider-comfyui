package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	Host                             types.String `tfsdk:"host"`
	Port                             types.Int64  `tfsdk:"port"`
	APIKey                           types.String `tfsdk:"api_key"`
	ComfyOrgAuthToken                types.String `tfsdk:"comfy_org_auth_token"`
	ComfyOrgAPIKey                   types.String `tfsdk:"comfy_org_api_key"`
	DefaultWorkflowExtraDataJSON     types.String `tfsdk:"default_workflow_extra_data_json"`
	UnsupportedDynamicValidationMode types.String `tfsdk:"unsupported_dynamic_validation_mode"`
}

type providerSettings struct {
	Host                             string
	Port                             int64
	APIKey                           string
	ComfyOrgAuthToken                string
	ComfyOrgAPIKey                   string
	DefaultWorkflowExtraData         map[string]interface{}
	UnsupportedDynamicValidationMode string
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
			"comfy_org_auth_token": schema.StringAttribute{
				Description: "Comfy partner auth token to inject into workflow execution extra_data for partner-backed nodes. Can also be set with COMFYUI_COMFY_ORG_AUTH_TOKEN or AUTH_TOKEN_COMFY_ORG.",
				Optional:    true,
				Sensitive:   true,
			},
			"comfy_org_api_key": schema.StringAttribute{
				Description: "Comfy partner API key to inject into workflow execution extra_data for partner-backed nodes. Can also be set with COMFYUI_COMFY_ORG_API_KEY or API_KEY_COMFY_ORG.",
				Optional:    true,
				Sensitive:   true,
			},
			"default_workflow_extra_data_json": schema.StringAttribute{
				Description: "Default JSON object merged into comfyui_workflow extra_data for every execution. Can also be set with COMFYUI_DEFAULT_WORKFLOW_EXTRA_DATA_JSON.",
				Optional:    true,
			},
			"unsupported_dynamic_validation_mode": schema.StringAttribute{
				Description: "How unsupported dynamic-expression validation should behave for generated node resource planning and comfyui_workflow preflight: error, warning, or ignore. Can also be set with COMFYUI_UNSUPPORTED_DYNAMIC_VALIDATION_MODE. Defaults to error.",
				Optional:    true,
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

	settings, err := resolveProviderSettings(config, os.Getenv)
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider configuration", err.Error())
		return
	}

	c := client.NewClient(settings.Host, settings.Port, settings.APIKey)
	c.ComfyOrgAuthToken = settings.ComfyOrgAuthToken
	c.ComfyOrgAPIKey = settings.ComfyOrgAPIKey
	c.DefaultWorkflowExtraData = settings.DefaultWorkflowExtraData
	c.UnsupportedDynamicValidationMode = settings.UnsupportedDynamicValidationMode

	tflog.Debug(ctx, "ComfyUI client configured", map[string]interface{}{
		"host": settings.Host,
		"port": settings.Port,
	})

	resp.DataSourceData = c
	resp.ResourceData = c
}

func resolveProviderSettings(config ComfyUIProviderModel, getenv func(string) string) (providerSettings, error) {
	settings := providerSettings{
		Host:                             "localhost",
		Port:                             8188,
		UnsupportedDynamicValidationMode: "error",
		DefaultWorkflowExtraData:         map[string]interface{}{},
	}

	if !config.Host.IsNull() && !config.Host.IsUnknown() {
		settings.Host = config.Host.ValueString()
	} else if v := getenv("COMFYUI_HOST"); v != "" {
		settings.Host = v
	}

	if !config.Port.IsNull() && !config.Port.IsUnknown() {
		settings.Port = config.Port.ValueInt64()
	} else if v := getenv("COMFYUI_PORT"); v != "" {
		var parsed int64
		if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil {
			settings.Port = parsed
		}
	}

	if !config.APIKey.IsNull() && !config.APIKey.IsUnknown() {
		settings.APIKey = config.APIKey.ValueString()
	} else if v := getenv("COMFYUI_API_KEY"); v != "" {
		settings.APIKey = v
	}

	if !config.ComfyOrgAuthToken.IsNull() && !config.ComfyOrgAuthToken.IsUnknown() {
		settings.ComfyOrgAuthToken = config.ComfyOrgAuthToken.ValueString()
	} else if v := getenv("COMFYUI_COMFY_ORG_AUTH_TOKEN"); v != "" {
		settings.ComfyOrgAuthToken = v
	} else if v := getenv("AUTH_TOKEN_COMFY_ORG"); v != "" {
		settings.ComfyOrgAuthToken = v
	}

	if !config.ComfyOrgAPIKey.IsNull() && !config.ComfyOrgAPIKey.IsUnknown() {
		settings.ComfyOrgAPIKey = config.ComfyOrgAPIKey.ValueString()
	} else if v := getenv("COMFYUI_COMFY_ORG_API_KEY"); v != "" {
		settings.ComfyOrgAPIKey = v
	} else if v := getenv("API_KEY_COMFY_ORG"); v != "" {
		settings.ComfyOrgAPIKey = v
	}

	extraDataJSON := ""
	if !config.DefaultWorkflowExtraDataJSON.IsNull() && !config.DefaultWorkflowExtraDataJSON.IsUnknown() {
		extraDataJSON = config.DefaultWorkflowExtraDataJSON.ValueString()
	} else if v := getenv("COMFYUI_DEFAULT_WORKFLOW_EXTRA_DATA_JSON"); v != "" {
		extraDataJSON = v
	}
	if strings.TrimSpace(extraDataJSON) != "" {
		if err := json.Unmarshal([]byte(extraDataJSON), &settings.DefaultWorkflowExtraData); err != nil {
			return providerSettings{}, fmt.Errorf("default_workflow_extra_data_json must be valid JSON: %w", err)
		}
		if settings.DefaultWorkflowExtraData == nil {
			settings.DefaultWorkflowExtraData = map[string]interface{}{}
		}
	}

	mode := ""
	if !config.UnsupportedDynamicValidationMode.IsNull() && !config.UnsupportedDynamicValidationMode.IsUnknown() {
		mode = config.UnsupportedDynamicValidationMode.ValueString()
	} else if v := getenv("COMFYUI_UNSUPPORTED_DYNAMIC_VALIDATION_MODE"); v != "" {
		mode = v
	}
	if strings.TrimSpace(mode) != "" {
		settings.UnsupportedDynamicValidationMode = strings.ToLower(strings.TrimSpace(mode))
	}

	switch settings.UnsupportedDynamicValidationMode {
	case "error", "warning", "ignore":
	default:
		return providerSettings{}, fmt.Errorf("unsupported_dynamic_validation_mode must be one of: error, warning, ignore")
	}

	return settings, nil
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
		datasources.NewNodeSchemaDataSource,
		datasources.NewInventoryDataSource,
		datasources.NewWorkflowHistoryDataSource,
		datasources.NewOutputDataSource,
		datasources.NewPromptJSONDataSource,
		datasources.NewPromptValidationDataSource,
		datasources.NewPromptToWorkspaceDataSource,
		datasources.NewPromptToTerraformDataSource,
		datasources.NewWorkspaceJSONDataSource,
		datasources.NewWorkspaceValidationDataSource,
		datasources.NewWorkspaceToPromptDataSource,
		datasources.NewWorkspaceToTerraformDataSource,
		datasources.NewSubgraphCatalogDataSource,
		datasources.NewSubgraphDefinitionDataSource,
		datasources.NewProviderInfoDataSource(p.version),
		datasources.NewJobDataSource,
		datasources.NewJobsDataSource,
	}
}
