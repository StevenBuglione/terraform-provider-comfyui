# AI Harness Maintainability Design

## Goal

Make this provider a generated-first, machine-readable system that allows an AI coding harness to create, inspect, refactor, validate, translate, and maintain ComfyUI workflows declaratively in Terraform with minimal inference and minimal provider-owned handwritten behavior.

The provider should be able to support the claim:

> An AI harness can use this provider to maintain ComfyUI workflows as Terraform source of truth, derive native ComfyUI artifacts from Terraform, derive Terraform from native ComfyUI artifacts, and prove correctness through provider-owned validation and real-runtime/browser evidence.

## Problem Statement

The provider is already strong at:

- generated node resource coverage
- workflow assembly
- workspace composition
- prompt/workspace translation
- execution-state normalization
- browser-based validation of layout and connectivity

But it does not yet fully satisfy AI-harness maintainability requirements.

Current gaps:

1. There is no first-class provider surface that converts native prompt/workspace artifacts back into canonical Terraform-maintainable source or Terraform-ready IR.
2. Node schema introspection is not fully normalized for machine reasoning; important input metadata is still exposed as raw JSON strings.
3. Validation does not clearly separate fragment validation from executable workflow validation.
4. The release harnesses prove topology and layout, but not deep enough semantic equivalence to support a “perfect maintenance” claim.
5. Some provider behavior is still hand-rolled where it should instead be generated from real ComfyUI server/frontend behavior.

## Non-Goals

This design does not aim to:

- guarantee execution correctness for every possible third-party custom node outside the extracted ComfyUI environment
- replace Terraform itself with a higher-level DSL
- support arbitrary hand-authored HCL styles on import; instead it should emit canonical provider-owned Terraform structure
- remove all provider-owned logic; Terraform composition, state, and lifecycle behavior will still require a narrow handwritten core

## Design Principles

### 1. Generated First

Anything that can be derived from actual ComfyUI behavior should be generated from ComfyUI rather than hand-maintained in provider code.

That includes:

- node schemas from server metadata
- defaults, enums, ranges, input/output types
- frontend UI sizing and widget hints
- compatibility metadata tied to exact ComfyUI revisions

### 2. Provider-Owned Logic Must Be Narrow

Handwritten logic should be limited to what Terraform uniquely needs:

- resource/state wiring
- workflow assembly from Terraform resources
- workspace composition from Terraform resources
- artifact lifecycle operations
- translation between canonical IRs
- validation orchestration and reporting
- runtime/browser harness orchestration

### 3. Machine-Readable Contracts Over Human Convention

An AI harness should not have to scrape examples or infer hidden semantics from docs where a provider data source or generated metadata file could make the contract explicit.

### 4. Canonical IRs

The provider should reason through canonical intermediate representations rather than ad hoc direct conversions.

Required IRs:

- Prompt IR: normalized ComfyUI API prompt representation
- Workspace IR: normalized ComfyUI workspace/subgraph representation
- Terraform IR: canonical provider-owned declarative graph representation suitable for rendering deterministic Terraform

### 5. Proof Before Trust

Every major provider-owned transformation should be provable by:

- static schema assertions
- semantic validation
- translation-fidelity reporting
- end-to-end runtime checks
- browser/canvas verification where UI artifacts are involved

## Target State

The target provider contract consists of five major machine-readable surfaces.

### A. Node Contract

The provider must expose an explicit generated node contract that includes:

- node type
- display name
- description
- category
- output-node flag
- deprecated/experimental flags
- normalized required inputs
- normalized optional inputs
- input defaults, min/max/step, enum values, and widget hints
- output slot definitions with names and types
- frontend UI hints where relevant

This contract should come directly from extracted ComfyUI data and be consumable without reparsing opaque strings.

### B. Artifact Contract

The provider must continue to support:

- prompt JSON import/normalization
- workspace JSON import/normalization
- prompt-to-workspace translation
- workspace-to-prompt translation

But these translation surfaces must expose deterministic, structured fidelity metadata:

- fidelity class
- preserved fields
- synthesized fields
- unsupported fields
- notes

### C. Terraform Synthesis Contract

The provider must add first-class surfaces that convert native artifacts into canonical Terraform-maintainable output.

Required new capabilities:

- prompt -> Terraform IR
- workspace -> Terraform IR
- Terraform IR -> canonical HCL rendering

Potential public surfaces:

- `comfyui_prompt_to_terraform`
- `comfyui_workspace_to_terraform`

or:

- `comfyui_prompt_to_terraform_ir`
- `comfyui_workspace_to_terraform_ir`
- `comfyui_terraform_render`

The specific naming is flexible, but the contract must be provider-owned and machine-readable.

### D. Executability Contract

Validation must distinguish between:

- structurally valid fragments
- executable workflows

This applies to both prompt and workspace validation.

Required validation modes:

- `fragment`
- `workspace_fragment`
- `executable_workflow`
- `executable_workspace`

