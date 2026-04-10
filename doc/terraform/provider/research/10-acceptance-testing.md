# Acceptance Testing for Terraform Providers (Plugin Framework)

## Overview

Acceptance tests are **end-to-end tests** that execute real Terraform operations (plan, apply, destroy) against your provider. They create, read, update, and delete real infrastructure (or real API objects) to verify that your provider works correctly in practice. Unlike unit tests, acceptance tests exercise the full Terraform lifecycle — including state management, plan/apply semantics, and import behavior.

Acceptance tests are the **primary** testing mechanism for Terraform providers. HashiCorp strongly recommends comprehensive acceptance test coverage for every resource and data source.

Key characteristics:

- They run real `terraform plan`, `terraform apply`, and `terraform destroy` under the hood.
- They require real credentials / API access (or a local test server).
- They are gated behind the `TF_ACC=1` environment variable so they don't run accidentally.
- They use the `github.com/hashicorp/terraform-plugin-testing` module.

---

## The `terraform-plugin-testing` Module

All acceptance tests depend on:

```
github.com/hashicorp/terraform-plugin-testing
```

Install it:

```bash
go get github.com/hashicorp/terraform-plugin-testing
```

Primary import paths used in test files:

```go
import (
    "testing"

    "github.com/hashicorp/terraform-plugin-testing/helper/resource"
    "github.com/hashicorp/terraform-plugin-testing/terraform"
)
```

> **Important:** This module is the successor to the older `helper/resource` package that lived inside `terraform-plugin-sdk`. For Plugin Framework providers, always use `terraform-plugin-testing` — it supports protocol version 6 natively.

Repository: <https://github.com/hashicorp/terraform-plugin-testing>

---

## ProtoV6ProviderFactories Setup

Every acceptance test must tell the test harness how to instantiate your provider. For Plugin Framework (protocol v6) providers, you set up `ProtoV6ProviderFactories` **once** in a shared test helper file (conventionally `provider_test.go`):

```go
// internal/provider/provider_test.go
package provider

import (
    "github.com/hashicorp/terraform-plugin-framework/providerserver"
    "github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is a map of provider factories keyed by
// provider name. It is used in every acceptance test's TestCase.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
    "comfyui": providerserver.NewProtocol6WithError(New("test")()),
}
```

Here, `New` is your provider constructor — the function that returns a `provider.Provider`. The `"test"` argument is typically the version string injected during testing.

You reference this variable in every `resource.TestCase`:

```go
resource.TestCase{
    ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
    // ...
}
```

---

## PreCheck Functions

A `PreCheck` function runs before the test harness starts. Use it to validate that required environment variables (credentials, API endpoints) are set. If they are missing, the test is skipped with a clear message.

```go
// internal/provider/provider_test.go
package provider

import (
    "os"
    "testing"
)

func testAccPreCheck(t *testing.T) {
    t.Helper()

    if v := os.Getenv("COMFYUI_API_ENDPOINT"); v == "" {
        t.Fatal("COMFYUI_API_ENDPOINT must be set for acceptance tests")
    }
    // Add additional checks for any required credentials:
    // if v := os.Getenv("COMFYUI_API_KEY"); v == "" {
    //     t.Fatal("COMFYUI_API_KEY must be set for acceptance tests")
    // }
}
```

Every test references it:

```go
PreCheck: func() { testAccPreCheck(t) },
```

---

## Full Test Structure

A complete acceptance test follows this pattern:

```go
// internal/provider/workflow_resource_test.go
package provider

import (
    "fmt"
    "testing"

    "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWorkflowResource_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Step 1: Create and verify
            {
                Config: testAccWorkflowResourceConfig("my-workflow"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr(
                        "comfyui_workflow.test", "name", "my-workflow",
                    ),
                    resource.TestCheckResourceAttrSet(
                        "comfyui_workflow.test", "id",
                    ),
                ),
            },
        },
    })
}

func testAccWorkflowResourceConfig(name string) string {
    return fmt.Sprintf(`
resource "comfyui_workflow" "test" {
  name = %[1]q
}
`, name)
}
```

### What happens when `resource.Test` runs:

1. `PreCheck` executes — aborts early if env is wrong.
2. For each `TestStep`:
   a. The `Config` is applied via `terraform apply`.
   b. The `Check` functions run against the resulting Terraform state.
3. After all steps complete (or on failure), `terraform destroy` is called automatically to clean up.

---

## TestStep Fields

Each `resource.TestStep` supports these important fields:

| Field | Type | Purpose |
|---|---|---|
| `Config` | `string` | The Terraform HCL configuration to apply. |
| `Check` | `resource.TestCheckFunc` | Assertions to run after apply. |
| `ImportState` | `bool` | If `true`, runs `terraform import` instead of apply. |
| `ImportStateVerify` | `bool` | Compares imported state to prior state. |
| `ImportStateId` | `string` | The ID to pass to the import command. |
| `ExpectError` | `*regexp.Regexp` | The step must produce an error matching this regex. |
| `Destroy` | `bool` | If `true`, runs destroy instead of apply. |
| `ExpectNonEmptyPlan` | `bool` | Asserts a non-empty plan after apply (rare). |
| `PlanOnly` | `bool` | Only run plan, do not apply. |

### ExpectError Example

```go
{
    Config: testAccWorkflowResourceConfig(""),
    ExpectError: regexp.MustCompile(`name must not be empty`),
},
```

---

## Check Functions

### `resource.TestCheckResourceAttr`

Verifies an attribute has an exact string value:

```go
resource.TestCheckResourceAttr("comfyui_workflow.test", "name", "my-workflow")
```

### `resource.TestCheckResourceAttrSet`

Verifies an attribute is set (non-empty) without checking the exact value. Useful for computed fields like `id`:

```go
resource.TestCheckResourceAttrSet("comfyui_workflow.test", "id")
```

### `resource.TestCheckNoResourceAttr`

Verifies an attribute is **not** present in state:

```go
resource.TestCheckNoResourceAttr("comfyui_workflow.test", "deprecated_field")
```

### `resource.TestMatchResourceAttr`

Verifies an attribute matches a regular expression:

```go
resource.TestMatchResourceAttr(
    "comfyui_workflow.test", "id",
    regexp.MustCompile(`^[a-f0-9-]{36}$`),
)
```

---

## ComposeAggregateTestCheckFunc vs ComposeTestCheckFunc

Both combine multiple check functions. The difference is error behavior:

- **`ComposeAggregateTestCheckFunc`** — Runs **all** checks and reports **all** failures. Preferred in most cases because you see every assertion that failed, not just the first one.
- **`ComposeTestCheckFunc`** — Stops at the **first** failure. Use when later checks depend on earlier ones passing.

```go
// Preferred: see all failures at once
Check: resource.ComposeAggregateTestCheckFunc(
    resource.TestCheckResourceAttr("comfyui_workflow.test", "name", "my-workflow"),
    resource.TestCheckResourceAttrSet("comfyui_workflow.test", "id"),
    resource.TestCheckResourceAttr("comfyui_workflow.test", "status", "active"),
),
```

---

## Testing Resource Updates (Multi-Step)

To test updates, provide multiple `TestStep` entries with different configs. Terraform detects the diff and runs an update:

```go
func TestAccWorkflowResource_update(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Step 1: Create
            {
                Config: testAccWorkflowResourceConfig("original-name"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr(
                        "comfyui_workflow.test", "name", "original-name",
                    ),
                ),
            },
            // Step 2: Update
            {
                Config: testAccWorkflowResourceConfig("updated-name"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr(
                        "comfyui_workflow.test", "name", "updated-name",
                    ),
                ),
            },
        },
    })
}
```

---

## Testing Resource Import

Import tests verify that `terraform import` correctly populates state:

```go
func TestAccWorkflowResource_import(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Step 1: Create the resource
            {
                Config: testAccWorkflowResourceConfig("importable"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttrSet("comfyui_workflow.test", "id"),
                ),
            },
            // Step 2: Import it
            {
                ResourceName:      "comfyui_workflow.test",
                ImportState:        true,
                ImportStateVerify:  true,
                // If some attributes can't survive import (e.g., write-only),
                // exclude them:
                // ImportStateVerifyIgnore: []string{"password"},
            },
        },
    })
}
```

`ImportStateVerify: true` compares the imported state to the state from step 1. Every attribute must match, unless listed in `ImportStateVerifyIgnore`.

---

## Testing Resource Deletion (CheckDestroy)

`CheckDestroy` runs after all steps complete and the final `terraform destroy` has executed. It verifies that the resource was actually deleted from the remote API:

```go
func TestAccWorkflowResource_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        CheckDestroy:             testAccCheckWorkflowDestroy,
        Steps: []resource.TestStep{
            {
                Config: testAccWorkflowResourceConfig("doomed"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr(
                        "comfyui_workflow.test", "name", "doomed",
                    ),
                ),
            },
        },
    })
}

func testAccCheckWorkflowDestroy(s *terraform.State) error {
    for _, rs := range s.RootModule().Resources {
        if rs.Type != "comfyui_workflow" {
            continue
        }

        // Call your API to verify the resource no longer exists.
        // Return an error if it still does.
        // client := testAccProvider.Meta().(*comfyui.Client)
        // _, err := client.GetWorkflow(rs.Primary.ID)
        // if err == nil {
        //     return fmt.Errorf("workflow %s still exists", rs.Primary.ID)
        // }
    }
    return nil
}
```

