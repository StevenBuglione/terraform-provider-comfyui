# Dynamic Inventory Plan Validation Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `terraform plan` fail for invalid built-in dynamic inventory-backed values by generating validation-kind metadata from pinned ComfyUI code and enforcing live inventory validation for recognized built-in runtime option sources.

**Architecture:** Extend the existing extraction and code-generation pipeline so every generated input is classified as `static_enum`, `dynamic_inventory`, `dynamic_expression`, or `freeform`, with normalized `inventory_kind` metadata for recognized built-in runtime option sources. Generated resources and validation data sources then consume a shared provider runtime inventory service so `terraform plan` becomes a strict correctness gate for built-in pinned ComfyUI configuration.

**Tech Stack:** Go, Terraform Plugin Framework, existing extraction scripts in Python, generated resource templates in Go, ComfyUI client/runtime integration, GNU Make, Go tests.

---

## File Structure

### Existing files to extend

- Modify: `scripts/extract/merge_specs.py`
  - classify input validation kinds and normalize dynamic inventory kinds from extracted ComfyUI metadata
- Modify: `scripts/extract/node_specs.json`
  - persist generated validation-kind and inventory-kind metadata
- Modify: `scripts/extract/node_spec_schema.json`
  - extend the extraction artifact schema for new input validation metadata
- Modify: `scripts/extract/test_extractors.py`
  - assert classification of built-in static enums, dynamic inventories, and unsupported dynamic expressions
- Modify: `cmd/generate/main.go`
  - load generated validation metadata and propagate it to generated Go artifacts
- Modify: `cmd/generate/types.go`
  - add generator model types for validation kind and inventory kind
- Modify: `cmd/generate/templates.go`
  - generate resource validators and node schema metadata from new fields
- Modify: `internal/nodeschema/generated.go`
  - carry generated validation-kind and inventory-kind metadata into runtime
- Modify: `internal/datasources/node_schema.go`
  - expose generated validation metadata to AI harnesses
- Modify: `internal/datasources/node_schema_test.go`
  - verify new structured output fields
- Modify: `internal/datasources/prompt_validation.go`
  - enforce dynamic inventory validation during executable validation
- Modify: `internal/datasources/workspace_validation.go`
  - enforce dynamic inventory validation during executable validation
- Modify: `internal/datasources/validation_semantic_test.go`
  - cover dynamic inventory validation paths through prompt/workspace validation
- Modify: `internal/validation/semantic.go`
  - add dynamic inventory validation checks to executable validation
- Modify: `internal/validation/semantic_test.go`
  - unit-test missing and present runtime inventory-backed values
- Modify: `internal/resources/generated/*.go`
  - generated resource schemas will change automatically once templates emit new validators
- Modify: `internal/provider/provider.go`
  - register any new datasource surface and ensure validator/runtime services can be initialized from provider config
- Modify: `README.md`
  - document strict plan-time dynamic inventory validation and failure behavior
- Modify: `docs/data-sources/node_schema.md`
  - generated docs will expose new validation metadata
- Modify: `docs/data-sources/prompt_validation.md`
  - generated docs should reflect stricter executable validation guarantees
- Modify: `docs/data-sources/workspace_validation.md`
  - generated docs should reflect stricter executable validation guarantees

### New files to create

- Create: `internal/validation/input_validation_kind.go`
  - canonical validation kind constants and helpers shared by generated/runtime code
- Create: `internal/inventory/service.go`
  - shared live inventory lookup service with request-scoped caching
- Create: `internal/inventory/service_test.go`
  - unit tests for category lookup, caching, and failure behavior
- Create: `internal/inventory/kind.go`
  - normalized built-in inventory kind constants and parsing helpers
- Create: `internal/inventory/comfyui.go`
  - ComfyUI client integration for inventory lookup against the configured server
- Create: `internal/inventory/comfyui_test.go`
  - tests for live inventory lookup behavior with mocked ComfyUI responses
- Create: `internal/provider/plan_validation_context.go`
  - wiring layer that makes the shared inventory service available to generated validators and validation data sources
- Create: `internal/datasources/inventory.go`
  - optional public `comfyui_inventory` datasource for AI/debugging visibility
- Create: `internal/datasources/inventory_test.go`
  - tests for the public inventory datasource
- Create: `internal/resources/dynamic_inventory_validator.go`
  - generated-schema-facing validator implementation for string attributes backed by live inventory
