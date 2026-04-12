# Workflow Authoring

This guide explains the current preferred authoring model for building ComfyUI workflows with Terraform.

## The Mental Model

The provider uses a node-per-resource model.

- Generated `comfyui_*` node resources represent built-in ComfyUI nodes in Terraform state.
- Those node resources are virtual. They do not make API calls on their own.
- `comfyui_workflow` assembles those node definitions into ComfyUI API-format JSON and optionally executes them.
- `comfyui_workspace` composes one or more workflows into a UI-oriented workspace export.

If you are used to authoring raw ComfyUI prompt JSON, think of Terraform resources as a typed declarative front-end for the same graph.

## When to Use `node_ids`

Use `node_ids` on `comfyui_workflow` when:

- you are authoring a workflow directly in Terraform
- you want typed resource schemas for every node
- you want Terraform references between node outputs and downstream inputs
- you want strict plan-time validation where the provider can support it

This is the default and preferred provider-native authoring path.

## When to Use `workflow_json`

Use raw `workflow_json` on `comfyui_workflow` when:

- you already have native ComfyUI prompt JSON
- you are importing or replaying an existing workflow
- you need to preserve a prompt artifact before translating it into Terraform

`workflow_json` is still a first-class path, but it is no longer the only serious authoring mode.

## `comfyui_workflow` Today

The current `comfyui_workflow` resource can:

- assemble from `node_ids`
- accept raw `workflow_json`
- validate against live `/object_info` metadata before queueing execution
- write assembled prompt JSON to `output_file`
- submit immediately or export only
- wait for completion or return after queueing
- preserve richer execution data from `/api/jobs`

The canonical execution fields are:

- `workflow_id`
- `outputs_count`
- `preview_output_json`
- `outputs_json`
- `execution_status_json`
- `execution_error_json`

The older coarse compatibility fields are not part of the current contract.

## `comfyui_workspace` Today

Use `comfyui_workspace` when you want a ComfyUI canvas-oriented export rather than just prompt JSON.

It is the right tool for:

- composing multiple workflows into one workspace
- laying out workflow islands on a shared canvas
- exporting deterministic workspace/subgraph JSON
- producing artifacts that are intended to be opened visually in ComfyUI

`comfyui_workspace` is layout-aware and depends on live ComfyUI metadata to reconstruct widget and UI-oriented information.

The main controls are:

- `layout`
  positions workflow islands on the canvas
- `node_layout`
  controls readability within each workflow
- per-workflow styling
  such as `group_color` and `title_font_size`

## Artifact Import and Translation

The provider supports two native ComfyUI artifact forms:

- prompt JSON
- workspace JSON

Use these data sources when starting from native artifacts:

- [comfyui_prompt_json](./data-sources/prompt_json.md)
- [comfyui_workspace_json](./data-sources/workspace_json.md)
- [comfyui_prompt_to_workspace](./data-sources/prompt_to_workspace.md)
- [comfyui_workspace_to_prompt](./data-sources/workspace_to_prompt.md)

These surfaces are useful when you need to normalize or translate artifacts before execution or before synthesis into Terraform.

## Prompt and Workspace to Terraform

Use these when you want the provider to own the mapping from native ComfyUI artifacts into canonical Terraform:

- [comfyui_prompt_to_terraform](./data-sources/prompt_to_terraform.md)
- [comfyui_workspace_to_terraform](./data-sources/workspace_to_terraform.md)

They return:

- canonical Terraform IR JSON
- rendered Terraform HCL
- fidelity reporting showing which parts were preserved, synthesized, or unsupported

That is the preferred path for AI harnesses or migration tooling starting from existing ComfyUI artifacts.

## Validation Modes

Validation is no longer one-size-fits-all.

Use:

- executable validation when the graph is meant to run now
- fragment validation when the graph is intentionally incomplete during editing or translation

The relevant surfaces are:

- [comfyui_prompt_validation](./data-sources/prompt_validation.md)
- [comfyui_workspace_validation](./data-sources/workspace_validation.md)

For most authoring flows, executable validation should be the default.

## Inventory-Aware Authoring

Some built-in node inputs are backed by live server inventory rather than fixed enums.

Examples include:

- checkpoints
- LoRAs
- text encoders
- other runtime-discovered model categories

For recognized built-in inventory-backed inputs:

- the provider exposes the normalized inventory kind through `comfyui_node_schema`
- `terraform plan` validates chosen values against the live ComfyUI inventory
- [comfyui_inventory](./data-sources/inventory.md) lets you inspect available values directly

This is the preferred way to keep authored workflows aligned with the real server state.

## Recommended Patterns

- Use generated node resources plus `node_ids` for provider-native workflows.
- Use `workflow_json` when importing existing prompt artifacts.
- Use synthesis data sources when you want canonical Terraform produced from native artifacts.
- Use executable validation by default.
- Use `comfyui_inventory` when choosing built-in dynamic model inputs.
- Export to `output_file` when you need a prompt artifact for debugging or sharing.
- Use `comfyui_workspace` when the output is intended for the ComfyUI canvas.

## Anti-Patterns

- Treating generated reference docs as the only authoring guide.
- Assuming a successful `terraform plan` means all arbitrary dynamic expressions are supported.
- Hardcoding inventory-backed model names without checking the live server when you are authoring against a specific runtime.
- Using workspace export as a substitute for prompt validation when what you need is an executable graph.

## Where to Go Next

- [Getting Started](./getting-started.md)
- [AI Harness Guide](./ai-harness-guide.md)
- [comfyui_workflow resource reference](./resources/workflow.md)
- [comfyui_workspace resource reference](./resources/workspace.md)
