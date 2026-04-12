resource "comfyui_checkpoint_loader_simple" "assembly_checkpoint" {
  ckpt_name = "v1-5-pruned-emaonly.safetensors"
}

resource "comfyui_clip_text_encode" "assembly_positive_base" {
  text = "hero product shot, reflective chrome, dramatic studio rim light"
  clip = comfyui_checkpoint_loader_simple.assembly_checkpoint.clip_output
}

resource "comfyui_clip_text_encode" "assembly_positive_detail" {
  text = "micro scratches, realistic reflections, premium packaging"
  clip = comfyui_checkpoint_loader_simple.assembly_checkpoint.clip_output
}

resource "comfyui_clip_text_encode" "assembly_positive_environment" {
  text = "black seamless backdrop, subtle haze, focused spotlight"
  clip = comfyui_checkpoint_loader_simple.assembly_checkpoint.clip_output
}

resource "comfyui_clip_text_encode" "assembly_negative" {
  text = "blurry, low quality, watermark, deformed geometry"
  clip = comfyui_checkpoint_loader_simple.assembly_checkpoint.clip_output
}

resource "comfyui_conditioning_combine" "assembly_positive_pair" {
  conditioning_1 = comfyui_clip_text_encode.assembly_positive_base.conditioning_output
  conditioning_2 = comfyui_clip_text_encode.assembly_positive_detail.conditioning_output
}

resource "comfyui_conditioning_combine" "assembly_positive_final" {
  conditioning_1 = comfyui_conditioning_combine.assembly_positive_pair.conditioning_output
  conditioning_2 = comfyui_clip_text_encode.assembly_positive_environment.conditioning_output
}

resource "comfyui_empty_latent_image" "assembly_square" {
  width      = 768
  height     = 768
  batch_size = 1
}

resource "comfyui_empty_latent_image" "assembly_wide" {
  width      = 1024
  height     = 576
  batch_size = 1
}

resource "comfyui_ksampler" "assembly_square_sampler" {
  model        = comfyui_checkpoint_loader_simple.assembly_checkpoint.model_output
  seed         = 4242
  steps        = 28
  cfg          = 7.5
  sampler_name = "euler"
  scheduler    = "normal"
  positive     = comfyui_conditioning_combine.assembly_positive_final.conditioning_output
  negative     = comfyui_clip_text_encode.assembly_negative.conditioning_output
  latent_image = comfyui_empty_latent_image.assembly_square.latent_output
  denoise      = 1.0
}

resource "comfyui_ksampler" "assembly_wide_sampler" {
  model        = comfyui_checkpoint_loader_simple.assembly_checkpoint.model_output
  seed         = 4343
  steps        = 32
  cfg          = 6.8
  sampler_name = "euler"
  scheduler    = "normal"
  positive     = comfyui_conditioning_combine.assembly_positive_final.conditioning_output
  negative     = comfyui_clip_text_encode.assembly_negative.conditioning_output
  latent_image = comfyui_empty_latent_image.assembly_wide.latent_output
  denoise      = 0.85
}

resource "comfyui_vae_decode" "assembly_square_decode" {
  samples = comfyui_ksampler.assembly_square_sampler.latent_output
  vae     = comfyui_checkpoint_loader_simple.assembly_checkpoint.vae_output
}

resource "comfyui_vae_decode" "assembly_wide_decode" {
  samples = comfyui_ksampler.assembly_wide_sampler.latent_output
  vae     = comfyui_checkpoint_loader_simple.assembly_checkpoint.vae_output
}

resource "comfyui_save_image" "assembly_square_output" {
  images          = comfyui_vae_decode.assembly_square_decode.image_output
  filename_prefix = "release_assembly_square"
}

resource "comfyui_save_image" "assembly_wide_output" {
  images          = comfyui_vae_decode.assembly_wide_decode.image_output
  filename_prefix = "release_assembly_wide"
}

resource "comfyui_workflow" "assembled_resource" {
  node_ids = [
    comfyui_checkpoint_loader_simple.assembly_checkpoint.id,
    comfyui_clip_text_encode.assembly_positive_base.id,
    comfyui_clip_text_encode.assembly_positive_detail.id,
    comfyui_clip_text_encode.assembly_positive_environment.id,
    comfyui_clip_text_encode.assembly_negative.id,
    comfyui_conditioning_combine.assembly_positive_pair.id,
    comfyui_conditioning_combine.assembly_positive_final.id,
    comfyui_empty_latent_image.assembly_square.id,
    comfyui_empty_latent_image.assembly_wide.id,
    comfyui_ksampler.assembly_square_sampler.id,
    comfyui_ksampler.assembly_wide_sampler.id,
    comfyui_vae_decode.assembly_square_decode.id,
    comfyui_vae_decode.assembly_wide_decode.id,
    comfyui_save_image.assembly_square_output.id,
    comfyui_save_image.assembly_wide_output.id,
  ]

  execute = false
}

resource "comfyui_workflow" "raw_import" {
  workflow_json = jsonencode(local.raw_import_prompt)
  execute       = false
}

resource "comfyui_workflow" "gallery_companion" {
  workflow_json = jsonencode(local.gallery_companion_prompt)
  execute       = false
}

data "comfyui_prompt_json" "assembled_resource" {
  json = comfyui_workflow.assembled_resource.assembled_json
}

data "comfyui_prompt_json" "raw_import" {
  json = comfyui_workflow.raw_import.assembled_json
}

data "comfyui_prompt_to_workspace" "assembled_resource" {
  name        = "assembled-resource-graph"
  prompt_json = comfyui_workflow.assembled_resource.assembled_json
}

data "comfyui_prompt_to_workspace" "raw_import" {
  name        = "raw-import-graph"
  prompt_json = comfyui_workflow.raw_import.assembled_json
}

data "comfyui_workspace_to_prompt" "assembled_roundtrip" {
  workspace_json = data.comfyui_prompt_to_workspace.assembled_resource.workspace_json
}

data "comfyui_prompt_json" "assembled_roundtrip" {
  json = data.comfyui_workspace_to_prompt.assembled_roundtrip.prompt_json
}

data "comfyui_prompt_to_workspace" "assembled_roundtrip" {
  name        = "assembled-roundtrip-graph"
  prompt_json = data.comfyui_workspace_to_prompt.assembled_roundtrip.prompt_json
}

resource "comfyui_workspace" "release_gallery" {
  name        = "release-gallery"
  output_file = "${path.module}/artifacts/generated/release_gallery.json"

  layout = {
    display  = "grid"
    columns  = 2
    gap      = 320
    origin_x = 120
    origin_y = 80
  }

  node_layout = {
    mode       = "dag"
    direction  = "left_to_right"
    column_gap = 240
    row_gap    = 120
  }

  workflows = [
    {
      name          = "Assembled Resource"
      workflow_json = comfyui_workflow.assembled_resource.assembled_json
      style = {
        group_color     = "#0f766e"
        title_font_size = 20
      }
    },
    {
      name          = "Raw Import"
      workflow_json = comfyui_workflow.raw_import.assembled_json
      style = {
        group_color     = "#2563eb"
        title_font_size = 18
      }
    },
    {
      name          = "Gallery Companion"
      workflow_json = comfyui_workflow.gallery_companion.assembled_json
      x             = 180
      y             = 2160
      style = {
        group_color     = "#db2777"
        title_font_size = 18
      }
    },
  ]
}
