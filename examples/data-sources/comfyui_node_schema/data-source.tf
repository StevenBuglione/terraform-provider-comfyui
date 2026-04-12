data "comfyui_node_schema" "example" {
  node_type = "KSampler"
}

output "required_inputs" {
  value = [for input in data.comfyui_node_schema.example.required_inputs : input.name]
}
