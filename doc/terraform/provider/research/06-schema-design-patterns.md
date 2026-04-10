# Schema Design Patterns — Terraform Plugin Framework

> For providers built with the **Plugin Framework** (`github.com/hashicorp/terraform-plugin-framework`). Does **NOT** apply to SDKv2.

---

## 1. Attribute Types

| Framework Type | Go Model Type | Terraform HCL | Use Case |
|---|---|---|---|
| `schema.StringAttribute` | `types.String` | `string` | Names, IDs, descriptions, URIs |
| `schema.Int64Attribute` | `types.Int64` | `number` | Counts, ports, sizes |
| `schema.Float64Attribute` | `types.Float64` | `number` | Percentages, rates |
| `schema.BoolAttribute` | `types.Bool` | `bool` | Flags, toggles |
| `schema.ListAttribute` | `types.List` | `list(...)` | Ordered collections (position matters) |
| `schema.SetAttribute` | `types.Set` | `set(...)` | Unordered unique collections |
| `schema.MapAttribute` | `types.Map` | `map(...)` | Key-value pairs with dynamic keys |
| `schema.ObjectAttribute` | `types.Object` | `object({...})` | Structured sub-objects with fixed shape |

`ListAttribute`, `SetAttribute`, and `MapAttribute` require `ElementType` for scalar elements (e.g., `types.StringType`). For structured elements, use the nested attribute variants in §4.

```go
package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Each attribute in a schema.Schema.Attributes map:
"name":          schema.StringAttribute{Required: true}
"port":          schema.Int64Attribute{Optional: true}
"sampling_rate": schema.Float64Attribute{Optional: true, Computed: true}
"enabled":       schema.BoolAttribute{Optional: true, Computed: true}
"groups":        schema.ListAttribute{Optional: true, ElementType: types.StringType}
"tags":          schema.SetAttribute{Optional: true, ElementType: types.StringType}
"labels":        schema.MapAttribute{Optional: true, ElementType: types.StringType}
"timeouts": schema.ObjectAttribute{
	Optional:       true,
	AttributeTypes: map[string]attr.Type{"create": types.StringType, "delete": types.StringType},
}
```

---

## 2. Required vs Optional vs Computed vs Optional+Computed

| Mode | In User Config? | Set by Provider? | Notes |
|---|---|---|---|
| **Required** | Must be present | No | Omission is a validation error |
| **Optional** | May be present | No | Null when absent |
| **Computed** | Never | Always | e.g., `id`, `created_at` |
| **Optional+Computed** | May be present | When absent | User overrides or provider defaults |

Rules: `Required` and `Optional` are mutually exclusive. `Computed` combines with `Optional` but **never** with `Required`.

```go
func (r *ExampleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{              // REQUIRED
				Required: true,
				MarkdownDescription: "Unique name. User must provide.",
			},
			"description": schema.StringAttribute{       // OPTIONAL
				Optional: true,
				MarkdownDescription: "Optional description.",
			},
			"id": schema.StringAttribute{                // COMPUTED
				Computed: true,
				MarkdownDescription: "ID assigned by the API.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"region": schema.StringAttribute{            // OPTIONAL + COMPUTED
				Optional: true, Computed: true,
				MarkdownDescription: "Region. Defaults to us-east-1.",
			},
		},
	}
}

type ExampleResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ID          types.String `tfsdk:"id"`
	Region      types.String `tfsdk:"region"`
}
```

---

## 3. Sensitive Attribute Marking

`Sensitive: true` causes Terraform to redact the value in CLI plan/apply output (`(sensitive value)`). The value is **still stored in plaintext** in state — protect state files independently.

```go
"api_key": schema.StringAttribute{
	Required:  true,
	Sensitive: true,
	MarkdownDescription: "API key for authentication. Redacted in plan output.",
},
```

Mark credentials, tokens, passwords, and secrets. Do **not** mark IDs or names — it hides useful plan information.

---

## 4. Nested Attributes vs Blocks

### 4.1 Nested Attribute Types

**SingleNestedAttribute** — exactly one structured sub-object:

```go
"schedule": schema.SingleNestedAttribute{
	Optional: true,
	Attributes: map[string]schema.Attribute{
		"cron":     schema.StringAttribute{Required: true},
		"timezone": schema.StringAttribute{Optional: true, Computed: true},
	},
}
// Model: Schedule *ScheduleModel `tfsdk:"schedule"`
```

