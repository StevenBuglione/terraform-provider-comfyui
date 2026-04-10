# Unit Testing for Terraform Providers (Plugin Framework)

## Overview

Unit tests verify **individual functions, validators, plan modifiers, and helper logic** in isolation — without running real Terraform operations or contacting any API. They are fast, deterministic, and do not require credentials or infrastructure.

### When to Use Unit Tests vs Acceptance Tests

| Criteria | Unit Tests | Acceptance Tests |
|---|---|---|
| **Speed** | Milliseconds | Seconds to minutes |
| **External dependencies** | None | Real API / infrastructure |
| **Scope** | Single function or component | Full Terraform lifecycle |
| **Credentials required** | No | Yes |
| **When to use** | Validators, plan modifiers, helpers, parsing logic, data transformations | Resource CRUD, data source reads, import, full provider behavior |
| **Gate** | Always run (`go test`) | Gated behind `TF_ACC=1` |

**Rule of thumb:** If the logic can be tested without Terraform state, plans, or applies, write a unit test. If it requires the Terraform lifecycle, write an acceptance test.

---

## Test File Organization

Go convention: test files live **alongside** the code they test, with a `_test.go` suffix.

```
internal/
  provider/
    provider.go
    provider_test.go          # Tests for provider configuration
    workflow_resource.go
    workflow_resource_test.go  # Acceptance tests for workflow resource
    validators.go
    validators_test.go        # Unit tests for custom validators
    helpers.go
    helpers_test.go           # Unit tests for helper functions
    planmodifiers.go
    planmodifiers_test.go     # Unit tests for plan modifiers
```

Go test files in the same package can access unexported (lowercase) functions. This is intentional — unit tests should be able to test internal helpers.

---

## Table-Driven Tests (Standard Go Pattern)

Table-driven tests are the idiomatic Go approach to testing multiple input/output cases. They are the **preferred** pattern throughout the Go ecosystem and all Terraform provider codebases.

### Basic Pattern

```go
// internal/provider/helpers_test.go
package provider

import "testing"

func TestParseWorkflowID(t *testing.T) {
    t.Parallel()

    tests := map[string]struct {
        input     string
        expected  string
        expectErr bool
    }{
        "valid UUID": {
            input:    "550e8400-e29b-41d4-a716-446655440000",
            expected: "550e8400-e29b-41d4-a716-446655440000",
        },
        "valid short ID": {
            input:    "wf-12345",
            expected: "wf-12345",
        },
        "empty string": {
            input:     "",
            expectErr: true,
        },
        "whitespace only": {
            input:     "   ",
            expectErr: true,
        },
        "contains invalid characters": {
            input:     "wf/../../etc/passwd",
            expectErr: true,
        },
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()

            result, err := parseWorkflowID(tc.input)

            if tc.expectErr {
                if err == nil {
                    t.Fatalf("expected error, got nil")
                }
                return
            }

            if err != nil {
                t.Fatalf("unexpected error: %s", err)
            }

            if result != tc.expected {
                t.Errorf("expected %q, got %q", tc.expected, result)
            }
        })
    }
}
```

### Why maps, not slices?

Using `map[string]struct{...}` for the test table instead of a slice of structs:

- Each case has a **descriptive name** (the map key) that becomes the `t.Run` subtest name.
- Go randomizes map iteration order, which helps surface order-dependent bugs.
- It's easier to add/remove cases without adjusting indices.

---

## Testing Helper Functions

Helpers that transform data between API responses and Terraform state are prime unit test targets.

```go
// internal/provider/helpers.go
package provider

import "strings"

// flattenTags converts a map[string]string to the format Terraform expects.
func flattenTags(tags map[string]string) map[string]string {
    result := make(map[string]string, len(tags))
    for k, v := range tags {
        result[strings.ToLower(k)] = v
    }
    return result
}

// expandNodeNames converts a comma-separated string to a slice.
func expandNodeNames(input string) []string {
    if input == "" {
        return nil
    }
    parts := strings.Split(input, ",")
    result := make([]string, 0, len(parts))
    for _, p := range parts {
        trimmed := strings.TrimSpace(p)
        if trimmed != "" {
            result = append(result, trimmed)
        }
    }
    return result
}
```

