# ComfyUI Validation and File Lifecycle Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add semantic validation driven by live ComfyUI metadata and implement Terraform-managed upload/download file resources for ComfyUI inputs and outputs.

**Architecture:** Build a shared semantic-validation core on top of phase-1 prompt/workspace models and `/object_info`, then expose that core through dedicated validation data sources and `comfyui_workflow` preflight checks. Implement file lifecycle as three focused resources: remote upload for images, mask upload with explicit original-image references, and local materialization for remote outputs.

**Tech Stack:** Go 1.25, Terraform Plugin Framework, existing internal prompt/workspace models, ComfyUI `/object_info`, `/upload/image`, `/upload/mask`, and `/view` endpoints, Go unit tests.

---

## Spec Reference

- `docs/superpowers/specs/2026-04-10-comfyui-validation-file-lifecycle-design.md`

## File Structure

**Create:**

- `internal/validation/report.go`
- `internal/validation/semantic.go`
- `internal/validation/semantic_test.go`
- `internal/datasources/prompt_validation.go`
- `internal/datasources/workspace_validation.go`
- `internal/datasources/validation_semantic_test.go`
- `internal/resources/uploaded_image_resource.go`
- `internal/resources/uploaded_mask_resource.go`
- `internal/resources/output_artifact_resource.go`
- `internal/resources/remote_file_helpers.go`
- `internal/resources/remote_file_helpers_test.go`

**Modify:**

- `internal/client/client.go`
- `internal/client/types.go`
- `internal/client/client_test.go`
- `internal/resources/workflow_resource.go`
- `internal/resources/workflow_resource_test.go`
- `internal/provider/provider.go`
- `README.md`

**Generate / refresh if needed:**

- `docs/resources/*.md`

---

## Chunk 1: Semantic Validation Core

### Task 1: Write failing semantic-validation tests

**Files:**
- Create: `internal/validation/semantic_test.go`

- [ ] **Step 1: Add a valid prompt test**

Cover a prompt whose nodes all exist in a stubbed `map[string]client.NodeInfo` and whose links and required inputs are correct.

- [ ] **Step 2: Add failing tests for missing node type, missing required input, unknown input name, bad output slot, and type mismatch**

Each test should assert the specific report entry text, not only `valid=false`.

- [ ] **Step 3: Add a no-output-node test**

Use `OutputNode: true` metadata rather than name heuristics.

- [ ] **Step 4: Run the focused test file to prove failure**

Run:

```bash
go test ./internal/validation -v
```

Expected: FAIL because the package does not exist yet.

### Task 2: Implement the shared semantic-validation package

**Files:**
- Create: `internal/validation/report.go`
- Create: `internal/validation/semantic.go`
- Modify: `internal/validation/semantic_test.go`

- [ ] **Step 1: Define the report model**

Include plain Go fields for `Valid`, `Errors`, `Warnings`, `ValidatedNodeCount`, `ErrorCount`, and `WarningCount`.

- [ ] **Step 2: Implement prompt validation helpers**

Add helpers for:

- known node lookup
- required input detection
- allowed input-name detection from required/optional/hidden metadata
- link resolution against prompt IDs
- output-slot and link-type validation
- native `output_node` detection

- [ ] **Step 3: Keep literal-value validation intentionally narrow**

Only validate link semantics and metadata-derived shape in this chunk. Do not invent deep scalar validators for every ComfyUI widget type.

- [ ] **Step 4: Re-run semantic-validation tests**

Run:

```bash
go test ./internal/validation -v
```

Expected: PASS.

## Chunk 2: Validation Data Sources and Workflow Preflight

### Task 3: Add failing data source tests

**Files:**
- Create: `internal/datasources/validation_semantic_test.go`

- [ ] **Step 1: Add a prompt-validation data source test**

Assert it returns `valid`, `error_count`, `warning_count`, `errors`, `warnings`, `validated_node_count`, and `normalized_json` for invalid prompt JSON.

- [ ] **Step 2: Add a workspace-validation data source test**

Assert it translates workspace JSON, returns `translated_prompt_json`, returns `translation_fidelity`, `translation_preserved_fields`, `translation_synthesized_fields`, `translation_unsupported_fields`, `translation_notes`, and reports validation errors against metadata.

- [ ] **Step 3: Run the focused tests to prove failure**

Run:

```bash
go test ./internal/datasources -run 'Validation' -v
```

Expected: FAIL because the new data sources do not exist yet.

### Task 4: Implement validation data sources

**Files:**
- Create: `internal/datasources/prompt_validation.go`
- Create: `internal/datasources/workspace_validation.go`
- Modify: `internal/datasources/validation_semantic_test.go`

- [ ] **Step 1: Implement `comfyui_prompt_validation`**

