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

var _ datasource.DataSource = &WorkflowHistoryDataSource{}
var _ datasource.DataSourceWithConfigure = &WorkflowHistoryDataSource{}

type WorkflowHistoryDataSource struct {
	client *client.Client
}

type WorkflowHistoryModel struct {
	PromptID  types.String `tfsdk:"prompt_id"`
	Status    types.String `tfsdk:"status"`
	Completed types.Bool   `tfsdk:"completed"`
	Outputs   types.String `tfsdk:"outputs"`
}

func NewWorkflowHistoryDataSource() datasource.DataSource {
	return &WorkflowHistoryDataSource{}
}

func (d *WorkflowHistoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow_history"
}

func (d *WorkflowHistoryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *WorkflowHistoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves workflow execution history for a specific prompt from the ComfyUI server.",
		Attributes: map[string]schema.Attribute{
			"prompt_id": schema.StringAttribute{
				Description: "The prompt ID to look up history for.",
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: "Execution status (e.g., completed, error, running).",
				Computed:    true,
			},
			"completed": schema.BoolAttribute{
				Description: "Whether the workflow execution has completed.",
				Computed:    true,
			},
			"outputs": schema.StringAttribute{
				Description: "JSON representation of all node outputs.",
				Computed:    true,
			},
		},
	}
}

func (d *WorkflowHistoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WorkflowHistoryModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	promptID := config.PromptID.ValueString()

	history, err := d.client.GetHistory(promptID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read workflow history",
			fmt.Sprintf("Could not read history for prompt %q: %s", promptID, err.Error()),
		)
		return
	}

	entry, ok := (*history)[promptID]
	if !ok {
		// Prompt not found in history — return empty/default state
		state := WorkflowHistoryModel{
			PromptID:  types.StringValue(promptID),
			Status:    types.StringValue("not_found"),
			Completed: types.BoolValue(false),
			Outputs:   types.StringValue("{}"),
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	outputsJSON, err := json.Marshal(entry.Outputs)
	if err != nil {
		resp.Diagnostics.AddError("Unable to marshal outputs", err.Error())
		return
	}

	status := entry.Status.StatusStr
	if status == "" {
		if entry.Status.Completed {
			status = "completed"
		} else {
			status = "running"
		}
	}

	state := WorkflowHistoryModel{
		PromptID:  types.StringValue(promptID),
		Status:    types.StringValue(status),
		Completed: types.BoolValue(entry.Status.Completed),
		Outputs:   types.StringValue(string(outputsJSON)),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
