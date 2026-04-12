# Release Validation Harness Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a release-confidence validation harness that exercises the provider's hand-rolled workflow, translation, workspace, execution-state, and artifact logic against a real ComfyUI runtime and browser UI.

**Architecture:** Extend the existing `validation/execution_e2e` and `validation/workspace_e2e` lanes into a scenario-driven release harness. Keep generated node wrappers as scenario inputs, while asserting provider-owned invariants through Terraform outputs, machine-readable artifacts, and Playwright metrics against the real ComfyUI canvas.

**Tech Stack:** Go 1.25, Terraform Plugin Framework, Terraform CLI, existing ComfyUI runtime scripts, Node.js, Playwright, Python 3, shell scripts, JSON artifact fixtures.

---

## Chunk 1: Inventory And Repair Existing Validation Lanes

### Task 1: Fix stale execution fixture contract

**Files:**
- Modify: `validation/execution_e2e/main.tf`
- Modify: `scripts/execution-e2e/run.sh`
- Test: `make execution-e2e`

- [ ] **Step 1: Remove the deleted legacy workflow fields from the execution fixture**

Update `validation/execution_e2e/main.tf` so `workflow_execution` outputs use only the rich execution surface:
- keep `prompt_id`
- remove `status`
- keep `workflow_id`
- keep `outputs_count`
- keep `outputs_json`
- keep `outputs_structured`
- keep `preview_output_json`
- keep `execution_status_json`
- add `execution_error_json` if useful for negative debugging

- [ ] **Step 2: Update the execution harness assertions to match the current resource contract**

Modify `scripts/execution-e2e/run.sh` so the Python assertions validate:
- `workflow["workflow_id"]` is populated
- `workflow["outputs_count"] >= 1`
- `workflow["execution_status_json"]` indicates completion instead of reading removed `workflow["status"]`
- `job["status"] == "completed"`
- downloaded artifact exists and is non-empty

- [ ] **Step 3: Run the execution harness**

Run: `make execution-e2e`
Expected:
- Terraform apply succeeds
- the script prints `Execution e2e validation passed`

- [ ] **Step 4: Commit**

```bash
git add validation/execution_e2e/main.tf scripts/execution-e2e/run.sh
git commit -m "Repair execution e2e contract checks"
```

## Chunk 2: Build Canonical Release Scenarios

### Task 2: Add release workflow scenarios that stress hand-rolled provider logic

**Files:**
- Create: `validation/release_e2e/providers.tf`
- Create: `validation/release_e2e/fixtures.tf`
- Create: `validation/release_e2e/workflows.tf`
- Create: `validation/release_e2e/outputs.tf`
- Create: `validation/release_e2e/browser/package.json`
- Create: `validation/release_e2e/browser/playwright.config.ts`
- Create: `validation/release_e2e/browser/tests/release_workflows.spec.ts`
- Create: `validation/release_e2e/browser/tests/helpers/comfyui.ts`
- Create: `validation/release_e2e/browser/tests/helpers/graph_metrics.ts`
- Create: `scripts/release-e2e/run.sh`
- Test: `./scripts/release-e2e/run.sh`

- [ ] **Step 1: Define scenario metadata and provider wiring**

Create `validation/release_e2e/providers.tf` and `fixtures.tf` that:
- configure the local dev override for the provider
- accept host/port runtime variables
- define scenario metadata locals for expected node counts, link counts, group counts, and workspace names

- [ ] **Step 2: Author a small set of canonical scenarios**

Implement `validation/release_e2e/workflows.tf` with scenarios that cover:
- assembly-heavy workflow built from Terraform resources
- raw `workflow_json` import scenario
- dense workspace builder scenario
- translation round-trip scenario
- artifact path scenario

Each scenario should expose enough metadata for later assertions:
- prompt/workflow IDs where applicable
- expected node count
- expected link count
- expected group count
- staged workspace name
- generated JSON artifact path

- [ ] **Step 3: Export scenario assertions through Terraform outputs**

Create `validation/release_e2e/outputs.tf` that returns one structured object per scenario with:
- identifiers
- generated/staged artifact paths
- expected UI counts
- round-trip metadata
- any file paths needed by the browser harness

- [ ] **Step 4: Build a dedicated Playwright harness for release workflows**

Create `validation/release_e2e/browser` files that:
- read the runtime base URL
- load each scenario workspace/workflow in the real ComfyUI UI
- verify expected node/group/link counts
- detect hidden/clipped nodes
- verify representative cross-group and fan-in/fan-out links are visible
- write screenshots and metrics JSON under `validation/release_e2e/artifacts/browser/`

- [ ] **Step 5: Run the release workflow harness**

Run: `./scripts/release-e2e/run.sh`
Expected:
- Terraform apply succeeds
- Playwright tests pass in Chromium
- artifacts are written under `validation/release_e2e/artifacts/`

- [ ] **Step 6: Commit**

```bash
git add validation/release_e2e scripts/release-e2e
git commit -m "Add release workflow browser validation harness"
```

## Chunk 3: Expand Workspace And Translation Coverage

### Task 3: Extend workspace/browser validation to certify provider-owned layout and translation logic

**Files:**
- Modify: `validation/workspace_e2e/fixtures.tf`
- Modify: `validation/workspace_e2e/workspaces.tf`
- Modify: `validation/workspace_e2e/outputs.tf`
- Modify: `validation/workspace_e2e/browser/tests/workspace_layout.spec.ts`
- Create: `validation/workspace_e2e/browser/tests/helpers/connection_metrics.ts`
- Test: `make workspace-e2e`

