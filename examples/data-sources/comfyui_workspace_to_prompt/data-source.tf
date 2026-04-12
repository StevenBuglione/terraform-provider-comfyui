data "comfyui_workspace_to_prompt" "example" {
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

output "prompt_fidelity" {
  value = data.comfyui_workspace_to_prompt.example.fidelity
}
