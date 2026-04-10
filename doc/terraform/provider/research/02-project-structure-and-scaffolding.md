# Project Structure and Scaffolding

## 1. The Official Scaffolding Repository

HashiCorp maintains an official starter template for Plugin Framework providers:

> **<https://github.com/hashicorp/terraform-provider-scaffolding-framework>**

This repository provides a minimal but complete starting point that includes:

- A working `main.go` entry point
- A stub provider with one example resource and one data source
- Acceptance tests
- CI/CD via GitHub Actions (test + release workflows)
- GoReleaser configuration for cross-platform binary builds
- A `GNUmakefile` with standard development targets
- Example Terraform configurations for documentation generation
- `tfplugindocs` templates

Always start from this scaffolding rather than building from scratch — it
encodes many non-obvious conventions that the Terraform Registry and tooling
expect.

---

## 2. Complete Directory Layout

Below is the canonical directory structure for a Plugin Framework provider,
annotated with the purpose of each file and directory:

```
terraform-provider-comfyui/
│
├── main.go                              # Entry point — calls providerserver.Serve
├── go.mod                               # Go module definition
├── go.sum                               # Go module checksums
│
├── internal/                            # All provider implementation code
│   └── provider/                        # Provider package
│       ├── provider.go                  # Provider struct, Metadata, Schema, Configure
│       ├── provider_test.go             # Provider-level acceptance tests
│       ├── workflow_resource.go         # comfyui_workflow resource implementation
│       ├── workflow_resource_test.go    # Acceptance tests for workflow resource
│       ├── workflow_data_source.go      # comfyui_workflow data source implementation
│       └── workflow_data_source_test.go # Acceptance tests for workflow data source
│
├── examples/                            # Example Terraform configurations
│   ├── provider/
│   │   └── provider.tf                  # Provider configuration example
│   ├── resources/
│   │   └── comfyui_workflow/
│   │       └── resource.tf              # Resource usage example
│   └── data-sources/
│       └── comfyui_workflow/
│           └── data-source.tf           # Data source usage example
│
├── docs/                                # Generated documentation (by tfplugindocs)
│   ├── index.md                         # Provider index page
│   ├── resources/
│   │   └── workflow.md                  # Resource documentation
│   └── data-sources/
│       └── workflow.md                  # Data source documentation
│
├── templates/                           # tfplugindocs Markdown templates
│   ├── index.md.tmpl                    # Provider doc template
│   └── resources/
│       └── workflow.md.tmpl             # Resource doc template (optional)
│
├── .github/
│   └── workflows/
│       ├── test.yml                     # CI: run tests on every PR
│       └── release.yml                  # CD: build + publish on tag push
│
├── .goreleaser.yml                      # GoReleaser config for cross-compilation
├── GNUmakefile                          # Dev commands (make build, make test, etc.)
├── terraform-registry-manifest.json     # Registry protocol version declaration
├── CHANGELOG.md                         # Version history
├── LICENSE                              # Must be MPL-2.0 for Registry publishing
└── README.md                            # Project overview
```

### 2.1 File-by-File Explanation

#### `main.go`

The sole job of `main.go` is to start the gRPC provider server:

```go
package main

import (
    "context"
    "flag"
    "log"

    "github.com/hashicorp/terraform-plugin-framework/providerserver"
    "github.com/sbuglione/terraform-provider-comfyui/internal/provider"
)

var version string = "dev"

func main() {
    var debug bool
    flag.BoolVar(&debug, "debug", false,
        "set to true to run the provider with support for debuggers like delve")
    flag.Parse()

    opts := providerserver.ServeOpts{
        Address: "registry.terraform.io/sbuglione/comfyui",
        Debug:   debug,
    }

    err := providerserver.Serve(context.Background(), provider.New(version), opts)
    if err != nil {
        log.Fatal(err.Error())
    }
}
```

#### `go.mod`

```go
module github.com/sbuglione/terraform-provider-comfyui

go 1.22

require (
    github.com/hashicorp/terraform-plugin-framework v1.13.0
    github.com/hashicorp/terraform-plugin-go        v0.25.0
    github.com/hashicorp/terraform-plugin-log        v0.9.0
    github.com/hashicorp/terraform-plugin-testing    v1.11.0
)
```

#### `terraform-registry-manifest.json`

This file tells the Terraform Registry which protocol version your provider
supports:

```json
{
  "version": 1,
  "metadata": {
    "protocol_versions": ["6.0"]
  }
}
```

#### `.goreleaser.yml`

