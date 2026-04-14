terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.18"
    }
  }
}

provider "comfyui" {}

locals {
  prompt_json = jsonencode({
    "1" = {
      class_type = "LoadImage"
      inputs = {
        image = "input.png"
      }
    }
    "2" = {
      class_type = "SaveImage"
      inputs = {
        images          = ["1", 0]
        filename_prefix = "ComfyUI"
      }
    }
  })

  workspace_json = jsonencode({
    nodes = [
      {
        id   = 1
        type = "LoadImage"
        inputs = [
          {
            name   = "image"
            type   = "STRING"
            widget = { name = "image" }
            link   = null
          }
        ]
        outputs = [
          {
            name  = "IMAGE"
            type  = "IMAGE"
            links = [1]
          }
        ]
        widgets_values = ["input.png"]
      },
      {
        id   = 2
        type = "SaveImage"
        inputs = [
          {
            name = "images"
            type = "IMAGE"
            link = 1
          },
          {
            name   = "filename_prefix"
            type   = "STRING"
            widget = { name = "filename_prefix" }
            link   = null
          }
        ]
        widgets_values = ["ComfyUI"]
      }
    ]
    links = [
      [1, 1, 0, 2, 0, "IMAGE"]
    ]
  })
}

data "comfyui_prompt_to_terraform" "prompt" {
  prompt_json = local.prompt_json
}

data "comfyui_workspace_to_terraform" "workspace" {
  workspace_json = local.workspace_json
}
