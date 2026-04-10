# Plugin Framework Fundamentals

## 1. Core Interfaces

The Plugin Framework is built around three primary Go interfaces. Every provider
implements `provider.Provider`; each managed object implements `resource.Resource`;
each read-only lookup implements `datasource.DataSource`.

### 1.1 `provider.Provider`

```go
package provider

import (
    "context"

    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/provider"
    "github.com/hashicorp/terraform-plugin-framework/provider/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure interface compliance at compile time.
var _ provider.Provider = &ComfyUIProvider{}

type ComfyUIProvider struct {
    // version is set by the build process via ldflags.
    version string
}

func (p *ComfyUIProvider) Metadata(ctx context.Context,
    req provider.MetadataRequest, resp *provider.MetadataResponse) {
    resp.TypeName = "comfyui"
    resp.Version = p.version
}

func (p *ComfyUIProvider) Schema(ctx context.Context,
    req provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "endpoint": schema.StringAttribute{
                Required:    true,
                Description: "The ComfyUI API endpoint URL.",
            },
        },
    }
}

func (p *ComfyUIProvider) Configure(ctx context.Context,
    req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var config ComfyUIProviderModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
    if resp.Diagnostics.HasError() {
        return
    }
    // Create API client and store in resp.DataSourceData / resp.ResourceData
    client := api.NewClient(config.Endpoint.ValueString())
    resp.DataSourceData = client
    resp.ResourceData = client
}

func (p *ComfyUIProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewWorkflowResource,
    }
}

func (p *ComfyUIProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        NewWorkflowDataSource,
    }
}
```

The interface methods and their purposes:

| Method | When Called | Purpose |
|---|---|---|
| `Metadata` | Schema retrieval | Sets the provider type name (prefix for all resources) and version |
| `Schema` | Schema retrieval | Defines provider-level configuration attributes |
| `Configure` | Before any resource/data source CRUD | Resolves config, creates API clients, passes them to resources |
| `Resources` | Schema retrieval | Returns factory functions for all resources |
| `DataSources` | Schema retrieval | Returns factory functions for all data sources |

### 1.2 `resource.Resource`

```go
// Ensure interface compliance at compile time.
var _ resource.Resource = &WorkflowResource{}
var _ resource.ResourceWithImportState = &WorkflowResource{}

type WorkflowResource struct {
    client *api.Client
}

func NewWorkflowResource() resource.Resource {
    return &WorkflowResource{}
}

func (r *WorkflowResource) Metadata(ctx context.Context,
    req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (r *WorkflowResource) Schema(ctx context.Context,
    req resource.SchemaRequest, resp *resource.SchemaResponse) {
    // See Section 2 for schema details
}

func (r *WorkflowResource) Configure(ctx context.Context,
    req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    client, ok := req.ProviderData.(*api.Client)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Resource Configure Type",
            fmt.Sprintf("Expected *api.Client, got: %T", req.ProviderData),
        )
        return
    }
    r.client = client
}

func (r *WorkflowResource) Create(ctx context.Context,
    req resource.CreateRequest, resp *resource.CreateResponse) { /* ... */ }

func (r *WorkflowResource) Read(ctx context.Context,
    req resource.ReadRequest, resp *resource.ReadResponse) { /* ... */ }

func (r *WorkflowResource) Update(ctx context.Context,
    req resource.UpdateRequest, resp *resource.UpdateResponse) { /* ... */ }

func (r *WorkflowResource) Delete(ctx context.Context,
    req resource.DeleteRequest, resp *resource.DeleteResponse) { /* ... */ }

func (r *WorkflowResource) ImportState(ctx context.Context,
    req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

Resource CRUD lifecycle:

| Method | Terraform Operation | What It Does |
|---|---|---|
| `Create` | `apply` (new resource) | Calls API to create the object; sets full state |
| `Read` | `plan`, `apply`, `refresh` | Reads current state from API; updates Terraform state |
| `Update` | `apply` (changed resource) | Calls API to update the object; sets new state |
| `Delete` | `destroy`, `apply` (removed) | Calls API to delete the object; removes state |
| `ImportState` | `terraform import` | Maps an external ID to Terraform state |

### 1.3 `datasource.DataSource`

```go
var _ datasource.DataSource = &WorkflowDataSource{}

