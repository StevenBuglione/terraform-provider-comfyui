# 21 — Reference Provider: terraform-provider-aws

> **Purpose**: Analyze the architecture, patterns, and conventions of the largest
> Terraform provider as a reference implementation for building our own provider.

---

## 1. Overview

`terraform-provider-aws` is the single largest Terraform provider in existence.
It is maintained by HashiCorp and the open-source community and covers the vast
majority of the AWS API surface area.

| Metric | Value |
|---|---|
| Resources | 1 300+ |
| Data Sources | 600+ |
| Go SDK | AWS SDK for Go v2 (`aws-sdk-go-v2`) |
| Plugin Framework | Plugin SDKv2 (ongoing Framework migration) |
| Repository | <https://github.com/hashicorp/terraform-provider-aws> |
| Dev Docs | <https://hashicorp.github.io/terraform-provider-aws/> |
| License | MPL-2.0 |

The provider was originally written entirely against **Plugin SDKv2**
(`github.com/hashicorp/terraform-plugin-sdk/v2`). Newer resources are being
written with the **Terraform Plugin Framework**
(`github.com/hashicorp/terraform-plugin-framework`), and the two coexist via
**terraform-plugin-mux**. This hybrid approach is the pragmatic path that any
large provider follows.

---

## 2. Repository Structure

```
terraform-provider-aws/
├── main.go                      # Entry point — calls plugin.Serve
├── internal/
│   ├── provider/                # Provider-level configuration & schema
│   │   ├── provider.go          # Returns *schema.Provider (SDKv2) or combines muxed servers
│   │   └── provider_test.go
│   ├── service/                 # Alias — actual code lives in services/
│   ├── services/                # *** One package per AWS service ***
│   │   ├── ec2/
│   │   │   ├── instance.go             # aws_instance resource
│   │   │   ├── instance_test.go        # Acceptance + unit tests
│   │   │   ├── vpc.go                  # aws_vpc resource
│   │   │   ├── security_group.go
│   │   │   ├── find.go                 # Finder/waiter helpers
│   │   │   ├── tags_gen.go             # Auto-generated tagging
│   │   │   └── ...
│   │   ├── s3/
│   │   ├── iam/
│   │   ├── lambda/
│   │   └── ... (200+ service packages)
│   ├── conns/                   # Connection / client management
│   │   ├── config.go            # AWSClient struct, session init
│   │   └── conns.go             # Per-service client accessors
│   ├── flex/                    # Type conversion helpers
│   │   ├── flex.go              # Flatten / expand between TF ↔ SDK types
│   │   └── flex_test.go
│   ├── verify/                  # Validation helpers
│   │   ├── validate.go          # ARN validators, CIDR validators, etc.
│   │   └── ...
│   ├── acctest/                 # Acceptance test helpers
│   │   ├── acctest.go           # PreCheck, providers, random names
│   │   └── ...
│   ├── errs/                    # Error handling wrappers
│   ├── tfresource/              # Retry / waiter utilities
│   └── generate/                # Code generation tooling
├── docs/                        # Internal contributor documentation
│   ├── provider-design.md
│   ├── error-handling.md
│   ├── retries-and-waiters.md
│   └── ...
├── website/docs/                # HashiCorp Registry documentation (HCL examples)
│   ├── r/                       # Resource docs (e.g. r/instance.html.markdown)
│   ├── d/                       # Data source docs
│   └── index.html.markdown      # Provider doc
├── GNUmakefile
├── go.mod
└── go.sum
```

### Why This Matters

The service-per-package layout means that adding a new AWS service never touches
existing packages. Each service package is a self-contained unit with its own
resources, data sources, tests, and helpers. This is the model we should follow
even for a provider with only a handful of resources.

---

## 3. Key Patterns

### 3.1 Service-Based Package Organization

Every AWS service gets its own Go package under `internal/services/<service>/`.
The package registers its resources and data sources with the provider at init
time through a `ServicePackage` type:

