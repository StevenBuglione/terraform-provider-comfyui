# Workspace Readability Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `comfyui_workspace` exports easier to read by reserving real header space for workflow titles, exposing proven-renderable group styling, and making left-to-right DAG node layout the default inside each workflow island without taking away top-level workflow placement control.

**Architecture:** Keep the existing two-layer model. Terraform still controls inter-workflow placement across the shared canvas, while the provider owns intra-workflow node layout and group serialization. Implement the feature in three passes: schema/config surface, internal layout engine, then browser-backed evidence and docs.

**Tech Stack:** Go 1.25, Terraform Plugin Framework, existing `comfyui_workspace` resource, Go unit tests, Terraform validation fixtures, Node.js + Playwright, local ComfyUI runtime harness.

---

## Spec Reference

- `docs/superpowers/specs/2026-04-10-workspace-readability-design.md`

## Recommended Worktree

Do the implementation in a dedicated worktree rooted from `main`.

```bash
git worktree add .worktrees/workspace-readability -b feat/workspace-readability main
cd .worktrees/workspace-readability
git submodule update --init --recursive
```

## File Structure

**Modify:**

- `internal/resources/workspace_resource.go`
  - Extend Terraform schema/model parsing for `workflows[].style` and `node_layout`
- `internal/resources/workspace_resource_test.go`
  - Schema/config validation coverage for the new surface
- `internal/resources/workspace_builder.go`
  - Header/body geometry, left-to-right DAG placement, style serialization
- `internal/resources/workspace_builder_test.go`
  - Failing and passing layout tests for header clearance, DAG ordering, containment, and style serialization
- `validation/workspace_e2e/fixtures.tf`
  - Add a branch-heavy readability fixture and a styled fixture
- `validation/workspace_e2e/workspaces.tf`
  - Wire `style` and `node_layout` into the generated workspace fixtures
- `validation/workspace_e2e/browser/tests/helpers/layout_metrics.ts`
  - Add header-overlap and body-containment metrics
- `validation/workspace_e2e/browser/tests/workspace_layout.spec.ts`
  - Assert title/header visibility, style rendering, and clearer layout
- `examples/resources/workspace/main.tf`
  - Show the new style and node-layout surface
- `README.md`
  - Note the readability/styling capability in the workspace workflow

**Generate / refresh:**

- `docs/resources/workspace.md` (via `make docs`, if generated docs change)

---

## Chunk 1: Schema Surface and Failing Contract Tests

### Task 1: Create the worktree and capture a clean baseline

**Files:**
- No repo file changes

- [ ] **Step 1: Create the worktree**

```bash
git worktree add .worktrees/workspace-readability -b feat/workspace-readability main
cd .worktrees/workspace-readability
git submodule update --init --recursive
```

- [ ] **Step 2: Run the current baseline checks**

Run:

```bash
go test ./internal/resources -run Workspace -v
make workspace-e2e
```

Expected:

- resource tests pass
- browser harness passes on the current baseline

- [ ] **Step 3: Commit nothing**

This task is setup only. Do not create a commit here.

### Task 2: Add failing schema tests for style and node-layout support

**Files:**
- Modify: `internal/resources/workspace_resource_test.go`

- [ ] **Step 1: Add a failing schema test for `workflows[].style`**

Add a test that expects these nested workflow attributes to exist:

```go
for _, attrName := range []string{"group_color", "title_font_size"} {
    if _, ok := styleAttr.Attributes[attrName]; !ok {
        t.Fatalf("expected style schema to include %q", attrName)
    }
}
```

- [ ] **Step 2: Add a failing schema test for `node_layout`**

Add a test that expects a top-level `node_layout` nested attribute with:

```go
for _, attrName := range []string{"mode", "direction", "column_gap", "row_gap"} {
    if _, ok := nodeLayoutAttr.Attributes[attrName]; !ok {
        t.Fatalf("expected node_layout schema to include %q", attrName)
    }
}
```

- [ ] **Step 3: Add failing validation tests**

Add tests that reject:

- `mode = "manual"`
- `direction = "top_to_bottom"`
- `style.group_color = ""`

- [ ] **Step 4: Run the focused tests to prove failure**

Run:

```bash
go test ./internal/resources -run 'WorkspaceResource|ValidateWorkspaceLayout' -v
```

Expected:

- FAIL because the new schema/config surface does not exist yet

- [ ] **Step 5: Commit the red test**

```bash
git add internal/resources/workspace_resource_test.go
git commit -m "test: cover workspace readability schema"
```

### Task 3: Implement the schema/model/config surface

**Files:**
- Modify: `internal/resources/workspace_resource.go`
- Modify: `internal/resources/workspace_resource_test.go`

- [ ] **Step 1: Extend the Terraform models**

Add concrete model structs:

```go
type workspaceWorkflowStyleModel struct {
    GroupColor    types.String `tfsdk:"group_color"`
    TitleFontSize types.Int64  `tfsdk:"title_font_size"`
}

type workspaceNodeLayoutModel struct {
    Mode      types.String  `tfsdk:"mode"`
    Direction types.String  `tfsdk:"direction"`
    ColumnGap types.Float64 `tfsdk:"column_gap"`
    RowGap    types.Float64 `tfsdk:"row_gap"`
}
```

