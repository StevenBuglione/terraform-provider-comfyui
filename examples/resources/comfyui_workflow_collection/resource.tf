locals {
  invert_workflow_json = jsonencode({
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
        filename_prefix = "invert_example"
      }
    }
  })

  blur_workflow_json = jsonencode({
    "1" = {
      class_type = "LoadImage"
      inputs = {
        image = "example.png"
      }
    }
    "2" = {
      class_type = "ImageBlur"
      inputs = {
        image = ["1", 0]
        blur_radius = 3
      }
    }
    "3" = {
      class_type = "SaveImage"
      inputs = {
        images          = ["2", 0]
        filename_prefix = "blur_example"
      }
    }
  })
}

resource "comfyui_workflow" "invert" {
  name                    = "invert"
  workflow_json           = local.invert_workflow_json
  execute                 = false
  wait_for_completion     = false
  validate_before_execute = false
}

resource "comfyui_workflow" "blur" {
  name                    = "blur"
  workflow_json           = local.blur_workflow_json
  execute                 = false
  wait_for_completion     = false
  validate_before_execute = false
}

resource "comfyui_workflow_collection" "example" {
  name        = "example-collection"
  description = "Small prompt workflows grouped into one exported index."
  workflows = [
    comfyui_workflow.invert.id,
    comfyui_workflow.blur.id,
  ]
  output_dir = "${path.module}/collection"
}
