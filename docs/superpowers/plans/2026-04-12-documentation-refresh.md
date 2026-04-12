# Documentation Refresh Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update the provider's narrative documentation, README, and supporting guides so that the repo accurately reflects the current product state for end users, AI harness authors, and contributors.

**Architecture:** Keep `README.md` as the concise landing page, move deeper narrative guidance into hand-maintained `docs/` guides organized by audience and responsibility, and preserve generated docs as exact API reference. Align every narrative document with the generated-first architecture, current synthesis/validation surfaces, and strict plan-time inventory validation story.

**Tech Stack:** Markdown, Terraform provider docs structure, existing generated docs, GNU Make validation commands, git.

---

## File map

### Narrative docs to modify
- Modify: `README.md`
- Modify: `docs/index.md`
- Modify: `docs/release-validation.md`
- Modify: `docs/ai-maintainability.md`

### New narrative docs to create
- Create: `docs/getting-started.md`
- Create: `docs/workflow-authoring.md`
- Create: `docs/ai-harness-guide.md`
- Create: `docs/contributing.md`
- Create: `docs/generation-architecture.md`
- Create: `docs/known-boundaries.md`

### Specs and plans
- Reference: `docs/superpowers/specs/2026-04-12-documentation-refresh-design.md`
- Create: `docs/superpowers/plans/2026-04-12-documentation-refresh.md`

### Verification targets
- Read: `docs/data-sources/*.md`
- Read: `docs/resources/workflow.md`
- Read: `docs/resources/workspace.md`
- Read: `README.md`
- Run: `make docs`
- Run: `rg -n "comfyui_workflow\.(status|outputs|error)|\bstatus\b.*comfyui_workflow|\boutputs\b.*comfyui_workflow|\berror\b.*comfyui_workflow" README.md docs examples`

## Chunk 1: Audit and outline the current documentation surface

### Task 1: Capture the current narrative docs and generated reference anchors

**Files:**
- Modify: `docs/superpowers/plans/2026-04-12-documentation-refresh.md`
- Read: `README.md`
- Read: `docs/index.md`
- Read: `docs/release-validation.md`
- Read: `docs/ai-maintainability.md`
- Read: `docs/data-sources/node_schema.md`
- Read: `docs/data-sources/inventory.md`
- Read: `docs/data-sources/prompt_to_terraform.md`
- Read: `docs/data-sources/workspace_to_terraform.md`
- Read: `docs/data-sources/prompt_validation.md`
- Read: `docs/data-sources/workspace_validation.md`
- Read: `docs/resources/workflow.md`
- Read: `docs/resources/workspace.md`

- [ ] **Step 1: Write the plan file with the approved scope and file map**

Create `docs/superpowers/plans/2026-04-12-documentation-refresh.md` with the approved goal, architecture, file map, and chunk structure from this implementation plan.

- [ ] **Step 2: Record current-state documentation anchors**

In the plan file, add a short audit section that lists the current narrative docs and the generated reference docs that the rewrite must stay aligned with.

- [ ] **Step 3: Verify the plan file exists and is staged cleanly**

Run: `test -f docs/superpowers/plans/2026-04-12-documentation-refresh.md && git diff -- docs/superpowers/plans/2026-04-12-documentation-refresh.md`
Expected: file exists and only intended content is present.

- [ ] **Step 4: Commit the plan file**

Run:
```bash
git add docs/superpowers/plans/2026-04-12-documentation-refresh.md
git commit -m "Add documentation refresh implementation plan"
```
Expected: plan commit created successfully.

## Chunk 2: Rewrite the landing page and docs index

### Task 2: Refresh `README.md` to reflect current product state

**Files:**
- Modify: `README.md`
- Read: `docs/superpowers/specs/2026-04-12-documentation-refresh-design.md`
- Read: `docs/resources/workflow.md`
- Read: `docs/data-sources/node_schema.md`
- Read: `docs/data-sources/inventory.md`
- Read: `docs/data-sources/prompt_to_terraform.md`
- Read: `docs/data-sources/workspace_to_terraform.md`

- [ ] **Step 1: Write a failing editorial checklist for `README.md`**

In a temporary checklist or TODO comment in the plan file, enumerate the required updates:
- equal audience positioning
- current synthesis surfaces
- current validation surfaces
- current plan-time inventory validation story
- removal of stale legacy execution-field references
- links to new guides

- [ ] **Step 2: Rewrite `README.md` to match the approved structure**

Update `README.md` so it contains:
- concise product positioning
- current quick start
- current core concepts
- explicit scope statement for pinned built-in ComfyUI support
- current validation and generation story
- links to deeper docs by audience

- [ ] **Step 3: Verify `README.md` for stale workflow legacy-field references**

Run: `rg -n "comfyui_workflow\.(status|outputs|error)|\bstatus\b.*comfyui_workflow|\boutputs\b.*comfyui_workflow|\berror\b.*comfyui_workflow" README.md`
Expected: no matches.

- [ ] **Step 4: Commit the README rewrite**

Run:
```bash
git add README.md
git commit -m "Refresh README for current provider state"
```
Expected: commit created successfully.

