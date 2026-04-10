# Terraform Provider: Overview and Architecture

## 1. What Is a Terraform Provider?

A Terraform **provider** is a standalone binary (written in Go) that acts as a bridge
between Terraform Core and a specific upstream API. Providers are responsible for:

| Responsibility | Description |
|---|---|
| **Authentication** | Managing credentials and sessions with the target API |
| **CRUD operations** | Creating, reading, updating, and deleting infrastructure objects |
| **Schema definition** | Declaring the shape of every resource and data source the provider exposes |
| **State mapping** | Translating between Terraform's state model and the API's object model |
| **Plan diffing** | Computing what must change to move from the current state to the desired state |

Every cloud, SaaS product, or internal tool that Terraform manages has at least one
provider. The Terraform Registry (<https://registry.terraform.io>) hosts thousands of
them — from first-party providers like `hashicorp/aws` to community providers.

### Provider Naming Convention

Provider binaries follow the naming pattern:

```
terraform-provider-<NAME>
```

For example, a provider for ComfyUI would produce a binary named
`terraform-provider-comfyui`. Terraform relies on this naming convention for
discovery and execution.

---

## 2. Plugin System Architecture

### 2.1 Terraform Core ↔ Provider Communication

Terraform uses a **client-server** architecture where:

- **Terraform Core** is the CLI/engine that parses HCL, builds the dependency graph,
  and orchestrates operations.
- **Provider binaries** are separate OS processes that Terraform Core launches and
  communicates with over **gRPC**.

```
┌─────────────────┐         gRPC (protocol v6)         ┌─────────────────────┐
│                 │ ◄──────────────────────────────────► │                     │
│  Terraform Core │         localhost / unix socket      │  Provider Binary    │
│  (CLI engine)   │                                      │  (Go process)       │
│                 │  GetProviderSchema                   │                     │
│                 │  ValidateProviderConfig               │  terraform-provider │
│                 │  ConfigureProvider                    │  -comfyui           │
│                 │  PlanResourceChange                  │                     │
│                 │  ApplyResourceChange                 │                     │
│                 │  ReadResource                        │                     │
│                 │  ReadDataSource                      │                     │
└─────────────────┘                                      └─────────────────────┘
```

Key gRPC service methods (protocol v6):

| Method | Purpose |
|---|---|
| `GetProviderSchema` | Returns the full schema for the provider, its resources, and data sources |
| `ValidateProviderConfig` | Validates provider-level configuration |
| `ConfigureProvider` | Passes resolved configuration to the provider (e.g., API keys) |
| `ValidateResourceConfig` | Validates a specific resource block |
| `PlanResourceChange` | Computes the diff/plan for a resource |
| `ApplyResourceChange` | Executes the planned create/update/delete |
| `ReadResource` | Refreshes the current state of a resource from the API |
| `ReadDataSource` | Reads data from the API for a data source |
| `ImportResourceState` | Imports an existing resource into Terraform state |
| `UpgradeResourceState` | Migrates state from an older schema version |
| `StopProvider` | Signals the provider to gracefully shut down |

### 2.2 go-plugin and HashiCorp's Plugin Library

The gRPC transport is managed by HashiCorp's **go-plugin** library
(<https://github.com/hashicorp/go-plugin>). This library handles:

- Launching the provider binary as a child process
- Negotiating the protocol version
- Setting up a gRPC server inside the provider process
- Health-checking the connection

Provider authors never interact with go-plugin directly — the Plugin Framework
abstracts it away behind `providerserver.Serve()`.

### 2.3 Protocol Versions

| Protocol | Status | Notes |
|---|---|---|
| **v6** | **Recommended** | Current default for Plugin Framework providers |
| v5 | Legacy / supported | Required for SDKv2 providers; Plugin Framework also supports it for mixed-protocol setups via `terraform-plugin-mux` |

Always target **protocol v6** unless you have a specific reason to support v5.

---

## 3. How Terraform Discovers and Launches Providers

Terraform resolves providers through multiple mechanisms, in priority order:

### 3.1 Dev Overrides (Development)

During development, you configure `~/.terraformrc` (or `%APPDATA%/terraform.rc` on
Windows) to point Terraform at your locally compiled binary:

```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "registry.terraform.io/sbuglione/comfyui" = "/home/sbuglione/go/bin"
  }

  direct {}
}
```

With `dev_overrides`, Terraform **skips** `terraform init` and loads the provider
directly from the given path. This is the primary workflow during development.

### 3.2 Terraform Registry (Production)

In production, users declare a `required_providers` block:

```hcl
terraform {
  required_providers {
    comfyui = {
      source  = "sbuglione/comfyui"
      version = "~> 1.0"
    }
  }
}
```

`terraform init` contacts the Terraform Registry API at
`https://registry.terraform.io`, downloads the appropriate binary for the user's
OS/architecture, and caches it in `.terraform/providers/`.

### 3.3 Local Filesystem Mirror

For air-gapped or enterprise environments, providers can be served from a local
filesystem mirror:

```hcl
provider_installation {
  filesystem_mirror {
    path    = "/usr/share/terraform/providers"
    include = ["sbuglione/comfyui"]
  }

  direct {
    exclude = ["sbuglione/comfyui"]
  }
}
```

### 3.4 Network Mirror

Similar to filesystem mirrors but served over HTTPS:

```hcl
provider_installation {
  network_mirror {
    url = "https://terraform.internal.example.com/providers/"
  }
}
```

---

## 4. Plugin Framework vs SDKv2

### Historical Context: SDKv2

The original Terraform Plugin SDK (often called "SDKv2") was the standard way to
build providers from Terraform 0.12 through much of the 1.x lifecycle. It used
`*schema.Resource` structs with map-based schemas:

```go
// SDKv2 pattern — DO NOT use for new providers
func resourceExample() *schema.Resource {
    return &schema.Resource{
        CreateContext: resourceExampleCreate,
        ReadContext:   resourceExampleRead,
        UpdateContext: resourceExampleUpdate,
        DeleteContext: resourceExampleDelete,
        Schema: map[string]*schema.Schema{
            "name": {
                Type:     schema.TypeString,
                Required: true,
            },
        },
    }
}
```

SDKv2 had several limitations:
- Schema defined as runtime maps — no compile-time safety
- Flat `map[string]interface{}` for reading/writing attributes
- Difficult to express nested objects and complex types
- Testing tied to the binary test driver

### The Plugin Framework (Recommended)

The **Plugin Framework** (`terraform-plugin-framework`) is the modern replacement.
It uses Go interfaces, strongly typed attributes, and a cleaner separation of
concerns:

```go
// Plugin Framework pattern — use this
type ExampleResource struct {
    client *api.Client
}

type ExampleResourceModel struct {
    ID   types.String `tfsdk:"id"`
    Name types.String `tfsdk:"name"`
}

func (r *ExampleResource) Schema(ctx context.Context, req resource.SchemaRequest,
    resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed: true,
            },
            "name": schema.StringAttribute{
                Required: true,
            },
        },
    }
}
```

| Feature | SDKv2 | Plugin Framework |
|---|---|---|
| Schema definition | Runtime maps | Strongly typed Go structs |
| Attribute access | `d.Get("name").(string)` | `model.Name.ValueString()` |
| Null/Unknown handling | Implicit, error-prone | Explicit `IsNull()`, `IsUnknown()` |
| Nested objects | Awkward `schema.TypeList` workarounds | First-class `schema.SingleNestedAttribute` |
| Diagnostics | `diag.Diagnostics` (basic) | `diag.Diagnostics` (rich, path-aware) |
| Plan modification | `DiffSuppressFunc` | `planmodifier` package |
| Validators | `ValidateFunc` | Composable `validator` package |
| Protocol support | v5 only | v5 and v6 |
| Status | Maintenance mode | **Actively developed** |

**All new providers should use the Plugin Framework exclusively.**

---

## 5. Key Go Modules

| Module | Import Path | Purpose |
|---|---|---|
| **terraform-plugin-framework** | `github.com/hashicorp/terraform-plugin-framework` | Core framework: schema, types, CRUD interfaces |
| **terraform-plugin-go** | `github.com/hashicorp/terraform-plugin-go` | Low-level protocol bindings; rarely used directly |
| **terraform-plugin-testing** | `github.com/hashicorp/terraform-plugin-testing` | Acceptance test harness (`resource.Test`, `TestStep`) |
| **terraform-plugin-log** | `github.com/hashicorp/terraform-plugin-log` | Structured logging (`tflog.Debug`, `tflog.Warn`, etc.) |
| **terraform-plugin-docs** | `github.com/hashicorp/terraform-plugin-docs` | Documentation generator (`tfplugindocs`) |
| **terraform-plugin-mux** | `github.com/hashicorp/terraform-plugin-mux` | Combine multiple provider servers (useful for SDKv2→Framework migration) |

Typical `go.mod` requires block:

```go
require (
    github.com/hashicorp/terraform-plugin-framework v1.13.0
    github.com/hashicorp/terraform-plugin-go        v0.25.0
    github.com/hashicorp/terraform-plugin-testing    v1.11.0
    github.com/hashicorp/terraform-plugin-log        v0.9.0
)
```

> **Note:** Pin to the latest stable versions at time of development. Check the
> Go module proxy at <https://pkg.go.dev> for current releases.

---

## 6. Provider Lifecycle

A Terraform run progresses through well-defined phases. The provider participates
in each one:

### 6.1 Init (`terraform init`)

- Terraform resolves the provider source and version constraint
- Downloads the binary (or uses dev_overrides)
- Stores the binary in `.terraform/providers/`
- The provider is **not** executed during init

### 6.2 Plan (`terraform plan`)

1. Terraform launches the provider binary (gRPC server starts)
2. Calls `GetProviderSchema` to learn the schema
3. Calls `ConfigureProvider` with the resolved provider config
4. For each resource in the config:
   - Calls `ReadResource` to refresh current state
   - Calls `PlanResourceChange` to compute the diff
5. Displays the execution plan to the user
6. Provider process exits

### 6.3 Apply (`terraform apply`)

1. Provider binary is launched again
2. `ConfigureProvider` is called again
3. For each resource that changed:
   - `ApplyResourceChange` is called with the planned diff
   - The provider calls the upstream API (create / update / delete)
   - Returns the new state to Terraform
4. Terraform writes the updated state file
5. Provider process exits

### 6.4 Destroy (`terraform destroy`)

- Same flow as apply, but every resource receives a delete plan
- Resources are destroyed in reverse dependency order
- State file is updated to remove destroyed resources

### Lifecycle Diagram

```
terraform init     terraform plan          terraform apply         terraform destroy
     │                  │                       │                       │
     ▼                  ▼                       ▼                       ▼
 Download           Launch provider         Launch provider         Launch provider
 provider           GetProviderSchema       ConfigureProvider       ConfigureProvider
 binary             ConfigureProvider       ApplyResourceChange     ApplyResourceChange
                    ReadResource              (create/update)         (delete all)
                    PlanResourceChange      Write state file        Clear state file
                    Show plan               Exit                    Exit
                    Exit
```

---

## 7. How State Files Work

Terraform state (`terraform.tfstate`) is the single source of truth for what
Terraform believes exists in the real world.

### 7.1 Structure

State is a JSON document containing:

```json
{
  "version": 4,
  "terraform_version": "1.9.0",
  "serial": 12,
  "lineage": "a1b2c3d4-...",
  "outputs": { },
  "resources": [
    {
      "mode": "managed",
      "type": "comfyui_workflow",
      "name": "my_workflow",
      "provider": "registry.terraform.io/sbuglione/comfyui",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "wf-12345",
            "name": "My Workflow"
          }
        }
      ]
    }
  ]
}
```

### 7.2 Key Concepts

| Concept | Description |
|---|---|
| **Serial** | Monotonically increasing counter; prevents concurrent writes |
| **Lineage** | UUID assigned on first `terraform apply`; prevents mixing states |
| **Schema version** | Per-resource version for state migrations |
| **Sensitive values** | Marked in state but still stored (consider remote backends with encryption) |

### 7.3 Backends

State can be stored:
- **Locally** — default `terraform.tfstate` file
- **Remotely** — S3, GCS, Azure Blob, Terraform Cloud, Consul, etc.

Remote backends enable team collaboration, state locking, and encryption at rest.

### 7.4 Provider's Role

The provider does **not** manage the state file directly. It:
1. Returns attribute values from `Read`, `Create`, `Update` operations
2. Terraform Core serializes those values into the state file
3. On the next run, Terraform deserializes state and passes it back to the provider

---

## 8. References

| Resource | URL |
|---|---|
| Terraform Plugin Development | <https://developer.hashicorp.com/terraform/plugin> |
| Plugin Framework Documentation | <https://developer.hashicorp.com/terraform/plugin/framework> |
| Plugin Framework GitHub Repository | <https://github.com/hashicorp/terraform-plugin-framework> |
| Plugin Framework Go Docs | <https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework> |
| Terraform Plugin Protocol | <https://developer.hashicorp.com/terraform/plugin/terraform-plugin-protocol> |
| go-plugin Library | <https://github.com/hashicorp/go-plugin> |
| Terraform Registry | <https://registry.terraform.io> |
| Terraform Registry Publishing | <https://developer.hashicorp.com/terraform/registry/providers/publishing> |
| Plugin Framework Tutorials | <https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework> |
| terraform-plugin-testing | <https://pkg.go.dev/github.com/hashicorp/terraform-plugin-testing> |
| terraform-plugin-log | <https://pkg.go.dev/github.com/hashicorp/terraform-plugin-log> |
| terraform-plugin-docs | <https://github.com/hashicorp/terraform-plugin-docs> |
| terraform-plugin-mux | <https://pkg.go.dev/github.com/hashicorp/terraform-plugin-mux> |
