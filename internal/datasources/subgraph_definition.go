package datasources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/artifacts"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SubgraphDefinitionDataSource{}
var _ datasource.DataSourceWithConfigure = &SubgraphDefinitionDataSource{}

type SubgraphDefinitionDataSource struct {
	client *client.Client
}

type SubgraphDefinitionModel struct {
	ID              types.String `tfsdk:"id"`
	Source          types.String `tfsdk:"source"`
	Name            types.String `tfsdk:"name"`
	NodePack        types.String `tfsdk:"node_pack"`
	InfoJSON        types.String `tfsdk:"info_json"`
	DataJSON        types.String `tfsdk:"data_json"`
	NormalizedJSON  types.String `tfsdk:"normalized_json"`
	WorkspaceID     types.String `tfsdk:"workspace_id"`
	WorkspaceName   types.String `tfsdk:"workspace_name"`
	NodeCount       types.Int64  `tfsdk:"node_count"`
	GroupCount      types.Int64  `tfsdk:"group_count"`
	DefinitionCount types.Int64  `tfsdk:"definition_count"`
	DefinitionIDs   types.List   `tfsdk:"definition_ids"`
}

func NewSubgraphDefinitionDataSource() datasource.DataSource {
	return &SubgraphDefinitionDataSource{}
}

func (d *SubgraphDefinitionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subgraph_definition"
}

func (d *SubgraphDefinitionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SubgraphDefinitionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a single read-only editor-native subgraph definition from ComfyUI /global_subgraphs/{id}.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "SHA-derived subgraph identifier from comfyui_subgraph_catalog.",
			},
			"source": schema.StringAttribute{
				Computed:    true,
				Description: "Catalog source reported by ComfyUI.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Subgraph display name.",
			},
			"node_pack": schema.StringAttribute{
				Computed:    true,
				Description: "Node pack metadata reported inside info.node_pack.",
			},
			"info_json": schema.StringAttribute{
				Computed:    true,
				Description: "Raw JSON serialization of the entry info object.",
			},
			"data_json": schema.StringAttribute{
				Computed:    true,
				Description: "Raw editor-native JSON string returned inside the upstream data field.",
			},
			"normalized_json": schema.StringAttribute{
				Computed:    true,
				Description: "Normalized editor-native JSON parsed through the faithful workspace model.",
			},
			"workspace_id": schema.StringAttribute{
				Computed:    true,
				Description: "Top-level workspace identifier, when present.",
			},
			"workspace_name": schema.StringAttribute{
				Computed:    true,
				Description: "Top-level workspace name, when present.",
			},
			"node_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of top-level editor nodes in the subgraph JSON.",
			},
			"group_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of top-level editor groups in the subgraph JSON.",
			},
			"definition_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of definition subgraphs embedded in the JSON payload.",
			},
			"definition_ids": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Definition identifiers embedded in definitions.subgraphs.",
			},
		},
	}
}

func (d *SubgraphDefinitionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SubgraphDefinitionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The ComfyUI client is required to read /global_subgraphs/{id}.")
		return
	}

	entry, err := d.client.GetGlobalSubgraph(config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read subgraph definition", err.Error())
		return
	}
	if entry == nil {
		resp.Diagnostics.AddError("Subgraph definition not found", fmt.Sprintf("ComfyUI returned null for subgraph id %q.", config.ID.ValueString()))
		return
	}

	state, err := subgraphDefinitionStateFromEntry(config.ID.ValueString(), entry)
	if err != nil {
		resp.Diagnostics.AddError("Unable to convert subgraph definition", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func subgraphDefinitionStateFromEntry(id string, entry *client.GlobalSubgraphDefinition) (SubgraphDefinitionModel, error) {
	if entry == nil {
		return SubgraphDefinitionModel{}, fmt.Errorf("subgraph definition %q not found", id)
	}

	infoJSON, err := json.Marshal(entry.Info)
	if err != nil {
		return SubgraphDefinitionModel{}, fmt.Errorf("marshal info: %w", err)
	}

	workspace, err := artifacts.ParseWorkspaceJSON(entry.Data)
	if err != nil {
		return SubgraphDefinitionModel{}, err
	}
	normalizedJSON, err := workspace.JSON()
	if err != nil {
		return SubgraphDefinitionModel{}, err
	}

	return SubgraphDefinitionModel{
		ID:              types.StringValue(id),
		Source:          types.StringValue(entry.Source),
		Name:            types.StringValue(entry.Name),
		NodePack:        stringValueOrNull(entry.Info.NodePack),
		InfoJSON:        types.StringValue(string(infoJSON)),
		DataJSON:        types.StringValue(entry.Data),
		NormalizedJSON:  types.StringValue(normalizedJSON),
		WorkspaceID:     stringValueOrNull(workspace.ID),
		WorkspaceName:   stringValueOrNull(workspace.Name),
		NodeCount:       types.Int64Value(int64(len(workspace.Nodes))),
		GroupCount:      types.Int64Value(int64(len(workspace.Groups))),
		DefinitionCount: types.Int64Value(int64(len(workspace.Definitions.Subgraphs))),
		DefinitionIDs:   stringsListValue(definitionIDsFromWorkspace(workspace)),
	}, nil
}

func definitionIDsFromWorkspace(workspace *artifacts.Workspace) []string {
	ids := make([]string, 0, len(workspace.Definitions.Subgraphs))
	for _, definition := range workspace.Definitions.Subgraphs {
		if definition.ID != "" {
			ids = append(ids, definition.ID)
		}
	}
	return ids
}
