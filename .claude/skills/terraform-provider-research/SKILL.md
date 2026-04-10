---
name: terraform-provider-research
description: "Progressive disclosure of 26 Terraform provider research documents. Use when you need to understand any Terraform provider concept — architecture, schema, testing, CI/CD, publishing, or ComfyUI-specific mapping."
---

# Terraform Provider Research Skill

## When to Use This Skill

Use this skill when:
- Starting a new phase of provider development and need background knowledge
- Encountering an unfamiliar Terraform provider concept
- Need best practices or patterns for a specific implementation task
- Want to understand how AWS or AzureRM providers handle something
- Need the ComfyUI API → Terraform resource mapping

Do NOT use this skill when:
- You already have the specific information loaded in context
- The question is about general Go programming (not Terraform-specific)
- You need to write code (use `terraform-provider-dev` skill instead)

## Research Library Location

All research documents are at: `doc/terraform/provider/research/`

## Progressive Disclosure Strategy

Load documents **on demand** based on the current development phase. Never load all 26 at once — that wastes context. Use this phased approach:

### Phase 1: Foundation (start here for new projects)

Read these first to understand the big picture:

| Priority | File | When to Read |
|----------|------|--------------|
| 🔴 Must | `00-overview-and-architecture.md` | Before any provider work |
| 🔴 Must | `01-plugin-framework-fundamentals.md` | Before writing any Go code |
| 🔴 Must | `02-project-structure-and-scaffolding.md` | When initializing the Go project |
| 🟡 Soon | `19-provider-design-principles.md` | Before making architecture decisions |
| 🟡 Soon | `24-comfyui-provider-mapping.md` | Before designing resources |

**Action:** Read files 00, 01, 02 in order. Then 19 and 24 to understand what we're building.

### Phase 2: Core Implementation

Read these when implementing the provider skeleton and first resources:

| Priority | File | When to Read |
|----------|------|--------------|
| 🔴 Must | `03-provider-implementation.md` | When creating the provider struct |
| 🔴 Must | `04-resource-implementation.md` | When creating your first resource |
| 🔴 Must | `06-schema-design-patterns.md` | When defining any schema |
| 🟡 Soon | `05-data-source-implementation.md` | When creating your first data source |
| 🟡 Soon | `07-plan-modifiers-and-validators.md` | When adding validators or defaults |
| 🟡 Soon | `08-state-management-and-import.md` | When handling state or import |
| 🟡 Soon | `09-error-handling-and-diagnostics.md` | When implementing error handling |

**Action:** Read 03 → 04 → 06 as a sequence. Add 05, 07, 08, 09 as needed.

### Phase 3: Testing & Quality

Read these when ready to write tests:

| Priority | File | When to Read |
|----------|------|--------------|
| 🔴 Must | `10-acceptance-testing.md` | Before writing acceptance tests |
| 🔴 Must | `11-unit-testing.md` | Before writing unit tests |
| 🟡 Soon | `12-debugging-and-development-workflow.md` | When debugging issues |
| 🟡 Soon | `14-naming-conventions-and-style.md` | When reviewing code quality |

**Action:** Read 10 and 11 together. Use 12 when stuck. Check 14 for naming review.

### Phase 4: Documentation & Release

Read these when preparing for release:

| Priority | File | When to Read |
|----------|------|--------------|
| 🔴 Must | `13-documentation-generation.md` | When setting up tfplugindocs |
| 🔴 Must | `17-goreleaser-configuration.md` | When setting up GoReleaser |
| 🔴 Must | `18-registry-publishing.md` | When publishing to registry |
| 🟡 Soon | `15-versioning-and-changelog.md` | When preparing first release |
| 🟡 Soon | `16-ci-cd-and-github-actions.md` | When setting up CI/CD |
| 🟡 Soon | `23-makefile-and-dev-commands.md` | When creating Makefile |

**Action:** Read 17 → 18 → 13 for release pipeline. Add 15, 16, 23 for CI.

### Phase 5: Advanced & Reference

Read these for edge cases and inspiration:

| Priority | File | When to Read |
|----------|------|--------------|
| 🟢 Later | `20-advanced-patterns.md` | For ephemeral resources, write-only |
| 🟢 Later | `25-provider-functions.md` | When implementing provider functions |
| 🟢 Later | `21-reference-provider-aws.md` | For architecture inspiration |
| 🟢 Later | `22-reference-provider-azurerm.md` | For architecture inspiration |

**Action:** Read on-demand when a specific advanced pattern is needed.

## Quick Lookup Guide

Use this to find the right document for a specific question:

| Question | Read |
|----------|------|
| "How does the Terraform plugin system work?" | `00` |
| "What interfaces do I implement?" | `01` |
| "How should I structure the Go project?" | `02` |
| "How do I implement the provider Configure?" | `03` |
| "How do I implement CRUD for a resource?" | `04` |
| "How do I create a data source?" | `05` |
| "What attribute types are available?" | `06` |
| "How do I add validation?" | `07` |
| "How does state management work?" | `08` |
| "How do I handle errors properly?" | `09` |
| "How do I write acceptance tests?" | `10` |
| "How do I write unit tests?" | `11` |
| "How do I debug a provider?" | `12` |
| "How do I generate docs?" | `13` |
| "What are the naming conventions?" | `14` |
| "How do I version my provider?" | `15` |
| "How do I set up CI/CD?" | `16` |
| "How do I configure GoReleaser?" | `17` |
| "How do I publish to the registry?" | `18` |
| "What are HashiCorp's design principles?" | `19` |
| "What are ephemeral resources?" | `20` |
| "How does the AWS provider work?" | `21` |
| "How does the AzureRM provider work?" | `22` |
| "What Makefile targets should I have?" | `23` |
| "How does ComfyUI API map to Terraform?" | `24` |
| "How do provider functions work?" | `25` |

## How to Load a Document

When you need information from a specific document, read it using:

```bash
cat doc/terraform/provider/research/<filename>.md
```

For targeted lookups within a document:

```bash
grep -n "<keyword>" doc/terraform/provider/research/<filename>.md
```

To search across all research docs:

```bash
grep -rn "<keyword>" doc/terraform/provider/research/
```

## Key Facts (Always Remember)

These facts apply to ALL development work on this provider:

1. **Plugin Framework only** — Never use SDKv2 (`terraform-plugin-sdk`). Always use `terraform-plugin-framework`.
2. **Go module path**: `github.com/StevenBuglworione/terraform-provider-comfyui` (verify actual GitHub username)
3. **Provider name**: `comfyui`
4. **Resource prefix**: `comfyui_`
5. **Required Go packages**:
   - `github.com/hashicorp/terraform-plugin-framework`
   - `github.com/hashicorp/terraform-plugin-go`
   - `github.com/hashicorp/terraform-plugin-testing`
   - `github.com/hashicorp/terraform-plugin-log`
6. **ComfyUI API**: REST over HTTP, default port 8188, WebSocket for real-time
7. **Target Terraform version**: 1.8+ (for provider functions support)
