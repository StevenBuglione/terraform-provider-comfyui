# Release Validation

This provider is mostly generated node wrappers plus a smaller hand-rolled core.

Approximate maintenance surface today:
- hand-rolled Go under `internal/` excluding `internal/resources/generated`: about `19.7k` lines
- generated Go under `internal/resources/generated`: about `99.4k` lines across `646` files

That split matters for upgrades. When ComfyUI changes node specs, most of the provider can be regenerated. The riskier upgrade points are the hand-rolled layers:
- workflow assembly and validation
- execution-state parsing and overlay logic
- prompt/workspace translation
- workspace building and staging
- upload/output/artifact handling
- browser-visible assumptions about the ComfyUI canvas

## Validation Lanes

Use these lanes together for release confidence:

- `make execution-e2e`
  - validates the execution-oriented file and metadata path
  - exercises `comfyui_workflow`, `comfyui_job`, `comfyui_jobs`, `comfyui_output`, and `comfyui_output_artifact`
- `make workspace-e2e`
  - validates staged workspaces in real ComfyUI with Playwright
  - checks node/group counts, layout integrity, link counts, and link health
- `make release-e2e`
  - validates canonical release scenarios in real ComfyUI with Playwright
  - covers assembled-resource workflows, raw JSON import, translation round-trips, and workspace-builder output

## Browser Prerequisites

Install Playwright dependencies before the browser lanes:

```bash
make workspace-e2e-browser-install
make release-e2e-browser-install
```

## Runtime Notes

The workspace and release harnesses start a local ComfyUI runtime under their own `.runtime/` directories and stage generated JSON files as global subgraphs.

Artifacts are written here:
- `validation/workspace_e2e/artifacts/browser/`
- `validation/workspace_e2e/artifacts/generated/`
- `validation/release_e2e/artifacts/browser/`
- `validation/release_e2e/artifacts/generated/`

Terraform machine-readable outputs are written under each harness runtime directory as `terraform-outputs.json`.

## Reading Failures

Use the emitted artifacts to localize failures quickly:
- missing or malformed JSON in `artifacts/generated/` usually points to provider-owned assembly, translation, or workspace-builder logic
- Terraform assertion failures usually point to execution-state or artifact-path regressions
- Playwright metric mismatches usually point to staging, link preservation, layout, or ComfyUI UI compatibility issues
