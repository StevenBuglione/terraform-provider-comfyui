# Provider Design Principles

> Research reference for the AI coding harness building `terraform-provider-comfyui`.
> All examples use the **Plugin Framework** (`terraform-plugin-framework`), not SDKv2.

---

## 1. HashiCorp's Official Provider Design Principles

### 1.1 A Provider Should Manage a Single API or Service Domain

A provider maps to one API. `hashicorp/aws` manages AWS, not GCP. `terraform-provider-comfyui` manages the ComfyUI API and nothing else. Don't mix unrelated services in one provider block.

### 1.2 Resources Should Map 1:1 to API Objects

Each Terraform resource corresponds to exactly one API object. If the API has "workflow" and "node" objects, those become `comfyui_workflow` and `comfyui_node` — not a combined `comfyui_workflow_with_nodes`.

```hcl
resource "comfyui_workflow" "example" { name = "my-pipeline" }
resource "comfyui_node" "sampler" {
  workflow_id = comfyui_workflow.example.id
  type        = "KSampler"
}
```

### 1.3 Resources Represent a Single API Object, Not a Workflow

A resource is a **noun** (a thing), not a **verb** (an action). Don't create resources that orchestrate multi-step processes — that's the module's job.

### 1.4 Complex Workflows Are Handled by Modules, Not the Provider

The provider exposes primitives. Modules compose them:

```hcl
# modules/comfyui-pipeline/main.tf
resource "comfyui_workflow" "pipeline" { name = var.pipeline_name }
resource "comfyui_node" "loader" {
  workflow_id = comfyui_workflow.pipeline.id
  type        = "CheckpointLoaderSimple"
  inputs      = { ckpt_name = var.model_name }
}
resource "comfyui_node" "sampler" {
  workflow_id = comfyui_workflow.pipeline.id
  type        = "KSampler"
  inputs      = { model = comfyui_node.loader.outputs["MODEL"] }
}
```

### 1.5 Resource Schemas Should Closely Mirror the API Structure

Schema attributes map to API request/response fields. If the API returns `api_format`, expose `api_format` — not `format_type`.

```go
func (r *WorkflowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id":          schema.StringAttribute{Computed: true},
            "name":        schema.StringAttribute{Required: true},
            "description": schema.StringAttribute{Optional: true},
            "api_format":  schema.StringAttribute{Computed: true},
        },
    }
}
```

### 1.6 Use Data Sources for Read-Only Lookups

Data sources query existing infrastructure — they never create, update, or delete.

```go
func (d *ModelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var config ModelDataSourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
    model, err := d.client.GetModel(ctx, config.Name.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Failed to read model", err.Error())
        return
    }
    config.ID = types.StringValue(model.ID)
    resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
```

### 1.7 Prefer API-Native Naming Unless It Harms UX

If the ComfyUI API calls it a `prompt`, the resource is `comfyui_prompt` — not `comfyui_job`. Only rename when the API term would genuinely mislead practitioners.

---

## 2. When to Create a New Resource vs Extend an Existing One

| Question | YES → New Resource | NO → Extend Existing |
|---|---|---|
| Does it have its own API endpoint? | ✅ | ❌ |
| Does it have its own lifecycle (create/delete independently)? | ✅ | ❌ |
| Can it exist without the parent object? | ✅ | ❌ |
| Does it have a unique identifier in the API? | ✅ | ❌ |

If ComfyUI "workflow settings" share the workflow's endpoint and can't exist independently, keep them on `comfyui_workflow`. If settings have their own CRUD endpoint, create `comfyui_workflow_settings`.

---

## 3. When to Use Data Sources vs Resources

- **Resource:** Terraform owns the object's lifecycle. `terraform destroy` deletes it.
- **Data source:** The object exists outside Terraform. Terraform reads it but never modifies it.
- **Neither:** If Terraform doesn't need to reference or manage the object.

```hcl
# Resource — Terraform manages this workflow
resource "comfyui_workflow" "mine" { name = "txt2img" }

# Data source — pre-installed model, read-only
data "comfyui_model" "sd15" { name = "v1-5-pruned.safetensors" }
```

---

## 4. Provider Scope

**In the provider:** CRUD on API objects (resources), read-only lookups (data sources), authentication, input validation.

**In a module:** Composing resources, conditional logic, defaults/opinions, cross-provider orchestration.

**Rule of thumb:** If it needs `for_each`, `count`, or `depends_on` to wire things together, it belongs in a module.

---

## 5. API Client Design

Use a **thin wrapper** — each method maps to one API endpoint. Keep Terraform concerns out of the client layer.

```go
// internal/client/client.go
type ComfyUIClient struct {
    BaseURL    string
    HTTPClient *http.Client
}

func (c *ComfyUIClient) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet,
        fmt.Sprintf("%s/api/workflows/%s", c.BaseURL, id), nil)
    if err != nil {
        return nil, fmt.Errorf("building request: %w", err)
    }
    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("executing request: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status %d for workflow %s", resp.StatusCode, id)
    }
    var workflow Workflow
    if err := json.NewDecoder(resp.Body).Decode(&workflow); err != nil {
        return nil, fmt.Errorf("decoding response: %w", err)
    }
    return &workflow, nil
}
```

