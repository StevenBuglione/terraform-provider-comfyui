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
	PromptID             types.String  `tfsdk:"prompt_id"`
	Status               types.String  `tfsdk:"status"`
	Completed            types.Bool    `tfsdk:"completed"`
	Outputs              types.String  `tfsdk:"outputs"`
	CreateTime           types.Int64   `tfsdk:"create_time"`
	ExecutionStartTime   types.Int64   `tfsdk:"execution_start_time"`
	ExecutionEndTime     types.Int64   `tfsdk:"execution_end_time"`
	OutputsCount         types.Int64   `tfsdk:"outputs_count"`
	WorkflowID           types.String  `tfsdk:"workflow_id"`
	PromptJSON           types.String  `tfsdk:"prompt_json"`
	ExtraDataJSON        types.String  `tfsdk:"extra_data_json"`
	OutputsToExecuteJSON types.String  `tfsdk:"outputs_to_execute_json"`
	ExecutionStatusJSON  types.String  `tfsdk:"execution_status_json"`
	ExecutionErrorJSON   types.String  `tfsdk:"execution_error_json"`
	Prompt               types.Dynamic `tfsdk:"prompt"`
	ExtraData            types.Dynamic `tfsdk:"extra_data"`
	OutputsToExecute     types.Dynamic `tfsdk:"outputs_to_execute"`
	ExecutionStatus      types.Dynamic `tfsdk:"execution_status"`
	ExecutionError       types.Dynamic `tfsdk:"execution_error"`
	OutputsStructured    types.Dynamic `tfsdk:"outputs_structured"`
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
			"create_time": schema.Int64Attribute{
				Description: "Unix timestamp recorded in extra_data when available.",
				Computed:    true,
			},
			"execution_start_time": schema.Int64Attribute{
				Description: "Unix timestamp of the execution_start event when available.",
				Computed:    true,
			},
			"execution_end_time": schema.Int64Attribute{
				Description: "Unix timestamp of the terminal execution event when available.",
				Computed:    true,
			},
			"outputs_count": schema.Int64Attribute{
				Description: "Count of output items recorded in history.",
				Computed:    true,
			},
			"workflow_id": schema.StringAttribute{
				Description: "Workflow identifier embedded in extra_pnginfo when available.",
				Computed:    true,
			},
			"prompt_json": schema.StringAttribute{
				Description: "JSON representation of the executed prompt graph.",
				Computed:    true,
			},
			"extra_data_json": schema.StringAttribute{
				Description: "JSON representation of the stored extra_data payload.",
				Computed:    true,
			},
			"outputs_to_execute_json": schema.StringAttribute{
				Description: "JSON representation of the stored outputs_to_execute payload.",
				Computed:    true,
			},
			"execution_status_json": schema.StringAttribute{
				Description: "JSON representation of the raw execution status payload.",
				Computed:    true,
			},
			"execution_error_json": schema.StringAttribute{
				Description: "JSON representation of execution_error when present in history messages.",
				Computed:    true,
			},
			"prompt": schema.DynamicAttribute{
				Description: "Structured prompt graph from the history entry.",
				Computed:    true,
			},
			"extra_data": schema.DynamicAttribute{
				Description: "Structured extra_data payload from the history entry.",
				Computed:    true,
			},
			"outputs_to_execute": schema.DynamicAttribute{
				Description: "Structured outputs_to_execute payload from the history entry.",
				Computed:    true,
			},
			"execution_status": schema.DynamicAttribute{
				Description: "Structured raw execution status payload from history.",
				Computed:    true,
			},
			"execution_error": schema.DynamicAttribute{
				Description: "Structured execution_error payload extracted from history messages.",
				Computed:    true,
			},
			"outputs_structured": schema.DynamicAttribute{
				Description: "Structured node outputs payload.",
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

	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The ComfyUI client is required to read workflow history.")
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
			PromptID:             types.StringValue(promptID),
			Status:               types.StringValue("not_found"),
			Completed:            types.BoolValue(false),
			Outputs:              types.StringValue("{}"),
			CreateTime:           types.Int64Null(),
			ExecutionStartTime:   types.Int64Null(),
			ExecutionEndTime:     types.Int64Null(),
			OutputsCount:         types.Int64Value(0),
			WorkflowID:           types.StringNull(),
			PromptJSON:           types.StringNull(),
			ExtraDataJSON:        types.StringNull(),
			OutputsToExecuteJSON: types.StringNull(),
			ExecutionStatusJSON:  types.StringNull(),
			ExecutionErrorJSON:   types.StringNull(),
			Prompt:               types.DynamicNull(),
			ExtraData:            types.DynamicNull(),
			OutputsToExecute:     types.DynamicNull(),
			ExecutionStatus:      types.DynamicNull(),
			ExecutionError:       types.DynamicNull(),
			OutputsStructured:    types.DynamicNull(),
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	state, err := buildWorkflowHistoryModel(promptID, &entry)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build workflow history model", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func buildWorkflowHistoryModel(promptID string, entry *client.HistoryEntry) (*WorkflowHistoryModel, error) {
	outputsValue := entry.Outputs
	if outputsValue == nil {
		outputsValue = map[string]interface{}{}
	}

	outputsJSON, err := json.Marshal(outputsValue)
	if err != nil {
		return nil, fmt.Errorf("marshal outputs: %w", err)
	}

	normalizedOutputs, err := normalizeJSONValue(outputsValue)
	if err != nil {
		return nil, fmt.Errorf("normalize outputs: %w", err)
	}
	outputsStructured, err := dynamicFromInterface(normalizedOutputs)
	if err != nil {
		return nil, fmt.Errorf("convert outputs: %w", err)
	}

	promptJSON := types.StringNull()
	promptValue := types.DynamicNull()
	extraDataJSON := types.StringNull()
	extraDataValue := types.DynamicNull()
	outputsToExecuteJSON := types.StringNull()
	outputsToExecuteValue := types.DynamicNull()
	createTime := types.Int64Null()
	workflowID := types.StringNull()
	if len(entry.Prompt) >= 4 {
		promptJSON, promptValue, extraDataJSON, extraDataValue, outputsToExecuteJSON, outputsToExecuteValue, createTime, workflowID, err = buildPromptTupleFields(entry.Prompt)
		if err != nil {
			return nil, err
		}
	}

	executionStatusJSON, err := jsonStringFromValue(entry.Status)
	if err != nil {
		return nil, fmt.Errorf("marshal execution_status: %w", err)
	}
	normalizedStatus, err := normalizeJSONValue(entry.Status)
	if err != nil {
		return nil, fmt.Errorf("normalize execution_status: %w", err)
	}
	executionStatusValue, err := dynamicFromInterface(normalizedStatus)
	if err != nil {
		return nil, fmt.Errorf("convert execution_status: %w", err)
	}

	startTime, endTime, executionError := extractExecutionEvents(entry.Status.Messages)
	executionErrorJSON, err := jsonStringFromValue(executionError)
	if err != nil {
		return nil, fmt.Errorf("marshal execution_error: %w", err)
	}
	executionErrorValue, err := dynamicFromInterface(executionError)
	if err != nil {
		return nil, fmt.Errorf("convert execution_error: %w", err)
	}

	outputsCount, err := countHistoryOutputs(entry.Outputs)
	if err != nil {
		return nil, fmt.Errorf("count outputs: %w", err)
	}

	status := entry.Status.StatusStr
	if status == "" {
		if entry.Status.Completed {
			status = "completed"
		} else {
			status = "running"
		}
	}

	return &WorkflowHistoryModel{
		PromptID:             types.StringValue(promptID),
		Status:               types.StringValue(status),
		Completed:            types.BoolValue(entry.Status.Completed),
		Outputs:              types.StringValue(string(outputsJSON)),
		CreateTime:           createTime,
		ExecutionStartTime:   startTime,
		ExecutionEndTime:     endTime,
		OutputsCount:         types.Int64Value(outputsCount),
		WorkflowID:           workflowID,
		PromptJSON:           promptJSON,
		ExtraDataJSON:        extraDataJSON,
		OutputsToExecuteJSON: outputsToExecuteJSON,
		ExecutionStatusJSON:  executionStatusJSON,
		ExecutionErrorJSON:   executionErrorJSON,
		Prompt:               promptValue,
		ExtraData:            extraDataValue,
		OutputsToExecute:     outputsToExecuteValue,
		ExecutionStatus:      executionStatusValue,
		ExecutionError:       executionErrorValue,
		OutputsStructured:    outputsStructured,
	}, nil
}
