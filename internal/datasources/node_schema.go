package datasources

import (
	"context"
	"fmt"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/resources"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &NodeSchemaDataSource{}

type NodeSchemaDataSource struct{}

type NodeSchemaModel struct {
	NodeType       types.String            `tfsdk:"node_type"`
	DisplayName    types.String            `tfsdk:"display_name"`
	Description    types.String            `tfsdk:"description"`
	Category       types.String            `tfsdk:"category"`
	OutputNode     types.Bool              `tfsdk:"output_node"`
	Deprecated     types.Bool              `tfsdk:"deprecated"`
	Experimental   types.Bool              `tfsdk:"experimental"`
	RequiredInputs []NodeSchemaInputModel  `tfsdk:"required_inputs"`
	OptionalInputs []NodeSchemaInputModel  `tfsdk:"optional_inputs"`
	HiddenInputs   []NodeSchemaHiddenModel `tfsdk:"hidden_inputs"`
	Outputs        []NodeSchemaOutputModel `tfsdk:"outputs"`
}

type NodeSchemaInputModel struct {
	Name                 types.String `tfsdk:"name"`
	Type                 types.String `tfsdk:"type"`
	IsLinkType           types.Bool   `tfsdk:"is_link_type"`
	DefaultValue         types.String `tfsdk:"default_value"`
	HasDefaultValue      types.Bool   `tfsdk:"has_default_value"`
	MinValue             types.String `tfsdk:"min_value"`
	HasMinValue          types.Bool   `tfsdk:"has_min_value"`
	MaxValue             types.String `tfsdk:"max_value"`
	HasMaxValue          types.Bool   `tfsdk:"has_max_value"`
	StepValue            types.String `tfsdk:"step_value"`
	HasStepValue         types.Bool   `tfsdk:"has_step_value"`
	EnumValues           types.List   `tfsdk:"enum_values"`
	Multiline            types.Bool   `tfsdk:"multiline"`
	DynamicOptions       types.Bool   `tfsdk:"dynamic_options"`
	DynamicOptionsSource types.String `tfsdk:"dynamic_options_source"`
	Tooltip              types.String `tfsdk:"tooltip"`
	DisplayName          types.String `tfsdk:"display_name"`
}

type NodeSchemaHiddenModel struct {
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

type NodeSchemaOutputModel struct {
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	SlotIndex types.Int64  `tfsdk:"slot_index"`
	IsList    types.Bool   `tfsdk:"is_list"`
}

func NewNodeSchemaDataSource() datasource.DataSource {
	return &NodeSchemaDataSource{}
}

func (d *NodeSchemaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node_schema"
}

func (d *NodeSchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the generated structured schema contract for a specific ComfyUI node type.",
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
			"required_inputs": nodeSchemaInputListAttribute("Structured required input contracts."),
			"optional_inputs": nodeSchemaInputListAttribute("Structured optional input contracts."),
			"hidden_inputs": schema.ListNestedAttribute{
				Description: "Structured hidden input contracts.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{Computed: true},
						"type": schema.StringAttribute{Computed: true},
					},
				},
			},
			"outputs": schema.ListNestedAttribute{
				Description: "Structured output slot contracts.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":       schema.StringAttribute{Computed: true},
						"type":       schema.StringAttribute{Computed: true},
						"slot_index": schema.Int64Attribute{Computed: true},
						"is_list":    schema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func nodeSchemaInputListAttribute(description string) schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Description: description,
		Computed:    true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"name":                   schema.StringAttribute{Computed: true},
				"type":                   schema.StringAttribute{Computed: true},
				"is_link_type":           schema.BoolAttribute{Computed: true},
				"default_value":          schema.StringAttribute{Computed: true},
				"has_default_value":      schema.BoolAttribute{Computed: true},
				"min_value":              schema.StringAttribute{Computed: true},
				"has_min_value":          schema.BoolAttribute{Computed: true},
				"max_value":              schema.StringAttribute{Computed: true},
				"has_max_value":          schema.BoolAttribute{Computed: true},
				"step_value":             schema.StringAttribute{Computed: true},
				"has_step_value":         schema.BoolAttribute{Computed: true},
				"enum_values":            schema.ListAttribute{Computed: true, ElementType: types.StringType},
				"multiline":              schema.BoolAttribute{Computed: true},
				"dynamic_options":        schema.BoolAttribute{Computed: true},
				"dynamic_options_source": schema.StringAttribute{Computed: true},
				"tooltip":                schema.StringAttribute{Computed: true},
				"display_name":           schema.StringAttribute{Computed: true},
			},
		},
	}
}

