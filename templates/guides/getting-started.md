---
page_title: "Getting Started - ComfyUI Provider"
subcategory: ""
description: |-
  Install the provider, connect to a ComfyUI server, run a minimal workflow, and inspect the resulting execution state.
---

# Getting Started

This guide is the fastest path from an empty Terraform module to a runnable ComfyUI workflow.

## Prerequisites

- A reachable ComfyUI server. By default the provider connects to `localhost:8188`.
- Terraform with modern provider installation support.
- Any built-in models your workflow references already available to the target ComfyUI server.

## Install the Provider

```hcl
terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.18"
    }
  }
}
```

Provider configuration can stay minimal:

```hcl
provider "comfyui" {}
```

Or be set explicitly:

```hcl
provider "comfyui" {
  host    = var.comfyui_host
  port    = var.comfyui_port
  api_key = var.comfyui_api_key
}
```

Environment variables are also supported:

- `COMFYUI_HOST`
- `COMFYUI_PORT`
- `COMFYUI_API_KEY`
- `COMFYUI_COMFY_ORG_AUTH_TOKEN`
- `COMFYUI_COMFY_ORG_API_KEY`
- `COMFYUI_DEFAULT_WORKFLOW_EXTRA_DATA_JSON`
- `COMFYUI_UNSUPPORTED_DYNAMIC_VALIDATION_MODE`

`COMFYUI_HOST` can be a bare hostname such as `localhost`, a `host:port` pair, or a full URL such as `http://127.0.0.1:8188`.

If you are executing partner-backed nodes such as hidden `comfy_org` resources, browser login state is not reused automatically. Set `comfy_org_auth_token` / `comfy_org_api_key` on the provider, or export the matching environment variables above, so workflow executions can inject them into `/prompt.extra_data`.

## Minimal Runnable Workflow

```hcl
resource "comfyui_checkpoint_loader_simple" "checkpoint" {
  ckpt_name = "v1-5-pruned-emaonly.safetensors"
}

resource "comfyui_clip_text_encode" "positive" {
  text = "a cinematic mountain sunrise"
  clip = comfyui_checkpoint_loader_simple.checkpoint.clip_output
}

resource "comfyui_clip_text_encode" "negative" {
  text = "blurry, low quality"
  clip = comfyui_checkpoint_loader_simple.checkpoint.clip_output
}

resource "comfyui_empty_latent_image" "latent" {
  width      = 512
  height     = 512
  batch_size = 1
}

resource "comfyui_ksampler" "sampler" {
  model        = comfyui_checkpoint_loader_simple.checkpoint.model_output
  seed         = 42
  steps        = 20
  cfg          = 7.0
  sampler_name = "euler"
  scheduler    = "normal"
  positive     = comfyui_clip_text_encode.positive.conditioning_output
  negative     = comfyui_clip_text_encode.negative.conditioning_output
  latent_image = comfyui_empty_latent_image.latent.latent_output
  denoise      = 1.0
}

resource "comfyui_vae_decode" "decode" {
  samples = comfyui_ksampler.sampler.latent_output
  vae     = comfyui_checkpoint_loader_simple.checkpoint.vae_output
}

resource "comfyui_save_image" "output" {
  images          = comfyui_vae_decode.decode.image_output
  filename_prefix = "getting_started"
}

resource "comfyui_workflow" "example" {
  node_ids = [
    comfyui_checkpoint_loader_simple.checkpoint.id,
    comfyui_clip_text_encode.positive.id,
    comfyui_clip_text_encode.negative.id,
    comfyui_empty_latent_image.latent.id,
    comfyui_ksampler.sampler.id,
    comfyui_vae_decode.decode.id,
    comfyui_save_image.output.id,
  ]

  execute             = true
  wait_for_completion = true
  timeout_seconds     = 300
}
```

## What `terraform plan` Checks

For this provider, `terraform plan` is more than a local schema check.

It validates:

- Terraform-level resource schemas for built-in ComfyUI nodes.
- Graph assembly inputs such as resource IDs and required attributes.
- Recognized runtime-backed inventory selections against the live ComfyUI server.

That last point matters for loader-style resources. If a built-in dynamic input maps to a live inventory kind such as `checkpoints`, `loras`, or `text_encoders`, a missing value should fail during `terraform plan`, not later during workflow execution.

Use [comfyui_inventory](../data-sources/inventory.md) to inspect those live values directly.

For unsupported dynamic-expression inputs that cannot be proved at plan time, the provider defaults to `error` but can be downgraded to `warning` or `ignore` with `unsupported_dynamic_validation_mode` when you intentionally accept runtime-only validation. That policy now applies consistently during generated-node plan validation and `comfyui_workflow` preflight.

## What `terraform apply` Does

`terraform apply` stores virtual node resources in state, assembles them into ComfyUI API-format JSON, and then lets `comfyui_workflow` decide what to do next.

When you author workflows from generated node resources, there are now two assembly options:

- `node_ids` alone keeps the older compatibility path backed by the process-local node registry. Normal refresh/apply cycles still work because generated nodes re-register during resource `Read`, but it is not the durable option for cold-start or cross-process registry gaps.
- `node_ids` plus `node_definition_jsons` is the durable path for cold-registry or cross-process updates. Each generated node resource exposes a computed `node_definition_json` snapshot that you can pass straight into the matching workflow list. Keep the two lists the same length and align each JSON entry by position with the matching `node_ids` entry.

Typical `comfyui_workflow` modes are:

- execute and wait for completion
- execute without waiting
- export assembled JSON to disk
- export and execute in the same resource

## Reading Results

`comfyui_workflow` exposes rich execution fields rather than the older coarse compatibility fields.

The most useful starting points are:

- `prompt_id`
- `workflow_id`
- `outputs_count`
- `preview_output_json`
- `outputs_json`
- `execution_status_json`
- `execution_error_json`

For richer state inspection after execution, use:

- [comfyui_job](../data-sources/job.md)
- [comfyui_jobs](../data-sources/jobs.md)
- [comfyui_queue](../data-sources/queue.md)
- [comfyui_workflow_history](../data-sources/workflow_history.md)

## Next Steps

- [Workflow Authoring](./workflow-authoring.md)
  for node-per-resource patterns, raw prompt import, and workspace export
- [AI Harness Guide](./ai-harness-guide.md)
  for generated node contracts, synthesis, and validation loops
- [Release Validation](./release-validation.md)
  for the verification lanes used in this repo
