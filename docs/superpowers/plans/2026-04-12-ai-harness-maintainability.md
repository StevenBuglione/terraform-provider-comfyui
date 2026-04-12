# AI Harness Maintainability Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the provider generated-first and machine-readable enough that an AI harness can reliably create, inspect, refactor, validate, translate, and maintain ComfyUI workflows declaratively in Terraform with minimal inference.

**Architecture:** Extend the current generated extraction pipeline into a three-IR system: Prompt IR, Workspace IR, and Terraform IR. Generate as much schema and UI behavior as possible from real ComfyUI server/frontend behavior, keep provider-owned logic narrow around Terraform orchestration and IR translation, and prove the full loop with stronger synthesis, validation, execution, and browser harnesses.

**Tech Stack:** Go, Terraform Plugin Framework, Playwright, Python extraction scripts, Node-based frontend extraction, GNU Make, GitHub Actions.

---

## File Structure

### Existing files to extend

- Modify: `scripts/extract/merge_specs.py`
  - extend generated server-side schema extraction for normalized node contracts
- Modify: `scripts/extract/node_specs.json`
  - expand generated node schema contract payload
- Modify: `scripts/extract/extract_ui_hints.mjs`
  - enrich frontend extraction with any remaining widget/UI contract fields needed by Terraform synthesis and workspace reasoning
- Modify: `scripts/extract/node_ui_hints.json`
  - persist enriched frontend-derived contract
- Modify: `cmd/generate/main.go`
  - load expanded generated artifacts and emit new generated metadata files
- Modify: `cmd/generate/types.go`
  - add template/input types for node schema contract and Terraform IR support
- Modify: `cmd/generate/templates.go`
  - emit new generated metadata structures
- Modify: `internal/provider/provider.go`
  - register new data sources and keep AI-facing surfaces discoverable
- Modify: `internal/datasources/node_info.go`
  - either deprecate in favor of or adapt alongside a richer structured schema surface
- Modify: `internal/datasources/prompt_validation.go`
  - add validation mode support
- Modify: `internal/datasources/workspace_validation.go`
  - add validation mode support
- Modify: `internal/artifacts/prompt.go`
  - support Prompt IR normalization/provenance expansion if needed
- Modify: `internal/artifacts/workspace.go`
  - support Workspace IR normalization/provenance expansion if needed
- Modify: `internal/artifacts/translation.go`
  - host shared translation/reporting primitives and possibly Terraform IR translation support
- Modify: `README.md`
  - document new AI-maintainability surfaces and validation defaults
- Modify: `.github/workflows/test.yml`
  - add any new deterministic verification steps and tests for synthesis surfaces

### New files to create

- Create: `internal/datasources/node_schema.go`
  - structured, generated-first node schema contract for AI harnesses
- Create: `internal/datasources/node_schema_test.go`
  - focused tests for structured node schema behavior
- Create: `internal/artifacts/terraform_ir.go`
  - canonical Terraform IR types and normalization helpers
- Create: `internal/artifacts/terraform_ir_test.go`
  - tests for Terraform IR normalization and deterministic behavior
- Create: `internal/datasources/prompt_to_terraform.go`
  - prompt artifact to Terraform IR/HCL synthesis surface
- Create: `internal/datasources/prompt_to_terraform_test.go`
  - tests for prompt synthesis behavior and fidelity reporting
- Create: `internal/datasources/workspace_to_terraform.go`
  - workspace artifact to Terraform IR/HCL synthesis surface
- Create: `internal/datasources/workspace_to_terraform_test.go`
  - tests for workspace synthesis behavior and fidelity reporting
- Create: `internal/artifacts/terraform_render.go`
  - deterministic HCL rendering from Terraform IR
- Create: `internal/artifacts/terraform_render_test.go`
  - golden tests for HCL rendering
- Create: `internal/testdata/terraform_synthesis/`
  - corpus of prompt/workspace fixtures and expected Terraform IR/HCL outputs
- Create: `validation/synthesis_e2e/`
  - end-to-end Terraform synthesis and re-assembly harness
