package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ datasource.DataSource = &JobDataSource{}
var _ datasource.DataSourceWithConfigure = &JobDataSource{}

type JobDataSource struct {
	client *client.Client
}

type JobModel struct {
	ID                  types.String  `tfsdk:"id"`
	Status              types.String  `tfsdk:"status"`
	Priority            types.Int64   `tfsdk:"priority"`
	CreateTime          types.Int64   `tfsdk:"create_time"`
	ExecutionStartTime  types.Int64   `tfsdk:"execution_start_time"`
	ExecutionEndTime    types.Int64   `tfsdk:"execution_end_time"`
	OutputsCount        types.Int64   `tfsdk:"outputs_count"`
	WorkflowID          types.String  `tfsdk:"workflow_id"`
	PreviewOutputJSON   types.String  `tfsdk:"preview_output_json"`
	OutputsJSON         types.String  `tfsdk:"outputs_json"`
	ExecutionStatusJSON types.String  `tfsdk:"execution_status_json"`
	ExecutionErrorJSON  types.String  `tfsdk:"execution_error_json"`
	WorkflowJSON        types.String  `tfsdk:"workflow_json"`
	PreviewOutput       types.Dynamic `tfsdk:"preview_output"`
	Outputs             types.Dynamic `tfsdk:"outputs"`
	ExecutionStatus     types.Dynamic `tfsdk:"execution_status"`
	ExecutionError      types.Dynamic `tfsdk:"execution_error"`
	Workflow            types.Dynamic `tfsdk:"workflow"`
}

func NewJobDataSource() datasource.DataSource {
	return &JobDataSource{}
}

func (d *JobDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job"
}

func (d *JobDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *JobDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves details of a specific job from the ComfyUI server.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the job.",
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the job (e.g., pending, running, completed, error).",
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
			"preview_output": schema.DynamicAttribute{
				Description: "Structured preview output data.",
				Computed:    true,
			},
			"outputs": schema.DynamicAttribute{
				Description: "Structured job outputs data.",
				Computed:    true,
			},
			"execution_status": schema.DynamicAttribute{
				Description: "Structured execution status data.",
				Computed:    true,
			},
			"execution_error": schema.DynamicAttribute{
				Description: "Structured execution error data.",
				Computed:    true,
			},
			"workflow": schema.DynamicAttribute{
				Description: "Structured workflow data.",
				Computed:    true,
			},
		},
	}
}

func (d *JobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config JobModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The ComfyUI client is required to read job data.")
		return
	}

	jobID := config.ID.ValueString()
	job, err := d.client.GetJob(jobID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read job", err.Error())
		return
	}

	state, err := buildJobModel(job)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build job model", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// buildJobModel converts a client.Job to a JobModel
func buildJobModel(job *client.Job) (*JobModel, error) {
	model := &JobModel{
		ID:                 types.StringValue(job.ID),
		Status:             types.StringValue(job.Status),
		Priority:           types.Int64Value(int64(job.Priority)),
		CreateTime:         int64PointerValueOrNull(job.CreateTime),
		ExecutionStartTime: int64PointerValueOrNull(job.ExecutionStartTime),
		ExecutionEndTime:   int64PointerValueOrNull(job.ExecutionEndTime),
		OutputsCount:       intPointerValueOrNull(job.OutputsCount),
	}

	// WorkflowID
	if job.WorkflowID != "" {
		model.WorkflowID = types.StringValue(job.WorkflowID)
	} else {
		model.WorkflowID = types.StringNull()
	}

	// PreviewOutputJSON
	if job.PreviewOutput != nil {
		jsonBytes, err := json.Marshal(job.PreviewOutput)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal preview_output: %w", err)
		}
		model.PreviewOutputJSON = types.StringValue(string(jsonBytes))
	} else {
		model.PreviewOutputJSON = types.StringNull()
	}

	// OutputsJSON
	if job.Outputs != nil {
		jsonBytes, err := json.Marshal(job.Outputs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal outputs: %w", err)
		}
		model.OutputsJSON = types.StringValue(string(jsonBytes))
	} else {
		model.OutputsJSON = types.StringNull()
	}

	// ExecutionStatusJSON
	if job.ExecutionStatus != nil {
		jsonBytes, err := json.Marshal(job.ExecutionStatus)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal execution_status: %w", err)
		}
		model.ExecutionStatusJSON = types.StringValue(string(jsonBytes))
	} else {
		model.ExecutionStatusJSON = types.StringNull()
	}

	// ExecutionErrorJSON
	if job.ExecutionError != nil {
		jsonBytes, err := json.Marshal(job.ExecutionError)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal execution_error: %w", err)
		}
		model.ExecutionErrorJSON = types.StringValue(string(jsonBytes))
	} else {
		model.ExecutionErrorJSON = types.StringNull()
	}

	// WorkflowJSON
	if job.Workflow != nil {
		jsonBytes, err := json.Marshal(job.Workflow)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal workflow: %w", err)
		}
		model.WorkflowJSON = types.StringValue(string(jsonBytes))
	} else {
		model.WorkflowJSON = types.StringNull()
	}

	// Structured PreviewOutput
	if job.PreviewOutput != nil {
		dynValue, err := mapToDynamic(job.PreviewOutput)
		if err != nil {
			return nil, fmt.Errorf("failed to convert preview_output to dynamic: %w", err)
		}
		model.PreviewOutput = dynValue
	} else {
		model.PreviewOutput = types.DynamicNull()
	}

	// Structured Outputs
	if job.Outputs != nil {
		dynValue, err := mapToDynamic(job.Outputs)
		if err != nil {
			return nil, fmt.Errorf("failed to convert outputs to dynamic: %w", err)
		}
		model.Outputs = dynValue
	} else {
		model.Outputs = types.DynamicNull()
	}

	// Structured ExecutionStatus
	if job.ExecutionStatus != nil {
		dynValue, err := mapToDynamic(job.ExecutionStatus)
		if err != nil {
			return nil, fmt.Errorf("failed to convert execution_status to dynamic: %w", err)
		}
		model.ExecutionStatus = dynValue
	} else {
		model.ExecutionStatus = types.DynamicNull()
	}

	// Structured ExecutionError
	if job.ExecutionError != nil {
		dynValue, err := mapToDynamic(job.ExecutionError)
		if err != nil {
			return nil, fmt.Errorf("failed to convert execution_error to dynamic: %w", err)
		}
		model.ExecutionError = dynValue
	} else {
		model.ExecutionError = types.DynamicNull()
	}

	// Structured Workflow
	if job.Workflow != nil {
		// Convert JobWorkflow to map[string]interface{}
		workflowMap := map[string]interface{}{
			"prompt":     job.Workflow.Prompt,
			"extra_data": job.Workflow.ExtraData,
		}
		dynValue, err := mapToDynamic(workflowMap)
		if err != nil {
			return nil, fmt.Errorf("failed to convert workflow to dynamic: %w", err)
		}
		model.Workflow = dynValue
	} else {
		model.Workflow = types.DynamicNull()
	}

	return model, nil
}