**ListNestedAttribute** — ordered list of structured objects:

```go
"steps": schema.ListNestedAttribute{
	Required: true,
	NestedObject: schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"name":    schema.StringAttribute{Required: true},
			"timeout": schema.Int64Attribute{Optional: true, Computed: true},
		},
	},
}
// Model: Steps []StepModel `tfsdk:"steps"`
```

**SetNestedAttribute** — unordered unique structured objects:

```go
"origins": schema.SetNestedAttribute{
	Optional: true,
	NestedObject: schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"url":         schema.StringAttribute{Required: true},
			"allow_creds": schema.BoolAttribute{Optional: true, Computed: true},
		},
	},
}
```

**MapNestedAttribute** — dynamic keys, structured values:

```go
"env_overrides": schema.MapNestedAttribute{
	Optional: true,
	NestedObject: schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"replicas":  schema.Int64Attribute{Optional: true},
			"cpu_limit": schema.StringAttribute{Optional: true},
		},
	},
}
```

### 4.2 Block Types (Legacy Compatibility)

Blocks use HCL block syntax (`block_name { ... }`) vs attribute assignment (`attr = { ... }`).

```go
Blocks: map[string]schema.Block{
	"retry_policy": schema.SingleNestedBlock{
		Attributes: map[string]schema.Attribute{
			"max_retries": schema.Int64Attribute{Optional: true},
		},
	},
	"ingress_rule": schema.ListNestedBlock{
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"port": schema.Int64Attribute{Required: true},
				"cidr": schema.StringAttribute{Required: true},
			},
		},
	},
}
```

### 4.3 Attributes vs Blocks Decision

| Criterion | Nested Attributes | Blocks |
|---|---|---|
| New providers | ✅ Preferred | Avoid |
| Can be Required/Computed/Sensitive | Yes | No |
| Map/Set variants | Yes | No |
| SDKv2 migration | Not HCL-compatible | Required for old configs |

**Rule: Always use nested attributes for new development.** Blocks exist only for SDKv2 migration compatibility.

---

## 5. Schema Descriptions

| Field | Rendered In | Format |
|---|---|---|
| `Description` | CLI (`terraform providers schema -json`) | Plain text |
| `MarkdownDescription` | Terraform Registry docs | Markdown |

When only `MarkdownDescription` is set, it serves as the plain-text fallback too. Set `Description` separately only when Markdown would render poorly as plain text.

```go
"workflow_id": schema.StringAttribute{
	Required:            true,
	Description:         "The unique identifier of the workflow.",
	MarkdownDescription: "The unique identifier of the workflow. Must match an existing workflow in the ComfyUI instance.",
}
```

**Best practice**: Always set `MarkdownDescription`. Add `Description` only when needed.

---

## 6. Complete Schema Design Example

A realistic `comfyui_workflow` resource using all patterns above.

