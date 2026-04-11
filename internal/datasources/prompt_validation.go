package datasources

import (
	"context"
	"fmt"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/artifacts"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/validation"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &PromptValidationDataSource{}
var _ datasource.DataSourceWithConfigure = &PromptValidationDataSource{}

type PromptValidationDataSource struct {
	client *client.Client
}

type PromptValidationModel struct {
	Path               types.String `tfsdk:"path"`
	JSON               types.String `tfsdk:"json"`
	Valid              types.Bool   `tfsdk:"valid"`
	ErrorCount         types.Int64  `tfsdk:"error_count"`
	WarningCount       types.Int64  `tfsdk:"warning_count"`
	Errors             types.List   `tfsdk:"errors"`
	Warnings           types.List   `tfsdk:"warnings"`
	ValidatedNodeCount types.Int64  `tfsdk:"validated_node_count"`
	NormalizedJSON     types.String `tfsdk:"normalized_json"`
}

func NewPromptValidationDataSource() datasource.DataSource {
	return &PromptValidationDataSource{}
}

func (d *PromptValidationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt_validation"
}

func (d *PromptValidationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PromptValidationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Validates native ComfyUI prompt JSON against live /object_info metadata.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Optional:    true,
				Description: "Optional file path to load prompt JSON from.",
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(path.MatchRelative().AtParent().AtName("json")),
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("json")),
				},
			},
			"json": schema.StringAttribute{
				Optional:    true,
				Description: "Optional raw ComfyUI prompt JSON string.",
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(path.MatchRelative().AtParent().AtName("path")),
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("path")),
				},
			},
			"valid": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the prompt passed semantic validation.",
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
				Description: "Number of prompt nodes inspected during validation.",
			},
			"normalized_json": schema.StringAttribute{
				Computed:    true,
				Description: "Normalized ComfyUI prompt JSON.",
			},
		},
	}
}

func (d *PromptValidationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PromptValidationModel
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

	state, err := promptValidationStateFromInput(stringValue(config.Path), stringValue(config.JSON), nodeInfo)
	if err != nil {
		resp.Diagnostics.AddError("Unable to validate prompt JSON", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func promptValidationStateFromInput(path string, raw string, nodeInfo map[string]client.NodeInfo) (PromptValidationModel, error) {
	rawJSON, err := loadJSONInput(path, raw)
	if err != nil {
		return PromptValidationModel{}, err
	}

	prompt, err := artifacts.ParsePromptJSON(rawJSON)
	if err != nil {
		return PromptValidationModel{}, err
	}

	normalizedJSON, err := prompt.JSON()
	if err != nil {
		return PromptValidationModel{}, err
	}

	report := validation.ValidatePrompt(prompt, nodeInfo, validation.Options{RequireOutputNode: true})
	return PromptValidationModel{
		Path:               stringValueOrNull(path),
		JSON:               stringValueOrNull(raw),
		Valid:              types.BoolValue(report.Valid),
		ErrorCount:         types.Int64Value(int64(report.ErrorCount)),
		WarningCount:       types.Int64Value(int64(report.WarningCount)),
		Errors:             stringsListValue(report.Errors),
		Warnings:           stringsListValue(report.Warnings),
		ValidatedNodeCount: types.Int64Value(int64(report.ValidatedNodeCount)),
		NormalizedJSON:     types.StringValue(normalizedJSON),
	}, nil
}
