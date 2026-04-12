# Dynamic Inventory Plan Validation Design

## Goal

Make `terraform plan` a strict correctness gate for built-in nodes supported by the pinned ComfyUI version in this repo.

For any generated resource input backed by:

- a static ComfyUI enum
- a live ComfyUI runtime inventory such as checkpoints, LoRAs, text encoders, VAEs, ControlNets, or similar built-in asset categories

a successful `terraform plan` must mean the configured value is valid for the target ComfyUI server and the workflow is executable according to provider validation rules.

## Problem Statement

The provider already handles one important class of validation well:

- static built-in enum values are generated into Terraform validators at schema time

Example:

- `comfyui_byte_dance_text_to_video_node.model`

is generated as a fixed enum and rejected during plan if an invalid value is supplied.

The remaining gap is dynamic built-in runtime inventories. Many stock ComfyUI nodes expose choices from sources such as:

- `folder_paths.get_filename_list('checkpoints')`
- `folder_paths.get_filename_list('loras')`
- `folder_paths.get_filename_list('text_encoders')`
- `folder_paths.get_filename_list('controlnet')`

Today the provider preserves that source metadata, but it does not yet enforce that the selected value actually exists on the configured ComfyUI server during `terraform plan`.

That means a plan can still succeed for invalid built-in runtime-backed values and fail later during workflow execution. This violates the desired contract.

## Desired Contract

For built-in nodes supported by the pinned ComfyUI version in this repository:

1. `terraform plan` must fail for any invalid static enum input.
2. `terraform plan` must fail for any invalid dynamic inventory-backed input.
3. `terraform plan` must fail if the provider cannot reach or resolve the live ComfyUI inventory needed to validate referenced dynamic inputs.
4. `terraform plan` must fail if a workflow is not executable under executable validation rules.
5. A successful `terraform plan` should mean any later runtime failure is either:
   - an upstream execution/runtime fault
   - a provider bug
   - an environmental issue outside configuration validity

There should be no best-effort fallback for dynamic built-in inventory validation.

## Non-Goals

This design does not aim to:

- support third-party custom nodes outside the pinned built-in ComfyUI surface
- heuristically infer semantic model compatibility beyond what the pinned ComfyUI code explicitly exposes
- validate arbitrary dynamic expressions that cannot be normalized into a known inventory contract
- allow plan-time success when the provider cannot prove correctness

## Design Principles

### 1. Generated First

Any validation behavior that can be derived from pinned ComfyUI code should be generated from extracted metadata, not hand-maintained in provider code.

### 2. Strict Over Best-Effort

If the provider cannot strictly validate a referenced built-in dynamic input, plan must fail.

### 3. Shared Runtime Validation Service

Live inventory lookups should go through one provider-owned inventory layer with request-scoped caching, not ad hoc resource logic.

### 4. Deterministic Upgrade Path

If ComfyUI changes how built-in dynamic options are sourced, regeneration should update provider behavior automatically wherever possible and fail loudly where it cannot.

## Current State

The provider already has the foundation needed for this feature:

- generated node specs in `scripts/extract/node_specs.json`
- structured node contracts via `comfyui_node_schema`
- generated resource schemas for static enums
- executable workflow validation defaults

Generated metadata already preserves fields such as:

- `dynamic_options`
- `dynamic_options_source`
- explicit static option lists for fixed combos

Examples already visible in generated docs:

- static enum: `comfyui_byte_dance_text_to_video_node.model`
- dynamic runtime inventory: `comfyui_checkpoint_loader_simple.ckpt_name`

The missing piece is turning recognized built-in dynamic option sources into strict plan-time live validators.

## Architecture

### Validation Kinds

Each generated input must be classified into exactly one validation kind:

1. `static_enum`
2. `dynamic_inventory`
3. `dynamic_expression`
4. `freeform`

#### `static_enum`

Used when ComfyUI exposes a fixed option list in pinned source metadata.

Behavior:

- generate Terraform schema validators like `stringvalidator.OneOf(...)`

