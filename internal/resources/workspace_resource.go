package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = &WorkspaceResource{}
	_ resource.ResourceWithConfigure = &WorkspaceResource{}
)

type WorkspaceResource struct {
	client *client.Client
}

type workspaceResourceModel struct {
	ID            types.String              `tfsdk:"id"`
	Name          types.String              `tfsdk:"name"`
	Workflows     []workspaceWorkflowModel  `tfsdk:"workflows"`
	Layout        workspaceLayoutModel      `tfsdk:"layout"`
	NodeLayout    *workspaceNodeLayoutModel `tfsdk:"node_layout"`
	OutputFile    types.String              `tfsdk:"output_file"`
	WorkspaceJSON types.String              `tfsdk:"workspace_json"`
	WorkflowCount types.Int64               `tfsdk:"workflow_count"`
}

type workspaceWorkflowModel struct {
	Name         types.String                 `tfsdk:"name"`
	WorkflowJSON types.String                 `tfsdk:"workflow_json"`
	X            types.Float64                `tfsdk:"x"`
	Y            types.Float64                `tfsdk:"y"`
	Style        *workspaceWorkflowStyleModel `tfsdk:"style"`
}

type workspaceLayoutModel struct {
	Display   types.String  `tfsdk:"display"`
	Direction types.String  `tfsdk:"direction"`
	Gap       types.Float64 `tfsdk:"gap"`
	Columns   types.Int64   `tfsdk:"columns"`
	OriginX   types.Float64 `tfsdk:"origin_x"`
	OriginY   types.Float64 `tfsdk:"origin_y"`
}

type workspaceWorkflowStyleModel struct {
	GroupColor    types.String `tfsdk:"group_color"`
	TitleFontSize types.Int64  `tfsdk:"title_font_size"`
}

type workspaceNodeLayoutModel struct {
	Mode      types.String  `tfsdk:"mode"`
	Direction types.String  `tfsdk:"direction"`
	ColumnGap types.Float64 `tfsdk:"column_gap"`
	RowGap    types.Float64 `tfsdk:"row_gap"`
}

type workspaceLayoutConfig struct {
	Display   string
	Direction string
	Gap       float64
	Columns   int64
	OriginX   float64
	OriginY   float64
}

type workspaceWorkflowStyleConfig struct {
	GroupColor    string
	TitleFontSize int
}

type workspaceNodeLayoutConfig struct {
	Mode      string
	Direction string
	ColumnGap float64
	RowGap    float64
}

func NewWorkspaceResource() resource.Resource {
	return &WorkspaceResource{}
}

func (r *WorkspaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

func (r *WorkspaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Composes multiple ComfyUI workflows into a single UI-oriented workspace export with deterministic layout.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Unique identifier for this workspace resource.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable name for the exported workspace.",
			},
			"workflows": schema.ListNestedAttribute{
				Required:    true,
				Description: "Ordered workflows to place on the shared canvas.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Display name for this workflow island.",
						},
						"workflow_json": schema.StringAttribute{
							Required:    true,
							Description: "ComfyUI API-format workflow JSON, typically from comfyui_workflow.<name>.assembled_json.",
						},
						"x": schema.Float64Attribute{
							Optional:    true,
							Description: "Optional absolute X position override for this workflow island.",
						},
						"y": schema.Float64Attribute{
							Optional:    true,
							Description: "Optional absolute Y position override for this workflow island.",
						},
						"style": schema.SingleNestedAttribute{
							Optional:    true,
							Description: "Optional visual styling for this workflow island.",
							Attributes: map[string]schema.Attribute{
								"group_color": schema.StringAttribute{
									Optional:    true,
									Description: "Optional group color used when rendering the workflow island.",
								},
								"title_font_size": schema.Int64Attribute{
									Optional:    true,
									Description: "Optional title font size used when rendering the workflow island.",
								},
							},
						},
					},
				},
			},
			"layout": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Typed, CSS-inspired layout controls used to position workflow islands on the canvas.",
				Attributes: map[string]schema.Attribute{
					"display": schema.StringAttribute{
						Required:    true,
						Description: "Layout engine to use. Supported values are flex and grid.",
					},
					"direction": schema.StringAttribute{
						Optional:    true,
						Description: "Primary flow direction for flex layouts.",
					},
					"gap": schema.Float64Attribute{
						Optional:    true,
						Description: "Default gap between workflow islands on both axes.",
					},
					"columns": schema.Int64Attribute{
						Optional:    true,
						Description: "Number of grid columns when display is grid.",
					},
					"origin_x": schema.Float64Attribute{
						Optional:    true,
						Description: "Canvas origin offset on the X axis.",
					},
					"origin_y": schema.Float64Attribute{
						Optional:    true,
						Description: "Canvas origin offset on the Y axis.",
					},
				},
			},
			"node_layout": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Node readability controls applied to workflow nodes before builder layout is added.",
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						Optional:    true,
						Description: "Node layout mode. v1 supports dag only.",
					},
					"direction": schema.StringAttribute{
						Optional:    true,
						Description: "Node layout direction. v1 supports left_to_right only.",
					},
					"column_gap": schema.Float64Attribute{
						Optional:    true,
						Description: "Horizontal gap between nodes.",
					},
					"row_gap": schema.Float64Attribute{
						Optional:    true,
						Description: "Vertical gap between nodes.",
					},
				},
			},
			"output_file": schema.StringAttribute{
				Optional:    true,
				Description: "Optional file path to write the composed workspace JSON.",
			},
			"workspace_json": schema.StringAttribute{
				Computed:    true,
				Description: "The composed ComfyUI workspace JSON.",
			},
			"workflow_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of workflows included in the workspace.",
			},
		},
	}
}

