data "comfyui_queue" "example" {}

output "queue_pending" {
  value = data.comfyui_queue.example.pending_count
}
