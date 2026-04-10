# ComfyUI Provider Configuration
#
# The provider connects to a running ComfyUI server instance.
# All connection parameters can be set via environment variables:
#   COMFYUI_HOST    — server hostname (default: "localhost")
#   COMFYUI_PORT    — server port     (default: 8188)
#   COMFYUI_API_KEY — optional API key for authenticated servers

terraform {
  required_providers {
    comfyui = {
      source  = "registry.terraform.io/StevenBuglione/comfyui"
      version = "~> 0.1"
    }
  }
}

# Minimal: relies entirely on environment variables / defaults
provider "comfyui" {}

# Explicit: useful for multi-server setups or CI/CD
# provider "comfyui" {
#   host    = var.comfyui_host
#   port    = var.comfyui_port
#   api_key = var.comfyui_api_key
# }

variable "comfyui_host" {
  description = "Hostname of the ComfyUI server"
  type        = string
  default     = "localhost"
}

variable "comfyui_port" {
  description = "Port of the ComfyUI server"
  type        = number
  default     = 8188
}

variable "comfyui_api_key" {
  description = "API key for ComfyUI authentication (leave empty if not required)"
  type        = string
  default     = ""
  sensitive   = true
}