func (r *WorkspaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *WorkspaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data workspaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(uuid.New().String())
	r.buildWorkspaceState(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data workspaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkspaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data workspaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state workspaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.ID.IsNull() || data.ID.IsUnknown() || data.ID.ValueString() == "" {
		data.ID = state.ID
	}

	r.buildWorkspaceState(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := cleanupPreviousWorkspaceFile(stringValue(state.OutputFile), stringValue(data.OutputFile)); err != nil {
		resp.Diagnostics.AddError("Failed to Clean Up Previous Workspace File", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, _ *resource.DeleteResponse) {
	var data workspaceResourceModel
	req.State.Get(ctx, &data)

	if !data.OutputFile.IsNull() && !data.OutputFile.IsUnknown() && data.OutputFile.ValueString() != "" {
		_ = os.Remove(data.OutputFile.ValueString())
		tflog.Info(ctx, "Removed workspace file", map[string]interface{}{"path": data.OutputFile.ValueString()})
	}
}

func (r *WorkspaceResource) buildWorkspaceState(ctx context.Context, data *workspaceResourceModel, diags *diag.Diagnostics) {
	if r.client == nil {
		diags.AddError("Client Not Configured", "The ComfyUI client is required to fetch node metadata for workspace export.")
		return
	}

	name, specs, layout, nodeLayout, err := workspaceConfigFromModel(*data)
	if err != nil {
		diags.AddError("Invalid Workspace Configuration", err.Error())
		return
	}

	objectInfo, err := r.client.GetObjectInfo()
	if err != nil {
		diags.AddError("Failed to Load Node Metadata", err.Error())
		return
	}

	subgraph, err := buildWorkspaceSubgraph(name, specs, layout, nodeLayout, objectInfo)
	if err != nil {
		diags.AddError("Failed to Build Workspace", err.Error())
		return
	}

	jsonBytes, err := json.MarshalIndent(subgraph, "", "  ")
	if err != nil {
		diags.AddError("Failed to Marshal Workspace", err.Error())
		return
	}

	data.WorkspaceJSON = types.StringValue(string(jsonBytes))
	data.WorkflowCount = types.Int64Value(int64(len(specs)))

	if !data.OutputFile.IsNull() && !data.OutputFile.IsUnknown() && data.OutputFile.ValueString() != "" {
		if err := writeWorkspaceFile(data.OutputFile.ValueString(), jsonBytes); err != nil {
			diags.AddError("Failed to Write Workspace File", err.Error())
			return
		}
		tflog.Info(ctx, "Wrote workspace JSON file", map[string]interface{}{"path": data.OutputFile.ValueString()})
	}
}

func workspaceConfigFromModel(model workspaceResourceModel) (string, []workspaceWorkflowSpec, workspaceLayoutConfig, workspaceNodeLayoutConfig, error) {
	if model.Name.IsNull() || model.Name.IsUnknown() || model.Name.ValueString() == "" {
		return "", nil, workspaceLayoutConfig{}, workspaceNodeLayoutConfig{}, fmt.Errorf("name must be provided")
	}

	nodeLayout, err := workspaceNodeLayoutConfigFromModel(model.NodeLayout)
	if err != nil {
		return "", nil, workspaceLayoutConfig{}, workspaceNodeLayoutConfig{}, err
	}

	layout := workspaceLayoutConfig{
		Display:   stringValue(model.Layout.Display),
		Direction: stringValue(model.Layout.Direction),
		Gap:       float64Value(model.Layout.Gap),
		Columns:   int64Value(model.Layout.Columns),
		OriginX:   float64Value(model.Layout.OriginX),
		OriginY:   float64Value(model.Layout.OriginY),
	}
	if err := validateWorkspaceLayout(layout); err != nil {
		return "", nil, workspaceLayoutConfig{}, workspaceNodeLayoutConfig{}, err
	}

	specs := make([]workspaceWorkflowSpec, 0, len(model.Workflows))
	for _, workflow := range model.Workflows {
		if workflow.Name.IsNull() || workflow.Name.IsUnknown() || workflow.Name.ValueString() == "" {
			return "", nil, workspaceLayoutConfig{}, workspaceNodeLayoutConfig{}, fmt.Errorf("each workflow entry must include a name")
		}
		if workflow.WorkflowJSON.IsNull() || workflow.WorkflowJSON.IsUnknown() || workflow.WorkflowJSON.ValueString() == "" {
			return "", nil, workspaceLayoutConfig{}, workspaceNodeLayoutConfig{}, fmt.Errorf("workflow %q must include workflow_json", workflow.Name.ValueString())
		}

		spec := workspaceWorkflowSpec{
			Name:         workflow.Name.ValueString(),
			WorkflowJSON: workflow.WorkflowJSON.ValueString(),
			Style:        workspaceWorkflowStyleConfigFromModel(workflow.Style),
		}
		if !workflow.X.IsNull() && !workflow.X.IsUnknown() {
			x := workflow.X.ValueFloat64()
			spec.X = &x
		}
		if !workflow.Y.IsNull() && !workflow.Y.IsUnknown() {
			y := workflow.Y.ValueFloat64()
			spec.Y = &y
		}
		specs = append(specs, spec)
	}

	return model.Name.ValueString(), specs, layout, nodeLayout, nil
}

func workspaceWorkflowStyleConfigFromModel(model *workspaceWorkflowStyleModel) workspaceWorkflowStyleConfig {
	cfg := workspaceWorkflowStyleConfig{}
	if model == nil {
		return cfg
	}
	if !model.GroupColor.IsNull() && !model.GroupColor.IsUnknown() {
		cfg.GroupColor = model.GroupColor.ValueString()
	}
	if !model.TitleFontSize.IsNull() && !model.TitleFontSize.IsUnknown() {
		cfg.TitleFontSize = int(model.TitleFontSize.ValueInt64())
	}
	return cfg
}

func workspaceNodeLayoutConfigFromModel(model *workspaceNodeLayoutModel) (workspaceNodeLayoutConfig, error) {
	cfg := workspaceNodeLayoutConfig{
		Mode:      "dag",
		Direction: "left_to_right",
	}

	if model == nil {
		return cfg, nil
	}

	if !model.Mode.IsNull() && !model.Mode.IsUnknown() {
		if mode := model.Mode.ValueString(); mode != "dag" {
			return workspaceNodeLayoutConfig{}, fmt.Errorf("node_layout.mode must be dag")
		}
	}
	if !model.Direction.IsNull() && !model.Direction.IsUnknown() {
		if direction := model.Direction.ValueString(); direction != "left_to_right" {
			return workspaceNodeLayoutConfig{}, fmt.Errorf("node_layout.direction must be left_to_right")
		}
	}
	if !model.ColumnGap.IsNull() && !model.ColumnGap.IsUnknown() {
		cfg.ColumnGap = model.ColumnGap.ValueFloat64()
	}
	if !model.RowGap.IsNull() && !model.RowGap.IsUnknown() {
		cfg.RowGap = model.RowGap.ValueFloat64()
	}

	return cfg, nil
}

func validateWorkspaceLayout(layout workspaceLayoutConfig) error {
	switch layout.Display {
	case "flex":
		if layout.Columns > 0 {
			return fmt.Errorf("columns is only valid when display is grid")
		}
		if layout.Direction != "" && layout.Direction != "row" && layout.Direction != "column" {
			return fmt.Errorf("direction must be either row or column when display is flex")
		}
	case "grid":
		if layout.Direction != "" {
			return fmt.Errorf("direction is only valid when display is flex")
		}
	default:
		return fmt.Errorf("display must be either flex or grid")
	}

	return nil
}

func writeWorkspaceFile(filePath string, contents []byte) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	if err := os.WriteFile(filePath, contents, 0o644); err != nil {
		return fmt.Errorf("writing file %s: %w", filePath, err)
	}
	return nil
}

func cleanupPreviousWorkspaceFile(previousPath, currentPath string) error {
	if previousPath == "" || previousPath == currentPath {
		return nil
	}
	if err := os.Remove(previousPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing previous file %s: %w", previousPath, err)
	}
	return nil
}

func stringValue(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

func float64Value(v types.Float64) float64 {
	if v.IsNull() || v.IsUnknown() {
		return 0
	}
	return v.ValueFloat64()
}

func int64Value(v types.Int64) int64 {
	if v.IsNull() || v.IsUnknown() {
		return 0
	}
	return v.ValueInt64()
}
