# 14 — Naming Conventions and Style

## Overview

Consistent naming is critical for Terraform providers. Users interact with
resource names, attribute names, and data source names directly in HCL. The
Go code backing these must also follow predictable patterns so that both
humans and automated tooling can navigate the codebase. This document covers
all naming conventions for a Plugin Framework provider.

**Primary reference:** [HashiCorp Naming Best Practices](https://developer.hashicorp.com/terraform/plugin/best-practices/naming)

---

## Provider Naming

The provider repository and binary must follow this pattern:

```
terraform-provider-<name>
```

Rules:

- `<name>` is **all lowercase**, no hyphens, no underscores.
- The name appears in `required_providers` and resource prefixes.
- Example: `terraform-provider-comfyui` → provider name is `comfyui`.

```hcl
terraform {
  required_providers {
    comfyui = {
      source  = "registry.terraform.io/sbuglione/comfyui"
      version = "~> 1.0"
    }
  }
}

provider "comfyui" {
  host = "http://localhost:8188"
}
```

The Go module path should match:

```go
module github.com/sbuglione/terraform-provider-comfyui
```

---

## Resource Naming

Resources follow the pattern `<provider>_<noun>`:

### Rules

| Rule                            | Good                          | Bad                               |
|---------------------------------|-------------------------------|-----------------------------------|
| Prefix with provider name       | `comfyui_workflow`            | `workflow`                        |
| Use singular nouns              | `comfyui_workflow`            | `comfyui_workflows`               |
| Use underscores to separate     | `comfyui_custom_node`        | `comfyui_customnode`              |
| Match API terminology           | `comfyui_workflow`            | `comfyui_pipeline` (if API says workflow) |
| Avoid redundant provider name   | `comfyui_workflow`            | `comfyui_comfyui_workflow`        |
| Avoid verbs                     | `comfyui_workflow`            | `comfyui_create_workflow`         |

### Examples for a ComfyUI provider

```hcl
resource "comfyui_workflow" "main" {
  name     = "txt2img"
  api_json = file("workflow.json")
}

resource "comfyui_custom_node" "controlnet" {
  name       = "controlnet-preprocessor"
  repository = "https://github.com/example/controlnet"
}
```

### Go type registration

In the Plugin Framework, resource type names are returned by the `Metadata` method:

```go
func (r *WorkflowResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_workflow"
}
```

The provider automatically prepends its name via `req.ProviderTypeName`, so the
resource implementation only appends `_workflow`. This yields `comfyui_workflow`.

---

## Data Source Naming

Data sources follow the same `<provider>_<noun>` pattern as resources.

### Singular vs. plural

- Use **singular** for single-object data sources (`comfyui_workflow`).
- Use **plural** for list/collection data sources (`comfyui_nodes`).

Examples across providers:

| Provider | Singular (one item)       | Plural (collection)           |
|----------|---------------------------|-------------------------------|
| AWS      | `aws_instance`            | `aws_availability_zones`      |
| ComfyUI  | `comfyui_workflow`        | `comfyui_nodes`               |

### Go type registration for data sources

```go
func (d *NodesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_nodes"
}
```

---

## Function Naming

Provider-defined functions (Terraform 1.8+, Plugin Framework 1.8+) do **not**
use the provider prefix because they are scoped at the call site:

```hcl
# Called as: provider::comfyui::parse_workflow_id("abc-123")
# The function name is just "parse_workflow_id"
```

### Rules

| Rule                                    | Good                    | Bad                             |
|-----------------------------------------|-------------------------|---------------------------------|
| Do NOT prefix with provider name        | `parse_workflow_id`     | `comfyui_parse_workflow_id`     |
| Use verbs or actions                    | `parse_workflow_id`     | `workflow_id`                   |
| All lowercase with underscores          | `encode_api_json`       | `encodeApiJson`                 |

### Go registration

```go
func (p *ComfyUIProvider) Functions(ctx context.Context) []func() function.Function {
    return []func() function.Function{
        NewParseWorkflowIDFunction,
    }
}

func (f *ParseWorkflowIDFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
    resp.Name = "parse_workflow_id"
}
```

---

## Attribute Naming

Schema attribute names appear directly in HCL and must be user-friendly.

### General rules

| Rule                              | Good                        | Bad                          |
|-----------------------------------|-----------------------------|------------------------------|
| Lowercase with underscores        | `api_key`                   | `apiKey`, `API_KEY`          |
| Singular for scalar values        | `name`                      | `names`                      |
| Plural for collections            | `tags`                      | `tag`                        |
| Use `id` for the primary ID       | `id`                        | `workflow_id` (for own ID)   |
| Foreign keys: `<thing>_id`        | `workspace_id`              | `workspace`                  |

### Boolean attributes

Boolean attributes should use nouns or passive verbs. Do **not** use `is_` or `has_` prefixes:

```go
// Good: "enabled", "monitoring", "delete_on_termination", "force_destroy", "auto_queue"
// Bad:  "is_enabled", "has_monitoring" (avoid is_/has_ prefixes)
```

### ID attributes

Every managed resource should have an `id` attribute:

```go
"id": schema.StringAttribute{
    MarkdownDescription: "The unique identifier of the workflow.",
    Computed:            true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(),
    },
},
```

For references to other resources, use `<resource>_id`:

```go
"workspace_id": schema.StringAttribute{
    MarkdownDescription: "The ID of the workspace this workflow belongs to.",
    Required:            true,
},
```

### Timeouts attribute

Use the Plugin Framework's built-in timeouts support:

```go
import "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
    Create: true,
    Read:   true,
    Update: true,
    Delete: true,
}),
```

---

## Go Code Style

### Standard Go conventions

- Format all code with `gofmt` (or `goimports`).
- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines.
- Use `golangci-lint run ./...` for linting.

### File naming

| Kind                 | File pattern                       | Example                          |
|----------------------|------------------------------------|----------------------------------|
| Resource             | `<name>_resource.go`               | `workflow_resource.go`           |
| Resource tests       | `<name>_resource_test.go`          | `workflow_resource_test.go`      |
| Data source          | `<name>_data_source.go`            | `node_data_source.go`            |
| Data source tests    | `<name>_data_source_test.go`       | `node_data_source_test.go`       |
| Function             | `<name>_function.go`               | `parse_workflow_id_function.go`  |
| Function tests       | `<name>_function_test.go`          | `parse_workflow_id_function_test.go` |
| Provider             | `provider.go`                      | `provider.go`                    |
| Provider tests       | `provider_test.go`                 | `provider_test.go`               |

All resource, data source, and function files live under `internal/provider/`.

### Struct naming

```go
type ComfyUIProvider struct{ version string }
type ComfyUIProviderModel struct {
    Host   types.String `tfsdk:"host"`
    APIKey types.String `tfsdk:"api_key"`
}

type WorkflowResource struct{ client *comfyui.Client }
type WorkflowResourceModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    Description types.String `tfsdk:"description"`
    APIJson     types.String `tfsdk:"api_json"`
}

type NodeDataSource struct{ client *comfyui.Client }
type NodeDataSourceModel struct {
    ClassType types.String `tfsdk:"class_type"`
    Inputs    types.List   `tfsdk:"inputs"`
}
```

Convention: `<Name>Resource`, `<Name>ResourceModel`, `<Name>DataSource`, `<Name>DataSourceModel`.

### Constructor pattern

Every resource and data source uses a constructor function returned by the
provider's `Resources()` or `DataSources()` method:

```go
func NewWorkflowResource() resource.Resource {
    return &WorkflowResource{}
}

func NewNodeDataSource() datasource.DataSource {
    return &NodeDataSource{}
}
```

### Interface implementation assertions

Place compile-time interface checks at the top of each file:

```go
// Ensure provider defined types fully satisfy framework interfaces.
var (
    _ resource.Resource                = &WorkflowResource{}
    _ resource.ResourceWithConfigure   = &WorkflowResource{}
    _ resource.ResourceWithImportState = &WorkflowResource{}
)
```

---

## Terraform HCL Style

When writing example HCL in documentation and `examples/` files, follow the
[Terraform Style Conventions](https://developer.hashicorp.com/terraform/language/style).
Key rules: 2-space indentation, align `=` signs within blocks, place
meta-arguments (`depends_on`, `count`, `for_each`, `lifecycle`) last,
use `#` for comments.

```hcl
resource "comfyui_workflow" "example" {
  name        = "my-workflow"
  description = "An example image generation workflow"
  api_json    = file("${path.module}/workflow.json")
  auto_queue  = true

  tags = {
    environment = "development"
    team        = "ml-ops"
  }

  lifecycle {
    prevent_destroy = true
  }
}
```

---

## Summary Table

| Entity         | Pattern                              | Example                                   |
|----------------|--------------------------------------|--------------------------------------------|
| Provider repo  | `terraform-provider-<name>`          | `terraform-provider-comfyui`               |
| Resource type  | `<provider>_<singular_noun>`         | `comfyui_workflow`                         |
| Data source    | `<provider>_<noun(s)>`               | `comfyui_node`, `comfyui_nodes`            |
| Function       | `<verb_phrase>` (no provider prefix) | `parse_workflow_id`                        |
| Attribute      | `lowercase_underscored`              | `api_json`, `class_type`, `workspace_id`   |
| Go resource    | `<Name>Resource`                     | `WorkflowResource`                         |
| Go data source | `<Name>DataSource`                   | `NodeDataSource`                           |
| Go model       | `<Name>ResourceModel`                | `WorkflowResourceModel`                    |
| Go file        | `<name>_resource.go`                 | `workflow_resource.go`                     |

---

## References

- [HashiCorp Provider Naming Best Practices](https://developer.hashicorp.com/terraform/plugin/best-practices/naming)
- [Terraform Style Conventions](https://developer.hashicorp.com/terraform/language/style)
- [Effective Go](https://go.dev/doc/effective_go)
- [Plugin Framework Documentation](https://developer.hashicorp.com/terraform/plugin/framework)
