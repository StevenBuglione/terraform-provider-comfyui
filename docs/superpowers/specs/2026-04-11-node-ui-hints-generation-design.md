# Node UI Hints Generation Design

## Summary

The provider should stop hard-coding frontend sizing rules for ComfyUI nodes in handwritten Go. Instead, `make generate` should extract server-side node specs from the vendored ComfyUI checkout, boot a temporary local ComfyUI frontend, introspect live LiteGraph node instances with Playwright, and write a committed `node_ui_hints.json` artifact that the Go generator turns into provider-consumable metadata.

This keeps the workspace builder aligned with the real ComfyUI frontend sizing contract without forcing us to manually maintain magic numbers.

## Goals

- Make node sizing data-driven and regenerate it from ComfyUI.
- Treat the running frontend as the source of truth for UI sizing behavior.
- Keep `make generate` as the single authoritative regeneration entrypoint.
- Produce committed artifacts and generated Go so ComfyUI UI changes are visible in diffs.
- Preserve a conservative runtime fallback for unknown or missing node hints.

## Non-Goals

- Perfect pixel-for-pixel reproduction of every frontend nuance in the provider.
- Exhaustively encoding every frontend behavior into provider code.
- Replacing existing server-side `node_specs.json` extraction.
- Supporting arbitrary third-party custom nodes outside the vendored ComfyUI baseline.

## Source Of Truth

There are two different contracts:

- `/object_info` is the source of truth for node schema metadata such as inputs, input ordering, primitive widget traits, and output metadata.
- The live frontend runtime is the source of truth for rendered node sizing behavior such as `computeSize`, widget classes, widget options, and widget-specific minimum node sizes.

Static parsing of frontend source is intentionally not the primary contract. It is more brittle against refactors, bundling changes, and runtime registration than inspecting the live node objects the browser actually renders.

## Proposed Architecture

### 1. Extraction pipeline

`make generate` should run a new extraction script before `go run ./cmd/generate`.

That extraction stage should:

1. Ensure the vendored `third_party/ComfyUI` checkout exists.
2. Start a temporary local ComfyUI runtime using the existing workspace-e2e bootstrap path.
3. Use Playwright to open the real frontend.
4. Fetch `/object_info`.
5. Iterate node types from `/object_info`.
6. Programmatically instantiate each node type in the browser and inspect the live node and widget objects.
7. Persist per-node UI sizing hints to `scripts/extract/node_ui_hints.json`.
8. Stop the temporary runtime.

This is fully automated. No manual per-node inspection is part of the workflow.

### 2. Extracted artifact shape

The committed UI hints artifact should be keyed by node type and contain only the sizing facts the provider needs:

- extractor metadata:
  - artifact version
  - extracted timestamp
  - ComfyUI git commit SHA
  - ComfyUI version string if available
- per-node hints:
  - `node_type`
  - `min_width`
  - `min_height`
  - `computed_width`
  - `computed_height`
  - widget hints keyed by input or widget name
- per-widget hints:
  - `widget_type`
  - `has_compute_size`
  - `min_node_width`
  - `min_node_height`
  - optional raw sizing fields when the frontend exposes them

The artifact should stay intentionally small and purpose-built for layout. It should not try to serialize the full frontend node object graph.

### 3. Generator changes

`cmd/generate` should consume both:

- `scripts/extract/node_specs.json`
- `scripts/extract/node_ui_hints.json`

It should emit a generated Go file in the `internal/resources` package containing:

- artifact metadata constants
- a typed map keyed by node class name
- helper accessors for workspace layout code

This file should be generated alongside the existing resource registry generation. It should not live in `internal/resources/generated`, because the `generated` package already imports `resources` and would create a cycle.

### 4. Workspace builder changes

The workspace builder should stop encoding node-specific UI sizing rules in handwritten code.

The sizing path should become:

1. Look up generated node UI hints for the node class.
2. Apply extracted min width and min height when available.
3. Use widget-level extracted hints to raise the node size when required.
4. Fall back to the generic estimator only when a node has no extracted hint coverage.

The row and column spacing logic fixed earlier remains correct and should continue to use actual node sizes plus configured empty gaps.

## Failure Behavior

`make generate` should be strict.

- If the temporary ComfyUI runtime cannot start, generation fails.
- If Playwright cannot launch or inspect the frontend, generation fails.
- If the extractor cannot fetch `/object_info`, generation fails.
- If a node type cannot be instantiated, the extractor should record that failure and continue collecting the rest.
- Generation should fail only when extractor failures cross a defined threshold or when a core set of node types cannot be inspected.

The provider runtime should still keep a generic fallback for missing hints so unknown/custom nodes remain usable, but that fallback is runtime safety, not the happy-path generation contract.

## Reuse Existing Infrastructure

The implementation should reuse the existing browser/runtime machinery where practical:

- `scripts/workspace-e2e/start-comfyui.sh`
- `scripts/workspace-e2e/stop-comfyui.sh`
- Playwright dependency conventions already used under `validation/*/browser`

The extractor should share patterns with the existing Playwright helpers that already load graphs through `window.app`.

## Testing Strategy

Testing should happen at four layers:

### Extractor tests

- validate the new `node_ui_hints.json` structure
- verify artifact metadata is written
- verify key canonical nodes like `CLIPTextEncode` produce non-zero min sizes and widget hints

### Generator tests

- verify `cmd/generate` reads the new artifact
- verify the generated `resources` package metadata is emitted deterministically
- verify missing or malformed UI hints fail generation clearly

### Workspace builder tests

- verify builder sizing uses generated UI hints instead of handwritten widget rules
- keep a generic fallback path test for nodes with no hints
- preserve the spacing regression tests added for `release_gallery`

### Browser regression tests

- keep the `release_gallery` Playwright minimum-gap assertion
- ensure a regenerated UI-hints artifact still produces readable graph spacing in the live frontend

## Operational Impact

Generation becomes heavier because it now boots ComfyUI and launches Playwright. That is acceptable because this is not a normal runtime path; it is the authoritative regeneration path for provider metadata. In exchange, the maintenance burden shifts from hand-curating magic numbers to reviewing generated artifact diffs when upstream UI behavior changes.

## File-Level Changes

- Add extraction script(s) under `scripts/extract/`
- Add committed `scripts/extract/node_ui_hints.json`
- Update `GNUmakefile` `generate` target to run the extractor pipeline before `go run ./cmd/generate`
- Update `cmd/generate` to read UI hints and emit generated resources-package metadata
- Add generated Go output in `internal/resources`
- Refactor `internal/resources/workspace_builder.go` to consume generated hints
- Add tests for extraction, generation, and workspace sizing

## Decision

Use automated runtime frontend introspection as part of `make generate`, with committed extracted UI hints and generated Go metadata. This is the most professional and maintainable way to keep provider layout aligned with real ComfyUI behavior without hand-maintained sizing constants.
