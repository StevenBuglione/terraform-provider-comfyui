# Terraform Provider for ComfyUI

[![Tests](https://github.com/StevenBuglione/terraform-provider-comfyui/actions/workflows/test.yml/badge.svg)](https://github.com/StevenBuglione/terraform-provider-comfyui/actions/workflows/test.yml)
[![Release](https://github.com/StevenBuglione/terraform-provider-comfyui/actions/workflows/release.yml/badge.svg)](https://github.com/StevenBuglione/terraform-provider-comfyui/actions/workflows/release.yml)
[![Terraform Registry](https://img.shields.io/badge/Terraform%20Registry-StevenBuglione%2Fcomfyui-623CE4?logo=terraform)](https://registry.terraform.io/providers/StevenBuglione/comfyui/latest)
[![Go 1.25.1](https://img.shields.io/badge/go-1.25.1-00ADD8?logo=go)](./go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)

Manage [ComfyUI](https://github.com/comfyanonymous/ComfyUI) workflows with Terraform by modeling nodes, assembling executable graphs, and querying runtime state from a running ComfyUI server.

This provider uses a node-per-resource model: generated Terraform resources describe ComfyUI nodes in state, and `comfyui_workflow` assembles and optionally executes them when you apply. The current build is generated from ComfyUI `v0.18.5` and includes `645` built-in node resources.

## Why This Provider

- Model ComfyUI graphs as typed Terraform resources instead of hand-editing raw workflow JSON.
- Assemble workflows from `node_ids` or submit raw ComfyUI API-format JSON directly.
- Export workflow files, execute runs, or do both from the same `comfyui_workflow` resource.
- Observe normalized execution state through `comfyui_workflow`, `comfyui_job`, `comfyui_jobs`, `comfyui_queue`, and `comfyui_workflow_history`.
- Cancel queued or running executions through normal Terraform lifecycle with `cancel_on_delete`.
- Import, translate, and validate native prompt/workspace artifacts before execution.
- Upload input images or masks to ComfyUI and materialize remote outputs back to local disk.
- Organize reusable workflow sets with `comfyui_workflow_collection`.
- Inspect server state plus artifact import/translation/validation surfaces with provider data sources.

## Requirements

- A reachable ComfyUI server. By default the provider connects to `localhost:8188`.
- A current Terraform CLI with support for modern provider installation syntax.
- Any models, custom nodes, or partner-node integrations required by the workflow you plan to run.

## Installation

Install the provider from the Terraform Registry:

```hcl
terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.1"
    }
  }
}
```

Minimal provider configuration:

```hcl
provider "comfyui" {}
```

The provider also supports environment-variable configuration:

- `COMFYUI_HOST`
- `COMFYUI_PORT`
- `COMFYUI_API_KEY`

## Quick Start

This is the smallest useful text-to-image workflow using generated node resources plus `comfyui_workflow` assembly:

```hcl
terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.1"
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
```

At apply time:

- Terraform stores node resources in state and assembles them into ComfyUI API-format JSON.
- `comfyui_workflow` submits the graph to ComfyUI and can wait for completion.
- The workflow resource returns execution fields like `prompt_id`, `workflow_id`, `outputs_count`, `preview_output_json`, `outputs_json`, `execution_status_json`, `execution_error_json`, and `assembled_json`.
- The job data sources let you re-read the same execution through normalized `/api/jobs` views for filtering and feedback loops.

## Core Concepts

### Virtual node resources

Most generated `comfyui_*` node resources are virtual. They define typed node inputs and outputs in Terraform state, but they do not send API requests on their own.

### `comfyui_workflow`

`comfyui_workflow` is the execution boundary. It can:

- assemble a workflow from node resource IDs
- accept raw `workflow_json`
- write the assembled graph to `output_file`
- preserve execution metadata via `extra_data_json` / `partial_execution_targets`
- expose normalized execution fields like `workflow_id`, `outputs_count`, `preview_output_json`, and `execution_status_json`
- cancel queued or running executions during destroy when `cancel_on_delete = true`
- execute immediately, export only, or do both

### `comfyui_workspace`

`comfyui_workspace` is a layout-aware meta resource. It can:

- accept multiple API-format workflows, including `comfyui_workflow.*.assembled_json`
- compose them into one UI-oriented workspace/subgraph export
- position workflow islands with a typed, CSS-inspired `layout` block (`display`, `direction`, `gap`, `columns`, `origin_*`)
- style individual workflows via per-workflow `group_color` and `title_font_size` settings
- control internal node layout within each workflow via top-level `node_layout` settings (`mode`, `direction`, `column_gap`, `row_gap`)
- write the composed workspace JSON to `output_file`

Unlike `comfyui_workflow` file-only export, `comfyui_workspace` still needs a live ComfyUI connection so it can fetch node metadata from `/object_info` and build UI slot/widget information.

### Data sources

Use data sources to inspect live server state, normalized job state, queue entries, workflow history, output files, or provider metadata generated from the current ComfyUI extraction.

### Artifact round-trip and file lifecycle

The provider also exposes dedicated artifact-management surfaces for AI harness workflows:

- `comfyui_prompt_json` / `comfyui_workspace_json` import and normalize native ComfyUI artifacts.
- `comfyui_prompt_to_workspace` / `comfyui_workspace_to_prompt` translate between API prompt JSON and workspace JSON with explicit fidelity reporting.
- `comfyui_prompt_to_terraform` / `comfyui_workspace_to_terraform` synthesize canonical Terraform IR and rendered HCL from native artifacts using the generated node contract.
- `comfyui_prompt_validation` / `comfyui_workspace_validation` validate artifacts against live `/object_info` metadata, defaulting to executable modes that require output nodes. Fragment-only modes remain available when an AI harness is editing incomplete graphs.
- `comfyui_uploaded_image` and `comfyui_uploaded_mask` manage ComfyUI-backed uploads.
- `comfyui_output_artifact` downloads a remote ComfyUI file from `/view` into a Terraform-managed local artifact.

## Examples

- [Provider configuration](./examples/provider/provider.tf): minimal and explicit provider setup patterns
- [Text to image](./examples/resources/txt2img/main.tf): generated node resources assembled with `node_ids`
- [Image to image](./examples/resources/img2img/main.tf): transform an input image with prompt guidance
- [Upscale](./examples/resources/upscale/main.tf): run an upscale model and save the result
- [Workflow JSON](./examples/resources/workflow_json/main.tf): submit raw ComfyUI API-format JSON
- [Workflow file export](./examples/resources/workflow_file/main.tf): write assembled workflows to disk with or without execution
- [Workflow collections](./examples/resources/collection/main.tf): group workflows and emit an index manifest
- [Workspace export](./examples/resources/workspace/main.tf): compose multiple workflows into one layout-aware workspace export
- [Video generation](./examples/resources/video_gen/main.tf): partner-node video workflows for Kling and Seedance
- [Data sources](./examples/data-sources/main.tf): inspect system stats, queue state, job state, node metadata, history, outputs, and provider metadata

## Provider Configuration

The provider accepts three optional attributes:

- `host`: ComfyUI hostname or IP address
- `port`: ComfyUI port
- `api_key`: optional API key for authenticated deployments

Provider arguments take precedence over environment variables, which take precedence over the built-in defaults (`localhost:8188`, no API key). For the full provider schema and generated docs, see the [Terraform Registry documentation](https://registry.terraform.io/providers/StevenBuglione/comfyui/latest/docs).

## Development

The repo keeps contributor workflow intentionally short in the root README:

```bash
make generate
make test
make lint
make docs
make hooks-install
make workspace-e2e-browser-install
make workspace-e2e
make release-e2e-browser-install
make release-e2e
make synthesis-e2e
make execution-e2e
```

`make workspace-e2e` launches a disposable local ComfyUI from `third_party/ComfyUI` (auto-selecting a free local port if 8188 is busy), renders the `validation/workspace_e2e` stress fixtures through `comfyui_workspace`, and verifies the real browser canvas with Playwright. Evidence lands under `validation/workspace_e2e/artifacts/browser/` as screenshots and metrics JSON files. In this repo, "clean and usable" means each staged workspace loads in ComfyUI, every workflow group remains visible, and the captured metrics show zero cross-group overlaps, zero header overlaps, zero body containment violations, zero backward links, and correctly styled group rendering.

`make release-e2e` stages four canonical release scenarios into a live ComfyUI canvas and verifies them with Playwright: an assembled-resource workflow, a raw `workflow_json` import, a workspace->prompt->workspace round trip, and a `comfyui_workspace` gallery export. It is aimed squarely at the provider-owned logic that is hardest to trust by inspection alone: assembly, translation, staging, and graph connectivity.

`make synthesis-e2e` exercises the AI-facing synthesis surfaces with real Terraform. It feeds native prompt and workspace artifacts into `comfyui_prompt_to_terraform` and `comfyui_workspace_to_terraform`, then asserts that both return non-empty Terraform IR, rendered HCL, and the expected workflow/resource blocks without needing a live ComfyUI server.

`make execution-e2e` launches a disposable local ComfyUI, uploads a tiny fixture image, runs a model-free `LoadImage -> ImageInvert -> SaveImage` workflow, verifies the structured execution fields through `comfyui_workflow`, re-reads the same run through `comfyui_job` / `comfyui_jobs`, and downloads the saved artifact back to local disk.

Generated node resources come from extracted ComfyUI metadata and are checked in under `internal/resources/generated`. For deeper project structure and development guidance, see [CLAUDE.md](./CLAUDE.md).

The maintenance surface is therefore mostly in the hand-rolled layers, not the generated wrappers: roughly `19.7k` custom Go lines versus `99.4k` generated lines across `646` resource files. Release validation is documented in [docs/release-validation.md](./docs/release-validation.md).

## License

This project is licensed under the [MIT License](./LICENSE).
