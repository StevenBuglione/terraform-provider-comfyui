data "comfyui_output" "latest" {
  prompt_id = "abc-123-def"
}

output "output_images" {
  value = data.comfyui_output.latest.images
}
