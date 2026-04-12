# Release Validation Design

**Date:** 2026-04-11

## Goal

Establish a release-confidence validation harness for the hand-rolled parts of the ComfyUI Terraform provider. The harness should prove that provider-owned logic works against a real ComfyUI runtime and browser UI, without attempting to exhaustively re-test every generated node wrapper.

## Scope Boundary

This validation effort should certify the provider-owned custom surface:

- workflow assembly from Terraform resources into ComfyUI prompt JSON
- prompt and workspace translation logic
- workflow execution-state enrichment and related data-source reads
- workspace building, grouping, layout, and staging
- artifact and upload/download resources
- browser-visible load fidelity in real ComfyUI for generated workspaces and imported workflows

It should not attempt to prove that every generated resource wrapper works independently. Generated node resources are treated as inputs used by the scenarios, not as the primary subject of testing.

## Maintenance Surface

The current codebase is mostly generated, but the maintenance risk is concentrated in the hand-rolled code:

- hand-rolled Go under `internal/` excluding `internal/resources/generated`: about `19,681` lines
- generated Go under `internal/resources/generated`: about `99,436` lines across `646` files

This means future ComfyUI upgrades are likely to be low-risk when only node schemas change and high-risk when upstream response shapes, workspace JSON, or UI structure changes. The release suite should therefore be organized around the custom layers most likely to break during upgrades.

## Recommended Validation Strategy

Use a small number of scenario-driven release fixtures rather than a broad matrix of tiny tests. Each scenario should stress multiple hand-rolled subsystems at once and produce evidence in three forms:

1. Terraform/runtime assertions against provider surfaces.
2. Playwright browser assertions against real ComfyUI.
3. Golden artifacts for release review.

This provides stronger release confidence than unit coverage alone and avoids building an unmaintainable test zoo.

## Canonical Release Scenarios

### 1. Assembly-heavy execution workflow

A large Terraform-built workflow using generated node resources plus provider-owned assembly, validation, prompt submission, jobs/history enrichment, and output surfaces.

This scenario should prove:

- assembled prompt JSON is valid and stable
- `comfyui_workflow`, `comfyui_job`, `comfyui_jobs`, `comfyui_workflow_history`, and output-related resources agree on core identifiers and metadata
- execution-state overlays preserve rich fields correctly

### 2. Raw `workflow_json` import and round-trip workflow

A large hand-authored JSON workflow submitted through `comfyui_workflow`.

This scenario should prove:

- direct JSON mode stays coherent with provider-owned data sources
- imported workflows can be re-read and inspected through provider surfaces
- browser-visible graph structure remains intact when loaded in real ComfyUI

### 3. Workspace builder stress case

Several very large `comfyui_workspace` definitions with dense groups, mixed layout overrides, cross-group links, and staged subgraphs.

This scenario should prove:

- workspaces stage correctly into ComfyUI
- nodes and groups remain visible and non-overlapping
- grouping, header/body containment, and link directionality are preserved
- browser screenshots and metrics are stable enough for release review

### 4. Translation round-trip corpus

Prompt JSON -> workspace -> prompt and workspace JSON -> prompt -> workspace paths using pathological graphs with fan-out/fan-in, sparse input slots, reroutes, and metadata-heavy nodes.

This scenario should prove:

- the translation layer preserves logical graph structure
- sparse inputs and metadata survive round trips
- generated artifacts are stable for regression review

### 5. Artifact and remote-file path

Upload inputs, execute a saving workflow, read outputs back, download artifacts, and verify local file metadata and provider surfaces.

This scenario should prove:

- uploaded file resources behave correctly
- output discovery and artifact downloads work end-to-end
- resource/data-source metadata is consistent

## Evidence Model

Every canonical scenario should emit evidence in three forms.

### Terraform/runtime assertions

Fixtures should assert invariants such as:

- prompt IDs and workflow IDs match across resources and data sources
- outputs are present where expected
- staged subgraph IDs are discoverable
- sparse slots and metadata remain preserved
- artifact files exist and have expected metadata

### Playwright UI assertions

Browser checks should validate:

- every expected workflow/workspace is discoverable in the real ComfyUI UI
- expected node and group counts match fixture metadata
- no clipped or hidden nodes
- no group/body/header overlap violations
- representative links for fan-in, fan-out, and cross-group wiring are present
- screenshots and metrics are emitted as review artifacts

### Golden artifacts

Artifacts under `validation/.../artifacts/` should include:

- generated prompt/workspace JSON
- Terraform outputs JSON
- Playwright metrics JSON
- browser screenshots
- downloaded artifacts where applicable

## Upgrade Safety Model

This suite is intended to fail quickly when upstream changes affect the provider-owned layers:

- client response parsing for queue/history/jobs
- execution-state overlay logic
- workflow assembly and validation
- prompt/workspace translation
- workspace builder and staging behavior
- artifact/upload/download handling
- browser-level assumptions about the ComfyUI canvas DOM

If a future ComfyUI release changes only node specs, regeneration should be cheap. If it changes runtime payloads or UI behavior, this suite should identify which custom layer broke.

## Success Criteria

The release suite is successful when:

- it exercises all major hand-rolled subsystems through real runtime scenarios
- it uses Playwright to confirm browser-visible graph fidelity in ComfyUI
- it produces stable machine-readable evidence for CI and manual release review
- failures point clearly to a provider-owned layer rather than leaving regressions ambiguous
