# Compose API-format workflow JSON into a deterministic ComfyUI workspace export.
locals {
  workspace_prompt_json = jsonencode({
    "1" = {
      class_type = "LoadImage"
      inputs = {
        image = "example.png"
      }
    }
    "2" = {
      class_type = "ImageInvert"
      inputs = {
        image = ["1", 0]
      }
    }
    "3" = {
      class_type = "SaveImage"
      inputs = {
        images          = ["2", 0]
        filename_prefix = "workspace_example"
      }
    }
  })
}

resource "comfyui_workspace" "example" {
  name = "example-workspace"

  workflows = [
    {
      name          = "invert-image"
      workflow_json = local.workspace_prompt_json
      style = {
        group_color    = "#3b82f6"
        title_font_size = 20
      }
    }
  ]

  layout = {
    display  = "flex"
    direction = "row"
    gap      = 240
    origin_x = 80
    origin_y = 80
  }

  node_layout = {
    mode       = "dag"
    direction  = "left_to_right"
    column_gap = 180
    row_gap    = 90
  }

  output_file = "${path.module}/workspace.json"
}