Executable modes must require output-node presence and any other run-critical invariants. Fragment modes remain permissive for editing/staging flows.

### E. Runtime Proof Contract

The provider must continue to provide real-runtime proof through:

- execution e2e
- workspace browser e2e
- release scenario browser e2e

But these must be extended so they prove semantic correctness, not only topology/counts.

## Architecture

### Generated Inputs

The system will use two generated upstream input classes.

#### 1. Server-derived metadata

Generated from real ComfyUI server metadata:

- node schemas
- defaults, ranges, enums
- output slots
- validation-relevant type information
- ComfyUI version compatibility snapshot

Primary sources:

- `/object_info`
- any upstream-exposed stable API metadata needed to normalize nodes

#### 2. Frontend-derived metadata

Generated from live ComfyUI frontend runtime:

- widget sizing behavior
- node size hints
- min node size
- widget-specific layout hints

Primary sources:

- live browser runtime via Playwright
- real `LiteGraph` node/widget behavior, not static source parsing alone

### Canonical Internal Representations

The provider should standardize around three core IRs.

#### Prompt IR

Represents normalized ComfyUI API prompt graphs.

Responsibilities:

- node IDs and class types
- typed inputs
- connection references
- prompt metadata normalization

#### Workspace IR

Represents normalized workspace/subgraph documents.

Responsibilities:

- nodes, links, groups, widgets
- UI metadata
- definitions/subgraphs
- top-level workspace metadata

#### Terraform IR

Represents canonical Terraform-maintainable workflow structure.

Responsibilities:

- resource inventory
- canonical resource names
- input expressions
- connection expressions
- workflow grouping/layout configuration
- provider/meta resources like `comfyui_workflow` and `comfyui_workspace`
- provenance and fidelity annotations linking each field back to source artifact semantics

### Provider-Owned Layers

Only the following layers should remain handwritten:

- Terraform plugin framework wiring
- canonical assembly from node resources into prompt IR
- canonical workspace composition from workflows into workspace IR
- IR translators
- HCL rendering from Terraform IR
- validation orchestration and reporting
- artifact lifecycle resources
- runtime/browser test harness orchestration

All raw node semantics should be generated, not re-described manually.

## New Public Surfaces

### 1. Structured Node Schema Data Source

Add a richer data source to supersede or complement `comfyui_node_info`.

Possible name:

- `comfyui_node_schema`

Required outputs:

- normalized required inputs list
- normalized optional inputs list
- each input’s type/default/range/options/widget metadata
- normalized outputs list
- frontend UI hints

`comfyui_node_info` can remain as a lightweight backward-compatible surface, but AI harnesses should prefer `comfyui_node_schema`.

### 2. Terraform Synthesis Data Sources

Add:

- `comfyui_prompt_to_terraform`
- `comfyui_workspace_to_terraform`

Each should return:

- canonical Terraform HCL
- canonical Terraform IR JSON
- fidelity class
- preserved/synthesized/unsupported fields
- notes
- inferred dependencies
- inferred workflow/workspace structure

### 3. Terraform IR Render/Parse Utilities

Optional but preferred:

- `comfyui_terraform_ir_render`
- `comfyui_terraform_ir_validate`

This keeps direct artifact translation and rendered-HCL concerns separate.

### 4. Validation Mode Support

Extend:

- `comfyui_prompt_validation`
- `comfyui_workspace_validation`

with a required or defaulted `mode` attribute.

Default recommendation:

- default to `executable_workflow` for prompt validation
- default to `executable_workspace` for workspace validation

Explicit fragment validation remains available when needed.

## Generated-First Extraction Pipeline

### Server Generation

`make generate` should continue to extract node specs from real ComfyUI behavior.

Generated artifacts should include:

- node specs
- normalized schema contract
- provider metadata snapshot

### Frontend Generation

`make generate` should continue to boot a temporary ComfyUI runtime and extract frontend UI hints from the live frontend.

Generated artifacts should include:

- node UI hints
- widget sizing hints
- version/commit metadata

### Determinism Rules

Generated artifacts must remain deterministic across environments when the ComfyUI commit is unchanged.

Version-metadata policy:

- the ComfyUI commit SHA is authoritative
- human-readable version labels may vary by environment
- if the same commit produces different version labels, the committed version label must remain canonical

This avoids meaningless CI drift.

## Canonical Terraform Synthesis

### Objective

Given a prompt or workspace artifact, the provider should emit a deterministic Terraform representation that:

- is canonical
- is easy for an AI harness to edit
- minimizes semantic ambiguity

### Rules

#### Resource naming

Resource names should be deterministic and provider-owned.

Rules should derive from:

- class type
- graph role
- occurrence order

Example patterns:

- `checkpoint_loader_simple.primary`
- `clip_text_encode.positive_main`
- `clip_text_encode.negative_main`

The exact naming scheme should be generated and deterministic, not agent-chosen ad hoc.

#### Connection rendering

Connections should always render as references to exported Terraform outputs where possible, not raw tuple literals.

#### Meta-resource rendering

