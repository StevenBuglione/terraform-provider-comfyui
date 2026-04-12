# Place a local input.png next to this example before applying it.
resource "comfyui_uploaded_image" "input" {
  file_path = "${path.module}/input.png"
  filename  = "workflow-example-input.png"
  overwrite = true
  type      = "input"
}

resource "comfyui_workflow" "example" {
  workflow_json = jsonencode({
    "1" = {
      class_type = "LoadImage"
      inputs = {
        image = comfyui_uploaded_image.input.filename
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
        filename_prefix = "workflow_example"
      }
    }
  })

  extra_data_json = jsonencode({
    extra_pnginfo = {
      workflow = {
        id = "workflow-example"
      }
    }
  })

  execute             = true
  wait_for_completion = true
  cancel_on_delete    = true
  timeout_seconds     = 120
}

output "workflow_prompt_id" {
  value = comfyui_workflow.example.prompt_id
}

output "workflow_id" {
  value = comfyui_workflow.example.workflow_id
}

output "workflow_outputs_count" {
  value = comfyui_workflow.example.outputs_count
}

output "workflow_preview_output_json" {
  value = comfyui_workflow.example.preview_output_json
}

output "workflow_execution_status_json" {
  value = comfyui_workflow.example.execution_status_json
}
