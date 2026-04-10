resource "comfyui_workflow" "example" {
  workflow_json = jsonencode({
    "1" = {
      class_type = "KSampler"
      inputs = {
        seed         = 42
        steps        = 20
        cfg          = 7.0
        sampler_name = "euler"
        scheduler    = "normal"
        denoise      = 1.0
        model        = ["2", 0]
        positive     = ["3", 0]
        negative     = ["4", 0]
        latent_image = ["5", 0]
      }
    }
  })
}
