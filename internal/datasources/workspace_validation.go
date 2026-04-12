package datasources

import (
	"context"
	"fmt"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/artifacts"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/inventory"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/validation"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &WorkspaceValidationDataSource{}
var _ datasource.DataSourceWithConfigure = &WorkspaceValidationDataSource{}

type WorkspaceValidationDataSource struct {
	client *client.Client
}

type WorkspaceValidationModel struct {
	Path                         types.String `tfsdk:"path"`
	JSON                         types.String `tfsdk:"json"`
	Mode                         types.String `tfsdk:"mode"`
	Valid                        types.Bool   `tfsdk:"valid"`
	ErrorCount                   types.Int64  `tfsdk:"error_count"`
	WarningCount                 types.Int64  `tfsdk:"warning_count"`
	Errors                       types.List   `tfsdk:"errors"`
	Warnings                     types.List   `tfsdk:"warnings"`
	ValidatedNodeCount           types.Int64  `tfsdk:"validated_node_count"`
	NormalizedJSON               types.String `tfsdk:"normalized_json"`
	TranslatedPromptJSON         types.String `tfsdk:"translated_prompt_json"`
	TranslationFidelity          types.String `tfsdk:"translation_fidelity"`
	TranslationPreservedFields   types.List   `tfsdk:"translation_preserved_fields"`
	TranslationSynthesizedFields types.List   `tfsdk:"translation_synthesized_fields"`
	TranslationUnsupportedFields types.List   `tfsdk:"translation_unsupported_fields"`
	TranslationNotes             types.List   `tfsdk:"translation_notes"`
}

func NewWorkspaceValidationDataSource() datasource.DataSource {
	return &WorkspaceValidationDataSource{}
}

func (d *WorkspaceValidationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_validation"
}

func (d *WorkspaceValidationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *WorkspaceValidationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Validates native ComfyUI workspace JSON by translating it to prompt JSON and checking it against live /object_info metadata.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Optional:    true,
				Description: "Optional file path to load workspace JSON from.",
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(path.MatchRelative().AtParent().AtName("json")),
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("json")),
				},
			},
			"json": schema.StringAttribute{
				Optional:    true,
				Description: "Optional raw ComfyUI workspace JSON string.",
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(path.MatchRelative().AtParent().AtName("path")),
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("path")),
				},
			},
			"mode": schema.StringAttribute{
				Optional:    true,
				Description: "Validation mode. Defaults to executable_workspace. Use workspace_fragment to validate incomplete workspace fragments without requiring an output node after translation.",
				Validators: []validator.String{
					stringvalidator.OneOf(string(validationModeWorkspaceFragment), string(validationModeExecutableWorkspace)),
				},
			},
			"valid": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the translated prompt passed semantic validation.",
			},
			"error_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of validation errors.",
			},
			"warning_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of validation warnings.",
			},
			"errors": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Validation errors reported as plain strings.",
			},
			"warnings": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Validation warnings reported as plain strings.",
			},
			"validated_node_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of translated prompt nodes inspected during validation.",
			},
			"normalized_json": schema.StringAttribute{
				Computed:    true,
				Description: "Normalized ComfyUI workspace JSON.",
			},
			"translated_prompt_json": schema.StringAttribute{
				Computed:    true,
				Description: "Prompt JSON translated from the source workspace.",
			},
			"translation_fidelity": schema.StringAttribute{
				Computed:    true,
				Description: "Overall translation fidelity classification.",
			},
			"translation_preserved_fields": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Field paths preserved exactly during translation.",
			},
			"translation_synthesized_fields": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Field paths synthesized during translation.",
			},
			"translation_unsupported_fields": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Field paths dropped or not representable during translation.",
			},
			"translation_notes": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Human-readable translation notes.",
			},
		},
	}
}

func (d *WorkspaceValidationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WorkspaceValidationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The ComfyUI client is required to fetch object_info for semantic validation.")
		return
	}

	nodeInfo, err := d.client.GetObjectInfo()
	if err != nil {
		resp.Diagnostics.AddError("Unable to fetch object info", err.Error())
		return
	}

	mode, err := parseWorkspaceValidationMode(config.Mode)
	if err != nil {
		resp.Diagnostics.AddError("Invalid validation mode", err.Error())
		return
	}

	state, err := workspaceValidationStateFromInput(stringValue(config.Path), stringValue(config.JSON), nodeInfo, inventory.NewService(d.client), mode)
	if err != nil {
		resp.Diagnostics.AddError("Unable to validate workspace JSON", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func workspaceValidationStateFromInput(path string, raw string, nodeInfo map[string]client.NodeInfo, inventoryService validation.InventoryService, mode validationMode) (WorkspaceValidationModel, error) {
	rawJSON, err := loadJSONInput(path, raw)
	if err != nil {
		return WorkspaceValidationModel{}, err
	}

	workspace, err := artifacts.ParseWorkspaceJSON(rawJSON)
	if err != nil {
		return WorkspaceValidationModel{}, err
	}

	normalizedJSON, err := workspace.JSON()
	if err != nil {
		return WorkspaceValidationModel{}, err
	}

	prompt, report, err := artifacts.TranslateWorkspaceToPrompt(workspace)
	if err != nil {
		return WorkspaceValidationModel{}, err
	}

	promptJSON, err := prompt.JSON()
	if err != nil {
		return WorkspaceValidationModel{}, err
	}

	validationReport := validation.ValidatePrompt(prompt, nodeInfo, validation.Options{Mode: mode.toValidationMode(), InventoryService: inventoryService})
	return WorkspaceValidationModel{
		Path:                         stringValueOrNull(path),
		JSON:                         stringValueOrNull(raw),
		Mode:                         types.StringValue(string(mode)),
		Valid:                        types.BoolValue(validationReport.Valid),
		ErrorCount:                   types.Int64Value(int64(validationReport.ErrorCount)),
		WarningCount:                 types.Int64Value(int64(validationReport.WarningCount)),
		Errors:                       stringsListValue(validationReport.Errors),
		Warnings:                     stringsListValue(validationReport.Warnings),
		ValidatedNodeCount:           types.Int64Value(int64(validationReport.ValidatedNodeCount)),
		NormalizedJSON:               types.StringValue(normalizedJSON),
		TranslatedPromptJSON:         types.StringValue(promptJSON),
		TranslationFidelity:          types.StringValue(report.Fidelity()),
		TranslationPreservedFields:   stringsListValue(report.PreservedFields),
		TranslationSynthesizedFields: stringsListValue(report.SynthesizedFields),
		TranslationUnsupportedFields: stringsListValue(report.UnsupportedFields),
		TranslationNotes:             stringsListValue(report.Notes),
	}, nil
}
