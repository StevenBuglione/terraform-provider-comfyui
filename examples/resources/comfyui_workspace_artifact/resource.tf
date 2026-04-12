locals {
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

resource "comfyui_workspace_artifact" "example" {
  path         = "${path.module}/workspace.json"
  content_json = local.workspace_json
}
