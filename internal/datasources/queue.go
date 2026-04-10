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
	RunningCount types.Int64  `tfsdk:"running_count"`
	PendingCount types.Int64  `tfsdk:"pending_count"`
	QueueRunning types.String `tfsdk:"queue_running"`
	QueuePending types.String `tfsdk:"queue_pending"`
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
		},
	}
}

func (d *QueueDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	queue, err := d.client.GetQueue()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read queue status", err.Error())
		return
	}

	runningJSON, err := json.Marshal(queue.QueueRunning)
	if err != nil {
		resp.Diagnostics.AddError("Unable to marshal running queue", err.Error())
		return
	}

	pendingJSON, err := json.Marshal(queue.QueuePending)
	if err != nil {
		resp.Diagnostics.AddError("Unable to marshal pending queue", err.Error())
		return
	}

	state := QueueModel{
		RunningCount: types.Int64Value(int64(len(queue.QueueRunning))),
		PendingCount: types.Int64Value(int64(len(queue.QueuePending))),
		QueueRunning: types.StringValue(string(runningJSON)),
		QueuePending: types.StringValue(string(pendingJSON)),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
