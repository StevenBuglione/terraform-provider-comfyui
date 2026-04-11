package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SubgraphCatalogDataSource{}
var _ datasource.DataSourceWithConfigure = &SubgraphCatalogDataSource{}

type SubgraphCatalogDataSource struct {
	client *client.Client
}

type SubgraphCatalogModel struct {
	EntryCount types.Int64 `tfsdk:"entry_count"`
	Entries    types.List  `tfsdk:"entries"`
}

type subgraphCatalogEntryModel struct {
	ID       types.String `tfsdk:"id"`
	Source   types.String `tfsdk:"source"`
	Name     types.String `tfsdk:"name"`
	NodePack types.String `tfsdk:"node_pack"`
	InfoJSON types.String `tfsdk:"info_json"`
}

func NewSubgraphCatalogDataSource() datasource.DataSource {
	return &SubgraphCatalogDataSource{}
}

func (d *SubgraphCatalogDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subgraph_catalog"
}

func (d *SubgraphCatalogDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SubgraphCatalogDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists read-only editor-native subgraph catalog entries exposed by ComfyUI /global_subgraphs.",
		Attributes: map[string]schema.Attribute{
			"entry_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of subgraph catalog entries returned by /global_subgraphs.",
			},
			"entries": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Catalog entries keyed by ComfyUI's SHA-derived subgraph identifier.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "SHA-derived subgraph catalog identifier.",
						},
						"source": schema.StringAttribute{
							Computed:    true,
							Description: "Catalog source reported by ComfyUI, such as templates or custom_node.",
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
							Description: "Raw JSON serialization of the entry info object for forward-compatible metadata preservation.",
						},
					},
				},
			},
		},
	}
}

func (d *SubgraphCatalogDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The ComfyUI client is required to read /global_subgraphs.")
		return
	}

	entries, err := d.client.GetGlobalSubgraphs()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read subgraph catalog", err.Error())
		return
	}

	state, err := subgraphCatalogStateFromEntries(entries)
	if err != nil {
		resp.Diagnostics.AddError("Unable to convert subgraph catalog", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func subgraphCatalogStateFromEntries(entries map[string]client.GlobalSubgraphCatalogEntry) (SubgraphCatalogModel, error) {
	ids := make([]string, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	models := make([]subgraphCatalogEntryModel, 0, len(ids))
	for _, id := range ids {
		entry := entries[id]
		infoJSON, err := json.Marshal(entry.Info)
		if err != nil {
			return SubgraphCatalogModel{}, fmt.Errorf("marshal info for %s: %w", id, err)
		}
		models = append(models, subgraphCatalogEntryModel{
			ID:       types.StringValue(id),
			Source:   types.StringValue(entry.Source),
			Name:     types.StringValue(entry.Name),
			NodePack: stringValueOrNull(entry.Info.NodePack),
			InfoJSON: types.StringValue(string(infoJSON)),
		})
	}

	entriesList, diags := types.ListValueFrom(context.Background(), subgraphCatalogEntryObjectType(), models)
	if diags.HasError() {
		return SubgraphCatalogModel{}, fmt.Errorf("build entries list: %s", diags.Errors()[0].Summary())
	}

	return SubgraphCatalogModel{
		EntryCount: types.Int64Value(int64(len(models))),
		Entries:    entriesList,
	}, nil
}

func subgraphCatalogEntryObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":        types.StringType,
			"source":    types.StringType,
			"name":      types.StringType,
			"node_pack": types.StringType,
			"info_json": types.StringType,
		},
	}
}
