data "comfyui_workspace_validation" "example" {
  mode = "fragment"
  json = jsonencode({
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

output "workspace_valid" {
  value = data.comfyui_workspace_validation.example.valid
}
