# Contributing

This guide covers the practical contributor workflow for this repo.

## Local Setup

Core prerequisites:

- Go `1.25+`
- Python `3.12+`
- Terraform CLI
- Node.js and npm for the Playwright-based browser lanes
- the pinned ComfyUI submodule checked out under `third_party/ComfyUI`

Useful bootstrap commands:

```bash
git submodule update --init
make hooks-install
make workspace-e2e-browser-install
make release-e2e-browser-install
```

## Day-to-Day Commands

The main commands are:

```bash
make generate
make docs
make lint
make test
make vet
make synthesis-e2e
make inventory-plan-e2e
make execution-e2e
make workspace-e2e
make release-e2e
```

Use the smallest verification set that fits the change, then widen it before merging.

## Generated vs Hand-Rolled Code

The repo is intentionally split between generated surface and provider-owned orchestration.

Generated:

- `internal/resources/generated/`
- generated node/schema metadata derived from `scripts/extract/node_specs.json`
- frontend UI-hints and related generated outputs

Hand-rolled:

- provider wiring
- client logic
- workflow, workspace, and artifact resources
- data sources not generated from node extraction
- validation semantics
- prompt/workspace/Terraform synthesis
- runtime and browser harnesses

Do not hand-edit generated files. Regenerate them instead.

## Generation Workflow

`make generate` is the canonical regeneration entrypoint.

It covers:

- frontend UI-hints extraction
- generated node resources
- generated node schema metadata

After generation, run:

```bash
make docs
```

if the schema or generated docs may have changed.

## Documentation Expectations

Narrative docs live under `docs/` and should be kept aligned with the current provider behavior.

Generated reference docs live under:

- `docs/resources/`
- `docs/data-sources/`

When changing product behavior, update both:

- the relevant narrative guides
- generated docs if schema or docstrings changed

## Verification Expectations

Typical guidance by change type:

- extraction or generation changes
  - `make generate`
  - `make docs`
  - `go test ./... -timeout 120s`
- synthesis changes
  - `make synthesis-e2e`
- dynamic inventory validation changes
  - `make inventory-plan-e2e`
- execution-state or artifact changes
  - `make execution-e2e`
- workspace layout or staging changes
  - `make workspace-e2e`
  - `make release-e2e`

Before merge, the broad confidence set is usually:

```bash
make generate
make docs
go test ./... -timeout 120s
make lint
make vet
make synthesis-e2e
make inventory-plan-e2e
```

Add the browser/runtime lanes when the change touches layout, translation, staging, or execution behavior.

## Updating the Pinned ComfyUI Version

The repo is version-pinned to a specific ComfyUI revision.

When that changes:

1. update the pinned ComfyUI source
2. run `make generate`
3. run `make docs`
4. run the relevant validation matrix
5. inspect generated diffs carefully

The most likely breakpoints are in the hand-rolled orchestration layer, not the bulk generated wrappers.

## Repo Conventions

- prefer generated contracts over handwritten schema duplication
- avoid editing `internal/resources/generated/` directly
- keep documentation aligned with the current provider state
- treat a passing `terraform plan` for supported built-in dynamic inventories as an important safety guarantee

For deeper architecture context, see [Generation Architecture](./generation-architecture.md) and [AI Maintainability](./ai-maintainability.md).
