# ComfyUI JSON Round-Trip Design

## Summary

Make Terraform the primary AI-facing control plane for ComfyUI JSON artifacts by adding a translation-first architecture that can:

1. import native ComfyUI prompt JSON
2. import native ComfyUI workspace/subgraph JSON
3. translate between those artifact types with explicit fidelity reporting
4. preserve native execution metadata such as `client_id`, `extra_data`, `partial_execution_targets`, and `extra_pnginfo`
5. manage persisted prompt/workspace artifacts as first-class Terraform-owned outputs

This design keeps two workspace surfaces intentionally separate:

- `comfyui_workspace` remains the provider-generated readable export surface
- a later `comfyui_subgraph` surface handles faithful workspace/subgraph round-tripping

## Problem Statement

The provider is already strong at two related but different tasks:

- authoring and executing ComfyUI API prompt JSON through `comfyui_workflow`
- generating readable synthetic workspace exports through `comfyui_workspace`

But it still lacks the surfaces required for real round-trip artifact management. Today the provider does not have:

- native internal models for both prompt JSON and workspace JSON
- import-oriented Terraform surfaces for existing ComfyUI artifacts
- explicit translation and lossiness reporting
- native request wrapper fidelity for the full `/prompt` payload
- a managed lifecycle for persisted prompt/workspace artifacts

That gap matters because upstream ComfyUI clearly treats these as distinct native artifacts:

- `POST /prompt` accepts `prompt`, `prompt_id`, `client_id`, `extra_data`, and `partial_execution_targets`
- execution injects hidden `PROMPT` and `EXTRA_PNGINFO` values from `extra_data`
- `app/subgraph_manager.py` loads real editor subgraph JSON and serves it from `/global_subgraphs`
- job/history metadata uses `extra_data.extra_pnginfo.workflow.id`

So the provider currently reconstructs some ComfyUI artifacts, but it does not yet manage them with believable round-trip fidelity.

## Goals

1. Make Terraform the structured interface an AI harness uses for ComfyUI JSON operations.
2. Model prompt JSON and workspace/subgraph JSON as first-class provider concepts instead of opaque strings.
3. Provide import and translation surfaces that are explicit about preserved versus synthesized fields.
4. Preserve native execution metadata whenever the provider can do so safely.
5. Add Terraform-managed prompt/workspace artifact resources for persisted outputs.
6. Keep the current readable workspace export path while adding a separate editor-native fidelity path.

## Non-Goals

- Do not replace the existing `comfyui_workspace` readable-export behavior with editor-native fidelity semantics.
- Do not absorb file upload/download lifecycle into phase 1.
- Do not regenerate all generated node resources in phase 1.
- Do not solve every semantic validation gap in the first slice.
- Do not reopen the completed workspace readability/layout work except where artifact-boundary behavior requires it.

## Design Principles

### 1. Terraform is the AI harness control plane

The AI harness should interact with ComfyUI through Terraform surfaces, not through ad hoc sidecar scripts or raw JSON editing. That means the provider should expose typed inputs, validations, diagnostics, and computed outputs wherever practical.

### 2. Respect Terraform semantics

The provider still needs to use the right Terraform surface for the right job:

- **provider functions** for pure deterministic transformations
- **data sources** for read-only imports, inspection, and remote lookups
- **resources** for Terraform-owned lifecycle boundaries and persisted artifacts

### 3. Translation first

Every user-facing surface in this design should build on shared internal models and shared translation logic. The provider should not duplicate parsing and serialization rules across resources, data sources, and functions.

### 4. Fidelity must be explicit

If a translation cannot preserve a field exactly, the provider must report that clearly. Silent dropping of editor or execution metadata is not acceptable for round-trip management.

## Phase-1 Surface Inventory

Phase 1 commits to the following concrete user-facing surfaces.

| Terraform type | Name | Purpose | Planned file |
| --- | --- | --- | --- |
| Data source | `comfyui_prompt_json` | Import and normalize native ComfyUI prompt JSON from a file or raw JSON string | `internal/datasources/prompt_json.go` |
| Data source | `comfyui_workspace_json` | Import and normalize native workspace/subgraph JSON from a file or raw JSON string | `internal/datasources/workspace_json.go` |
| Data source | `comfyui_prompt_to_workspace` | Translate prompt JSON into workspace/subgraph JSON plus a translation report | `internal/datasources/prompt_to_workspace.go` |
| Data source | `comfyui_workspace_to_prompt` | Translate workspace/subgraph JSON into prompt JSON plus a translation report | `internal/datasources/workspace_to_prompt.go` |
| Resource | `comfyui_prompt_artifact` | Persist Terraform-owned prompt JSON to disk | `internal/resources/prompt_artifact_resource.go` |
| Resource | `comfyui_workspace_artifact` | Persist Terraform-owned workspace/subgraph JSON to disk | `internal/resources/workspace_artifact_resource.go` |
| Resource extension | `comfyui_workflow` | Extend execution requests with native `/prompt` wrapper fidelity | `internal/resources/workflow_resource.go` |

