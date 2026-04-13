---
page_title: "ComfyUI Provider"
description: |-
  Overview, configuration, guides, and generated reference documentation for the Terraform provider for ComfyUI.
---

# ComfyUI Provider

The ComfyUI provider combines generated node contracts with hand-written orchestration, artifact, validation, and translation surfaces.

Use it to:

- model ComfyUI nodes as typed Terraform resources instead of hand-editing prompt JSON
- assemble and optionally execute workflows through `comfyui_workflow`
- compose prompt workflows into editor-oriented canvases with `comfyui_workspace`
- materialize prompt, workspace, and subgraph artifacts on disk
- inspect live runtime inventory, queue state, jobs, and history
- translate prompt and workspace artifacts into canonical Terraform IR and rendered HCL

## Capability Model

| Surface | What it covers |
|---|---|
| Generated node resources | `645` generated `comfyui_*` node resources derived from pinned ComfyUI metadata. These resources are virtual modeling surfaces used for workflow assembly. |
| Orchestration resources | `comfyui_workflow`, `comfyui_workflow_collection`, and `comfyui_workspace` assemble, group, export, and optionally execute workflows. |
| Artifact resources | `comfyui_prompt_artifact`, `comfyui_workspace_artifact`, `comfyui_subgraph`, `comfyui_uploaded_image`, `comfyui_uploaded_mask`, and `comfyui_output_artifact` manage local and remote artifacts. |
| Runtime and schema data sources | `comfyui_inventory`, `comfyui_job`, `comfyui_jobs`, `comfyui_output`, `comfyui_queue`, `comfyui_subgraph_catalog`, `comfyui_subgraph_definition`, `comfyui_system_stats`, `comfyui_workflow_history`, `comfyui_node_info`, `comfyui_node_schema`, and `comfyui_provider_info`. |
| Translation and validation data sources | `comfyui_prompt_json`, `comfyui_workspace_json`, `comfyui_prompt_to_workspace`, `comfyui_workspace_to_prompt`, `comfyui_prompt_to_terraform`, `comfyui_workspace_to_terraform`, `comfyui_prompt_validation`, and `comfyui_workspace_validation`. |

## Provider Configuration

```terraform
# ComfyUI Provider Configuration
#
# The provider connects to a running ComfyUI server instance.
# All connection parameters can be set via environment variables:
#   COMFYUI_HOST                               — bare host or full URL (default: "localhost")
#   COMFYUI_PORT                               — server port when host is not a full URL (default: 8188)
#   COMFYUI_API_KEY                            — optional API key for authenticated servers
#   COMFYUI_COMFY_ORG_AUTH_TOKEN               — optional partner auth token for partner-backed nodes
#   COMFYUI_COMFY_ORG_API_KEY                  — optional partner API key for partner-backed nodes
#   COMFYUI_DEFAULT_WORKFLOW_EXTRA_DATA_JSON   — optional default workflow extra_data JSON
#   COMFYUI_UNSUPPORTED_DYNAMIC_VALIDATION_MODE — error | warning | ignore

terraform {
  required_providers {
    comfyui = {
      source  = "StevenBuglione/comfyui"
      version = "~> 0.18"
    }
  }
}

# Minimal: relies entirely on environment variables / defaults
provider "comfyui" {}

# Explicit: useful for multi-server setups or CI/CD
# provider "comfyui" {
#   host                                = var.comfyui_host
#   port                                = var.comfyui_port
#   api_key                             = var.comfyui_api_key
#   comfy_org_auth_token                = var.comfyui_comfy_org_auth_token
#   comfy_org_api_key                   = var.comfyui_comfy_org_api_key
#   unsupported_dynamic_validation_mode = "warning"
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

variable "comfyui_comfy_org_auth_token" {
  description = "Partner auth token for comfy_org-backed execution (leave empty if not required)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "comfyui_comfy_org_api_key" {
  description = "Partner API key for comfy_org-backed execution (leave empty if not required)"
  type        = string
  default     = ""
  sensitive   = true
}
```

## Versioning Policy

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

## Start Here

- [Getting Started](./guides/getting-started.md)
  Install the provider, connect to ComfyUI, run a minimal workflow, and inspect execution results.
- [Workflow Authoring](./guides/workflow-authoring.md)
  Build workflows with generated node resources, `comfyui_workflow`, `comfyui_workspace`, and the artifact translation surfaces.
- [AI Harness Guide](./guides/ai-harness-guide.md)
  Use generated node contracts, live inventory, synthesis, and validation to author Terraform safely with AI tooling.

## Architecture and Boundaries

- [Known Boundaries](./guides/known-boundaries.md)
  Explicit support boundary for built-in ComfyUI workflows, dynamic inventory validation, and AI-facing guarantees.
- [Generation Architecture](./guides/generation-architecture.md)
  How metadata is extracted from the pinned ComfyUI server and live frontend behavior.
- [AI Maintainability](./guides/ai-maintainability.md)
  Why the generated-first boundary is maintainable for AI-authored Terraform workflows.

## Contributor and Release Guides

- [Contributing](./guides/contributing.md)
  Local workflow, regeneration steps, documentation expectations, and verification guidance.
- [Release Validation](./guides/release-validation.md)
  What each verification lane proves and how to localize failures.

## Generated Reference Docs

Use the generated references when you need exact schema details and runnable examples:

- [Data Sources](./data-sources/)
- [Resources](./resources/)

## Key Boundaries to Know

- Generated node resources are mostly virtual. They model ComfyUI nodes in Terraform state and are assembled later by `comfyui_workflow`.
- The main supported path is the built-in ComfyUI surface pinned in this repository, not every arbitrary custom-node ecosystem.
- Strict plan-time validation is strongest where generated metadata and live ComfyUI inventory make correctness provable.

For the full support statement, see [Known Boundaries](./guides/known-boundaries.md).

## Related Repo Material

- [README](../README.md)
  Product overview, quick start, and high-signal links.
- [Examples](../examples/)
  Runnable Terraform examples for common provider patterns.
- [Superpowers specs and plans](./superpowers/)
  Internal design history and execution records.