- [ ] **Step 1: Add higher-complexity workspace fixtures**

Extend `validation/workspace_e2e/workspaces.tf` with additional scenarios that stress:
- dense groups
- cross-group links
- fan-in and fan-out structures
- staged subgraphs
- mixed layout overrides
- sparse slot ordering

- [ ] **Step 2: Export stronger expected metrics from Terraform**

Update `validation/workspace_e2e/outputs.tf` so browser tests can read:
- expected node counts
- expected group counts
- expected link counts
- workspace names
- any scenario-specific assertions such as group titles or style overrides

- [ ] **Step 3: Teach Playwright to validate connectivity, not just visibility**

Update `validation/workspace_e2e/browser/tests/workspace_layout.spec.ts` and add `connection_metrics.ts` so the browser checks:
- expected links are present
- no broken links render
- no backward links render where the provider should preserve left-to-right flow
- representative fan-in/fan-out links remain connected after load

- [ ] **Step 4: Run the workspace harness**

Run: `make workspace-e2e`
Expected:
- all browser assertions pass
- new screenshots and metrics appear under `validation/workspace_e2e/artifacts/browser/`

- [ ] **Step 5: Commit**

```bash
git add validation/workspace_e2e
git commit -m "Expand workspace release validation coverage"
```

## Chunk 4: Add Round-Trip And Artifact Regression Checks

### Task 4: Certify translation and artifact behavior with machine-readable golden outputs

**Files:**
- Create: `validation/release_e2e/translation.tf`
- Create: `validation/release_e2e/artifact_checks.tf`
- Modify: `scripts/release-e2e/run.sh`
- Test: `./scripts/release-e2e/run.sh`

- [ ] **Step 1: Add translation round-trip fixtures**

Create `validation/release_e2e/translation.tf` to exercise:
- prompt JSON -> workspace JSON
- workspace JSON -> prompt JSON
- round-trip preservation of sparse inputs
- metadata-heavy nodes and reroutes

Expose resulting JSON blobs and selected invariants via Terraform outputs.

- [ ] **Step 2: Add artifact-path checks to the release harness**

Create `validation/release_e2e/artifact_checks.tf` so the release scenario includes:
- uploaded input resources
- output discovery
- output artifact downloads
- metadata assertions such as local path, checksum, and content length

- [ ] **Step 3: Write golden artifacts and compare invariants**

Update `scripts/release-e2e/run.sh` so it:
- captures `terraform output -json`
- writes generated prompt/workspace JSON to `validation/release_e2e/artifacts/generated/`
- validates round-trip invariants with Python or shell checks
- fails loudly on missing IDs, missing files, or count mismatches

- [ ] **Step 4: Run the release harness again**

Run: `./scripts/release-e2e/run.sh`
Expected:
- translation and artifact assertions pass
- JSON artifacts are written deterministically

- [ ] **Step 5: Commit**

```bash
git add validation/release_e2e scripts/release-e2e/run.sh
git commit -m "Add round-trip and artifact release checks"
```

## Chunk 5: Wire Release Verification Into The Repo Entry Points

### Task 5: Add a stable top-level release validation entrypoint and documentation

**Files:**
- Modify: `GNUmakefile`
- Modify: `README.md`
- Create: `docs/release-validation.md`
- Test: `make release-e2e`

- [ ] **Step 1: Add a top-level release validation make target**

Modify `GNUmakefile` to add a target such as:
- `release-e2e`

That target should run:
- the release harness script
- any dependent browser install step if required by the local setup

- [ ] **Step 2: Document what is hand-rolled versus generated**

Add a short section in `README.md` or `docs/release-validation.md` that explains:
- generated node wrappers come from upstream ComfyUI specs
- the release harness is focused on the hand-rolled provider logic
- the current approximate maintenance split is `~19.7k` hand-rolled lines vs `~99.4k` generated lines

- [ ] **Step 3: Document how to run the release suite locally**

In `docs/release-validation.md`, document:
- runtime prerequisites
- how the ComfyUI runtime is started
- where artifacts are written
- which provider-owned layers each scenario certifies

- [ ] **Step 4: Run the final top-level command**

Run: `make release-e2e`
Expected:
- the release harness completes successfully
- generated artifacts and screenshots are present

- [ ] **Step 5: Commit**

```bash
git add GNUmakefile README.md docs/release-validation.md
git commit -m "Add release validation entrypoint and docs"
```

## Chunk 6: Final Verification And Release Readiness

### Task 6: Run the full verification stack and record the evidence

**Files:**
- Modify: `docs/release-validation.md`
- Test: `go test ./... -timeout 120s`
- Test: `make lint test vet`
- Test: `make execution-e2e`
- Test: `make workspace-e2e`
- Test: `make release-e2e`

- [ ] **Step 1: Run unit and repo verification**

Run: `go test ./... -timeout 120s`
Expected: all Go tests pass

Run: `make lint test vet`
Expected: lint, tests, codegen verification, and vet all pass

- [ ] **Step 2: Run runtime/browser verification lanes**

Run: `make execution-e2e`
Expected: execution harness passes

Run: `make workspace-e2e`
Expected: workspace Playwright harness passes

Run: `make release-e2e`
Expected: release workflow harness passes

- [ ] **Step 3: Record evidence locations in documentation**

Update `docs/release-validation.md` with:
- artifact directories
- expected screenshots/metrics files
- how to inspect failures by scenario

- [ ] **Step 4: Commit**

```bash
git add docs/release-validation.md
git commit -m "Record release validation evidence workflow"
```
