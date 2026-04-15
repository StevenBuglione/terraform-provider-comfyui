# Terraform Provider for ComfyUI

[![Tests](https://github.com/StevenBuglione/terraform-provider-comfyui/actions/workflows/test.yml/badge.svg)](https://github.com/StevenBuglione/terraform-provider-comfyui/actions/workflows/test.yml)
[![Release](https://github.com/StevenBuglione/terraform-provider-comfyui/actions/workflows/release.yml/badge.svg)](https://github.com/StevenBuglione/terraform-provider-comfyui/actions/workflows/release.yml)
[![Terraform Registry](https://img.shields.io/badge/Terraform%20Registry-StevenBuglione%2Fcomfyui-623CE4?logo=terraform)](https://registry.terraform.io/providers/StevenBuglione/comfyui/latest)
[![Go 1.25.1](https://img.shields.io/badge/go-1.25.1-00ADD8?logo=go)](./go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)

Manage [ComfyUI](https://github.com/comfyanonymous/ComfyUI) workflows with Terraform by modeling nodes, assembling executable graphs, validating runtime-backed inputs during `terraform plan`, and translating native ComfyUI artifacts into canonical Terraform.

This provider is generated from the ComfyUI version pinned in this repo and currently exposes:

- `645` generated node resources from ComfyUI `v0.18.5`
- `9` hand-written resources for workflow orchestration, artifact management, and workspace export
- `20` data sources for runtime inspection, validation, translation, and Terraform synthesis

## What This Provider Does

- Model built-in ComfyUI nodes as typed Terraform resources instead of hand-editing prompt JSON.
- Assemble and optionally execute workflows with `comfyui_workflow`.
- Group workflows with `comfyui_workflow_collection`.
- Compose multiple workflows into one UI-oriented canvas export with `comfyui_workspace`.
- Import, translate, validate, and synthesize native prompt and workspace artifacts.
- Fail `terraform plan` when recognized runtime-backed inventory selections such as checkpoints or LoRAs do not exist on the target ComfyUI server.
- Expose normalized execution state through `comfyui_workflow`, `comfyui_job`, `comfyui_jobs`, `comfyui_queue`, and `comfyui_workflow_history`.
- Manage uploaded inputs, local prompt/workspace/subgraph artifacts, and downloaded outputs through Terraform-managed resources.

## DynamicCombo Schema Migration

Some generated nodes now expose `COMFY_DYNAMICCOMBO_V3` inputs as nested objects instead of opaque strings. The required field is `selection`; any option-specific child inputs now live inside that same object and are documented in the generated resource page.

Prefer this typed resource path with `comfyui_workflow.node_ids`. Keep raw `workflow_json` for importing existing prompt artifacts or for edge cases the generated schema still cannot model cleanly.

The nested fields shown under a DynamicCombo input are a union across its options. The provider validates which child fields are required and allowed for the selected option and rejects fields from unselected options.

WAN2-style migration example:

```hcl
resource "comfyui_wan2_text_to_video_api" "clip" {
  model = {
    selection       = "wan2.7-t2v"
    prompt          = "cinematic storm over the ocean"
    negative_prompt = ""
    resolution      = "720P"
    ratio           = "16:9"
    duration        = 5
  }

  prompt_extend = true
  seed          = 42
  watermark     = false
}
```

If you previously set `model = "wan2.7-t2v"`, move that value to `model.selection` and then populate the child fields required for that selection. Execute the node through `comfyui_workflow.node_ids` as the typed path; keep raw `workflow_json` for imports and true schema edge cases only.

## Provider Surface at a Glance

| Surface | What it covers |
|---|---|
| Generated node resources | `645` typed `comfyui_*` node resources generated from pinned ComfyUI metadata. These resources are virtual and participate in workflow assembly. |
| Orchestration resources | `comfyui_workflow`, `comfyui_workflow_collection`, and `comfyui_workspace`. |
| Artifact resources | `comfyui_prompt_artifact`, `comfyui_workspace_artifact`, `comfyui_subgraph`, `comfyui_uploaded_image`, `comfyui_uploaded_mask`, and `comfyui_output_artifact`. |
| Runtime and schema data sources | `comfyui_inventory`, `comfyui_job`, `comfyui_jobs`, `comfyui_output`, `comfyui_queue`, `comfyui_subgraph_catalog`, `comfyui_subgraph_definition`, `comfyui_system_stats`, `comfyui_workflow_history`, `comfyui_node_info`, `comfyui_node_schema`, and `comfyui_provider_info`. |
| Translation and validation data sources | `comfyui_prompt_json`, `comfyui_workspace_json`, `comfyui_prompt_to_workspace`, `comfyui_workspace_to_prompt`, `comfyui_prompt_to_terraform`, `comfyui_workspace_to_terraform`, `comfyui_prompt_validation`, and `comfyui_workspace_validation`. |

## Intended Audiences

- Provider users who want to author and run ComfyUI workflows declaratively in Terraform.
- AI coding harnesses that need machine-readable node contracts, synthesis surfaces, and strict validation before writing or changing Terraform.
- Contributors maintaining the generated extraction pipeline, hand-rolled orchestration layer, and release validation harnesses.

## Scope

The supported path is centered on the built-in ComfyUI behavior pinned in this repo. The provider is designed to make built-in workflows and their runtime-backed inventories safe to author, validate, and maintain in Terraform. Arbitrary exotic custom-node ecosystems are not the central compatibility promise.

## Requirements

- A reachable ComfyUI server. By default the provider connects to `localhost:8188`.
- A current Terraform CLI with modern provider installation support.
- Any built-in or partner-node models required by the workflow you plan to run.

## Installation

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

### Versioning Policy

Provider versions follow the **ComfyUI compatibility line** model:

- Provider `0.18.x` is the compatibility line for ComfyUI `v0.18.5`
- The first release in this line is `v0.18.5`
- Later provider-only fixes are `v0.18.6`, `v0.18.7`, etc.
- The exact upstream pin remains authoritative in generated metadata and `comfyui_provider_info`
- Users should constrain the provider with `~> 0.18` for this line
- If the pinned upstream ComfyUI version changes materially, a new provider line is started rather than silently continuing `0.18.x`

Query the exact ComfyUI version at runtime:

```hcl
data "comfyui_provider_info" "current" {}

output "compatibility" {
  value = "Provider ${data.comfyui_provider_info.current.provider_version} for ComfyUI ${data.comfyui_provider_info.current.comfyui_version}"
}
```

Minimal provider configuration:

```hcl
provider "comfyui" {}
```

Environment variables are also supported:

- `COMFYUI_HOST`
- `COMFYUI_PORT`
- `COMFYUI_API_KEY`
- `COMFYUI_COMFY_ORG_AUTH_TOKEN`
- `COMFYUI_COMFY_ORG_API_KEY`
- `COMFYUI_DEFAULT_WORKFLOW_EXTRA_DATA_JSON`
- `COMFYUI_UNSUPPORTED_DYNAMIC_VALIDATION_MODE`

`COMFYUI_HOST` can be either a bare hostname such as `localhost` or a full URL such as `http://127.0.0.1:8188`.

For partner-backed nodes that rely on hidden `comfy_org` auth, browser login state is not inherited by Terraform. The provider now inspects the assembled workflow, detects when hidden partner auth is required, and fails before queueing if no valid partner credential is configured.

Auth model:

- `api_key` authenticates Terraform to the ComfyUI service itself.
- `comfy_org_api_key` is the preferred automation credential for partner-backed node execution.
- `comfy_org_auth_token` is a manual/session-token fallback when you intentionally provide it.
- Browser login is frontend state only and does **not** authenticate provider-submitted workflows.

When both `comfy_org_api_key` and `comfy_org_auth_token` are configured, the provider prefers `comfy_org_api_key` and injects only that credential family into workflow `extra_data`.

If you are unauthenticated for partner-backed nodes, your options are:

1. **Preferred:** create a ComfyOrg API key and set `comfy_org_api_key` or `COMFYUI_COMFY_ORG_API_KEY`
2. **Fallback:** explicitly provide a frontend auth token with `comfy_org_auth_token` or `COMFYUI_COMFY_ORG_AUTH_TOKEN`
3. **Alternative:** switch to a runtime/node surface that uses direct provider credentials instead of `comfy_org`

The locally inspected `comfy` CLI does not currently expose a supported partner-execution login/token acquisition command for this provider path.

You can inspect the provider's current auth posture with `data "comfyui_provider_info" "current" {}`. It now reports whether service API auth is configured, whether partner auth is configured, and which auth families are currently available.

## Quick Start

This is the smallest useful text-to-image workflow built from generated node resources and executed through `comfyui_workflow`.

```hcl
terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.18"
    }
  }
}

provider "comfyui" {}

resource "comfyui_checkpoint_loader_simple" "checkpoint" {
  ckpt_name = "v1-5-pruned-emaonly.safetensors"
}

resource "comfyui_clip_text_encode" "positive" {
  text = "a cinematic mountain sunrise, volumetric light, highly detailed"
  clip = comfyui_checkpoint_loader_simple.checkpoint.clip_output
}

resource "comfyui_clip_text_encode" "negative" {
  text = "blurry, low quality, watermark"
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
  filename_prefix = "quickstart"
}

resource "comfyui_workflow" "txt2img" {
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

output "workflow_id" {
  value = comfyui_workflow.txt2img.workflow_id
}

output "outputs_json" {
  value = comfyui_workflow.txt2img.outputs_json
}

output "execution_status_json" {
  value = comfyui_workflow.txt2img.execution_status_json
}
```

At a high level:

- `terraform plan` validates the typed node graph and checks recognized runtime-backed inventory values against the live ComfyUI server.
- `terraform apply` assembles the graph into ComfyUI API-format JSON and submits it when `execute = true`.
- `comfyui_workflow` returns rich execution surfaces such as `workflow_id`, `outputs_json`, `preview_output_json`, `execution_status_json`, and `execution_error_json`.

## Core Concepts

### Generated node resources

Most `comfyui_*` node resources are generated from ComfyUI metadata and are virtual: they exist in Terraform state and participate in workflow assembly, but they do not make API calls on their own.

### `comfyui_workflow`

`comfyui_workflow` is the execution boundary. It can:

- assemble a workflow from `node_ids`
- accept raw `workflow_json`
- validate against live `/object_info` metadata before execution
- export assembled prompt JSON to disk
- execute immediately, export only, or do both
- expose rich execution fields from `/api/jobs`

### `comfyui_workspace`

`comfyui_workspace` is the layout-aware canvas export resource. It can:

- accept multiple API-format workflows, including `comfyui_workflow.*.assembled_json`
- compose them into a single workspace/subgraph JSON file
- lay out workflow islands with typed `layout` settings
- control internal node readability with `node_layout`
- preserve UI-oriented styling and deterministic placement

### Artifact import, translation, and synthesis

The provider also exposes narrative and AI-facing artifact surfaces:

- `comfyui_prompt_json` and `comfyui_workspace_json` import and normalize native ComfyUI artifacts.
- `comfyui_prompt_to_workspace` and `comfyui_workspace_to_prompt` translate between prompt and workspace forms with fidelity reporting.
- `comfyui_prompt_to_terraform` and `comfyui_workspace_to_terraform` synthesize canonical Terraform IR and rendered HCL from native ComfyUI artifacts.
- `comfyui_prompt_validation` and `comfyui_workspace_validation` validate artifacts in executable or fragment-oriented modes.
- `comfyui_inventory` exposes live runtime-backed inventory values by normalized kind.

## Validation and Confidence

The provider’s strongest guarantees come from combining generated contracts with real runtime verification:

- `make generate`
  - regenerates node resources, generated schema metadata, and UI sizing hints from the pinned ComfyUI source and live frontend behavior
- `make synthesis-e2e`
  - proves prompt/workspace-to-Terraform synthesis through real Terraform runs
- `make inventory-plan-e2e`
  - proves bad dynamic inventory values fail during `terraform plan`
- `make execution-e2e`
  - proves model-free execution, job-state reads, and artifact download behavior
- `make workspace-e2e`
  - proves workspace layout, spacing, and connectivity in a real ComfyUI browser session
- `make release-e2e`
  - proves assembled workflows, raw imports, translation round trips, and workspace exports in real ComfyUI

## Documentation Map

Start here based on what you are doing:

- Provider user: [Getting Started](./docs/guides/getting-started.md)
- Workflow authoring: [Workflow Authoring](./docs/guides/workflow-authoring.md)
- AI harness authoring: [AI Harness Guide](./docs/guides/ai-harness-guide.md)
- Contributor workflow: [Contributing](./docs/guides/contributing.md)
- Generation internals: [Generation Architecture](./docs/guides/generation-architecture.md)
- Release confidence: [Release Validation](./docs/guides/release-validation.md)
- Maintainability boundary: [AI Maintainability](./docs/guides/ai-maintainability.md)
- Scope and boundaries: [Known Boundaries](./docs/guides/known-boundaries.md)
- Full docs map: [docs/index.md](./docs/index.md)

Generated API references live under:

- [Resource docs](./docs/resources/)
- [Data source docs](./docs/data-sources/)

## Examples

- [Provider configuration](./examples/provider/provider.tf)
- [Text to image](./examples/resources/txt2img/main.tf)
- [Image to image](./examples/resources/img2img/main.tf)
- [Upscale](./examples/resources/upscale/main.tf)
- [Workflow JSON](./examples/resources/workflow_json/main.tf)
- [Workflow file export](./examples/resources/workflow_file/main.tf)
- [Workflow collections](./examples/resources/collection/main.tf)
- [Workspace export](./examples/resources/workspace/main.tf)
- [Video generation](./examples/resources/video_gen/main.tf)
- [Data sources](./examples/data-sources/main.tf)

## Development

Core contributor commands:

```bash
make comfyui-start                     # start local ComfyUI in the background
make comfyui-status                    # print pid, base URL, and log path
make comfyui-stop                      # stop the background ComfyUI process
make generate
make docs
make docs-validate
make lint
make test
make vet
make synthesis-e2e
make inventory-plan-e2e
make execution-e2e
make workspace-e2e
make release-e2e
```

The ComfyUI helper targets reuse the existing workspace lifecycle scripts but default to a dedicated runtime directory at `validation/comfyui_dev/.runtime`. Override the port or runtime location when needed:

```bash
make comfyui-start COMFYUI_PORT=8190
make comfyui-start COMFYUI_RUNTIME_DIR=$PWD/.runtime/comfyui
```

For deeper development context and repo conventions, see [Contributing](./docs/guides/contributing.md) and [CLAUDE.md](./CLAUDE.md).

## License

This project is licensed under the [MIT License](./LICENSE).
