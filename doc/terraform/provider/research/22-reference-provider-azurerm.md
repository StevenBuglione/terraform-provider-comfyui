# 22 — Reference Provider: terraform-provider-azurerm

> **Purpose**: Analyze the architecture, layered design, and key patterns of
> Azure's Terraform provider as a second reference point for our provider.

---

## 1. Overview

`terraform-provider-azurerm` is HashiCorp's official provider for Microsoft
Azure. It emphasizes **typed resources**, **layered architecture**, and a custom
**auto-generated SDK** (`go-azure-sdk`).

| Metric | Value |
|---|---|
| Resources | 900+ |
| Data Sources | 400+ |
| Go SDK | `go-azure-sdk` (auto-generated typed ARM client) |
| Plugin Framework | Plugin SDKv2 (Framework migration in progress) |
| Repository | <https://github.com/hashicorp/terraform-provider-azurerm> |
| Dev Docs | <https://hashicorp.github.io/terraform-provider-azurerm/> |

---

## 2. Layered Architecture

```
┌─────────────────────────────────────────┐
│  Layer 1: Provider                      │
│  internal/provider/ — schema, config,   │
│  service registration loop              │
├─────────────────────────────────────────┤
│  Layer 2: Services                      │
│  internal/services/<service>/ — CRUD    │
│  logic, validation, business rules      │
├─────────────────────────────────────────┤
│  Layer 3: Clients                       │
│  internal/clients/ — API client factory │
│  per-service client construction        │
├─────────────────────────────────────────┤
│  Layer 4: SDK                           │
│  go-azure-sdk — auto-generated typed    │
│  ARM clients, models, LRO polling       │
└─────────────────────────────────────────┘
```

### Service Package Structure

```
internal/services/compute/
├── registration.go                     # ServiceRegistration implementation
├── virtual_machine_resource.go         # Resource CRUD
├── virtual_machine_resource_test.go    # Acceptance tests
├── virtual_machine_data_source.go      # Data source
├── client/
│   └── client.go                       # Service-specific API client struct
├── parse/
│   └── virtual_machine.go              # Resource ID parser
└── validate/
    └── virtual_machine.go              # Service-specific validators
```

---

## 3. Key Patterns

### 3.1 Registration Pattern

Every service package exposes a `Registration` type. The provider loops over
these at startup — no central file needs editing when a new service is added:

```go
// internal/services/compute/registration.go
type Registration struct{}

func (r Registration) Name() string { return "Compute" }

func (r Registration) Resources() []sdk.Resource {
    return []sdk.Resource{VirtualMachineResource{}}
}

func (r Registration) DataSources() []sdk.DataSource {
    return []sdk.DataSource{VirtualMachineDataSource{}}
}
```

### 3.2 TypedResource Interface

Resources implement a standard interface that enforces consistent CRUD:

```go
type TypedResource interface {
    Arguments() map[string]*schema.Schema
    Attributes() map[string]*schema.Schema
    ResourceType() string
    Create() sdk.ResourceFunc
    Read() sdk.ResourceFunc
    Update() sdk.ResourceFunc
    Delete() sdk.ResourceFunc
    IDValidationFunc() pluginsdk.SchemaValidateFunc
}
```

A concrete Create implementation:

```go
func (r VirtualMachineResource) Create() sdk.ResourceFunc {
    return sdk.ResourceFunc{
        Timeout: 45 * time.Minute,
        Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
            client := metadata.Client.Compute.VirtualMachinesClient
            var model VirtualMachineModel
            if err := metadata.Decode(&model); err != nil {
                return fmt.Errorf("decoding: %+v", err)
            }
            id := virtualmachines.NewVirtualMachineID(
                metadata.Client.Account.SubscriptionId,
                model.ResourceGroupName, model.Name,
            )
            input := virtualmachines.VirtualMachine{Location: model.Location}
            if err := client.CreateOrUpdateThenPoll(ctx, id, input); err != nil {
                return fmt.Errorf("creating %s: %+v", id, err)
            }
            metadata.SetID(id)
            return nil
        },
    }
}
```

### 3.3 Layered Client Construction

The client layer builds authenticated API clients per service:

```go
// internal/clients/client.go
type Client struct {
    Compute *compute.Client
    Network *network.Client
    Storage *storage.Client
}

// internal/services/compute/client/client.go
type Client struct {
    VirtualMachinesClient *virtualmachines.VirtualMachinesClient
}
```

