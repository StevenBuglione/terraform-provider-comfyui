# Documentation Refresh Design

## Goal

Bring the provider documentation fully in line with the current product state so that three audiences can rely on it without inference:
- end users authoring and operating ComfyUI workflows with Terraform
- AI coding harnesses generating and maintaining Terraform workflow code
- contributors maintaining the provider, generation pipeline, and validation harnesses

The documentation should clearly separate narrative guidance from generated reference material, reflect all recent provider capabilities, and remove stale descriptions of older resource/data-source surfaces.

## Recommended approach

Use an audience-layered documentation architecture with one canonical landing page and a small set of narrative guides.

This is preferred over a README-heavy or full documentation-site restructure because it:
- keeps the root README useful and current without turning it into a handbook
- gives equal weight to provider users, AI harness authors, and contributors
- preserves generated resource/data-source docs as exact schema reference rather than forcing them to teach product concepts
- makes ongoing maintenance realistic as the provider evolves

## Target documentation architecture

### 1. `README.md`

`README.md` should be the canonical landing page and release-positioning document.

It should answer:
- what the provider is
- what is generated vs hand-rolled
- what the provider can do today
- why it is credible for both humans and AI harnesses
- what the main entrypoints are
- where to go next in the docs

### 2. Narrative guides under `docs/`

These should be the hand-maintained teaching surface.

They should be split by audience and concern rather than by raw API surface:
- getting started and workflow authoring for provider users
- AI harness authoring and maintainability guidance
- release validation and confidence guidance
- contributor and generation-architecture guidance
- known boundaries and scope statements

### 3. Generated reference docs under `docs/resources/` and `docs/data-sources/`

These remain the exact generated API reference surface.
They are not the primary teaching surface and should not be asked to carry product positioning or architectural explanation.

### 4. Superpowers design/plan docs under `docs/superpowers/`

These remain internal design history and execution records.
They should not be treated as user-facing documentation, but the narrative guides can summarize conclusions that were validated through them.

## Deliverables

### Update existing files

- `README.md`
- `docs/index.md`
- `docs/release-validation.md`
- `docs/ai-maintainability.md`

### Create new files

- `docs/getting-started.md`
- `docs/workflow-authoring.md`
- `docs/ai-harness-guide.md`
- `docs/contributing.md`
- `docs/generation-architecture.md`
- `docs/known-boundaries.md`

## Content responsibilities

### `README.md`

`README.md` should include:
- concise product positioning for all three audiences
- current capability summary only
- one strong quick-start example
- current core concepts (`comfyui_workflow`, `comfyui_workspace`, generated nodes, artifact and synthesis surfaces)
- current validation and generation story
- an explicit scope statement centered on pinned built-in ComfyUI support
- links to deeper guides

It should be trimmed where necessary so it does not duplicate entire guide sections.

### `docs/index.md`

`docs/index.md` should be reduced to a lightweight documentation map.

It should:
- route readers by audience
- point to the main narrative guides
- point to generated reference docs separately
- avoid restating large amounts of README content

### `docs/getting-started.md`

This should cover:
- installation and provider configuration
- the minimal runnable workflow path
- what `terraform plan` and `terraform apply` do in this provider
- how to inspect outputs and execution fields
- how strict plan-time inventory validation affects authoring

### `docs/workflow-authoring.md`

This should be the main end-user guide for building workflows.

It should explain:
- the node-per-resource mental model
- how `comfyui_workflow` works today
- how `comfyui_workspace` works today
- when to use assembled node resources vs raw `workflow_json`
- prompt/workspace import and translation surfaces
- export-only, execute-only, and execute-plus-export paths
- recommended authoring patterns and anti-patterns

### `docs/ai-harness-guide.md`

This should be the primary AI-facing guide.

It should explain:
- how to inspect the generated node contract with `comfyui_node_schema`
- how to use executable vs fragment validation correctly
- how to use `comfyui_prompt_to_terraform` and `comfyui_workspace_to_terraform`
- how to use `comfyui_inventory` for inventory-aware planning
- the recommended AI harness loop:
  inspect -> synthesize -> validate -> plan -> runtime/browser proof
