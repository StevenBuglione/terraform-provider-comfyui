# Advanced Terraform Plugin Framework Patterns

> Research reference for the AI coding harness. All examples use the
> **Plugin Framework** (`terraform-plugin-framework`), never SDKv2.

---

## 1. Custom Types

**Why:** Schema primitives (`types.String`, etc.) carry no domain semantics. A custom type attaches domain-specific validation, parsing, and **semantic equality** so every attribute of that type inherits the behaviour. Examples: ARNs, timestamps, case-insensitive identifiers.

A custom type has two parts: a **Type** (implements `basetypes.StringTypable`) that describes the type and produces values, and a **Value** (implements `basetypes.StringValuable`) that holds data and implements semantic equality. Wire it in via `CustomType`:

```go
schema.StringAttribute{
    CustomType: CaseInsensitiveStringType{},
    Required:   true,
}
```

**Semantic equality** lets two textually different values compare as equal so Terraform suppresses a spurious diff. For example `"MyWorkflow"` and `"myworkflow"` are semantically equal under case-insensitive comparison. The framework calls `StringSemanticEquals` on the proposed new value, passing the prior value — return `true` to suppress the diff.

### Complete Go Example

```go
package provider

import (
    "context"
    "strings"

    "github.com/hashicorp/terraform-plugin-framework/attr"
    "github.com/hashicorp/terraform-plugin-framework/diag"
    "github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Type
type CaseInsensitiveStringType struct{ basetypes.StringType }

func (t CaseInsensitiveStringType) Equal(o attr.Type) bool {
    other, ok := o.(CaseInsensitiveStringType)
    if !ok { return false }
    return t.StringType.Equal(other.StringType)
}

func (t CaseInsensitiveStringType) ValueFromString(
    ctx context.Context, in basetypes.StringValue,
) (basetypes.StringValuable, diag.Diagnostics) {
    return CaseInsensitiveStringValue{StringValue: in}, nil
}

func (t CaseInsensitiveStringType) ValueType(_ context.Context) attr.Value {
    return CaseInsensitiveStringValue{}
}

// Value
type CaseInsensitiveStringValue struct{ basetypes.StringValue }

func (v CaseInsensitiveStringValue) Type(_ context.Context) attr.Type {
    return CaseInsensitiveStringType{}
}

func (v CaseInsensitiveStringValue) StringSemanticEquals(
    _ context.Context, newValuable basetypes.StringValuable,
) (bool, diag.Diagnostics) {
    newValue, ok := newValuable.(CaseInsensitiveStringValue)
    if !ok { return false, nil }
    return strings.EqualFold(v.ValueString(), newValue.ValueString()), nil
}
```

> **Ref:** <https://developer.hashicorp.com/terraform/plugin/framework/handling-data/types/custom>

---

## 2. Ephemeral Resources (Terraform 1.10+)

**Purpose:** An ephemeral resource produces a value **never written to state or plan**. Each `terraform apply` re-derives it; it exists only for that run. Use cases: temporary auth tokens, runtime-only credentials.

### HCL

```hcl
ephemeral "comfyui_session_token" "this" {
  api_endpoint = "https://comfy.example.com"
}

resource "comfyui_workflow" "main" {
  auth_token = ephemeral.comfyui_session_token.this.token
}
```

### Lifecycle

| Method | When | Purpose |
|--------|------|---------|
| `Open` | Each apply | Produce the ephemeral result. |
| `Renew` | Before expiry during long applies | Refresh a token; return new `RenewAt`. |
| `Close` | End of apply | Clean up (e.g., revoke token). |

### Go Example

```go
var _ ephemeral.EphemeralResource = &SessionTokenEphemeral{}
var _ ephemeral.EphemeralResourceWithRenew = &SessionTokenEphemeral{}
var _ ephemeral.EphemeralResourceWithClose = &SessionTokenEphemeral{}

type SessionTokenEphemeral struct{ client *ComfyUIClient }

type SessionTokenModel struct {
    APIEndpoint types.String `tfsdk:"api_endpoint"`
    Token       types.String `tfsdk:"token"`
}

func (e *SessionTokenEphemeral) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_session_token"
}

func (e *SessionTokenEphemeral) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "api_endpoint": schema.StringAttribute{Required: true},
            "token":        schema.StringAttribute{Computed: true, Sensitive: true},
        },
    }
}

func (e *SessionTokenEphemeral) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
    var data SessionTokenModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() { return }

    token, err := e.client.CreateSessionToken(ctx, data.APIEndpoint.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Token creation failed", err.Error())
        return
    }
    data.Token = types.StringValue(token)
    resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
    resp.RenewAt = time.Now().Add(55 * time.Minute)
}

func (e *SessionTokenEphemeral) Renew(ctx context.Context, req ephemeral.RenewRequest, resp *ephemeral.RenewResponse) {
    resp.RenewAt = time.Now().Add(55 * time.Minute)
}

func (e *SessionTokenEphemeral) Close(ctx context.Context, req ephemeral.CloseRequest, resp *ephemeral.CloseResponse) {
    _ = e.client.RevokeSessionToken(ctx)
}
```

Register: return it from `provider.EphemeralResources()`.

> **Ref:** <https://developer.hashicorp.com/terraform/plugin/framework/ephemeral-resources>

---

## 3. Write-Only Attributes (Terraform 1.11+)

**Purpose:** A config value that is **never persisted to state or plan**. After apply, it is gone.
**Use cases:** passwords, API keys, one-time bootstrap secrets.

**Key rules:**

