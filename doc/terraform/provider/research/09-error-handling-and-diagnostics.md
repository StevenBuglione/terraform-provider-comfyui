# 09 — Error Handling and Diagnostics

## Overview

The Terraform Plugin Framework uses a **diagnostics** system instead of returning Go
`error` values from CRUD methods. Every response struct carries a `Diagnostics` field
that accumulates errors and warnings. This document covers diagnostics, effective error
messages, common patterns, and the `tflog` structured logging package.

---

## 1. The Diagnostics System

`diag.Diagnostics` is a slice of diagnostic entries. Each entry has a severity, summary,
detail, and optional attribute path.

### 1.1 Two Severity Levels

| Severity    | Effect                                                                       |
|-------------|------------------------------------------------------------------------------|
| **Error**   | Halts execution for the current resource. Terraform marks it tainted/failed. |
| **Warning** | Displayed to the practitioner but does **not** stop execution.               |

### 1.2 How Diagnostics Render

```
╷
│ Error: Error Creating Workflow
│
│   with comfyui_workflow.main,
│   on main.tf line 5, in resource "comfyui_workflow" "main":
│    5: resource "comfyui_workflow" "main" {
│
│ Could not create workflow "my-workflow": 403 Forbidden
│
│ Please verify that your API token has the "workflows:write" permission.
╵
```

---

## 2. Adding Diagnostics

### 2.1 Basic Error and Warning

```go
resp.Diagnostics.AddError(
    "Error Creating Workflow",
    fmt.Sprintf("Could not create workflow %q: %s", name, err),
)

resp.Diagnostics.AddWarning(
    "Deprecated API Version",
    "API v1 will be removed in provider v2.0. Please migrate to the v2 resource.",
)
```

### 2.2 Attribute-Scoped Diagnostics

Tie the diagnostic to a specific schema attribute so Terraform points the practitioner
to the exact configuration line:

```go
resp.Diagnostics.AddAttributeError(
    path.Root("name"),
    "Invalid Workflow Name",
    fmt.Sprintf("Name %q contains invalid characters. Must match [a-z0-9-]+.",
        plan.Name.ValueString()),
)

resp.Diagnostics.AddAttributeWarning(
    path.Root("description"),
    "Description Truncated",
    "The API truncated the description to 500 characters.",
)
```

### 2.3 Appending Diagnostics from Sub-Calls

Many framework functions return `diag.Diagnostics`. Always append and check:

```go
diags := req.Plan.Get(ctx, &plan)
resp.Diagnostics.Append(diags...)
if resp.Diagnostics.HasError() {
    return
}
```

---

## 3. Writing Effective Error Messages

### 3.1 Summary: Concise, Action-Oriented, Title Case

| ✅ Good                       | ❌ Bad                    |
|------------------------------|---------------------------|
| `"Error Creating Workflow"`  | `"Error"`                 |
| `"Error Reading Server"`    | `"API call failed"`       |
| `"Invalid Workflow Name"`   | `"Validation error"`      |

### 3.2 Detail: What + Why + How to Fix

```go
// ❌ BAD — generic, unhelpful.
resp.Diagnostics.AddError("Error", err.Error())

// ✅ GOOD — specific, actionable.
resp.Diagnostics.AddError(
    "Error Creating Workflow",
    fmt.Sprintf(
        "Could not create workflow %q in organization %q: %s\n\n"+
            "Please verify that:\n"+
            "  - Your API token has the 'workflows:write' permission\n"+
            "  - The organization ID is correct\n"+
            "  - The workflow name is unique within the organization",
        plan.Name.ValueString(), plan.OrgID.ValueString(), err,
    ),
)
```

### 3.3 Avoid Leaking Implementation Details