Examples:

- ByteDance `model`
- ByteDance `resolution`
- scheduler and aspect-ratio style built-in combos with explicit values

#### `dynamic_inventory`

Used when ComfyUI exposes runtime-backed choices from a known inventory source that the provider can resolve live.

Behavior:

- generate a strict live validator
- validator queries the live ComfyUI inventory for the normalized inventory kind
- plan fails if the selected value is not present

Examples:

- checkpoints
- loras
- text_encoders
- controlnet
- diffusion_models
- hypernetworks
- style_models
- clip_vision
- audio_encoders
- other recognized `folder_paths.get_filename_list(...)` categories

#### `dynamic_expression`

Used when ComfyUI exposes a dynamic choice that is not a flat built-in inventory and cannot yet be safely normalized into a strict live contract.

Behavior:

- generated metadata marks the input as unsupported for strict plan-time validation
- configurations that reference such an input must fail plan with a diagnostic explaining why validation cannot yet be guaranteed

This preserves the “plan means correct” contract.

#### `freeform`

Used for values that are intentionally unrestricted beyond type/range/shape validation.

Behavior:

- rely on normal generated scalar validation only

Examples:

- `STRING`
- numeric ranges
- booleans
- link inputs

### Generated Input Contract

The generated node contract must be expanded so every input includes:

- `validation_kind`
- `dynamic_options_source`
- `inventory_kind`
- `supports_strict_plan_validation`

`inventory_kind` should be normalized whenever possible from extracted source.

Examples:

- `folder_paths.get_filename_list('checkpoints')` -> `checkpoints`
- `folder_paths.get_filename_list('loras')` -> `loras`
- `folder_paths.get_filename_list('text_encoders')` -> `text_encoders`

This mapping must be derived in the extraction/generation pipeline from pinned ComfyUI code patterns, not hand-maintained in runtime provider logic.

## Generated-First Extraction Model

### Server Extraction

The extractor should continue reading the pinned ComfyUI source and existing spec generation pipeline, but it must now additionally normalize recognized dynamic option sources into inventory contracts.

The output artifact should preserve:

- original dynamic source expression
- normalized validation kind
- normalized inventory kind when recognized
- whether strict validation is supported

This should live inside `scripts/extract/node_specs.json` and downstream generated Go metadata.

### Normalization Rules

The extraction pipeline should classify dynamic sources using generated pattern logic such as:

- `folder_paths.get_filename_list('<name>')` -> `dynamic_inventory` with `inventory_kind=<name>`
- known constant lists exposed by built-in ComfyUI code -> `static_enum` when they are fully enumerable at extraction time
- anything else -> `dynamic_expression`

If upstream changes a built-in source pattern and the extractor no longer recognizes it, generation should downgrade that input to `dynamic_expression` and surface it clearly in generated artifacts and tests.

That is preferable to silently accepting incorrect plans.

## Provider Runtime Architecture

### Shared Inventory Service

Add one provider-owned live inventory service responsible for:

- querying the configured ComfyUI server for runtime-backed built-in inventories
- normalizing responses by `inventory_kind`
- caching results for the lifetime of a Terraform operation
- returning deterministic diagnostics when inventory lookup fails

This service must be reusable by:

- generated plan validators on resources
- `comfyui_prompt_validation`
- `comfyui_workspace_validation`
- workflow preflight checks if needed

### Inventory Query Surface

The provider should rely on a single internal client surface for runtime inventory resolution.

Whether the underlying ComfyUI lookup is implemented via:

- `/object_info`
- a built-in listing endpoint
- existing API structures
- or a provider-owned resolution shim over stock ComfyUI behavior

the public contract remains the same: inventory kinds resolve to concrete valid values on the configured server.

If a required inventory kind cannot be resolved, plan must fail.

## Plan-Time Validation Behavior

### Generated Resource Schemas

Generated schemas should behave as follows:

- `static_enum`: compile in existing `OneOf(...)` validation
- `dynamic_inventory`: attach generated live validators bound to normalized `inventory_kind`
- `dynamic_expression`: attach a strict validator that fails with a clear unsupported diagnostic
- `freeform`: use existing scalar validators only

### Workflow and Artifact Validation

`comfyui_prompt_validation` and `comfyui_workspace_validation` must also enforce dynamic inventory correctness when validating executable workflows.

That means validation cannot stop at:

- node presence
- link correctness
- static types
- output-node existence

It must also ensure that every referenced built-in dynamic inventory-backed value exists on the live server.

### Failure Semantics

Plan must fail when:

- a configured value is missing from the live inventory
- the inventory source is required but unreachable
- a dynamic built-in input resolves to an unsupported validation kind
- executable validation fails

There should be no warning-only path for these conditions.

## Public Surfaces

### `comfyui_node_schema`

Expand the structured node schema surface to expose:

- `validation_kind`
- `supports_strict_plan_validation`
- `inventory_kind`

for each input where relevant.

This allows AI harnesses to reason explicitly about:

- which values are static
- which require live server inventory
- which are not yet supported for strict plan correctness

### Optional Inventory Data Source

Add a public data source for machine reasoning and debugging, for example:

- `comfyui_inventory`

This should expose live built-in runtime inventories by category.

This data source is not the primary enforcement mechanism. The primary enforcement remains generated plan-time validation. But the public inventory surface gives AI harnesses better observability and debugging support.

## Testing and Proof

### Unit Tests

Add extraction and generation tests that prove:

- known built-in static enums become `static_enum`
- known built-in inventory-backed inputs become `dynamic_inventory`
- unknown dynamic expressions become `dynamic_expression`
- normalized `inventory_kind` values are emitted correctly

Add runtime validation tests that prove:

- a missing checkpoint fails plan
- a missing LoRA fails plan
- an unavailable ComfyUI server fails plan for referenced dynamic inventory inputs
- unsupported dynamic expressions fail plan

### Acceptance and Harness Tests

Add validation fixtures that prove:

- a valid built-in loader configuration plans successfully
- an invalid built-in loader configuration fails at plan time
- executable workflow validation catches missing runtime-backed assets before apply

### Regression Guarantees

Extend `make generate` and generated-clean verification so changes in dynamic source classification show up as explicit diffs in generated artifacts and docs.

This ensures upstream ComfyUI changes produce reviewable contract changes rather than silent behavior drift.

## Upgrade Story

This design is specifically intended to be resilient to pinned ComfyUI updates.

When the pinned ComfyUI version changes:

1. extraction re-reads built-in node definitions
2. dynamic option sources are reclassified
3. generated metadata is updated
4. generated resource schemas update automatically
5. plan-time validation behavior changes deterministically with the new generated contract

If upstream introduces a new built-in dynamic source pattern that the extractor does not yet understand, generation should:

- classify it as unsupported
- make strict plan validation fail for that field
- force provider maintainers to extend the extractor deliberately

That keeps the safety contract intact.

## Implementation Summary

This work should add:

- generated validation-kind metadata per input
- normalized built-in inventory-kind metadata
- shared provider runtime inventory service
- generated live validators for dynamic inventory-backed inputs
- strict failure semantics for unsupported dynamic expressions
- expanded `comfyui_node_schema`
- optional public `comfyui_inventory`
- stronger prompt/workspace executable validation using live inventory checks

## Success Criteria

This design is successful when all of the following are true:

1. Invalid static built-in combos fail during `terraform plan`.
2. Invalid built-in runtime-backed model names fail during `terraform plan`.
3. Referenced runtime-backed dynamic inputs fail plan if live inventory cannot be reached.
4. Unknown or unsupported dynamic built-in validation cases fail plan rather than silently deferring to runtime.
5. `make generate` is the authoritative path for updating validation behavior after pinned ComfyUI changes.
6. AI harnesses can inspect generated schema metadata and know whether a field is statically enumerable, inventory-backed, or unsupported for strict validation.