- [ ] **Step 2: Add `style` under `workflows` and `node_layout` at the top level**

Follow the existing schema style in `workspace_resource.go`. Keep all new fields optional, but make the block presence valid only when each contained field validates.

- [ ] **Step 3: Add config structs used by the builder**

Add concrete config types with plain Go values:

```go
type workspaceWorkflowStyleConfig struct {
    GroupColor    string
    TitleFontSize int
}

type workspaceNodeLayoutConfig struct {
    Mode      string
    Direction string
    ColumnGap float64
    RowGap    float64
}
```

- [ ] **Step 4: Parse the new config in `workspaceConfigFromModel`**

Rules:

- default `node_layout.mode` to `dag`
- default `node_layout.direction` to `left_to_right`
- default style fields to empty/zero values so the builder can fill defaults

- [ ] **Step 5: Reject unsupported values with explicit errors**

Return diagnostics mentioning the exact field name, for example:

- `"node_layout.mode must be dag"`
- `"node_layout.direction must be left_to_right"`

- [ ] **Step 6: Re-run the focused tests**

Run:

```bash
go test ./internal/resources -run 'WorkspaceResource|ValidateWorkspaceLayout' -v
```

Expected:

- PASS for the new schema and config tests

- [ ] **Step 7: Commit**

```bash
git add internal/resources/workspace_resource.go internal/resources/workspace_resource_test.go
git commit -m "feat: add workspace readability config"
```

---

## Chunk 2: Internal Layout Engine and Serialization

### Task 4: Add failing builder tests for header clearance, DAG flow, containment, and style export

**Files:**
- Modify: `internal/resources/workspace_builder_test.go`

- [ ] **Step 1: Add a failing header-clearance test**

Add a test that builds a single workflow and asserts:

```go
groupTop := subgraph.Groups[0].Bounding[1]
firstNodeTop := subgraph.Nodes[0].Pos[1]
if firstNodeTop-groupTop < 80 {
    t.Fatalf("expected at least 80px of clearance, got %v", firstNodeTop-groupTop)
}
```

- [ ] **Step 2: Add a failing left-to-right DAG ordering test**

Use a fixture with:

- one source node
- two parallel branch nodes
- one merge node

Assert:

- branch nodes are to the right of the source
- merge node is to the right of both branches

- [ ] **Step 3: Add a failing containment test**

Assert that every node rectangle remains inside the group body:

```go
group := subgraph.Groups[0]
for _, node := range subgraph.Nodes {
    if node.Pos[1] < group.Bounding[1]+80 {
        t.Fatalf("node intrudes into header area: %+v", node)
    }
}
```

- [ ] **Step 4: Add a failing style serialization test**

Build a workflow with:

```go
style: workspaceWorkflowStyleConfig{
    GroupColor: "#ff00ff",
    TitleFontSize: 28,
}
```

Assert:

- `subgraph.Groups[0].Color == "#ff00ff"`
- `subgraph.Groups[0].FontSize == 28`

- [ ] **Step 5: Run the focused builder tests and confirm failure**

Run:

```bash
go test ./internal/resources -run 'BuildWorkspaceSubgraph|WorkspaceBuilder' -v
```

Expected:

- FAIL on header clearance and DAG ordering under the current builder

- [ ] **Step 6: Commit the red tests**

```bash
git add internal/resources/workspace_builder_test.go
git commit -m "test: cover workspace readability layout"
```

### Task 5: Implement header/body layout, left-to-right DAG placement, and style serialization

**Files:**
- Modify: `internal/resources/workspace_builder.go`
- Modify: `internal/resources/workspace_builder_test.go`

- [ ] **Step 1: Extend the workflow spec passed into the builder**

Add the parsed style and node-layout config to `workspaceWorkflowSpec` / builder inputs so the layout engine can see them.

- [ ] **Step 2: Add explicit defaults**

Introduce concrete defaults near the existing gap constants:

```go
const (
    defaultGroupHeaderHeight = 40.0
    defaultGroupBodyTopPad   = 40.0
    defaultNodeColumnGap     = 260.0
    defaultNodeRowGap        = 140.0
    defaultGroupFontSize     = 24
)
```

- [ ] **Step 3: Replace the simple per-workflow vertical stacking algorithm**

Implement a dedicated helper for v1:

```go
func layoutWorkflowNodesLeftToRight(...)
```

Use:

1. stable topological ordering
2. longest-upstream-path level assignment
3. average-parent-row preferred placement for merge nodes
4. downward collision resolution within each column

- [ ] **Step 4: Compute group bounds from header + body**

When building the workflow group:

- reserve at least `40px` header band
- reserve at least `40px` body-top padding below the header
- ensure the first node starts below `group_top + 80`

- [ ] **Step 5: Serialize only proven-renderable style fields**

Map:

- `style.group_color -> workspaceGroup.Color`
- `style.title_font_size -> workspaceGroup.FontSize`

Do not add unsupported styling keys.

