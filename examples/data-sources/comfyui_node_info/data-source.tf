data "comfyui_node_info" "ksampler" {
  class_type = "KSampler"
}

output "ksampler_inputs" {
  value = data.comfyui_node_info.ksampler.inputs
}
