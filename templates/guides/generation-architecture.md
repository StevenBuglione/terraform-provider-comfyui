---
page_title: "Generation Architecture - ComfyUI Provider"
subcategory: ""
description: |-
  Understand how pinned ComfyUI source, live frontend behavior, and generated metadata drive the provider's node contracts, validation, and workspace layout.
---

# Generation Architecture

This repo is built around a generated-first architecture. The provider tries to extract as much contract surface as possible from the pinned ComfyUI source and live frontend behavior, then keeps the handwritten layer focused on orchestration.

## Source of Truth Layers

### Pinned ComfyUI server code

The pinned ComfyUI source under `third_party/ComfyUI/` is used to extract:

- built-in node definitions
- input and output contracts
- enum values and ranges
- dynamic option metadata
- inventory-backed dynamic option sources

That extraction pipeline ultimately feeds `scripts/extract/node_specs.json`.

### Live ComfyUI frontend behavior

The provider also extracts UI-oriented hints from the running frontend.

This is used for:

- node sizing hints
- workspace readability and spacing behavior

That keeps the workspace builder aligned with real ComfyUI rendering behavior rather than hand-maintained layout constants.

## Main Pipeline

At a high level:

1. extract metadata from the pinned ComfyUI source
2. extract frontend UI hints from a running ComfyUI frontend
3. write generated artifacts
4. run the Go generator
5. regenerate provider resource and schema code

The canonical entrypoint is:

```bash
make generate
```

## Generated Outputs

The generation path produces several categories of output:

- generated node resource files under `internal/resources/generated/`
- generated node-schema metadata used by `comfyui_node_schema`
- generated UI-hints metadata used by workspace layout
- generated validation metadata such as `validation_kind`, `inventory_kind`, and strict-plan-validation support markers

## Dynamic Inventory Generation

Dynamic inventory validation is generation-driven rather than manually curated.

The extractor classifies inputs into categories such as:

- `static_enum`
- `dynamic_inventory`
- `dynamic_expression`
- `freeform`

For recognized built-in dynamic inventory sources, the generator normalizes them into inventory kinds such as:

- `checkpoints`
- `loras`
- `text_encoders`

That generated metadata then drives:

- `comfyui_node_schema`
- live inventory lookup through `comfyui_inventory`
- strict plan-time validation for supported built-in dynamic inventory inputs

If a dynamic expression is not recognized as a supported inventory-backed input, the provider should fail plan rather than pretend it can validate it safely.

## Synthesis Architecture

The provider also owns canonical translation from native ComfyUI artifacts into Terraform-oriented representations.

That includes:

- prompt JSON normalization
- workspace JSON normalization
- prompt/workspace translation
- Terraform IR synthesis
- rendered HCL synthesis

These surfaces exist so the provider, not an external AI harness, owns the canonical mapping from native ComfyUI artifacts to Terraform.

## Hand-Written Layer

The following parts are intentionally hand-rolled:

- Terraform provider wiring
- client logic
- workflow and workspace orchestration
- validation semantics and diagnostics
- translation and synthesis logic
- runtime/browser verification harnesses

The design goal is not to eliminate handwritten code completely. The goal is to keep handwritten code bounded and driven by generated metadata wherever possible.

## When the Pinned ComfyUI Version Changes

Upgrading the pinned ComfyUI version should follow this pattern:

1. update the pinned ComfyUI source
2. run `make generate`
3. inspect generated diffs
4. run `make docs`
5. run the relevant verification lanes

Expected drift from ComfyUI upgrades usually shows up in:

- generated resource/schema diffs
- generated UI-hints diffs
- dynamic inventory classification changes
- failures in synthesis, execution, or browser validation if the hand-rolled layer no longer matches upstream behavior

## Why This Matters

This architecture is what makes the provider credible for AI-authored Terraform workflows:

- node contracts are extracted rather than guessed
- inventory metadata is generated rather than manually listed
- workspace sizing reflects real frontend behavior
- synthesis and validation sit on top of generated contracts instead of duplicating them ad hoc

For contributor workflow details, see [Contributing](./contributing.md).