- Create: `internal/resources/dynamic_inventory_validator_test.go`
  - validator unit tests for matching, missing, and unreachable inventory
- Create: `internal/resources/unsupported_dynamic_validator.go`
  - validator implementation that hard-fails unsupported dynamic expressions
- Create: `internal/resources/unsupported_dynamic_validator_test.go`
  - unit tests for unsupported dynamic-expression failures
- Create: `internal/testdata/inventory_validation/`
  - plan-time and validation fixtures for built-in runtime-backed resources
- Create: `validation/inventory_plan_e2e/main.tf`
  - end-to-end fixture proving valid plan success and invalid plan failure for runtime-backed inventory inputs
- Create: `validation/inventory_plan_e2e/outputs.tf`
  - machine-readable outputs for the valid fixture
- Create: `scripts/inventory-plan-e2e/run.sh`
  - runner that boots a disposable ComfyUI runtime, stages inventory fixtures, runs valid/invalid plan checks, and asserts expected diagnostics

### Generated files expected from implementation

- Modify: `internal/nodeschema/generated.go`
  - generated node schema metadata extended with validation-kind and inventory-kind fields
- Modify: `internal/resources/generated/resource_*.go`
  - generated schemas emit either static enum validators, dynamic inventory validators, or unsupported dynamic validators
- Modify: generated docs under `docs/resources/*.md`
  - generated descriptions reflect strict dynamic inventory validation support where relevant

---

## Chunk 1: Generate Validation Metadata

### Task 1: Extend extraction artifacts with generated input validation kinds

**Files:**
- Modify: `scripts/extract/merge_specs.py`
- Modify: `scripts/extract/node_spec_schema.json`
- Modify: `scripts/extract/node_specs.json`
- Modify: `scripts/extract/test_extractors.py`
- Modify: `cmd/generate/main.go`
- Modify: `cmd/generate/types.go`
- Modify: `internal/nodeschema/generated.go`

- [ ] **Step 1: Write failing extractor tests for validation-kind classification**

Add tests in `scripts/extract/test_extractors.py` that assert:
- `ByteDanceTextToVideoNode.model` is classified as `static_enum`
- `CheckpointLoaderSimple.ckpt_name` is classified as `dynamic_inventory` with `inventory_kind=checkpoints`
- `LoraLoader.lora_name` is classified as `dynamic_inventory` with `inventory_kind=loras`
- a known unsupported dynamic expression node/input is classified as `dynamic_expression`

- [ ] **Step 2: Run extractor tests to verify failure**

Run:

```bash
python3 scripts/extract/test_extractors.py
```

Expected:
- FAIL because validation-kind and inventory-kind metadata do not exist yet

- [ ] **Step 3: Extend extraction schema**

Update `scripts/extract/node_spec_schema.json` so each input may include:
- `validation_kind`
- `inventory_kind`
- `supports_strict_plan_validation`

- [ ] **Step 4: Implement extraction-time normalization**

In `scripts/extract/merge_specs.py`, classify each input as one of:
- `static_enum`
- `dynamic_inventory`
- `dynamic_expression`
- `freeform`

Normalization rules:
- explicit static options -> `static_enum`
- recognized `folder_paths.get_filename_list('<kind>')` -> `dynamic_inventory` with normalized `inventory_kind=<kind>`
- unknown dynamic source expressions -> `dynamic_expression`
- ordinary scalar or link inputs -> `freeform`

- [ ] **Step 5: Regenerate `node_specs.json`**

Run:

```bash
python3 scripts/extract/merge_specs.py
```

Expected:
- `scripts/extract/node_specs.json` now contains the new validation metadata

- [ ] **Step 6: Extend generator input models**

Update `cmd/generate/main.go` and `cmd/generate/types.go` to decode and carry:
- `validation_kind`
- `inventory_kind`
- `supports_strict_plan_validation`

- [ ] **Step 7: Emit generated node schema metadata**

Update generator output so `internal/nodeschema/generated.go` includes the new fields for required and optional inputs.

- [ ] **Step 8: Re-run extractor tests to verify pass**

Run:

```bash
python3 scripts/extract/test_extractors.py
```

Expected:
- PASS

- [ ] **Step 9: Commit**

```bash
git add scripts/extract/merge_specs.py scripts/extract/node_spec_schema.json scripts/extract/node_specs.json scripts/extract/test_extractors.py cmd/generate/main.go cmd/generate/types.go internal/nodeschema/generated.go
git commit -m "Generate dynamic inventory validation metadata"
```

