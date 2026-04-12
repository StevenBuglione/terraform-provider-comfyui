package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/artifacts"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/validation"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = &WorkflowResource{}
	_ resource.ResourceWithConfigure = &WorkflowResource{}
)

type WorkflowResource struct {
	client *client.Client
}

type WorkflowModel struct {
	ID                    types.String `tfsdk:"id"`
	WorkflowJSON          types.String `tfsdk:"workflow_json"`
	NodeIDs               types.List   `tfsdk:"node_ids"`
	Execute               types.Bool   `tfsdk:"execute"`
	WaitForCompletion     types.Bool   `tfsdk:"wait_for_completion"`
	TimeoutSeconds        types.Int64  `tfsdk:"timeout_seconds"`
	ValidateBeforeExecute types.Bool   `tfsdk:"validate_before_execute"`
	PromptID              types.String `tfsdk:"prompt_id"`
	ClientID              types.String `tfsdk:"client_id"`
	ExtraDataJSON         types.String `tfsdk:"extra_data_json"`
	PartialTargets        types.List   `tfsdk:"partial_execution_targets"`
	ValidationSummaryJSON types.String `tfsdk:"validation_summary_json"`

	// Metadata
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Tags        types.List   `tfsdk:"tags"`
	Category    types.String `tfsdk:"category"`

	// File output
	OutputFile types.String `tfsdk:"output_file"`

	// Computed
	AssembledJSON types.String `tfsdk:"assembled_json"`

	// Chunk 3: Cancel behavior
	CancelOnDelete types.Bool `tfsdk:"cancel_on_delete"`

	// Chunk 3: Execution fields from /api/jobs
	CreateTime            types.Int64   `tfsdk:"create_time"`
	ExecutionStartTime    types.Int64   `tfsdk:"execution_start_time"`
	ExecutionEndTime      types.Int64   `tfsdk:"execution_end_time"`
	OutputsCount          types.Int64   `tfsdk:"outputs_count"`
	WorkflowID            types.String  `tfsdk:"workflow_id"`
	PreviewOutputJSON     types.String  `tfsdk:"preview_output_json"`
	PreviewOutput         types.Dynamic `tfsdk:"preview_output"`
	OutputsJSON           types.String  `tfsdk:"outputs_json"`
	OutputsStructured     types.Dynamic `tfsdk:"outputs_structured"`
	ExecutionStatusJSON   types.String  `tfsdk:"execution_status_json"`
	ExecutionStatus       types.Dynamic `tfsdk:"execution_status"`
	ExecutionErrorJSON    types.String  `tfsdk:"execution_error_json"`
	ExecutionError        types.Dynamic `tfsdk:"execution_error"`
	ExecutionWorkflowJSON types.String  `tfsdk:"execution_workflow_json"`
	ExecutionWorkflow     types.Dynamic `tfsdk:"execution_workflow"`
}

type workflowExecutionRequestConfig struct {
	PromptID                string
	ClientID                string
	ExtraDataJSON           string
	PartialExecutionTargets []string
}

func NewWorkflowResource() resource.Resource {
	return &WorkflowResource{}
}

