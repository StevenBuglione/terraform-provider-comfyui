# Workflow File Output
#
# Demonstrates the output_file attribute for writing ComfyUI-loadable JSON to disk.
#
# Three modes of operation:
#   1. File-only   — output_file set, execute = false  → writes JSON, no server call
#   2. Execute-only — output_file omitted, execute = true → runs on server, no file
#   3. Both         — output_file set, execute = true  → writes file AND runs on server
#
# In every mode, assembled_json is populated so you can reference the workflow
# JSON in other resources or outputs without needing file output.

terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.18"
    }
  }
}

provider "comfyui" {}

# ---------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------

variable "checkpoint" {
  description = "Stable Diffusion checkpoint filename (must exist in ComfyUI models/checkpoints/)"
  type        = string
  default     = "v1-5-pruned-emaonly.safetensors"
}

variable "output_dir" {
  description = "Directory where workflow JSON files are written"
  type        = string
  default     = "workflows"
}

# ---------------------------------------------------------------------------
# Locals — reusable workflow definition
# ---------------------------------------------------------------------------

locals {
  # A complete txt2img workflow in ComfyUI API format.
  # Each top-level key is a node ID; the object contains class_type and inputs.
  landscape_workflow = {
    "1" = {
      class_type = "CheckpointLoaderSimple"
      inputs = {
        ckpt_name = var.checkpoint
      }
    }
    "2" = {
      class_type = "CLIPTextEncode"
      inputs = {
        text = "a sweeping mountain landscape at golden hour, dramatic clouds, 8k, detailed"
        clip = ["1", 1]
      }
    }
    "3" = {
      class_type = "CLIPTextEncode"
      inputs = {
        text = "blurry, low quality, watermark, text"
        clip = ["1", 1]
      }
    }
    "4" = {
      class_type = "EmptyLatentImage"
      inputs = {
        width      = 768
        height     = 512
        batch_size = 1
      }
    }
    "5" = {
      class_type = "KSampler"
      inputs = {
        model        = ["1", 0]
        seed         = 42
        steps        = 25
        cfg          = 7.5
        sampler_name = "euler_ancestral"
        scheduler    = "normal"
        positive     = ["2", 0]
        negative     = ["3", 0]
        latent_image = ["4", 0]
        denoise      = 1.0
      }
    }
    "6" = {
      class_type = "VAEDecode"
      inputs = {
        samples = ["5", 0]
        vae     = ["1", 2]
      }
    }
    "7" = {
      class_type = "SaveImage"
      inputs = {
        images          = ["6", 0]
        filename_prefix = "landscape"
      }
    }
  }
}

# ---------------------------------------------------------------------------
# Mode 1: File-only — export without executing
# ---------------------------------------------------------------------------
# Writes a ComfyUI-loadable JSON file to disk. The file can be opened directly
# in ComfyUI's "Load" dialog or fed to other tooling. No server connection is
# needed because execute = false skips the prompt queue entirely.
# Status will be "file_only" after apply.

resource "comfyui_workflow" "exportable" {
  name        = "landscape-preset"
  description = "Standard landscape generation preset — file export only"
  tags        = ["landscape", "sd15", "preset"]
  category    = "txt2img"

  workflow_json = jsonencode(local.landscape_workflow)

  # Write the assembled JSON to disk — creates parent directories automatically
  output_file = "${var.output_dir}/landscape-preset.json"

  # Don't execute — just produce the file
  execute = false
}

# ---------------------------------------------------------------------------
# Mode 3: Both — write file AND execute on the server
# ---------------------------------------------------------------------------
# Useful when you want a local copy for version control or sharing while also
# running the workflow immediately. The file is written first; execution
# follows only if the file write succeeds.

resource "comfyui_workflow" "live_run" {
  name        = "landscape-live"
  description = "Generate a landscape and save the workflow JSON for reuse"
  tags        = ["landscape", "sd15", "live"]
  category    = "txt2img"

  workflow_json = jsonencode(local.landscape_workflow)

  # Save a copy of the workflow for reuse or version control
  output_file = "${var.output_dir}/landscape-live.json"

  # Also submit to ComfyUI for execution
  execute             = true
  wait_for_completion = true
  timeout_seconds     = 300
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

# assembled_json is always populated regardless of output_file or execute.
# Use it to inspect the workflow or pass it to other resources.
output "exportable_assembled_json" {
  description = "The assembled workflow JSON for the file-only resource"
  value       = comfyui_workflow.exportable.assembled_json
}

output "live_run_prompt_id" {
  description = "ComfyUI prompt ID for the executed workflow"
  value       = comfyui_workflow.live_run.prompt_id
}

output "live_run_workflow_id" {
  description = "Workflow identifier embedded in execution metadata"
  value       = comfyui_workflow.live_run.workflow_id
}

output "live_run_outputs_json" {
  description = "JSON string of execution outputs (image filenames, etc.)"
  value       = comfyui_workflow.live_run.outputs_json
}

output "live_run_execution_status_json" {
  description = "Structured execution status of the live workflow"
  value       = comfyui_workflow.live_run.execution_status_json
}

output "live_run_file_path" {
  description = "Path where the live workflow JSON was written"
  value       = comfyui_workflow.live_run.output_file
}
