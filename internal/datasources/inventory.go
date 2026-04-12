package datasources

import (
	"context"
	"fmt"
	"sort"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/inventory"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/validation"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &InventoryDataSource{}
var _ datasource.DataSourceWithConfigure = &InventoryDataSource{}

type InventoryDataSource struct {
	client *client.Client
}

type InventoryModel struct {
	ID          types.String         `tfsdk:"id"`
	Kinds       types.List           `tfsdk:"kinds"`
	Inventories []InventoryKindModel `tfsdk:"inventories"`
}

type InventoryKindModel struct {
	Kind       types.String `tfsdk:"kind"`
	Values     types.List   `tfsdk:"values"`
	ValueCount types.Int64  `tfsdk:"value_count"`
}

func NewInventoryDataSource() datasource.DataSource {
	return &InventoryDataSource{}
}

func (d *InventoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inventory"
}

func (d *InventoryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InventoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns live ComfyUI runtime inventory values for built-in dynamic model categories such as checkpoints and LoRAs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this inventory query.",
				Computed:    true,
			},
			"kinds": schema.ListAttribute{
				Description: "Optional subset of inventory kinds to query. When omitted, all generated built-in kinds are queried.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"inventories": schema.ListNestedAttribute{
				Description: "Inventory values discovered from the live ComfyUI server, grouped by normalized kind.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"kind": schema.StringAttribute{
							Description: "Normalized inventory kind such as checkpoints or loras.",
							Computed:    true,
						},
						"values": schema.ListAttribute{
							Description: "Sorted live values available for this inventory kind.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"value_count": schema.Int64Attribute{
							Description: "Number of live values returned for this inventory kind.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *InventoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config InventoryModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The ComfyUI client is required to query live inventory.")
		return
	}

	kinds, diags := inventoryKindsFromConfig(ctx, config.Kinds)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := inventoryStateFromKinds(ctx, inventory.NewService(d.client), kinds)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Kinds = config.Kinds
	if state.Kinds.IsNull() {
		elements := make([]types.String, 0, len(kinds))
		for _, kind := range kinds {
			elements = append(elements, types.StringValue(string(kind)))
		}
		state.Kinds, _ = types.ListValueFrom(ctx, types.StringType, elements)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func inventoryKindsFromConfig(ctx context.Context, kinds types.List) ([]inventory.Kind, diag.Diagnostics) {
	var diags diag.Diagnostics
	if kinds.IsNull() || kinds.IsUnknown() {
		return inventory.AllKinds(), diags
	}

	var raw []string
	diags.Append(kinds.ElementsAs(ctx, &raw, false)...)
	if diags.HasError() {
		return nil, diags
	}

	seen := map[inventory.Kind]struct{}{}
	result := make([]inventory.Kind, 0, len(raw))
	for _, value := range raw {
		kind, ok := inventory.ParseKind(value)
		if !ok {
			diags.AddError("Unsupported Inventory Kind", fmt.Sprintf("Inventory kind %q is not supported. Supported kinds: %v", value, inventory.AllKinds()))
			continue
		}
		if _, exists := seen[kind]; exists {
			continue
		}
		seen[kind] = struct{}{}
		result = append(result, kind)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, diags
}

func inventoryStateFromKinds(ctx context.Context, service validation.InventoryService, kinds []inventory.Kind) (InventoryModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	state := InventoryModel{
		ID:          types.StringValue("inventory"),
		Inventories: make([]InventoryKindModel, 0, len(kinds)),
	}

	for _, kind := range kinds {
		values, err := service.GetInventory(ctx, kind)
		if err != nil {
			diags.AddError("Unable to Query Live Inventory", fmt.Sprintf("Failed to query live %s inventory: %s", kind, err.Error()))
			continue
		}

		sort.Strings(values)
		listValue, listDiags := types.ListValueFrom(ctx, types.StringType, values)
		diags.Append(listDiags...)
		if diags.HasError() {
			continue
		}

		state.Inventories = append(state.Inventories, InventoryKindModel{
			Kind:       types.StringValue(string(kind)),
			Values:     listValue,
			ValueCount: types.Int64Value(int64(len(values))),
		})
	}

	return state, diags
}