func (r *WorkflowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (r *WorkflowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assembles ComfyUI node definitions into an executable workflow, submits it to the ComfyUI server, and optionally waits for completion.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for this workflow resource instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workflow_json": schema.StringAttribute{
				Optional:    true,
				Description: "JSON string containing the full ComfyUI workflow (API format). Each top-level key is a node ID mapping to an object with class_type and inputs.",
			},
			"node_ids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "List of node resource IDs to include when assembling a workflow from virtual node resources.",
			},
			"execute": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether to submit the workflow for execution. Defaults to true.",
			},
			"wait_for_completion": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether to wait for execution to finish before marking the resource as created. Defaults to true.",
			},
			"timeout_seconds": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(300),
				Description: "Maximum seconds to wait for workflow execution. Defaults to 300.",
			},
			"validate_before_execute": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether to validate the prompt against live /object_info metadata before queueing execution. Defaults to true.",
			},
			"prompt_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Optional ComfyUI prompt ID to submit. If omitted, ComfyUI assigns one.",
			},
			"client_id": schema.StringAttribute{
				Optional:    true,
				Description: "Optional ComfyUI client_id to include in the /prompt request wrapper.",
			},
			"extra_data_json": schema.StringAttribute{
				Optional:    true,
				Description: "Optional JSON object to include as extra_data in the /prompt request wrapper.",
			},
			"partial_execution_targets": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Optional list of node IDs to send as partial_execution_targets in the /prompt request wrapper.",
			},
			"validation_summary_json": schema.StringAttribute{
				Computed:    true,
				Description: "Structured JSON summary of semantic validation results when workflow preflight validation runs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Human-readable name for this workflow.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of what this workflow does.",
			},
			"tags": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Tags for categorizing and searching workflows (e.g., 'landscape', 'portrait', 'video').",
			},
			"category": schema.StringAttribute{
				Optional:    true,
				Description: "Workflow category (e.g., 'txt2img', 'img2img', 'video', 'audio', '3d').",
			},
			"output_file": schema.StringAttribute{
				Optional:    true,
				Description: "File path to write the assembled workflow JSON. The file is in ComfyUI API format and can be loaded by ComfyUI.",
			},
			"assembled_json": schema.StringAttribute{
				Computed:    true,
				Description: "The assembled workflow in ComfyUI API format JSON. Populated when workflow_json is provided or node assembly is complete.",
			},
			"cancel_on_delete": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to cancel the workflow execution on resource deletion. Defaults to false.",
			},
			"create_time": schema.Int64Attribute{
				Computed:    true,
				Description: "Timestamp when the workflow was created (from /api/jobs).",
			},
			"execution_start_time": schema.Int64Attribute{
				Computed:    true,
				Description: "Timestamp when execution started (from /api/jobs).",
			},
			"execution_end_time": schema.Int64Attribute{
				Computed:    true,
				Description: "Timestamp when execution ended (from /api/jobs).",
			},
			"outputs_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of outputs produced (from /api/jobs).",
			},
			"workflow_id": schema.StringAttribute{
				Computed:    true,
				Description: "Workflow ID associated with this execution (from /api/jobs).",
			},
			"preview_output_json": schema.StringAttribute{
				Computed:    true,
				Description: "Preview output as JSON string (from /api/jobs).",
			},
			"preview_output": schema.DynamicAttribute{
				Computed:    true,
				Description: "Structured preview output (from /api/jobs).",
			},
			"outputs_json": schema.StringAttribute{
				Computed:    true,
				Description: "Full outputs as JSON string (from /api/jobs).",
			},
			"outputs_structured": schema.DynamicAttribute{
				Computed:    true,
				Description: "Structured outputs (from /api/jobs).",
			},
			"execution_status_json": schema.StringAttribute{
				Computed:    true,
				Description: "Execution status as JSON string (from /api/jobs).",
			},
			"execution_status": schema.DynamicAttribute{
				Computed:    true,
				Description: "Structured execution status (from /api/jobs).",
			},
			"execution_error_json": schema.StringAttribute{
				Computed:    true,
				Description: "Execution error as JSON string (from /api/jobs).",
			},
			"execution_error": schema.DynamicAttribute{
				Computed:    true,
				Description: "Structured execution error (from /api/jobs).",
			},
			"execution_workflow_json": schema.StringAttribute{
				Computed:    true,
				Description: "Execution workflow as JSON string (from /api/jobs).",
			},
			"execution_workflow": schema.DynamicAttribute{
				Computed:    true,
				Description: "Structured execution workflow (from /api/jobs).",
			},
		},
	}
}

