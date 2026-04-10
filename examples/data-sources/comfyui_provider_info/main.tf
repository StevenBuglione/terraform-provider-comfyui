# Retrieve provider version and compatibility information
data "comfyui_provider_info" "current" {}

# Use version info in outputs for visibility
output "provider_version" {
  description = "Current provider version"
  value       = data.comfyui_provider_info.current.provider_version
}

output "comfyui_version" {
  description = "ComfyUI version this provider was built for"
  value       = data.comfyui_provider_info.current.comfyui_version
}

output "supported_nodes" {
  description = "Number of ComfyUI node resources available"
  value       = data.comfyui_provider_info.current.node_count
}

output "extraction_timestamp" {
  description = "When node specs were extracted from ComfyUI"
  value       = data.comfyui_provider_info.current.extracted_at
}
