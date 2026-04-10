---
name: terraform-provider-dev
description: "Hands-on development guidance for implementing terraform-provider-comfyui. Use when writing provider code, creating resources, data sources, tests, CI/CD, or release configuration. Provides step-by-step workflows grounded in the research documentation."
---

# Terraform Provider Development Skill

## When to Use This Skill

Use this skill when:
- Writing Go code for the provider
- Creating new resources or data sources
- Writing tests (acceptance or unit)
- Setting up build, CI/CD, or release pipelines
- Making architecture decisions for the provider

Do NOT use this skill when:
- You only need conceptual understanding (use `terraform-provider-research` instead)
- Working on non-provider code

## Core Principles

1. **Plugin Framework only** — All code uses `terraform-plugin-framework`. Never import `terraform-plugin-sdk`.
2. **One resource = one file** — Each resource gets its own file in `internal/resources/`.
3. **Tests for everything** — Every resource/data source must have acceptance tests.
4. **Diagnostics, not panics** — Use `resp.Diagnostics.AddError()`, never `panic()` or `log.Fatal()`.
5. **Schema is documentation** — Write clear `MarkdownDescription` for every attribute.

## Project Initialization Workflow

When initializing the Go project from scratch:

```bash
# 1. Initialize Go module
go mod init github.com/<owner>/terraform-provider-comfyui

# 2. Create directory structure
mkdir -p internal/provider internal/resources internal/datasources internal/client
mkdir -p examples/resources examples/data-sources
mkdir -p templates

# 3. Add dependencies
go get github.com/hashicorp/terraform-plugin-framework
go get github.com/hashicorp/terraform-plugin-go
go get github.com/hashicorp/terraform-plugin-log
go get github.com/hashicorp/terraform-plugin-testing

# 4. Create main.go
# See: doc/terraform/provider/research/03-provider-implementation.md

# 5. Create GNUmakefile
# See: doc/terraform/provider/research/23-makefile-and-dev-commands.md
```

Reference docs: `02-project-structure-and-scaffolding.md`, `03-provider-implementation.md`

## Resource Implementation Checklist

When creating a new resource (e.g., `comfyui_workflow`):

### Step 1: Define the resource file

Create `internal/resources/workflow_resource.go`:

```go
package resources

import (
    "context"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var _ resource.Resource = &WorkflowResource{}
var _ resource.ResourceWithImportState = &WorkflowResource{}

type WorkflowResource struct {
    client *client.ComfyUIClient
}

func NewWorkflowResource() resource.Resource {
    return &WorkflowResource{}
}

func (r *WorkflowResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_workflow"
}
```

### Step 2: Define the schema

```go
func (r *WorkflowResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Manages a ComfyUI workflow.",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed:            true,
                MarkdownDescription: "Workflow identifier.",
            },
            // Add more attributes...
        },
    }
}
```

### Step 3: Implement CRUD methods

```go
func (r *WorkflowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan WorkflowModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }
    // API call to create
    // Set state from response
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state WorkflowModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }
    // API call to read
    // If not found: resp.State.RemoveResource(ctx)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WorkflowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan WorkflowModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }
    // API call to update
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state WorkflowModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }
    // API call to delete
}
```

### Step 4: Define the model

```go
type WorkflowModel struct {
    ID   types.String `tfsdk:"id"`
    Name types.String `tfsdk:"name"`
    // Map all attributes from schema
}
```

### Step 5: Register in provider

Add to `internal/provider/provider.go`:

```go
func (p *ComfyUIProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        resources.NewWorkflowResource,
    }
}
```

### Step 6: Write acceptance tests

Create `internal/resources/workflow_resource_test.go`:

```go
func TestAccWorkflowResource_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccWorkflowResourceConfig("test"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("comfyui_workflow.test", "name", "test"),
                ),
            },
        },
    })
}
```

Reference docs: `04-resource-implementation.md`, `06-schema-design-patterns.md`, `10-acceptance-testing.md`

## Data Source Implementation Checklist

When creating a new data source (e.g., `comfyui_system_stats`):

1. Create `internal/datasources/system_stats_data_source.go`
2. Implement `datasource.DataSource` interface: `Metadata`, `Schema`, `Read`
3. All attributes should be `Computed: true`
4. Register in `provider.DataSources()`
5. Write acceptance tests

Reference docs: `05-data-source-implementation.md`

## Provider Function Implementation Checklist

When creating a provider function (e.g., `parse_workflow_json`):

1. Create `internal/functions/parse_workflow_json.go`
2. Implement `function.Function` interface: `Metadata`, `Definition`, `Run`
3. Register in `provider.Functions()`
4. Write unit tests

