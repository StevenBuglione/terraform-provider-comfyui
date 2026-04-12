package datasources

import (
	"context"
	"fmt"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &JobsDataSource{}
var _ datasource.DataSourceWithConfigure = &JobsDataSource{}

type JobsDataSource struct {
	client *client.Client
}

type JobsModel struct {
	ID         types.String       `tfsdk:"id"`
	Statuses   types.List         `tfsdk:"statuses"`
	WorkflowID types.String       `tfsdk:"workflow_id"`
	SortBy     types.String       `tfsdk:"sort_by"`
	SortOrder  types.String       `tfsdk:"sort_order"`
	Limit      types.Int64        `tfsdk:"limit"`
	Offset     types.Int64        `tfsdk:"offset"`
	HasMore    types.Bool         `tfsdk:"has_more"`
	JobCount   types.Int64        `tfsdk:"job_count"`
	Jobs       []JobListItemModel `tfsdk:"jobs"`
}

type JobListItemModel struct {
	ID                  types.String `tfsdk:"id"`
	Status              types.String `tfsdk:"status"`
	Priority            types.Int64  `tfsdk:"priority"`
	CreateTime          types.Int64  `tfsdk:"create_time"`
	ExecutionStartTime  types.Int64  `tfsdk:"execution_start_time"`
	ExecutionEndTime    types.Int64  `tfsdk:"execution_end_time"`
	OutputsCount        types.Int64  `tfsdk:"outputs_count"`
	WorkflowID          types.String `tfsdk:"workflow_id"`
	PreviewOutputJSON   types.String `tfsdk:"preview_output_json"`
	OutputsJSON         types.String `tfsdk:"outputs_json"`
	ExecutionStatusJSON types.String `tfsdk:"execution_status_json"`
	ExecutionErrorJSON  types.String `tfsdk:"execution_error_json"`
	WorkflowJSON        types.String `tfsdk:"workflow_json"`
}

func NewJobsDataSource() datasource.DataSource {
	return &JobsDataSource{}
}

func (d *JobsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jobs"
}

func (d *JobsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *JobsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of jobs from the ComfyUI server with optional filtering.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this data source.",
				Computed:    true,
			},
			"statuses": schema.ListAttribute{
				Description: "Filter jobs by status (e.g., pending, running, completed, error).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"workflow_id": schema.StringAttribute{
				Description: "Filter jobs by workflow ID.",
				Optional:    true,
			},
			"sort_by": schema.StringAttribute{
				Description: "Sort jobs by field (e.g., create_time, priority).",
				Optional:    true,
			},
			"sort_order": schema.StringAttribute{
				Description: "Sort order (asc or desc).",
				Optional:    true,
			},
			"limit": schema.Int64Attribute{
				Description: "Maximum number of jobs to return.",
				Optional:    true,
			},
			"offset": schema.Int64Attribute{
				Description: "Number of jobs to skip (for pagination).",
				Optional:    true,
			},
			"has_more": schema.BoolAttribute{
				Description: "Indicates if there are more jobs available beyond the current result set.",
				Computed:    true,
			},
			"job_count": schema.Int64Attribute{
				Description: "Number of jobs returned in this result set.",
				Computed:    true,
			},
			"jobs": schema.ListNestedAttribute{
				Description: "List of jobs matching the filter criteria.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the job.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The current status of the job.",
							Computed:    true,
						},
						"priority": schema.Int64Attribute{
							Description: "The priority of the job.",
							Computed:    true,
						},
						"create_time": schema.Int64Attribute{
							Description: "Unix timestamp when the job was created.",
							Computed:    true,
						},
						"execution_start_time": schema.Int64Attribute{
							Description: "Unix timestamp when job execution started.",
							Computed:    true,
						},
						"execution_end_time": schema.Int64Attribute{
							Description: "Unix timestamp when job execution ended.",
							Computed:    true,
						},
						"outputs_count": schema.Int64Attribute{
							Description: "Number of outputs produced by the job.",
							Computed:    true,
						},
						"workflow_id": schema.StringAttribute{
							Description: "The workflow ID associated with this job.",
							Computed:    true,
						},
						"preview_output_json": schema.StringAttribute{
							Description: "JSON representation of preview output.",
							Computed:    true,
						},
						"outputs_json": schema.StringAttribute{
							Description: "JSON representation of job outputs.",
							Computed:    true,
						},
						"execution_status_json": schema.StringAttribute{
							Description: "JSON representation of execution status.",
							Computed:    true,
						},
						"execution_error_json": schema.StringAttribute{
							Description: "JSON representation of execution error (if any).",
							Computed:    true,
						},
						"workflow_json": schema.StringAttribute{
							Description: "JSON representation of the workflow.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *JobsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config JobsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The ComfyUI client is required to list jobs.")
		return
	}

	filter, diags := buildJobListFilter(&config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobsResp, err := d.client.ListJobs(filter)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list jobs", err.Error())
		return
	}

	state, err := buildJobsModel(jobsResp, &config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build jobs model", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// buildJobListFilter converts a JobsModel to a client.JobListFilter
func buildJobListFilter(model *JobsModel) (client.JobListFilter, diag.Diagnostics) {
	var diags diag.Diagnostics
	filter := client.JobListFilter{}

	// Statuses
	if !model.Statuses.IsNull() && !model.Statuses.IsUnknown() {
		var statuses []string
		elemDiags := model.Statuses.ElementsAs(context.Background(), &statuses, false)
		diags.Append(elemDiags...)
		if diags.HasError() {
			return filter, diags
		}
		filter.Status = statuses
	}

	// WorkflowID
	if !model.WorkflowID.IsNull() && !model.WorkflowID.IsUnknown() {
		filter.WorkflowID = model.WorkflowID.ValueString()
	}

	// SortBy
	if !model.SortBy.IsNull() && !model.SortBy.IsUnknown() {
		filter.SortBy = model.SortBy.ValueString()
	}

	// SortOrder
	if !model.SortOrder.IsNull() && !model.SortOrder.IsUnknown() {
		filter.SortOrder = model.SortOrder.ValueString()
	}

	// Limit - validate and convert safely
	if !model.Limit.IsNull() && !model.Limit.IsUnknown() {
		limitVal := model.Limit.ValueInt64()

		// Reject negative values
		if limitVal < 0 {
			diags.AddError(
				"Invalid Pagination Parameter",
				fmt.Sprintf("Limit must be non-negative, got: %d", limitVal),
			)
			return filter, diags
		}

		// Prevent overflow: use a safe bound that's valid for this provider
		// 1 billion is a reasonable limit for pagination (well below int32 max)
		const maxSafePaginationValue = 1_000_000_000
		if limitVal > maxSafePaginationValue {
			diags.AddError(
				"Invalid Pagination Parameter",
				fmt.Sprintf("Limit exceeds maximum safe value of %d, got: %d", maxSafePaginationValue, limitVal),
			)
			return filter, diags
		}

		limit := int(limitVal)
		filter.Limit = &limit
	}

	// Offset - validate and convert safely
	if !model.Offset.IsNull() && !model.Offset.IsUnknown() {
		offsetVal := model.Offset.ValueInt64()

		// Reject negative values
		if offsetVal < 0 {
			diags.AddError(
				"Invalid Pagination Parameter",
				fmt.Sprintf("Offset must be non-negative, got: %d", offsetVal),
			)
			return filter, diags
		}

		// Prevent overflow: use the same safe bound
		const maxSafePaginationValue = 1_000_000_000
		if offsetVal > maxSafePaginationValue {
			diags.AddError(
				"Invalid Pagination Parameter",
				fmt.Sprintf("Offset exceeds maximum safe value of %d, got: %d", maxSafePaginationValue, offsetVal),
			)
			return filter, diags
		}

		offset := int(offsetVal)
		filter.Offset = &offset
	}

	return filter, diags
}

// buildJobsModel converts a client.JobsResponse to a JobsModel
func buildJobsModel(resp *client.JobsResponse, inputModel *JobsModel) (*JobsModel, error) {
	model := &JobsModel{
		ID:       types.StringValue("jobs"),
		HasMore:  types.BoolValue(resp.HasMore),
		JobCount: types.Int64Value(int64(len(resp.Jobs))),
		Jobs:     make([]JobListItemModel, 0, len(resp.Jobs)),
	}

	// Preserve input filter fields
	if inputModel != nil {
		model.Statuses = inputModel.Statuses
		model.WorkflowID = inputModel.WorkflowID
		model.SortBy = inputModel.SortBy
		model.SortOrder = inputModel.SortOrder
		model.Limit = inputModel.Limit
		model.Offset = inputModel.Offset
	}

	for _, job := range resp.Jobs {
		jobModel, err := buildJobModel(&job)
		if err != nil {
			return nil, fmt.Errorf("failed to build job model for job %s: %w", job.ID, err)
		}
		model.Jobs = append(model.Jobs, jobListItemModelFromJobModel(jobModel))
	}

	return model, nil
}

func jobListItemModelFromJobModel(model *JobModel) JobListItemModel {
	return JobListItemModel{
		ID:                  model.ID,
		Status:              model.Status,
		Priority:            model.Priority,
		CreateTime:          model.CreateTime,
		ExecutionStartTime:  model.ExecutionStartTime,
		ExecutionEndTime:    model.ExecutionEndTime,
		OutputsCount:        model.OutputsCount,
		WorkflowID:          model.WorkflowID,
		PreviewOutputJSON:   model.PreviewOutputJSON,
		OutputsJSON:         model.OutputsJSON,
		ExecutionStatusJSON: model.ExecutionStatusJSON,
		ExecutionErrorJSON:  model.ExecutionErrorJSON,
		WorkflowJSON:        model.WorkflowJSON,
	}
}
