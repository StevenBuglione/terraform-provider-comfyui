# AI Maintainability

The provider now exposes a generated-first contract intended for AI coding harnesses that need to create and maintain ComfyUI workflows declaratively in Terraform.

## Machine-readable surfaces

- `comfyui_node_schema` exposes structured generated node metadata instead of raw JSON strings.
- `comfyui_prompt_validation` and `comfyui_workspace_validation` default to executable modes so incomplete graphs fail unless fragment validation is explicitly requested.
- `comfyui_prompt_to_terraform` synthesizes canonical Terraform IR and rendered HCL from native prompt JSON.
- `comfyui_workspace_to_terraform` translates workspace JSON to prompt form and then synthesizes canonical Terraform IR and rendered HCL.

## Generated-first boundary

The provider uses ComfyUI as the source of truth for:

- node resource schemas and node contracts extracted from server metadata
- frontend sizing hints extracted from the running ComfyUI UI
- canonical Terraform synthesis built against the generated node contract

Handwritten logic is intentionally limited to:

- Terraform resource and data source orchestration
- prompt, workspace, and Terraform IR translation
- validation/reporting semantics
- runtime and browser verification harnesses

## Verification lanes

- `make generate`
  Regenerates node resources, UI hints, and generated node-schema metadata from real ComfyUI behavior.
- `make synthesis-e2e`
  Proves prompt/workspace-to-Terraform synthesis through real Terraform runs.
- `make workspace-e2e`
  Validates workspace rendering, spacing, connectivity, and group layout in a real ComfyUI browser session.
- `make release-e2e`
  Validates canonical provider-owned release scenarios through Terraform plus Playwright.
- `make execution-e2e`
  Proves model-free workflow execution and artifact lifecycle behavior against a disposable ComfyUI runtime.

## Intended AI workflow

1. Inspect `comfyui_node_schema` for structured input and output contracts.
2. Use executable validation modes by default while authoring or refactoring workflows.
3. Use `comfyui_prompt_to_terraform` or `comfyui_workspace_to_terraform` when starting from native ComfyUI artifacts.
4. Re-run `make synthesis-e2e` and browser/runtime harnesses before release.
