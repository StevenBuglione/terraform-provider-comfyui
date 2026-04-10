package datasources

import (
	"context"
	"fmt"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/artifacts"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &PromptToWorkspaceDataSource{}
var _ datasource.DataSourceWithConfigure = &PromptToWorkspaceDataSource{}

type PromptToWorkspaceDataSource struct {
	client *client.Client
}

type PromptToWorkspaceModel struct {
	Name              types.String `tfsdk:"name"`
	PromptJSON        types.String `tfsdk:"prompt_json"`
	WorkspaceJSON     types.String `tfsdk:"workspace_json"`
	Fidelity          types.String `tfsdk:"fidelity"`
	PreservedFields   types.List   `tfsdk:"preserved_fields"`
	SynthesizedFields types.List   `tfsdk:"synthesized_fields"`
	UnsupportedFields types.List   `tfsdk:"unsupported_fields"`
	Notes             types.List   `tfsdk:"notes"`
}

func NewPromptToWorkspaceDataSource() datasource.DataSource {
	return &PromptToWorkspaceDataSource{}
}

func (d *PromptToWorkspaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt_to_workspace"
}

func (d *PromptToWorkspaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PromptToWorkspaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Translates ComfyUI prompt JSON into a synthetic workspace JSON representation with structured fidelity reporting.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Optional name for the translated workspace.",
			},
			"prompt_json": schema.StringAttribute{
				Required:    true,
				Description: "ComfyUI prompt JSON to translate.",
			},
			"workspace_json": schema.StringAttribute{
				Computed:    true,
				Description: "Translated workspace JSON.",
			},
			"fidelity": schema.StringAttribute{
				Computed:    true,
				Description: "Overall translation fidelity classification.",
			},
			"preserved_fields": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Field paths preserved exactly during translation.",
			},
			"synthesized_fields": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Field paths synthesized during translation.",
			},
			"unsupported_fields": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Field paths dropped or not representable during translation.",
			},
			"notes": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Human-readable translation notes.",
			},
		},
	}
}

func (d *PromptToWorkspaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PromptToWorkspaceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The ComfyUI client is required to fetch object_info for prompt-to-workspace translation.")
		return
	}

	nodeInfo, err := d.client.GetObjectInfo()
	if err != nil {
		resp.Diagnostics.AddError("Unable to fetch object info", err.Error())
		return
	}

	state, err := promptToWorkspaceStateFromInput(stringValue(config.Name), stringValue(config.PromptJSON), nodeInfo)
	if err != nil {
		resp.Diagnostics.AddError("Unable to translate prompt JSON", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func promptToWorkspaceStateFromInput(name string, raw string, nodeInfo map[string]client.NodeInfo) (PromptToWorkspaceModel, error) {
	if name == "" {
		name = "translated-workspace"
	}
	prompt, err := artifacts.ParsePromptJSON(raw)
	if err != nil {
		return PromptToWorkspaceModel{}, err
	}
	workspace, report, err := artifacts.TranslatePromptToWorkspace(name, prompt, nodeInfo)
	if err != nil {
		return PromptToWorkspaceModel{}, err
	}
	workspaceJSON, err := workspace.JSON()
	if err != nil {
		return PromptToWorkspaceModel{}, err
	}
	return PromptToWorkspaceModel{
		Name:              types.StringValue(name),
		PromptJSON:        types.StringValue(raw),
		WorkspaceJSON:     types.StringValue(workspaceJSON),
		Fidelity:          types.StringValue(report.Fidelity()),
		PreservedFields:   stringsListValue(report.PreservedFields),
		SynthesizedFields: stringsListValue(report.SynthesizedFields),
		UnsupportedFields: stringsListValue(report.UnsupportedFields),
		Notes:             stringsListValue(report.Notes),
	}, nil
}
