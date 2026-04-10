# 04 — Resource Implementation (Plugin Framework)

> **Audience**: AI coding harness. This covers the complete anatomy of a
> managed resource built with `terraform-plugin-framework`. SDKv2 is **not** used.

---

## 1. The `resource.Resource` Interface

```go
type Resource interface {
    Metadata(ctx context.Context, req MetadataRequest, resp *MetadataResponse)
    Schema(ctx context.Context, req SchemaRequest, resp *SchemaResponse)
    Create(ctx context.Context, req CreateRequest, resp *CreateResponse)
    Read(ctx context.Context, req ReadRequest, resp *ReadResponse)
    Update(ctx context.Context, req UpdateRequest, resp *UpdateResponse)
    Delete(ctx context.Context, req DeleteRequest, resp *DeleteResponse)
}
```

Every resource should also implement `resource.ResourceWithConfigure` to
receive the API client from the provider's `Configure` method.

---

## 2. Optional Interfaces

The framework detects these via type assertion. Implement only what you need.

### 2.1 `resource.ResourceWithImportState`

Enables `terraform import`. Minimal implementation using the helper:

```go
func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

For compound keys (e.g., `project_id/server_id`), parse manually:

```go
func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    parts := strings.SplitN(req.ID, "/", 2)
    if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
        resp.Diagnostics.AddError("Invalid Import ID", "Expected <project_id>/<server_id>. Got: "+req.ID)
        return
    }
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
```

### 2.2 `resource.ResourceWithModifyPlan`

Hook into plan generation for custom warnings or computed-value overrides:

```go
func (r *serverResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
    if req.Plan.Raw.IsNull() { return } // being destroyed
    var plan, state serverResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() { return }
    if !plan.Region.Equal(state.Region) {
        resp.Diagnostics.AddWarning("Region Change", "Changing region destroys and recreates the server.")
    }
}
```

### 2.3 `resource.ResourceWithValidateConfig`

Cross-attribute validation during `terraform validate`:

```go
func (r *serverResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
    var config serverResourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
    if resp.Diagnostics.HasError() { return }
    if config.ImageID.IsNull() && config.ImageName.IsNull() {
        resp.Diagnostics.AddError("Missing Image", "Either \"image_id\" or \"image_name\" must be set.")
    }
}
```

---

## 3. Resource Model Struct

Maps every schema attribute. `tfsdk` tag must exactly match the attribute key.

```go
type serverResourceModel struct {
    ID        types.String `tfsdk:"id"`
    Name      types.String `tfsdk:"name"`
    Region    types.String `tfsdk:"region"`
    CPU       types.Int64  `tfsdk:"cpu"`
    Memory    types.Int64  `tfsdk:"memory"`
    Status    types.String `tfsdk:"status"`
    CreatedAt types.String `tfsdk:"created_at"`
}
```

Always use `types.*` — never raw Go primitives.

---

## 4. API Client Access

```go
type serverResource struct{ client *client.ComfyUIClient }

func NewServerResource() resource.Resource { return &serverResource{} }

func (r *serverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil { return } // nil during ValidateConfig phase
    c, ok := req.ProviderData.(*client.ComfyUIClient)
    if !ok {
        resp.Diagnostics.AddError("Unexpected Type", fmt.Sprintf("Expected *client.ComfyUIClient, got %T.", req.ProviderData))
        return
    }
    r.client = c
}
```

---

## 5. Metadata & Schema

```go
func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_server" // becomes "comfyui_server"
}

func (r *serverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Manages a ComfyUI server instance.",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed: true, Description: "Server UUID.",
                PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
            },
            "name":   schema.StringAttribute{Required: true, Description: "Server name."},
            "region": schema.StringAttribute{
                Required: true, Description: "Deployment region.",
                PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
            },
            "cpu":        schema.Int64Attribute{Required: true, Description: "vCPU count."},
            "memory":     schema.Int64Attribute{Required: true, Description: "Memory in MB."},
            "status":     schema.StringAttribute{Computed: true, Description: "Current status."},
            "created_at": schema.StringAttribute{
                Computed: true, Description: "RFC 3339 timestamp.",
                PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
            },
        },
    }
}
```

Key plan modifiers: `UseStateForUnknown()` keeps prior value so `id` doesn't
show `(known after apply)` on every plan. `RequiresReplace()` forces destroy +
recreate when the attribute changes.

Import paths:
```
"github.com/hashicorp/terraform-plugin-framework/resource/schema"
"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
```

---

## 6. CRUD Lifecycle

### 6.1 Create — Read plan → call API → set state

```go
func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan serverResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() { return }

    server, err := r.client.CreateServer(ctx, client.CreateServerInput{
        Name: plan.Name.ValueString(), Region: plan.Region.ValueString(),
        CPU: plan.CPU.ValueInt64(), Memory: plan.Memory.ValueInt64(),
    })
    if err != nil {
        resp.Diagnostics.AddError("Error Creating Server", err.Error())
        return
    }

    plan.ID = types.StringValue(server.ID)
    plan.Status = types.StringValue(server.Status)
    plan.CreatedAt = types.StringValue(server.CreatedAt)
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