```go
// ❌ BAD — exposes internal Go types.
resp.Diagnostics.AddError("Error",
    fmt.Sprintf("*net/http.Response statusCode=403 body=%v", respBody))

// ✅ GOOD — translates for the practitioner.
resp.Diagnostics.AddError("Insufficient Permissions",
    "The API returned 403 Forbidden. Ensure your API token has the required permissions.")
```

---

## 4. Error Handling Patterns

### 4.1 Early Return After Error Check

```go
func (r *WorkflowResource) Create(ctx context.Context,
    req resource.CreateRequest, resp *resource.CreateResponse) {

    var plan WorkflowResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return // ← Stop before calling the API.
    }

    workflow, err := r.client.CreateWorkflow(ctx, plan.Name.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Error Creating Workflow",
            fmt.Sprintf("Could not create workflow %q: %s", plan.Name.ValueString(), err))
        return // ← Stop before writing state.
    }

    plan.ID = types.StringValue(workflow.ID)
    plan.Name = types.StringValue(workflow.Name)
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

### 4.2 Handling API-Specific HTTP Status Codes

```go
workflow, err := r.client.GetWorkflow(ctx, state.ID.ValueString())
if err != nil {
    var apiErr *APIError
    if errors.As(err, &apiErr) {
        switch apiErr.StatusCode {
        case 404:
            // Resource deleted outside Terraform — remove from state.
            resp.State.RemoveResource(ctx)
            return
        case 403:
            resp.Diagnostics.AddError("Insufficient Permissions",
                fmt.Sprintf("Cannot read workflow %q: access denied.\n\n"+
                    "Ensure your API token has 'workflows:read' permission.",
                    state.ID.ValueString()))
            return
        }
    }
    resp.Diagnostics.AddError("Error Reading Workflow",
        fmt.Sprintf("Could not read workflow %q: %s", state.ID.ValueString(), err))
    return
}
```

### 4.3 Wrapping Go Errors With Context

Use `fmt.Errorf` with `%w` to preserve the error chain for `errors.As`/`errors.Is`:

```go
func (c *ComfyUIClient) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
    resp, err := c.httpClient.Get(fmt.Sprintf("/api/workflows/%s", id))
    if err != nil {
        return nil, fmt.Errorf("fetching workflow %s: %w", id, err)
    }
    if resp.StatusCode != http.StatusOK {
        return nil, &APIError{StatusCode: resp.StatusCode,
            Message: fmt.Sprintf("unexpected status %d for workflow %s", resp.StatusCode, id)}
    }
    // ...
}
```

### 4.4 Accumulating Multiple Validation Errors

```go
if len(plan.Name.ValueString()) > 64 {
    resp.Diagnostics.AddAttributeError(path.Root("name"),
        "Workflow Name Too Long", "Names must be 64 characters or fewer.")
}
if strings.Contains(plan.Name.ValueString(), " ") {
    resp.Diagnostics.AddAttributeError(path.Root("name"),
        "Workflow Name Contains Spaces", "Use hyphens instead of spaces.")
}
// Return all errors at once so the practitioner fixes them in a single pass.
if resp.Diagnostics.HasError() {
    return
}
```

---

## 5. Logging vs. Diagnostics

| Channel        | Audience            | When to Use                                            |
|----------------|---------------------|--------------------------------------------------------|
| **Diagnostics** | Practitioners      | Errors/warnings shown in `terraform plan/apply` output |
| **tflog**       | Provider developers | Structured logs written to `TF_LOG` for debugging      |

**Rule of thumb:** Will the practitioner need to see this? → Diagnostic.
Only useful when debugging the provider? → `tflog`. Both? → Add both.

---

## 6. The `tflog` Package

`github.com/hashicorp/terraform-plugin-log/tflog` provides structured logging
integrated with Terraform's `TF_LOG` environment variable.

### 6.1 Log Levels

```go
tflog.Trace(ctx, "entering Read method", map[string]any{"id": state.ID.ValueString()})
tflog.Debug(ctx, "API response received", map[string]any{"status_code": 200})
tflog.Info(ctx, "workflow created", map[string]any{"id": workflow.ID})
tflog.Warn(ctx, "API deprecation header present", map[string]any{"header": val})
tflog.Error(ctx, "failed to parse response", map[string]any{"error": err.Error()})
```

### 6.2 Structured Fields — Always Use Them

```go
// ❌ BAD — unstructured.
tflog.Debug(ctx, fmt.Sprintf("got workflow id=%s name=%s", w.ID, w.Name))