```go
// internal/provider/helpers_test.go
package provider

import (
    "reflect"
    "testing"
)

func TestFlattenTags(t *testing.T) {
    t.Parallel()

    tests := map[string]struct {
        input    map[string]string
        expected map[string]string
    }{
        "mixed case keys": {
            input:    map[string]string{"Environment": "prod", "TEAM": "infra"},
            expected: map[string]string{"environment": "prod", "team": "infra"},
        },
        "already lowercase": {
            input:    map[string]string{"env": "dev"},
            expected: map[string]string{"env": "dev"},
        },
        "empty map": {
            input:    map[string]string{},
            expected: map[string]string{},
        },
        "nil map": {
            input:    nil,
            expected: map[string]string{},
        },
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            result := flattenTags(tc.input)
            if !reflect.DeepEqual(result, tc.expected) {
                t.Errorf("expected %v, got %v", tc.expected, result)
            }
        })
    }
}

func TestExpandNodeNames(t *testing.T) {
    t.Parallel()

    tests := map[string]struct {
        input    string
        expected []string
    }{
        "single node": {
            input:    "KSampler",
            expected: []string{"KSampler"},
        },
        "multiple nodes": {
            input:    "KSampler,CLIPTextEncode,VAEDecode",
            expected: []string{"KSampler", "CLIPTextEncode", "VAEDecode"},
        },
        "with spaces": {
            input:    " KSampler , CLIPTextEncode , VAEDecode ",
            expected: []string{"KSampler", "CLIPTextEncode", "VAEDecode"},
        },
        "empty string": {
            input:    "",
            expected: nil,
        },
        "only commas": {
            input:    ",,,",
            expected: nil,
        },
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            result := expandNodeNames(tc.input)
            if !reflect.DeepEqual(result, tc.expected) {
                t.Errorf("expected %v, got %v", tc.expected, result)
            }
        })
    }
}
```

---

## Testing Custom Validators

The Plugin Framework lets you create custom validators for schema attributes. These validators implement interfaces like `validator.String`, `validator.Int64`, etc. They can be tested in isolation using the framework's test utilities.

```go
// internal/provider/validators.go
package provider

import (
    "context"
    "regexp"

    "github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// workflowNameValidator validates that a workflow name matches a pattern.
type workflowNameValidator struct{}

func (v workflowNameValidator) Description(_ context.Context) string {
    return "must be 1-64 characters, alphanumeric, hyphens, and underscores only"
}

func (v workflowNameValidator) MarkdownDescription(ctx context.Context) string {
    return v.Description(ctx)
}

func (v workflowNameValidator) ValidateString(
    ctx context.Context,
    req validator.StringRequest,
    resp *validator.StringResponse,
) {
    if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
        return
    }

    value := req.ConfigValue.ValueString()
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]{1,64}$`, value)
    if !matched {
        resp.Diagnostics.AddAttributeError(
            req.Path,
            "Invalid Workflow Name",
            v.Description(ctx),
        )
    }
}
```

```go
// internal/provider/validators_test.go
package provider

import (
    "context"
    "testing"

    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/schema/validator"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkflowNameValidator(t *testing.T) {
    t.Parallel()

    tests := map[string]struct {
        value       types.String
        expectError bool
    }{
        "valid simple name": {
            value: types.StringValue("my-workflow"),
        },
        "valid with underscores": {
            value: types.StringValue("my_workflow_v2"),
        },
        "valid single char": {
            value: types.StringValue("a"),
        },
        "invalid contains spaces": {
            value:       types.StringValue("my workflow"),
            expectError: true,
        },
        "invalid contains dots": {
            value:       types.StringValue("my.workflow"),
            expectError: true,
        },
        "invalid too long": {
            value:       types.StringValue(string(make([]byte, 65))),
            expectError: true,
        },
        "invalid empty": {
            value:       types.StringValue(""),
            expectError: true,
        },
        "null value is ok": {
            value: types.StringNull(),
        },
        "unknown value is ok": {
            value: types.StringUnknown(),
        },
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()

            req := validator.StringRequest{
                Path:        path.Root("name"),
                ConfigValue: tc.value,
            }
            resp := &validator.StringResponse{}

            v := workflowNameValidator{}
            v.ValidateString(context.Background(), req, resp)

            if tc.expectError && !resp.Diagnostics.HasError() {
                t.Fatal("expected validation error, got none")
            }
            if !tc.expectError && resp.Diagnostics.HasError() {
                t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
            }
        })
    }
}
```

---

## Testing Plan Modifiers

Plan modifiers (e.g., `UseStateForUnknown`, `RequiresReplace`, or custom ones) can also be unit-tested.

```go
// internal/provider/planmodifiers.go
package provider

