# 08 — State Management and Import

## Overview

Terraform state is the core mechanism that ties declared configuration to real-world
infrastructure. This document covers how the Plugin Framework exposes state, plan, and
config data; how resource import works; drift detection; and private state.

---

## 1. How Terraform State Works

Terraform maintains a **state file** (`terraform.tfstate`) that records the last-known
real-world attributes of every managed resource. It is the single source of truth for:

1. **Determining what exists** — mapping configuration blocks to real infrastructure.
2. **Computing diffs** — comparing desired config against current state to build a plan.
3. **Detecting drift** — during `terraform plan`/`refresh`, `Read` fetches live data and
   updates state so Terraform can surface out-of-band changes.

> **Key insight:** The provider is responsible for writing state. If your `Create` or
> `Read` method does not call `resp.State.Set(...)`, Terraform has no record of the
> resource.

---

## 2. State, Plan, and Config — The Three Data Sources

### 2.1 State — `req.State`

The **prior state**: attribute values stored after the last successful apply or import.

```go
func (r *WorkflowResource) Read(ctx context.Context,
    req resource.ReadRequest, resp *resource.ReadResponse) {
    var state WorkflowResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

### 2.2 Plan — `req.Plan`

The **desired state**: user configuration merged with `Computed`/`Default` values.

```go
func (r *WorkflowResource) Create(ctx context.Context,
    req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan WorkflowResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

### 2.3 Config — `req.Config`

The **raw user configuration** — exactly what was written in `.tf`. `Computed` attributes
the user did not set are `Null`. Useful for distinguishing "user explicitly set this"
from "framework filled a default."

```go
var config WorkflowResourceModel
diags := req.Config.Get(ctx, &config)
// config.Description is types.StringNull() if the user didn't specify it.
```

### 2.4 Writing State — `resp.State.Set`

After any mutation (Create, Read, Update) you **must** write the new state:

```go
state.ID = types.StringValue(apiResp.ID)
state.Name = types.StringValue(apiResp.Name)
diags = resp.State.Set(ctx, &state)
resp.Diagnostics.Append(diags...)
```

### 2.5 Availability Matrix

| Method | `req.State` | `req.Plan` | `req.Config` | `resp.State` (write) |
|--------|:-----------:|:----------:|:------------:|:--------------------:|
| Create | ✗           | ✓          | ✓            | ✓                    |
| Read   | ✓           | ✗          | ✗            | ✓                    |
| Update | ✓           | ✓          | ✓            | ✓                    |
| Delete | ✓           | ✗          | ✗            | ✗ (auto-removed)     |

> In `Delete`, the framework automatically removes the resource from state after a
> successful return (no errors). You do **not** call `resp.State.Set`.

---

## 3. Resource Import

Import lets practitioners adopt existing infrastructure into Terraform without recreating it.

### 3.1 Implementing `ResourceWithImportState`

```go
var (
    _ resource.Resource                = &WorkflowResource{}
    _ resource.ResourceWithImportState = &WorkflowResource{}
)

func (r *WorkflowResource) ImportState(ctx context.Context,
    req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // req.ID contains the string from: terraform import <addr> <ID>
}
```

### 3.2 Simple Import — `ImportStatePassthroughID`

When the import identifier maps directly to the `id` attribute:

```go
func (r *WorkflowResource) ImportState(ctx context.Context,
    req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

Terraform then calls `Read` to populate all other attributes from the API.

### 3.3 Complex Import — Composite IDs

Some resources need multiple identifiers, e.g., `"org_id/workflow_id"`:

```go
func (r *WorkflowResource) ImportState(ctx context.Context,
    req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

    parts := strings.SplitN(req.ID, "/", 2)
    if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
        resp.Diagnostics.AddError(
            "Invalid Import ID",
            fmt.Sprintf("Expected format 'org_id/workflow_id', got: %q", req.ID),
        )
        return
    }
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), parts[0])...)
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
```

### 3.4 Full Import Example

```go
package provider

import (
    "context"
    "fmt"
    "strings"

    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
    _ resource.Resource                = &WorkflowResource{}
    _ resource.ResourceWithImportState = &WorkflowResource{}
)

type WorkflowResourceModel struct {
    OrgID       types.String `tfsdk:"org_id"`
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    Description types.String `tfsdk:"description"`
}

