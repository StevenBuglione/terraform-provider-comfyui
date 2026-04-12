data "comfyui_workspace_json" "example" {
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

output "normalized_workspace_json" {
  value = data.comfyui_workspace_json.example.normalized_json
}