- **Cannot combine with `Computed`** — a write-only attribute has no stored value, so nothing to compute. Marking both is a schema error.
- **Read from `req.Config` only** — the value is null in `req.Plan` and `req.State`. It is only available during apply from the configuration.
- **Always null in state** — subsequent plans see null in prior state.

### Go Example

```go
// Schema
"secret_key": schema.StringAttribute{
    WriteOnly:   true,
    Required:    true,
    Description: "Never stored in state.",
},

// Create — read from Config, not Plan
func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var config APIKeyResourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
    if resp.Diagnostics.HasError() { return }

    secret := config.SecretKey.ValueString() // available here
    id, err := r.client.RegisterKey(ctx, config.Name.ValueString(), secret)
    if err != nil {
        resp.Diagnostics.AddError("Create failed", err.Error())
        return
    }

    var state APIKeyResourceModel
    state.ID = types.StringValue(id)
    state.Name = config.Name
    // Do NOT set state.SecretKey — must remain null in state.
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

> **Ref:** <https://developer.hashicorp.com/terraform/plugin/framework/resources/write-only-attributes>

---

## 4. Private State

**Purpose:** Store internal provider data that is opaque to practitioners. Persisted across CRUD calls in the state file but never shown in CLI output, plans, or diffs.
**Use cases:** ETags for optimistic concurrency, internal revision counters, API version tracking.

### API — `GetKey` / `SetKey`

```go
// Read a key (returns []byte)
etagBytes, diags := req.Private.GetKey(ctx, "etag")

// Write a key
diags := resp.Private.SetKey(ctx, "etag", []byte(newEtag))
```

### Go Example — Read & Update with ETag

```go
func (r *WorkflowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state WorkflowResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() { return }

    workflow, etag, err := r.client.GetWorkflow(ctx, state.ID.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Read failed", err.Error())
        return
    }
    state.Name = types.StringValue(workflow.Name)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
    resp.Diagnostics.Append(resp.Private.SetKey(ctx, "etag", []byte(etag))...)
}

func (r *WorkflowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan WorkflowResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() { return }

    etagBytes, diags := req.Private.GetKey(ctx, "etag")
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() { return }

    newEtag, err := r.client.UpdateWorkflow(ctx, plan.ID.ValueString(), plan.Name.ValueString(), string(etagBytes))
    if err != nil {
        resp.Diagnostics.AddError("Update failed", err.Error())
        return
    }
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
    resp.Diagnostics.Append(resp.Private.SetKey(ctx, "etag", []byte(newEtag))...)
}
```

> **Ref:** <https://developer.hashicorp.com/terraform/plugin/framework/resources/private-state>

---

## 5. Resource Timeouts

**Purpose:** User-configurable timeouts for long-running operations (image generation, model loading). Uses the `terraform-plugin-framework-timeouts` module.

```
go get github.com/hashicorp/terraform-plugin-framework-timeouts
```

### HCL

```hcl
resource "comfyui_workflow_run" "render" {
  workflow_id = comfyui_workflow.main.id
  timeouts { create = "30m"; delete = "10m" }
}
```

### Go Example

```go
import "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

type WorkflowRunModel struct {
    ID         types.String   `tfsdk:"id"`
    WorkflowID types.String   `tfsdk:"workflow_id"`
    Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

// In Schema — add as a block:
"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Delete: true}),

// In Create — read and apply:
func (r *WorkflowRunResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan WorkflowRunModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() { return }

    createTimeout, diags := plan.Timeouts.Create(ctx, 20*time.Minute) // default 20m
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() { return }

    ctx, cancel := context.WithTimeout(ctx, createTimeout)
    defer cancel()

    id, err := r.client.RunWorkflow(ctx, plan.WorkflowID.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Create timed out or failed", err.Error())
        return
    }
    plan.ID = types.StringValue(id)
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

> **Ref:** <https://developer.hashicorp.com/terraform/plugin/framework/resources/timeouts>

---

## 6. Sensitive Attributes

`Sensitive: true` tells Terraform to **redact** the value in CLI output and flag it in state. The value **is still stored in state** — it is just hidden from terminal output.

### Sensitive vs Write-Only

| Kind | In state? | In CLI? | When to use |
|------|-----------|---------|-------------|
| `Sensitive: true` | Yes | Redacted | Provider needs to read the value back later (e.g., generated password passed to other resources). |
| `WriteOnly: true` | No | No | Provider only consumes it once during apply; must never persist. |

### Go Example

```go
"password": schema.StringAttribute{
    Required:  true,
    Sensitive: true,
},
"api_key": schema.StringAttribute{
    Computed:  true,
    Sensitive: true,
},
```

Plan output renders as:

```
+ password  = (sensitive value)
+ api_key   = (sensitive value)
```

Ensure the backend encrypts state at rest (S3+KMS, Terraform Cloud) when sensitive attributes are in use.

> **Ref:** <https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#sensitive>

---

## References

| Topic | URL |
|-------|-----|
| Custom Types | <https://developer.hashicorp.com/terraform/plugin/framework/handling-data/types/custom> |
| Ephemeral Resources | <https://developer.hashicorp.com/terraform/plugin/framework/ephemeral-resources> |
| Write-Only Attributes | <https://developer.hashicorp.com/terraform/plugin/framework/resources/write-only-attributes> |
| Private State | <https://developer.hashicorp.com/terraform/plugin/framework/resources/private-state> |
| Resource Timeouts | <https://developer.hashicorp.com/terraform/plugin/framework/resources/timeouts> |
| Sensitive Attributes | <https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#sensitive> |
