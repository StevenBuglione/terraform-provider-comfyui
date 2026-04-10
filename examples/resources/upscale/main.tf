# Image Upscale Workflow
#
# Upscales an image using a dedicated upscale model (e.g., RealESRGAN).
#
# Pipeline:
#   LoadImage → ImageUpscaleWithModel (via UpscaleModelLoader) → SaveImage

terraform {
  required_providers {
    comfyui = {
      source  = "sbuglione/comfyui"
      version = "~> 0.1"
    }
  }
}

provider "comfyui" {}

# ---------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------

variable "input_image" {
  description = "Image filename to upscale (must exist in ComfyUI input/ directory)"
  type        = string
  default     = "example.png"
}

variable "upscale_model" {
  description = "Upscale model filename (must exist in ComfyUI models/upscale_models/)"
  type        = string
  default     = "RealESRGAN_x4plus.pth"
}

# ---------------------------------------------------------------------------
# Node Resources
# ---------------------------------------------------------------------------

# Load the source image
resource "comfyui_load_image" "input" {
  image = var.input_image
}

# Load the upscale model (e.g., RealESRGAN, SwinIR)
resource "comfyui_upscale_model_loader" "upscaler" {
  model_name = var.upscale_model
}

# Apply the upscale model to the image
resource "comfyui_image_upscale_with_model" "upscale" {
  upscale_model = comfyui_upscale_model_loader.upscaler.upscale_model_output
  image         = comfyui_load_image.input.image_output
}

# Save the upscaled image
resource "comfyui_save_image" "output" {
  images          = comfyui_image_upscale_with_model.upscale.image_output
  filename_prefix = "upscaled"
}

# ---------------------------------------------------------------------------
# Workflow Execution
# ---------------------------------------------------------------------------

resource "comfyui_workflow" "upscale" {
  node_ids = [
    comfyui_load_image.input.id,
    comfyui_upscale_model_loader.upscaler.id,
    comfyui_image_upscale_with_model.upscale.id,
    comfyui_save_image.output.id,
  ]

  execute             = true
  wait_for_completion = true
  timeout_seconds     = 600
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

output "prompt_id" {
  description = "ComfyUI prompt ID for this execution"
  value       = comfyui_workflow.upscale.prompt_id
}

output "status" {
  description = "Workflow execution status"
  value       = comfyui_workflow.upscale.status
}

output "outputs" {
  description = "JSON string of execution outputs"
  value       = comfyui_workflow.upscale.outputs
}