- Create: `validation/synthesis_e2e/main.tf`
  - canonical synthesized workflow fixtures rendered back through provider assembly
- Create: `validation/synthesis_e2e/outputs.tf`
  - machine-readable assertions for synthesis round-trip
- Create: `scripts/synthesis-e2e/run.sh`
  - harness runner
- Create: `docs/ai-maintainability.md`
  - focused user-facing guidance for AI harness workflows

### Generated files expected from implementation

- Create or modify: `internal/resources/node_schema_generated.go`
  - generated server-side node contract
- Modify or expand: `internal/resources/node_ui_hints_generated.go`
  - frontend-derived UI contract
- Optionally create: `internal/resources/terraform_ir_hints_generated.go`
  - any generated rendering hints needed for deterministic Terraform synthesis

---

## Chunk 1: Generated Node Contract

### Task 1: Expand generated node contract beyond raw JSON strings

**Files:**
- Modify: `scripts/extract/merge_specs.py`
- Modify: `scripts/extract/node_specs.json`
- Modify: `cmd/generate/main.go`
- Modify: `cmd/generate/types.go`
- Modify: `cmd/generate/templates.go`
- Create: `internal/resources/node_schema_generated.go`
- Create: `internal/datasources/node_schema.go`
- Create: `internal/datasources/node_schema_test.go`
- Modify: `internal/provider/provider.go`

- [ ] **Step 1: Write the failing node schema data source tests**

Add tests that prove a new structured data source can return:
- required inputs as structured attributes
- optional inputs as structured attributes
- defaults/ranges/enums/widget metadata
- outputs with names and slot types

Use fixtures derived from a stable node such as `CLIPTextEncode` and `KSampler`.

- [ ] **Step 2: Run focused tests to verify failure**

Run:

```bash
go test ./internal/datasources -run 'TestNodeSchema' -v
```

Expected:
- FAIL because `comfyui_node_schema` does not exist yet

- [ ] **Step 3: Expand extraction schema in `merge_specs.py`**

Update extraction output so the generated node-spec artifact includes normalized fields for:
- required inputs
- optional inputs
- input defaults
- numeric bounds and step
- enum/combo values
- output slot names and types
- any server-derived flags that are currently only implicit

- [ ] **Step 4: Update generator input/output types**

Extend generator types and templates so generated node schema metadata is emitted into a Go file under `internal/resources/`.

- [ ] **Step 5: Implement `comfyui_node_schema`**

Add a new data source that reads from generated schema metadata and returns structured Terraform attributes instead of JSON string blobs.

- [ ] **Step 6: Register the new data source**

Update `internal/provider/provider.go` so the new surface is publicly available.

- [ ] **Step 7: Run focused tests to verify pass**

Run:

```bash
go test ./internal/datasources -run 'TestNodeSchema' -v
```

Expected:
- PASS

- [ ] **Step 8: Run generator verification**

Run:

```bash
make generate
bash scripts/hooks/verify_generated_clean.sh
```

Expected:
- generated node contract files are deterministic and clean

- [ ] **Step 9: Commit**

```bash
git add scripts/extract/merge_specs.py scripts/extract/node_specs.json cmd/generate/main.go cmd/generate/types.go cmd/generate/templates.go internal/resources/node_schema_generated.go internal/datasources/node_schema.go internal/datasources/node_schema_test.go internal/provider/provider.go
git commit -m "Add generated structured node schema contract"
```

---

## Chunk 2: Validation Modes

### Task 2: Distinguish fragment validation from executable validation

**Files:**
- Modify: `internal/datasources/prompt_validation.go`
- Modify: `internal/datasources/workspace_validation.go`
- Modify: `internal/datasources/validation_semantic_test.go`
- Modify: `internal/validation/semantic.go`
- Modify: `internal/validation/semantic_test.go`
- Modify: `README.md`

- [ ] **Step 1: Write failing validation-mode tests**

Add tests covering:
- fragment mode accepts graphs without output nodes
- executable workflow mode rejects prompts without output nodes
- executable workspace mode rejects workspace translations without output nodes
- default behavior matches executable validation

