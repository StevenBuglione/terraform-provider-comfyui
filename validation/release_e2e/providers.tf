terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.1"
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

provider "comfyui" {
  host = var.comfyui_host
  port = var.comfyui_port
}
