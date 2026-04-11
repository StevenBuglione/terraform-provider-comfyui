# ComfyUI Semantic Validation and File Lifecycle Design

## Summary

Implement the next two round-trip phases by adding:

1. native-metadata-driven semantic validation for prompt and workspace artifacts
2. ComfyUI-backed file lifecycle surfaces for uploaded inputs and downloaded outputs

This keeps Terraform as the AI harness control plane while respecting Terraform semantics: validation remains read-only and explicit, while file resources only own lifecycle boundaries the provider can describe honestly.

## Problem Statement

Phase 1 gave the provider truthful prompt/workspace models, translation data sources, managed local artifact files, and `/prompt` wrapper fidelity. Two important gaps remain:

- the provider still cannot validate prompt/workspace semantics against live `/object_info` metadata
- the provider still cannot manage ComfyUI-backed file movement for uploaded inputs and downloaded outputs

Today the provider can only:

- do lightweight local workflow validation for virtual-node assembly
- construct output URLs and check existence through `GET /view`

It still cannot:

- detect missing node types, missing required inputs, unexpected inputs, bad link targets, or output-node presence from native metadata
- upload input images or masks through `POST /upload/image` and `POST /upload/mask`
- materialize remote output files to local disk as Terraform-managed artifacts

## Goals

1. Add semantic validation that uses live `/object_info` metadata instead of string heuristics.
2. Make validation available as explicit Terraform surfaces for prompt and workspace artifacts.
3. Fail `comfyui_workflow` execution before queueing when semantic validation finds hard errors.
4. Add Terraform-managed file resources for ComfyUI uploads and downloaded outputs.
5. Keep lifecycle contracts honest where the ComfyUI API does not expose delete endpoints.

## Non-Goals

- Do not solve generated node-schema fidelity in this slice.
- Do not add the editor-native `comfyui_subgraph` resource in this slice.
- Do not invent remote delete behavior that ComfyUI does not support.
- Do not replace the existing `comfyui_output` data source.

## Approaches Considered

### Option 1: Extend existing import/translation data sources in place

Add validation outputs directly onto `comfyui_prompt_json`, `comfyui_workspace_json`, and translation data sources, then add one generic file resource for uploads and downloads.

**Pros:** less surface area, faster to wire.
**Cons:** overloads data sources with multiple responsibilities, muddies phase boundaries, and creates one awkward file resource that mixes upload and download semantics.

### Option 2: Add explicit validation data sources plus separate upload/output resources (**recommended**)

Introduce dedicated validation data sources backed by a shared semantic-validation package, then add separate resources for uploaded images, uploaded masks, and downloaded output artifacts.

**Pros:** clearest Terraform semantics, matches the phase split, keeps import/translation surfaces stable, and lets workflow execution reuse the same validation core.
**Cons:** adds more Terraform nouns.

### Option 3: Push validation into `comfyui_workflow` only and keep file handling as data sources

Use workflow preflight validation before `/prompt`, but skip standalone validation surfaces and keep file operations as read-only lookups.

**Pros:** smallest change to public API.
**Cons:** weak for AI feedback loops because validation is only available at execution time, and file movement remains under-modeled.

## Recommended Design

Choose **Option 2**.

Validation should be a first-class read-only capability, and file movement should use focused resources whose lifecycle boundaries are explicit. This gives the AI harness stable Terraform surfaces for both feedback loops and artifact management without overloading existing resources.

## Phase-2 Surface Inventory

| Terraform type | Name | Purpose |
| --- | --- | --- |
| Data source | `comfyui_prompt_validation` | Validate native prompt JSON against live `/object_info` metadata |
| Data source | `comfyui_workspace_validation` | Translate workspace/subgraph JSON to prompt form, then validate semantics |
| Resource extension | `comfyui_workflow` | Add optional preflight semantic validation before queueing |

### Validation contract

#### `comfyui_prompt_validation` inputs

- `path` — optional file path to prompt JSON
- `json` — optional raw prompt JSON string

Exactly one of `path` or `json` must be set.

#### `comfyui_workspace_validation` inputs

