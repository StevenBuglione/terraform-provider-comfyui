# Workspace Readability and Styling Design

## Summary

Improve `comfyui_workspace` so exported workflow islands are easier to read in the ComfyUI canvas. The provider should reserve visible header space for each workflow title, expose Terraform-controlled container styling, and make a left-to-right DAG node layout the default inside each workflow island while preserving the existing top-level workspace placement controls.

## Problem Statement

The current workspace export is functional but visually cramped in real ComfyUI:

- the first node sits too close to the workflow container title area
- workflow container styling is not configurable through Terraform
- nodes inside each workflow island are vertically stacked, which causes overlapping and hard-to-follow connections in dense graphs

Fresh Playwright evidence was captured during planning:

- `validation/workspace_e2e/artifacts/browser/plan-review/dense_grid.png`
- `validation/workspace_e2e/artifacts/browser/plan-review/wide_gallery.png`
- `validation/workspace_e2e/artifacts/browser/plan-review/dense_grid.summary.json`
- `validation/workspace_e2e/artifacts/browser/plan-review/wide_gallery.summary.json`

Those summaries show only `40px` of clearance between the group top and the first node in the first workflow island, which matches the observed title/header crowding.

## Goals

1. Keep workflow titles readable by reserving explicit header space above the node body.
2. Let Terraform style workflow containers with good defaults and optional deeper control.
3. Make internal workflow-node placement clearer by default with left-to-right DAG layout.
4. Preserve the current workspace-level control over where workflow islands are placed on the shared canvas.
5. Prove the improvements with Playwright screenshots and browser-derived layout assertions.

## Non-Goals

- Do not replace top-level workspace island placement with a fully automatic global canvas solver.
- Do not add a full theming system beyond workflow container presentation.
- Do not attempt custom link routing beyond what is needed to materially improve readability.
- Do not change the existing `comfyui_workflow` resource contract.

## User-Facing Design

### 1. Preserve current workspace island placement

Keep the current `comfyui_workspace` placement surface:

- workspace `layout` block
- `display`, `direction`, `gap`, `columns`, `origin_x`, `origin_y`
- per-workflow `x` / `y` overrides

This remains the mechanism for arranging multiple workflows across the shared ComfyUI canvas.

### 2. Add workflow presentation controls

Extend each `workflows` entry with an optional presentation block, tentatively named `style`.

The block should support:

- `background_color`
- `border_color`
- `border_width`
- `border_radius`
- `header_background_color`
- `title_color`
- `title_font_size`

Defaults:

- every field remains optional
- omitting the block produces a readable default container
- setting only one or two fields is allowed; the provider fills the rest with defaults

### 3. Add internal node layout controls

Extend the resource with an internal node layout block, tentatively named `node_layout`.

Default behavior:

- `mode = "dag"`
- `direction = "left_to_right"`

Initial surface:

- `mode` — initially only `dag`
- `direction` — initially `left_to_right`, with room for future `top_to_bottom`
- `column_gap`
- `row_gap`

The default should improve readability without requiring any new Terraform arguments.

## Provider Behavior Design

### 1. Split workflow container bounds into header and body

Replace the current single padding approach with explicit container geometry:

- header band: reserved title/styling area
- body area: node placement region

Rules:

- the first node must begin below the header band
- nodes must remain inside the workflow body area
- body padding remains separate from header padding

### 2. Default internal node layout: left-to-right DAG

Inside each workflow island:

1. parse the prompt graph and build dependency relationships
2. assign nodes to horizontal levels based on upstream dependencies
3. place source nodes in the leftmost column
4. place downstream nodes progressively to the right
5. stack sibling branches vertically within a column
6. center or rebalance merge nodes as needed for readability

Expected outcome:

- source-to-sink flow is visually left-to-right
- branches are easier to follow
- connections overlap less than in the current vertical stack

### 3. Containment and overflow protection

After computing node positions:

- measure node extents
- size the workflow body from those extents plus body padding
- compute final group bounds from header + body
- shift nodes if needed so no node intrudes into the header band or exits the body region

### 4. Workflow island placement remains separate

After internal layout is computed for each workflow island:

- measure the final island extents
- use the existing workspace-level layout algorithm to place workflow islands on the full canvas
- continue honoring explicit per-workflow `x` / `y` overrides

This preserves the existing “best of both worlds” behavior:

- provider decides how nodes are arranged inside each workflow
- Terraform decides how workflows are arranged on the overall canvas

## Serialization Design

The exported group JSON should include the chosen styling values through the existing group structure where possible.

Current JSON already supports `workspaceGroup.Color`; the implementation should map the richer Terraform styling surface into the ComfyUI-exported group shape as far as the runtime supports it. Where ComfyUI group JSON has limits, the provider should:

- serialize supported styling fields directly
- keep unsupported fields out of the exported JSON rather than inventing incompatible keys
- document any styling fields that are provider-level abstractions versus directly serialized values

## Validation Strategy

### Go tests

Add or extend unit tests for:

- reserved header clearance above the first node
- bounded node placement inside the workflow body
- left-to-right ordering for simple chains, branches, and merges
- style/config parsing from Terraform models
- JSON export for supported group styling

### Browser evidence

Extend the Playwright harness so it proves:

- workflow titles remain visible
- no node overlaps the header band
- nodes remain inside their workflow island body
- configured style changes appear in the rendered workspace
- left-to-right placement improves branch readability in dense workflows

### Fixtures

Reuse the current browser-e2e fixtures and add focused cases for:

- title/header collision regression
- branch-heavy internal workflow readability
- styled workflow islands with distinct header/background treatment

## Implementation Boundaries

The work naturally splits into four bounded units:

1. **Schema/config unit**
   - extend Terraform models and validation for `style` and `node_layout`
2. **Internal layout unit**
   - compute header/body spacing and left-to-right DAG placement inside each workflow island
3. **Serialization unit**
   - convert the computed layout and supported styling into exported ComfyUI group/node JSON
4. **Evidence unit**
   - extend Playwright and Go tests to prove the behavior

Each unit should remain understandable and testable on its own.

## Risks and Mitigations

### Risk: richer styling exceeds ComfyUI group JSON support

Mitigation:

- serialize only supported fields
- keep defaults readable even if some styling options are provider-side only
- document any field with runtime limitations

### Risk: DAG layout becomes too aggressive for hand-positioned workflows

Mitigation:

- keep top-level workflow island placement controls unchanged
- scope automatic behavior to internal node placement only
- expose spacing knobs in `node_layout`

### Risk: branch-heavy graphs still produce confusing crossings

Mitigation:

- validate with targeted branch-heavy fixtures
- tune level assignment and vertical ordering before expanding scope

## Recommended Rollout

1. Add failing Go and Playwright evidence for header overlap and poor branch readability.
2. Extend schema for styling and node-layout configuration.
3. Refactor internal workflow placement to reserve header space and compute left-to-right DAG layout.
4. Serialize supported styling into workspace export JSON.
5. Re-run the browser harness and refresh screenshots/metrics.

## Planning Readiness

This spec is ready for implementation planning once reviewed. It defines the user-facing surface, internal behavior split, evidence expectations, and non-goals clearly enough to break into implementation steps.
