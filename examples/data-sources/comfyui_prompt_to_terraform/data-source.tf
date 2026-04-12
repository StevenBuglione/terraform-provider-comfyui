data "comfyui_prompt_to_terraform" "example" {
  name = "synthesized-workflow"
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
        filename_prefix = "synthesized_prompt"
      }
    }
  })
}

output "terraform_hcl" {
  value = data.comfyui_prompt_to_terraform.example.terraform_hcl
}
