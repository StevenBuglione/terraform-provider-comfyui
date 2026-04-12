package datasources

import (
	"context"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/artifacts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &WorkspaceToTerraformDataSource{}

type WorkspaceToTerraformDataSource struct{}

type WorkspaceToTerraformModel struct {
	Name                 types.String `tfsdk:"name"`
	WorkspaceJSON        types.String `tfsdk:"workspace_json"`
	TranslatedPromptJSON types.String `tfsdk:"translated_prompt_json"`
	TerraformIRJSON      types.String `tfsdk:"terraform_ir_json"`
	TerraformHCL         types.String `tfsdk:"terraform_hcl"`
	Fidelity             types.String `tfsdk:"fidelity"`
	PreservedFields      types.List   `tfsdk:"preserved_fields"`
	SynthesizedFields    types.List   `tfsdk:"synthesized_fields"`
	UnsupportedFields    types.List   `tfsdk:"unsupported_fields"`
	Notes                types.List   `tfsdk:"notes"`
}

func NewWorkspaceToTerraformDataSource() datasource.DataSource {
	return &WorkspaceToTerraformDataSource{}
}

func (d *WorkspaceToTerraformDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_to_terraform"
}

func (d *WorkspaceToTerraformDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Translates ComfyUI workspace JSON to prompt JSON, then synthesizes canonical Terraform IR and rendered HCL.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Optional logical workflow name used for future synthesis surfaces.",
			},
			"workspace_json": schema.StringAttribute{
				Required:    true,
				Description: "ComfyUI workspace JSON to synthesize into Terraform.",
			},
			"translated_prompt_json": schema.StringAttribute{
				Computed:    true,
				Description: "Prompt JSON translated from the source workspace.",
			},
			"terraform_ir_json": schema.StringAttribute{
				Computed:    true,
				Description: "Canonical Terraform IR JSON.",
			},
			"terraform_hcl": schema.StringAttribute{
				Computed:    true,
				Description: "Canonical rendered Terraform HCL.",
			},
			"fidelity": schema.StringAttribute{
				Computed:    true,
				Description: "Overall translation plus synthesis fidelity classification.",
			},
			"preserved_fields": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"synthesized_fields": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"unsupported_fields": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"notes": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *WorkspaceToTerraformDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WorkspaceToTerraformModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := workspaceToTerraformStateFromInput(stringValue(config.Name), stringValue(config.WorkspaceJSON))
	if err != nil {
		resp.Diagnostics.AddError("Unable to synthesize Terraform from workspace JSON", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func workspaceToTerraformStateFromInput(name string, raw string) (WorkspaceToTerraformModel, error) {
	if name == "" {
		name = "workflow"
	}
	workspace, err := artifacts.ParseWorkspaceJSON(raw)
	if err != nil {
		return WorkspaceToTerraformModel{}, err
	}
	prompt, report, err := artifacts.TranslateWorkspaceToPrompt(workspace)
	if err != nil {
		return WorkspaceToTerraformModel{}, err
	}
	promptJSON, err := prompt.JSON()
	if err != nil {
		return WorkspaceToTerraformModel{}, err
	}
	ir, synthesisReport, err := artifacts.BuildTerraformIRFromPrompt(prompt)
	if err != nil {
		return WorkspaceToTerraformModel{}, err
	}
	irJSON, err := ir.JSON()
	if err != nil {
		return WorkspaceToTerraformModel{}, err
	}
	hcl, err := artifacts.RenderTerraformHCL(ir)
	if err != nil {
		return WorkspaceToTerraformModel{}, err
	}

	combined := artifacts.NewTranslationReport()
	combined.PreservedFields = append(combined.PreservedFields, report.PreservedFields...)
	combined.PreservedFields = append(combined.PreservedFields, synthesisReport.PreservedFields...)
	combined.SynthesizedFields = append(combined.SynthesizedFields, report.SynthesizedFields...)
	combined.SynthesizedFields = append(combined.SynthesizedFields, synthesisReport.SynthesizedFields...)
	combined.UnsupportedFields = append(combined.UnsupportedFields, report.UnsupportedFields...)
	combined.UnsupportedFields = append(combined.UnsupportedFields, synthesisReport.UnsupportedFields...)
	combined.Notes = append(combined.Notes, report.Notes...)
	combined.Notes = append(combined.Notes, synthesisReport.Notes...)

	return WorkspaceToTerraformModel{
		Name:                 types.StringValue(name),
		WorkspaceJSON:        types.StringValue(raw),
		TranslatedPromptJSON: types.StringValue(promptJSON),
		TerraformIRJSON:      types.StringValue(irJSON),
		TerraformHCL:         types.StringValue(hcl),
		Fidelity:             types.StringValue(combined.Fidelity()),
		PreservedFields:      stringsListValue(combined.PreservedFields),
		SynthesizedFields:    stringsListValue(combined.SynthesizedFields),
		UnsupportedFields:    stringsListValue(combined.UnsupportedFields),
		Notes:                stringsListValue(combined.Notes),
	}, nil
}