---

## The "Disappears" Test Pattern

This pattern tests what happens when a resource is deleted outside of Terraform (e.g., someone deletes it via the UI). The provider should handle the next plan/apply gracefully — typically by detecting the resource is gone and recreating it.

```go
func TestAccWorkflowResource_disappears(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccWorkflowResourceConfig("disappearing"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    // After creation, delete the resource outside Terraform
                    testAccCheckWorkflowDisappears("comfyui_workflow.test"),
                ),
                // The framework detects the resource is gone and reports
                // a non-empty plan (it wants to recreate).
                ExpectNonEmptyPlan: true,
            },
        },
    })
}

func testAccCheckWorkflowDisappears(resourceName string) resource.TestCheckFunc {
    return func(s *terraform.State) error {
        rs, ok := s.RootModule().Resources[resourceName]
        if !ok {
            return fmt.Errorf("resource not found: %s", resourceName)
        }

        // Delete via API:
        // client := testAccProvider.Meta().(*comfyui.Client)
        // err := client.DeleteWorkflow(rs.Primary.ID)
        // if err != nil {
        //     return fmt.Errorf("error deleting workflow: %w", err)
        // }

        _ = rs // placeholder
        return nil
    }
}
```

---

## Running Acceptance Tests

Acceptance tests are gated behind the `TF_ACC` environment variable:

```bash
# Run all acceptance tests in the provider package
TF_ACC=1 go test ./internal/provider -v -timeout 120m

# Run a single test by name
TF_ACC=1 go test ./internal/provider -v -timeout 120m -run TestAccWorkflowResource_basic

# Run tests matching a pattern
TF_ACC=1 go test ./internal/provider -v -timeout 120m -run TestAccWorkflow

# With additional environment variables for your provider
COMFYUI_API_ENDPOINT="http://localhost:8188" \
TF_ACC=1 go test ./internal/provider -v -timeout 120m
```

The `-timeout 120m` flag is important — acceptance tests can be slow because they create real resources.

---

## Test Naming Conventions

Follow this pattern strictly:

```
TestAcc<ResourceOrDataSource>_<scenario>
```

Examples:

```go
func TestAccWorkflowResource_basic(t *testing.T)        {}
func TestAccWorkflowResource_update(t *testing.T)       {}
func TestAccWorkflowResource_import(t *testing.T)       {}
func TestAccWorkflowResource_disappears(t *testing.T)   {}
func TestAccWorkflowResource_fullConfig(t *testing.T)   {}
func TestAccNodeDataSource_basic(t *testing.T)           {}
```

Config helper functions follow:

```
testAcc<ResourceOrDataSource>Config<Variant>
```

Examples:

```go
func testAccWorkflowResourceConfig(name string) string           { ... }
func testAccWorkflowResourceConfigFull(name, desc string) string { ... }
```

---

## Parallel Test Execution and Randomization

### Running tests in parallel

Call `t.Parallel()` at the start of each test to allow Go to run them concurrently:

```go
func TestAccWorkflowResource_basic(t *testing.T) {
    t.Parallel()
    // ...
}
```

### Resource naming with randomization

When tests run in parallel (or on shared infrastructure), resource names must be unique. Use `acctest.RandStringFromCharSet` or `acctest.RandomWithPrefix`:

```go
import "github.com/hashicorp/terraform-plugin-testing/helper/acctest"

func TestAccWorkflowResource_basic(t *testing.T) {
    rName := acctest.RandomWithPrefix("tf-acc-test")

    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccWorkflowResourceConfig(rName),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr(
                        "comfyui_workflow.test", "name", rName,
                    ),
                ),
            },
        },
    })
}
```

This avoids name collisions when multiple tests or CI runners execute simultaneously.

---

## Summary Checklist

For every resource you implement, aim to write these acceptance tests:

| Test | Purpose |
|---|---|
| `TestAcc<Resource>_basic` | Minimal creation and attribute checks. |
| `TestAcc<Resource>_update` | Multi-step: create → update → verify. |
| `TestAcc<Resource>_import` | Create → import → verify state matches. |
| `TestAcc<Resource>_disappears` | Resource deleted externally → handled gracefully. |
| `TestAcc<Resource>_fullConfig` | All optional attributes set and verified. |

---

## References

- HashiCorp Acceptance Testing Guide: <https://developer.hashicorp.com/terraform/plugin/framework/acctests>
- `terraform-plugin-testing` Repository: <https://github.com/hashicorp/terraform-plugin-testing>
- `terraform-plugin-testing` GoDoc: <https://pkg.go.dev/github.com/hashicorp/terraform-plugin-testing>
- Testing Patterns: <https://developer.hashicorp.com/terraform/plugin/testing/acceptance-tests>