---

## Chunk 2: Shared Live Inventory Service

### Task 2: Build the provider-owned runtime inventory layer

**Files:**
- Create: `internal/inventory/kind.go`
- Create: `internal/inventory/service.go`
- Create: `internal/inventory/service_test.go`
- Create: `internal/inventory/comfyui.go`
- Create: `internal/inventory/comfyui_test.go`
- Create: `internal/provider/plan_validation_context.go`
- Modify: `internal/provider/provider.go`

- [ ] **Step 1: Write failing inventory-service unit tests**

Add tests for:
- normalized inventory-kind parsing
- successful live lookup for `checkpoints`
- caching repeated lookups within one operation
- hard failure when the ComfyUI server cannot be reached
- hard failure for unknown inventory kinds

- [ ] **Step 2: Run focused inventory tests to verify failure**

Run:

```bash
go test ./internal/inventory -run 'TestInventory' -v
```

Expected:
- FAIL because the inventory service does not exist yet

- [ ] **Step 3: Implement normalized inventory kinds**

Create `internal/inventory/kind.go` with constants/helpers for built-in recognized categories such as:
- `checkpoints`
- `loras`
- `text_encoders`
- `controlnet`
- `vae`
- `configs`
- `diffusion_models`
- `hypernetworks`
- `style_models`
- `clip_vision`
- `audio_encoders`

- [ ] **Step 4: Implement the ComfyUI-backed lookup client**

Create `internal/inventory/comfyui.go` to resolve inventory values from the configured ComfyUI server for a normalized inventory kind. Use existing provider/client patterns rather than inventing a parallel HTTP layer.

- [ ] **Step 5: Implement the shared cached service**

Create `internal/inventory/service.go` with request-scoped caching and strict error semantics. No fallback to stale or partial results.

- [ ] **Step 6: Wire provider context**

Add `internal/provider/plan_validation_context.go` and update `internal/provider/provider.go` so validators and validation data sources can obtain the shared inventory service from provider configuration/state.

- [ ] **Step 7: Re-run focused inventory tests to verify pass**

Run:

```bash
go test ./internal/inventory -run 'TestInventory' -v
```

Expected:
- PASS

- [ ] **Step 8: Commit**

```bash
git add internal/inventory/kind.go internal/inventory/service.go internal/inventory/service_test.go internal/inventory/comfyui.go internal/inventory/comfyui_test.go internal/provider/plan_validation_context.go internal/provider/provider.go
git commit -m "Add shared live inventory validation service"
```

---

## Chunk 3: Generated Plan Validators

### Task 3: Make generated resource schemas enforce strict dynamic inventory validation

**Files:**
- Create: `internal/resources/dynamic_inventory_validator.go`
- Create: `internal/resources/dynamic_inventory_validator_test.go`
- Create: `internal/resources/unsupported_dynamic_validator.go`
- Create: `internal/resources/unsupported_dynamic_validator_test.go`
- Modify: `cmd/generate/templates.go`
- Modify: generated files under `internal/resources/generated/`

- [ ] **Step 1: Write failing validator unit tests**

Add tests that prove:
- a dynamic inventory validator accepts a present value
- it rejects a missing value
- it rejects an unreachable inventory service
- an unsupported dynamic-expression validator always fails with a clear diagnostic

- [ ] **Step 2: Run focused validator tests to verify failure**

Run:

```bash
go test ./internal/resources -run 'TestDynamicInventoryValidator|TestUnsupportedDynamicValidator' -v
```

Expected:
- FAIL because the validators do not exist yet

- [ ] **Step 3: Implement strict dynamic inventory validator**

Create `internal/resources/dynamic_inventory_validator.go` that:
- reads normalized `inventory_kind`
- queries the shared inventory service
- rejects unknown values
- rejects unavailable inventory lookups

- [ ] **Step 4: Implement unsupported dynamic validator**

Create `internal/resources/unsupported_dynamic_validator.go` that fails plan with a deterministic message explaining the generated input cannot yet be strictly validated from pinned ComfyUI metadata.

- [ ] **Step 5: Update generator templates**

In `cmd/generate/templates.go`, emit schema validators by `validation_kind`:
- `static_enum` -> `OneOf(...)`
- `dynamic_inventory` -> new dynamic inventory validator
- `dynamic_expression` -> unsupported dynamic validator
- `freeform` -> existing scalar validation only