GoReleaser cross-compiles the binary for all platforms Terraform supports:

```yaml
version: 2

builds:
  - env:
      - CGO_ENABLED=0
    mod_timestamp: "{{ .CommitTimestamp }}"
    flags:
      - -trimpath
    ldflags:
      - "-s -w -X main.version={{ .Version }}"
    goos:
      - freebsd
      - windows
      - linux
      - darwin
    goarch:
      - amd64
      - "386"
      - arm
      - arm64
    ignore:
      - goos: darwin
        goarch: "386"
    binary: "{{ .ProjectName }}_v{{ .Version }}"

archives:
  - format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "{{ .ProjectName }}_{{ .Version }}_SHA256SUMS"
  algorithm: sha256

signs:
  - artifacts: checksum
    args:
      - "--batch"
      - "--local-user"
      - "{{ .Env.GPG_FINGERPRINT }}"
      - "--output"
      - "${signature}"
      - "--detach-sign"
      - "${artifact}"

release:
  draft: true

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

#### `GNUmakefile`

```makefile
default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

# Build the provider binary
.PHONY: build
build:
	go build -o terraform-provider-comfyui

# Install locally for dev_overrides
.PHONY: install
install:
	go install .

# Generate documentation
.PHONY: docs
docs:
	go generate ./...

# Lint
.PHONY: lint
lint:
	golangci-lint run ./...
```

---

## 3. How to Initialize from Scaffolding

### Step-by-Step

```bash
# 1. Clone the scaffolding repository
git clone https://github.com/hashicorp/terraform-provider-scaffolding-framework.git \
    terraform-provider-comfyui

cd terraform-provider-comfyui

# 2. Remove the upstream git history
rm -rf .git
git init

# 3. Rename the Go module
#    Replace the scaffolding module path with your own
find . -type f -name '*.go' -exec sed -i \
    's|github.com/hashicorp/terraform-provider-scaffolding-framework|github.com/sbuglione/terraform-provider-comfyui|g' {} +

# Also update go.mod
sed -i 's|github.com/hashicorp/terraform-provider-scaffolding-framework|github.com/sbuglione/terraform-provider-comfyui|g' go.mod

# 4. Rename the provider type name in provider.go
#    Change "scaffolding" to "comfyui" in Metadata
sed -i 's|resp.TypeName = "scaffolding"|resp.TypeName = "comfyui"|g' \
    internal/provider/provider.go

# 5. Update the provider address in main.go
sed -i 's|registry.terraform.io/hashicorp/scaffolding|registry.terraform.io/sbuglione/comfyui|g' \
    main.go

# 6. Tidy Go modules
go mod tidy

# 7. Verify it builds
go build -o terraform-provider-comfyui

# 8. Verify tests pass (unit tests only — no TF_ACC)
go test ./...

