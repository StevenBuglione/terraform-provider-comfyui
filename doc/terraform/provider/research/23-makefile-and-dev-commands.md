# 23 — Makefile and Development Commands

> **Purpose**: Define the standard development workflow and build commands for
> the ComfyUI Terraform provider, following HashiCorp conventions.

---

## 1. Why GNUmakefile (Not Makefile)

HashiCorp providers use `GNUmakefile` — not `Makefile` — for specific reasons:

- On **Linux**, `make` looks for `GNUmakefile` first, then `makefile`, then
  `Makefile`. Using `GNUmakefile` is unambiguous on case-sensitive systems.
- It signals the project requires **GNU Make** features (conditionals, `.PHONY`).
- This is the convention across all HashiCorp providers: `terraform-provider-aws`,
  `terraform-provider-azurerm`, `terraform-provider-google`, etc.

**Rule**: Name the file `GNUmakefile` in the repository root.

---

## 2. Complete GNUmakefile

```makefile
# GNUmakefile — terraform-provider-comfyui

BINARY_NAME := terraform-provider-comfyui
GOBIN        ?= $(shell go env GOPATH)/bin

default: testacc

# ─── Build ──────────────────────────────────────────────────────────────────

.PHONY: build
build:
	go build -o $(BINARY_NAME)

.PHONY: install
install: build
	mkdir -p $(GOBIN)
	mv $(BINARY_NAME) $(GOBIN)/$(BINARY_NAME)

# ─── Testing ────────────────────────────────────────────────────────────────

.PHONY: test
test:
	go test ./... -v $(TESTARGS)

.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

# ─── Code Quality ──────────────────────────────────────────────────────────

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: fmt
fmt:
	gofmt -s -w .

.PHONY: vet
vet:
	go vet ./...

# ─── Dependencies ──────────────────────────────────────────────────────────

.PHONY: deps
deps:
	go mod tidy

# ─── Documentation ─────────────────────────────────────────────────────────

.PHONY: generate
generate:
	go generate ./...

# ─── Helpers ───────────────────────────────────────────────────────────────

.PHONY: tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	go clean -testcache

.PHONY: ci
ci: fmt vet lint test build
```

---

## 3. Target-by-Target Explanation

### `make build`

Compiles the provider binary into the current directory. Terraform requires
binaries named `terraform-provider-<name>`. Use after every code change to
catch compile errors early.

### `make install`

Builds then copies the binary to `$GOPATH/bin`. Configure Terraform to use it
with a `dev_overrides` block in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/sbuglione/comfyui" = "/home/sbuglione/go/bin"
  }
  direct {}
}
```

### `make test`

Runs **unit tests only** — no infrastructure required. Tests schema validation,
type helpers, ID parsing, and pure functions. Completes in seconds. The absence
of `TF_ACC=1` causes acceptance tests to call `t.Skip()`.

### `make testacc`

Runs **acceptance tests** that create and destroy real infrastructure. The
`TF_ACC=1` variable enables them. Timeout is 120 minutes for slow API ops.

⚠️ Acceptance tests create real resources that may incur costs.

### `make lint`

Runs `golangci-lint`, aggregating `govet`, `staticcheck`, `errcheck`,
`gosimple`, and `ineffassign`. Configure via `.golangci.yml`.

### `make fmt`

Formats all Go files with `gofmt -s -w .`. The `-s` flag simplifies code
(e.g., `[]T{T{}}` → `[]T{{}}`). Non-negotiable in Go.

### `make vet`

Runs Go's built-in static analyzer: printf mismatches, unreachable code,
suspicious assignments, incorrect struct tags.

### `make generate`

Runs all `//go:generate` directives. For providers, this typically runs
`tfplugindocs` to generate registry documentation from schemas.

### `make deps`

Runs `go mod tidy` — adds missing deps, removes unused ones. Run before
committing to keep `go.sum` clean.

### `make tools`

Installs development tools. Run once when setting up a new environment.

