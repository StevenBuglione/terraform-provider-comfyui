# Provider-Defined Functions (Terraform 1.8+)

> Research reference for the AI coding harness. All examples use the
> **terraform-plugin-framework** (Plugin Framework). SDKv2 is NOT covered.

---

## 1. What Provider Functions Are

Starting with Terraform 1.8, providers can expose **custom computational functions** that practitioners call directly in HCL. Unlike data sources, provider functions are **pure computations** — no state, no API calls, no side effects.

### HCL Syntax

```hcl
# provider::<provider_name>::<function_name>(args...)
locals {
  parsed = provider::comfyui::parse_id("workflow-abc-123")
}
```

### Functions vs Data Sources

| Aspect | Provider Function | Data Source |
|---|---|---|
| Side effects | **None** — pure computation | May call remote APIs |
| Managed state | **No** | Yes — stored in state |
| Execution phase | Plan and apply | Refresh/plan/apply |
| Error model | `function.FuncError` | `diag.Diagnostics` |
| Terraform version | **1.8+** | Any |
| HCL call style | `provider::name::fn(args)` | `data "type" "name" {}` |

**Rule of thumb:** deterministic transformation of inputs (string parsing, ID formatting, encoding) → function. Needs credentials, network, or dependency graph → data source.

---

## 2. The `function.Function` Interface

```go
type Function interface {
    Metadata(ctx context.Context, req MetadataRequest, resp *MetadataResponse)
    Definition(ctx context.Context, req DefinitionRequest, resp *DefinitionResponse)
    Run(ctx context.Context, req RunRequest, resp *RunResponse)
}
```

- **Metadata** — sets `resp.Name`, which becomes the callable name after `provider::<provider>::`.
- **Definition** — declares `Parameters` (ordered), optional `VariadicParameter`, `Return` type, `Summary`/`Description`.
- **Run** — reads args from `req.Arguments.Get(ctx, &vars...)`, writes result via `resp.Result.Set(ctx, val)`, reports errors via `resp.Error`.

---

## 3. Parameter Types

Parameters are positional. Each maps to a Go type via `req.Arguments.Get`.

| Framework Type | Go Type | Extra Fields |
|---|---|---|
| `function.StringParameter` | `string` / `types.String` | — |
| `function.Int64Parameter` | `int64` / `types.Int64` | — |
| `function.BoolParameter` | `bool` / `types.Bool` | — |
| `function.Float64Parameter` | `float64` / `types.Float64` | — |
| `function.ListParameter` | `types.List` | `ElementType` required |
| `function.SetParameter` | `types.Set` | `ElementType` required |
| `function.MapParameter` | `types.Map` | `ElementType` required |
| `function.ObjectParameter` | `types.Object` | `AttributeTypes` required |
| `function.DynamicParameter` | `types.Dynamic` | — |

All parameters have `Name` and `Description` fields. Set `AllowNullValue: true` and use Framework types (`types.String`) to accept nulls.

### Examples

```go
function.StringParameter{Name: "input", Description: "The raw string to process."}

function.Int64Parameter{Name: "count", Description: "Number of items."}

function.BoolParameter{Name: "enabled", Description: "Whether the feature is on."}

function.ListParameter{
    Name: "tags", Description: "List of tag strings.",
    ElementType: types.StringType,
}

function.MapParameter{
    Name: "labels", Description: "Key-value pairs.",
    ElementType: types.StringType,
}

function.ObjectParameter{
    Name: "config", Description: "A configuration object.",
    AttributeTypes: map[string]attr.Type{
        "host": types.StringType,
        "port": types.Int64Type,
    },
}
```

---

## 4. Return Types

Exactly **one** return type per function. Determines what `resp.Result.Set` accepts.

| Framework Type | Go Type | Extra Fields |
|---|---|---|
| `function.StringReturn` | `string` / `types.String` | — |
| `function.Int64Return` | `int64` / `types.Int64` | — |
| `function.BoolReturn` | `bool` / `types.Bool` | — |
| `function.Float64Return` | `float64` / `types.Float64` | — |
| `function.ListReturn` | `types.List` | `ElementType` |
| `function.SetReturn` | `types.Set` | `ElementType` |
| `function.MapReturn` | `types.Map` | `ElementType` |
| `function.ObjectReturn` | `types.Object` | `AttributeTypes` |
| `function.DynamicReturn` | `types.Dynamic` | — |

```go
Return: function.ListReturn{ElementType: types.StringType}
Return: function.ObjectReturn{AttributeTypes: map[string]attr.Type{"name": types.StringType, "count": types.Int64Type}}
```

---

## 5. Variadic Parameters

A function may accept zero or more trailing arguments of the same type via `VariadicParameter`. Only one is allowed; it is always last.

```go
resp.Definition = function.Definition{
    Summary: "Joins strings with a separator",
    Parameters: []function.Parameter{
        function.StringParameter{Name: "separator", Description: "The separator."},
    },
    VariadicParameter: function.StringParameter{
        Name: "parts", Description: "The strings to join.",
    },
    Return: function.StringReturn{},
}
```

```hcl
output "joined" {
  value = provider::comfyui::join_strings("-", "alpha", "beta", "gamma")
}
```

Read variadic args as a slice — one variable per fixed param, then a slice:

```go
func (f JoinStringsFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
    var separator string
    var parts []string
    resp.Error = function.ConcatFuncErrors(req.Arguments.Get(ctx, &separator, &parts))
    if resp.Error != nil {
        return
    }
    resp.Error = function.ConcatFuncErrors(resp.Result.Set(ctx, strings.Join(parts, separator)))
}
```

