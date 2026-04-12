# Place a local input.png next to this example before applying it.
resource "comfyui_uploaded_image" "example" {
  file_path = "${path.module}/input.png"
  filename  = "docs-example-input.png"
  overwrite = true
  type      = "input"
}
