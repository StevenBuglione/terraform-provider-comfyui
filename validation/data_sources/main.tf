terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.18"
    }
  }
}

provider "comfyui" {}

data "comfyui_system_stats" "server" {}

data "comfyui_queue" "current" {}

data "comfyui_node_info" "ksampler" {
  node_type = "KSampler"
}

output "comfyui_version" {
  value = data.comfyui_system_stats.server.comfyui_version
}

output "queue_pending" {
  value = data.comfyui_queue.current.pending_count
}

output "ksampler_category" {
  value = data.comfyui_node_info.ksampler.category
}
