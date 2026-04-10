package datasources

import (
	"context"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/artifacts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &WorkspaceJSONDataSource{}

type WorkspaceJSONDataSource struct{}

type WorkspaceJSONModel struct {
	Path           types.String `tfsdk:"path"`
	JSON           types.String `tfsdk:"json"`
	NormalizedJSON types.String `tfsdk:"normalized_json"`
	NodeCount      types.Int64  `tfsdk:"node_count"`
	GroupCount     types.Int64  `tfsdk:"group_count"`
	SubgraphCount  types.Int64  `tfsdk:"subgraph_count"`
}

func NewWorkspaceJSONDataSource() datasource.DataSource {
	return &WorkspaceJSONDataSource{}
}

func (d *WorkspaceJSONDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_json"
}

func (d *WorkspaceJSONDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Imports and normalizes native ComfyUI workspace or subgraph JSON from a file path or raw JSON string.",
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
				Description: "Optional raw ComfyUI workspace or subgraph JSON string.",
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(path.MatchRelative().AtParent().AtName("path")),
					stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("path")),
				},
			},
			"normalized_json": schema.StringAttribute{
				Computed:    true,
				Description: "Normalized ComfyUI workspace or subgraph JSON.",
			},
			"node_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of nodes in the imported workspace.",
			},
			"group_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of groups in the imported workspace.",
			},
			"subgraph_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of subgraph definitions in the imported workspace.",
			},
		},
	}
}

func (d *WorkspaceJSONDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WorkspaceJSONModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := workspaceJSONStateFromInput(stringValue(config.Path), stringValue(config.JSON))
	if err != nil {
		resp.Diagnostics.AddError("Unable to import workspace JSON", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func workspaceJSONStateFromInput(path string, raw string) (WorkspaceJSONModel, error) {
	rawJSON, err := loadJSONInput(path, raw)
	if err != nil {
		return WorkspaceJSONModel{}, err
	}

	workspace, err := artifacts.ParseWorkspaceJSON(rawJSON)
	if err != nil {
		return WorkspaceJSONModel{}, err
	}

	normalizedJSON, err := workspace.JSON()
	if err != nil {
		return WorkspaceJSONModel{}, err
	}

	return WorkspaceJSONModel{
		Path:           stringValueOrNull(path),
		JSON:           stringValueOrNull(raw),
		NormalizedJSON: types.StringValue(normalizedJSON),
		NodeCount:      types.Int64Value(int64(len(workspace.Nodes))),
		GroupCount:     types.Int64Value(int64(len(workspace.Groups))),
		SubgraphCount:  types.Int64Value(int64(len(workspace.Definitions.Subgraphs))),
	}, nil
}