import (
    "context"

    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// lowercasePlanModifier converts a string value to lowercase during planning.
type lowercasePlanModifier struct{}

func (m lowercasePlanModifier) Description(_ context.Context) string {
    return "converts value to lowercase"
}

func (m lowercasePlanModifier) MarkdownDescription(ctx context.Context) string {
    return m.Description(ctx)
}

func (m lowercasePlanModifier) PlanModifyString(
    ctx context.Context,
    req planmodifier.StringRequest,
    resp *planmodifier.StringResponse,
) {
    if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
        return
    }

    resp.PlanValue = types.StringValue(
        strings.ToLower(req.PlanValue.ValueString()),
    )
}
```

```go
// internal/provider/planmodifiers_test.go
package provider

import (
    "context"
    "testing"

    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

func TestLowercasePlanModifier(t *testing.T) {
    t.Parallel()

    tests := map[string]struct {
        planValue types.String
        expected  types.String
    }{
        "mixed case": {
            planValue: types.StringValue("MyWorkflow"),
            expected:  types.StringValue("myworkflow"),
        },
        "already lowercase": {
            planValue: types.StringValue("myworkflow"),
            expected:  types.StringValue("myworkflow"),
        },
        "null value unchanged": {
            planValue: types.StringNull(),
            expected:  types.StringNull(),
        },
        "unknown value unchanged": {
            planValue: types.StringUnknown(),
            expected:  types.StringUnknown(),
        },
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()

            req := planmodifier.StringRequest{
                PlanValue: tc.planValue,
            }
            resp := &planmodifier.StringResponse{
                PlanValue: tc.planValue,
            }

            m := lowercasePlanModifier{}
            m.PlanModifyString(context.Background(), req, resp)

            if !resp.PlanValue.Equal(tc.expected) {
                t.Errorf("expected %s, got %s", tc.expected, resp.PlanValue)
            }
        })
    }
}
```

---

## Mocking API Clients

When your helper functions or resource logic depends on an API client, use interfaces to enable mocking.

### Define an interface

```go
// internal/client/client.go
package client

type ComfyUIClient interface {
    GetWorkflow(id string) (*Workflow, error)
    CreateWorkflow(req CreateWorkflowRequest) (*Workflow, error)
    DeleteWorkflow(id string) error
}

type Workflow struct {
    ID   string
    Name string
}
```

### Create a mock

```go
// internal/client/mock_client_test.go
package client

import "fmt"

type mockClient struct {
    workflows map[string]*Workflow
}

func newMockClient() *mockClient {
    return &mockClient{
        workflows: make(map[string]*Workflow),
    }
}

func (m *mockClient) GetWorkflow(id string) (*Workflow, error) {
    wf, ok := m.workflows[id]
    if !ok {
        return nil, fmt.Errorf("workflow %s not found", id)
    }
    return wf, nil
}

func (m *mockClient) CreateWorkflow(req CreateWorkflowRequest) (*Workflow, error) {
    wf := &Workflow{ID: "mock-id", Name: req.Name}
    m.workflows[wf.ID] = wf
    return wf, nil
}

