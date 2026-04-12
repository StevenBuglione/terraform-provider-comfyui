terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.1"
    }
  }
}

variable "comfyui_host" {
  type    = string
  default = "127.0.0.1"
}

variable "comfyui_port" {
  type    = number
  default = 8188
}

provider "comfyui" {
  host = var.comfyui_host
  port = var.comfyui_port
}

locals {
  workflow_id = "execution-e2e-workflow"
  saved_image = jsondecode(comfyui_workflow.execution.outputs_json)["3"].images[0]
}

resource "comfyui_uploaded_image" "input" {
  file_path = "${path.module}/input.png"
  filename  = "execution-e2e-input.png"
  overwrite = true
  type      = "input"
}

resource "comfyui_workflow" "execution" {
  workflow_json = jsonencode({
    "1" = {
      class_type = "LoadImage"
      inputs = {
        image = comfyui_uploaded_image.input.filename
      }
    }
    "2" = {
      class_type = "ImageInvert"
      inputs = {
        image = ["1", 0]
      }
    }
    "3" = {
      class_type = "SaveImage"
      inputs = {
        images          = ["2", 0]
        filename_prefix = "execution_e2e"
      }
    }
  })

  extra_data_json = jsonencode({
    extra_pnginfo = {
      workflow = {
        id = local.workflow_id
      }
    }
  })

  execute                 = true
  validate_before_execute = false
  wait_for_completion     = true
  timeout_seconds         = 120
  cancel_on_delete        = true
}

data "comfyui_job" "execution" {
  id = comfyui_workflow.execution.prompt_id
}

data "comfyui_jobs" "by_workflow" {
  workflow_id = data.comfyui_job.execution.workflow_id
  statuses    = ["completed"]
}

data "comfyui_output" "saved" {
  filename  = local.saved_image.filename
  subfolder = try(local.saved_image.subfolder, null)
  type      = try(local.saved_image.type, "output")
}

resource "comfyui_output_artifact" "saved" {
  filename  = data.comfyui_output.saved.filename
  subfolder = data.comfyui_output.saved.subfolder
  type      = data.comfyui_output.saved.type
  path      = "${path.module}/artifacts/downloaded/${data.comfyui_output.saved.filename}"
}

output "workflow_execution" {
  value = {
    prompt_id             = comfyui_workflow.execution.prompt_id
    workflow_id           = comfyui_workflow.execution.workflow_id
    outputs_count         = comfyui_workflow.execution.outputs_count
    outputs_json          = comfyui_workflow.execution.outputs_json
    outputs_structured    = comfyui_workflow.execution.outputs_structured
    preview_output_json   = comfyui_workflow.execution.preview_output_json
    execution_status_json = comfyui_workflow.execution.execution_status_json
    execution_error_json  = comfyui_workflow.execution.execution_error_json
  }
}

output "job_execution" {
  value = {
    id                   = data.comfyui_job.execution.id
    status               = data.comfyui_job.execution.status
    workflow_id          = data.comfyui_job.execution.workflow_id
    outputs_count        = data.comfyui_job.execution.outputs_count
    execution_start_time = data.comfyui_job.execution.execution_start_time
    execution_end_time   = data.comfyui_job.execution.execution_end_time
  }
}

output "job_filter_results" {
  value = {
    has_more = data.comfyui_jobs.by_workflow.has_more
    job_ids  = [for job in data.comfyui_jobs.by_workflow.jobs : job.id]
  }
}

output "downloaded_artifact" {
  value = {
    filename       = data.comfyui_output.saved.filename
    exists         = data.comfyui_output.saved.exists
    url            = data.comfyui_output.saved.url
    local_path     = comfyui_output_artifact.saved.id
    local_sha256   = comfyui_output_artifact.saved.sha256
    content_length = comfyui_output_artifact.saved.content_length
  }
}