// ✅ GOOD — structured, machine-parseable.
tflog.Debug(ctx, "got workflow", map[string]any{"id": w.ID, "name": w.Name})
```

### 6.3 Scoped Fields and Masking Sensitive Values

```go
ctx = tflog.SetField(ctx, "resource", "comfyui_workflow")
tflog.Debug(ctx, "reading workflow") // output includes resource=comfyui_workflow

ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "api_token")
tflog.Debug(ctx, "configuring provider", map[string]any{
    "host":      config.Host.ValueString(),
    "api_token": config.APIToken.ValueString(), // Logged as "***"
})
```

---

## 7. Complete Example — Error Handling in Create

```go
func (r *WorkflowResource) Create(ctx context.Context,
    req resource.CreateRequest, resp *resource.CreateResponse) {

    tflog.Debug(ctx, "starting workflow creation")

    var plan WorkflowResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }

    workflow, err := r.client.CreateWorkflow(ctx, &CreateWorkflowInput{
        OrgID: plan.OrgID.ValueString(),
        Name:  plan.Name.ValueString(),
    })
    if err != nil {
        var apiErr *APIError
        if errors.As(err, &apiErr) {
            switch apiErr.StatusCode {
            case 409:
                resp.Diagnostics.AddAttributeError(path.Root("name"),
                    "Workflow Already Exists",
                    fmt.Sprintf("A workflow named %q already exists in org %q.",
                        plan.Name.ValueString(), plan.OrgID.ValueString()))
                return
            case 422:
                resp.Diagnostics.AddError("Invalid Workflow Configuration",
                    fmt.Sprintf("The API rejected the config: %s", apiErr.Message))
                return
            }
        }
        resp.Diagnostics.AddError("Error Creating Workflow",
            fmt.Sprintf("Could not create workflow %q in org %q: %s\n\n"+
                "Check provider logs with TF_LOG=DEBUG.",
                plan.Name.ValueString(), plan.OrgID.ValueString(), err))
        return
    }

    tflog.Info(ctx, "workflow created", map[string]any{"id": workflow.ID})

    plan.ID = types.StringValue(workflow.ID)
    plan.Name = types.StringValue(workflow.Name)
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

    if workflow.ETag != "" {
        resp.Diagnostics.Append(resp.Private.SetKey(ctx, "etag", []byte(workflow.ETag))...)
    }
}
```

---

## 8. Anti-Patterns to Avoid

| Anti-Pattern | Why It's Bad | Fix |
|---|---|---|
| `AddError("Error", err.Error())` | Generic summary; raw Go error | Specific summary + contextual detail |
| Continuing after `HasError()` | Nil pointer panics, confusing secondary errors | Always `return` after check |
| Using `log.Printf` | Bypasses Terraform log filtering | Use `tflog` |
| Returning `error` from CRUD | Won't compile — framework uses diagnostics | Use `resp.Diagnostics.AddError()` |
| Adding error for 404 in `Read` | Terraform expects silent removal | Use `resp.State.RemoveResource(ctx)` |
| Logging sensitive values | Tokens visible in `TF_LOG` | Use `tflog.MaskFieldValuesWithFieldKeys` |

---

## References

- [Diagnostics](https://developer.hashicorp.com/terraform/plugin/framework/diagnostics)
- [Logging](https://developer.hashicorp.com/terraform/plugin/log)
- [terraform-plugin-log](https://github.com/hashicorp/terraform-plugin-log)