- [ ] **Step 2: Run focused tests to verify failure**

Run:

```bash
go test ./internal/datasources ./internal/validation -run 'Test.*ValidationMode|TestValidatePrompt' -v
```

Expected:
- FAIL because mode-aware validation is not implemented yet

- [ ] **Step 3: Add mode attributes and validation branching**

Implement explicit modes:
- `fragment`
- `workspace_fragment`
- `executable_workflow`
- `executable_workspace`

Default prompt validation to `executable_workflow`.
Default workspace validation to `executable_workspace`.

- [ ] **Step 4: Ensure diagnostics remain machine-readable**

Keep current summary/error surfaces, but make sure mode selection is reflected in descriptions and tests.

- [ ] **Step 5: Run focused tests to verify pass**

Run:

```bash
go test ./internal/datasources ./internal/validation -run 'Test.*ValidationMode|TestValidatePrompt' -v
```

Expected:
- PASS

- [ ] **Step 6: Update docs**

Document the new validation defaults and when fragment modes should be used by AI harnesses.

- [ ] **Step 7: Commit**

```bash
git add internal/datasources/prompt_validation.go internal/datasources/workspace_validation.go internal/datasources/validation_semantic_test.go internal/validation/semantic.go internal/validation/semantic_test.go README.md
git commit -m "Add executable and fragment validation modes"
```

---

## Chunk 3: Canonical Terraform IR

### Task 3: Introduce a provider-owned Terraform IR

**Files:**
- Create: `internal/artifacts/terraform_ir.go`
- Create: `internal/artifacts/terraform_ir_test.go`
- Modify: `internal/artifacts/translation.go`
- Modify: `internal/artifacts/prompt.go`
- Modify: `internal/artifacts/workspace.go`

- [ ] **Step 1: Write failing IR normalization tests**

Add tests that define the expected IR shape for:
- a simple assembled prompt
- a workspace with groups
- a graph with fan-in/fan-out
- sparse-slot inputs

The tests should prove:
- deterministic resource naming
- preserved provenance
- stable connection expression modeling

- [ ] **Step 2: Run focused tests to verify failure**

Run:

```bash
go test ./internal/artifacts -run 'TestTerraformIR' -v
```

Expected:
- FAIL because Terraform IR types do not exist yet

- [ ] **Step 3: Implement canonical Terraform IR types**

Create explicit Go types for:
- resources
- input expressions
- connection expressions
- workflow/meta resources
- provenance/fidelity annotations

- [ ] **Step 4: Add normalization helpers**

Translate normalized Prompt IR and Workspace IR into Terraform IR without rendering HCL yet.

- [ ] **Step 5: Run focused tests to verify pass**

Run:

```bash
go test ./internal/artifacts -run 'TestTerraformIR' -v
```

Expected:
- PASS

- [ ] **Step 6: Commit**

```bash
git add internal/artifacts/terraform_ir.go internal/artifacts/terraform_ir_test.go internal/artifacts/translation.go internal/artifacts/prompt.go internal/artifacts/workspace.go
git commit -m "Add canonical Terraform IR for workflow synthesis"
```

---

## Chunk 4: Terraform Rendering

### Task 4: Render deterministic HCL from Terraform IR

**Files:**
- Create: `internal/artifacts/terraform_render.go`
- Create: `internal/artifacts/terraform_render_test.go`
- Create: `internal/testdata/terraform_synthesis/`

- [ ] **Step 1: Write failing renderer golden tests**

Create golden tests for:
- prompt-derived Terraform
- workspace-derived Terraform
- canonical resource naming
- canonical ordering
- stable meta-resource rendering for `comfyui_workflow` and `comfyui_workspace`

- [ ] **Step 2: Run focused tests to verify failure**

Run:

```bash
go test ./internal/artifacts -run 'TestTerraformRender' -v
```

Expected:
- FAIL because the renderer does not exist yet

- [ ] **Step 3: Implement deterministic renderer**

Render canonical Terraform HCL from Terraform IR with stable:
- block ordering
- argument ordering
- resource naming
- connection expression formatting

- [ ] **Step 4: Add golden fixtures**

