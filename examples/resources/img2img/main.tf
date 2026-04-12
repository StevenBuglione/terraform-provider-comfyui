# Image-to-Image Workflow
#
# Takes an existing image and transforms it guided by a text prompt.
# Uses a lower denoise value to preserve the original composition.
#
# Pipeline:
#   LoadImage → VAEEncode ─┐
#   CheckpointLoader ──────┤
#   CLIPTextEncode (pos) ──┤→ KSampler → VAEDecode → SaveImage
#   CLIPTextEncode (neg) ──┘

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
  description = "Stable Diffusion checkpoint filename"
  type        = string
  default     = "v1-5-pruned-emaonly.safetensors"
}

variable "input_image" {
  description = "Input image filename (must exist in ComfyUI input/ directory)"
  type        = string
  default     = "example.png"
}

variable "positive_prompt" {
  description = "Text describing the desired transformation"
  type        = string
  default     = "oil painting style, vibrant colors, masterpiece"
}

variable "negative_prompt" {
  description = "Text describing what to avoid"
  type        = string
  default     = "blurry, low quality, distorted"
}

variable "denoise" {
  description = "Denoise strength (0.0 = keep original, 1.0 = full regeneration)"
  type        = number
  default     = 0.6
}

variable "seed" {
  description = "Random seed for reproducibility"
  type        = number
  default     = 42
}

# ---------------------------------------------------------------------------
# Node Resources
# ---------------------------------------------------------------------------

# Load the input image from ComfyUI's input directory
resource "comfyui_load_image" "input" {
  image = var.input_image
}

# Load checkpoint for model, clip, and vae
resource "comfyui_checkpoint_loader_simple" "checkpoint" {
  ckpt_name = var.checkpoint
}

# Encode the input image into latent space using the checkpoint's VAE
resource "comfyui_vae_encode" "encode" {
  pixels = comfyui_load_image.input.image_output
  vae    = comfyui_checkpoint_loader_simple.checkpoint.vae_output
}

# Positive prompt conditioning
resource "comfyui_clip_text_encode" "positive" {
  text = var.positive_prompt
  clip = comfyui_checkpoint_loader_simple.checkpoint.clip_output
}

# Negative prompt conditioning
resource "comfyui_clip_text_encode" "negative" {
  text = var.negative_prompt
  clip = comfyui_checkpoint_loader_simple.checkpoint.clip_output
}

# Sample with lower denoise to blend the prompt with the original image
resource "comfyui_ksampler" "sampler" {
  model        = comfyui_checkpoint_loader_simple.checkpoint.model_output
  seed         = var.seed
  steps        = 20
  cfg          = 7.0
  sampler_name = "euler"
  scheduler    = "normal"
  positive     = comfyui_clip_text_encode.positive.conditioning_output
  negative     = comfyui_clip_text_encode.negative.conditioning_output
  latent_image = comfyui_vae_encode.encode.latent_output
  denoise      = var.denoise
}

# Decode back to pixel space
resource "comfyui_vae_decode" "decode" {
  samples = comfyui_ksampler.sampler.latent_output
  vae     = comfyui_checkpoint_loader_simple.checkpoint.vae_output
}

# Save the result
resource "comfyui_save_image" "output" {
  images          = comfyui_vae_decode.decode.image_output
  filename_prefix = "img2img"
}

# ---------------------------------------------------------------------------
# Workflow Execution
# ---------------------------------------------------------------------------

resource "comfyui_workflow" "img2img" {
  node_ids = [
    comfyui_load_image.input.id,
    comfyui_checkpoint_loader_simple.checkpoint.id,
    comfyui_vae_encode.encode.id,
    comfyui_clip_text_encode.positive.id,
    comfyui_clip_text_encode.negative.id,
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
  value       = comfyui_workflow.img2img.prompt_id
}

output "workflow_id" {
  description = "Workflow identifier embedded in execution metadata"
  value       = comfyui_workflow.img2img.workflow_id
}

output "outputs_json" {
  description = "JSON string of execution outputs"
  value       = comfyui_workflow.img2img.outputs_json
}