```go
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *WorkflowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a ComfyUI workflow deployment.",
		Attributes: map[string]schema.Attribute{
			// Computed
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier assigned by the API.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{Computed: true, MarkdownDescription: "RFC 3339 creation timestamp."},
			"status":     schema.StringAttribute{Computed: true, MarkdownDescription: "Current status: `active`, `paused`, `error`."},

			// Required
			"name": schema.StringAttribute{Required: true, MarkdownDescription: "Workflow display name. Must be unique."},
			"workflow_json": schema.StringAttribute{
				Required: true, Sensitive: true,
				MarkdownDescription: "Workflow definition as JSON. Sensitive — may embed keys.",
			},

			// Optional
			"description":    schema.StringAttribute{Optional: true, MarkdownDescription: "Human-readable description."},
			"max_queue_size": schema.Int64Attribute{Optional: true, MarkdownDescription: "Max queued jobs."},

			// Optional + Computed
			"enabled": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				MarkdownDescription: "Whether the workflow accepts jobs. Defaults to `true`.",
			},
			"timeout_seconds":  schema.Int64Attribute{Optional: true, Computed: true, MarkdownDescription: "Max execution time. Defaults to `600`."},
			"priority_weight":  schema.Float64Attribute{Optional: true, Computed: true, MarkdownDescription: "Priority 0.0–1.0. Defaults to `0.5`."},

			// Collections
			"tags":     schema.SetAttribute{Optional: true, ElementType: types.StringType, MarkdownDescription: "Categorization tags."},
			"labels":   schema.MapAttribute{Optional: true, ElementType: types.StringType, MarkdownDescription: "Key-value labels."},
			"node_ids": schema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Extracted node IDs."},

			// SingleNestedAttribute
			"schedule": schema.SingleNestedAttribute{
				Optional: true, MarkdownDescription: "Periodic execution schedule.",
				Attributes: map[string]schema.Attribute{
					"cron_expression": schema.StringAttribute{Required: true, MarkdownDescription: "Cron expression (e.g., `0 */6 * * *`)."},
					"timezone":        schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "IANA timezone. Defaults to `UTC`."},
					"enabled":         schema.BoolAttribute{Optional: true, Computed: true, MarkdownDescription: "Schedule active. Defaults to `true`."},
				},
			},

			// ListNestedAttribute
			"input_mappings": schema.ListNestedAttribute{
				Optional: true, MarkdownDescription: "Input parameter mappings.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"parameter_name": schema.StringAttribute{Required: true, MarkdownDescription: "Input parameter name."},
						"node_id":        schema.StringAttribute{Required: true, MarkdownDescription: "Target node ID."},
						"field_name":     schema.StringAttribute{Required: true, MarkdownDescription: "Field on target node."},
						"default_value":  schema.StringAttribute{Optional: true, MarkdownDescription: "Default if unset at execution."},
					},
				},
			},

			// Sensitive credential
			"api_token": schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "Auth token. Redacted in output."},
		},
	}
}
```

### Go Model

```go
package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type WorkflowResourceModel struct {
	ID             types.String  `tfsdk:"id"`
	CreatedAt      types.String  `tfsdk:"created_at"`
	Status         types.String  `tfsdk:"status"`
	Name           types.String  `tfsdk:"name"`
	WorkflowJSON   types.String  `tfsdk:"workflow_json"`
	Description    types.String  `tfsdk:"description"`
	MaxQueueSize   types.Int64   `tfsdk:"max_queue_size"`
	Enabled        types.Bool    `tfsdk:"enabled"`
	TimeoutSeconds types.Int64   `tfsdk:"timeout_seconds"`
	PriorityWeight types.Float64 `tfsdk:"priority_weight"`
	Tags           types.Set     `tfsdk:"tags"`
	Labels         types.Map     `tfsdk:"labels"`
	NodeIDs        types.List    `tfsdk:"node_ids"`
	Schedule       *ScheduleModel      `tfsdk:"schedule"`
	InputMappings  []InputMappingModel `tfsdk:"input_mappings"`
	APIToken       types.String  `tfsdk:"api_token"`
}

type ScheduleModel struct {
	CronExpression types.String `tfsdk:"cron_expression"`
	Timezone       types.String `tfsdk:"timezone"`
	Enabled        types.Bool   `tfsdk:"enabled"`
}

type InputMappingModel struct {
	ParameterName types.String `tfsdk:"parameter_name"`
	NodeID        types.String `tfsdk:"node_id"`
	FieldName     types.String `tfsdk:"field_name"`
	DefaultValue  types.String `tfsdk:"default_value"`
}
```

### Example HCL Configuration

```hcl
resource "comfyui_workflow" "image_gen" {
  name          = "image-generation-pipeline"
  description   = "Generates images from text prompts"
  workflow_json = file("${path.module}/workflows/image_gen.json")

  enabled         = true
  timeout_seconds = 900
  priority_weight = 0.8
  max_queue_size  = 50

  tags   = ["production", "image-gen"]
  labels = { team = "ml-ops", cost_center = "ai-platform" }

  schedule {
    cron_expression = "0 */6 * * *"
    timezone        = "America/New_York"
    enabled         = true
  }

  input_mappings {
    parameter_name = "prompt"
    node_id        = "3"
    field_name     = "text"
  }
  input_mappings {
    parameter_name = "seed"
    node_id        = "5"
    field_name     = "seed"
    default_value  = "42"
  }

  api_token = var.comfyui_api_token
}
```

---

## References

- [Schema Documentation](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/schemas)
- [Attribute Types](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/types)
- [Nested Attributes](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#nested-attributes)
- [Blocks](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/blocks)
- [Plan Modifiers](https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification)
- [Sensitive Values](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/schemas#sensitive)
