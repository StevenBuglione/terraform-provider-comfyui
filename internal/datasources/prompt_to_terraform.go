package datasources

import (
	"context"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/artifacts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &PromptToTerraformDataSource{}

type PromptToTerraformDataSource struct{}

type PromptToTerraformModel struct {
	Name              types.String `tfsdk:"name"`
	PromptJSON        types.String `tfsdk:"prompt_json"`
	TerraformIRJSON   types.String `tfsdk:"terraform_ir_json"`
	TerraformHCL      types.String `tfsdk:"terraform_hcl"`
	Fidelity          types.String `tfsdk:"fidelity"`
	PreservedFields   types.List   `tfsdk:"preserved_fields"`
	SynthesizedFields types.List   `tfsdk:"synthesized_fields"`
	UnsupportedFields types.List   `tfsdk:"unsupported_fields"`
	Notes             types.List   `tfsdk:"notes"`
}

func NewPromptToTerraformDataSource() datasource.DataSource {
	return &PromptToTerraformDataSource{}
}

func (d *PromptToTerraformDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt_to_terraform"
}

func (d *PromptToTerraformDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Synthesizes canonical Terraform IR and rendered HCL from ComfyUI prompt JSON using the generated node contract.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Optional logical workflow name used for future synthesis surfaces.",
			},
			"prompt_json": schema.StringAttribute{
				Required:    true,
				Description: "ComfyUI prompt JSON to synthesize into Terraform.",
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
				Description: "Overall synthesis fidelity classification.",
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

func (d *PromptToTerraformDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PromptToTerraformModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := promptToTerraformStateFromInput(stringValue(config.Name), stringValue(config.PromptJSON))
	if err != nil {
		resp.Diagnostics.AddError("Unable to synthesize Terraform from prompt JSON", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func promptToTerraformStateFromInput(name string, raw string) (PromptToTerraformModel, error) {
	prompt, err := artifacts.ParsePromptJSON(raw)
	if err != nil {
		return PromptToTerraformModel{}, err
	}
	ir, report, err := artifacts.BuildTerraformIRFromPrompt(prompt)
	if err != nil {
		return PromptToTerraformModel{}, err
	}
	irJSON, err := ir.JSON()
	if err != nil {
		return PromptToTerraformModel{}, err
	}
	hcl, err := artifacts.RenderTerraformHCL(ir)
	if err != nil {
		return PromptToTerraformModel{}, err
	}

	if name == "" {
		name = "workflow"
	}

	return PromptToTerraformModel{
		Name:              types.StringValue(name),
		PromptJSON:        types.StringValue(raw),
		TerraformIRJSON:   types.StringValue(irJSON),
		TerraformHCL:      types.StringValue(hcl),
		Fidelity:          types.StringValue(report.Fidelity()),
		PreservedFields:   stringsListValue(report.PreservedFields),
		SynthesizedFields: stringsListValue(report.SynthesizedFields),
		UnsupportedFields: stringsListValue(report.UnsupportedFields),
		Notes:             stringsListValue(report.Notes),
	}, nil
}
