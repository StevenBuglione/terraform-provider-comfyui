---
page_title: "ComfyUI Provider"
subcategory: ""
description: |-
  Terraform provider for ComfyUI — manage workflows, nodes, and image generation infrastructure as code.
---

# ComfyUI Provider

The ComfyUI provider enables Terraform to manage resources on a
[ComfyUI](https://github.com/comfyanonymous/ComfyUI) server — a powerful,
node-based Stable Diffusion GUI and backend.

## How It Works

This provider maps every ComfyUI node type to a Terraform resource using a
**node-per-resource** model. A code generator reads the ComfyUI `/object_info`
endpoint and produces ~645 typed Terraform resources — one for each built-in
node (e.g., `comfyui_ksampler`, `comfyui_clip_text_encode`,
`comfyui_vae_decode`).

In addition to the generated node resources the provider includes:

- **`comfyui_workflow`** — a hand-written resource that submits a complete
  workflow (API-format JSON) to ComfyUI for execution.
- **`comfyui_workflow_collection`** — a hand-written resource for grouping
  workflow definitions and writing an index manifest.
- **Six data sources** — `comfyui_system_stats`, `comfyui_queue`,
  `comfyui_node_info`, `comfyui_workflow_history`, `comfyui_output`, and
  `comfyui_provider_info` — for reading server state and provider metadata.

Node resources are *virtual*: they exist only in Terraform state and are
assembled into a workflow graph at plan/apply time. No data is written to
ComfyUI until a `comfyui_workflow` resource references them.

## Example Usage

```terraform
# ComfyUI Provider Configuration
#
# The provider connects to a running ComfyUI server instance.
# All connection parameters can be set via environment variables:
#   COMFYUI_HOST    — server hostname (default: "localhost")
#   COMFYUI_PORT    — server port     (default: 8188)
#   COMFYUI_API_KEY — optional API key for authenticated servers

terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.1"
    }
  }
}

# Minimal: relies entirely on environment variables / defaults
provider "comfyui" {}

# Explicit: useful for multi-server setups or CI/CD
# provider "comfyui" {
#   host    = var.comfyui_host
#   port    = var.comfyui_port
#   api_key = var.comfyui_api_key
# }

variable "comfyui_host" {
  description = "Hostname of the ComfyUI server"
  type        = string
  default     = "localhost"
}

variable "comfyui_port" {
  description = "Port of the ComfyUI server"
  type        = number
  default     = 8188
}

variable "comfyui_api_key" {
  description = "API key for ComfyUI authentication (leave empty if not required)"
  type        = string
  default     = ""
  sensitive   = true
}
```

## Configuration Reference

### Optional

- **`host`** (String) — ComfyUI server hostname or IP address. Defaults to
  `localhost`. Can also be set with the `COMFYUI_HOST` environment variable.
- **`port`** (Number) — ComfyUI server port. Defaults to `8188`. Can also be
  set with the `COMFYUI_PORT` environment variable.
- **`api_key`** (String, Sensitive) — API key for ComfyUI authentication, if
  enabled on the server. Can also be set with the `COMFYUI_API_KEY` environment
  variable.

## Authentication

The provider supports three ways to supply credentials, checked in order:

1. Explicit provider block attributes (`host`, `port`, `api_key`).
2. Environment variables (`COMFYUI_HOST`, `COMFYUI_PORT`, `COMFYUI_API_KEY`).
3. Built-in defaults (`localhost:8188`, no API key).

For CI/CD pipelines, environment variables are recommended so that secrets
stay out of HCL files.