### 6.2 Read — Read state → call API → update state (handle not-found)

```go
func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state serverResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() { return }

    server, err := r.client.GetServer(ctx, state.ID.ValueString())
    if err != nil {
        if client.IsNotFound(err) {
            resp.State.RemoveResource(ctx) // "disappears" pattern
            return
        }
        resp.Diagnostics.AddError("Error Reading Server", err.Error())
        return
    }

    state.Name = types.StringValue(server.Name)
    state.Region = types.StringValue(server.Region)
    state.CPU = types.Int64Value(server.CPU)
    state.Memory = types.Int64Value(server.Memory)
    state.Status = types.StringValue(server.Status)
    state.CreatedAt = types.StringValue(server.CreatedAt)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

### 6.3 Update — Read plan → call API → set state

```go
func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan serverResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() { return }

    server, err := r.client.UpdateServer(ctx, plan.ID.ValueString(), client.UpdateServerInput{
        Name: plan.Name.ValueString(), CPU: plan.CPU.ValueInt64(), Memory: plan.Memory.ValueInt64(),
    })
    if err != nil {
        resp.Diagnostics.AddError("Error Updating Server", err.Error())
        return
    }

    plan.Status = types.StringValue(server.Status)
    plan.CreatedAt = types.StringValue(server.CreatedAt)
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

### 6.4 Delete — Read state → call API → state auto-removed

```go
func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state serverResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() { return }

    err := r.client.DeleteServer(ctx, state.ID.ValueString())
    if err != nil {
        if client.IsNotFound(err) { return } // already gone — fine
        resp.Diagnostics.AddError("Error Deleting Server", err.Error())
        return
    }
    // State is automatically removed. Do NOT call resp.State.RemoveResource.
}
```

---

## 7. The "Disappears" Pattern

When a resource is deleted outside Terraform, the next `Read` should:

1. Detect 404 from the API.
2. Call `resp.State.RemoveResource(ctx)` and return **without error**.
3. Terraform sees empty state and proposes re-creation.

**Do not** add a diagnostic error — that would make `terraform plan` fail.

Acceptance test pattern:

```go
func TestAccServer_disappears(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{{
            Config: testAccServerConfig("test-disappears"),
            Check: resource.ComposeTestCheckFunc(
                testAccCheckServerExists("comfyui_server.test"),
                testAccCheckServerDisappears("comfyui_server.test"), // deletes via API
            ),
            ExpectNonEmptyPlan: true,
        }},
    })
}
```

---

## 8. Timeouts Support

Uses the `terraform-plugin-framework-timeouts` module.

### Schema integration

```go
import "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

// In the model:
type serverResourceModel struct {
    // ... fields ...
    Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// In Schema(), add a block:
Blocks: map[string]schema.Block{
    "timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
},
```

### Using in CRUD

```go
createTimeout, diags := plan.Timeouts.Create(ctx, 20*time.Minute) // default 20m
resp.Diagnostics.Append(diags...)
if resp.Diagnostics.HasError() { return }
ctx, cancel := context.WithTimeout(ctx, createTimeout)
defer cancel()
// use ctx for API calls — respects the timeout
```

HCL usage:

```hcl
resource "comfyui_server" "example" {
  name = "my-server"
  timeouts { create = "30m"; delete = "10m" }
}
```

---

## 9. Compile-Time Interface Checks

```go
var (
    _ resource.Resource                = &serverResource{}
    _ resource.ResourceWithConfigure   = &serverResource{}
    _ resource.ResourceWithImportState = &serverResource{}
)
```

---

## 10. Assembling the Complete File

Combine all pieces above into a single `resource_server.go`. The file structure:

```go
package provider

import (
    "context"
    "fmt"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/yourorg/terraform-provider-comfyui/internal/client"
)

// 1. Compile-time checks (Section 9)
// 2. Model struct         (Section 3)
// 3. Resource struct       (Section 4)
// 4. Constructor           (Section 4: NewServerResource)
// 5. Metadata + Schema     (Section 5)
// 6. Configure             (Section 4)
// 7. Create/Read/Update/Delete (Section 6)
// 8. ImportState           (Section 2.1)
```

Each section above contains the complete, copy-pasteable code for that piece.
Concatenate them in this order for a working file.

---

## References

| Source | URL |
|---|---|
| Framework — Resources overview | https://developer.hashicorp.com/terraform/plugin/framework/resources |
| Framework — CRUD lifecycle | https://developer.hashicorp.com/terraform/plugin/framework/resources/create |
| Framework — Import state | https://developer.hashicorp.com/terraform/plugin/framework/resources/import |
| Framework — Plan modification | https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification |
| Framework — Validate config | https://developer.hashicorp.com/terraform/plugin/framework/resources/validate-configuration |
| Framework — Timeouts | https://developer.hashicorp.com/terraform/plugin/framework/resources/timeouts |
| Tutorial — Implement resources | https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-resource-create |
| `terraform-plugin-framework-timeouts` | https://github.com/hashicorp/terraform-plugin-framework-timeouts |