- `path` — optional file path to workspace/subgraph JSON
- `json` — optional raw workspace/subgraph JSON string

Exactly one of `path` or `json` must be set. This data source performs translation internally so callers do not need to manually chain `comfyui_workspace_to_prompt` just to get semantic validation.

#### Shared outputs

Both validation data sources return:

- `valid` — boolean
- `error_count`
- `warning_count`
- `errors` — `list(string)`
- `warnings` — `list(string)`
- `validated_node_count`
- `normalized_json` — normalized source artifact JSON

`comfyui_workspace_validation` also returns:

- `translated_prompt_json`
- `translation_fidelity`
- `translation_preserved_fields` — `list(string)`
- `translation_synthesized_fields` — `list(string)`
- `translation_unsupported_fields` — `list(string)`
- `translation_notes` — `list(string)`

This keeps the translation-report contract aligned with the phase-1 translation data sources while still exposing validation separately.

### Hard-error rules

The first validation slice should treat these as errors:

1. node class type not found in `/object_info`
2. required input missing, excluding server-injected `hidden` inputs
3. input name not present in required/optional metadata
4. linked source node missing
5. linked source output slot out of range
6. linked source output type incompatible with target input type
7. workflow has no native `output_node=true` node

Hidden inputs such as `prompt`, `extra_pnginfo`, and `unique_id` are server-injected and must not be required from user-supplied prompt JSON.

### Warning rules

This slice should keep warnings narrow:

1. node metadata is incomplete enough to prevent a stronger type check
2. workspace validation required a lossy translation before semantic checks ran
3. wildcard type compatibility (`*`) bypassed a stricter type comparison

If either the source output type or the target input type is `*`, the validator treats the connection as compatible.

### `comfyui_workflow` behavior

Add `validate_before_execute` as an optional+computed attribute defaulting to `true`.

If `execute=true` and validation is enabled:

1. build the prompt as today
2. fetch `/object_info`
3. run the shared semantic validator
4. fail before `POST /prompt` when hard validation errors exist

If `execute=false`, the provider should not fail the resource on missing output-node semantics that only matter for queueable execution.

The resource should expose a computed `validation_summary_json` with this JSON shape:

- `valid`
- `error_count`
- `warning_count`
- `errors` — `list(string)`
- `warnings` — `list(string)`
- `validated_node_count`

Preflight validation complements ComfyUI's own server-side `node_errors`; it does not replace them. If `/prompt` still returns `node_errors`, `comfyui_workflow` must surface that failure as it does today.

## Phase-3 Surface Inventory

| Terraform type | Name | Purpose |
| --- | --- | --- |
| Resource | `comfyui_uploaded_image` | Upload a local image into ComfyUI via `POST /upload/image` |
| Resource | `comfyui_uploaded_mask` | Upload a mask via `POST /upload/mask` against an existing ComfyUI image reference |
| Resource | `comfyui_output_artifact` | Download a remote ComfyUI output from `GET /view` to a local file |

### `comfyui_uploaded_image`

Recommended attributes:

- `id` — computed from the actual server path returned by upload (`{type}/{subfolder}/{filename}`)
- `file_path` — required local source path
- `filename` — optional requested filename, computed to actual server name
- `subfolder` — optional
- `type` — optional, allowed values `input`, `output`, or `temp`, default `input`
- `overwrite` — optional, default `true`
- `sha256` — computed local file hash
- `url` — computed view URL

Only one Terraform resource should manage a given remote server path. The provider should treat the response filename as source of truth if ComfyUI renames the upload.

Lifecycle:

- **Create:** multipart upload to ComfyUI
- serialize `overwrite=true` into the multipart form as the literal string `"true"`
- **Read:** verify remote presence through `GET /view`/HEAD-compatible check and refresh computed fields
- **Update:** re-upload when local file hash or placement arguments change
- **Delete:** remove from Terraform state and emit a warning that ComfyUI exposes no file-delete endpoint

This delete contract is intentionally explicit. The provider must not pretend remote deletion occurred.

### `comfyui_uploaded_mask`

