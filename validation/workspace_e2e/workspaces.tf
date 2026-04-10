resource "comfyui_workflow" "fixture" {
  for_each = local.workflow_definitions

  workflow_json = jsonencode(each.value)
  execute       = false
}

resource "comfyui_workspace" "fixture" {
  for_each = local.workspace_definitions

  name        = each.value.name
  output_file = "${path.module}/artifacts/generated/${each.key}.json"
  layout      = each.value.layout

  workflows = [
    for workflow_name in each.value.members : merge(
      {
        name          = title(replace(workflow_name, "_", " "))
        workflow_json = comfyui_workflow.fixture[workflow_name].assembled_json
      },
      lookup(lookup(each.value, "overrides", {}), workflow_name, {})
    )
  ]
}
