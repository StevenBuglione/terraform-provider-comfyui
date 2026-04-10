# 03 — Provider Implementation (Plugin Framework)

> **Audience**: AI coding harness. This covers the full anatomy of a
> `terraform-plugin-framework` provider. SDKv2 patterns are **not** used.

---

## 1. The `provider.Provider` Interface

```go
type Provider interface {
    Metadata(ctx context.Context, req MetadataRequest, resp *MetadataResponse)
    Schema(ctx context.Context, req SchemaRequest, resp *SchemaResponse)
    Configure(ctx context.Context, req ConfigureRequest, resp *ConfigureResponse)
    Resources(ctx context.Context) []func() resource.Resource
    DataSources(ctx context.Context) []func() datasource.DataSource
}
```

Implement on a concrete struct. The struct holds a version string and (after
`Configure`) an API client.

---

## 2. Method-by-Method Breakdown

### 2.1 `Metadata` — Sets Provider Type Name

The prefix for all resources/data sources. If `TypeName` is `comfyui`, a
resource returning `comfyui_workflow` gets type `comfyui_workflow`.

```go
func (p *comfyuiProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
    resp.TypeName = "comfyui"
}
```

### 2.2 `Schema` — Provider Configuration Block

Defines attributes practitioners set in HCL (endpoint, api_key, etc.).
Import: `github.com/hashicorp/terraform-plugin-framework/provider/schema`
(**not** `resource/schema`).

```go
func (p *comfyuiProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Interact with ComfyUI.",
        Attributes: map[string]schema.Attribute{
            "endpoint": schema.StringAttribute{
                Description: "API endpoint. Also settable via COMFYUI_ENDPOINT.",
                Optional:    true,
            },
            "api_key": schema.StringAttribute{
                Description: "API key. Also settable via COMFYUI_API_KEY.",
                Optional:    true,
                Sensitive:   true,  // redacts from CLI output and logs
            },
        },
    }
}
```

### 2.3 `Configure` — Build API Client

Called once after `Schema`. Reads config → validates → falls back to env vars →
creates client → stores in `resp.ResourceData` / `resp.DataSourceData`.

```go
func (p *comfyuiProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var config comfyuiProviderModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
    if resp.Diagnostics.HasError() { return }

    // Env-var fallback — HCL value wins over env var.
    endpoint := os.Getenv("COMFYUI_ENDPOINT")
    if !config.Endpoint.IsNull() { endpoint = config.Endpoint.ValueString() }
    apiKey := os.Getenv("COMFYUI_API_KEY")
    if !config.APIKey.IsNull() { apiKey = config.APIKey.ValueString() }

    if endpoint == "" {
        resp.Diagnostics.AddAttributeError(path.Root("endpoint"),
            "Missing ComfyUI API Endpoint",
            "Set in provider block or via COMFYUI_ENDPOINT env var.")
        return
    }

    client, err := comfyui.NewClient(endpoint, apiKey)
    if err != nil {
        resp.Diagnostics.AddError("Unable to Create API Client", err.Error())
        return
    }
    resp.DataSourceData = client
    resp.ResourceData = client
}
```

### 2.4 `Resources` & `DataSources` — Constructor Lists

Return slices of **factory functions**. Each returns a new zero-value struct.

```go
func (p *comfyuiProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{ NewWorkflowResource, NewServerResource }
}
func (p *comfyuiProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{ NewWorkflowDataSource }
}
```

---

## 3. Provider Model Struct

Maps schema attributes to Go types. `tfsdk` tag must exactly match the key.

```go
type comfyuiProviderModel struct {
    Endpoint types.String `tfsdk:"endpoint"`
    APIKey   types.String `tfsdk:"api_key"`
}
```

Always use `types.*` — never raw Go primitives. Raw types cannot represent
Terraform's null/unknown semantics.

---

## 4. Provider Struct & Constructor

`New` returns a **factory** (`func() provider.Provider`), which is what
`providerserver.Serve` expects.

```go
type comfyuiProvider struct {
    version string
}

func New(version string) func() provider.Provider {
    return func() provider.Provider { return &comfyuiProvider{version: version} }
}
```

---

## 5. Passing API Client to Resources

1. Provider `Configure` stores client in `resp.ResourceData`.
2. Resource `Configure` type-asserts `req.ProviderData`.

```go
type workflowResource struct { client *comfyui.Client }

func (r *workflowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil { return } // nil during ValidateConfig phase
    client, ok := req.ProviderData.(*comfyui.Client)
    if !ok {
        resp.Diagnostics.AddError("Unexpected Type", fmt.Sprintf("Expected *comfyui.Client, got %T", req.ProviderData))
        return
    }
    r.client = client
}
```

---

## 6. Env-Var Fallback & Unknown Handling

**Priority:** HCL value > env var > hard-coded default.

