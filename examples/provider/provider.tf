# ComfyUI Provider Configuration
#
# The provider connects to a running ComfyUI server instance.
# All connection parameters can be set via environment variables:
#   COMFYUI_HOST                               — bare host or full URL (default: "localhost")
#   COMFYUI_PORT                               — server port when host is not a full URL (default: 8188)
#   COMFYUI_API_KEY                            — optional API key for authenticated servers
#   COMFYUI_COMFY_ORG_AUTH_TOKEN               — optional partner auth token for partner-backed nodes
#   COMFYUI_COMFY_ORG_API_KEY                  — optional partner API key for partner-backed nodes
#   COMFYUI_DEFAULT_WORKFLOW_EXTRA_DATA_JSON   — optional default workflow extra_data JSON
#   COMFYUI_UNSUPPORTED_DYNAMIC_VALIDATION_MODE — error | warning | ignore

terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.18"
    }
  }
}

# Minimal: relies entirely on environment variables / defaults
provider "comfyui" {}

# Explicit: useful for multi-server setups or CI/CD
# provider "comfyui" {
#   host                                = var.comfyui_host
#   port                                = var.comfyui_port
#   api_key                             = var.comfyui_api_key
#   comfy_org_auth_token                = var.comfyui_comfy_org_auth_token
#   comfy_org_api_key                   = var.comfyui_comfy_org_api_key
#   unsupported_dynamic_validation_mode = "warning"
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

variable "comfyui_comfy_org_auth_token" {
  description = "Partner auth token for comfy_org-backed execution (leave empty if not required)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "comfyui_comfy_org_api_key" {
  description = "Partner API key for comfy_org-backed execution (leave empty if not required)"
  type        = string
  default     = ""
  sensitive   = true
}
