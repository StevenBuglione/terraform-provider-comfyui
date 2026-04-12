data "comfyui_workspace_to_terraform" "example" {
  name = "workspace-synthesis"
  workspace_json = jsonencode({
    version = 1
    id      = "workspace-example"
    name    = "Workspace Example"
    nodes   = []
    links   = []
    groups  = []
    config  = {}
    extra   = {}
    definitions = {
      subgraphs = []
    }
  })
}

output "terraform_ir_json" {
  value = data.comfyui_workspace_to_terraform.example.terraform_ir_json
}
