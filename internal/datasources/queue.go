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

var _ datasource.DataSource = &QueueDataSource{}
var _ datasource.DataSourceWithConfigure = &QueueDataSource{}

type QueueDataSource struct {
	client *client.Client
}

type QueueModel struct {
	RunningCount types.Int64      `tfsdk:"running_count"`
	PendingCount types.Int64      `tfsdk:"pending_count"`
	QueueRunning types.String     `tfsdk:"queue_running"`
	QueuePending types.String     `tfsdk:"queue_pending"`
	RunningItems []QueueItemModel `tfsdk:"running_items"`
	PendingItems []QueueItemModel `tfsdk:"pending_items"`
}

type QueueItemModel struct {
	Priority             types.Int64  `tfsdk:"priority"`
	PromptID             types.String `tfsdk:"prompt_id"`
	CreateTime           types.Int64  `tfsdk:"create_time"`
	WorkflowID           types.String `tfsdk:"workflow_id"`
	PromptJSON           types.String `tfsdk:"prompt_json"`
	ExtraDataJSON        types.String `tfsdk:"extra_data_json"`
	OutputsToExecuteJSON types.String `tfsdk:"outputs_to_execute_json"`
}

func NewQueueDataSource() datasource.DataSource {
	return &QueueDataSource{}
}

func (d *QueueDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_queue"
}

func (d *QueueDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *QueueDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves queue status from the ComfyUI server.",
		Attributes: map[string]schema.Attribute{
			"running_count": schema.Int64Attribute{
				Description: "Number of currently running queue items.",
				Computed:    true,
			},
			"pending_count": schema.Int64Attribute{
				Description: "Number of pending queue items.",
				Computed:    true,
			},
			"queue_running": schema.StringAttribute{
				Description: "JSON representation of running queue items.",
				Computed:    true,
			},
			"queue_pending": schema.StringAttribute{
				Description: "JSON representation of pending queue items.",
				Computed:    true,
			},
			"running_items": schema.ListNestedAttribute{
				Description: "Structured details for currently running queue items.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: queueItemAttributes(),
				},
			},
			"pending_items": schema.ListNestedAttribute{
				Description: "Structured details for pending queue items.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: queueItemAttributes(),
				},
			},
		},
	}
}

func (d *QueueDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The ComfyUI client is required to read queue data.")
		return
	}

	queue, err := d.client.GetQueue()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read queue status", err.Error())
		return
	}

	state, err := buildQueueModel(queue)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build queue model", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func queueItemAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"priority": schema.Int64Attribute{
			Description: "Queue priority/number assigned by ComfyUI.",
			Computed:    true,
		},
		"prompt_id": schema.StringAttribute{
			Description: "Prompt identifier for the queued workflow.",
			Computed:    true,
		},
		"create_time": schema.Int64Attribute{
			Description: "Unix timestamp recorded in queue extra_data when available.",
			Computed:    true,
		},
		"workflow_id": schema.StringAttribute{
			Description: "Workflow identifier embedded in extra_pnginfo when available.",
			Computed:    true,
		},
		"prompt_json": schema.StringAttribute{
			Description: "JSON representation of the queued prompt graph.",
			Computed:    true,
		},
		"extra_data_json": schema.StringAttribute{
			Description: "JSON representation of queue extra_data.",
			Computed:    true,
		},
		"outputs_to_execute_json": schema.StringAttribute{
			Description: "JSON representation of the node targets selected for execution.",
			Computed:    true,
		},
	}
}

func buildQueueModel(queue *client.QueueStatus) (*QueueModel, error) {
	runningJSON, err := json.Marshal(queue.QueueRunning)
	if err != nil {
		return nil, fmt.Errorf("marshal running queue: %w", err)
	}

	pendingJSON, err := json.Marshal(queue.QueuePending)
	if err != nil {
		return nil, fmt.Errorf("marshal pending queue: %w", err)
	}

	runningItems, err := buildQueueItemModels(queue.QueueRunning)
	if err != nil {
		return nil, fmt.Errorf("build running items: %w", err)
	}

	pendingItems, err := buildQueueItemModels(queue.QueuePending)
	if err != nil {
		return nil, fmt.Errorf("build pending items: %w", err)
	}

	return &QueueModel{
		RunningCount: types.Int64Value(int64(len(queue.QueueRunning))),
		PendingCount: types.Int64Value(int64(len(queue.QueuePending))),
		QueueRunning: types.StringValue(string(runningJSON)),
		QueuePending: types.StringValue(string(pendingJSON)),
		RunningItems: runningItems,
		PendingItems: pendingItems,
	}, nil
}

func buildQueueItemModels(items [][]interface{}) ([]QueueItemModel, error) {
	models := make([]QueueItemModel, 0, len(items))
	for _, item := range items {
		model, err := buildQueueItemModel(item)
		if err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, nil
}

func buildQueueItemModel(item []interface{}) (QueueItemModel, error) {
	if len(item) < 4 {
		return QueueItemModel{}, fmt.Errorf("expected queue item with at least 4 elements, got %d", len(item))
	}

	promptJSON, _, extraDataJSON, _, outputsToExecuteJSON, _, createTime, workflowID, err := buildPromptTupleFields(item)
	if err != nil {
		return QueueItemModel{}, err
	}

	priority, _ := int64FromAny(item[0])

	return QueueItemModel{
		Priority:             types.Int64Value(priority),
		PromptID:             types.StringValue(stringFromAny(item[1])),
		CreateTime:           createTime,
		WorkflowID:           workflowID,
		PromptJSON:           promptJSON,
		ExtraDataJSON:        extraDataJSON,
		OutputsToExecuteJSON: outputsToExecuteJSON,
	}, nil
}