Resources receive a pre-configured client via `metadata.Client` — they never
construct their own.

### 3.4 Typed Resource ID Parsing

Azure resource IDs follow a standard URI format. The SDK provides typed parsers:

```go
id, err := virtualmachines.ParseVirtualMachineID(d.Id())
// id.SubscriptionId, id.ResourceGroupName, id.VirtualMachineName
```

This eliminates string splitting. For our ComfyUI provider, we should define
typed ID structs for each resource.

### 3.5 Feature Toggles and Resource Locks

The provider supports a `features {}` block for toggling behavior — not needed
for new providers initially. It also uses in-process **resource locks** keyed by
ID to prevent Azure API conflicts under Terraform's parallelism:

```go
locks.ByID(id.ID())
defer locks.UnlockByID(id.ID())
```

---

## 4. Testing Patterns

```go
func TestAccVirtualMachine_basic(t *testing.T) {
    data := acceptance.BuildTestData(t, "azurerm_virtual_machine", "test")
    r := VirtualMachineResource{}
    data.ResourceTest(t, r, []acceptance.TestStep{
        {
            Config: r.basicConfig(data),
            Check: acceptance.ComposeTestCheckFunc(
                check.That(data.ResourceName).ExistsInAzure(r),
            ),
        },
        data.ImportStep(),
    })
}
```

Key patterns: `BuildTestData` generates random names, `data.ImportStep()`
auto-tests import, and each resource implements an `Exists` function:

```go
func (r VirtualMachineResource) Exists(ctx context.Context, client *clients.Client,
    state *pluginsdk.InstanceState) (*bool, error) {
    id, err := virtualmachines.ParseVirtualMachineID(state.ID)
    if err != nil {
        return nil, err
    }
    resp, err := client.Compute.VirtualMachinesClient.Get(ctx, *id,
        virtualmachines.DefaultGetOperationOptions())
    if err != nil {
        return nil, fmt.Errorf("retrieving %s: %+v", id, err)
    }
    return utils.Bool(resp.Model != nil), nil
}
```

---

## 5. Comparison: AWS vs AzureRM

| Aspect | AWS Provider | AzureRM Provider |
|---|---|---|
| Client management | Single `AWSClient` struct | Layered `Client` → service `Client` |
| SDK | Official `aws-sdk-go-v2` | Custom `go-azure-sdk` |
| Resource definition | Functions → `*schema.Resource` | Structs implementing `TypedResource` |
| ID handling | String-based (ARNs) | Typed ID structs with parsers |
| Registration | `ServicePackage` with factory lists | `Registration` interface |

Both converge on **service-per-package with registration-based discovery**. The
AzureRM style is easier to follow for new providers.

---

## 6. What to Learn for New Providers

| AzureRM Pattern | Applicability |
|---|---|
| Layered architecture | ✅ Even 2 layers (provider + services) helps |
| Service registration | ✅ Avoids a massive switch; scales cleanly |
| TypedResource interface | ✅ Standardize CRUD signatures |
| Typed resource ID parsing | ✅ Define ID types per resource |
| Feature toggles | ❌ Not needed initially |
| Resource locks | ⚠️ Only if API has concurrency issues |
| Import step in every test | ✅ Test import from day one |

### Core Lessons

1. **Define a common resource interface** for consistent CRUD signatures.
2. **Use typed IDs** — don't pass raw strings. Define `Parse` and `ID()`.
3. **Separate client construction from resource logic**.
4. **Test import from day one** — include an import step in every test.
5. **Registration-based discovery** — let packages register themselves.

---

## 7. References

- **Repository**: <https://github.com/hashicorp/terraform-provider-azurerm>
- **Developer Docs**: <https://hashicorp.github.io/terraform-provider-azurerm/>
- **go-azure-sdk**: <https://github.com/hashicorp/go-azure-sdk>
- **go-azure-helpers**: <https://github.com/hashicorp/go-azure-helpers>
- **Typed Resource Guide**: <https://hashicorp.github.io/terraform-provider-azurerm/guide/typed-resources/>

---

*Previous*: [21-reference-provider-aws.md](21-reference-provider-aws.md)
*Next*: [23-makefile-and-dev-commands.md](23-makefile-and-dev-commands.md)