```go
// internal/services/ec2/service_package_gen.go (auto-generated)
package ec2

import (
    "context"
    "github.com/hashicorp/terraform-provider-aws/internal/types"
)

type servicePackage struct{}

func (p *servicePackage) SDKResources(ctx context.Context) []*types.ServicePackageSDKResource {
    return []*types.ServicePackageSDKResource{
        {
            Factory: ResourceInstance,
            TypeName: "aws_instance",
        },
        {
            Factory: ResourceVPC,
            TypeName: "aws_vpc",
        },
        // ...
    }
}
```

The provider iterates over all service packages at startup to build the full
resource map. This pattern is **registration-based** rather than having one
massive switch statement.

### 3.2 Shared Connection Management (`internal/conns`)

The `conns` package holds a single `AWSClient` struct that contains pre-configured
API clients for every service:

```go
// internal/conns/config.go (simplified)
type AWSClient struct {
    AccountID   string
    Region      string
    EC2Client   *ec2.Client
    S3Client    *s3.Client
    IAMClient   *iam.Client
    // ... hundreds more
}
```

Resource CRUD functions receive the provider's `meta` argument and type-assert
it to `*conns.AWSClient`:

```go
func resourceInstanceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
    conn := meta.(*conns.AWSClient).EC2Client(ctx)
    // Use conn to make API calls ...
}
```

**Lesson**: Even a small provider should centralize client construction in a
single struct so every resource gets a pre-configured, authenticated client.

### 3.3 Type Flattening and Expansion (`internal/flex`)

Terraform state stores everything as `map[string]interface{}` (SDKv2) or as
Framework attribute types. AWS SDK types are strongly typed structs. The `flex`
package provides bidirectional helpers:

```go
// Expand: Terraform → SDK
func ExpandStringValueList(tfList []interface{}) []string { ... }

// Flatten: SDK → Terraform
func FlattenStringValueList(apiList []string) []interface{} { ... }
```

These are used pervasively in every Read and Create function. The pattern avoids
repetitive type-conversion boilerplate scattered across resources.

### 3.4 Tag Management

Tags are so common across AWS resources that the provider auto-generates tagging
code. Each service package gets `tags_gen.go` with functions like:

```go
func Tags(tags tftags.KeyValueTags) []types.Tag { ... }
func KeyValueTags(ctx context.Context, tags []types.Tag) tftags.KeyValueTags { ... }
```

The provider also has `default_tags` at the provider level that automatically
merge into every resource's tags. This is a sophisticated pattern; for a small
provider, simpler manual tag handling is fine.

### 3.5 Waiters for Eventual Consistency

AWS APIs are eventually consistent. The provider uses **retry waiters** to poll
until a resource reaches a desired state:

```go
func waitInstanceRunning(ctx context.Context, conn *ec2.Client, id string, timeout time.Duration) (*ec2.Instance, error) {
    stateConf := &retry.StateChangeConf{
        Pending: []string{"pending"},
        Target:  []string{"running"},
        Refresh: statusInstance(ctx, conn, id),
        Timeout: timeout,
    }
    outputRaw, err := stateConf.WaitForStateContext(ctx)
    // ...
}
```

This is from `github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry`. The
pattern is reusable for any provider that talks to async APIs.

### 3.6 Error Retries with Backoff

The `tfresource` and `errs` packages provide wrappers for retryable errors:

```go
err := retry.RetryContext(ctx, propagationTimeout, func() *retry.RetryError {
    _, err := conn.CreateBucket(ctx, input)
    if tfawserr.ErrCodeEquals(err, "OperationAborted") {
        return retry.RetryableError(err)
    }
    if err != nil {
        return retry.NonRetryableError(err)
    }
    return nil
})
```

**Lesson**: Classify errors as retryable or not. Wrap retries around known
transient failure codes.

### 3.7 Acceptance Test Helpers (`internal/acctest`)

