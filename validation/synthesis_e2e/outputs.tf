output "prompt_synthesis" {
  value = {
    fidelity           = data.comfyui_prompt_to_terraform.prompt.fidelity
    terraform_hcl      = data.comfyui_prompt_to_terraform.prompt.terraform_hcl
    terraform_ir_json  = data.comfyui_prompt_to_terraform.prompt.terraform_ir_json
    synthesized_fields = data.comfyui_prompt_to_terraform.prompt.synthesized_fields
  }
}

output "workspace_synthesis" {
  value = {
    fidelity               = data.comfyui_workspace_to_terraform.workspace.fidelity
    terraform_hcl          = data.comfyui_workspace_to_terraform.workspace.terraform_hcl
    terraform_ir_json      = data.comfyui_workspace_to_terraform.workspace.terraform_ir_json
    translated_prompt_json = data.comfyui_workspace_to_terraform.workspace.translated_prompt_json
  }
}