# 9. Initialize git and make the first commit
git add .
git commit -m "Initial scaffold from terraform-provider-scaffolding-framework"
```

### Post-Scaffolding Checklist

| Task | File(s) | Description |
|---|---|---|
| Rename resource files | `internal/provider/*_resource.go` | Rename from `example_resource.go` to your resource names |
| Rename data source files | `internal/provider/*_data_source.go` | Same pattern for data sources |
| Update `terraform-registry-manifest.json` | Root | Ensure `protocol_versions` is `["6.0"]` |
| Update `.goreleaser.yml` | Root | Verify binary name matches `terraform-provider-comfyui` |
| Update `README.md` | Root | Replace scaffolding docs with your provider description |
| Update `LICENSE` | Root | Must be MPL-2.0 for Terraform Registry publishing |
| Set up `~/.terraformrc` | Home directory | Add `dev_overrides` for local development |
| Update examples | `examples/` | Add real example configurations |
| Update `CHANGELOG.md` | Root | Start your version history |

---

## 4. Why Code Lives in `internal/`

Go's `internal/` directory has special semantics enforced by the Go compiler:

> Packages inside `internal/` can only be imported by code rooted in the **parent
> directory** of `internal/`.

For a Terraform provider, this means:

```
terraform-provider-comfyui/
├── main.go                  # ✅ CAN import internal/provider
├── internal/
│   └── provider/
│       ├── provider.go      # ✅ CAN import internal/provider/subpkg
│       └── subpkg/
│           └── helpers.go
└── other-module/            # ❌ CANNOT import internal/provider
```

### Why This Matters

1. **Encapsulation** — Provider implementation details are not part of the public
   API. Users should only interact via Terraform HCL, not by importing Go packages.

2. **Convention** — The scaffolding and all official HashiCorp providers use
   `internal/`. Deviating from this would confuse contributors and tooling.

3. **API stability** — Keeping code in `internal/` means you can refactor freely
   without worrying about external consumers.

4. **Go module proxy** — The Go module proxy indexes exported packages; `internal/`
   packages are excluded from this indexing.

---

## 5. Alternative Structures for Large Providers

The single-package `internal/provider/` structure works well for small to medium
providers (up to ~20 resources). For larger providers, consider a service-based
package structure.

### 5.1 Service-Based Package Structure

Inspired by the AWS provider (`terraform-provider-aws`), which has hundreds of
resources organized by AWS service:

```
terraform-provider-comfyui/
├── main.go
├── internal/
│   ├── provider/
│   │   ├── provider.go          # Provider definition, Configure
│   │   └── provider_test.go
│   ├── services/
│   │   ├── workflow/
│   │   │   ├── resource_workflow.go
│   │   │   ├── resource_workflow_test.go
│   │   │   ├── data_source_workflow.go
│   │   │   └── data_source_workflow_test.go
│   │   ├── model/
│   │   │   ├── resource_model.go
│   │   │   ├── resource_model_test.go
│   │   │   ├── data_source_model.go
│   │   │   └── data_source_model_test.go
│   │   └── node/
│   │       ├── resource_custom_node.go
│   │       ├── resource_custom_node_test.go
│   │       └── data_source_node.go
│   └── common/
│       ├── client.go             # Shared API client
│       ├── errors.go             # Shared error handling
│       └── validators.go         # Shared validators
```

### 5.2 Registering Resources from Multiple Packages

When resources live in separate packages, the provider must import and register
each one:

```go
package provider

import (
    "context"

    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/sbuglione/terraform-provider-comfyui/internal/services/model"
    "github.com/sbuglione/terraform-provider-comfyui/internal/services/node"
    "github.com/sbuglione/terraform-provider-comfyui/internal/services/workflow"
)

func (p *ComfyUIProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        workflow.NewWorkflowResource,
        model.NewModelResource,
        node.NewCustomNodeResource,
    }
}
```

### 5.3 Shared Client Pattern

With a service-based structure, the API client is typically defined in a shared
package and passed through `Configure`:

```go
// internal/common/client.go
package common

type ComfyUIClient struct {
    Endpoint   string
    HTTPClient *http.Client
}

func NewClient(endpoint string) *ComfyUIClient {
    return &ComfyUIClient{
        Endpoint:   endpoint,
        HTTPClient: &http.Client{Timeout: 30 * time.Second},
    }
}
```

```go
// internal/provider/provider.go
func (p *ComfyUIProvider) Configure(ctx context.Context,
    req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var config ComfyUIProviderModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
    if resp.Diagnostics.HasError() {
        return
    }

    client := common.NewClient(config.Endpoint.ValueString())

    // Both resource and data source Configure methods receive this
    resp.DataSourceData = client
    resp.ResourceData = client
}
```

```go
// internal/services/workflow/resource_workflow.go
package workflow

import "github.com/sbuglione/terraform-provider-comfyui/internal/common"

type WorkflowResource struct {
    client *common.ComfyUIClient
}

func (r *WorkflowResource) Configure(ctx context.Context,
    req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    client, ok := req.ProviderData.(*common.ComfyUIClient)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Resource Configure Type",
            fmt.Sprintf("Expected *common.ComfyUIClient, got: %T", req.ProviderData),
        )
        return
    }
    r.client = client
}
```

### 5.4 When to Use Which Structure

| Criteria | Single Package (`internal/provider/`) | Service Packages (`internal/services/`) |
|---|---|---|
| Number of resources | 1–20 | 20+ |
| Team size | 1–3 developers | 3+ developers |
| API surface | Single service | Multiple distinct services |
| Shared logic | Minimal | Significant (common validators, client helpers) |
| Test isolation | Tests in one package | Tests scoped to service |
| Complexity | Low | Moderate |

For the ComfyUI provider, starting with a single `internal/provider/` package
is recommended. Refactor to service packages only when the resource count and
codebase complexity justify the overhead.

---

## 6. Development Workflow with `dev_overrides`

During development, use `dev_overrides` to skip `terraform init` and load your
locally-built binary directly:

### 6.1 Configure `~/.terraformrc`

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/sbuglione/comfyui" = "/home/sbuglione/go/bin"
  }

  direct {}
}
```

### 6.2 Build and Install

```bash
# Install the binary to $GOPATH/bin (or $GOBIN)
go install .

# Verify
ls -la $(go env GOPATH)/bin/terraform-provider-comfyui
```

### 6.3 Write a Test Configuration

```hcl
# test.tf
terraform {
  required_providers {
    comfyui = {
      source = "registry.terraform.io/sbuglione/comfyui"
    }
  }
}

provider "comfyui" {
  endpoint = "http://localhost:8188"
}

resource "comfyui_workflow" "example" {
  name = "test-workflow"
}
```

### 6.4 Run Terraform

```bash
# No 'terraform init' needed with dev_overrides!
terraform plan
terraform apply
```

> **Warning:** Terraform will display a warning when `dev_overrides` is active:
>
> ```
> │ Warning: Provider development overrides are in effect
> ```
>
> This is expected and confirms your local binary is being used.

---

## 7. CI/CD Configuration

### 7.1 Test Workflow (`.github/workflows/test.yml`)

```yaml
name: Tests

on:
  pull_request:
    branches: [main]
    paths-ignore:
      - "README.md"

permissions:
  contents: read

jobs:
  build:
    name: Build
    runs-on: ubuntu-latest
    timeout-minutes: 5
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: "go.mod"
          cache: true
      - run: go mod download
      - run: go build -v .
      - name: Run linters
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

  test:
    name: Acceptance Tests (Terraform ${{ matrix.terraform }})
    needs: build
    runs-on: ubuntu-latest
    timeout-minutes: 15
    strategy:
      fail-fast: false
      matrix:
        terraform:
          - "1.8.*"
          - "1.9.*"
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: "go.mod"
          cache: true
      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: ${{ matrix.terraform }}
          terraform_wrapper: false
      - run: go mod download
      - env:
          TF_ACC: "1"
        run: go test -v -cover ./internal/provider/
        timeout-minutes: 10
```

### 7.2 Release Workflow (`.github/workflows/release.yml`)

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: "go.mod"
          cache: true
      - name: Import GPG key
        uses: crazy-max/ghaction-import-gpg@v6
        id: import_gpg
        with:
          gpg_private_key: ${{ secrets.GPG_PRIVATE_KEY }}
          passphrase: ${{ secrets.PASSPHRASE }}
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## 8. Documentation Generation with `tfplugindocs`

The `terraform-plugin-docs` tool (`tfplugindocs`) generates Markdown documentation
from your provider schema and example files.

### 8.1 Install

```bash
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
```

### 8.2 Generate

Add a `go:generate` directive to your `main.go` or a separate file:

```go
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate -provider-name comfyui
```

Then run:

```bash
go generate ./...
```

This reads:
- Your provider schema (by running the binary)
- `examples/` directory for HCL snippets
- `templates/` directory for custom Markdown templates

And produces the `docs/` directory that the Terraform Registry renders.

### 8.3 Example Templates

```markdown
<!-- templates/index.md.tmpl -->
---
page_title: "ComfyUI Provider"
description: |-
  The ComfyUI provider manages resources in a ComfyUI instance.
---

# ComfyUI Provider

The ComfyUI provider allows you to manage workflows, models, and custom nodes
in a [ComfyUI](https://github.com/comfyanonymous/ComfyUI) instance.

## Example Usage

{{ tffile "examples/provider/provider.tf" }}

{{ .SchemaMarkdown | trimspace }}
```

---

## 9. References

| Resource | URL |
|---|---|
| Scaffolding Repository | <https://github.com/hashicorp/terraform-provider-scaffolding-framework> |
| Plugin Framework Tutorials | <https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework> |
| Code Organization | <https://developer.hashicorp.com/terraform/plugin/framework/getting-started/code-walkthrough> |
| terraform-plugin-docs | <https://github.com/hashicorp/terraform-plugin-docs> |
| GoReleaser for Terraform | <https://developer.hashicorp.com/terraform/registry/providers/publishing> |
| GitHub Actions for Terraform Providers | <https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-release-publish> |
| AWS Provider Structure (example of large provider) | <https://github.com/hashicorp/terraform-provider-aws> |
| Go `internal/` Convention | <https://go.dev/doc/modules/layout#internal> |
| terraform-plugin-mux (multi-protocol) | <https://pkg.go.dev/github.com/hashicorp/terraform-plugin-mux> |
| Registry Manifest Specification | <https://developer.hashicorp.com/terraform/registry/providers/publishing#terraform-registry-manifest> |
