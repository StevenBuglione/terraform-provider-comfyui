locals {
  prompt_json = jsonencode({
    "1" = {
      class_type = "LoadImage"
      inputs = {
        image = "example.png"
      }
    }
    "2" = {
      class_type = "SaveImage"
      inputs = {
        images          = ["1", 0]
        filename_prefix = "artifact_example"
      }
    }
  })
}

resource "comfyui_prompt_artifact" "example" {
  path         = "${path.module}/prompt.json"
  content_json = local.prompt_json
}