- [ ] **Step 6: Regenerate resources**

Run:

```bash
make generate
```

Expected:
- generated resource files now include the new validator wiring

- [ ] **Step 7: Re-run focused validator tests to verify pass**

Run:

```bash
go test ./internal/resources -run 'TestDynamicInventoryValidator|TestUnsupportedDynamicValidator' -v
```

Expected:
- PASS

- [ ] **Step 8: Verify generated cleanliness**

Run:

```bash
bash scripts/hooks/verify_generated_clean.sh
```

Expected:
- PASS

- [ ] **Step 9: Commit**

```bash
git add internal/resources/dynamic_inventory_validator.go internal/resources/dynamic_inventory_validator_test.go internal/resources/unsupported_dynamic_validator.go internal/resources/unsupported_dynamic_validator_test.go cmd/generate/templates.go internal/resources/generated internal/nodeschema/generated.go
git commit -m "Generate strict dynamic inventory plan validators"
```

---

## Chunk 4: Expose and Enforce Validation Metadata

### Task 4: Expand AI-facing schema surfaces and executable validation

**Files:**
- Modify: `internal/datasources/node_schema.go`
- Modify: `internal/datasources/node_schema_test.go`
- Modify: `internal/validation/semantic.go`
- Modify: `internal/validation/semantic_test.go`
- Modify: `internal/datasources/prompt_validation.go`
- Modify: `internal/datasources/workspace_validation.go`
- Modify: `internal/datasources/validation_semantic_test.go`
- Create: `internal/validation/input_validation_kind.go`

- [ ] **Step 1: Write failing node-schema and executable-validation tests**

Add tests that prove:
- `comfyui_node_schema` exposes `validation_kind`, `inventory_kind`, and `supports_strict_plan_validation`
- prompt validation fails when a checkpoint or LoRA reference is missing from live inventory
- workspace validation fails on the same conditions after translation to prompt form

- [ ] **Step 2: Run focused tests to verify failure**

Run:

```bash
go test ./internal/datasources ./internal/validation -run 'TestNodeSchema|Test.*DynamicInventory|TestValidatePrompt|TestValidateWorkspace' -v
```

Expected:
- FAIL because the new schema fields and runtime checks are not implemented yet

- [ ] **Step 3: Add validation-kind helpers**

Create `internal/validation/input_validation_kind.go` with shared constants/helpers so datasources and runtime validators use one canonical set of kind names.

- [ ] **Step 4: Expand `comfyui_node_schema`**

Update `internal/datasources/node_schema.go` to expose:
- `validation_kind`
- `inventory_kind`
- `supports_strict_plan_validation`

for both required and optional inputs.

- [ ] **Step 5: Extend executable validation**

Update `internal/validation/semantic.go` so executable prompt/workspace validation:
- inspects generated input validation metadata
- checks live inventory for referenced `dynamic_inventory` values
- hard-fails for `dynamic_expression` when strict validation is required

- [ ] **Step 6: Wire prompt/workspace validation datasources**

Update `internal/datasources/prompt_validation.go` and `internal/datasources/workspace_validation.go` so they pass the shared inventory service into semantic validation.

- [ ] **Step 7: Re-run focused tests to verify pass**

Run:

```bash
go test ./internal/datasources ./internal/validation -run 'TestNodeSchema|Test.*DynamicInventory|TestValidatePrompt|TestValidateWorkspace' -v
```

Expected:
- PASS

- [ ] **Step 8: Commit**

```bash
git add internal/datasources/node_schema.go internal/datasources/node_schema_test.go internal/validation/input_validation_kind.go internal/validation/semantic.go internal/validation/semantic_test.go internal/datasources/prompt_validation.go internal/datasources/workspace_validation.go internal/datasources/validation_semantic_test.go
git commit -m "Enforce dynamic inventory validation in schema and executable checks"
```

---

## Chunk 5: Public Inventory Surface and E2E Proof

### Task 5: Add inventory visibility and end-to-end plan proof