- what the provider guarantees for pinned built-in ComfyUI support

### `docs/ai-maintainability.md`

This should become a narrower architecture and maintenance note.

It should focus on:
- what is generated from ComfyUI server metadata
- what is generated from live frontend UI behavior
- what remains hand-rolled and why
- how regeneration and pinned-version support work
- why this boundary is maintainable for AI-authored Terraform workflows

It should not be the only AI-facing guide; that role belongs to `docs/ai-harness-guide.md`.

### `docs/release-validation.md`

This should become the release-confidence guide.

It should describe:
- the exact validation lanes and what each proves
- local prerequisites
- where generated/browser/runtime artifacts are emitted
- how to read failures
- how the validation matrix supports the current release claim

It should explicitly include:
- `make generate`
- `make synthesis-e2e`
- `make inventory-plan-e2e`
- `make execution-e2e`
- `make workspace-e2e`
- `make release-e2e`

### `docs/contributing.md`

This should be the contributor workflow guide.

It should include:
- local setup expectations
- generator prerequisites
- docs generation expectations
- validation commands and when to run them
- generated vs hand-rolled code boundaries
- expectations for touching generated files, regeneration, and verification

### `docs/generation-architecture.md`

This should document the generated-first architecture.

It should explain:
- extraction from pinned ComfyUI server metadata
- frontend UI-hints extraction from the live ComfyUI frontend
- resource and schema generation outputs
- prompt/workspace/Terraform synthesis architecture
- dynamic inventory classification and strict plan-time validation generation
- what changes when the pinned ComfyUI version changes

### `docs/known-boundaries.md`

This should state what is intentionally in scope and what is not.

It should document:
- support is centered on pinned built-in ComfyUI behavior in this repo
- AI-authored Terraform workflow generation is a first-class supported path
- strict plan-time validation exists where dynamic inventory sources are recognized and supported
- unsupported dynamic expressions fail plan rather than silently degrade
- arbitrary exotic custom-node ecosystems are not the core release promise

## Current-state updates the refresh must reflect

The docs refresh must update all narrative documentation to match the current provider state, including:
- legacy `status`, `outputs`, and `error` fields were removed from `comfyui_workflow`
- rich execution fields on `comfyui_workflow` are the canonical execution surface
- prompt/workspace-to-Terraform synthesis data sources now exist
- validation now distinguishes fragment and executable modes, with executable modes as the default teaching path
- workspace sizing hints are generated from live ComfyUI frontend behavior rather than maintained as manual layout rules
- strict live plan-time validation exists for recognized dynamic inventory-backed inputs
- `comfyui_inventory` is part of the supported authoring loop
- AI-harness support is a first-class product story rather than an implicit side effect

## Editorial standards

The refresh should follow these principles:
- one canonical explanation per topic, then link instead of repeating
- no stale examples or references to removed fields
- generated reference docs should remain schema-focused, not tutorial-heavy
- narrative docs should explain intent, trade-offs, and current guarantees
- examples should reflect the current preferred user path, not legacy compatibility paths
- documentation should state boundaries explicitly rather than leaving users or AI harnesses to infer them

## Verification criteria

The documentation refresh should not be considered complete until:
- all narrative docs align with the current README and current generated reference docs
- no narrative docs refer to removed `comfyui_workflow` legacy fields
- AI-facing docs mention synthesis surfaces, executable validation defaults, and strict plan-time inventory validation
- contributor docs explain the current generation and validation workflow accurately
- doc links resolve correctly
- generated docs are refreshed with `make docs` if any source docs or templates require regeneration

## Outcome

After this refresh, the repo should have:
- a concise and accurate landing page
- clear audience-specific guidance for users, AI harnesses, and contributors
- explicit release and maintainability claims tied to real provider capabilities
- documentation that is aligned with the current generated-first architecture and current validation story