func (m *mockClient) DeleteWorkflow(id string) error {
    delete(m.workflows, id)
    return nil
}
```

### Use the mock in tests

```go
func TestSomeHelperThatUsesClient(t *testing.T) {
    client := newMockClient()
    client.workflows["existing-id"] = &Workflow{
        ID:   "existing-id",
        Name: "test-workflow",
    }

    wf, err := client.GetWorkflow("existing-id")
    if err != nil {
        t.Fatalf("unexpected error: %s", err)
    }
    if wf.Name != "test-workflow" {
        t.Errorf("expected name %q, got %q", "test-workflow", wf.Name)
    }
}
```

---

## Using Testify Assertions

While the standard library `testing` package is sufficient, many Go projects use `github.com/stretchr/testify` for more expressive assertions.

### Install

```bash
go get github.com/stretchr/testify
```

### `assert` vs `require`

- **`assert`** — Reports failures but continues the test. Use for non-fatal checks.
- **`require`** — Reports failures and **stops** the test immediately. Use when subsequent logic depends on this check passing.

```go
import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExpandNodeNames_Testify(t *testing.T) {
    // require stops the test if this fails — subsequent lines won't run
    result := expandNodeNames("KSampler,CLIPTextEncode")
    require.NotNil(t, result)
    require.Len(t, result, 2)

    // assert reports but continues — we see all failures
    assert.Equal(t, "KSampler", result[0])
    assert.Equal(t, "CLIPTextEncode", result[1])
}
```

### Common assert/require functions

```go
assert.Equal(t, expected, actual)            // Deep equality
assert.NotEqual(t, unexpected, actual)       // Not equal
assert.Nil(t, value)                         // Is nil
assert.NotNil(t, value)                      // Is not nil
assert.True(t, condition)                    // Boolean true
assert.False(t, condition)                   // Boolean false
assert.Contains(t, "hello world", "world")   // Substring/element
assert.Len(t, slice, 3)                      // Length check
assert.Empty(t, value)                       // Zero value or empty
assert.Error(t, err)                         // err is not nil
assert.NoError(t, err)                       // err is nil
assert.ErrorContains(t, err, "not found")    // Error message check
assert.Regexp(t, `^[a-f0-9]+$`, value)      // Regex match
```

---

## Testing Schema Validation Logic

If you have functions that validate configuration before it reaches the Terraform framework, unit-test them directly:

```go
// internal/provider/validation.go
package provider

import "fmt"

func validatePort(port int) error {
    if port < 1 || port > 65535 {
        return fmt.Errorf("port must be between 1 and 65535, got %d", port)
    }
    return nil
}

func validateEndpoint(endpoint string) error {
    if endpoint == "" {
        return fmt.Errorf("endpoint must not be empty")
    }
    // Add URL parsing validation as needed
    return nil
}
```

```go
// internal/provider/validation_test.go
package provider

import "testing"

func TestValidatePort(t *testing.T) {
    t.Parallel()

    tests := map[string]struct {
        port      int
        expectErr bool
    }{
        "valid min":     {port: 1},
        "valid max":     {port: 65535},
        "valid common":  {port: 8188},
        "zero":          {port: 0, expectErr: true},
        "negative":      {port: -1, expectErr: true},
        "too high":      {port: 65536, expectErr: true},
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            err := validatePort(tc.port)
            if tc.expectErr && err == nil {
                t.Error("expected error, got nil")
            }
            if !tc.expectErr && err != nil {
                t.Errorf("unexpected error: %s", err)
            }
        })
    }
}
```

---

## Running Unit Tests

Unit tests run with the standard `go test` command — no special environment variables needed:

```bash
# Run all unit tests (excludes acceptance tests since TF_ACC is not set)
go test ./internal/provider/... -v

# Run tests matching a pattern
go test ./internal/provider/... -v -run TestValidatePort

# Run with race detection
go test ./internal/provider/... -v -race

# Run with coverage
go test ./internal/provider/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Since acceptance tests check for `TF_ACC=1` at the start of each test function, they are automatically skipped when you run plain `go test`.

---

## Summary: What to Unit Test

| Component | Unit Test? | Why |
|---|---|---|
| Custom validators | ✅ Yes | Pure logic, no Terraform state needed |
| Plan modifiers | ✅ Yes | Pure logic with well-defined inputs/outputs |
| Helper functions | ✅ Yes | Data transformation, parsing, formatting |
| Schema validation | ✅ Yes | Input validation rules |
| API client methods | ✅ Yes (with mocks) | Verify request/response handling |
| Resource CRUD | ❌ Use acceptance tests | Requires full Terraform lifecycle |
| Data source reads | ❌ Use acceptance tests | Requires full Terraform lifecycle |
| Provider configuration | ❌ Use acceptance tests | Requires Terraform to inject config |

---

## References

- Go Testing Package: <https://pkg.go.dev/testing>
- Table-Driven Tests: <https://go.dev/wiki/TableDrivenTests>
- Testify: <https://github.com/stretchr/testify>
- Plugin Framework Validator Testing: <https://developer.hashicorp.com/terraform/plugin/framework/validation>
- Plugin Framework Plan Modifier Testing: <https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification>