func (d *NodeSchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config NodeSchemaModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nodeType := config.NodeType.ValueString()
	generatedSchema, ok := lookupGeneratedNodeSchema(nodeType)
	if !ok {
		resp.Diagnostics.AddError(
			"Unknown Node Type",
			fmt.Sprintf("No generated node schema exists for %q.", nodeType),
		)
		return
	}

	state := NodeSchemaModel{
		NodeType:       types.StringValue(generatedSchema.NodeType),
		DisplayName:    types.StringValue(generatedSchema.DisplayName),
		Description:    types.StringValue(generatedSchema.Description),
		Category:       types.StringValue(generatedSchema.Category),
		OutputNode:     types.BoolValue(generatedSchema.OutputNode),
		Deprecated:     types.BoolValue(generatedSchema.Deprecated),
		Experimental:   types.BoolValue(generatedSchema.Experimental),
		RequiredInputs: buildNodeSchemaInputModels(ctx, generatedSchema.RequiredInputs, &resp.Diagnostics),
		OptionalInputs: buildNodeSchemaInputModels(ctx, generatedSchema.OptionalInputs, &resp.Diagnostics),
		HiddenInputs:   buildNodeSchemaHiddenModels(generatedSchema.HiddenInputs),
		Outputs:        buildNodeSchemaOutputModels(generatedSchema.Outputs),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func buildNodeSchemaInputModels(ctx context.Context, inputs []resources.GeneratedNodeSchemaInput, diags *diag.Diagnostics) []NodeSchemaInputModel {
	models := make([]NodeSchemaInputModel, 0, len(inputs))
	for _, input := range inputs {
		enumValues, enumDiags := types.ListValueFrom(ctx, types.StringType, input.EnumValues)
		diags.Append(enumDiags...)
		models = append(models, NodeSchemaInputModel{
			Name:                 types.StringValue(input.Name),
			Type:                 types.StringValue(input.Type),
			IsLinkType:           types.BoolValue(input.IsLinkType),
			DefaultValue:         types.StringValue(input.DefaultValue),
			HasDefaultValue:      types.BoolValue(input.HasDefaultValue),
			MinValue:             types.StringValue(input.MinValue),
			HasMinValue:          types.BoolValue(input.HasMinValue),
			MaxValue:             types.StringValue(input.MaxValue),
			HasMaxValue:          types.BoolValue(input.HasMaxValue),
			StepValue:            types.StringValue(input.StepValue),
			HasStepValue:         types.BoolValue(input.HasStepValue),
			EnumValues:           enumValues,
			Multiline:            types.BoolValue(input.Multiline),
			DynamicOptions:       types.BoolValue(input.DynamicOptions),
			DynamicOptionsSource: types.StringValue(input.DynamicOptionsSource),
			Tooltip:              types.StringValue(input.Tooltip),
			DisplayName:          types.StringValue(input.DisplayName),
		})
	}
	return models
}

func buildNodeSchemaHiddenModels(inputs []resources.GeneratedNodeSchemaHiddenInput) []NodeSchemaHiddenModel {
	models := make([]NodeSchemaHiddenModel, 0, len(inputs))
	for _, input := range inputs {
		models = append(models, NodeSchemaHiddenModel{
			Name: types.StringValue(input.Name),
			Type: types.StringValue(input.Type),
		})
	}
	return models
}

func buildNodeSchemaOutputModels(outputs []resources.GeneratedNodeSchemaOutput) []NodeSchemaOutputModel {
	models := make([]NodeSchemaOutputModel, 0, len(outputs))
	for _, output := range outputs {
		models = append(models, NodeSchemaOutputModel{
			Name:      types.StringValue(output.Name),
			Type:      types.StringValue(output.Type),
			SlotIndex: types.Int64Value(int64(output.SlotIndex)),
			IsList:    types.BoolValue(output.IsList),
		})
	}
	return models
}

func lookupGeneratedNodeSchema(nodeType string) (resources.GeneratedNodeSchema, bool) {
	return resources.LookupGeneratedNodeSchema(nodeType)
}
