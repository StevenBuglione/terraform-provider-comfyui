data "comfyui_prompt_validation" "example" {
  mode = "fragment"
  json = jsonencode({
    "1" = {
      class_type = "LoadImage"
      inputs = {
        image = "example.png"
      }
    }
  })
}

output "prompt_valid" {
  value = data.comfyui_prompt_validation.example.valid
}
