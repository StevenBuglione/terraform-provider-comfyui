data "comfyui_inventory" "example" {
  kinds = ["checkpoints", "loras"]
}

output "inventories" {
  value = data.comfyui_inventory.example.inventories
}
