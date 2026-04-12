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

output "workspace_metrics" {
  value = {
    for name, workspace in local.all_workspaces : name => {
      node_count  = length(try(jsondecode(workspace.workspace_json).nodes, []))
      group_count = length(try(jsondecode(workspace.workspace_json).groups, []))
      link_count  = length(try(jsondecode(workspace.workspace_json).links, []))
      max_in_degree = max([
        for node in try(jsondecode(workspace.workspace_json).nodes, []) : sum(concat([0], [
          for input in try(node.inputs, []) : try(input.link, null) != null ? 1 : 0
        ]))
      ]...)
      max_out_degree = max([
        for node in try(jsondecode(workspace.workspace_json).nodes, []) : sum(concat([0], [
          for output in try(node.outputs, []) : length(try(output.links, []))
        ]))
      ]...)
    }
  }
}
