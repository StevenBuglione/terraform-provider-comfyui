package datasources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &NodeInfoDataSource{}
var _ datasource.DataSourceWithConfigure = &NodeInfoDataSource{}

type NodeInfoDataSource struct {
	client *client.Client
}

type NodeInfoModel struct {
	NodeType      types.String `tfsdk:"node_type"`
	DisplayName   types.String `tfsdk:"display_name"`
	Description   types.String `tfsdk:"description"`
	Category      types.String `tfsdk:"category"`
	OutputNode    types.Bool   `tfsdk:"output_node"`
	Deprecated    types.Bool   `tfsdk:"deprecated"`
	Experimental  types.Bool   `tfsdk:"experimental"`
	InputRequired types.String `tfsdk:"input_required"`
	InputOptional types.String `tfsdk:"input_optional"`
	OutputTypes   types.List   `tfsdk:"output_types"`
	OutputNames   types.List   `tfsdk:"output_names"`
}

func NewNodeInfoDataSource() datasource.DataSource {
	return &NodeInfoDataSource{}
}

func (d *NodeInfoDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node_info"
}

func (d *NodeInfoDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NodeInfoDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a specific ComfyUI node type.",
		Attributes: map[string]schema.Attribute{
			"node_type": schema.StringAttribute{
				Description: "The ComfyUI node type to look up (e.g., KSampler).",
				Required:    true,
			},
			"display_name": schema.StringAttribute{
				Description: "Display name of the node.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the node.",
				Computed:    true,
			},
			"category": schema.StringAttribute{
				Description: "Category the node belongs to.",
				Computed:    true,
			},
			"output_node": schema.BoolAttribute{
				Description: "Whether this is an output node.",
				Computed:    true,
			},
			"deprecated": schema.BoolAttribute{
				Description: "Whether this node is deprecated.",
				Computed:    true,
			},
			"experimental": schema.BoolAttribute{
				Description: "Whether this node is experimental.",
				Computed:    true,
			},
			"input_required": schema.StringAttribute{
				Description: "JSON representation of required inputs.",
				Computed:    true,
			},
			"input_optional": schema.StringAttribute{
				Description: "JSON representation of optional inputs.",
				Computed:    true,
			},
			"output_types": schema.ListAttribute{
				Description: "List of output types.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"output_names": schema.ListAttribute{
				Description: "List of output names.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *NodeInfoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config NodeInfoModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nodeType := config.NodeType.ValueString()

	info, err := d.client.GetObjectInfoSingle(nodeType)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read node info",
			fmt.Sprintf("Could not read node info for %q: %s", nodeType, err.Error()),
		)
		return
	}

	requiredJSON, err := json.Marshal(info.Input.Required)
	if err != nil {
		resp.Diagnostics.AddError("Unable to marshal required inputs", err.Error())
		return
	}

	optionalJSON, err := json.Marshal(info.Input.Optional)
	if err != nil {
		resp.Diagnostics.AddError("Unable to marshal optional inputs", err.Error())
		return
	}

	outputTypes, diags := types.ListValueFrom(ctx, types.StringType, info.Output)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	outputNames, diags := types.ListValueFrom(ctx, types.StringType, info.OutputName)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := NodeInfoModel{
		NodeType:      types.StringValue(nodeType),
		DisplayName:   types.StringValue(info.DisplayName),
		Description:   types.StringValue(info.Description),
		Category:      types.StringValue(info.Category),
		OutputNode:    types.BoolValue(info.OutputNode),
		Deprecated:    types.BoolValue(info.Deprecated),
		Experimental:  types.BoolValue(info.Experimental),
		InputRequired: types.StringValue(string(requiredJSON)),
		InputOptional: types.StringValue(string(optionalJSON)),
		OutputTypes:   outputTypes,
		OutputNames:   outputNames,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