### `make clean`

Removes the binary and clears the Go test cache. Useful when cached test
results mask real failures.

### `make ci`

Runs the full local CI pipeline: `fmt → vet → lint → test → build`.

---

## 4. Environment Variables

### `TF_ACC`

```bash
TF_ACC=1     # Enable acceptance tests
TF_ACC=       # (unset) Skip acceptance tests — default
```

### `TESTARGS`

Interpolated directly into the `go test` command:

```bash
make testacc TESTARGS="-run TestAccWorkflow_basic"      # Single test
make test    TESTARGS="-run TestUnit -count=1"           # No cache
make test    TESTARGS="-race"                            # Race detector
make testacc TESTARGS="-parallel=2 -run TestAcc"         # Limit parallelism
```

### `TF_LOG`

Controls Terraform's logging. Extremely useful for debugging CRUD operations:

```bash
TF_LOG=TRACE   # Maximum verbosity — all SDK calls
TF_LOG=DEBUG   # Debug-level messages
TF_LOG=INFO    # Informational only
TF_LOG=WARN    # Warnings only
TF_LOG=ERROR   # Errors only

# Log to file
TF_LOG_PATH=./terraform.log TF_LOG=TRACE terraform plan

# Debug a failing acceptance test
TF_LOG=DEBUG make testacc TESTARGS="-run TestAccWorkflow_basic -count=1"
```

### Provider-Specific Variables

```bash
COMFYUI_HOST=http://localhost:8188
COMFYUI_API_KEY=your-api-key
```

---

## 5. Development Workflow

### Typical Session Order

```
1. make deps        — clean up go.mod/go.sum
2. make fmt         — format code
3. make vet         — static analysis
4. make build       — compile the binary
5. make test        — run unit tests
6. make lint        — check code quality
7. make install     — install to GOPATH/bin
8. make testacc     — run acceptance tests
9. make generate    — regenerate docs
```

### Quick Iteration Cycle

```bash
make build && make test TESTARGS="-run TestUnit_MyResource"
```

### Full Validation Before PR

```bash
make ci && make testacc TESTARGS="-run TestAcc"
```

### Debugging a Failing Test

```bash
TF_LOG=DEBUG make testacc TESTARGS="-run TestAccWorkflow_basic -count=1"
```

The `-count=1` flag disables test caching so the test always runs fresh.

---

## 6. Local Provider Testing

After `make install`, use this Terraform config:

```hcl
terraform {
  required_providers {
    comfyui = {
      source = "registry.terraform.io/sbuglione/comfyui"
    }
  }
}

provider "comfyui" {
  host = "http://localhost:8188"
}

resource "comfyui_workflow" "example" {
  name = "test-workflow"
}
```

Run `terraform plan` — it uses the local binary via `dev_overrides`.

> **Important**: Remove `dev_overrides` before running acceptance tests.
> Acceptance tests use their own provider factories.

---

## 7. CI/CD Integration

Minimal GitHub Actions workflow using Makefile targets:

```yaml
name: CI
on: [push, pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: make ci
  acceptance:
    runs-on: ubuntu-latest
    needs: build
    if: github.event_name == 'push'
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: make testacc
        env:
          TF_ACC: "1"
          COMFYUI_HOST: ${{ secrets.COMFYUI_HOST }}
```

---

## 8. References

- **HashiCorp Makefile Convention**: <https://hashicorp.github.io/terraform-provider-aws/makefile-cheat-sheet/>
- **Go Test Flags**: <https://pkg.go.dev/cmd/go#hdr-Testing_flags>
- **golangci-lint**: <https://golangci-lint.run/>
- **tfplugindocs**: <https://github.com/hashicorp/terraform-plugin-docs>
- **Terraform Logging**: <https://developer.hashicorp.com/terraform/plugin/log/managing>

---

*Previous*: [22-reference-provider-azurerm.md](22-reference-provider-azurerm.md)
