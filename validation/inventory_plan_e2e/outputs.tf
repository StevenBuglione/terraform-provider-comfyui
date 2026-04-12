output "checkpoint_inventory" {
  value = data.comfyui_inventory.live.inventories
}

output "checkpoint_loader_schema" {
  value = {
    validation_kind                 = data.comfyui_node_schema.checkpoint_loader.required_inputs[0].validation_kind
    inventory_kind                  = data.comfyui_node_schema.checkpoint_loader.required_inputs[0].inventory_kind
    supports_strict_plan_validation = data.comfyui_node_schema.checkpoint_loader.required_inputs[0].supports_strict_plan_validation
  }
}
