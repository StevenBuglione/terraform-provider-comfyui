---
page_title: "Release Validation - ComfyUI Provider"
subcategory: ""
description: |-
  Understand what each validation lane proves, where evidence is written, and how to interpret failures before release.
---

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
make docs-validate
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

## Provider Versioning

Provider versions follow the **ComfyUI compatibility line** model:

- Provider `0.18.x` is the compatibility line for ComfyUI `v0.18.5`
- The first release in this line is `v0.18.5`
- Later provider-only fixes (bug fixes, documentation improvements, non-schema changes) increment the patch version: `v0.18.6`, `v0.18.7`, etc.
- The exact upstream pin remains authoritative in `generated.ComfyUIVersion` and the `comfyui_provider_info` data source
- Users should constrain the provider with `~> 0.18` for this line
- If the pinned upstream ComfyUI version changes materially (e.g., to `v0.19.0`), a new provider line (`0.19.x`) is started rather than silently continuing `0.18.x`

This ensures users can trust that provider `0.18.x` releases maintain compatibility with their ComfyUI `v0.18.5` workflows while receiving provider-level improvements.

## Release Procedures

This section documents the repeatable release procedures for the `0.18.x` compatibility line.

Run these procedures from a **clean worktree**. `make release-preflight` includes `docs-check`, so it expects the generated docs and other release-prep changes to already be committed.

### First Release: v0.18.5

The initial release in the `0.18.x` line requires comprehensive validation since it establishes the baseline for the compatibility line.

**Prerequisites:**

1. ComfyUI submodule is pinned to `v0.18.5`
2. Generated metadata reflects ComfyUI `v0.18.5` (check `scripts/extract/node_specs.json`)
3. Browser dependencies installed:
   ```bash
   make workspace-e2e-browser-install
   make release-e2e-browser-install
   ```

**Procedure:**

1. **Confirm generated metadata alignment:**
   ```bash
   cd third_party/ComfyUI && git describe --tags
   # Should output: v0.18.5
   
   grep comfyui_version scripts/extract/node_specs.json
   # Should contain: "comfyui_version": "v0.18.5"
   ```

2. **Update CHANGELOG.md:**
   - Replace `YYYY-MM-DD` with actual release date in the `[0.18.5]` entry
   - Verify all features, enhancements, and notes are accurate
   - Ensure version constraint recommendation matches `~> 0.18`

3. **Run local preflight validation:**
   ```bash
   make release-preflight VERSION=v0.18.5
   ```
   
   This runs:
   - `make verify` (fmt-check, generate, vet, lint, test)
   - `./scripts/validate-release-version.sh v0.18.5` (version alignment checks)

4. **Run comprehensive validation lanes:**
   
   Since this is the first release in the line, validate all surfaces:
   
   ```bash
   # Core generation and documentation
   make generate
   make docs
   make docs-validate
   
   # Unit tests
   go test ./... -timeout 120s
   
   # E2E validation lanes
   make synthesis-e2e       # Prove AI synthesis surfaces
   make inventory-plan-e2e  # Prove runtime inventory validation
   make execution-e2e       # Prove workflow execution path
   make workspace-e2e       # Prove workspace layout in browser
   make release-e2e         # Prove canonical release scenarios
   ```

5. **Review validation artifacts:**
   
   Check for regressions or unexpected failures:
   - `validation/workspace_e2e/artifacts/browser/` (screenshots, traces)
   - `validation/release_e2e/artifacts/browser/` (screenshots, traces)
   - Terraform outputs in each lane's `terraform-outputs.json`

6. **Tag and push:**
   ```bash
   git tag -a v0.18.5 -m "Release v0.18.5: Initial 0.18.x compatibility line for ComfyUI v0.18.5"
   git push origin v0.18.5
   ```

7. **Monitor GitHub Actions:**
   - The `release.yml` workflow triggers on tag push
   - GoReleaser builds binaries for all platforms
   - Artifacts are GPG-signed and published to GitHub Releases
   - Verify `terraform-provider-comfyui_<version>_SHA256SUMS` includes every zip asset **and** `terraform-provider-comfyui_<version>_manifest.json`; Terraform Registry rejects the release if the manifest is uploaded but not represented in the checksum file

