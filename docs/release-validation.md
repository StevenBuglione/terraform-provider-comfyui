# Release Validation

This guide explains what the repo’s validation lanes prove and how to interpret failures before a release.

## Why Multiple Lanes Exist

This provider is mostly generated node wrappers plus a smaller hand-rolled core. The generated surface gives broad schema coverage, but the risky behavior lives in the orchestration layer:

- workflow assembly
- semantic validation
- prompt and workspace translation
- Terraform synthesis
- dynamic inventory validation
- execution-state handling
- workspace staging and browser-visible layout

That is why the validation strategy is layered rather than relying on one test command.

## Validation Matrix

### `make generate`

Regenerates:

- node resources
- structured node schema metadata
- frontend UI-hints used by workspace layout

This is the first drift detector when the pinned ComfyUI version changes.

### `make synthesis-e2e`

Proves the AI-facing synthesis surfaces:

- `comfyui_prompt_to_terraform`
- `comfyui_workspace_to_terraform`

It verifies that native prompt and workspace artifacts synthesize into non-empty Terraform IR and rendered HCL through real Terraform runs.

### `make inventory-plan-e2e`

Proves strict plan-time validation for recognized built-in dynamic inventories.

It stages a known inventory value in a disposable ComfyUI runtime, then asserts:

- live inventory is discoverable through `comfyui_inventory`
- generated node schema exposes the expected inventory metadata
- a valid `terraform plan` succeeds
- an invalid runtime-backed selection fails during plan before apply

### `make execution-e2e`

Proves the execution-oriented path without depending on an external model.

It verifies:

- workflow submission
- execution metadata
- `/api/jobs`-backed state reads
- output artifact download

### `make workspace-e2e`

Proves workspace builder behavior in a real browser against a disposable ComfyUI runtime.

It checks:

- workflow-group visibility
- layout integrity
- node and group spacing
- link counts and directionality
- absence of geometry regressions such as overlaps or containment failures

### `make release-e2e`

Proves the canonical provider-owned release scenarios in real ComfyUI.

It covers:

- assembled-resource workflows
- raw `workflow_json` import
- workspace and prompt round-trip behavior
- workspace export layout and connectivity

## Local Prerequisites

Install browser dependencies before running the Playwright lanes:

```bash
make workspace-e2e-browser-install
make release-e2e-browser-install
```

Then use the validation lanes appropriate to the change.

For broad release confidence, the most useful sequence is:

```bash
make generate
make docs
go test ./... -timeout 120s
make synthesis-e2e
make inventory-plan-e2e
make execution-e2e
make workspace-e2e
make release-e2e
```

## Artifact Locations

The runtime and browser lanes emit evidence under `validation/`.

Most useful locations:

- `validation/workspace_e2e/artifacts/generated/`
- `validation/workspace_e2e/artifacts/browser/`
- `validation/release_e2e/artifacts/generated/`
- `validation/release_e2e/artifacts/browser/`

Terraform machine-readable outputs land under each harness runtime directory as `terraform-outputs.json`.

## How to Read Failures

Common failure patterns:

- `make generate` changes files unexpectedly
  - usually means the pinned ComfyUI extraction contract or generated metadata drifted
- synthesis-e2e failures
  - usually point to prompt/workspace translation or Terraform synthesis regressions
- inventory-plan-e2e failures
  - usually point to dynamic inventory classification, inventory lookup, or plan-time validation regressions
- execution-e2e failures
  - usually point to execution-state handling, artifact paths, or output lifecycle regressions
- workspace-e2e or release-e2e metric failures
  - usually point to layout, staging, connectivity, or browser-visible compatibility regressions

## Current Release Claim

These lanes are intended to support a narrow but strong release claim:

- built-in ComfyUI behavior pinned in this repo is represented by generated node contracts
- AI- and human-authored Terraform workflows can be validated against those contracts
- recognized runtime-backed inventory choices are caught during plan
- provider-owned assembly, translation, execution, and workspace export behavior are proved against real ComfyUI runtime and browser behavior

For the explicit product boundary, see [Known Boundaries](./known-boundaries.md).
