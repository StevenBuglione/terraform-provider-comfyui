data "comfyui_prompt_json" "example" {
  json = jsonencode({
    "1" = {
      class_type = "EmptyImage"
      inputs = {
        width      = 512
        height     = 512
        batch_size = 1
      }
    }
  })
}

output "normalized_prompt_json" {
  value = data.comfyui_prompt_json.example.normalized_json
}
