# CLAUDE.md — terraform-provider-comfyui

## Project Overview

Terraform provider for [ComfyUI](https://github.com/comfyanonymous/ComfyUI), a node-based Stable Diffusion GUI.
Built with the **Terraform Plugin Framework** (not SDKv2). Language: Go. License: MIT.

## Tech Stack

- Go 1.21+
- Terraform Plugin Framework (`terraform-plugin-framework`)
- Terraform Plugin Testing (`terraform-plugin-testing`)
- Terraform Plugin Log (`terraform-plugin-log`)
- Terraform Plugin Docs (`terraform-plugin-docs`)
- GoReleaser for builds/releases
- GitHub Actions for CI/CD

## Key Directories

```
.
├── CLAUDE.md                  # This file — project instructions for AI agents
├── LICENSE                    # MIT License
├── doc/terraform/provider/research/  # 26 comprehensive research docs (00–25)
├── .claude/skills/            # Claude Code skills for this project
│   ├── terraform-provider-research/  # Progressive research disclosure skill
│   └── terraform-provider-dev/       # Hands-on development guidance skill
├── internal/provider/         # (planned) Provider implementation
├── internal/resources/        # (planned) Resource implementations
├── internal/datasources/      # (planned) Data source implementations
├── internal/client/           # (planned) ComfyUI API client
└── examples/                  # (planned) HCL usage examples
```

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
