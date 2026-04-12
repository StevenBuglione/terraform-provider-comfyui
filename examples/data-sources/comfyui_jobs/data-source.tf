data "comfyui_jobs" "example" {
  statuses   = ["running", "completed"]
  sort_by    = "create_time"
  sort_order = "desc"
  limit      = 10
}

output "job_count" {
  value = data.comfyui_jobs.example.job_count
}
