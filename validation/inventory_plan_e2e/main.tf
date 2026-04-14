terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.18"
    }
  }
}

variable "comfyui_host" {
  type    = string
  default = "127.0.0.1"
}

variable "comfyui_port" {
  type    = number
  default = 8188
}

variable "checkpoint_name" {
  type = string
}

provider "comfyui" {
  host = var.comfyui_host
  port = var.comfyui_port
}

data "comfyui_inventory" "live" {
  kinds = ["checkpoints"]
}

data "comfyui_node_schema" "checkpoint_loader" {
  node_type = "CheckpointLoaderSimple"
}

resource "comfyui_checkpoint_loader_simple" "selected" {
  ckpt_name = var.checkpoint_name
}
