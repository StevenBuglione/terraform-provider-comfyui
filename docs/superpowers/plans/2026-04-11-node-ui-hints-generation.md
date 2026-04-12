# Node UI Hints Generation Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `make generate` extract real ComfyUI frontend node sizing hints with Playwright and use generated metadata in the workspace builder instead of handwritten magic numbers.

**Architecture:** Extend the existing extraction pipeline with a runtime UI-hints stage that boots a temporary ComfyUI frontend, introspects live nodes, writes `node_ui_hints.json`, and has `cmd/generate` emit a resources-package metadata file consumed by workspace layout code. Keep conservative runtime fallback behavior only for nodes without extracted hints.

**Tech Stack:** Go, Python, Bash, Playwright, ComfyUI frontend runtime, GNU Make

---

## Chunk 1: Spec And Extraction Entry Point

### Task 1: Add the new spec and plan documents

**Files:**
- Create: `docs/superpowers/specs/2026-04-11-node-ui-hints-generation-design.md`
- Create: `docs/superpowers/plans/2026-04-11-node-ui-hints-generation.md`

- [ ] **Step 1: Confirm the new spec and plan files exist**

Run: `test -f docs/superpowers/specs/2026-04-11-node-ui-hints-generation-design.md && test -f docs/superpowers/plans/2026-04-11-node-ui-hints-generation.md`
Expected: command exits `0`

- [ ] **Step 2: Commit the planning docs**

```bash
git add docs/superpowers/specs/2026-04-11-node-ui-hints-generation-design.md docs/superpowers/plans/2026-04-11-node-ui-hints-generation.md
git commit -m "Add node UI hints generation design"
```

### Task 2: Add a strict UI-hints extraction pipeline

**Files:**
- Create: `scripts/extract/extract_ui_hints.ts` or `scripts/extract/extract_ui_hints.mjs`
- Create: `scripts/extract/run_ui_hints.sh`
- Create: `scripts/extract/node_ui_hints.json`
- Modify: `GNUmakefile`
- Modify: `scripts/hooks/generate_if_needed.sh`
- Modify: `scripts/extract/test_extractors.py`

- [ ] **Step 1: Write failing extractor structure tests**

Add tests that assert:
- `node_ui_hints.json` exists after extraction
- the artifact has top-level metadata fields
- a canonical node like `CLIPTextEncode` has non-zero min sizes

- [ ] **Step 2: Run the failing extractor tests**

Run: `python3 scripts/extract/test_extractors.py`
Expected: FAIL because `node_ui_hints.json` and extraction support do not exist yet

- [ ] **Step 3: Implement the extractor and Makefile entrypoint**

Implement:
- temporary runtime startup via existing `scripts/workspace-e2e/start-comfyui.sh`
- automated Playwright-based live-node introspection
- strict failure behavior in `make generate`

- [ ] **Step 4: Run extractor tests**

Run: `python3 scripts/extract/test_extractors.py`
Expected: PASS

- [ ] **Step 5: Commit extraction pipeline changes**

```bash
git add GNUmakefile scripts/extract scripts/hooks/generate_if_needed.sh
git commit -m "Extract ComfyUI node UI hints during generation"
```

## Chunk 2: Generator Metadata

### Task 3: Teach `cmd/generate` to consume UI hints

**Files:**
- Modify: `cmd/generate/main.go`
- Modify: `cmd/generate/types.go`
- Modify: `cmd/generate/generate_test.go`
- Create: `internal/resources/node_ui_hints_generated.go`

- [ ] **Step 1: Write failing generator tests**

Add tests that assert:
- `cmd/generate` loads `node_ui_hints.json`
- generated metadata file contains canonical node sizing hints
- malformed UI-hints artifacts fail clearly

- [ ] **Step 2: Run generator tests to verify failure**

Run: `go test ./cmd/generate -timeout 120s`
Expected: FAIL because UI-hints support is not implemented yet

- [ ] **Step 3: Implement generator support**

Implement:
- UI-hints JSON types
- parsing and validation
- generated resources-package metadata emission

- [ ] **Step 4: Run generator tests**

Run: `go test ./cmd/generate -timeout 120s`
Expected: PASS

- [ ] **Step 5: Run full generation**

Run: `make generate`
Expected: PASS and update committed generated artifacts deterministically

- [ ] **Step 6: Commit generator changes**

```bash
git add cmd/generate internal/resources/node_ui_hints_generated.go scripts/extract/node_ui_hints.json
git commit -m "Generate node UI hint metadata"
```

## Chunk 3: Workspace Builder Refactor

### Task 4: Replace handwritten sizing rules with generated hints

**Files:**
- Modify: `internal/resources/workspace_builder.go`
- Modify: `internal/resources/workspace_builder_test.go`

- [ ] **Step 1: Write failing workspace sizing tests**

Add tests that assert:
- extracted UI hints drive min node width and height
- missing hints fall back to generic estimation
- multiline sizing is sourced from generated hints, not hardcoded constants

- [ ] **Step 2: Run the focused workspace tests**

Run: `go test ./internal/resources -run 'TestBuildWorkspaceNodeHonors|TestWorkspaceBuilder' -timeout 120s`
Expected: FAIL because the builder still embeds handwritten UI sizing logic

- [ ] **Step 3: Implement builder refactor**

Refactor the builder so it:
- consults generated node UI hints first
- applies widget-level hint overrides
- falls back only when hints are absent

- [ ] **Step 4: Run focused workspace tests**

Run: `go test ./internal/resources -run 'TestBuildWorkspaceNodeHonors|TestWorkspaceBuilder' -timeout 120s`
Expected: PASS

- [ ] **Step 5: Commit workspace builder refactor**

```bash
git add internal/resources/workspace_builder.go internal/resources/workspace_builder_test.go
git commit -m "Use generated UI hints for workspace sizing"
```

## Chunk 4: End-To-End Verification

### Task 5: Verify generation and browser regression coverage

**Files:**
- Modify: `validation/release_e2e/browser/tests/release_workflows.spec.ts`
- Modify: any supporting helper only if needed

- [ ] **Step 1: Add or preserve the browser regression assertion**

Ensure `release_gallery` still enforces minimum readable spacing after the refactor.

- [ ] **Step 2: Run targeted verification**

Run: `make release-e2e`
Expected: PASS

- [ ] **Step 3: Run full repo verification**

Run: `go test ./... -timeout 120s`
Expected: PASS

Run: `make lint test vet`
Expected: PASS

- [ ] **Step 4: Review generated diffs**

Run: `git diff -- scripts/extract/node_ui_hints.json internal/resources/node_ui_hints_generated.go`
Expected: deterministic, reviewable artifact updates only

- [ ] **Step 5: Final commit**

```bash
git add .
git commit -m "Automate ComfyUI node UI hint generation"
```