- `IsNull()` — attribute omitted or set to `null`.
- `IsUnknown()` — depends on a not-yet-created resource (rare for provider config).

If unknown at plan time, do **not** fall back — add a diagnostic instead:

```go
if config.Endpoint.IsUnknown() {
    resp.Diagnostics.AddAttributeWarning(path.Root("endpoint"),
        "Unknown Endpoint", "Cannot configure when endpoint is not yet known.")
    return
}
```

---

## 7. Complete `main.go`

```go
package main

import (
    "context"
    "flag"
    "log"
    "github.com/hashicorp/terraform-plugin-framework/providerserver"
    "github.com/yourorg/terraform-provider-comfyui/internal/provider"
)

var version string = "dev" // overridden via: go build -ldflags="-X main.version=1.2.3"

func main() {
    var debug bool
    flag.BoolVar(&debug, "debug", false, "enable debugger support (delve)")
    flag.Parse()
    opts := providerserver.ServeOpts{
        Address: "registry.terraform.io/yourorg/comfyui",
        Debug:   debug,
    }
    err := providerserver.Serve(context.Background(), provider.New(version), opts)
    if err != nil { log.Fatal(err.Error()) }
}
```

`Address` format: `registry.terraform.io/<namespace>/<type>`.
When `Debug` is true, the provider prints a `TF_REATTACH_PROVIDERS` value to
stderr for attaching Delve.

---

## 8. Full Working `provider.go`

```go
package provider

import (
    "context"
    "os"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/provider"
    "github.com/hashicorp/terraform-plugin-framework/provider/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/yourorg/terraform-provider-comfyui/internal/client"
)

var _ provider.Provider = &comfyuiProvider{} // compile-time check

type comfyuiProviderModel struct {
    Endpoint types.String `tfsdk:"endpoint"`
    APIKey   types.String `tfsdk:"api_key"`
}

type comfyuiProvider struct{ version string }

func New(version string) func() provider.Provider {
    return func() provider.Provider { return &comfyuiProvider{version: version} }
}

func (p *comfyuiProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
    resp.TypeName = "comfyui"
}

func (p *comfyuiProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Manage ComfyUI resources.",
        Attributes: map[string]schema.Attribute{
            "endpoint": schema.StringAttribute{
                Description: "ComfyUI API endpoint. Also via COMFYUI_ENDPOINT env var.",
                Optional: true,
            },
            "api_key": schema.StringAttribute{
                Description: "API key. Also via COMFYUI_API_KEY env var.",
                Optional: true, Sensitive: true,
            },
        },
    }
}

func (p *comfyuiProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var config comfyuiProviderModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
    if resp.Diagnostics.HasError() { return }

    if config.Endpoint.IsUnknown() {
        resp.Diagnostics.AddAttributeError(path.Root("endpoint"), "Unknown Endpoint", "Not yet known.")
    }
    if config.APIKey.IsUnknown() {
        resp.Diagnostics.AddAttributeError(path.Root("api_key"), "Unknown API Key", "Not yet known.")
    }
    if resp.Diagnostics.HasError() { return }

    endpoint := os.Getenv("COMFYUI_ENDPOINT")
    if !config.Endpoint.IsNull() { endpoint = config.Endpoint.ValueString() }
    apiKey := os.Getenv("COMFYUI_API_KEY")
    if !config.APIKey.IsNull() { apiKey = config.APIKey.ValueString() }

    if endpoint == "" {
        resp.Diagnostics.AddAttributeError(path.Root("endpoint"),
            "Missing Endpoint", "Set in provider block or COMFYUI_ENDPOINT env var.")
        return
    }

    c, err := client.NewComfyUIClient(endpoint, apiKey)
    if err != nil {
        resp.Diagnostics.AddError("Unable to Create API Client", err.Error())
        return
    }
    resp.DataSourceData = c
    resp.ResourceData = c
}

func (p *comfyuiProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{ NewWorkflowResource, NewServerResource }
}

func (p *comfyuiProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{ NewWorkflowDataSource }
}
```

---

## 9. Compile-Time Interface Checks

Always verify interface satisfaction at compile time with blank-identifier
assignments. Add one per optional interface:

```go
var _ provider.Provider              = &comfyuiProvider{}
var _ provider.ProviderWithFunctions = &comfyuiProvider{} // if you add functions
```

---

## References

| Source | URL |
|---|---|
| Framework — Provider docs | https://developer.hashicorp.com/terraform/plugin/framework/providers |
| Framework — Configure method | https://developer.hashicorp.com/terraform/plugin/framework/providers#configure-method |
| Framework — Handling Data | https://developer.hashicorp.com/terraform/plugin/framework/handling-data |
| Framework — providerserver pkg | https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework/providerserver |
| Tutorial — Implement a provider | https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-provider |
| Source — terraform-plugin-framework | https://github.com/hashicorp/terraform-plugin-framework |