// mapToDynamic converts a map[string]interface{} to a types.Dynamic value
func mapToDynamic(data map[string]interface{}) (types.Dynamic, error) {
	attrValue, err := interfaceToAttrValue(data)
	if err != nil {
		return types.DynamicNull(), err
	}
	return types.DynamicValue(attrValue), nil
}

func int64PointerValueOrNull(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*value)
}

func intPointerValueOrNull(value *int) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*value))
}

// interfaceToAttrValue recursively converts interface{} to attr.Value
// Faithfully supports arbitrary JSON-like shapes including:
// - nulls (preserved as dynamic nulls, not string nulls)
// - heterogeneous arrays (converted to tuples to support mixed types)
// - nested objects with mixed types and nulls
func interfaceToAttrValue(data interface{}) (attr.Value, error) {
	if data == nil {
		// Preserve null semantics: return a dynamic null, not a string null
		return types.DynamicNull(), nil
	}

	switch v := data.(type) {
	case map[string]interface{}:
		// Check for typed-nil: nil map that isn't actually nil at the interface{} level
		if v == nil {
			return types.DynamicNull(), nil
		}

		attrTypes := make(map[string]attr.Type)
		attrValues := make(map[string]attr.Value)

		for key, val := range v {
			attrVal, err := interfaceToAttrValue(val)
			if err != nil {
				return nil, fmt.Errorf("failed to convert map value for key %q: %w", key, err)
			}
			attrTypes[key] = attrVal.Type(context.Background())
			attrValues[key] = attrVal
		}

		return types.ObjectValueMust(attrTypes, attrValues), nil

	case []interface{}:
		// Check for typed-nil slice
		if v == nil {
			return types.DynamicNull(), nil
		}

		if len(v) == 0 {
			// Empty array - use dynamic type to avoid assuming element type
			return types.TupleValueMust([]attr.Type{}, []attr.Value{}), nil
		}

		// Convert to tuple to support heterogeneous arrays (mixed types)
		// This faithfully represents arbitrary JSON arrays which can contain mixed types
		attrTypes := make([]attr.Type, len(v))
		attrValues := make([]attr.Value, len(v))

		for i, item := range v {
			attrVal, err := interfaceToAttrValue(item)
			if err != nil {
				return nil, fmt.Errorf("failed to convert array item at index %d: %w", i, err)
			}
			attrTypes[i] = attrVal.Type(context.Background())
			attrValues[i] = attrVal
		}

		return types.TupleValueMust(attrTypes, attrValues), nil

	case string:
		return types.StringValue(v), nil

	case float64:
		// Convert float64 to big.Float
		bigFloat := big.NewFloat(v)
		numVal := basetypes.NewNumberValue(bigFloat)
		return numVal, nil

	case int:
		// Convert int to big.Float
		bigFloat := new(big.Float).SetInt64(int64(v))
		numVal := basetypes.NewNumberValue(bigFloat)
		return numVal, nil

	case int64:
		// Convert int64 to big.Float
		bigFloat := new(big.Float).SetInt64(v)
		numVal := basetypes.NewNumberValue(bigFloat)
		return numVal, nil

	case bool:
		return types.BoolValue(v), nil

	default:
		return nil, fmt.Errorf("unsupported type: %T", v)
	}
}