**Separation:** Provider converts between Terraform types ↔ Go structs. Client converts between Go structs ↔ HTTP. Neither leaks into the other.

---

## 6. Idempotency Requirements

All CRUD operations must be idempotent: calling them repeatedly with the same input produces the same result.

### Create — Handle "Already Exists"

```go
func (r *WorkflowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan WorkflowResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() { return }

    workflow, err := r.client.CreateWorkflow(ctx, &client.CreateWorkflowRequest{
        Name: plan.Name.ValueString(),
    })
    if err != nil {
        var conflict *client.ConflictError
        if errors.As(err, &conflict) {
            existing, readErr := r.client.GetWorkflow(ctx, conflict.ExistingID)
            if readErr != nil {
                resp.Diagnostics.AddError("Workflow exists but could not be read",
                    fmt.Sprintf("ID: %s, error: %s", conflict.ExistingID, readErr))
                return
            }
            plan.ID = types.StringValue(existing.ID)
            resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
            return
        }
        resp.Diagnostics.AddError("Failed to create workflow", err.Error())
        return
    }
    plan.ID = types.StringValue(workflow.ID)
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

### Read — Must Never Modify State

```go
func (r *WorkflowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state WorkflowResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() { return }

    workflow, err := r.client.GetWorkflow(ctx, state.ID.ValueString())
    if err != nil {
        var notFound *client.NotFoundError
        if errors.As(err, &notFound) {
            resp.State.RemoveResource(ctx) // deleted outside Terraform
            return
        }
        resp.Diagnostics.AddError("Failed to read workflow", err.Error())
        return
    }
    state.Name = types.StringValue(workflow.Name)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

### Update — Same Config Twice = No Change

```go
func (r *WorkflowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan WorkflowResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    _, err := r.client.UpdateWorkflow(ctx, plan.ID.ValueString(), &client.UpdateWorkflowRequest{
        Name: plan.Name.ValueString(),
    })
    if err != nil {
        resp.Diagnostics.AddError("Failed to update workflow", err.Error())
        return
    }
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

### Delete — Must Succeed if Already Gone

```go
func (r *WorkflowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state WorkflowResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    err := r.client.DeleteWorkflow(ctx, state.ID.ValueString())
    if err != nil {
        var notFound *client.NotFoundError
        if errors.As(err, &notFound) {
            return // already gone — not an error
        }
        resp.Diagnostics.AddError("Failed to delete workflow", err.Error())
    }
}
```

---

## 7. The Principle of Least Surprise

### Naming Conventions

| Convention | Example |
|---|---|
| Resource: `<provider>_<noun>` | `comfyui_workflow` |
| Data source (list): `<provider>_<noun_plural>` | `comfyui_models` |
| Attributes: `snake_case` | `vram_total`, `client_id` |
| Booleans: positive phrasing | `enabled` not `disabled` |
| IDs: always `Computed: true` | `schema.StringAttribute{Computed: true}` |

### ForceNew vs In-Place Update

Only use `RequiresReplace()` when the API cannot update in place. Practitioners expect in-place updates by default.

```go
"instance_type": schema.StringAttribute{
    Required: true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.RequiresReplace(),
    },
},
```

### Computed and Optional+Computed Attributes

```go
"created_at": schema.StringAttribute{Computed: true},              // API-only
"priority":   schema.Int64Attribute{Optional: true, Computed: true}, // user or API default
```

---

## 8. Actionable Error Messages

Errors must say **what went wrong** and **what to do about it**.

```go
// BAD
resp.Diagnostics.AddError("Error", err.Error())

// GOOD — actionable with context
resp.Diagnostics.AddError(
    "Unable to connect to ComfyUI server",
    fmt.Sprintf("Could not reach %q. Verify the server is running and 'host' is correct.\n\nError: %s",
        r.client.BaseURL, err),
)

// GOOD — attribute-level validation
resp.Diagnostics.AddAttributeError(
    path.Root("name"),
    "Invalid workflow name",
    fmt.Sprintf("Names must be 1-128 chars matching [a-zA-Z0-9_-]+. Got: %q", plan.Name.ValueString()),
)

// GOOD — warning for deprecation
resp.Diagnostics.AddWarning(
    "Deprecated API field",
    "The 'legacy_format' attribute is deprecated. Migrate to 'api_format'.",
)
```

---

## References

- [HashiCorp Provider Design Principles](https://developer.hashicorp.com/terraform/plugin/best-practices/hashicorp-provider-design-principles)
- [Plugin Framework Documentation](https://developer.hashicorp.com/terraform/plugin/framework)
- [Plugin Framework Diagnostics](https://developer.hashicorp.com/terraform/plugin/framework/diagnostics)
- [Terraform Resources](https://developer.hashicorp.com/terraform/plugin/framework/resources)
- [Data Sources](https://developer.hashicorp.com/terraform/plugin/framework/data-sources)
- [Plan Modifiers](https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification)