The provider should render:

- `comfyui_workflow` for execution/export graphs
- `comfyui_workspace` for staged UI-oriented compositions

#### Provenance

The Terraform IR should record where each rendered field came from:

- preserved from source artifact
- synthesized by provider
- unsupported and omitted

That provenance is essential for AI reasoning during refactors.

## Validation Strategy

### Static Validation

Validate:

- schema shape
- generated metadata integrity
- deterministic generation behavior
- Terraform synthesis determinism

### Semantic Validation

Prompt/workspace validation should report:

- structural validity
- executability status
- exact diagnostics
- normalized artifact outputs

### Translation Validation

Each translation path must publish a fidelity report and be testable via golden fixtures.

Required paths:

- prompt -> workspace
- workspace -> prompt
- prompt -> Terraform IR
- workspace -> Terraform IR
- Terraform IR -> HCL

### Browser Validation

Playwright must prove:

- staged workspaces load
- expected workspaces are discoverable
- no broken links
- no invalid geometry
- no overlapping groups
- no containment violations
- node/group/link counts match expectations

### Execution Validation

Execution harnesses must prove:

- executable Terraform-generated workflows queue and complete
- execution metadata is correctly reflected in provider surfaces
- artifacts can be downloaded and verified

## Proof Requirements for the “Perfect for AI Harnesses” Claim

The provider can reasonably claim AI-harness maintainability only when all of these are true.

### 1. Canonical Synthesis

For supported workflows, the provider can synthesize deterministic Terraform from prompt/workspace artifacts.

### 2. Round-Trip Explainability

For every translation, the provider tells the AI exactly what was preserved, synthesized, or unsupported.

### 3. Executability Guarantees

Validation defaults align with runnable workflows, not merely fragments.

### 4. Generated Source of Truth

Node semantics and UI hints are derived from actual ComfyUI behavior, not hand-maintained magic numbers or hand-curated per-node schemas.

### 5. Runtime Proof

Release harnesses prove that canonical scenarios work in a real ComfyUI runtime and browser UI.

## Required Test and Validation Expansion

### A. Terraform Synthesis Golden Corpus

Add a corpus of prompt/workspace fixtures and expected Terraform IR/HCL outputs.

Assertions:

- stable rendering
- stable naming
- stable connection mapping
- explicit fidelity diagnostics

### B. Semantic Round-Trip Corpus

Add cases for:

- sparse slots
- fan-in/fan-out
- groups
- subgraphs
- metadata-heavy nodes
- optional widgets
- multiline text widgets

Assertions should inspect actual field-level equivalence, not only counts.

### C. Executable Validation Modes

Add tests proving:

- fragment mode accepts non-runnable graphs
- executable modes reject graphs missing output nodes
- workspace executable mode fails on non-runnable translated prompts

### D. Release Harness Extension

Add canonical scenarios that use Terraform synthesized from native artifacts, not just handwritten Terraform fixtures.

That proves the AI-maintainability loop end to end:

- native artifact
- provider synthesis to Terraform
- provider assembly/staging
- runtime/browser proof

## Migration Strategy

### Phase 1

Add new structured surfaces without removing current JSON-oriented ones.

### Phase 2

Teach examples and release harnesses to prefer:

- structured schema data
- executable validation modes
- Terraform synthesis outputs

### Phase 3

Relegate older opaque/raw surfaces to escape-hatch roles rather than primary AI-facing contracts.

## Risks

### 1. Overfitting to Current ComfyUI Frontend

Mitigation:

- derive UI hints from live runtime
- keep fallback behavior minimal and explicit
- prove determinism across CI/local environments

### 2. Canonical Terraform Rendering Drift

Mitigation:

- golden tests
- deterministic naming rules
- Terraform IR as the canonical layer

### 3. Translation Fidelity Ambiguity

Mitigation:

- field-level fidelity reporting
- explicit unsupported-field accounting

### 4. Excessive Provider-Owned Logic

Mitigation:

- prefer generated contracts
- confine handwritten logic to orchestration and IR translation
- reject feature additions that duplicate upstream semantics without necessity

## Deliverables

The implementation should produce:

1. Generated structured node schema metadata from real ComfyUI.
2. Generated frontend UI hint metadata from live ComfyUI frontend behavior.
3. A canonical Terraform IR.
4. Prompt/workspace -> Terraform synthesis data sources.
5. Validation modes that distinguish fragments from executable workflows.
6. Golden synthesis and translation corpora.
7. Extended release/runtime/browser harnesses that prove the AI-maintainability loop.

## Success Criteria

This design is successful when:

- an AI harness can inspect node schemas without reparsing opaque JSON strings
- native prompt/workspace artifacts can be converted into deterministic Terraform output by the provider
- the provider explains fidelity and unsupported semantics explicitly
- executable validation fails incomplete workflows by default
- release harnesses prove synthesized Terraform workflows load, connect, and execute correctly
- upgrade burden stays concentrated in generated extraction paths rather than expanding handwritten provider semantics

