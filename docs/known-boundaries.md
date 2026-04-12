# Known Boundaries

This page states the intended support boundary clearly so users, AI harnesses, and contributors do not have to infer it from examples or commit history.

## Core Support Promise

The provider is primarily aimed at:

- the built-in ComfyUI behavior pinned in this repo
- Terraform-authored workflows using generated node resources
- native prompt and workspace artifacts translated through provider-owned surfaces
- AI-authored Terraform workflows using generated contracts, synthesis, and validation surfaces

## What the Provider Tries to Guarantee

For the supported built-in path, the provider aims to make these statements true:

- built-in node schemas are generated from the pinned ComfyUI source
- AI and human authors can inspect the same structured node contract
- executable validation is available and should be the default for runnable graphs
- recognized built-in runtime-backed inventory values are validated during `terraform plan`
- unsupported dynamic expressions fail plan rather than silently degrading
- provider-owned translation, synthesis, and workspace export behavior are proved through runtime and browser validation lanes

## What Is Not the Central Promise

These are not the core release promise:

- perfect support for arbitrary exotic custom-node ecosystems
- semantic model compatibility inference beyond what stock ComfyUI exposes directly
- guaranteeing every possible native artifact in the wild can be synthesized into Terraform with perfect lossless fidelity

The provider may still work in some of those cases, but that is not the main compatibility target.

## Dynamic Inventory Boundary

Strict plan-time validation exists where the provider can prove correctness from generated metadata and live ComfyUI inventory.

That means:

- recognized built-in inventory-backed inputs should fail plan when the chosen value is not live on the target server
- unsupported dynamic expressions should fail clearly instead of being treated as safely validated

This is intentional. A strict failure is better than letting an apparently valid plan fail later at runtime.

## AI Harness Boundary

The current AI-facing promise is:

- an AI harness can inspect built-in node contracts with `comfyui_node_schema`
- it can query live inventory with `comfyui_inventory`
- it can synthesize Terraform from native prompt/workspace artifacts
- it can validate executable workflows before apply
- it can rely on the provider’s generated-first contracts for the pinned built-in ComfyUI version

That is the target story. The harness is not expected to reverse-engineer ComfyUI or invent missing provider semantics on its own.

## Upgrade Boundary

When the pinned ComfyUI version changes, the repo still depends on regeneration and revalidation.

The important distinction is:

- generated schema drift should be surfaced by generation
- hand-rolled orchestration drift should be surfaced by the validation matrix

So the main long-term risk is upgrade drift, not whether the current pinned version has enough contract surface for supported built-in workflows.