**Files:**
- Create: `internal/datasources/inventory.go`
- Create: `internal/datasources/inventory_test.go`
- Create: `validation/inventory_plan_e2e/main.tf`
- Create: `validation/inventory_plan_e2e/outputs.tf`
- Create: `scripts/inventory-plan-e2e/run.sh`
- Modify: `GNUmakefile`
- Modify: `README.md`
- Modify: `docs/data-sources/node_schema.md`
- Modify: `docs/data-sources/prompt_validation.md`
- Modify: `docs/data-sources/workspace_validation.md`
- Create: `docs/data-sources/inventory.md`

- [ ] **Step 1: Write failing inventory-datasource and e2e harness tests**

Add tests that prove:
- `comfyui_inventory` returns built-in inventory categories from the live server
- the e2e harness succeeds for a valid built-in runtime-backed model name
- the e2e harness fails plan for an invalid built-in runtime-backed model name with the expected diagnostic

- [ ] **Step 2: Run focused datasource tests to verify failure**

Run:

```bash
go test ./internal/datasources -run 'TestInventoryDataSource' -v
```

Expected:
- FAIL because the datasource does not exist yet

- [ ] **Step 3: Implement `comfyui_inventory`**

Create `internal/datasources/inventory.go` exposing live inventory values by normalized kind for AI/debugging use. Use the shared inventory service; do not duplicate lookup logic.

- [ ] **Step 4: Add the e2e harness**

Create `validation/inventory_plan_e2e` and `scripts/inventory-plan-e2e/run.sh` that:
- boot a disposable ComfyUI runtime
- stage known built-in inventory-backed configuration
- run a valid `terraform plan`
- run an invalid `terraform plan`
- assert the invalid case fails before apply with the expected message

- [ ] **Step 5: Add Make integration**

Update `GNUmakefile` with a target such as:

```bash
make inventory-plan-e2e
```

- [ ] **Step 6: Regenerate docs**

Run:

```bash
make docs
```

Expected:
- docs updated for `comfyui_node_schema`, `comfyui_inventory`, and stricter validation surfaces

- [ ] **Step 7: Re-run focused datasource tests**

Run:

```bash
go test ./internal/datasources -run 'TestInventoryDataSource' -v
```

Expected:
- PASS

- [ ] **Step 8: Run end-to-end harness**

Run:

```bash
make inventory-plan-e2e
```

Expected:
- valid plan passes
- invalid plan fails at plan time with deterministic diagnostics

- [ ] **Step 9: Commit**

```bash
git add internal/datasources/inventory.go internal/datasources/inventory_test.go validation/inventory_plan_e2e scripts/inventory-plan-e2e/run.sh GNUmakefile README.md docs/data-sources/node_schema.md docs/data-sources/prompt_validation.md docs/data-sources/workspace_validation.md docs/data-sources/inventory.md
git commit -m "Add dynamic inventory validation proofs and docs"
```

---

## Chunk 6: Full Verification

### Task 6: Run full verification and record the contract

**Files:**
- Modify if needed after verification: any files touched by earlier chunks

- [ ] **Step 1: Run generator verification**

Run:

```bash
make generate
bash scripts/hooks/verify_generated_clean.sh
```

Expected:
- PASS

- [ ] **Step 2: Run targeted Go tests**

Run:

```bash
go test ./internal/inventory ./internal/resources ./internal/datasources ./internal/validation -timeout 120s
```

Expected:
- PASS

- [ ] **Step 3: Run full Go test suite**

Run:

```bash
go test ./... -timeout 120s
```

Expected:
- PASS

- [ ] **Step 4: Run docs and lint suite**

Run:

```bash
make docs
make lint test vet
```

Expected:
- PASS

- [ ] **Step 5: Run end-to-end inventory validation harness**

Run:

```bash
make inventory-plan-e2e
```

Expected:
- PASS

- [ ] **Step 6: Commit any final verification-driven fixes**

```bash
git add -A
git commit -m "Finalize dynamic inventory plan validation"
```

---

## Notes for Implementers

- Do not hand-maintain per-resource allowlists. The entire point of this work is that the validation contract should follow generated ComfyUI metadata.
- Reuse the existing provider/client wiring for server access. Do not build a separate ad hoc HTTP stack unless the existing client cannot represent inventory lookups cleanly.
- Unknown or unsupported dynamic expressions must fail strictly. Do not degrade them to warnings or runtime-only validation.
- Keep the inventory service bounded to built-in pinned ComfyUI support. Do not expand scope to third-party custom-node heuristics.
- Any new public datasource should be helpful to AI harnesses, but enforcement must remain in generated plan validators and executable validation.

