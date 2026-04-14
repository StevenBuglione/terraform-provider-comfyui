terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.18"
    }
  }
}

provider "comfyui" {}

resource "comfyui_ksampler" "invalid" {
  model        = "00000000-0000-0000-0000-000000000001:0"
  seed         = 42
  steps        = 0
  cfg          = -1
  sampler_name = "euler"
  scheduler    = "normal"
  positive     = "00000000-0000-0000-0000-000000000002:0"
  negative     = "00000000-0000-0000-0000-000000000003:0"
  latent_image = "00000000-0000-0000-0000-000000000004:0"
  denoise      = 1.0
}
