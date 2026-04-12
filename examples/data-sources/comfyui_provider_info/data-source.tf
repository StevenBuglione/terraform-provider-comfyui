data "comfyui_provider_info" "example" {}

output "provider_compatibility" {
  value = "Provider ${data.comfyui_provider_info.example.provider_version} for ComfyUI ${data.comfyui_provider_info.example.comfyui_version}"
}
