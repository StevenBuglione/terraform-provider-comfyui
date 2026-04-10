data "comfyui_system_stats" "example" {}

output "gpu_name" {
  value = data.comfyui_system_stats.example.devices
}
