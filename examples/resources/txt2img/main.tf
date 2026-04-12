# Text-to-Image Workflow
#
# Generates an image from a text prompt using Stable Diffusion.
#
# Pipeline:
#   CheckpointLoaderSimple → CLIPTextEncode (pos/neg)
#                          → EmptyLatentImage → KSampler → VAEDecode → SaveImage

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
# Variables
# ---------------------------------------------------------------------------

variable "checkpoint" {
  description = "Stable Diffusion checkpoint filename (must exist in ComfyUI models/checkpoints/)"
  type        = string
  default     = "v1-5-pruned-emaonly.safetensors"
}

variable "positive_prompt" {
  description = "Text describing the desired image"
  type        = string
  default     = "a beautiful sunset over mountains, dramatic lighting, 8k, detailed"
}

variable "negative_prompt" {
  description = "Text describing what to avoid in the image"
  type        = string
  default     = "blurry, low quality, watermark, text"
}

variable "width" {
  description = "Image width in pixels"
  type        = number
  default     = 512
}

variable "height" {
  description = "Image height in pixels"
  type        = number
  default     = 512
}

variable "seed" {
  description = "Random seed for reproducibility (0 = random)"
  type        = number
  default     = 42
}

variable "steps" {
  description = "Number of sampling steps"
  type        = number
  default     = 20
}

variable "cfg_scale" {
  description = "Classifier-free guidance scale"
  type        = number
  default     = 7.0
}

# ---------------------------------------------------------------------------
# Node Resources (virtual — stored in state only, no API calls)
# ---------------------------------------------------------------------------

# Load the Stable Diffusion checkpoint — outputs model, clip, and vae
resource "comfyui_checkpoint_loader_simple" "checkpoint" {
  ckpt_name = var.checkpoint
}

# Encode the positive (desired) prompt
resource "comfyui_clip_text_encode" "positive" {
  text = var.positive_prompt
  clip = comfyui_checkpoint_loader_simple.checkpoint.clip_output
}

# Encode the negative (undesired) prompt
resource "comfyui_clip_text_encode" "negative" {
  text = var.negative_prompt
  clip = comfyui_checkpoint_loader_simple.checkpoint.clip_output
}

# Create an empty latent image at the desired resolution
resource "comfyui_empty_latent_image" "latent" {
  width      = var.width
  height     = var.height
  batch_size = 1
}

# Run the sampler — the core diffusion step
resource "comfyui_ksampler" "sampler" {
  model        = comfyui_checkpoint_loader_simple.checkpoint.model_output
  seed         = var.seed
  steps        = var.steps
  cfg          = var.cfg_scale
  sampler_name = "euler"
  scheduler    = "normal"
  positive     = comfyui_clip_text_encode.positive.conditioning_output
  negative     = comfyui_clip_text_encode.negative.conditioning_output
  latent_image = comfyui_empty_latent_image.latent.latent_output
  denoise      = 1.0
}

# Decode the latent image back to pixel space
resource "comfyui_vae_decode" "decode" {
  samples = comfyui_ksampler.sampler.latent_output
  vae     = comfyui_checkpoint_loader_simple.checkpoint.vae_output
}

# Save the resulting image
resource "comfyui_save_image" "output" {
  images          = comfyui_vae_decode.decode.image_output
  filename_prefix = "txt2img"
}

# ---------------------------------------------------------------------------
# Workflow Execution
# ---------------------------------------------------------------------------

# Assemble all nodes and execute the workflow on the ComfyUI server
resource "comfyui_workflow" "txt2img" {
  node_ids = [
    comfyui_checkpoint_loader_simple.checkpoint.id,
    comfyui_clip_text_encode.positive.id,
    comfyui_clip_text_encode.negative.id,
    comfyui_empty_latent_image.latent.id,
    comfyui_ksampler.sampler.id,
    comfyui_vae_decode.decode.id,
    comfyui_save_image.output.id,
  ]

  execute             = true
  wait_for_completion = true
  timeout_seconds     = 300
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

output "prompt_id" {
  description = "ComfyUI prompt ID for this execution"
  value       = comfyui_workflow.txt2img.prompt_id
}

output "workflow_id" {
  description = "Workflow identifier embedded in execution metadata"
  value       = comfyui_workflow.txt2img.workflow_id
}

output "outputs_json" {
  description = "JSON string of execution outputs (image filenames, etc.)"
  value       = comfyui_workflow.txt2img.outputs_json
}

output "execution_status_json" {
  description = "Structured execution status payload"
  value       = comfyui_workflow.txt2img.execution_status_json
}