Phase 1 deliberately does **not** add provider functions. Translation remains data-source based in the first slice so the provider can return richer report objects without introducing a second translation interface at the same time. Provider functions may be added later for pure local transformations once the shared translation contract has stabilized.

## Native Artifact Model

Phase 1 introduces three internal domains:

1. **Prompt model**
   - models ComfyUI API prompt JSON
   - includes per-node class type, inputs, optional `_meta`, and request wrapper metadata
2. **Workspace model**
   - models editor/workspace/subgraph JSON
   - includes nodes, links, groups, widget state, definitions, top-level metadata, and editor-specific counters/config
3. **Translation report**
   - records preserved fields
   - records synthesized fields
   - records lossy or unsupported fields

These shared models become the contract behind all later import, translation, execution, and managed-artifact surfaces.

The shared implementation home for phase 1 is:

- `internal/artifacts/prompt.go`
- `internal/artifacts/workspace.go`
- `internal/artifacts/translation.go`

Those files own parsing, serialization, and translation-report types so resources and data sources stay thin.

## User-Facing Surface Design

### 1. Read-only import and inspection

Add import-oriented data sources for native artifacts:

- `data.comfyui_prompt_json`
- `data.comfyui_workspace_json`

These surfaces should let users start from an existing native artifact without forcing immediate lifecycle ownership.

The optional metadata recovery data source is **deferred** from phase 1. It is useful, but it is not required to establish the prompt/workspace translation boundary.

### 2. Pure translation surfaces

Add translation data sources that expose deterministic conversion without creating Terraform-managed state:

- `data.comfyui_prompt_to_workspace`
- `data.comfyui_workspace_to_prompt`

Phase 1 chooses data sources instead of provider functions because the translation outputs need a rich structured report object and may need file-oriented import ergonomics. This keeps one clear interface per translation direction during the first implementation slice.

Both directions expose:

- translated artifact JSON
- normalized source summary
- translation report

#### Translation report schema

Phase 1 uses the following translation report shape across translation surfaces:

- `fidelity` — one of `lossless`, `lossy`, or `synthetic`
- `preserved_fields` — list of field paths preserved exactly
- `synthesized_fields` — list of field paths the provider generated
- `unsupported_fields` — list of field paths dropped or not representable
- `notes` — list of human-readable translation notes

This report must be available as structured Terraform state, not only as diagnostics.

### 3. Managed lifecycle surfaces

Phase 1 should prioritize Terraform-owned persisted artifacts:

- `comfyui_prompt_artifact`
- `comfyui_workspace_artifact`

These resources own file materialization and persistent state for generated or translated artifacts. They are the first managed lifecycle because the user explicitly wants Terraform to be the harness-facing system of record for ComfyUI artifact management.

#### Artifact resource lifecycle semantics

Both artifact resources share the same lifecycle rules:

- `path` is required and is the lifecycle anchor
- `content_json` is required and is the exact JSON to persist
- `id` is the absolute managed path
- `sha256` is computed from the persisted content
- `Create` creates parent directories as needed and writes `content_json` to `path`, replacing any existing file at that path
- `Read` verifies the file still exists and removes the resource from state if it does not
- `Update` rewrites the file and cleans up the previous path if `path` changes
- `Delete` removes the managed file
- `ImportState` uses the file path as the import ID and reads file contents into state

The phase-1 artifact resources are file-materialization resources, not remote ComfyUI API objects.

### 4. Execution fidelity on `comfyui_workflow`

Extend `comfyui_workflow` instead of replacing it.

Phase 1 should add support for the native `/prompt` wrapper fields:

- `prompt_id`
- `client_id`
- `extra_data`
- `partial_execution_targets`

The provider should preserve `extra_pnginfo` automatically when it can do so safely and should surface what was preserved versus synthesized.

#### `extra_pnginfo` preservation rule

Phase 1 defines "safe" preservation as follows:

1. if the user explicitly provides `extra_data`, the provider preserves it unchanged except for adding missing provider-derived metadata fields that do not conflict with explicit user values
2. if execution originates from a workspace/subgraph artifact or a translation that has native workspace metadata, and the user did not already provide `extra_data.extra_pnginfo.workflow`, the provider populates `extra_data.extra_pnginfo.workflow` from that workspace metadata
3. if the provider knows the exact prompt being submitted and the user did not already provide `extra_data.extra_pnginfo.prompt`, the provider populates it with the exact prompt payload
4. if no native workspace metadata is available, the provider does **not** invent a synthetic workflow object; it only preserves explicit user data and provider-known prompt data

This rule keeps preservation automatic where the provider has trustworthy inputs, while avoiding false fidelity claims.

## Workspace Surface Split

This separation is intentional and must remain explicit:

### `comfyui_workspace`

Keep `comfyui_workspace` as the provider-generated readable export path:

- optimized for authoring, inspection, and browser readability
- may synthesize layout and presentation details
- does not promise editor-native round-trip fidelity

### New editor-native workspace surface

Add a later `comfyui_subgraph` surface for faithful workspace/subgraph management:

- preserves authored positions, groups, reroutes, widget state, and top-level metadata where possible
- handles definitions/subgraphs explicitly
- can participate in import and translation workflows without pretending to be the readability-oriented export

Phase 1 does **not** implement `comfyui_subgraph`; it only establishes the internal models and import/translation contracts that make the later resource viable.

This avoids overloading one resource with two incompatible contracts.

## Provider Behavior Design

### 1. Shared parsing and serialization

Create shared helpers for:

- parsing prompt JSON into the internal prompt model
- parsing workspace/subgraph JSON into the internal workspace model
- serializing each model back to native JSON
- converting between the two models

These helpers should live in focused internal packages or files so later resources do not repeat logic.

### 2. Translation contract

Every translation should distinguish:

- **preserved**
- **synthesized**
- **unsupported**

Examples:

- prompt JSON does not natively preserve editor positions or group styling
- workspace JSON may contain widget serialization or definitions that do not map cleanly into a plain execution prompt
- provider-generated readable workspaces may intentionally synthesize layout data

### 3. Automatic execution metadata preservation

When the provider has enough information to carry native ComfyUI execution metadata safely, it should do so by default. This includes `extra_pnginfo` preservation when translating or executing artifacts that contain recoverable workflow metadata.

The provider must not silently invent high-confidence metadata it does not actually know. Synthesized metadata should be reported as synthesized.

## Validation Strategy

### Phase-1 unit tests

Add failing tests first for:

- prompt JSON import parsing
- workspace JSON import parsing
- prompt -> workspace translation fidelity reporting
- workspace -> prompt translation fidelity reporting
- `comfyui_workflow` request wrapper serialization for `client_id`, `extra_data`, and `partial_execution_targets`
- automatic `extra_pnginfo` preservation behavior
- prompt/workspace artifact resource file materialization

### Phase-1 integration tests

Add focused tests that verify:

- translated prompt payloads are accepted by ComfyUI
- translated or imported workspace artifacts can be surfaced through the existing workspace/browser harness where appropriate
- preserved execution metadata survives through history/output surfaces when ComfyUI returns it

Phase-1 fixtures should include:

- one native prompt JSON fixture already accepted by ComfyUI
- one native workspace/subgraph JSON fixture from upstream ComfyUI blueprints
- one translation case that necessarily becomes lossy so the report contract is exercised
- one execution case that proves `extra_pnginfo.workflow.id` survives through history metadata

### Evidence requirement

Phase 1 is not primarily a layout feature, but the browser harness remains useful for validating the editor-native workspace boundary. If a translation claims to produce workspace/subgraph JSON, at least one real ComfyUI browser validation path should prove it loads as expected.

## Phase Structure

### Phase 1

1. shared prompt/workspace/translation models
2. prompt/workspace import data sources
3. prompt/workspace translation data sources
4. managed prompt/workspace artifact resources
5. `comfyui_workflow` request wrapper fidelity and safe metadata preservation

### Later phases

1. semantic validation against native metadata
2. input/output file lifecycle resources
3. structured execution outputs and interruption controls
4. generated node-schema fidelity improvements
5. `comfyui_subgraph` editor-native surface and import polish

## Risks and Mitigations

### Risk: too many overlapping surfaces

Mitigation:

- keep pure translation separate from managed lifecycle
- keep readable export separate from editor-native fidelity
- use shared internal models so each surface is a thin wrapper over a common contract

### Risk: silent metadata loss

Mitigation:

- require explicit translation reporting
- add tests for preserved versus synthesized fields
- surface lossy conversions in diagnostics or computed report outputs

### Risk: phase 1 expands into the full backlog

Mitigation:

- limit managed lifecycle in phase 1 to prompt/workspace artifacts
- defer file upload/download lifecycle and broader validation work
- extend `comfyui_workflow` only for request fidelity, not for every later execution feature

## Recommended Rollout

1. add the written design/spec
2. implement shared internal models and translation-report types
3. add failing tests for import/translation and request-wrapper fidelity
4. implement prompt/workspace import surfaces
5. implement managed prompt/workspace artifact resources
6. extend `comfyui_workflow` for native wrapper fidelity and metadata preservation
7. register the new resources and data sources in `internal/provider/provider.go`
8. validate with Go tests plus the existing ComfyUI browser/runtime harness where applicable

## Planning Readiness

This design is ready for an implementation plan centered on the approved phase-1 slice:

- internal models
- import/translation surfaces
- managed prompt/workspace artifact resources
- `comfyui_workflow` request wrapper fidelity