Support the same `path`/`json` input pattern used by the phase-1 import data sources and expose the full shared validation contract (`valid`, counts, `errors`, `warnings`, `validated_node_count`, `normalized_json`) with `errors`/`warnings` modeled as `list(string)`.

- [ ] **Step 2: Implement `comfyui_workspace_validation`**

Parse workspace JSON, translate to prompt with existing artifacts helpers, then run semantic validation.

- [ ] **Step 3: Expose `translated_prompt_json` and translation-report outputs**

Match the spec by returning the translated prompt JSON plus `translation_fidelity`, `translation_preserved_fields`, `translation_synthesized_fields`, `translation_unsupported_fields`, and `translation_notes` alongside validation results.

- [ ] **Step 4: Reuse shared list and input-loading helpers**

If helper extraction is needed, move it into a shared datasources helper file instead of duplicating it.

- [ ] **Step 5: Re-run focused validation data source tests**

Run:

```bash
go test ./internal/datasources -run 'Validation' -v
```

Expected: PASS.

### Task 5: Add failing workflow preflight tests

**Files:**
- Modify: `internal/resources/workflow_resource_test.go`

- [ ] **Step 1: Add a schema test for `validate_before_execute` and `validation_summary_json`**

Assert `validate_before_execute` is optional+computed with a default of `true`.

- [ ] **Step 2: Add a behavior test that invalid metadata blocks queueing before `/prompt`**

- [ ] **Step 3: Add a behavior test that disabling preflight skips validation**

- [ ] **Step 4: Run the focused workflow tests to prove failure**

Run:

```bash
go test ./internal/resources -run 'Workflow.*Validation|WorkflowSchema' -v
```

Expected: FAIL because the workflow resource does not expose the new surface yet.

### Task 6: Implement workflow preflight validation

**Files:**
- Modify: `internal/resources/workflow_resource.go`
- Modify: `internal/resources/workflow_resource_test.go`

- [ ] **Step 1: Add `validate_before_execute` and `validation_summary_json` to the schema/model**

Make `validate_before_execute` optional+computed with a default of `true` so existing workflows gain safe preflight behavior without new configuration.

- [ ] **Step 2: Fetch `/object_info` and run the shared validator before queueing when enabled**

- [ ] **Step 3: Persist the structured validation summary into state**

- [ ] **Step 4: Re-run focused workflow validation tests**

Run:

```bash
go test ./internal/resources -run 'Workflow.*Validation|WorkflowSchema' -v
```

Expected: PASS.

## Chunk 3: Client Upload/Download Primitives

### Task 7: Add failing client tests for upload/download helpers

**Files:**
- Modify: `internal/client/client_test.go`

- [ ] **Step 1: Add an upload-image request test**

Assert multipart upload hits `/upload/image` with `image`, `type`, `subfolder`, and `overwrite`, with `overwrite=true` serialized as the literal string `"true"`.

- [ ] **Step 2: Add an upload-mask request test**

Assert multipart upload hits `/upload/mask` and includes the required `original_ref` payload.

- [ ] **Step 3: Add a download test for `/view` binary content**

Assert filename/subfolder/type query propagation and returned bytes.

- [ ] **Step 4: Run the focused client tests to prove failure**

Run:

```bash
go test ./internal/client -run 'Upload|DownloadView' -v
```

Expected: FAIL because the helpers do not exist yet.

### Task 8: Implement client upload/download helpers

**Files:**
- Modify: `internal/client/client.go`
- Modify: `internal/client/types.go`
- Modify: `internal/client/client_test.go`

- [ ] **Step 1: Add result types for upload and downloaded output metadata**

- [ ] **Step 2: Implement multipart upload helpers**

Expose separate methods for image and mask uploads so the resource layer stays thin.

- [ ] **Step 3: Implement binary download helper for `/view`**

Return bytes plus metadata needed by the resource layer.

- [ ] **Step 4: Re-run focused client tests**

Run:

```bash
go test ./internal/client -run 'Upload|DownloadView' -v
```

Expected: PASS.

## Chunk 4: File Lifecycle Resources

### Task 9: Add failing remote file helper and resource tests

**Files:**
- Create: `internal/resources/remote_file_helpers.go`
- Create: `internal/resources/remote_file_helpers_test.go`
- Modify: `internal/resources/workflow_resource_test.go` only if shared fixtures are helpful

- [ ] **Step 1: Add tests for local source hashing and managed output-file writes**

- [ ] **Step 2: Add tests for upload delete-warning behavior**

- [ ] **Step 3: Add tests for upload Read remote-presence checks**

Cover the case where an uploaded file no longer exists remotely and ensure the resource responds according to the documented lifecycle.

- [ ] **Step 4: Add tests for output artifact path-change cleanup**

- [ ] **Step 5: Run the focused resource-helper tests to prove failure**

Run:

```bash
go test ./internal/resources -run 'RemoteFile|OutputArtifact|Uploaded(Image|Mask)' -v
```

Expected: FAIL because the helper/resource code does not exist yet.

### Task 10: Implement shared remote-file helpers and upload resources

**Files:**
- Create: `internal/resources/uploaded_image_resource.go`
- Create: `internal/resources/uploaded_mask_resource.go`
- Create: `internal/resources/remote_file_helpers.go`
- Modify: `internal/resources/remote_file_helpers_test.go`

- [ ] **Step 1: Implement shared remote-file helper functions**

Include local hash computation, local write helpers, and old-path cleanup.

- [ ] **Step 2: Add the full `comfyui_uploaded_image` schema**

Include at minimum:

- `id` computed from the actual uploaded server path returned by ComfyUI (`{type}/{subfolder}/{filename}`)
- `file_path` required
- `filename` optional+computed
- `subfolder` optional
- `type` optional+computed with allowed values `input|output|temp` and default `input`
- `overwrite` optional+computed with default `true`
- `sha256` computed
- `url` computed

- [ ] **Step 3: Implement `comfyui_uploaded_image` Create/Read/Update/Delete**

Use the client upload helper and store server-returned `filename`, `subfolder`, `type`, `url`, and `sha256`. Treat the response filename as source of truth for ID construction.

- [ ] **Step 4: Add the full `comfyui_uploaded_mask` schema**

Include the upload attributes plus:

- `original_filename` required
- `original_subfolder` optional
- `original_type` optional+computed default `output`
- `type` optional+computed with allowed values `input|output|temp` and default `input`

- [ ] **Step 5: Implement `comfyui_uploaded_mask` Create/Read/Update/Delete**

Construct the upstream `original_ref` JSON from typed Terraform attributes instead of exposing raw JSON.

- [ ] **Step 6: Re-run focused upload-resource tests**

Run:

```bash
go test ./internal/resources -run 'RemoteFile|Uploaded(Image|Mask)' -v
```

Expected: PASS.

### Task 11: Implement `comfyui_output_artifact`

**Files:**
- Create: `internal/resources/output_artifact_resource.go`
- Modify: `internal/resources/remote_file_helpers.go`
- Modify: `internal/resources/remote_file_helpers_test.go`

- [ ] **Step 1: Add the full `comfyui_output_artifact` schema**

Include:

- `id` computed absolute local path
- `filename` required
- `subfolder` optional
- `type` optional+computed default `output`
- `path` required
- `sha256` computed
- `content_length` computed
- `url` computed

- [ ] **Step 2: Implement `comfyui_output_artifact` Create/Read/Update/Delete**

Download remote bytes, materialize them to `path`, compute `sha256`, compute `content_length`, return a hard error when the remote file is missing during Create, and on Read re-download only when the managed local file is absent.

- [ ] **Step 3: Re-run focused output-artifact tests**

Run:

```bash
go test ./internal/resources -run 'RemoteFile|OutputArtifact' -v
```

Expected: PASS.

## Chunk 5: Registration, Docs, and Full Verification

### Task 12: Register new provider surfaces

**Files:**
- Modify: `internal/provider/provider.go`

- [ ] **Step 1: Register the validation data sources**

- [ ] **Step 2: Register the upload and output-artifact resources**

- [ ] **Step 3: Run a provider-focused smoke test**

Run:

```bash
go test ./internal/provider ./internal/datasources ./internal/resources -v
```

Expected: PASS.

### Task 13: Update docs and examples

**Files:**
- Modify: `README.md`
- Generate / refresh: `docs/resources/*.md` if docs output changes

- [ ] **Step 1: Add a short README section for validation and file lifecycle**

- [ ] **Step 2: Regenerate provider docs if schema docs changed**

Run:

```bash
make docs
```

- [ ] **Step 3: Verify only intended docs changed**

Run:

```bash
git --no-pager diff -- docs README.md
```

### Task 14: Run full verification

**Files:**
- No new file targets

- [ ] **Step 1: Run the full Go suite**

Run:

```bash
go test ./... -v -timeout 120s
```

Expected: PASS.

- [ ] **Step 2: Commit phase 2 immediately after the validation chunk is complete**

This commit should happen after Task 6 so semantic validation lands as its own logical unit.

```bash
git add .
git commit -m "feat: add ComfyUI semantic validation"
```

- [ ] **Step 3: Commit phase 3 immediately after the file-lifecycle chunk is complete**

This commit should happen after Task 11 so file lifecycle lands as its own logical unit.

```bash
git add .
git commit -m "feat: add ComfyUI file lifecycle resources"
```

- [ ] **Step 4: Run provider verification if repo-wide checks are needed**

Run:

```bash
make verify
```

Expected: PASS, or capture any pre-existing failures before proceeding.