8. **Publish to Terraform Registry:**
   - After successful GitHub release, submit to Terraform Registry
   - Include release notes explaining the compatibility line model:
     - This is the initial release for ComfyUI `v0.18.5` compatibility
     - Users should use `version = "~> 0.18"` to stay within this line
     - Later `0.18.x` patches will maintain ComfyUI `v0.18.5` compatibility

### Later Patch Releases: v0.18.6+

Provider-only patches increment the patch version while maintaining the same ComfyUI `v0.18.5` pin.

**Allowed changes in-line:**
- Provider bug fixes (client, workflow resource, data sources)
- Documentation improvements
- Validation tightening (stricter plan-time checks)
- Workflow assembly/execution fixes
- Pipeline/tooling fixes
- CI/CD improvements

**Changes requiring a new compatibility line:**
- Material ComfyUI pin change (e.g., updating to `v0.19.0`)
- Breaking schema changes from upstream node modifications
- New generated node resources from ComfyUI updates

**Procedure:**

1. **Confirm ComfyUI pin unchanged:**
   ```bash
   cd third_party/ComfyUI && git describe --tags
   # MUST output: v0.18.5 (same as initial release)
   
   grep comfyui_version scripts/extract/node_specs.json
   # MUST contain: "comfyui_version": "v0.18.5"
   ```

2. **Update CHANGELOG.md:**
   - Add new `[0.18.6]` section (or appropriate patch version) under `[Unreleased]`
   - Document bug fixes, enhancements, or changes
   - Note that the release maintains ComfyUI `v0.18.5` compatibility
   - Example:
     ```markdown
     ## [0.18.6] - YYYY-MM-DD
     
     Provider-only patch release for ComfyUI `v0.18.5` compatibility line.
     
     ### BUG FIXES
     
     * Fixed workflow resource state handling for ... (#123)
     * Corrected inventory validation edge case for ... (#124)
     
     ### NOTES
     
     * Maintains ComfyUI `v0.18.5` compatibility
     * No schema changes - safe upgrade within `~> 0.18` constraint
     ```

3. **Run local preflight validation:**
   ```bash
   make release-preflight VERSION=v0.18.6
   ```

4. **Run targeted validation lanes:**
   
   Focus on lanes relevant to the changes:
   
   - **For workflow/execution fixes:** Run `make execution-e2e` and `make release-e2e`
   - **For inventory/validation changes:** Run `make inventory-plan-e2e`
   - **For synthesis changes:** Run `make synthesis-e2e`
   - **For workspace changes:** Run `make workspace-e2e` and `make release-e2e`
   - **For documentation-only changes:** Run `make docs-validate`
   
   Always run at minimum:
   ```bash
   make verify                  # Core validation
   go test ./... -timeout 120s  # Unit tests
   ```

5. **Tag and push:**
   ```bash
   git tag -a v0.18.6 -m "Release v0.18.6: Provider fixes for ComfyUI v0.18.5 line"
   git push origin v0.18.6
   ```

6. **Monitor and publish:**
   - Monitor GitHub Actions `release.yml` workflow
   - After successful GitHub release, publish to Terraform Registry
   - Do not mutate an already-published GitHub release to fix missing required assets; cut the next patch release in the same compatibility line instead
   - Release notes should clarify:
     - This is a provider-only patch for the `0.18.x` line
     - Still targets ComfyUI `v0.18.5`
     - Safe upgrade for users on `version = "~> 0.18"`

### When to Start a New Compatibility Line

Start a new compatibility line (e.g., `0.19.x`) when:

1. **Updating ComfyUI pin materially:**
   ```bash
   ./scripts/update-comfyui.sh v0.19.0
   # Regenerate resources
   python3 scripts/extract/extract_v1_nodes.py third_party/ComfyUI > v1.json
   python3 scripts/extract/extract_v3_nodes.py third_party/ComfyUI > v3.json
   python3 scripts/extract/merge_specs.py v1.json v3.json > scripts/extract/node_specs.json
   make generate
   ```

2. **Breaking schema changes from upstream** that affect existing resources

3. **Major feature additions** that change the provider's compatibility contract

The new line starts with version matching the upstream ComfyUI version (e.g., `v0.19.0` for ComfyUI `v0.19.0`), and follows the same release procedures documented here.