func (r *WorkflowResource) ImportState(ctx context.Context,
    req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

    parts := strings.SplitN(req.ID, "/", 2)
    if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
        resp.Diagnostics.AddError(
            "Invalid Import ID",
            fmt.Sprintf(
                "Expected 'org_id/workflow_id', got: %q.\n\n"+
                    "Example: terraform import comfyui_workflow.example my-org/wf-123",
                req.ID,
            ),
        )
        return
    }

    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), parts[0])...)
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
    // Terraform automatically calls Read() next to hydrate the full state.
}
```

---

## 4. Drift Detection

**Drift** occurs when infrastructure is modified outside Terraform. `Read` is the
detection point.

### 4.1 Resource Deleted Outside Terraform

If the API returns 404, remove the resource from state so Terraform recreates it:

```go
func (r *WorkflowResource) Read(ctx context.Context,
    req resource.ReadRequest, resp *resource.ReadResponse) {

    var state WorkflowResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    workflow, err := r.client.GetWorkflow(ctx, state.ID.ValueString())
    if err != nil {
        if isNotFoundError(err) {
            tflog.Warn(ctx, "Workflow not found, removing from state",
                map[string]any{"id": state.ID.ValueString()})
            resp.State.RemoveResource(ctx)
            return
        }
        resp.Diagnostics.AddError("Error Reading Workflow",
            fmt.Sprintf("Could not read workflow %s: %s", state.ID.ValueString(), err))
        return
    }

    state.Name = types.StringValue(workflow.Name)
    state.Description = types.StringValue(workflow.Description)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

### 4.2 Attribute-Level Drift

When the resource exists but attributes changed, write live values back. Terraform's
diff engine surfaces changes in the next `plan`:

```
~ resource "comfyui_workflow" "main" {
    ~ name = "Original Name" -> "Changed Name"
  }
```

---

## 5. Private State

Private state stores provider metadata **not visible** to practitioners in
`terraform show`, but persisted across runs.

### 5.1 Use Cases

| Use Case     | Example                                                    |
|--------------|------------------------------------------------------------|
| ETags        | Store `ETag` header for optimistic concurrency             |
| Internal IDs | API-internal ID different from user-facing one             |
| Version hash | Content hash to detect server-side changes                 |

### 5.2 Reading and Writing Private State

```go
// Reading — available via req.Private
etagBytes, diags := req.Private.GetKey(ctx, "etag")
resp.Diagnostics.Append(diags...)

var etag string
if etagBytes != nil {
    etag = string(etagBytes)
}

// Writing — available via resp.Private
diags = resp.Private.SetKey(ctx, "etag", []byte(apiResp.ETag))
resp.Diagnostics.Append(diags...)
```

> Private state keys are arbitrary strings. Values are `[]byte`. The data is stored
> in the state file but not displayed by Terraform CLI commands.

---

## 6. State Move (Advanced)

`MoveState` migrates state when a resource type is renamed or refactored:

```go
var _ resource.ResourceWithMoveState = &WorkflowResource{}

func (r *WorkflowResource) MoveState(ctx context.Context) []resource.StateMover {
    return []resource.StateMover{
        {
            SourceSchema: &schema.Schema{
                Attributes: map[string]schema.Attribute{
                    "workflow_id": schema.StringAttribute{Required: true},
                },
            },
            StateMover: func(ctx context.Context,
                req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
                var oldModel struct {
                    WorkflowID types.String `tfsdk:"workflow_id"`
                }
                resp.Diagnostics.Append(req.SourceState.Get(ctx, &oldModel)...)
                if resp.Diagnostics.HasError() {
                    return
                }
                var newModel WorkflowResourceModel
                newModel.ID = oldModel.WorkflowID
                resp.Diagnostics.Append(resp.TargetState.Set(ctx, &newModel)...)
            },
        },
    }
}
```

---

## References

- [Resource Import](https://developer.hashicorp.com/terraform/plugin/framework/resources/import)
- [State Move](https://developer.hashicorp.com/terraform/plugin/framework/resources/state-move)
- [Private State](https://developer.hashicorp.com/terraform/plugin/framework/resources/private-state)
- [Terraform State Internals](https://developer.hashicorp.com/terraform/language/state)
