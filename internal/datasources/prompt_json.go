package datasources

import (
	"context"
	"fmt"
	"os"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/artifacts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &PromptJSONDataSource{}

type PromptJSONDataSource struct{}

type PromptJSONModel struct {
	Path           types.String `tfsdk:"path"`
	JSON           types.String `tfsdk:"json"`
	NormalizedJSON types.String `tfsdk:"normalized_json"`
	NodeCount      types.Int64  `tfsdk:"node_count"`
}

func NewPromptJSONDataSource() datasource.DataSource {
	return &PromptJSONDataSource{}
}

func (d *PromptJSONDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prompt_json"
}

func (d *PromptJSONDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Imports and normalizes native ComfyUI prompt JSON from a file path or raw JSON string.",
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
			"normalized_json": schema.StringAttribute{
				Computed:    true,
				Description: "Normalized ComfyUI prompt JSON.",
			},
			"node_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of nodes in the imported prompt.",
			},
		},
	}
}

func (d *PromptJSONDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PromptJSONModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := promptJSONStateFromInput(stringValue(config.Path), stringValue(config.JSON))
	if err != nil {
		resp.Diagnostics.AddError("Unable to import prompt JSON", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func promptJSONStateFromInput(path string, raw string) (PromptJSONModel, error) {
	rawJSON, err := loadJSONInput(path, raw)
	if err != nil {
		return PromptJSONModel{}, err
	}

	prompt, err := artifacts.ParsePromptJSON(rawJSON)
	if err != nil {
		return PromptJSONModel{}, err
	}

	normalizedJSON, err := prompt.JSON()
	if err != nil {
		return PromptJSONModel{}, err
	}

	return PromptJSONModel{
		Path:           stringValueOrNull(path),
		JSON:           stringValueOrNull(raw),
		NormalizedJSON: types.StringValue(normalizedJSON),
		NodeCount:      types.Int64Value(int64(len(prompt.Nodes))),
	}, nil
}

func loadJSONInput(path string, raw string) (string, error) {
	if path != "" && raw != "" {
		return "", fmt.Errorf("path and json are mutually exclusive")
	}
	if raw != "" {
		return raw, nil
	}
	if path == "" {
		return "", fmt.Errorf("either path or json must be provided")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	return string(data), nil
}

func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func stringValue(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func stringsListValue(values []string) types.List {
	list, diags := types.ListValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		return types.ListNull(types.StringType)
	}
	return list
}
