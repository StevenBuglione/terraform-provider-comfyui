package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	ID                types.String `tfsdk:"id"`
	WorkflowJSON      types.String `tfsdk:"workflow_json"`
	NodeIDs           types.List   `tfsdk:"node_ids"`
	Execute           types.Bool   `tfsdk:"execute"`
	WaitForCompletion types.Bool   `tfsdk:"wait_for_completion"`
	TimeoutSeconds    types.Int64  `tfsdk:"timeout_seconds"`
	PromptID          types.String `tfsdk:"prompt_id"`
	Status            types.String `tfsdk:"status"`
	Outputs           types.String `tfsdk:"outputs"`
	Error             types.String `tfsdk:"error"`

	// Metadata
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Tags        types.List   `tfsdk:"tags"`
	Category    types.String `tfsdk:"category"`

	// File output
	OutputFile types.String `tfsdk:"output_file"`

	// Computed
	AssembledJSON types.String `tfsdk:"assembled_json"`
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
				Description: "List of node resource IDs to include in the workflow. Reserved for future use with virtual node resources.",
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
			"prompt_id": schema.StringAttribute{
				Computed:    true,
				Description: "ComfyUI prompt ID assigned after submission.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Execution status: pending, queued, running, completed, or error.",
			},
			"outputs": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string of execution outputs (images, audio, etc.).",
			},
			"error": schema.StringAttribute{
				Computed:    true,
				Description: "Error message if execution failed.",
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
		if !data.OutputFile.IsNull() && !data.OutputFile.IsUnknown() && data.OutputFile.ValueString() != "" {
			data.Status = types.StringValue("file_only")
		} else {
			data.Status = types.StringValue("pending")
		}
		data.Outputs = types.StringValue("{}")
		data.Error = types.StringValue("")
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
		history, err := r.client.GetHistory(data.PromptID.ValueString())
		if err != nil {
			tflog.Warn(ctx, "Failed to refresh workflow status", map[string]interface{}{
				"prompt_id": data.PromptID.ValueString(),
				"error":     err.Error(),
			})
		} else if entry, ok := (*history)[data.PromptID.ValueString()]; ok {
			r.updateFromHistoryEntry(&data, &entry)
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
		if !data.OutputFile.IsNull() && !data.OutputFile.IsUnknown() && data.OutputFile.ValueString() != "" {
			data.Status = types.StringValue("file_only")
		} else {
			data.Status = types.StringValue("pending")
		}
		data.Outputs = types.StringValue("{}")
		data.Error = types.StringValue("")
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	r.executeWorkflow(ctx, prompt, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowResource) Delete(ctx context.Context, req resource.DeleteRequest, _ *resource.DeleteResponse) {
	var data WorkflowModel
	req.State.Get(ctx, &data)

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

	tflog.Info(ctx, "Submitting workflow to ComfyUI", map[string]interface{}{
		"node_count": len(prompt),
	})

	queueResp, err := r.client.QueuePrompt(prompt)
	if err != nil {
		data.PromptID = types.StringValue("")
		data.Status = types.StringValue("error")
		data.Outputs = types.StringValue("{}")
		data.Error = types.StringValue(fmt.Sprintf("Failed to queue prompt: %s", err.Error()))
		return
	}

	data.PromptID = types.StringValue(queueResp.PromptID)
	data.Status = types.StringValue("queued")
	data.Outputs = types.StringValue("{}")
	data.Error = types.StringValue("")

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
		data.Status = types.StringValue("error")
		data.Error = types.StringValue(err.Error())
		return
	}

	r.updateFromHistoryEntry(data, entry)
}

func (r *WorkflowResource) updateFromHistoryEntry(data *WorkflowModel, entry *client.HistoryEntry) {
	if entry.Status.Completed {
		data.Status = types.StringValue("completed")
	} else {
		data.Status = types.StringValue(entry.Status.StatusStr)
	}

	outputsJSON, err := json.Marshal(entry.Outputs)
	if err != nil {
		data.Outputs = types.StringValue("{}")
	} else {
		data.Outputs = types.StringValue(string(outputsJSON))
	}

	data.Error = types.StringValue("")
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