### Task 3: Replace `docs/index.md` with a lightweight docs map

**Files:**
- Modify: `docs/index.md`
- Read: `README.md`
- Read: `docs/getting-started.md`
- Read: `docs/workflow-authoring.md`
- Read: `docs/ai-harness-guide.md`
- Read: `docs/contributing.md`

- [ ] **Step 1: Write the new routing structure in `docs/index.md`**

Replace the old overview content with a lightweight docs map that routes readers by audience and links to the new narrative guides plus generated references.

- [ ] **Step 2: Verify `docs/index.md` does not duplicate the README narrative**

Run: `sed -n '1,220p' docs/index.md`
Expected: concise navigation-oriented content rather than a second full product overview.

- [ ] **Step 3: Commit the docs index rewrite**

Run:
```bash
git add docs/index.md
git commit -m "Turn docs index into audience map"
```
Expected: commit created successfully.

## Chunk 3: Add user and AI narrative guides

### Task 4: Create `docs/getting-started.md`

**Files:**
- Create: `docs/getting-started.md`
- Read: `README.md`
- Read: `docs/resources/workflow.md`
- Read: `docs/data-sources/inventory.md`

- [ ] **Step 1: Draft the getting-started guide**

Write `docs/getting-started.md` covering installation, provider config, minimal runnable workflow, plan/apply behavior, and how strict plan-time inventory validation affects authoring.

- [ ] **Step 2: Verify examples align with current resource/data-source surfaces**

Run: `rg -n "prompt_id|workflow_id|outputs_json|execution_status_json|inventory" docs/getting-started.md`
Expected: guide references current surfaces, not removed legacy fields.

- [ ] **Step 3: Commit the getting-started guide**

Run:
```bash
git add docs/getting-started.md
git commit -m "Add getting started guide"
```
Expected: commit created successfully.

### Task 5: Create `docs/workflow-authoring.md`

**Files:**
- Create: `docs/workflow-authoring.md`
- Read: `docs/resources/workflow.md`
- Read: `docs/resources/workspace.md`
- Read: `docs/data-sources/prompt_json.md`
- Read: `docs/data-sources/workspace_json.md`
- Read: `docs/data-sources/prompt_to_workspace.md`
- Read: `docs/data-sources/workspace_to_prompt.md`

- [ ] **Step 1: Draft the workflow authoring guide**

Write `docs/workflow-authoring.md` covering node-per-resource authoring, `comfyui_workflow`, `comfyui_workspace`, artifact import/translation, and recommended authoring patterns.

- [ ] **Step 2: Verify the guide avoids stale execution-field terminology**

Run: `rg -n "comfyui_workflow\.(status|outputs|error)|legacy" docs/workflow-authoring.md`
Expected: no stale field references unless explicitly describing removed legacy behavior historically.

- [ ] **Step 3: Commit the workflow authoring guide**

Run:
```bash
git add docs/workflow-authoring.md
git commit -m "Add workflow authoring guide"
```
Expected: commit created successfully.

### Task 6: Create `docs/ai-harness-guide.md`

**Files:**
- Create: `docs/ai-harness-guide.md`
- Read: `docs/data-sources/node_schema.md`
- Read: `docs/data-sources/inventory.md`
- Read: `docs/data-sources/prompt_to_terraform.md`
- Read: `docs/data-sources/workspace_to_terraform.md`
- Read: `docs/data-sources/prompt_validation.md`
- Read: `docs/data-sources/workspace_validation.md`

- [ ] **Step 1: Draft the AI harness guide**

Write `docs/ai-harness-guide.md` covering generated node contracts, validation modes, synthesis, inventory-aware planning, and the recommended AI harness loop.

- [ ] **Step 2: Verify the guide names the current AI-facing contracts explicitly**

Run: `rg -n "comfyui_node_schema|comfyui_inventory|comfyui_prompt_to_terraform|comfyui_workspace_to_terraform|executable" docs/ai-harness-guide.md`
Expected: all current AI-facing surfaces are explicitly documented.

- [ ] **Step 3: Commit the AI harness guide**

Run:
```bash
git add docs/ai-harness-guide.md
git commit -m "Add AI harness authoring guide"
```
Expected: commit created successfully.

## Chunk 4: Add contributor and architecture guides

### Task 7: Refresh `docs/ai-maintainability.md` and `docs/release-validation.md`

**Files:**
- Modify: `docs/ai-maintainability.md`
- Modify: `docs/release-validation.md`
- Read: `README.md`
- Read: `docs/generation-architecture.md`
- Read: `docs/known-boundaries.md`

- [ ] **Step 1: Rewrite `docs/ai-maintainability.md` as a narrower architecture note**

Update it to focus on generated vs hand-rolled boundaries, regeneration, pinned-version support, and maintainability rationale.

- [ ] **Step 2: Rewrite `docs/release-validation.md` as the release-confidence guide**

Update it to cover all current validation lanes, prerequisites, artifact locations, and failure interpretation.

- [ ] **Step 3: Verify both docs mention the current validation and generation surfaces**

