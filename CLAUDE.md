# CLAUDE.md — terraform-provider-comfyui

## Project Overview

Terraform provider for [ComfyUI](https://github.com/comfyanonymous/ComfyUI), a node-based Stable Diffusion GUI.
Built with the **Terraform Plugin Framework** (not SDKv2). Language: Go + Python. License: MIT.

Fully implemented with **645 generated node resources** (one per ComfyUI node), **9 hand-written resources**, and **20 data sources**.

## Tech Stack

- Go 1.25+ (provider, code generator)
- Python 3.12+ (node extraction pipeline)
- Terraform Plugin Framework (`terraform-plugin-framework`)
- Terraform Plugin Log (`terraform-plugin-log`)
- GoReleaser for builds/releases
- GitHub Actions for CI/CD (test + release workflows)
- Dependabot for dependency updates

## Key Directories

```
.
├── CLAUDE.md                         # This file — project instructions for AI agents
├── LICENSE                           # MIT License
├── main.go                           # Provider entry point
├── generate.go                       # go:generate directive → cmd/generate
├── GNUmakefile                       # Build, test, generate, lint targets
├── cmd/generate/                     # Go code generator (node_specs.json → Go files)
│   ├── main.go                       # Generator entry point
│   ├── templates.go                  # Go template definitions
│   ├── types.go                      # Shared types and helper functions
│   └── generate_test.go              # Generator unit tests
├── scripts/extract/                  # Python extraction pipeline
│   ├── extract_v1_nodes.py           # V1-pattern node extractor (AST parsing)
│   ├── extract_v3_nodes.py           # V3-pattern node extractor (ComfyNode subclass)
│   ├── merge_specs.py                # Merges V1+V3 extracts into node_specs.json
│   ├── node_specs.json               # Generated: all 645 node specifications
│   ├── node_spec_schema.json         # JSON schema for node specs
│   └── test_extractors.py            # 16 Python tests (pytest)
├── scripts/update-comfyui.sh         # Re-pin ComfyUI submodule to a new tag
├── internal/provider/                # Provider implementation (1 file)
│   └── provider.go                   # ComfyUIProvider: schema, configure, resource/DS registration
├── internal/client/                  # HTTP client for ComfyUI REST API
│   ├── client.go                     # Client implementation (all API methods)
│   ├── client_test.go                # Client unit tests (httptest-based)
│   └── types.go                      # API response types
├── internal/resources/               # Hand-written resources and orchestration helpers
│   ├── workflow_resource.go          # comfyui_workflow: queue & execute workflows
│   ├── workflow_collection.go        # comfyui_workflow_collection: group workflows into manifests
│   ├── workspace_resource.go         # comfyui_workspace: compose editor-oriented workspace exports
│   ├── prompt_artifact_resource.go   # comfyui_prompt_artifact: write prompt JSON to disk
│   ├── workspace_artifact_resource.go # comfyui_workspace_artifact: write workspace JSON to disk
│   ├── subgraph_resource.go          # comfyui_subgraph: manage local subgraph/workspace JSON
│   ├── uploaded_image_resource.go    # comfyui_uploaded_image: upload local images
│   ├── uploaded_mask_resource.go     # comfyui_uploaded_mask: upload typed masks
│   └── output_artifact_resource.go   # comfyui_output_artifact: download ComfyUI output files
├── internal/resources/generated/     # 645 generated node resources + registry
│   ├── registry.go                   # AllResources() — lists all generated constructors
│   └── resource_*.go                 # One file per ComfyUI node (e.g., resource_ksampler.go)
├── internal/datasources/             # 20 data sources
│   ├── system_stats.go               # comfyui_system_stats
│   ├── queue.go                      # comfyui_queue
│   ├── node_info.go                  # comfyui_node_info
│   ├── node_schema.go                # comfyui_node_schema
│   ├── inventory.go                  # comfyui_inventory
│   ├── workflow_history.go           # comfyui_workflow_history
│   ├── output.go                     # comfyui_output
│   ├── prompt_json.go                # comfyui_prompt_json
│   ├── prompt_validation.go          # comfyui_prompt_validation
│   ├── prompt_to_workspace.go        # comfyui_prompt_to_workspace
│   ├── prompt_to_terraform.go        # comfyui_prompt_to_terraform
│   ├── workspace_json.go             # comfyui_workspace_json
│   ├── workspace_validation.go       # comfyui_workspace_validation
│   ├── workspace_to_prompt.go        # comfyui_workspace_to_prompt
│   ├── workspace_to_terraform.go     # comfyui_workspace_to_terraform
│   ├── subgraph_catalog.go           # comfyui_subgraph_catalog
│   ├── subgraph_definition.go        # comfyui_subgraph_definition
│   ├── provider_info.go              # comfyui_provider_info
│   ├── job.go                        # comfyui_job
│   └── jobs.go                       # comfyui_jobs
├── third_party/ComfyUI/              # ComfyUI source (git submodule, pinned to tag)
├── doc/terraform/provider/research/  # 27 comprehensive research docs (00–26)
├── .claude/skills/                   # Claude Code skills for this project
│   ├── terraform-provider-research/  # Progressive research disclosure skill
│   └── terraform-provider-dev/       # Hands-on development guidance skill
├── .github/workflows/                # CI/CD
│   ├── test.yml                      # Build + test + verify generated code
│   └── release.yml                   # GoReleaser + GPG signing on tag push
├── .goreleaser.yml                   # GoReleaser configuration
└── .golangci.yml                     # Linter configuration
```

## ComfyUI Submodule

The full ComfyUI source is included as a **git submodule** at `third_party/ComfyUI/` for AI
agent code inspection. This is read-only reference — never modify ComfyUI source directly.

```bash
# After cloning this repo, initialize the submodule:
git submodule update --init

# Update to a new ComfyUI version:
./scripts/update-comfyui.sh v0.19.0

# Check current pinned version:
cd third_party/ComfyUI && git describe --tags
```

**Current version:** v0.18.5

### Key ComfyUI paths for agent inspection

| Path | What's There |
|------|-------------|
| `third_party/ComfyUI/nodes.py` | Core built-in nodes |
| `third_party/ComfyUI/comfy_extras/` | Additional node implementations |
| `third_party/ComfyUI/comfy/` | Core engine (model loading, sampling, etc.) |
| `third_party/ComfyUI/server.py` | REST API server implementation |
| `third_party/ComfyUI/execution.py` | Workflow execution engine |
| `third_party/ComfyUI/folder_paths.py` | Model/output path management |

## Research Documentation

Extensive research lives in `doc/terraform/provider/research/`. **27 files** covering every aspect of Terraform provider development. Use the `terraform-provider-research` skill for guided access, or browse directly:

| # | File | Topic |
|---|------|-------|
| 00 | `00-overview-and-architecture.md` | Plugin system, gRPC, Framework vs SDKv2 |
| 01 | `01-plugin-framework-fundamentals.md` | Interfaces, type system, null/unknown |
| 02 | `02-project-structure-and-scaffolding.md` | Scaffolding, directory layout, go.mod |
| 03 | `03-provider-implementation.md` | Provider interface, main.go, Configure |
| 04 | `04-resource-implementation.md` | Full CRUD lifecycle |
| 05 | `05-data-source-implementation.md` | DataSource interface, Read |
| 06 | `06-schema-design-patterns.md` | Attribute types, nested attributes |
| 07 | `07-plan-modifiers-and-validators.md` | Plan modifiers, validators |
| 08 | `08-state-management-and-import.md` | State, ImportState, drift |
| 09 | `09-error-handling-and-diagnostics.md` | Diagnostics, tflog |
| 10 | `10-acceptance-testing.md` | terraform-plugin-testing |
| 11 | `11-unit-testing.md` | Unit tests, mocking |
| 12 | `12-debugging-and-development-workflow.md` | TF_LOG, Delve, dev_overrides |
| 13 | `13-documentation-generation.md` | tfplugindocs |
| 14 | `14-naming-conventions-and-style.md` | Naming, Go/Terraform style |
| 15 | `15-versioning-and-changelog.md` | SemVer, changelogs |
| 16 | `16-ci-cd-and-github-actions.md` | CI workflows |
| 17 | `17-goreleaser-configuration.md` | GoReleaser config |
| 18 | `18-registry-publishing.md` | Registry publishing |
| 19 | `19-provider-design-principles.md` | HashiCorp design principles |
| 20 | `20-advanced-patterns.md` | Ephemeral resources, write-only attrs |
| 21 | `21-reference-provider-aws.md` | AWS provider architecture |
| 22 | `22-reference-provider-azurerm.md` | AzureRM architecture |
| 23 | `23-makefile-and-dev-commands.md` | Makefile, dev commands |
| 24 | `24-comfyui-provider-mapping.md` | ComfyUI API → Terraform mapping |
| 25 | `25-provider-functions.md` | Provider functions (TF 1.8+) |
| 26 | `26-partner-nodes-and-api-integrations.md` | Partner nodes, API providers, categories |

## Architecture Decisions

1. **Plugin Framework only** — No SDKv2. All resources, data sources, and the provider use `terraform-plugin-framework`.
2. **Single API focus** — Provider wraps the ComfyUI REST API exclusively.
3. **Virtual node resources** — The 645 generated node resources are virtual/plan-only (no API calls in CRUD). They represent ComfyUI nodes in Terraform state for workflow composition; the actual execution happens through `comfyui_workflow`. Of these, **180 are partner/API nodes** that call third-party AI services (see Partner Nodes section below).
4. **Code generation pipeline** — Python AST extractors parse ComfyUI source → `node_specs.json` → Go generator produces one resource file per node + registry. This allows automated updates when ComfyUI adds/changes nodes.
5. **Resources**: `comfyui_workflow`, `comfyui_workflow_collection`, `comfyui_workspace`, `comfyui_prompt_artifact`, `comfyui_workspace_artifact`, `comfyui_subgraph`, `comfyui_uploaded_image`, `comfyui_uploaded_mask`, `comfyui_output_artifact`, plus 645 generated node resources.
6. **Data sources** (20): runtime inspection (`system_stats`, `queue`, `job`, `jobs`, `workflow_history`, `output`), schema and inventory (`node_info`, `node_schema`, `inventory`, `provider_info`, `subgraph_catalog`, `subgraph_definition`), and prompt/workspace translation and validation surfaces.
7. **Multi-modal capabilities** — Through partner nodes, the provider supports image generation/editing, video generation, audio synthesis/processing, text/LLM chat, and 3D model generation — all orchestrated via `comfyui_workflow`.
8. **Version alignment** — The provider version is tightly coupled to the ComfyUI version it was generated from. The ComfyUI version (`v0.18.5`) is embedded in `generated.ComfyUIVersion`, exposed via the `comfyui_provider_info` data source, and logged at provider startup. The `node_specs.json` records the exact ComfyUI version and extraction timestamp. See "Versioning" section below.

## Commands

```bash
# Build the provider binary
make build                  # or: go build -o terraform-provider-comfyui

# Run Go unit tests (cmd/generate + internal/client)
make test                   # or: go test ./... -v -timeout 120s

# Run acceptance tests (requires running ComfyUI instance)
make testacc                # or: TF_ACC=1 go test ./... -v -timeout 120m

# Run Python extractor tests (16 tests)
python3 -m pytest scripts/extract/test_extractors.py -v

# Regenerate node resources from node_specs.json
make generate               # or: go run ./cmd/generate
make docs                   # or: go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name comfyui
make docs-validate          # validate generated provider docs
make docs-check             # fail on docs drift or untracked generated docs files

# Install locally for development
make install

# Lint / format / vet
make lint                   # golangci-lint run ./...
make fmt                    # gofmt -s -w .
make fmt-check              # fail if tracked Go files are unformatted
make tidy                   # go mod tidy
make vet                    # go vet ./...
make verify                 # fmt-check + generate + docs-check + vet + lint + test
make hooks-install          # install pinned lefthook hooks locally

# Clean build artifacts
make clean
```

## Code Generation Pipeline

The provider's 645 node resources are generated automatically from ComfyUI source code:

```
ComfyUI source (third_party/ComfyUI/)
  │
  ├─→ extract_v1_nodes.py   (AST-parses nodes.py + comfy_extras/ for V1-pattern nodes)
  ├─→ extract_v3_nodes.py   (AST-parses V3-pattern nodes using io.ComfyNode/define_schema)
  │
  └─→ merge_specs.py        (deduplicates, validates, writes node_specs.json)
        │
        └─→ cmd/generate/main.go  (reads node_specs.json, applies Go templates)
              │
              ├─→ internal/resources/generated/resource_*.go  (645 resource files)
              └─→ internal/resources/generated/registry.go    (AllResources() function)
```

**Triggers**:
- `make generate` runs only the Go resource generator (`cmd/generate`)
- `make docs` runs `tfplugindocs`
- `go generate ./...` runs both via `generate.go`

## How to Add/Update Nodes

When a new ComfyUI version adds or changes nodes:

```bash
# 1. Update ComfyUI submodule to new tag
./scripts/update-comfyui.sh v0.19.0

# 2. Run Python extractors to regenerate node_specs.json
python3 scripts/extract/extract_v1_nodes.py third_party/ComfyUI > v1.json
python3 scripts/extract/extract_v3_nodes.py third_party/ComfyUI > v3.json
python3 scripts/extract/merge_specs.py v1.json v3.json > scripts/extract/node_specs.json

# 3. Regenerate Go resource files
make generate

# 4. Build and test
make build && make test
python3 -m pytest scripts/extract/test_extractors.py -v
```

## Conventions

- Follow HashiCorp naming: resources as `comfyui_<noun>`, data sources as `comfyui_<noun>`
- Attributes: snake_case, Required/Optional/Computed per schema design patterns (doc 06)
- Errors: Use diagnostics system, never panic (doc 09)
- Testing: Unit tests for client and generator (Go), extraction pipeline tests (Python/pytest)
- Generated code: Never edit files in `internal/resources/generated/` — regenerate instead
- Documentation: Generated via tfplugindocs from schema + templates (doc 13)
- Commits: Conventional Commits format
- Versioning: See "Versioning" section below

## Versioning

The provider version is **tightly coupled to the ComfyUI version** it was generated from.

### Versioning Policy

Provider versions follow the **ComfyUI compatibility line** model:

- Provider `0.18.x` is the compatibility line for ComfyUI `v0.18.5`
- The first release in this line is `v0.18.5`
- Later provider-only fixes (bug fixes, documentation improvements, non-schema changes) increment the patch version: `v0.18.6`, `v0.18.7`, etc.
- The exact upstream pin remains authoritative in `generated.ComfyUIVersion` and the `comfyui_provider_info` data source
- Users should constrain the provider with `~> 0.18` for this line
- If the pinned upstream ComfyUI version changes materially (e.g., to `v0.19.0`), a new provider line (`0.19.x`) is started rather than silently continuing `0.18.x`

This ensures users can trust that provider `0.18.x` releases maintain compatibility with their ComfyUI `v0.18.5` workflows while receiving provider-level improvements.

### Version Sources

| Source | Location | Example |
|--------|----------|---------|
| ComfyUI version | `node_specs.json` → `comfyui_version` | `v0.18.5` |
| Generated constant | `internal/resources/generated/registry.go` → `ComfyUIVersion` | `v0.18.5` |
| Provider version | `main.go` → `version` (set by GoReleaser ldflags) | `0.18.5` |
| Node count | `internal/resources/generated/registry.go` → `NodeCount` | `645` |
| Extraction timestamp | `internal/resources/generated/registry.go` → `ExtractedAt` | ISO 8601 |

### How Version Flows

1. `scripts/update-comfyui.sh v0.19.0` → pins submodule to tag
2. Python extractors → `node_specs.json` with `comfyui_version` field
3. `go run ./cmd/generate` → reads `comfyui_version` from JSON, embeds as `generated.ComfyUIVersion` constant
4. Provider schema description includes ComfyUI version and node count
5. `comfyui_provider_info` data source exposes `comfyui_version`, `provider_version`, `node_count`, `extracted_at`
6. Provider logs ComfyUI version at startup via `tflog.Info`

### Querying Version at Runtime

```hcl
data "comfyui_provider_info" "current" {}

output "compatibility" {
  value = "Provider ${data.comfyui_provider_info.current.provider_version} for ComfyUI ${data.comfyui_provider_info.current.comfyui_version} (${data.comfyui_provider_info.current.node_count} nodes)"
}
```

## Test Suite

| Area | Framework | Count | Command |
|------|-----------|-------|---------|
| Code generator (`cmd/generate/`) | Go `testing` | 9 tests | `go test ./cmd/generate/ -v` |
| HTTP client (`internal/client/`) | Go `testing` + `httptest` | 15 tests | `go test ./internal/client/ -v` |
| Data sources (`internal/datasources/`) | Go `testing` | 2 tests | `go test ./internal/datasources/ -v` |
| Python extractors (`scripts/extract/`) | pytest | 16 tests | `python3 -m pytest scripts/extract/test_extractors.py -v` |

## ComfyUI API Reference

Base URL: `http://<host>:<port>` (default port 8188)

Key endpoints:
- `POST /prompt` — Queue a workflow
- `GET /history/{id}` — Get execution history
- `GET /system_stats` — System information
- `GET /object_info` — Available nodes
- `GET /queue` — Queue status
- `POST /upload/image` — Upload image
- `WebSocket /ws` — Real-time updates

## Skills Available

- **terraform-provider-research** — Progressive disclosure of the 26 research docs. Invoke when you need to understand any Terraform provider concept.
- **terraform-provider-dev** — Hands-on development guidance for implementing this specific provider. Invoke when writing code.

## Partner Nodes (API Integrations)

Of the 645 generated node resources, **180 are partner/API nodes** that integrate with
third-party AI services. These nodes call external APIs (not the local ComfyUI server)
and are organized into 5 categories:

| Category | Nodes | Key Providers |
|----------|-------|--------------|
| **Video** | 77 | Kling (22), Vidu (13), Wan (8), ByteDance (4), Grok (4), MiniMax (4), PixVerse (4), Luma, Moonvalley, Runway, Sora |
| **Image** | 61 | Recraft (15), BFL/Flux (5), Stability AI (5), Magnific (5), Gemini (3), Ideogram (3), Kling (3), Luma (3), OpenAI (3) |
| **3D** | 26 | Tripo (8), Meshy (7), Tencent/Hunyuan3D (6), Rodin (5) |
| **Audio** | 11 | ElevenLabs (8), Stability AI (3) |
| **Text** | 5 | OpenAI (3), Gemini (2) |

**Key points:**
- All 180 partner nodes already have Terraform resources (generated alongside core nodes)
- Partner nodes require API keys from their respective providers (e.g., `KLING_API_KEY`, `OPENAI_API_KEY`)
- They follow the same virtual/plan-only pattern — actual execution happens through `comfyui_workflow`
- Distinguished by `category` prefix `api node/` in `node_specs.json`
- Full details: `doc/terraform/provider/research/26-partner-nodes-and-api-integrations.md`

## Safety

- Never commit secrets or API keys
- Use environment variables for ComfyUI connection details (`COMFYUI_HOST`, `COMFYUI_PORT`)
- Do not modify `.claude/` hooks or agents configuration
