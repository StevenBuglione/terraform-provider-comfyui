terraform {
  required_providers {
    comfyui = {
      source = "registry.terraform.io/StevenBuglione/comfyui"
    }
  }
}

# Configure the ComfyUI provider.
# Connection details can also be set via environment variables:
#   COMFYUI_HOST, COMFYUI_PORT, COMFYUI_API_KEY
provider "comfyui" {
  host = "localhost"
  port = 8188
}
