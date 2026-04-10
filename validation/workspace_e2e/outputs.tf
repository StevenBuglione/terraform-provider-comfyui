locals {
  all_workspaces = merge(
    comfyui_workspace.fixture,
    comfyui_workspace.fixture_with_node_layout
  )
}

output "workspace_files" {
  value = {
    for name, workspace in local.all_workspaces : name => workspace.output_file
  }
}

output "workspace_workflow_counts" {
  value = {
    for name, workspace in local.all_workspaces : name => workspace.workflow_count
  }
}

output "workspace_json_lengths" {
  value = {
    for name, workspace in local.all_workspaces : name => length(workspace.workspace_json)
  }
}

output "workspace_payloads" {
  value = {
    for name, workspace in local.all_workspaces : name => workspace.workspace_json
  }
}
