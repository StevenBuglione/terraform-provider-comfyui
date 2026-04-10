# CLAUDE.md — terraform-provider-comfyui

## Project Overview

Terraform provider for [ComfyUI](https://github.com/comfyanonymous/ComfyUI), a node-based Stable Diffusion GUI.
Built with the **Terraform Plugin Framework** (not SDKv2). Language: Go + Python. License: MIT.

Fully implemented with **645 generated node resources** (one per ComfyUI node), **1 hand-written workflow resource**, and **5 data sources**.

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
├── internal/resources/               # Hand-written resources
│   └── workflow_resource.go          # comfyui_workflow: queue & execute workflows
├── internal/resources/generated/     # 645 generated node resources + registry
│   ├── registry.go                   # AllResources() — lists all generated constructors
│   └── resource_*.go                 # One file per ComfyUI node (e.g., resource_ksampler.go)
├── internal/datasources/             # 5 data sources
│   ├── system_stats.go               # comfyui_system_stats
│   ├── queue.go                      # comfyui_queue
│   ├── node_info.go                  # comfyui_node_info
│   ├── workflow_history.go           # comfyui_workflow_history
│   └── output.go                     # comfyui_output
├── third_party/ComfyUI/              # ComfyUI source (git submodule, pinned to tag)
├── doc/terraform/provider/research/  # 26 comprehensive research docs (00–25)
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

Extensive research lives in `doc/terraform/provider/research/`. **26 files** covering every aspect of Terraform provider development. Use the `terraform-provider-research` skill for guided access, or browse directly:

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

## Architecture Decisions

1. **Plugin Framework only** — No SDKv2. All resources, data sources, and functions use `terraform-plugin-framework`.
2. **Single API focus** — Provider wraps the ComfyUI REST API exclusively.
3. **Proposed resources**: `comfyui_workflow`, `comfyui_workflow_execution`, `comfyui_uploaded_image`, `comfyui_uploaded_mask`
4. **Proposed data sources**: `comfyui_system_stats`, `comfyui_queue`, `comfyui_node_info`, `comfyui_workflow_history`, `comfyui_output`
5. **Proposed functions**: `parse_workflow_json`, `node_output_name`

## Commands (planned)

```bash
# Build
go build -o terraform-provider-comfyui

# Run tests
go test ./... -v

# Run acceptance tests (requires running ComfyUI instance)
TF_ACC=1 go test ./... -v -timeout 120m

# Generate documentation
go generate ./...

# Install locally for development
go install .

# Lint
golangci-lint run ./...
```

## Conventions

- Follow HashiCorp naming: resources as `comfyui_<noun>`, data sources as `comfyui_<noun>`
- Attributes: snake_case, Required/Optional/Computed per schema design patterns (doc 06)
- Errors: Use diagnostics system, never panic (doc 09)
- Testing: Acceptance tests for every resource/data source (doc 10), unit tests for logic (doc 11)
- Documentation: Generated via tfplugindocs from schema + templates (doc 13)
- Commits: Conventional Commits format
- Versioning: Semantic Versioning (doc 15)

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

## Safety

- Never commit secrets or API keys
- Use environment variables for ComfyUI connection details (`COMFYUI_HOST`, `COMFYUI_PORT`)
- Do not modify `.claude/` hooks or agents configuration
