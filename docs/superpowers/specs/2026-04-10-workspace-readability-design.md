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

Extend each `workflows` entry with an optional `style` block.

The v1 block should expose only styling that the live ComfyUI group renderer is proven to support:

- `group_color`
- `title_font_size`

Defaults:

- every field remains optional
- omitting the block produces a readable default container
- setting one field is allowed; the provider fills the rest with defaults

This design is intentionally constrained to renderable fields. The live ComfyUI group object currently serializes only `color` and `font_size`, and its `draw()` implementation uses that same single color for the header fill, body fill, border stroke, resize handle, and title text. Separate border/header/title color controls are therefore out of scope for v1 because they cannot be guaranteed to render.

### 3. Add internal node layout controls

Extend the resource with an internal `node_layout` block.

Default behavior:

- `mode = "dag"`
- `direction = "left_to_right"`

Initial surface:

- `mode` — only `dag`
- `direction` — only `left_to_right` in v1
- `column_gap`
- `row_gap`

The default should improve readability without requiring any new Terraform arguments.

Validation:

- reject any `mode` other than `dag`
- reject any `direction` other than `left_to_right`
- return explicit diagnostics when unsupported values are requested

## Provider Behavior Design

### 1. Split workflow container bounds into header and body

Replace the current single padding approach with explicit container geometry:

- header band: reserved title/styling area
- body area: node placement region

Rules:

- the first node must begin below the header band
- nodes must remain inside the workflow body area
- body padding remains separate from header padding

Default geometry:

- minimum header band height: `40px`
- default body top padding below the header band: `40px`
- therefore the first node must begin at least `80px` below the group top in the default case

This replaces the current effective `40px` clearance baseline that caused title crowding.

### 2. Default internal node layout: left-to-right DAG

Inside each workflow island:

1. parse the prompt graph and build dependency relationships
2. topologically order nodes with stable tie-breaking based on original prompt order
3. assign each node a horizontal level equal to the longest upstream path into that node
4. place source nodes in the leftmost column
5. place downstream nodes progressively to the right using `column_gap`
6. within each column, sort nodes by:
   - average parent row if the node has parents
   - otherwise original prompt order
7. for merge nodes, use the rounded average of parent rows as the preferred row anchor, then resolve collisions by shifting downward within the column
8. place sibling branches vertically using `row_gap`

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

The exported group JSON should include only styling values that the live ComfyUI group renderer reads and serializes.

### Supported mapping table

| Terraform field | Exported group field | Runtime effect |
| --- | --- | --- |
| `workflows[].style.group_color` | `groups[].color` | Drives the header fill, body fill, border stroke, resize handle, and title text color together |
| `workflows[].style.title_font_size` | `groups[].font_size` | Drives the title text size and header/titlebar height |

### Unsupported in v1

The following are deliberately excluded because the current live group renderer does not serialize or render them as distinct fields:

- separate border color
- separate header background color
- separate title text color
- border width
- border radius

Unsupported styling fields must not be added to the Terraform surface until a concrete ComfyUI runtime field and rendering path is identified.

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

- title/header collision regression using a fixture where the first node would have overlapped under the old `40px` clearance
- branch-heavy internal workflow readability using a workflow with three parallel branches merging into a shared downstream node
- styled workflow islands using a non-default `group_color` plus non-default `title_font_size`

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
- keep the v1 Terraform styling surface limited to renderable fields
- expand the styling surface only after tracing a concrete runtime field and draw path in ComfyUI

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