type WorkflowDataSource struct {
    client *api.Client
}

func (d *WorkflowDataSource) Metadata(ctx context.Context,
    req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (d *WorkflowDataSource) Schema(ctx context.Context,
    req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    // Define read-only schema
}

func (d *WorkflowDataSource) Configure(ctx context.Context,
    req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
    // Same pattern as resource.Configure
}

func (d *WorkflowDataSource) Read(ctx context.Context,
    req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // Fetch from API, populate resp.State
}
```

Data sources only have a `Read` method — they are purely read-only.

---

## 2. The Type System

The Plugin Framework uses a custom type system in the `types` package that wraps
Go primitives to support Terraform's three-value logic: **Known**, **Null**, and
**Unknown**.

### 2.1 Primitive Types

| Framework Type | Go Underlying | Terraform HCL Type | Example |
|---|---|---|---|
| `types.String` | `string` | `string` | `"hello"` |
| `types.Int64` | `int64` | `number` (integer) | `42` |
| `types.Float64` | `float64` | `number` (float) | `3.14` |
| `types.Bool` | `bool` | `bool` | `true` |
| `types.Number` | `*big.Float` | `number` (arbitrary) | `99999999999999999` |

### 2.2 Collection Types

| Framework Type | Terraform HCL Type | Element Constraint |
|---|---|---|
| `types.List` | `list(T)` | Ordered, elements of same type |
| `types.Set` | `set(T)` | Unordered, unique elements of same type |
| `types.Map` | `map(T)` | String keys, values of same type |
| `types.Object` | `object({...})` | Named attributes with potentially different types |

### 2.3 Using Types in Go Models

Terraform state models use `tfsdk` struct tags to map Go fields to schema
attributes:

```go
type WorkflowResourceModel struct {
    ID          types.String  `tfsdk:"id"`
    Name        types.String  `tfsdk:"name"`
    Description types.String  `tfsdk:"description"`
    NodeCount   types.Int64   `tfsdk:"node_count"`
    IsActive    types.Bool    `tfsdk:"is_active"`
    Tags        types.List    `tfsdk:"tags"`        // list(string)
    Metadata    types.Map     `tfsdk:"metadata"`    // map(string)
    Config      types.Object  `tfsdk:"config"`      // object({...})
}
```

### 2.4 Reading and Writing Values

```go
// Reading a value from plan/state
name := model.Name.ValueString()          // Get the Go string
isNull := model.Name.IsNull()             // Check if null
isUnknown := model.Name.IsUnknown()       // Check if unknown

// Writing a value to state
model.Name = types.StringValue("my-workflow")       // Set a known value
model.Name = types.StringNull()                      // Set to null
model.Name = types.StringUnknown()                   // Set to unknown (plan only)

// Int64
model.NodeCount = types.Int64Value(5)
count := model.NodeCount.ValueInt64()

// Bool
model.IsActive = types.BoolValue(true)

// List of strings
tags, diags := types.ListValueFrom(ctx, types.StringType, []string{"a", "b"})
model.Tags = tags
```

---

## 3. Null vs Unknown vs Known

This three-value logic is **fundamental** to how Terraform planning works and is
one of the most important concepts to understand.

| State | Meaning | When It Occurs | `IsNull()` | `IsUnknown()` |
|---|---|---|---|---|
| **Known** | Attribute has a definite value | After apply; literals in config | `false` | `false` |
| **Null** | Attribute is explicitly absent | Optional attribute not set; set to `null` | `true` | `false` |
| **Unknown** | Value will be determined at apply time | Computed attributes during plan; values from other resources | `false` | `true` |

### 3.1 Why Unknown Matters

During `terraform plan`, Terraform cannot know the value of computed attributes
(e.g., an auto-generated ID). These are represented as **unknown**. Your provider
must:

1. **Never** try to use `.ValueString()` on an unknown value in a plan — the result
   is meaningless.
2. Return unknown values from `PlanResourceChange` for any attribute that will be
   computed at apply time.
3. Replace unknowns with real values in `Create` / `Update`.

```go
func (r *WorkflowResource) Create(ctx context.Context,
    req resource.CreateRequest, resp *resource.CreateResponse) {

    var plan WorkflowResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // plan.ID is Unknown here — do NOT read it
    // Call the API to create the resource
    result, err := r.client.CreateWorkflow(ctx, plan.Name.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Create failed", err.Error())
        return
    }

    // Now set the ID to a Known value
    plan.ID = types.StringValue(result.ID)

    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

### 3.2 Null Handling

Optional attributes that the user omits will be **Null**. Always check before using:

```go
if !model.Description.IsNull() {
    apiRequest.Description = model.Description.ValueString()
}
```

---

## 4. Go Compatibility

The Plugin Framework follows HashiCorp's Go version support policy:

> **Support the latest two minor versions of Go.**

For example, if Go 1.23 is the latest release, the framework supports Go 1.23
and Go 1.22. This is declared in `go.mod`:

```go
module github.com/sbuglione/terraform-provider-comfyui

go 1.22
```

Always test against both supported Go versions in CI.

---

## 5. Key Packages

### 5.1 Package Reference

| Package | Import Path | Purpose |
|---|---|---|
| `provider` | `terraform-plugin-framework/provider` | `Provider` interface and request/response types |
| `resource` | `terraform-plugin-framework/resource` | `Resource` interface, CRUD request/response types |
| `datasource` | `terraform-plugin-framework/datasource` | `DataSource` interface |
| `schema` (provider) | `terraform-plugin-framework/provider/schema` | Provider schema definition |
| `schema` (resource) | `terraform-plugin-framework/resource/schema` | Resource schema definition |
| `schema` (datasource) | `terraform-plugin-framework/datasource/schema` | Data source schema definition |
| `types` | `terraform-plugin-framework/types` | All framework types (`String`, `Int64`, etc.) |
| `diag` | `terraform-plugin-framework/diag` | Diagnostics (errors and warnings) |
| `path` | `terraform-plugin-framework/path` | Attribute path expressions |
| `tfsdk` | `terraform-plugin-framework/tfsdk` | State/plan/config access helpers |
| `planmodifier` | `terraform-plugin-framework/resource/schema/planmodifier` | Plan modification interfaces |
| `stringplanmodifier` | `terraform-plugin-framework/resource/schema/stringplanmodifier` | Built-in string plan modifiers |
| `int64planmodifier` | `terraform-plugin-framework/resource/schema/int64planmodifier` | Built-in int64 plan modifiers |
| `validator` | `terraform-plugin-framework-validators/...` | Composable validators (separate module) |
| `tflog` | `terraform-plugin-log/tflog` | Structured logging |

### 5.2 Important: Separate Schema Packages

The framework has **separate schema packages** for providers, resources, and data
sources. This is a common source of confusion:

```go
import (
    // Provider schema
    providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"

    // Resource schema
    resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"

    // Data source schema
    datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)
```

In practice, you typically only import the schema package relevant to the file
you are working in (e.g., `resource/schema` in a resource file).

---

## 6. Provider Server: `providerserver.Serve`

The entry point for any Plugin Framework provider is `providerserver.Serve` (or the
newer `providerserver.NewProtocol6Server` for testing).

### 6.1 main.go

```go
package main

import (
    "context"
    "flag"
    "log"

    "github.com/hashicorp/terraform-plugin-framework/providerserver"
    "github.com/sbuglione/terraform-provider-comfyui/internal/provider"
)

// These are set by goreleaser via ldflags.
var version string = "dev"

func main() {
    var debug bool

    flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
    flag.Parse()

    opts := providerserver.ServeOpts{
        Address: "registry.terraform.io/sbuglione/comfyui",
        Debug:   debug,
    }

    err := providerserver.Serve(context.Background(), provider.New(version), opts)
    if err != nil {
        log.Fatal(err.Error())
    }
}
```

### 6.2 `ServeOpts` Fields

| Field | Type | Purpose |
|---|---|---|
| `Address` | `string` | Full registry address (e.g., `registry.terraform.io/sbuglione/comfyui`) |
| `Debug` | `bool` | Enables debug mode: provider prints reattach config to stdout, waits for debugger |
| `ProtocolVersion` | `int` | `5` or `6`; defaults to `6` if omitted |

### 6.3 The Provider Factory Function

```go
// internal/provider/provider.go
func New(version string) func() provider.Provider {
    return func() provider.Provider {
        return &ComfyUIProvider{
            version: version,
        }
    }
}
```

`providerserver.Serve` expects `func() provider.Provider` — a factory function,
not a provider instance. This allows the framework to create fresh instances as
needed.

---

## 7. Protocol Version 6 vs Protocol Version 5

| Aspect | Protocol v5 | Protocol v6 |
|---|---|---|
| SDKv2 support | Yes | No |
| Plugin Framework support | Yes | **Yes (default)** |
| Nested attributes | No (workarounds) | Native support |
| `terraform-plugin-mux` | Required to mix v5 + v6 | Used standalone |
| Recommendation | Only for SDKv2 compat | **Use for all new providers** |

To explicitly set protocol version 6 (the default):

```go
opts := providerserver.ServeOpts{
    Address:         "registry.terraform.io/sbuglione/comfyui",
    ProtocolVersion: 6,
}
```

If you need to serve **both** protocol v5 (for legacy SDKv2 resources during
migration) and v6, use `terraform-plugin-mux`:

```go
import "github.com/hashicorp/terraform-plugin-mux/tf5to6server"
import "github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
```

This is an advanced pattern used during gradual provider migration and is not
needed for new providers.

---

## 8. Context Usage

Every framework method receives a `context.Context` as its first parameter. This
context carries:

### 8.1 Logging

```go
func (r *WorkflowResource) Create(ctx context.Context,
    req resource.CreateRequest, resp *resource.CreateResponse) {

    tflog.Debug(ctx, "Creating workflow resource")
    tflog.SetField(ctx, "workflow_name", plan.Name.ValueString())
    tflog.Trace(ctx, "API request prepared")
}
```

The `tflog` package reads provider metadata from the context to produce structured
log output. Log levels: `Trace`, `Debug`, `Info`, `Warn`, `Error`.

### 8.2 Cancellation

Terraform Core may cancel a long-running operation (e.g., user presses Ctrl+C).
Always pass the context to API calls:

```go
result, err := r.client.CreateWorkflow(ctx, request)
```

### 8.3 Diagnostics Context

The context is also used internally by the framework for path-aware diagnostics:

```go
resp.Diagnostics.AddAttributeError(
    path.Root("name"),
    "Invalid Workflow Name",
    "Workflow name must be non-empty.",
)
```

---

## 9. Diagnostics (`diag` Package)

Diagnostics are the framework's way of reporting errors and warnings back to
Terraform Core (and ultimately to the user).

```go
// Add an error — halts execution for this resource
resp.Diagnostics.AddError(
    "Unable to Create Workflow",
    fmt.Sprintf("API returned error: %s", err.Error()),
)

// Add a warning — execution continues
resp.Diagnostics.AddWarning(
    "Workflow Deprecated",
    "This workflow version is deprecated and will be removed in a future release.",
)

// Check for errors before continuing
if resp.Diagnostics.HasError() {
    return
}

// Attribute-specific error (shows in plan output next to the attribute)
resp.Diagnostics.AddAttributeError(
    path.Root("node_count"),
    "Invalid Node Count",
    "Node count must be between 1 and 100.",
)

// Append diagnostics from another operation
resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
```

The `Append` pattern with `...` is used throughout the framework because
`Get`, `Set`, and other operations return `diag.Diagnostics` (a slice).

---

## 10. References

| Resource | URL |
|---|---|
| Plugin Framework Documentation | <https://developer.hashicorp.com/terraform/plugin/framework> |
| Plugin Framework Go Docs | <https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework> |
| Provider Interface Docs | <https://developer.hashicorp.com/terraform/plugin/framework/providers> |
| Resource Interface Docs | <https://developer.hashicorp.com/terraform/plugin/framework/resources> |
| Data Source Interface Docs | <https://developer.hashicorp.com/terraform/plugin/framework/data-sources> |
| Handling Data / Types | <https://developer.hashicorp.com/terraform/plugin/framework/handling-data> |
| Schema Documentation | <https://developer.hashicorp.com/terraform/plugin/framework/handling-data/schemas> |
| Diagnostics | <https://developer.hashicorp.com/terraform/plugin/framework/diagnostics> |
| Plan Modifiers | <https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification> |
| Validators | <https://developer.hashicorp.com/terraform/plugin/framework/validation> |
| terraform-plugin-log | <https://developer.hashicorp.com/terraform/plugin/log> |
| terraform-plugin-framework-validators | <https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework-validators> |
