# Workflow Collection Example
#
# Demonstrates organizing multiple ComfyUI workflows into
# a labeled collection with metadata and an index manifest.
#
# Collections help manage large numbers of workflows by
# grouping them into logical categories with searchable metadata.

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
# Workflow 1: Simple txt2img landscape
# ---------------------------------------------------------------------------

resource "comfyui_workflow" "landscape" {
  workflow_json = jsonencode({
    "1" = {
      class_type = "CheckpointLoaderSimple"
      inputs     = { ckpt_name = "v1-5-pruned-emaonly.safetensors" }
    }
    "2" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "a sweeping mountain landscape, golden hour", clip = ["1", 1] }
    }
    "3" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "blurry, watermark", clip = ["1", 1] }
    }
    "4" = {
      class_type = "EmptyLatentImage"
      inputs     = { width = 768, height = 512, batch_size = 1 }
    }
    "5" = {
      class_type = "KSampler"
      inputs = {
        model        = ["1", 0]
        seed         = 42
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
      inputs     = { samples = ["5", 0], vae = ["1", 2] }
    }
    "7" = {
      class_type = "SaveImage"
      inputs     = { images = ["6", 0], filename_prefix = "landscape" }
    }
  })

  execute = false # Define only — do not execute
}

# ---------------------------------------------------------------------------
# Workflow 2: Simple txt2img portrait
# ---------------------------------------------------------------------------

resource "comfyui_workflow" "portrait" {
  workflow_json = jsonencode({
    "1" = {
      class_type = "CheckpointLoaderSimple"
      inputs     = { ckpt_name = "v1-5-pruned-emaonly.safetensors" }
    }
    "2" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "studio portrait, soft lighting, detailed face", clip = ["1", 1] }
    }
    "3" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "blurry, low quality", clip = ["1", 1] }
    }
    "4" = {
      class_type = "EmptyLatentImage"
      inputs     = { width = 512, height = 768, batch_size = 1 }
    }
    "5" = {
      class_type = "KSampler"
      inputs = {
        model        = ["1", 0]
        seed         = 123
        steps        = 25
        cfg          = 7.5
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
      inputs     = { samples = ["5", 0], vae = ["1", 2] }
    }
    "7" = {
      class_type = "SaveImage"
      inputs     = { images = ["6", 0], filename_prefix = "portrait" }
    }
  })

  execute = false
}

# ---------------------------------------------------------------------------
# Collection grouping both workflows
# ---------------------------------------------------------------------------

resource "comfyui_workflow_collection" "sd15_basics" {
  name        = "sd15-collection"
  description = "Standard SD 1.5 generation workflows for landscape and portrait use cases"
  output_dir  = "workflows/sd15"

  workflows = [
    comfyui_workflow.landscape.id,
    comfyui_workflow.portrait.id,
  ]
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

output "collection_index" {
  description = "JSON index manifest of all workflows in the collection"
  value       = comfyui_workflow_collection.sd15_basics.index_json
}

output "workflow_count" {
  description = "Number of workflows in the collection"
  value       = comfyui_workflow_collection.sd15_basics.workflow_count
}
