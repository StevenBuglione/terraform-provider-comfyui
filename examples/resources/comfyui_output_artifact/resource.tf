# Download a known ComfyUI output file to a local path.
resource "comfyui_output_artifact" "example" {
  filename = "example-output.png"
  type     = "output"
  path     = "${path.module}/downloads/example-output.png"
}
