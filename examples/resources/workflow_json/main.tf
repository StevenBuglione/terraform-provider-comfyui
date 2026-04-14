# Raw Workflow JSON Example
#
# Instead of composing individual node resources, you can submit a
# pre-built ComfyUI API-format JSON directly via comfyui_workflow.
#
# This is useful when:
#   - Exporting a workflow from the ComfyUI web UI ("Save (API Format)")
#   - Reusing workflows from community templates
#   - Migrating existing pipelines without rewriting them as Terraform resources

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
  description = "Checkpoint filename"
  type        = string
  default     = "v1-5-pruned-emaonly.safetensors"
}

variable "positive_prompt" {
  description = "Text prompt for image generation"
  type        = string
  default     = "a beautiful landscape, mountains, sunset, photorealistic, 8k"
}

variable "negative_prompt" {
  description = "Negative prompt"
  type        = string
  default     = "blurry, low quality, watermark"
}

variable "seed" {
  description = "Random seed"
  type        = number
  default     = 12345
}

# ---------------------------------------------------------------------------
# Workflow — raw JSON submitted directly to the ComfyUI API
# ---------------------------------------------------------------------------

# This JSON mirrors the ComfyUI API format: each key is a node ID,
# and the value contains `class_type` and `inputs`.
# Links between nodes use the format ["node_id", output_slot_index].
resource "comfyui_workflow" "from_json" {
  workflow_json = jsonencode({
    "1" = {
      class_type = "CheckpointLoaderSimple"
      inputs = {
        ckpt_name = var.checkpoint
      }
    }
    "2" = {
      class_type = "CLIPTextEncode"
      inputs = {
        text = var.positive_prompt
        clip = ["1", 1]
      }
    }
    "3" = {
      class_type = "CLIPTextEncode"
      inputs = {
        text = var.negative_prompt
        clip = ["1", 1]
      }
    }
    "4" = {
      class_type = "EmptyLatentImage"
      inputs = {
        width      = 512
        height     = 512
        batch_size = 1
      }
    }
    "5" = {
      class_type = "KSampler"
      inputs = {
        model        = ["1", 0]
        seed         = var.seed
        steps        = 20
        cfg          = 7.0
        sampler_name = "euler"
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
        filename_prefix = "workflow_json"
      }
    }
  })

  execute             = true
  wait_for_completion = true
  timeout_seconds     = 300
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

output "prompt_id" {
  description = "ComfyUI prompt ID for this execution"
  value       = comfyui_workflow.from_json.prompt_id
}

output "workflow_id" {
  description = "Workflow identifier embedded in execution metadata"
  value       = comfyui_workflow.from_json.workflow_id
}

output "outputs_json" {
  description = "JSON string of execution outputs"
  value       = comfyui_workflow.from_json.outputs_json
}

output "execution_status_json" {
  description = "Structured execution status payload"
  value       = comfyui_workflow.from_json.execution_status_json
}

output "execution_error_json" {
  description = "Structured execution error payload when execution fails"
  value       = comfyui_workflow.from_json.execution_error_json
}
