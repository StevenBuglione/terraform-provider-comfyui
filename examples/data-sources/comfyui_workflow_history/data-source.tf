data "comfyui_workflow_history" "recent" {
  prompt_id = "abc-123-def"
}

output "workflow_status" {
  value = data.comfyui_workflow_history.recent.status
}