`POST /upload/mask` is **not** the same contract as image upload. Upstream ComfyUI requires an `original_ref` describing the image whose alpha channel should be replaced. The provider should keep Terraform inputs typed and construct that `original_ref` object internally.

Recommended attributes:

- `id` — computed from the actual server path returned by upload (`{type}/{subfolder}/{filename}`)
- `file_path` — required local mask path
- `filename` — optional+computed requested destination filename
- `subfolder` — optional
- `type` — optional+computed, allowed values `input`, `output`, or `temp`, default `input`
- `overwrite` — optional+computed, default `true`
- `original_filename` — required
- `original_subfolder` — optional
- `original_type` — optional+computed, default `output`
- `sha256` — computed local file hash
- `url` — computed view URL

Lifecycle:

- **Create:** multipart upload to `/upload/mask` with provider-constructed `original_ref`
- **Read:** verify remote presence through `GET /view`/HEAD-compatible check and refresh computed fields
- **Update:** re-upload when the local mask hash or original-reference coordinates change
- **Delete:** remove from Terraform state and emit a warning that ComfyUI exposes no file-delete endpoint

### `comfyui_output_artifact`

Recommended attributes:

- `id` — computed absolute local path
- `filename` — required remote filename
- `subfolder` — optional
- `type` — optional, default `output`
- `path` — required local destination path
- `sha256` — computed downloaded hash
- `content_length` — computed byte count
- `url` — computed view URL

Lifecycle:

- **Create:** download from `GET /view` and write local file; if the remote file is missing, return a hard error instead of retrying
- **Read:** re-check remote existence through HEAD-compatible checks and ensure the managed local file still exists; if the local file is missing, re-download it
- **Update:** re-download when remote coordinates or local destination change
- **Delete:** remove the managed local file only and succeed if the file is already absent

This resource owns the local artifact lifecycle, not the remote output lifecycle.

Callers should derive `filename`/`subfolder`/`type` from workflow outputs and use normal Terraform dependency edges (including `depends_on` where needed) so the download only runs after the generating workflow has completed.

## Shared Implementation Design

Add a shared internal validation package:

- `internal/validation/semantic.go`
- `internal/validation/report.go`

Responsibilities:

- normalize node metadata from `client.NodeInfo`
- validate prompt models against `/object_info`
- adapt workspace models through the existing translation layer
- produce a structured report reusable by data sources and `comfyui_workflow`

Add client helpers:

- multipart upload helper for image/mask endpoints
- binary download helper for `GET /view`
- hash-aware remote file existence / fetch helpers where useful

This slice does **not** add provider-level `/object_info` caching. Validation surfaces may fetch fresh metadata independently so results always reflect current ComfyUI state. Shared caching can be revisited later if it becomes a measurable bottleneck.

## Testing Strategy

### Go tests

Add focused tests for:

- prompt validation success
- missing node types
- missing required inputs
- unexpected inputs
- link type mismatch and bad output-slot references
- native `output_node` detection
- workspace validation through translation
- workflow preflight validation behavior
- upload request construction for image and mask endpoints
- output download/materialization lifecycle
- explicit delete-warning behavior for uploaded resources

### Acceptance verification

Use the existing Go test suite plus focused ComfyUI-backed acceptance checks where practical:

- upload an image fixture and confirm it becomes viewable through `/view`
- materialize one generated output to local disk and verify its hash is stable

## Rollout

1. implement shared semantic validation core and tests
2. add validation data sources
3. wire `comfyui_workflow` preflight validation
4. add client upload/download helpers
5. add uploaded-image and uploaded-mask resources with honest lifecycle semantics
6. add output-artifact resource
7. register provider surfaces and update docs/examples

## Success Criteria

- Terraform can validate prompt/workspace artifacts against live ComfyUI metadata before execution
- `comfyui_workflow` blocks invalid prompts before queueing by default
- Terraform can upload input images/masks and download output files through dedicated resources
- lifecycle semantics stay explicit where ComfyUI lacks delete APIs

If `/object_info` is unreachable, validation data sources and workflow preflight should return hard errors rather than silently downgrading validation quality.
