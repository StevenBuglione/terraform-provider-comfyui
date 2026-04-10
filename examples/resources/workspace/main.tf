# Workspace Export Example
#
# Demonstrates composing multiple API-format workflows into a single
# UI-oriented workspace/subgraph JSON export using the comfyui_workspace
# meta resource.
#
# Note: unlike comfyui_workflow file-only export, comfyui_workspace still
# needs a live ComfyUI connection so it can fetch node metadata from
# GET /object_info and build slot/widget information for the UI graph.

terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.1"
    }
  }
}

provider "comfyui" {}

locals {
  landscape_workflow = {
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
  }

  portrait_workflow = {
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
  }
}

resource "comfyui_workflow" "landscape" {
  workflow_json = jsonencode(local.landscape_workflow)
  execute       = false
}

resource "comfyui_workflow" "portrait" {
  workflow_json = jsonencode(local.portrait_workflow)
  execute       = false
}

resource "comfyui_workspace" "gallery" {
  name        = "sd15-gallery"
  output_file = "workspaces/sd15-gallery.json"

  layout = {
    display  = "grid"
    columns  = 2
    gap      = 320
    origin_x = 120
    origin_y = 80
  }

  workflows = [
    {
      name          = "Landscape"
      workflow_json = comfyui_workflow.landscape.assembled_json
    },
    {
      name          = "Portrait"
      workflow_json = comfyui_workflow.portrait.assembled_json
    }
  ]
}

output "workspace_json" {
  description = "The composed workspace/subgraph JSON"
  value       = comfyui_workspace.gallery.workspace_json
}

output "workspace_workflow_count" {
  description = "Number of workflow islands included in the export"
  value       = comfyui_workspace.gallery.workflow_count
}