Store expected rendered Terraform outputs under `internal/testdata/terraform_synthesis/`.

- [ ] **Step 5: Run focused tests to verify pass**

Run:

```bash
go test ./internal/artifacts -run 'TestTerraformRender' -v
```

Expected:
- PASS

- [ ] **Step 6: Commit**

```bash
git add internal/artifacts/terraform_render.go internal/artifacts/terraform_render_test.go internal/testdata/terraform_synthesis
git commit -m "Render canonical Terraform from Terraform IR"
```

---

## Chunk 5: Prompt and Workspace Synthesis Surfaces

### Task 5: Expose prompt/workspace -> Terraform synthesis data sources

**Files:**
- Create: `internal/datasources/prompt_to_terraform.go`
- Create: `internal/datasources/prompt_to_terraform_test.go`
- Create: `internal/datasources/workspace_to_terraform.go`
- Create: `internal/datasources/workspace_to_terraform_test.go`
- Modify: `internal/provider/provider.go`
- Modify: `README.md`

- [ ] **Step 1: Write failing synthesis data source tests**

Add tests proving the new data sources return:
- canonical HCL
- Terraform IR JSON
- fidelity class
- preserved/synthesized/unsupported fields
- notes

- [ ] **Step 2: Run focused tests to verify failure**

Run:

```bash
go test ./internal/datasources -run 'TestPromptToTerraform|TestWorkspaceToTerraform' -v
```

Expected:
- FAIL because the synthesis surfaces do not exist yet

- [ ] **Step 3: Implement prompt synthesis surface**

Translate prompt JSON to Prompt IR, then Terraform IR, then rendered HCL plus fidelity metadata.

- [ ] **Step 4: Implement workspace synthesis surface**

Translate workspace JSON to Workspace IR, then Terraform IR, then rendered HCL plus fidelity metadata.

- [ ] **Step 5: Register new data sources**

Expose the new surfaces through provider registration.

- [ ] **Step 6: Run focused tests to verify pass**

Run:

```bash
go test ./internal/datasources -run 'TestPromptToTerraform|TestWorkspaceToTerraform' -v
```

Expected:
- PASS

- [ ] **Step 7: Update README**

Document the new AI-facing authoring and maintenance surfaces.

- [ ] **Step 8: Commit**

```bash
git add internal/datasources/prompt_to_terraform.go internal/datasources/prompt_to_terraform_test.go internal/datasources/workspace_to_terraform.go internal/datasources/workspace_to_terraform_test.go internal/provider/provider.go README.md
git commit -m "Add Terraform synthesis data sources"
```

---

## Chunk 6: Deep Translation Proofs

### Task 6: Strengthen semantic translation validation beyond counts

**Files:**
- Modify: `internal/artifacts/translation_test.go`
- Modify: `validation/release_e2e/outputs.tf`
- Modify: `validation/release_e2e/browser/tests/release_workflows.spec.ts`
- Modify: `docs/release-validation.md`

- [ ] **Step 1: Write failing semantic equivalence tests**

Add artifact tests that compare:
- preserved fields
- unsupported fields
- sparse slots
- metadata fidelity
- expected lossy vs lossless paths

- [ ] **Step 2: Run focused tests to verify failure**

Run:

```bash
go test ./internal/artifacts -run 'TestTranslation' -v
```

Expected:
- FAIL because field-level golden assertions are not comprehensive yet

- [ ] **Step 3: Expand release harness outputs**

Add stronger Terraform outputs for release scenarios:
- fidelity expectations
- preserved-field expectations
- unsupported-field expectations
- canonical JSON equality where applicable

- [ ] **Step 4: Expand Playwright release checks only where browser evidence adds value**

Keep browser checks focused on:
- graph discoverability
- connectivity
- geometry/layout

Use Terraform/golden assertions for semantic equivalence instead of overloading Playwright.

- [ ] **Step 5: Run the release harness**

Run:

```bash
make release-e2e
```

Expected:
- PASS with stronger semantic outputs and unchanged browser cleanliness

- [ ] **Step 6: Commit**

