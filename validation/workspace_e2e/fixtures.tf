locals {
  workflow_definitions = {
    compact_reference = {
      "1" = {
        class_type = "CheckpointLoaderSimple"
        inputs     = { ckpt_name = "v1-5-pruned-emaonly.safetensors" }
      }
      "2" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "reference subject, centered, clean background", clip = ["1", 1] }
      }
      "3" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "blurry, noisy", clip = ["1", 1] }
      }
      "4" = {
        class_type = "EmptyLatentImage"
        inputs     = { width = 512, height = 512, batch_size = 1 }
      }
      "5" = {
        class_type = "KSampler"
        inputs = {
          model        = ["1", 0]
          seed         = 7
          steps        = 20
          cfg          = 7.0
          sampler_name = "euler"
          scheduler    = "normal"
          positive     = ["2", 0]
          negative     = ["3", 0]
          latent_image = ["4", 0]
          denoise      = 1.0
        }
      }
      "6" = {
        class_type = "VAEDecode"
        inputs     = { samples = ["5", 0], vae = ["1", 2] }
      }
      "7" = {
        class_type = "SaveImage"
        inputs     = { images = ["6", 0], filename_prefix = "compact_reference" }
      }
    }

    branchy_landscape = {
      "1" = {
        class_type = "CheckpointLoaderSimple"
        inputs     = { ckpt_name = "v1-5-pruned-emaonly.safetensors" }
      }
      "2" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "panoramic mountain vista, cinematic, ultra detailed", clip = ["1", 1] }
      }
      "3" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "storm clouds over alpine lake", clip = ["1", 1] }
      }
      "4" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "distant village lights, atmospheric haze", clip = ["1", 1] }
      }
      "5" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "blurry, watermark, deformed", clip = ["1", 1] }
      }
      "6" = {
        class_type = "EmptyLatentImage"
        inputs     = { width = 1024, height = 576, batch_size = 1 }
      }
      "7" = {
        class_type = "KSampler"
        inputs = {
          model        = ["1", 0]
          seed         = 42
          steps        = 28
          cfg          = 7.5
          sampler_name = "euler"
          scheduler    = "normal"
          positive     = ["2", 0]
          negative     = ["5", 0]
          latent_image = ["6", 0]
          denoise      = 1.0
        }
      }
      "8" = {
        class_type = "KSampler"
        inputs = {
          model        = ["1", 0]
          seed         = 84
          steps        = 32
          cfg          = 8.0
          sampler_name = "euler"
          scheduler    = "normal"
          positive     = ["3", 0]
          negative     = ["5", 0]
          latent_image = ["6", 0]
          denoise      = 0.8
        }
      }
      "9" = {
        class_type = "VAEDecode"
        inputs     = { samples = ["7", 0], vae = ["1", 2] }
      }
      "10" = {
        class_type = "SaveImage"
        inputs     = { images = ["9", 0], filename_prefix = "branchy_landscape" }
      }
    }

    tall_prompt_ladder = {
      "1" = {
        class_type = "CheckpointLoaderSimple"
        inputs     = { ckpt_name = "v1-5-pruned-emaonly.safetensors" }
      }
      "2" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "editorial portrait, rim lighting", clip = ["1", 1] }
      }
      "3" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "fashion editorial, dramatic contrast", clip = ["1", 1] }
      }
      "4" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "studio background sweep", clip = ["1", 1] }
      }
      "5" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "subtle film grain", clip = ["1", 1] }
      }
      "6" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "high detail skin texture", clip = ["1", 1] }
      }
      "7" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "clean catchlights", clip = ["1", 1] }
      }
      "8" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "low quality, blur, distorted anatomy", clip = ["1", 1] }
      }
      "9" = {
        class_type = "EmptyLatentImage"
        inputs     = { width = 640, height = 896, batch_size = 1 }
      }
      "10" = {
        class_type = "KSampler"
        inputs = {
          model        = ["1", 0]
          seed         = 101
          steps        = 30
          cfg          = 8.0
          sampler_name = "euler"
          scheduler    = "normal"
          positive     = ["2", 0]
          negative     = ["8", 0]
          latent_image = ["9", 0]
          denoise      = 1.0
        }
      }
      "11" = {
        class_type = "KSampler"
        inputs = {
          model        = ["1", 0]
          seed         = 202
          steps        = 36
          cfg          = 6.5
          sampler_name = "euler"
          scheduler    = "normal"
          positive     = ["6", 0]
          negative     = ["8", 0]
          latent_image = ["9", 0]
          denoise      = 0.75
        }
      }
      "12" = {
        class_type = "VAEDecode"
        inputs     = { samples = ["10", 0], vae = ["1", 2] }
      }
      "13" = {
        class_type = "VAEDecode"
        inputs     = { samples = ["11", 0], vae = ["1", 2] }
      }
      "14" = {
        class_type = "SaveImage"
        inputs     = { images = ["12", 0], filename_prefix = "tall_prompt_ladder" }
      }
    }

    dual_sampler_variants = {
      "1" = {
        class_type = "CheckpointLoaderSimple"
        inputs     = { ckpt_name = "v1-5-pruned-emaonly.safetensors" }
      }
      "2" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "product shot, clean reflections, softbox lighting", clip = ["1", 1] }
      }
      "3" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "dust, scratches, clutter", clip = ["1", 1] }
      }
      "4" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "matte studio table, minimal styling", clip = ["1", 1] }
      }
      "5" = {
        class_type = "EmptyLatentImage"
        inputs     = { width = 768, height = 768, batch_size = 1 }
      }
      "6" = {
        class_type = "EmptyLatentImage"
        inputs     = { width = 768, height = 512, batch_size = 1 }
      }
      "7" = {
        class_type = "KSampler"
        inputs = {
          model        = ["1", 0]
          seed         = 303
          steps        = 26
          cfg          = 7.0
          sampler_name = "euler"
          scheduler    = "normal"
          positive     = ["2", 0]
          negative     = ["3", 0]
          latent_image = ["5", 0]
          denoise      = 1.0
        }
      }
      "8" = {
        class_type = "KSampler"
        inputs = {
          model        = ["1", 0]
          seed         = 404
          steps        = 24
          cfg          = 6.0
          sampler_name = "euler"
          scheduler    = "normal"
          positive     = ["4", 0]
          negative     = ["3", 0]
          latent_image = ["6", 0]
          denoise      = 0.9
        }
      }
      "9" = {
        class_type = "VAEDecode"
        inputs     = { samples = ["7", 0], vae = ["1", 2] }
      }
      "10" = {
        class_type = "VAEDecode"
        inputs     = { samples = ["8", 0], vae = ["1", 2] }
      }
      "11" = {
        class_type = "SaveImage"
        inputs     = { images = ["9", 0], filename_prefix = "dual_sampler_square" }
      }
      "12" = {
        class_type = "SaveImage"
        inputs     = { images = ["10", 0], filename_prefix = "dual_sampler_wide" }
      }
    }

    postprocess_cluster = {
      "1" = {
        class_type = "CheckpointLoaderSimple"
        inputs     = { ckpt_name = "v1-5-pruned-emaonly.safetensors" }
      }
      "2" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "architectural interior, editorial realism", clip = ["1", 1] }
      }
      "3" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "people, motion blur, overexposure", clip = ["1", 1] }
      }
      "4" = {
        class_type = "CLIPTextEncode"
        inputs     = { text = "warm wood textures, natural daylight", clip = ["1", 1] }
      }
      "5" = {
        class_type = "EmptyLatentImage"
        inputs     = { width = 896, height = 640, batch_size = 1 }
      }
      "6" = {
        class_type = "KSampler"
        inputs = {
          model        = ["1", 0]
          seed         = 505
          steps        = 22
          cfg          = 7.2
          sampler_name = "euler"
          scheduler    = "normal"
          positive     = ["2", 0]
          negative     = ["3", 0]
          latent_image = ["5", 0]
          denoise      = 1.0
        }
      }
      "7" = {
        class_type = "KSampler"
        inputs = {
          model        = ["1", 0]
          seed         = 606
          steps        = 18
          cfg          = 5.8
          sampler_name = "euler"
          scheduler    = "normal"
          positive     = ["4", 0]
          negative     = ["3", 0]
          latent_image = ["5", 0]
          denoise      = 0.6
        }
      }
      "8" = {
        class_type = "VAEDecode"
        inputs     = { samples = ["6", 0], vae = ["1", 2] }
      }
      "9" = {
        class_type = "SaveImage"
        inputs     = { images = ["8", 0], filename_prefix = "postprocess_cluster" }
      }
    }

    negative_matrix = {
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
        inputs     = { images = ["12", 0], filename_prefix = "negative_matrix" }
      }
    }

    triple_branch_merge = {
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
        inputs     = { images = ["12", 0], filename_prefix = "triple_branch_merge" }
      }
    }
  }

  workspace_definitions = {
    dense_grid = {
      name    = "dense-grid"
      members = ["compact_reference", "branchy_landscape", "tall_prompt_ladder", "dual_sampler_variants", "postprocess_cluster", "negative_matrix", "triple_branch_merge"]
      layout = {
        display  = "grid"
        columns  = 2
        gap      = 300
        origin_x = 120
        origin_y = 80
      }
    }
    mixed_overrides = {
      name    = "mixed-overrides"
      members = ["tall_prompt_ladder", "compact_reference", "negative_matrix", "branchy_landscape"]
      layout = {
        display  = "grid"
        columns  = 2
        gap      = 280
        origin_x = 160
        origin_y = 100
      }
      node_layout = {
        mode       = "dag"
        direction  = "left_to_right"
        column_gap = 280
        row_gap    = 160
      }
      overrides = {
        compact_reference = {
          x = 1840
          y = 120
          style = {
            group_color     = "#ff00ff"
            title_font_size = 28
          }
        }
        negative_matrix = {
          x = 180
          y = 2440
        }
      }
    }
    vertical_stack = {
      name    = "vertical-stack"
      members = ["compact_reference", "postprocess_cluster", "tall_prompt_ladder", "dual_sampler_variants"]
      layout = {
        display   = "flex"
        direction = "column"
        gap       = 260
        origin_x  = 140
        origin_y  = 80
      }
    }
    wide_gallery = {
      name    = "wide-gallery"
      members = ["compact_reference", "branchy_landscape", "negative_matrix", "postprocess_cluster", "dual_sampler_variants", "triple_branch_merge"]
      layout = {
        display   = "flex"
        direction = "row"
        gap       = 320
        origin_x  = 80
        origin_y  = 120
      }
    }
  }
}