Reference docs: `25-provider-functions.md`

## Testing Workflow

### Run all tests

```bash
go test ./... -v
```

### Run acceptance tests (requires live ComfyUI)

```bash
COMFYUI_HOST=localhost COMFYUI_PORT=8188 TF_ACC=1 go test ./... -v -timeout 120m
```

### Run a single test

```bash
TF_ACC=1 go test ./internal/resources/ -run TestAccWorkflowResource_basic -v
```

### Debug a failing test

```bash
TF_LOG=DEBUG TF_ACC=1 go test ./internal/resources/ -run TestAccWorkflowResource_basic -v
```

Reference docs: `10-acceptance-testing.md`, `11-unit-testing.md`, `12-debugging-and-development-workflow.md`

## ComfyUI API Client

The API client should live in `internal/client/` and wrap all ComfyUI REST endpoints:

```go
type ComfyUIClient struct {
    BaseURL    string
    HTTPClient *http.Client
}

func NewComfyUIClient(host string, port int) *ComfyUIClient {
    return &ComfyUIClient{
        BaseURL:    fmt.Sprintf("http://%s:%d", host, port),
        HTTPClient: &http.Client{Timeout: 30 * time.Second},
    }
}
```

### Key API methods to implement

| Method | Endpoint | Used By |
|--------|----------|---------|
| `QueuePrompt()` | `POST /prompt` | `comfyui_workflow_execution` |
| `GetHistory(id)` | `GET /history/{id}` | `comfyui_workflow_history` |
| `GetSystemStats()` | `GET /system_stats` | `comfyui_system_stats` |
| `GetObjectInfo()` | `GET /object_info` | `comfyui_node_info` |
| `GetQueue()` | `GET /queue` | `comfyui_queue` |
| `UploadImage()` | `POST /upload/image` | `comfyui_uploaded_image` |
| `UploadMask()` | `POST /upload/mask` | `comfyui_uploaded_mask` |
| `GetImage()` | `GET /view` | `comfyui_output` |
| `Interrupt()` | `POST /interrupt` | Provider-level action |

Reference docs: `24-comfyui-provider-mapping.md`

## Provider Configuration

The provider schema should accept:

```hcl
provider "comfyui" {
  host = "localhost"  # or COMFYUI_HOST env var
  port = 8188         # or COMFYUI_PORT env var
}
```

Implementation pattern:

```go
func (p *ComfyUIProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "host": schema.StringAttribute{
                Optional:            true,
                MarkdownDescription: "ComfyUI server host. Can also be set via COMFYUI_HOST env var.",
            },
            "port": schema.Int64Attribute{
                Optional:            true,
                MarkdownDescription: "ComfyUI server port. Can also be set via COMFYUI_PORT env var.",
            },
        },
    }
}
```

Reference docs: `03-provider-implementation.md`

## Release Workflow

### Setup GoReleaser

Create `.goreleaser.yml`:
- Reference: `17-goreleaser-configuration.md`

### Setup GitHub Actions CI

Create `.github/workflows/`:
- `test.yml` — Run tests on PR
- `release.yml` — Build and release on tag
- Reference: `16-ci-cd-and-github-actions.md`

### Publish to Terraform Registry

1. Sign manifest at `terraform-registry-manifest.json`
2. Setup GPG key for signing
3. Push tagged release
- Reference: `18-registry-publishing.md`

## Error Patterns

### Always use diagnostics

```go
// ✅ Correct
resp.Diagnostics.AddError(
    "Error creating workflow",
    fmt.Sprintf("Could not create workflow: %s", err),
)
return

// ❌ Wrong — never do this
panic(err)
log.Fatalf("error: %s", err)
```

### Handle 404 in Read

```go
if resp.StatusCode == http.StatusNotFound {
    tflog.Warn(ctx, "Resource not found, removing from state")
    resp.State.RemoveResource(ctx)
    return
}
```

Reference docs: `09-error-handling-and-diagnostics.md`

## Naming Rules

| Element | Convention | Example |
|---------|-----------|---------|
| Resource type | `comfyui_<noun>` | `comfyui_workflow` |
| Data source type | `comfyui_<noun>` | `comfyui_system_stats` |
| Function name | `<verb>_<noun>` | `parse_workflow_json` |
| Go resource struct | `<Noun>Resource` | `WorkflowResource` |
| Go model struct | `<Noun>Model` | `WorkflowModel` |
| Go file name | `<noun>_resource.go` | `workflow_resource.go` |
| Test file name | `<noun>_resource_test.go` | `workflow_resource_test.go` |
| Attribute name | `snake_case` | `prompt_id` |

Reference docs: `14-naming-conventions-and-style.md`