- [ ] **Step 6: Re-run the focused builder tests**

Run:

```bash
go test ./internal/resources -run 'BuildWorkspaceSubgraph|WorkspaceBuilder' -v
```

Expected:

- PASS

- [ ] **Step 7: Re-run the full Go test suite**

Run:

```bash
go test ./... -v -timeout 120s
```

Expected:

- PASS

- [ ] **Step 8: Commit**

```bash
git add internal/resources/workspace_builder.go internal/resources/workspace_builder_test.go
git commit -m "feat: improve workspace readability layout"
```

### Task 6: Update Terraform fixtures to exercise the new contract

**Files:**
- Modify: `validation/workspace_e2e/fixtures.tf`
- Modify: `validation/workspace_e2e/workspaces.tf`

- [ ] **Step 1: Add a branch-heavy workflow definition**

Create a raw workflow with:

- one source/root section
- three parallel branches
- one merge/downstream section

- [ ] **Step 2: Add a styled workspace member**

Use non-default values such as:

```hcl
style = {
  group_color     = "#ff00ff"
  title_font_size = 28
}
```

- [ ] **Step 3: Add default `node_layout` coverage**

At least one fixture should omit `node_layout` entirely so the browser suite proves the new left-to-right default.

- [ ] **Step 4: Add explicit `node_layout` coverage**

At least one fixture should set:

```hcl
node_layout = {
  mode      = "dag"
  direction = "left_to_right"
  column_gap = 280
  row_gap    = 160
}
```

- [ ] **Step 5: Render the fixtures through Terraform**

Run:

```bash
./scripts/workspace-e2e/run.sh
```

Expected:

- generated JSON files are refreshed
- staged subgraphs are visible in ComfyUI

- [ ] **Step 6: Commit**

```bash
git add validation/workspace_e2e/fixtures.tf validation/workspace_e2e/workspaces.tf
git commit -m "test: add readability workspace fixtures"
```

---

## Chunk 3: Browser Evidence, Examples, Docs, and Final Verification

### Task 7: Extend browser metrics and assertions for readability

**Files:**
- Modify: `validation/workspace_e2e/browser/tests/helpers/layout_metrics.ts`
- Modify: `validation/workspace_e2e/browser/tests/workspace_layout.spec.ts`

- [ ] **Step 1: Add header-overlap metrics**

Extend the metrics helper to compute:

- each group header bottom (`group.y + 40` minimum baseline, or derived from serialized `font_size`)
- list of nodes whose rectangles intrude into that header band

- [ ] **Step 2: Add group-body containment metrics**

Record nodes that extend beyond:

- left/right body edges
- bottom edge

- [ ] **Step 3: Add style assertions**

Read the live first group object and assert:

- expected `color`
- expected `font_size`

- [ ] **Step 4: Update the Playwright spec**

For each workspace:

- assert no header-overlap violations
- assert no body-containment violations
- assert group visibility still passes
- keep the drag interaction
- keep screenshot + metrics artifact output

- [ ] **Step 5: Run the browser suite**

Run:

```bash
cd validation/workspace_e2e/browser
npx playwright test tests/workspace_layout.spec.ts --project=chromium
```

Expected:

- PASS
- fresh screenshots and metrics written under `validation/workspace_e2e/artifacts/browser/`

- [ ] **Step 6: Commit**

```bash
git add validation/workspace_e2e/browser/tests/helpers/layout_metrics.ts validation/workspace_e2e/browser/tests/workspace_layout.spec.ts
git commit -m "test: verify workspace readability in browser"
```

### Task 8: Refresh example and docs

**Files:**
- Modify: `examples/resources/workspace/main.tf`
- Modify: `README.md`
- Generate: docs via `make docs` if the provider docs change

- [ ] **Step 1: Update the workspace example**

Show:

- one workflow with `style`
- one workspace with explicit `node_layout`
- comments explaining that workflow placement and node layout are separate concerns

- [ ] **Step 2: Update the README**

Document:

- reserved title/header behavior
- renderable styling fields
- left-to-right default node layout

- [ ] **Step 3: Regenerate docs if needed**

Run:

```bash
make docs
```

Expected:

- generated docs update only if schema changes are reflected there

- [ ] **Step 4: Commit**

```bash
git add examples/resources/workspace/main.tf README.md docs
git commit -m "docs: describe workspace readability controls"
```

### Task 9: Final verification and cleanup

**Files:**
- No new source files

- [ ] **Step 1: Run the final Go validation**

Run:

```bash
go test ./... -v -timeout 120s
make lint
```

Expected:

- PASS

- [ ] **Step 2: Run the final browser-backed validation**

Run:

```bash
make workspace-e2e
```

Expected:

- PASS
- screenshots and metrics prove the readability improvements

- [ ] **Step 3: Review the final diff**

Run:

```bash
git --no-pager status
git --no-pager diff --stat
```

- [ ] **Step 4: Commit any remaining generated-doc changes**

```bash
git add -A
git commit -m "chore: finalize workspace readability rollout"
```

- [ ] **Step 5: Request code review**

Use `superpowers:requesting-code-review` before merging.
