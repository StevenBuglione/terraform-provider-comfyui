package datasources

import (
	"context"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/artifacts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &WorkspaceToPromptDataSource{}

type WorkspaceToPromptDataSource struct{}

type WorkspaceToPromptModel struct {
	WorkspaceJSON     types.String `tfsdk:"workspace_json"`
	PromptJSON        types.String `tfsdk:"prompt_json"`
	Fidelity          types.String `tfsdk:"fidelity"`
	PreservedFields   types.List   `tfsdk:"preserved_fields"`
	SynthesizedFields types.List   `tfsdk:"synthesized_fields"`
	UnsupportedFields types.List   `tfsdk:"unsupported_fields"`
	Notes             types.List   `tfsdk:"notes"`
}

func NewWorkspaceToPromptDataSource() datasource.DataSource {
	return &WorkspaceToPromptDataSource{}
}

func (d *WorkspaceToPromptDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_to_prompt"
}

func (d *WorkspaceToPromptDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Translates ComfyUI workspace or subgraph JSON into prompt JSON with structured fidelity reporting.",
		Attributes: map[string]schema.Attribute{
			"workspace_json": schema.StringAttribute{
				Required:    true,
				Description: "ComfyUI workspace or subgraph JSON to translate.",
			},
			"prompt_json": schema.StringAttribute{
				Computed:    true,
				Description: "Translated prompt JSON.",
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

func (d *WorkspaceToPromptDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WorkspaceToPromptModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := workspaceToPromptStateFromInput(stringValue(config.WorkspaceJSON))
	if err != nil {
		resp.Diagnostics.AddError("Unable to translate workspace JSON", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func workspaceToPromptStateFromInput(raw string) (WorkspaceToPromptModel, error) {
	workspace, err := artifacts.ParseWorkspaceJSON(raw)
	if err != nil {
		return WorkspaceToPromptModel{}, err
	}
	prompt, report, err := artifacts.TranslateWorkspaceToPrompt(workspace)
	if err != nil {
		return WorkspaceToPromptModel{}, err
	}
	promptJSON, err := prompt.JSON()
	if err != nil {
		return WorkspaceToPromptModel{}, err
	}
	return WorkspaceToPromptModel{
		WorkspaceJSON:     types.StringValue(raw),
		PromptJSON:        types.StringValue(promptJSON),
		Fidelity:          types.StringValue(report.Fidelity()),
		PreservedFields:   stringsListValue(report.PreservedFields),
		SynthesizedFields: stringsListValue(report.SynthesizedFields),
		UnsupportedFields: stringsListValue(report.UnsupportedFields),
		Notes:             stringsListValue(report.Notes),
	}, nil
}
