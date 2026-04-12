locals {
  raw_import_prompt = {
    "1" = {
      class_type = "CheckpointLoaderSimple"
      inputs     = { ckpt_name = "v1-5-pruned-emaonly.safetensors" }
    }
    "2" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "shared root prompt, cinematic lighting", clip = ["1", 1] }
    }
    "3" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "branch A: sunset atmosphere", clip = ["1", 1] }
    }
    "4" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "branch B: morning mist", clip = ["1", 1] }
    }
    "5" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "branch C: midday harsh light", clip = ["1", 1] }
    }
    "6" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "generic negative", clip = ["1", 1] }
    }
    "7" = {
      class_type = "ConditioningCombine"
      inputs     = { conditioning_1 = ["2", 0], conditioning_2 = ["3", 0] }
    }
    "8" = {
      class_type = "ConditioningCombine"
      inputs     = { conditioning_1 = ["7", 0], conditioning_2 = ["4", 0] }
    }
    "9" = {
      class_type = "ConditioningCombine"
      inputs     = { conditioning_1 = ["8", 0], conditioning_2 = ["5", 0] }
    }
    "10" = {
      class_type = "EmptyLatentImage"
      inputs     = { width = 512, height = 512, batch_size = 1 }
    }
    "11" = {
      class_type = "KSampler"
      inputs = {
        model        = ["1", 0]
        seed         = 1111
        steps        = 24
        cfg          = 7.5
        sampler_name = "euler"
        scheduler    = "normal"
        positive     = ["9", 0]
        negative     = ["6", 0]
        latent_image = ["10", 0]
        denoise      = 1.0
      }
    }
    "12" = {
      class_type = "VAEDecode"
      inputs     = { samples = ["11", 0], vae = ["1", 2] }
    }
    "13" = {
      class_type = "SaveImage"
      inputs     = { images = ["12", 0], filename_prefix = "release_raw_import" }
    }
  }

  gallery_companion_prompt = {
    "1" = {
      class_type = "CheckpointLoaderSimple"
      inputs     = { ckpt_name = "v1-5-pruned-emaonly.safetensors" }
    }
    "2" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "industrial sci-fi corridor, moody volumetrics", clip = ["1", 1] }
    }
    "3" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "bad hands", clip = ["1", 1] }
    }
    "4" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "low quality", clip = ["1", 1] }
    }
    "5" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "artifacts", clip = ["1", 1] }
    }
    "6" = {
      class_type = "CLIPTextEncode"
      inputs     = { text = "washed out highlights", clip = ["1", 1] }
    }
    "7" = {
      class_type = "EmptyLatentImage"
      inputs     = { width = 832, height = 512, batch_size = 1 }
    }
    "8" = {
      class_type = "KSampler"
      inputs = {
        model        = ["1", 0]
        seed         = 707
        steps        = 34
        cfg          = 8.2
        sampler_name = "euler"
        scheduler    = "normal"
        positive     = ["2", 0]
        negative     = ["3", 0]
        latent_image = ["7", 0]
        denoise      = 1.0
      }
    }
    "9" = {
      class_type = "KSampler"
      inputs = {
        model        = ["1", 0]
        seed         = 808
        steps        = 34
        cfg          = 8.2
        sampler_name = "euler"
        scheduler    = "normal"
        positive     = ["2", 0]
        negative     = ["4", 0]
        latent_image = ["7", 0]
        denoise      = 1.0
      }
    }
    "10" = {
      class_type = "KSampler"
      inputs = {
        model        = ["1", 0]
        seed         = 909
        steps        = 34
        cfg          = 8.2
        sampler_name = "euler"
        scheduler    = "normal"
        positive     = ["2", 0]
        negative     = ["5", 0]
        latent_image = ["7", 0]
        denoise      = 1.0
      }
    }
    "11" = {
      class_type = "KSampler"
      inputs = {
        model        = ["1", 0]
        seed         = 1001
        steps        = 34
        cfg          = 8.2
        sampler_name = "euler"
        scheduler    = "normal"
        positive     = ["2", 0]
        negative     = ["6", 0]
        latent_image = ["7", 0]
        denoise      = 1.0
      }
    }
    "12" = {
      class_type = "VAEDecode"
      inputs     = { samples = ["8", 0], vae = ["1", 2] }
    }
    "13" = {
      class_type = "SaveImage"
      inputs     = { images = ["12", 0], filename_prefix = "release_gallery_companion" }
    }
  }
}
