locals {
  subgraph_json = jsonencode({
    version = 1
    id      = "subgraph-example"
    name    = "Subgraph Example"
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

resource "comfyui_subgraph" "example" {
  path = "${path.module}/subgraph.json"
  json = local.subgraph_json
}
