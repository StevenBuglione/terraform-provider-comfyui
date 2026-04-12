# AI Harness Guide

This guide is for AI coding harnesses and the humans shaping them.

The goal is not to make the harness guess how ComfyUI works. The goal is to make the provider expose enough generated and validated contract surface that the harness can author Terraform workflows against explicit facts.

## Supported Promise

The main supported path is:

- built-in ComfyUI behavior pinned in this repo
- Terraform-authored workflows using generated node resources
- native prompt and workspace artifacts translated or synthesized through provider-owned surfaces
- strict validation against the live ComfyUI server where built-in runtime-backed inventories are recognized

That is the path this guide describes.

## 1. Inspect the Node Contract

Use [comfyui_node_schema](./data-sources/node_schema.md) as the machine-readable contract for built-in nodes.

It exposes:

- required and optional inputs
- output slots
- default values
- ranges and enum values
- whether an input is link-typed
- whether an input uses dynamic options
- `validation_kind`
- `inventory_kind`
- `supports_strict_plan_validation`

That means an AI harness can reason about node capabilities without parsing raw JSON strings or reverse-engineering the docs.

## 2. Use Inventory-Aware Planning

For built-in inputs backed by live runtime inventory, use [comfyui_inventory](./data-sources/inventory.md).

The provider normalizes recognized inventory-backed inputs into kinds such as:

- `checkpoints`
- `loras`
- `text_encoders`

Recommended loop:

1. Inspect the node schema for a candidate node and input.
2. If the input has `validation_kind = "dynamic_inventory"`, read `inventory_kind`.
3. Query `comfyui_inventory` for that kind.
4. Choose a live value from the returned inventory.

This keeps generated Terraform aligned with the target ComfyUI runtime before `terraform plan`.

## 3. Validate Executable Workflows by Default

The provider distinguishes fragment-oriented editing from executable workflows.

Use these validation surfaces:

- [comfyui_prompt_validation](./data-sources/prompt_validation.md)
- [comfyui_workspace_validation](./data-sources/workspace_validation.md)

Default guidance:

- use executable modes for normal authoring
- use fragment modes only when intentionally working on incomplete graphs

That matters because a structurally valid fragment is not the same thing as a runnable workflow.

## 4. Prefer Provider-Owned Synthesis

If the starting point is a native ComfyUI artifact, do not invent Terraform structure heuristically if the provider can synthesize it.

Use:

- [comfyui_prompt_to_terraform](./data-sources/prompt_to_terraform.md)
- [comfyui_workspace_to_terraform](./data-sources/workspace_to_terraform.md)

These return:

- canonical Terraform IR JSON
- rendered Terraform HCL
- fidelity classification
- preserved, synthesized, and unsupported field lists

This gives the harness a provider-owned canonical target instead of an ad hoc translation strategy.

## 5. Treat `terraform plan` as the Authoring Gate

For the supported built-in path, `terraform plan` should be the main gate before apply.

Why:

- static built-in enums are validated by generated Terraform schemas
- recognized runtime-backed inventory values are validated against the live ComfyUI server
- unsupported dynamic expressions fail plan rather than silently degrading

This makes plan success a meaningful confidence signal for AI-authored Terraform.

## 6. Use Runtime and Browser Proof Before Release

The provider includes runtime and Playwright-backed verification lanes because Terraform-only checks are not enough for a graph-oriented system.

Use:

- `make synthesis-e2e`
- `make inventory-plan-e2e`
- `make execution-e2e`
- `make workspace-e2e`
- `make release-e2e`

Together they prove:

- canonical synthesis works
- dynamic inventory validation fails where it should
- execution state and artifacts work
- workspace layout and connectivity hold in real ComfyUI
- provider-owned release scenarios load and round-trip correctly

## Recommended Harness Loop

The recommended authoring loop is:

1. Inspect
   Read `comfyui_node_schema` and, when needed, `comfyui_inventory`.
2. Synthesize or author
   Build Terraform directly or use `comfyui_prompt_to_terraform` / `comfyui_workspace_to_terraform`.
3. Validate
   Use executable validation by default.
4. Plan
   Require a clean `terraform plan` before considering the graph sound.
5. Prove
   Run synthesis/runtime/browser verification appropriate to the change scope.

## What Remains Hand-Rolled

The provider is generated-first, not generation-only.

Hand-rolled logic still exists in:

- Terraform resource and data source orchestration
- prompt, workspace, and Terraform IR translation
- validation semantics and diagnostics
- workspace staging and runtime/browser harnesses

The important point is that the harness should not need to reproduce those semantics on its own. It should consume them from the provider.

## Boundaries

This guide does not assume support for every arbitrary custom-node ecosystem.

The supported claim is narrower and more defensible:

- built-in ComfyUI behavior pinned in this repo
- AI-authored Terraform workflows against generated node contracts
- strict validation where the provider can prove correctness

For the broader product boundary, see [Known Boundaries](./known-boundaries.md).
