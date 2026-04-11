package resources

import (
	"context"
	"os"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/artifacts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SubgraphResource{}
var _ resource.ResourceWithImportState = &SubgraphResource{}

type SubgraphResource struct{}

type SubgraphResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Path            types.String `tfsdk:"path"`
	JSON            types.String `tfsdk:"json"`
	NormalizedJSON  types.String `tfsdk:"normalized_json"`
	SHA256          types.String `tfsdk:"sha256"`
	WorkspaceID     types.String `tfsdk:"workspace_id"`
	WorkspaceName   types.String `tfsdk:"workspace_name"`
	NodeCount       types.Int64  `tfsdk:"node_count"`
	GroupCount      types.Int64  `tfsdk:"group_count"`
	DefinitionCount types.Int64  `tfsdk:"definition_count"`
	DefinitionIDs   types.List   `tfsdk:"definition_ids"`
}

func NewSubgraphResource() resource.Resource {
	return &SubgraphResource{}
}

func (r *SubgraphResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subgraph"
}

func (r *SubgraphResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages local editor-native ComfyUI subgraph/workspace JSON on disk. This resource validates and summarizes the native JSON locally and does not mutate remote /global_subgraphs, which is read-only upstream.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Managed local file identifier, equal to the configured path.",
			},
			"path": schema.StringAttribute{
				Required:    true,
				Description: "Local file path where the editor-native JSON should be materialized.",
			},
			"json": schema.StringAttribute{
				Required:    true,
				Description: "Exact editor-native ComfyUI subgraph or blueprint JSON to persist locally.",
			},
			"normalized_json": schema.StringAttribute{
				Computed:    true,
				Description: "Normalized editor-native JSON parsed through the faithful workspace model.",
			},
			"sha256": schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the persisted local JSON file.",
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
				Description: "Number of top-level editor nodes in the JSON payload.",
			},
			"group_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of top-level editor groups in the JSON payload.",
			},
			"definition_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of embedded definitions.subgraphs entries.",
			},
			"definition_ids": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Identifiers from definitions.subgraphs for quick discovery and import polish.",
			},
		},
	}
}

func (r *SubgraphResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SubgraphResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.materialize(&data, ""); err != nil {
		resp.Diagnostics.AddError("Unable to write subgraph file", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubgraphResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SubgraphResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.refresh(&data)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read subgraph file", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubgraphResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SubgraphResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SubgraphResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.materialize(&data, state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to update subgraph file", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubgraphResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SubgraphResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.remove(data.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete subgraph file", err.Error())
	}
}

func (r *SubgraphResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import path", "Import path must not be empty.")
		return
	}

	state, err := subgraphStateFromFile(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to import subgraph file", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), state.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), state.Path)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("json"), state.JSON)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("normalized_json"), state.NormalizedJSON)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("sha256"), state.SHA256)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), state.WorkspaceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_name"), state.WorkspaceName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("node_count"), state.NodeCount)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_count"), state.GroupCount)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("definition_count"), state.DefinitionCount)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("definition_ids"), state.DefinitionIDs)...)
}

func (r *SubgraphResource) materialize(data *SubgraphResourceModel, previousPath string) error {
	state, err := subgraphStateFromInput(stringValue(data.Path), stringValue(data.JSON))
	if err != nil {
		return err
	}

	sha, err := writeManagedArtifactFile(stringValue(data.Path), stringValue(data.JSON))
	if err != nil {
		return err
	}
	if err := cleanupPreviousArtifactFile(previousPath, stringValue(data.Path)); err != nil {
		return err
	}

	state.ID = types.StringValue(stringValue(data.Path))
	state.SHA256 = types.StringValue(sha)
	*data = state
	return nil
}

func (r *SubgraphResource) refresh(data *SubgraphResourceModel) (bool, error) {
	state, err := subgraphStateFromFile(stringValue(data.Path))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	*data = state
	return true, nil
}

func (r *SubgraphResource) remove(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func subgraphStateFromInput(path string, raw string) (SubgraphResourceModel, error) {
	workspace, err := artifacts.ParseWorkspaceJSON(raw)
	if err != nil {
		return SubgraphResourceModel{}, err
	}
	normalizedJSON, err := workspace.JSON()
	if err != nil {
		return SubgraphResourceModel{}, err
	}

	return SubgraphResourceModel{
		ID:              stringValueOrNullResource(path),
		Path:            stringValueOrNullResource(path),
		JSON:            types.StringValue(raw),
		NormalizedJSON:  types.StringValue(normalizedJSON),
		SHA256:          types.StringNull(),
		WorkspaceID:     stringValueOrNullResource(workspace.ID),
		WorkspaceName:   stringValueOrNullResource(workspace.Name),
		NodeCount:       types.Int64Value(int64(len(workspace.Nodes))),
		GroupCount:      types.Int64Value(int64(len(workspace.Groups))),
		DefinitionCount: types.Int64Value(int64(len(workspace.Definitions.Subgraphs))),
		DefinitionIDs:   stringListValueResource(definitionIDsFromWorkspaceResource(workspace)),
	}, nil
}

func subgraphStateFromFile(path string) (SubgraphResourceModel, error) {
	content, sha, err := readManagedArtifactFile(path)
	if err != nil {
		return SubgraphResourceModel{}, err
	}

	state, err := subgraphStateFromInput(path, content)
	if err != nil {
		return SubgraphResourceModel{}, err
	}
	state.ID = types.StringValue(path)
	state.Path = types.StringValue(path)
	state.JSON = types.StringValue(content)
	state.SHA256 = types.StringValue(sha)
	return state, nil
}

func stringValueOrNullResource(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func stringListValueResource(values []string) types.List {
	list, diags := types.ListValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		return types.ListNull(types.StringType)
	}
	return list
}

func definitionIDsFromWorkspaceResource(workspace *artifacts.Workspace) []string {
	ids := make([]string, 0, len(workspace.Definitions.Subgraphs))
	for _, definition := range workspace.Definitions.Subgraphs {
		if definition.ID != "" {
			ids = append(ids, definition.ID)
		}
	}
	return ids
}