Run: `rg -n "make generate|make synthesis-e2e|make inventory-plan-e2e|make execution-e2e|make workspace-e2e|make release-e2e" docs/ai-maintainability.md docs/release-validation.md`
Expected: release-validation doc covers the full current matrix; maintainability doc mentions relevant generation/verification concepts.

- [ ] **Step 4: Commit the narrative doc refresh**

Run:
```bash
git add docs/ai-maintainability.md docs/release-validation.md
git commit -m "Refresh maintainability and release validation docs"
```
Expected: commit created successfully.

### Task 8: Create contributor, generation, and boundary guides

**Files:**
- Create: `docs/contributing.md`
- Create: `docs/generation-architecture.md`
- Create: `docs/known-boundaries.md`
- Read: `README.md`
- Read: `docs/ai-maintainability.md`
- Read: `docs/release-validation.md`
- Read: `GNUmakefile`
- Read: `CLAUDE.md`

- [ ] **Step 1: Draft `docs/contributing.md`**

Cover local setup, generation prerequisites, docs expectations, validation commands, and generated vs hand-rolled boundaries.

- [ ] **Step 2: Draft `docs/generation-architecture.md`**

Describe extraction from pinned ComfyUI server metadata, frontend UI hints, resource/schema generation, synthesis architecture, and dynamic inventory validation generation.

- [ ] **Step 3: Draft `docs/known-boundaries.md`**

State the scoped support promise, current guarantees, and explicit non-goals.

- [ ] **Step 4: Verify cross-guide consistency**

Run: `rg -n "pinned built-in|dynamic inventory|unsupported dynamic expressions|AI" docs/contributing.md docs/generation-architecture.md docs/known-boundaries.md`
Expected: the three docs use consistent scope and terminology.

- [ ] **Step 5: Commit the contributor and architecture guides**

Run:
```bash
git add docs/contributing.md docs/generation-architecture.md docs/known-boundaries.md
git commit -m "Add contributor and architecture documentation"
```
Expected: commit created successfully.

## Chunk 5: Regenerate docs and verify the full narrative surface

### Task 9: Regenerate and verify documentation consistency

**Files:**
- Modify: generated docs if `make docs` updates them
- Read: all new and updated narrative docs

- [ ] **Step 1: Run docs generation**

Run: `make docs`
Expected: generated docs refresh successfully.

- [ ] **Step 2: Run stale-reference checks**

Run: `rg -n "comfyui_workflow\.(status|outputs|error)|\bstatus\b.*comfyui_workflow|\boutputs\b.*comfyui_workflow|\berror\b.*comfyui_workflow" README.md docs examples`
Expected: no stale narrative references to removed workflow legacy fields.

- [ ] **Step 3: Run link and topic sanity checks**

Run:
```bash
rg -n "getting-started|workflow-authoring|ai-harness-guide|contributing|generation-architecture|known-boundaries" README.md docs/index.md docs/*.md
```
Expected: new guides are linked from the landing docs and referenced consistently.

- [ ] **Step 4: Review the final diff for coherence**

Run: `git diff --stat HEAD~1..HEAD || git diff --stat`
Expected: only intended documentation changes remain.

- [ ] **Step 5: Commit regenerated docs and final doc alignment changes**

Run:
```bash
git add README.md docs
git commit -m "Refresh provider documentation for current state"
```
Expected: final documentation refresh commit created successfully.

### Task 10: Final validation before handoff

**Files:**
- Read: `README.md`
- Read: `docs/index.md`
- Read: all created/modified docs

- [ ] **Step 1: Run the minimal documentation verification set**

Run:
```bash
make docs
rg -n "comfyui_workflow\.(status|outputs|error)|\bstatus\b.*comfyui_workflow|\boutputs\b.*comfyui_workflow|\berror\b.*comfyui_workflow" README.md docs examples
```
Expected: `make docs` succeeds and the grep returns no stale matches.

- [ ] **Step 2: Summarize the documentation changes and risks in the plan file**

Add a final short section to `docs/superpowers/plans/2026-04-12-documentation-refresh.md` noting which docs were added or rewritten and whether any generated docs changed.

- [ ] **Step 3: Commit the final plan-note update if needed**

Run:
```bash
git add docs/superpowers/plans/2026-04-12-documentation-refresh.md
git commit -m "Record documentation refresh completion notes"
```
Expected: only if the plan file changed in this step.

## Completion Notes

- Rewrote `README.md` around the current provider state, audience split, synthesis surfaces, and strict plan-time inventory validation.
- Replaced the generated docs index template in `templates/index.md.tmpl` so `make docs` now emits an audience-oriented `docs/index.md`.
- Added new narrative guides:
  - `docs/getting-started.md`
  - `docs/workflow-authoring.md`
  - `docs/ai-harness-guide.md`
  - `docs/contributing.md`
  - `docs/generation-architecture.md`
  - `docs/known-boundaries.md`
- Refreshed `docs/ai-maintainability.md` and `docs/release-validation.md` to match the current generated-first architecture and verification matrix.
- `make docs` completed successfully and did not introduce unexpected generated reference diffs beyond the intended `docs/index.md` update from the template change.
