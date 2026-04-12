# Place local input.png and mask.png files next to this example before applying it.
resource "comfyui_uploaded_image" "original" {
  file_path = "${path.module}/input.png"
  filename  = "docs-example-original.png"
  overwrite = true
  type      = "input"
}

resource "comfyui_uploaded_mask" "example" {
  file_path          = "${path.module}/mask.png"
  filename           = "docs-example-mask.png"
  overwrite          = true
  type               = "input"
  original_filename  = comfyui_uploaded_image.original.filename
  original_subfolder = comfyui_uploaded_image.original.subfolder
  original_type      = comfyui_uploaded_image.original.type
}
