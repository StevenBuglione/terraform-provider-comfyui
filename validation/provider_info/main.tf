terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.1"
    }
  }
}

provider "comfyui" {}

data "comfyui_provider_info" "current" {}

output "provider_version" {
  value = data.comfyui_provider_info.current.provider_version
}

output "comfyui_version" {
  value = data.comfyui_provider_info.current.comfyui_version
}

output "node_count" {
  value = data.comfyui_provider_info.current.node_count
}

output "extracted_at" {
  value = data.comfyui_provider_info.current.extracted_at
}
