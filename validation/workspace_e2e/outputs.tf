output "workspace_files" {
  value = {
    for name, workspace in comfyui_workspace.fixture : name => workspace.output_file
  }
}

output "workspace_workflow_counts" {
  value = {
    for name, workspace in comfyui_workspace.fixture : name => workspace.workflow_count
  }
}

output "workspace_json_lengths" {
  value = {
    for name, workspace in comfyui_workspace.fixture : name => length(workspace.workspace_json)
  }
}

output "workspace_payloads" {
  value = {
    for name, workspace in comfyui_workspace.fixture : name => workspace.workspace_json
  }
}
