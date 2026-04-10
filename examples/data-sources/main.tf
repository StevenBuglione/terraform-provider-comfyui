# Data Sources Example
#
# Demonstrates all five ComfyUI data sources for querying server state.
# These are read-only — they fetch live information from the ComfyUI API.

terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.1"
    }
  }
}

provider "comfyui" {}

# ---------------------------------------------------------------------------
# 1. System Stats — server hardware and software info
# ---------------------------------------------------------------------------

data "comfyui_system_stats" "server" {}

output "comfyui_version" {
  description = "ComfyUI version running on the server"
  value       = data.comfyui_system_stats.server.comfyui_version
}

output "python_version" {
  description = "Python version on the server"
  value       = data.comfyui_system_stats.server.python_version
}

output "os" {
  description = "Server operating system"
  value       = data.comfyui_system_stats.server.os
}

output "gpu_devices" {
  description = "Available compute devices (GPU/CPU)"
  value       = data.comfyui_system_stats.server.devices
}

# ---------------------------------------------------------------------------
# 2. Queue — current execution queue status
# ---------------------------------------------------------------------------

data "comfyui_queue" "current" {}

output "queue_running" {
  description = "Number of currently executing workflows"
  value       = data.comfyui_queue.current.running_count
}

output "queue_pending" {
  description = "Number of workflows waiting to execute"
  value       = data.comfyui_queue.current.pending_count
}

# ---------------------------------------------------------------------------
# 3. Node Info — inspect a specific ComfyUI node type
# ---------------------------------------------------------------------------

variable "node_type" {
  description = "ComfyUI node class name to inspect"
  type        = string
  default     = "KSampler"
}

data "comfyui_node_info" "sampler" {
  node_type = var.node_type
}

output "node_display_name" {
  description = "Human-readable name of the node"
  value       = data.comfyui_node_info.sampler.display_name
}

output "node_category" {
  description = "Node category in the ComfyUI menu"
  value       = data.comfyui_node_info.sampler.category
}

output "node_inputs_required" {
  description = "JSON of required inputs for this node"
  value       = data.comfyui_node_info.sampler.input_required
}

output "node_output_types" {
  description = "Output types produced by this node"
  value       = data.comfyui_node_info.sampler.output_types
}

# ---------------------------------------------------------------------------
# 4. Workflow History — look up a past execution by prompt ID
# ---------------------------------------------------------------------------

variable "prompt_id" {
  description = "Prompt ID of a previous workflow execution to inspect"
  type        = string
  default     = ""
}

data "comfyui_workflow_history" "previous" {
  count     = var.prompt_id != "" ? 1 : 0
  prompt_id = var.prompt_id
}

output "history_status" {
  description = "Execution status of the looked-up workflow"
  value       = var.prompt_id != "" ? data.comfyui_workflow_history.previous[0].status : "no prompt_id provided"
}

output "history_outputs" {
  description = "Outputs from the looked-up workflow"
  value       = var.prompt_id != "" ? data.comfyui_workflow_history.previous[0].outputs : "no prompt_id provided"
}

# ---------------------------------------------------------------------------
# 5. Output — get download URL for a specific output file
# ---------------------------------------------------------------------------

variable "output_filename" {
  description = "Filename of an output image to look up"
  type        = string
  default     = ""
}

data "comfyui_output" "image" {
  count    = var.output_filename != "" ? 1 : 0
  filename = var.output_filename
}

output "output_url" {
  description = "Download URL for the output file"
  value       = var.output_filename != "" ? data.comfyui_output.image[0].url : "no filename provided"
}

output "output_exists" {
  description = "Whether the output file exists on the server"
  value       = var.output_filename != "" ? data.comfyui_output.image[0].exists : false
}
