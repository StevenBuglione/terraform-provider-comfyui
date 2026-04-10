# 13 — Documentation Generation

## Overview

Terraform Registry providers need documentation in a specific format. The
`tfplugindocs` tool automates this by reading provider schemas and generating Markdown.

---

## The tfplugindocs Tool

**Repository:** [github.com/hashicorp/terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs)

`tfplugindocs` is a CLI tool maintained by HashiCorp that:

1. Builds and runs your provider binary to extract the schema via the gRPC interface.
2. Reads optional Go `text/template` files from `templates/`.
3. Reads example `.tf` and `.sh` files from `examples/`.
4. Generates Markdown files in `docs/` matching the Terraform Registry layout.

The generated Markdown uses the [Registry documentation format](https://developer.hashicorp.com/terraform/registry/providers/docs)
expected by the Terraform Registry and HCP Terraform.

---

## Installation

Install `tfplugindocs`:

```bash
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
tfplugindocs --version
```

In a project `Makefile`:

```makefile
# Makefile
.PHONY: docs
docs:
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	tfplugindocs generate

.PHONY: docs-validate
docs-validate:
	tfplugindocs validate
```

---

## Directory Structure

`tfplugindocs` expects and produces the following layout at the repository root:

```
terraform-provider-comfyui/
├── templates/                    # Optional Go text/template overrides
│   ├── index.md.tmpl             # Provider index page template
│   ├── resources/
│   │   └── workflow.md.tmpl      # Per-resource template overrides
│   └── data-sources/
│       └── node.md.tmpl          # Per-data-source template overrides
├── examples/                     # Example HCL configurations
│   ├── provider/
│   │   └── provider.tf           # Provider configuration example
│   ├── resources/
│   │   └── comfyui_workflow/
│   │       ├── resource.tf       # Resource usage example
│   │       └── import.sh         # Import command example
│   └── data-sources/
│       └── comfyui_node/
│           └── data-source.tf    # Data source usage example
└── docs/                         # Generated output (committed to repo)
    ├── index.md                  # Provider documentation index
    ├── resources/
    │   └── workflow.md           # Generated resource docs
    └── data-sources/
        └── node.md               # Generated data source docs
```

### Key points

- The `templates/` directory is **optional**. If absent, built-in default templates are used.
- The `examples/` directory holds raw `.tf` and `.sh` files that are embedded into the docs.
- The `docs/` directory is the **generated output** — it is committed to version control.
- Subdirectory names under `resources/` and `data-sources/` must match the full resource
  type name (e.g., `comfyui_workflow`, not just `workflow`).

---

## Running Generation

Generate documentation from the repository root:

```bash
tfplugindocs generate
```

This command performs the following steps:

1. Runs `go build` on the provider to produce a temporary binary.
2. Starts the provider binary and queries its schema via the plugin protocol.
3. For each resource and data source, renders a Markdown file using schema data,
   templates (if present), and example files (if present).
4. Writes output to `docs/`.

### Common flags

| Flag                     | Default      | Description                        |
|--------------------------|--------------|------------------------------------|
| `--provider-name`        | dir name     | Override the provider name         |
| `--rendered-website-dir` | `docs`       | Output directory for generated docs|
| `--examples-dir`         | `examples`   | Directory containing example files |
| `--templates-dir`        | `templates`  | Directory containing templates     |

---

## Validating Documentation

Validate that existing docs match the current schema:

```bash
tfplugindocs validate
```

This checks that:

- Every resource and data source in the schema has a corresponding doc file.
- No orphaned doc files exist for resources/data sources not in the schema.
- Front matter and structure are valid for the Terraform Registry.

Use validation in CI to catch documentation drift.

---

## Template Customization

Templates use Go `text/template` syntax. The default templates produce standard
Registry-compatible Markdown. Override them by placing `.md.tmpl` files in `templates/`.

### Available template data

Templates receive a data object with these fields (among others):

| Field                | Type     | Description                                      |
|----------------------|----------|--------------------------------------------------|
| `.Name`              | string   | Resource or data source name                     |
| `.Type`              | string   | `"Resource"` or `"Data Source"`                   |
| `.Description`       | string   | Schema description                                |
| `.HasExample`        | bool     | Whether an example file exists                    |
| `.ExampleFile`       | string   | Path to the example `.tf` file                    |
| `.HasImport`         | bool     | Whether an import example exists                  |
| `.ImportFile`        | string   | Path to the import `.sh` file                     |
| `.SchemaAttributes`  | []Attr   | List of schema attributes                        |

### Example: custom resource template

Create `templates/resources/comfyui_workflow.md.tmpl`:

```gotemplate
---
page_title: "comfyui_workflow Resource - terraform-provider-comfyui"
subcategory: ""
description: |-
  Manages a ComfyUI workflow.
---

# comfyui_workflow (Resource)

{{ .Description | trimspace }}

## Example Usage

{{ tffile .ExampleFile }}

{{ .SchemaMarkdown | trimspace }}

{{ if .HasImport -}}
## Import

Import is supported using the following syntax:

{{ codefile "shell" .ImportFile }}
{{- end }}
```

### Template functions

`tfplugindocs` provides these template functions:

- `{{ tffile "path/to/file.tf" }}` — Embeds a `.tf` file as a fenced HCL code block.
- `{{ codefile "language" "path/to/file" }}` — Embeds a file as a fenced code block with language.
- `{{ .SchemaMarkdown | trimspace }}` — Renders the auto-generated schema documentation.
- `{{ trimspace .Description }}` — Trims whitespace from strings.
- `{{ plainmarkdown .Description }}` — Renders as plain Markdown.

---

## Schema Descriptions

In the Plugin Framework, use the `MarkdownDescription` field on schema attributes
to provide rich documentation. `tfplugindocs` uses this field (falling back to
`Description`) when generating docs.

```go
func (r *WorkflowResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Manages a ComfyUI workflow definition, including its nodes and connections.",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                MarkdownDescription: "The unique identifier of the workflow.",
                Computed:            true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "name": schema.StringAttribute{
                MarkdownDescription: "The display name of the workflow. Must be unique within the workspace.",
                Required:            true,
            },
            "description": schema.StringAttribute{
                MarkdownDescription: "A human-readable description of what the workflow does.",
                Optional:            true,
            },
            "api_json": schema.StringAttribute{
                MarkdownDescription: "The ComfyUI API-format JSON representation of the workflow. " +
                    "See [ComfyUI API docs](https://example.com) for the schema.",
                Required: true,
            },
        },
    }
}
```

**Best practices for descriptions:**

- Use `MarkdownDescription` instead of `Description` to support inline links, code, and emphasis.
- Keep descriptions concise but complete — they are the primary documentation for users.
- Reference external docs with Markdown links when helpful.
- Note defaults, constraints, and relationships with other attributes.
- If both `MarkdownDescription` and `Description` are set, `MarkdownDescription` takes precedence.

---

## Example Configurations

Place example Terraform configurations under `examples/` to be embedded in docs.

### Provider example

`examples/provider/provider.tf`:

```hcl
terraform {
  required_providers {
    comfyui = {
      source = "registry.terraform.io/sbuglione/comfyui"
    }
  }
}

provider "comfyui" {
  host    = "http://localhost:8188"
  api_key = var.comfyui_api_key
}
```

### Resource example

`examples/resources/comfyui_workflow/resource.tf`:

```hcl
resource "comfyui_workflow" "example" {
  name        = "my-workflow"
  description = "An image generation workflow"
  api_json    = file("${path.module}/workflow.json")
}
```

### Import example

`examples/resources/comfyui_workflow/import.sh`:

```shell
# Import a workflow by its ID
terraform import comfyui_workflow.example abc-123-def
```

### Data source example

`examples/data-sources/comfyui_node/data-source.tf`:

```hcl
data "comfyui_node" "ksampler" {
  class_type = "KSampler"
}

output "ksampler_inputs" {
  value = data.comfyui_node.ksampler.inputs
}
```

---

## CI Integration

Integrate documentation generation into CI to prevent drift:

```makefile
.PHONY: generate
generate:
	go generate ./...
	tfplugindocs generate

.PHONY: check-docs
check-docs: generate
	@git diff --compact-summary --exit-code docs/
```

### GitHub Actions example

```yaml
name: Documentation
on:
  pull_request:
    paths: ['internal/**', 'templates/**', 'examples/**']
jobs:
  docs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
      - run: |
          go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
          tfplugindocs generate
      - name: Check for drift
        run: git diff --compact-summary --exit-code docs/
      - run: tfplugindocs validate
```

---

## References

- [Terraform Registry Provider Documentation](https://developer.hashicorp.com/terraform/registry/providers/docs)
- [terraform-plugin-docs GitHub Repository](https://github.com/hashicorp/terraform-plugin-docs)
- [Provider Documentation Best Practices](https://developer.hashicorp.com/terraform/plugin/best-practices/hashicorp-provider-design-principles)
- [Go text/template Package](https://pkg.go.dev/text/template)
