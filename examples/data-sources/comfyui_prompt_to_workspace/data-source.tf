data "comfyui_prompt_to_workspace" "example" {
  name = "translated-workspace"
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
        filename_prefix = "translated_prompt"
      }
    }
  })
}

output "workspace_fidelity" {
  value = data.comfyui_prompt_to_workspace.example.fidelity
}
