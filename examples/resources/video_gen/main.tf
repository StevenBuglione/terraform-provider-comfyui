# Video Generation Workflow
#
# Generates a video from a text prompt using partner node APIs.
# This example shows two approaches:
#   1. Kling Text-to-Video  — simple text-to-video generation
#   2. ByteDance Seedance   — text-to-video with more options
#
# Both use the SaveVideo node to write output to the ComfyUI output directory.
#
# Prerequisites:
#   - ComfyUI server running with the ComfyUI-API-Manager custom node installed
#   - Kling API key set via the API Manager UI (or KLING_API_KEY env var)
#   - ByteDance API key set via the API Manager UI (or BYTEDANCE_API_KEY env var)
#   - COMFYUI_HOST / COMFYUI_PORT env vars (defaults: localhost / 8188)

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

variable "prompt" {
  description = "Text describing the desired video content"
  type        = string
  default     = "A golden retriever running through a sunlit meadow, slow motion, cinematic"
}

variable "negative_prompt" {
  description = "Text describing what to avoid in the video"
  type        = string
  default     = "blurry, low quality, watermark, text, distorted"
}

variable "aspect_ratio" {
  description = "Video aspect ratio"
  type        = string
  default     = "16:9"
}

# ---------------------------------------------------------------------------
# Approach 1 — Kling Text-to-Video
# ---------------------------------------------------------------------------
# Kling's text-to-video node generates video from a prompt.
# Requires a Kling API key configured in the ComfyUI API Manager.

resource "comfyui_kling_text_to_video_node" "kling_video" {
  prompt          = var.prompt
  negative_prompt = var.negative_prompt
  cfg_scale       = 0.7 # Guidance scale (0–1)
  aspect_ratio    = var.aspect_ratio
  mode            = "std" # "std" (standard) or "pro" (higher quality, slower)
}

# Save the Kling-generated video to disk
resource "comfyui_save_video" "kling_output" {
  video           = comfyui_kling_text_to_video_node.kling_video.video_output
  filename_prefix = "video/kling"
  format          = "auto"
  codec           = "auto"
}

# Execute the Kling workflow
resource "comfyui_workflow" "kling_video_gen" {
  node_ids = [
    comfyui_kling_text_to_video_node.kling_video.id,
    comfyui_save_video.kling_output.id,
  ]

  execute             = true
  wait_for_completion = true
  timeout_seconds     = 600 # Videos take longer than images
}

# ---------------------------------------------------------------------------
# Approach 2 — ByteDance Seedance Text-to-Video
# ---------------------------------------------------------------------------
# ByteDance's Seedance model offers resolution/duration controls and
# optional audio generation. Requires a ByteDance API key.

resource "comfyui_byte_dance_text_to_video_node" "seedance_video" {
  model          = "seedance-1-0-pro-fast-251015" # Fast variant
  prompt         = var.prompt
  resolution     = "720p" # "480p", "720p", or "1080p"
  aspect_ratio   = var.aspect_ratio
  duration       = 5     # Video length in seconds (3–12)
  seed           = 42    # 0 = random
  camera_fixed   = false # Lock camera movement
  watermark      = false # Add provider watermark
  generate_audio = false # Auto-generate matching audio
}

# Save the Seedance-generated video to disk
resource "comfyui_save_video" "seedance_output" {
  video           = comfyui_byte_dance_text_to_video_node.seedance_video.video_output
  filename_prefix = "video/seedance"
  format          = "auto"
  codec           = "auto"
}

# Execute the ByteDance workflow
resource "comfyui_workflow" "seedance_video_gen" {
  node_ids = [
    comfyui_byte_dance_text_to_video_node.seedance_video.id,
    comfyui_save_video.seedance_output.id,
  ]

  execute             = true
  wait_for_completion = true
  timeout_seconds     = 600
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

output "kling_prompt_id" {
  description = "ComfyUI prompt ID for the Kling video execution"
  value       = comfyui_workflow.kling_video_gen.prompt_id
}

output "kling_status" {
  description = "Kling workflow execution status payload"
  value       = comfyui_workflow.kling_video_gen.execution_status_json
}

output "seedance_prompt_id" {
  description = "ComfyUI prompt ID for the Seedance video execution"
  value       = comfyui_workflow.seedance_video_gen.prompt_id
}

output "seedance_status" {
  description = "Seedance workflow execution status payload"
  value       = comfyui_workflow.seedance_video_gen.execution_status_json
}