```bash
git add internal/artifacts/translation_test.go validation/release_e2e/outputs.tf validation/release_e2e/browser/tests/release_workflows.spec.ts docs/release-validation.md
git commit -m "Strengthen release translation proof coverage"
```

---

## Chunk 7: Synthesis End-to-End Harness

### Task 7: Prove the full native artifact -> Terraform -> runtime loop

**Files:**
- Create: `validation/synthesis_e2e/main.tf`
- Create: `validation/synthesis_e2e/outputs.tf`
- Create: `scripts/synthesis-e2e/run.sh`
- Modify: `GNUmakefile`
- Modify: `README.md`
- Modify: `.github/workflows/test.yml`

- [ ] **Step 1: Write the synthesis harness fixture**

Create a fixture that:
- imports native prompt/workspace artifacts
- synthesizes Terraform via new provider data sources
- reassembles/stages the result through provider-owned resources
- asserts expected fidelity and canonical structure

- [ ] **Step 2: Add a runnable harness script**

Create `scripts/synthesis-e2e/run.sh` that prepares the runtime, runs Terraform apply, and emits machine-readable outputs.

- [ ] **Step 3: Add a Make target**

Add:

```bash
make synthesis-e2e
```

- [ ] **Step 4: Run the harness**

Run:

```bash
make synthesis-e2e
```

Expected:
- PASS and emit outputs proving the provider can synthesize maintainable Terraform from native artifacts and round-trip them back through provider assembly

- [ ] **Step 5: Wire CI**

Add deterministic synthesis checks to `test.yml` so the AI-maintainability contract is guarded in CI.

- [ ] **Step 6: Commit**

```bash
git add validation/synthesis_e2e scripts/synthesis-e2e/run.sh GNUmakefile README.md .github/workflows/test.yml
git commit -m "Add synthesis end-to-end validation harness"
```

---

## Chunk 8: Documentation and Final Verification

### Task 8: Document the AI harness contract and verify the full stack

**Files:**
- Create: `docs/ai-maintainability.md`
- Modify: `README.md`
- Modify: `docs/release-validation.md`

- [ ] **Step 1: Document the AI-facing workflow**

Write clear guidance for:
- discovering node contracts
- validating executable workflows
- synthesizing Terraform from native artifacts
- understanding fidelity reports
- using release/runtime/browser harnesses as proof

- [ ] **Step 2: Run the full verification suite**

Run:

```bash
make generate
python3 scripts/extract/test_extractors.py
go test ./... -timeout 120s
make lint test vet
make execution-e2e
make workspace-e2e
make release-e2e
make synthesis-e2e
```

Expected:
- every lane passes
- generated files remain clean after regeneration
- browser artifacts and synthesis artifacts are emitted as expected

- [ ] **Step 3: Verify CI workflow locally where possible**

Run:

```bash
bash scripts/hooks/verify_generated_clean.sh
```

Expected:
- PASS with no generated drift

- [ ] **Step 4: Commit**

```bash
git add docs/ai-maintainability.md README.md docs/release-validation.md
git commit -m "Document AI harness maintainability contract"
```

---

## Execution Notes

- Keep generated artifacts authoritative when the ComfyUI commit SHA is unchanged, even if environment-specific version labels differ.
- Prefer introducing new AI-facing data sources rather than overloading legacy/raw JSON surfaces.
- Keep old raw JSON surfaces as escape hatches, not primary contracts.
- Do not widen handwritten provider logic where generated metadata or canonical IRs can own the behavior.
- Every new translation or synthesis surface must publish explicit fidelity diagnostics.

## Final Definition of Done

The implementation is complete when:

- `comfyui_node_schema` exposes structured machine-readable node contracts
- prompt/workspace validation defaults to executable modes
- canonical Terraform IR exists and is tested
- native prompt/workspace artifacts can be synthesized into deterministic Terraform by provider-owned surfaces
- synthesis outputs include fidelity/provenance metadata
- release/runtime/browser/synthesis harnesses all pass
- CI verifies generated stability and synthesis determinism
- docs clearly explain the AI-harness maintenance loop

