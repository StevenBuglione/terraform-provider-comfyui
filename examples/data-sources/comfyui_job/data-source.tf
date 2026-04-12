data "comfyui_job" "example" {
  id = "replace-with-job-id"
}

output "job_status" {
  value = data.comfyui_job.example.status
}