func (r *WorkflowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkflowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkflowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(uuid.New().String())

	// Parse the workflow JSON
	prompt, err := r.parseWorkflow(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Workflow", err.Error())
		return
	}

	// Store assembled JSON
	if prompt != nil {
		jsonBytes, _ := json.MarshalIndent(prompt, "", "  ")
		data.AssembledJSON = types.StringValue(string(jsonBytes))
	}

	// Write file if output_file is set
	if !data.OutputFile.IsNull() && !data.OutputFile.IsUnknown() && data.OutputFile.ValueString() != "" {
		if err := r.writeWorkflowFile(ctx, data.OutputFile.ValueString(), prompt); err != nil {
			resp.Diagnostics.AddError("Failed to write workflow file", err.Error())
			return
		}
	}

	if !data.Execute.ValueBool() {
		data.PromptID = types.StringValue("")
		data.ValidationSummaryJSON = types.StringValue("")
		clearWorkflowExecutionFields(&data)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	r.executeWorkflow(ctx, prompt, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkflowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If we have a prompt_id, refresh status from the server
	if data.PromptID.ValueString() != "" && r.client != nil {
		if err := r.refreshWorkflowExecutionState(ctx, &data); err != nil {
			tflog.Warn(ctx, "Failed to refresh workflow status", map[string]interface{}{
				"prompt_id": data.PromptID.ValueString(),
				"error":     err.Error(),
			})
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WorkflowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	prompt, err := r.parseWorkflow(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Workflow", err.Error())
		return
	}

	// Store assembled JSON
	if prompt != nil {
		jsonBytes, _ := json.MarshalIndent(prompt, "", "  ")
		data.AssembledJSON = types.StringValue(string(jsonBytes))
	}

	// Write file if output_file is set
	if !data.OutputFile.IsNull() && !data.OutputFile.IsUnknown() && data.OutputFile.ValueString() != "" {
		if err := r.writeWorkflowFile(ctx, data.OutputFile.ValueString(), prompt); err != nil {
			resp.Diagnostics.AddError("Failed to write workflow file", err.Error())
			return
		}
	}

	if !data.Execute.ValueBool() {
		data.PromptID = types.StringValue("")
		data.ValidationSummaryJSON = types.StringValue("")
		clearWorkflowExecutionFields(&data)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	r.executeWorkflow(ctx, prompt, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkflowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.cancelWorkflowOnDelete(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Failed to cancel workflow on delete", err.Error())
		return
	}

	// Clean up output file if it was written
	if !data.OutputFile.IsNull() && data.OutputFile.ValueString() != "" {
		os.Remove(data.OutputFile.ValueString())
		tflog.Info(ctx, "Removed workflow file", map[string]interface{}{"path": data.OutputFile.ValueString()})
	}
}

// parseWorkflow extracts the prompt map from workflow_json or assembles it
// from registered virtual node resources referenced by node_ids.
func (r *WorkflowResource) parseWorkflow(ctx context.Context, data WorkflowModel) (map[string]interface{}, error) {
	if !data.WorkflowJSON.IsNull() && !data.WorkflowJSON.IsUnknown() && data.WorkflowJSON.ValueString() != "" {
		var prompt map[string]interface{}
		if err := json.Unmarshal([]byte(data.WorkflowJSON.ValueString()), &prompt); err != nil {
			return nil, fmt.Errorf("workflow_json must be valid JSON: %w", err)
		}

		if len(prompt) == 0 {
			return nil, fmt.Errorf("workflow_json must contain at least one node")
		}

		return prompt, nil
	}

	if data.NodeIDs.IsNull() || data.NodeIDs.IsUnknown() {
		return nil, fmt.Errorf("either workflow_json or node_ids must be provided")
	}

	nodeIDs, diags := listValueToStrings(ctx, data.NodeIDs)
	if diags.HasError() {
		return nil, fmt.Errorf("node_ids must be a list of strings")
	}

	assembled, err := AssembleWorkflowFromNodeIDs(nodeIDs)
	if err != nil {
		return nil, err
	}

	return assembled.Prompt, nil
}

// executeWorkflow queues the prompt and optionally waits for completion.
func (r *WorkflowResource) executeWorkflow(ctx context.Context, prompt map[string]interface{}, data *WorkflowModel, diags *diag.Diagnostics) {
	if r.client == nil {
		diags.AddError("Client Not Configured", "The ComfyUI client is not available. Ensure the provider is properly configured.")
		return
	}

	if validationEnabled(data.ValidateBeforeExecute) {
		report, err := r.validatePromptForExecution(prompt)
		if err != nil {
			clearWorkflowExecutionFields(data)
			data.PromptID = types.StringValue("")
			diags.AddError("Unable to validate workflow", err.Error())
			return
		}

		summaryJSON, err := report.JSON()
		if err != nil {
			clearWorkflowExecutionFields(data)
			data.PromptID = types.StringValue("")
			diags.AddError("Unable to encode validation summary", err.Error())
			return
		}
		data.ValidationSummaryJSON = types.StringValue(summaryJSON)

		if !report.Valid {
			clearWorkflowExecutionFields(data)
			data.PromptID = types.StringValue("")
			joinedErrors := strings.Join(report.Errors, "\n- ")
			diags.AddError("Workflow validation failed", fmt.Sprintf("The workflow failed semantic validation:\n- %s", joinedErrors))
			return
		}
	} else {
		data.ValidationSummaryJSON = types.StringValue("")
	}

	tflog.Info(ctx, "Submitting workflow to ComfyUI", map[string]interface{}{
		"node_count": len(prompt),
	})

	partialTargets := []string{}
	if !data.PartialTargets.IsNull() && !data.PartialTargets.IsUnknown() {
		targets, targetDiags := listValueToStrings(ctx, data.PartialTargets)
		diags.Append(targetDiags...)
		if diags.HasError() {
			return
		}
		partialTargets = targets
	}

	queueReq, err := buildQueuePromptRequest(prompt, workflowExecutionRequestConfig{
		PromptID:                stringValue(data.PromptID),
		ClientID:                stringValue(data.ClientID),
		ExtraDataJSON:           stringValue(data.ExtraDataJSON),
		PartialExecutionTargets: partialTargets,
	})
	if err != nil {
		clearWorkflowExecutionFields(data)
		data.PromptID = types.StringValue("")
		diags.AddError("Failed to prepare prompt request", err.Error())
		return
	}

	queueResp, err := r.client.QueuePrompt(queueReq)
	if err != nil {
		clearWorkflowExecutionFields(data)
		data.PromptID = types.StringValue("")
		diags.AddError("Failed to queue workflow", err.Error())
		return
	}

	data.PromptID = types.StringValue(queueResp.PromptID)
	clearWorkflowExecutionFields(data)

	tflog.Info(ctx, "Workflow queued", map[string]interface{}{
		"prompt_id": queueResp.PromptID,
	})

	if !data.WaitForCompletion.ValueBool() {
		return
	}

	timeout := time.Duration(data.TimeoutSeconds.ValueInt64()) * time.Second
	tflog.Info(ctx, "Waiting for workflow completion", map[string]interface{}{
		"prompt_id": queueResp.PromptID,
		"timeout":   timeout.String(),
	})

	entry, err := r.client.WaitForCompletion(queueResp.PromptID, timeout)
	if err != nil {
		clearWorkflowExecutionFields(data)
		diags.AddError("Failed to wait for workflow completion", err.Error())
		return
	}

	r.updateFromHistoryEntry(data, entry)

	// Chunk 3: Also try to fetch job data for richer fields
	if job, err := r.client.GetJob(queueResp.PromptID); err == nil && job != nil {
		if err := applyJobStateToWorkflowModel(data, job); err != nil {
			tflog.Warn(ctx, "Failed to apply job state after completion", map[string]interface{}{
				"prompt_id": queueResp.PromptID,
				"error":     err.Error(),
			})
		}
	}
}

func (r *WorkflowResource) validatePromptForExecution(prompt map[string]interface{}) (validation.Report, error) {
	rawPrompt, err := json.Marshal(prompt)
	if err != nil {
		return validation.Report{}, fmt.Errorf("marshal prompt for validation: %w", err)
	}

	parsedPrompt, err := artifacts.ParsePromptJSON(string(rawPrompt))
	if err != nil {
		return validation.Report{}, fmt.Errorf("parse prompt for validation: %w", err)
	}

	nodeInfo, err := r.client.GetObjectInfo()
	if err != nil {
		return validation.Report{}, err
	}

	return validation.ValidatePrompt(parsedPrompt, nodeInfo, validation.Options{RequireOutputNode: true}), nil
}

func validationEnabled(value types.Bool) bool {
	return value.IsNull() || value.IsUnknown() || value.ValueBool()
}

func buildQueuePromptRequest(prompt map[string]interface{}, config workflowExecutionRequestConfig) (client.QueuePromptRequest, error) {
	request := client.QueuePromptRequest{
		Prompt:                  prompt,
		PromptID:                config.PromptID,
		ClientID:                config.ClientID,
		PartialExecutionTargets: append([]string(nil), config.PartialExecutionTargets...),
	}

	extraData := map[string]interface{}{}
	if config.ExtraDataJSON != "" {
		if err := json.Unmarshal([]byte(config.ExtraDataJSON), &extraData); err != nil {
			return client.QueuePromptRequest{}, fmt.Errorf("extra_data_json must be valid JSON: %w", err)
		}
	}

	extraPNGInfo, _ := extraData["extra_pnginfo"].(map[string]interface{})
	if extraPNGInfo == nil {
		extraPNGInfo = map[string]interface{}{}
	}
	if _, ok := extraPNGInfo["prompt"]; !ok {
		extraPNGInfo["prompt"] = prompt
	}
	if len(extraPNGInfo) > 0 {
		extraData["extra_pnginfo"] = extraPNGInfo
	}
	request.ExtraData = extraData

	return request, nil
}

func (r *WorkflowResource) updateFromHistoryEntry(data *WorkflowModel, entry *client.HistoryEntry) {
	outputsValue, err := jsonCompatibleValue(entry.Outputs)
	if err != nil {
		data.OutputsJSON = types.StringNull()
		data.OutputsStructured = types.DynamicNull()
	} else if outputsValue == nil {
		// Preserve existing rich output state when history omits outputs entirely.
	} else {
		outputsJSON, err := json.Marshal(outputsValue)
		if err != nil {
			data.OutputsJSON = types.StringNull()
			data.OutputsStructured = types.DynamicNull()
		} else {
			encoded := string(outputsJSON)
			data.OutputsJSON = types.StringValue(encoded)
			if dynamicOutputs, dynamicErr := workflowDynamicFromAny(outputsValue); dynamicErr == nil {
				data.OutputsStructured = dynamicOutputs
			} else {
				data.OutputsStructured = types.DynamicNull()
			}

			if outputsCount, countErr := historyOutputsCount(outputsValue); countErr == nil {
				data.OutputsCount = types.Int64Value(outputsCount)
			}
		}
	}

	if len(entry.Prompt) >= 4 {
		if extraData, ok := entry.Prompt[3].(map[string]interface{}); ok {
			if createTime, ok := historyInt64FromAny(extraData["create_time"]); ok {
				data.CreateTime = types.Int64Value(createTime)
			}
			if extraPNGInfo, ok := extraData["extra_pnginfo"].(map[string]interface{}); ok {
				if workflow, ok := extraPNGInfo["workflow"].(map[string]interface{}); ok {
					if workflowID, ok := workflow["id"].(string); ok && workflowID != "" {
						data.WorkflowID = types.StringValue(workflowID)
					}
				}
			}
			workflowPayload := map[string]interface{}{
				"prompt":     entry.Prompt[2],
				"extra_data": extraData,
			}
			if workflowJSON := marshalExecutionJSON(workflowPayload, ""); !workflowJSON.IsNull() {
				data.ExecutionWorkflowJSON = workflowJSON
				if workflowValue, workflowErr := workflowDynamicFromAny(workflowPayload); workflowErr == nil {
					data.ExecutionWorkflow = workflowValue
				}
			}
		}
	}

	startTime, endTime, executionError := historyExecutionEvents(entry.Status.Messages)
	if startTime != 0 {
		data.ExecutionStartTime = types.Int64Value(startTime)
	}
	if endTime != 0 {
		data.ExecutionEndTime = types.Int64Value(endTime)
	}
	if len(executionError) > 0 {
		data.ExecutionErrorJSON = marshalExecutionJSON(executionError, "")
		if executionErrorValue, executionErrorErr := workflowDynamicFromAny(executionError); executionErrorErr == nil {
			data.ExecutionError = executionErrorValue
		}
	}

	if statusValue, statusValueErr := jsonCompatibleValue(entry.Status); statusValueErr == nil {
		if dynamicStatus, dynamicErr := workflowDynamicFromAny(statusValue); dynamicErr == nil {
			data.ExecutionStatus = dynamicStatus
		} else {
			data.ExecutionStatus = types.DynamicNull()
		}
	}

	statusJSON, statusErr := json.Marshal(entry.Status)
	if statusErr == nil {
		data.ExecutionStatusJSON = types.StringValue(string(statusJSON))
	} else {
		data.ExecutionStatusJSON = types.StringNull()
		data.ExecutionStatus = types.DynamicNull()
	}
}

// writeWorkflowFile creates parent directories and writes the prompt as JSON.
func (r *WorkflowResource) writeWorkflowFile(ctx context.Context, filePath string, prompt map[string]interface{}) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	jsonBytes, err := json.MarshalIndent(prompt, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling workflow JSON: %w", err)
	}

	if err := os.WriteFile(filePath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", filePath, err)
	}

	tflog.Info(ctx, "Wrote workflow JSON file", map[string]interface{}{"path": filePath})
	return nil
}

func (r *WorkflowResource) refreshWorkflowExecutionState(ctx context.Context, data *WorkflowModel) error {
	if r.client == nil || data.PromptID.IsNull() || data.PromptID.ValueString() == "" {
		return nil
	}

	promptID := data.PromptID.ValueString()
	var historyErr error
	historyApplied := false

	history, err := r.client.GetHistory(promptID)
	if err != nil {
		historyErr = err
	} else if entry, ok := (*history)[promptID]; ok {
		r.updateFromHistoryEntry(data, &entry)
		historyApplied = true
	}

	job, err := r.client.GetJob(promptID)
	if err == nil && job != nil {
		if err := applyJobStateToWorkflowModel(data, job); err != nil {
			return fmt.Errorf("apply job state: %w", err)
		}
		return nil
	}

	if historyApplied {
		return nil
	}
	if err != nil {
		return err
	}
	return historyErr
}

func (r *WorkflowResource) cancelWorkflowOnDelete(ctx context.Context, data *WorkflowModel) error {
	if r.client == nil || data.PromptID.IsNull() || data.PromptID.ValueString() == "" {
		return nil
	}

	var job *client.Job
	if j, err := r.client.GetJob(data.PromptID.ValueString()); err == nil {
		job = j
	} else if shouldUseExecutionStatusFallbackOnDelete(err) {
		tflog.Warn(ctx, "Failed to refresh workflow state before delete; falling back to stored status", map[string]interface{}{
			"prompt_id": data.PromptID.ValueString(),
			"error":     err.Error(),
		})
	}

	action := determineDeleteAction(data, job)
	switch action {
	case deleteActionRemoveFromQueue:
		tflog.Info(ctx, "Removing queued workflow from queue", map[string]interface{}{
			"prompt_id": data.PromptID.ValueString(),
		})
		if err := r.client.DeleteQueuedPrompt(data.PromptID.ValueString()); err != nil {
			return fmt.Errorf("remove queued workflow %q: %w", data.PromptID.ValueString(), err)
		}
	case deleteActionInterrupt:
		tflog.Info(ctx, "Interrupting running workflow", map[string]interface{}{
			"prompt_id": data.PromptID.ValueString(),
		})
		if err := r.client.InterruptPrompt(data.PromptID.ValueString()); err != nil {
			return fmt.Errorf("interrupt workflow %q: %w", data.PromptID.ValueString(), err)
		}
	}

	return nil
}

// Chunk 3: Delete actions
type deleteAction int

const (
	deleteActionNoop deleteAction = iota
	deleteActionRemoveFromQueue
	deleteActionInterrupt
)

// determineDeleteAction decides what cancellation action to take on delete
func determineDeleteAction(data *WorkflowModel, job *client.Job) deleteAction {
	// Only cancel if explicitly requested
	if data.CancelOnDelete.IsNull() || !data.CancelOnDelete.ValueBool() {
		return deleteActionNoop
	}

	// Need a prompt ID to cancel
	if data.PromptID.IsNull() || data.PromptID.ValueString() == "" {
		return deleteActionNoop
	}

	status := ""
	if job != nil {
		status = job.Status
	} else {
		status = storedExecutionStatus(data)
	}

	if status == "" {
		return deleteActionNoop
	}

	switch status {
	case "pending", "queued":
		return deleteActionRemoveFromQueue
	case "running", "executing":
		return deleteActionInterrupt
	case "completed", "error", "failed", "cancelled":
		return deleteActionNoop
	default:
		return deleteActionNoop
	}
}

func jsonCompatibleValue(value interface{}) (interface{}, error) {
	if isNilLikeValue(value) {
		return nil, nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var normalized interface{}
	if err := json.Unmarshal(data, &normalized); err != nil {
		return nil, err
	}

	return normalized, nil
}

func isUnexpectedHTTPStatus(err error, statusCode int) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), fmt.Sprintf("unexpected status %d", statusCode))
}

func shouldUseExecutionStatusFallbackOnDelete(err error) bool {
	if err == nil {
		return false
	}

	if !isUnexpectedHTTPStatus(err, http.StatusNotFound) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "page not found")
}

func storedExecutionStatus(data *WorkflowModel) string {
	if data == nil || data.ExecutionStatusJSON.IsNull() || data.ExecutionStatusJSON.IsUnknown() {
		return ""
	}

	var decoded client.ExecutionStatus
	if err := json.Unmarshal([]byte(data.ExecutionStatusJSON.ValueString()), &decoded); err != nil {
		return ""
	}

	return decoded.StatusStr
}

func historyInt64FromAny(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

func historyExecutionEvents(messages [][]interface{}) (int64, int64, map[string]interface{}) {
	var startTime int64
	var endTime int64
	var executionError map[string]interface{}

	for _, message := range messages {
		if len(message) < 2 {
			continue
		}

		eventName, _ := message[0].(string)
		eventData, _ := message[1].(map[string]interface{})
		if eventData == nil {
			continue
		}

		switch eventName {
		case "execution_start":
			startTime, _ = historyInt64FromAny(eventData["timestamp"])
		case "execution_success", "execution_error", "execution_interrupted":
			endTime, _ = historyInt64FromAny(eventData["timestamp"])
			if eventName == "execution_error" {
				executionError = eventData
			}
		}
	}

	return startTime, endTime, executionError
}

func historyOutputsCount(outputs interface{}) (int64, error) {
	if outputs == nil {
		return 0, nil
	}

	raw, err := json.Marshal(outputs)
	if err != nil {
		return 0, err
	}

	var decoded map[string]map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return 0, err
	}

	var count int64
	for _, nodeOutputs := range decoded {
		for mediaType, items := range nodeOutputs {
			if mediaType == "animated" {
				continue
			}
			list, ok := items.([]interface{})
			if !ok {
				continue
			}
			count += int64(len(list))
		}
	}

	return count, nil
}

func isNilLikeValue(value interface{}) bool {
	if value == nil {
		return true
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Map, reflect.Slice, reflect.Pointer, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}

func applyJobStateToWorkflowModel(data *WorkflowModel, job *client.Job) error {
	if job == nil {
		return nil
	}

	if job.CreateTime != nil {
		data.CreateTime = int64ValueOrNull(*job.CreateTime)
	}
	if job.ExecutionStartTime != nil {
		data.ExecutionStartTime = int64ValueOrNull(*job.ExecutionStartTime)
	}
	if job.ExecutionEndTime != nil {
		data.ExecutionEndTime = int64ValueOrNull(*job.ExecutionEndTime)
	}
	if job.OutputsCount != nil {
		data.OutputsCount = types.Int64Value(int64(*job.OutputsCount))
	}
	if job.WorkflowID != "" {
		data.WorkflowID = stringValueOrNull(job.WorkflowID)
	}

	var err error
	if !isNilLikeValue(job.PreviewOutput) {
		if data.PreviewOutputJSON = marshalExecutionJSON(job.PreviewOutput, ""); !data.PreviewOutputJSON.IsNull() {
			data.PreviewOutput, err = workflowDynamicFromAny(job.PreviewOutput)
			if err != nil {
				return fmt.Errorf("preview_output: %w", err)
			}
		}
	}

	if !isNilLikeValue(job.Outputs) {
		if data.OutputsJSON = marshalExecutionJSON(job.Outputs, ""); !data.OutputsJSON.IsNull() {
			data.OutputsStructured, err = workflowDynamicFromAny(job.Outputs)
			if err != nil {
				return fmt.Errorf("outputs: %w", err)
			}
		} else {
			data.OutputsStructured = types.DynamicNull()
		}
	}

	if !isNilLikeValue(job.ExecutionStatus) {
		if data.ExecutionStatusJSON = marshalExecutionJSON(job.ExecutionStatus, ""); !data.ExecutionStatusJSON.IsNull() {
			data.ExecutionStatus, err = workflowDynamicFromAny(job.ExecutionStatus)
			if err != nil {
				return fmt.Errorf("execution_status: %w", err)
			}
		}
	}

	if len(job.ExecutionError) > 0 {
		if data.ExecutionErrorJSON = marshalExecutionJSON(job.ExecutionError, ""); !data.ExecutionErrorJSON.IsNull() {
			data.ExecutionError, err = workflowDynamicFromAny(job.ExecutionError)
			if err != nil {
				return fmt.Errorf("execution_error: %w", err)
			}
		}
	}

	if job.Workflow != nil {
		workflowPayload := map[string]interface{}{
			"prompt":     job.Workflow.Prompt,
			"extra_data": job.Workflow.ExtraData,
		}
		data.ExecutionWorkflowJSON = marshalExecutionJSON(workflowPayload, "")
		data.ExecutionWorkflow, err = workflowDynamicFromAny(workflowPayload)
		if err != nil {
			return fmt.Errorf("execution_workflow: %w", err)
		}
	}

	return nil
}

func clearWorkflowExecutionFields(data *WorkflowModel) {
	data.CreateTime = types.Int64Null()
	data.ExecutionStartTime = types.Int64Null()
	data.ExecutionEndTime = types.Int64Null()
	data.OutputsCount = types.Int64Value(0)
	data.WorkflowID = types.StringNull()
	data.PreviewOutputJSON = types.StringNull()
	data.PreviewOutput = types.DynamicNull()
	data.OutputsJSON = types.StringNull()
	data.OutputsStructured = types.DynamicNull()
	data.ExecutionStatusJSON = types.StringNull()
	data.ExecutionStatus = types.DynamicNull()
	data.ExecutionErrorJSON = types.StringNull()
	data.ExecutionError = types.DynamicNull()
	data.ExecutionWorkflowJSON = types.StringNull()
	data.ExecutionWorkflow = types.DynamicNull()
}

func int64ValueOrNull(value int64) types.Int64 {
	if value == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(value)
}

func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func marshalExecutionJSON(value interface{}, emptyFallback string) types.String {
	if isNilLikeValue(value) {
		if emptyFallback == "" {
			return types.StringNull()
		}
		return types.StringValue(emptyFallback)
	}

	data, err := json.Marshal(value)
	if err != nil {
		if emptyFallback == "" {
			return types.StringNull()
		}
		return types.StringValue(emptyFallback)
	}

	return types.StringValue(string(data))
}

func workflowDynamicFromAny(data interface{}) (types.Dynamic, error) {
	if data == nil {
		return types.DynamicNull(), nil
	}

	attrValue, err := workflowAttrValueFromAny(data)
	if err != nil {
		return types.DynamicNull(), err
	}

	return types.DynamicValue(attrValue), nil
}

func workflowAttrValueFromAny(data interface{}) (attr.Value, error) {
	if data == nil {
		return types.DynamicNull(), nil
	}

	switch v := data.(type) {
	case map[string]interface{}:
		if v == nil {
			return types.DynamicNull(), nil
		}

		attrTypes := make(map[string]attr.Type, len(v))
		attrValues := make(map[string]attr.Value, len(v))
		for key, item := range v {
			itemValue, err := workflowAttrValueFromAny(item)
			if err != nil {
				return nil, fmt.Errorf("key %q: %w", key, err)
			}
			attrTypes[key] = itemValue.Type(context.Background())
			attrValues[key] = itemValue
		}

		return types.ObjectValueMust(attrTypes, attrValues), nil
	case []interface{}:
		if v == nil {
			return types.DynamicNull(), nil
		}

		attrTypes := make([]attr.Type, 0, len(v))
		attrValues := make([]attr.Value, 0, len(v))
		for idx, item := range v {
			itemValue, err := workflowAttrValueFromAny(item)
			if err != nil {
				return nil, fmt.Errorf("index %d: %w", idx, err)
			}
			attrTypes = append(attrTypes, itemValue.Type(context.Background()))
			attrValues = append(attrValues, itemValue)
		}

		return types.TupleValueMust(attrTypes, attrValues), nil
	case string:
		return types.StringValue(v), nil
	case bool:
		return types.BoolValue(v), nil
	case float64:
		return basetypes.NewNumberValue(big.NewFloat(v)), nil
	case int:
		return basetypes.NewNumberValue(new(big.Float).SetInt64(int64(v))), nil
	case int64:
		return basetypes.NewNumberValue(new(big.Float).SetInt64(v)), nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}