---

## 6. Complete Implementation Example

```go
package provider

import (
    "context"
    "strings"

    "github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = ParseIDFunction{}

type ParseIDFunction struct{}

func (f ParseIDFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
    resp.Name = "parse_id"
}

func (f ParseIDFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
    resp.Definition = function.Definition{
        Summary:     "Parses a composite ID",
        Parameters: []function.Parameter{
            function.StringParameter{
                Name:        "id",
                Description: "The composite ID",
            },
        },
        Return: function.StringReturn{},
    }
}

func (f ParseIDFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
    var id string
    resp.Error = function.ConcatFuncErrors(req.Arguments.Get(ctx, &id))
    if resp.Error != nil {
        return
    }

    parts := strings.SplitN(id, "-", 2)
    if len(parts) < 2 {
        resp.Error = function.NewArgumentFuncError(0, "id must contain at least one hyphen")
        return
    }
    parsed := parts[1]

    resp.Error = function.ConcatFuncErrors(resp.Result.Set(ctx, parsed))
}
```

```hcl
output "parsed" {
  value = provider::comfyui::parse_id("workflow-abc-123")
  # → "abc-123"
}
```

---

## 7. Registering Functions in the Provider

Implement `provider.ProviderWithFunctions` — return constructor functions:

```go
func (p *ComfyUIProvider) Functions(ctx context.Context) []func() function.Function {
    return []func() function.Function{
        func() function.Function { return ParseIDFunction{} },
        func() function.Function { return JoinStringsFunction{} },
    }
}
```

Each constructor returns a **new instance**. Functions do not have access to the configured provider (API clients, etc.) by default — if needed, embed a pointer to the provider struct at construction time.

---

## 8. Testing Provider Functions

Functions execute during planning, so tests use plan-time checks. Require Terraform ≥ 1.8 with `tfversion.SkipBelow`.

```go
package provider_test

import (
    "regexp"
    "testing"

    "github.com/hashicorp/terraform-plugin-testing/helper/resource"
    "github.com/hashicorp/terraform-plugin-testing/knownvalue"
    "github.com/hashicorp/terraform-plugin-testing/plancheck"
    "github.com/hashicorp/terraform-plugin-testing/statecheck"
    "github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestParseIDFunction_basic(t *testing.T) {
    resource.UnitTest(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        TerraformVersionChecks: []tfversion.TerraformVersionCheck{
            tfversion.SkipBelow(tfversion.Version1_8_0),
        },
        Steps: []resource.TestStep{
            {
                Config: `output "test" { value = provider::comfyui::parse_id("workflow-abc-123") }`,
                ConfigPlanChecks: resource.ConfigPlanChecks{
                    PreApply: []plancheck.PlanCheck{
                        plancheck.ExpectKnownOutputValue("test", knownvalue.StringExact("abc-123")),
                    },
                },
            },
        },
    })
}

func TestParseIDFunction_error(t *testing.T) {
    resource.UnitTest(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        TerraformVersionChecks: []tfversion.TerraformVersionCheck{
            tfversion.SkipBelow(tfversion.Version1_8_0),
        },
        Steps: []resource.TestStep{
            {
                Config:      `output "test" { value = provider::comfyui::parse_id("nohyphen") }`,
                ExpectError: regexp.MustCompile(`must contain at least one hyphen`),
            },
        },
    })
}
```

**Key packages:** `plancheck` (plan-time asserts), `statecheck` (post-apply asserts), `knownvalue` (type-safe matchers), `tfversion` (version gating).

Reference: <https://developer.hashicorp.com/terraform/plugin/framework/functions/testing>

---

## 9. Error Handling in Functions

Functions use `function.FuncError`, **not** `diag.Diagnostics`.

```go
// General error
resp.Error = function.NewFuncError("something went wrong")

// Argument-specific error (0-based index) — Terraform highlights the offending arg
resp.Error = function.NewArgumentFuncError(0, "id must not be empty")
resp.Error = function.NewArgumentFuncError(1, "count must be positive")

// Combine errors (nil values are safely ignored)
resp.Error = function.ConcatFuncErrors(
    req.Arguments.Get(ctx, &arg1, &arg2),
)

// Chain custom errors onto framework errors
resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(0, "custom validation failed"))
```

### Complete Pattern

```go
func (f MyFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
    var input string
    var count int64
    resp.Error = function.ConcatFuncErrors(req.Arguments.Get(ctx, &input, &count))
    if resp.Error != nil {
        return
    }
    if input == "" {
        resp.Error = function.NewArgumentFuncError(0, "input must not be empty")
        return
    }
    if count < 1 {
        resp.Error = function.NewArgumentFuncError(1, "count must be at least 1")
        return
    }
    resp.Error = function.ConcatFuncErrors(resp.Result.Set(ctx, strings.Repeat(input, int(count))))
}
```

---

## References

- [Provider Functions — Overview](https://developer.hashicorp.com/terraform/plugin/framework/functions)
- [Provider Functions — Parameters](https://developer.hashicorp.com/terraform/plugin/framework/functions/parameters)
- [Provider Functions — Returns](https://developer.hashicorp.com/terraform/plugin/framework/functions/returns)
- [Provider Functions — Errors](https://developer.hashicorp.com/terraform/plugin/framework/functions/errors)
- [Provider Functions — Testing](https://developer.hashicorp.com/terraform/plugin/framework/functions/testing)
- [`terraform-plugin-framework` — `function` package](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework/function)