```go
func TestAccInstance_basic(t *testing.T) {
    resource.ParallelTest(t, resource.TestCase{
        PreCheck:                 func() { acctest.PreCheck(ctx, t) },
        ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
        CheckDestroy:             testAccCheckInstanceDestroy(ctx),
        Steps: []resource.TestStep{
            {
                Config: testAccInstanceConfig_basic(),
                Check: resource.ComposeTestCheckFunc(
                    testAccCheckInstanceExists(ctx, "aws_instance.test"),
                    resource.TestCheckResourceAttr("aws_instance.test", "instance_type", "t3.micro"),
                ),
            },
        },
    })
}
```

The `acctest` package provides:
- `PreCheck` — skip tests when credentials are missing
- `ProtoV5ProviderFactories` — pre-built provider factory map
- `RandomWithPrefix` — generate unique names to avoid conflicts
- `CheckDestroy` helpers — verify resources are cleaned up

---

## 4. Makefile Cheat Sheet (AWS Provider)

The AWS provider's `GNUmakefile` includes:

| Target | Description |
|---|---|
| `make build` | `go install .` |
| `make gen` | Run all code generators |
| `make test` | Unit tests (`go test ./...`) |
| `make testacc` | Acceptance tests (`TF_ACC=1 go test ./...`) |
| `make lint` | Run `golangci-lint` |
| `make docs-lint` | Validate registry documentation |
| `make website` | Build website docs locally |
| `make tools` | Install required tooling |

Full reference: <https://hashicorp.github.io/terraform-provider-aws/makefile-cheat-sheet/>

---

## 5. Contributor Guide Highlights

1. **One resource per file** — `internal/services/<svc>/<resource>.go`
2. **Tests beside code** — `<resource>_test.go` in the same package
3. **Use `d.SetId("")`** to signal deletion in Read functions
4. **Return `diag.Diagnostics`**, not bare errors
5. **Never hard-code regions** — always use the provider-configured region
6. **Changelog entry required** — `.changelog/<PR>.txt` using `changie`
7. **Run `make gen`** after modifying any generated code

---

## 6. What to Learn for Smaller Providers

| AWS Pattern | Applicability to Small Providers |
|---|---|
| Service-per-package | ✅ Even with 3 services, keep packages separate |
| Shared `AWSClient` struct | ✅ Create a single `ProviderClient` struct |
| `flex` helpers | ⚠️ Useful if your API SDK has different types |
| Tag auto-generation | ❌ Overkill — handle tags manually |
| Waiter pattern | ✅ Essential if your API is async |
| Error retries | ✅ Essential for any HTTP API |
| `acctest` helpers | ✅ Start with a small acctest package from day one |
| Code generation | ❌ Not needed until 50+ resources |
| Plugin mux (SDKv2 + Framework) | ❌ Pick Framework only for new providers |

**Key takeaway**: Copy the *organizational patterns* (package layout, client
struct, test helpers) but not the *scale-driven machinery* (code generation,
auto-tagging, muxing). A new provider should be Framework-only from day one.

---

## 7. References

- **Repository**: <https://github.com/hashicorp/terraform-provider-aws>
- **Developer Documentation**: <https://hashicorp.github.io/terraform-provider-aws/>
- **Makefile Cheat Sheet**: <https://hashicorp.github.io/terraform-provider-aws/makefile-cheat-sheet/>
- **Contribution Guide**: <https://github.com/hashicorp/terraform-provider-aws/blob/main/.github/CONTRIBUTING.md>
- **Retries and Waiters**: <https://hashicorp.github.io/terraform-provider-aws/retries-and-waiters/>
- **Error Handling**: <https://hashicorp.github.io/terraform-provider-aws/error-handling/>
- **Plugin Framework Migration**: <https://developer.hashicorp.com/terraform/plugin/framework>

---

*Next*: [22-reference-provider-azurerm.md](22-reference-provider-azurerm.md) —
Azure provider architecture and patterns.
