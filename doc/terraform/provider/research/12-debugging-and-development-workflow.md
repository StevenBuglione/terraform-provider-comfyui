# Debugging and Development Workflow (Plugin Framework)

## Overview

This document covers the full local development cycle for a Terraform provider built with the Plugin Framework: building, installing, configuring dev overrides, logging, debugging with Delve, and VS Code integration.

---

## Local Development Setup

### Step 1: Build and Install

The simplest workflow is to use `go install`, which compiles the binary and places it in your `$GOPATH/bin` (typically `~/go/bin`):

```bash
go install .
```

This produces a binary named after the module — e.g., `terraform-provider-comfyui` — in `~/go/bin/`.

Verify:

```bash
ls -la ~/go/bin/terraform-provider-comfyui
```

### Step 2: Configure `dev_overrides` in `~/.terraformrc`

Terraform normally downloads providers from the registry. During development, you want it to use your locally-built binary instead. The `dev_overrides` block in `~/.terraformrc` does this:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/sbuglione/comfyui" = "/home/sbuglione/go/bin"
  }
  direct {}
}
```

Key details:

- The key is the **full provider address** as it would appear in a `required_providers` block.
- The value is the **directory** containing the binary (not the binary path itself).
- The `direct {}` block is required — it tells Terraform to fall back to direct downloads for all other providers.

### Step 3: Skip `terraform init`, Go Straight to Plan/Apply

With `dev_overrides` configured, you **do not need** to run `terraform init` for your provider. Terraform finds it directly from the override path.

```bash
cd examples/basic
terraform plan
terraform apply
```

You will see a warning:

```
╷
│ Warning: Provider development overrides are in effect
│
│ The following provider development overrides are set in the CLI
│ configuration:
│  - sbuglione/comfyui in /home/sbuglione/go/bin
│
│ The behavior may therefore not match any released version of the
│ provider and applying changes may cause the state to become
│ incompatible with published releases.
╵
```

This warning is normal and expected during development.

### The Full Rebuild Loop

```bash
# 1. Edit provider code
# 2. Rebuild
go install .
# 3. Test with Terraform
cd examples/basic
terraform plan
terraform apply
# 4. Repeat
```

---

## Logging

### TF_LOG Environment Variable

Terraform supports granular log levels via the `TF_LOG` environment variable:

| Level | Description |
|---|---|
| `TRACE` | Most verbose — every internal operation, including RPC calls. |
| `DEBUG` | Detailed diagnostic messages. |
| `INFO` | General operational messages. |
| `WARN` | Warnings about potential issues. |
| `ERROR` | Only errors. |

```bash
TF_LOG=TRACE terraform plan
```

### TF_LOG_PROVIDER: Provider-Only Logs

To see **only** your provider's logs (not Terraform core or other providers):

```bash
TF_LOG_PROVIDER=TRACE terraform plan
```

This is extremely useful because `TF_LOG=TRACE` produces enormous output from Terraform core. `TF_LOG_PROVIDER` filters to just your provider's log lines.

### TF_LOG_PATH: Log to a File

When output is too verbose for the terminal, redirect logs to a file:

```bash
TF_LOG=TRACE TF_LOG_PATH=./terraform.log terraform plan
```

Or combine with provider-specific logging:

```bash
TF_LOG_PROVIDER=TRACE TF_LOG_PATH=./provider.log terraform plan
```

### Combining Variables

```bash
# Only provider logs, debug level, to a file
TF_LOG_PROVIDER=DEBUG TF_LOG_PATH=./debug.log terraform apply

# All logs at trace level to a file, for deep investigation
TF_LOG=TRACE TF_LOG_PATH=./full-trace.log terraform plan
```

---

## Structured Logging with `tflog`

The Plugin Framework provides the `tflog` package for structured, leveled logging that integrates with Terraform's log system. **Always use `tflog` instead of `fmt.Println` or `log.Printf`.**

### Import

```go
import "github.com/hashicorp/terraform-plugin-log/tflog"
```

### Usage in Resource Methods

```go
func (r *workflowResource) Create(
    ctx context.Context,
    req resource.CreateRequest,
    resp *resource.CreateResponse,
) {
    var plan workflowResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    tflog.Debug(ctx, "creating workflow", map[string]interface{}{
        "name": plan.Name.ValueString(),
    })

    // ... API call ...

    tflog.Info(ctx, "workflow created", map[string]interface{}{
        "id":   result.ID,
        "name": result.Name,
    })
}
```

### Log Levels

```go
tflog.Trace(ctx, "entering function", map[string]interface{}{...})
tflog.Debug(ctx, "intermediate state", map[string]interface{}{...})
tflog.Info(ctx, "operation completed", map[string]interface{}{...})
tflog.Warn(ctx, "potential issue", map[string]interface{}{...})
tflog.Error(ctx, "operation failed", map[string]interface{}{...})
```

### Subsystems

You can create sub-loggers for different components:

```go
ctx = tflog.NewSubsystem(ctx, "comfyui_client")
tflog.SubsystemDebug(ctx, "comfyui_client", "making API request", map[string]interface{}{
    "endpoint": "/api/workflow",
    "method":   "POST",
})
```

### Important Rules

1. Always pass the `ctx` from the CRUD method — it carries the logging configuration.
2. Never log sensitive values (API keys, passwords).
3. Use structured fields (the map argument) instead of string interpolation.
4. `tflog` output only appears when `TF_LOG` or `TF_LOG_PROVIDER` is set.

---

## Debugging with Delve

For interactive debugging (breakpoints, stepping, variable inspection), use Delve — the standard Go debugger.

### Install Delve

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

### Step 1: Build with Debug Symbols

By default, Go optimizes binaries and strips some debug info. Build with flags that disable optimization:

```bash
go build -gcflags="all=-N -l" -o terraform-provider-comfyui
```

- `-N` disables optimizations.
- `-l` disables inlining.
- `all=` applies these flags to all packages, including dependencies.

### Step 2: Run the Provider with `-debug`

The Plugin Framework supports a `-debug` flag that starts the provider in debug mode. It waits for a debugger (or a Terraform process) to connect:

```bash
./terraform-provider-comfyui -debug
```

The provider will print a `TF_REATTACH_PROVIDERS` JSON blob to stderr:

```
Provider started, to attach Terraform set the TF_REATTACH_PROVIDERS env var:

    TF_REATTACH_PROVIDERS='{"registry.terraform.io/sbuglione/comfyui":{"Protocol":"grpc","ProtocolVersion":6,"Pid":12345,"Test":true,"Addr":{"Network":"unix","String":"/tmp/plugin12345"}}}'
```

### Step 3: Copy the Environment Variable

In a **separate terminal**, export the `TF_REATTACH_PROVIDERS` value:

```bash
export TF_REATTACH_PROVIDERS='{"registry.terraform.io/sbuglione/comfyui":{"Protocol":"grpc","ProtocolVersion":6,"Pid":12345,"Test":true,"Addr":{"Network":"unix","String":"/tmp/plugin12345"}}}'
```

### Step 4: Run Terraform

Now run Terraform commands in that second terminal. Terraform will connect to your already-running provider process:

```bash
cd examples/basic
terraform plan
terraform apply
```

### Using Delve Directly

Instead of running the binary yourself, let Delve launch and manage it:

```bash
dlv exec ./terraform-provider-comfyui -- -debug
```

Then in the Delve console:

```
(dlv) break internal/provider/workflow_resource.go:42
(dlv) continue
```

Switch to the second terminal, set `TF_REATTACH_PROVIDERS`, and run `terraform apply`. Delve will hit your breakpoint.

### Useful Delve Commands

| Command | Description |
|---|---|
| `break <file>:<line>` | Set a breakpoint |
| `continue` (or `c`) | Resume execution |
| `next` (or `n`) | Step over |
| `step` (or `s`) | Step into |
| `print <var>` (or `p`) | Print a variable |
| `locals` | Show all local variables |
| `goroutines` | List all goroutines |
| `stack` | Show call stack |
| `quit` | Exit debugger |

---

## VS Code Launch Configuration

You can debug your provider directly in VS Code with the Go extension.

### `.vscode/launch.json`

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Terraform Provider",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}",
            "args": ["-debug"],
            "env": {},
            "buildFlags": "-gcflags='all=-N -l'"
        },
        {
            "name": "Debug Acceptance Tests",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${workspaceFolder}/internal/provider",
            "env": {
                "TF_ACC": "1",
                "COMFYUI_API_ENDPOINT": "http://localhost:8188"
            },
            "args": [
                "-test.v",
                "-test.run",
                "TestAccWorkflowResource_basic",
                "-test.timeout",
                "120m"
            ]
        }
    ]
}
```

### Workflow with VS Code

1. Set breakpoints by clicking in the gutter next to line numbers.
2. Select "Debug Terraform Provider" from the Run and Debug panel.
3. Press F5 to start debugging.
4. Copy the `TF_REATTACH_PROVIDERS` value from the Debug Console output.
5. In VS Code's integrated terminal, export the variable and run `terraform plan` or `terraform apply`.
6. VS Code will stop at your breakpoints.

---

## The Development Workflow Loop

Here is the recommended iterative development workflow:

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│   1. EDIT                                                       │
│      Modify provider code in internal/provider/                 │
│                              │                                  │
│                              ▼                                  │
│   2. BUILD                                                      │
│      go install .                                               │
│                              │                                  │
│                              ▼                                  │
│   3. TEST (quick)                                               │
│      cd examples/basic && terraform plan                        │
│                              │                                  │
│                              ▼                                  │
│   4. TEST (thorough)                                            │
│      TF_ACC=1 go test ./internal/provider -v -run TestAcc...   │
│                              │                                  │
│                              ▼                                  │
│   5. DEBUG (if needed)                                          │
│      Use TF_LOG_PROVIDER=DEBUG or Delve                         │
│                              │                                  │
│                              ▼                                  │
│   6. ITERATE                                                    │
│      Go back to step 1                                          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Fast Feedback: The Quick Check

For the tightest feedback loop during active development:

```bash
# One-liner: build and run a quick terraform plan
go install . && cd examples/basic && terraform plan && cd -
```

### Full Verification

Before committing:

```bash
# Run unit tests
go test ./internal/provider/... -v -race

# Run acceptance tests
TF_ACC=1 go test ./internal/provider/... -v -timeout 120m

# Run linter (if configured)
golangci-lint run ./...
```

---

## Common Issues and Troubleshooting

### "Provider not found" or "registry.terraform.io/.../comfyui: not available"

**Cause:** `dev_overrides` is not configured, or the path is wrong.

**Fix:** Verify `~/.terraformrc` exists and the path points to the directory containing the provider binary:

```bash
cat ~/.terraformrc
ls ~/go/bin/terraform-provider-comfyui
```

### "Inconsistent result after apply"

**Cause:** The state returned by `Create` or `Update` doesn't match what Terraform planned. Common reasons:
- A field is computed by the API but not set in the response model.
- A field is normalized by the API (e.g., lowercased) but stored differently in state.

**Fix:** Ensure your `Create`/`Update` methods read back the resource from the API and set **all** attributes in state, including computed ones.

### "Provider produced inconsistent final plan"

**Cause:** The plan modifier or default value logic is producing different values on second plan.

**Fix:** Use `TF_LOG_PROVIDER=TRACE` to see exact planned vs actual values. Check plan modifiers and `UseStateForUnknown`.

### State drift / phantom changes on every plan

**Cause:** A read operation returns data in a different format than what was stored.

**Fix:** Normalize values before storing in state. Common culprits:
- JSON field ordering differences
- Trailing whitespace
- Numeric precision

### "rpc error: code = Unavailable"

**Cause:** The provider process crashed or isn't running.

**Fix:** Check for panics in provider logs. Ensure proper nil checks in all CRUD methods. Use `TF_LOG=TRACE` to see the full error chain.

### Tests pass locally but fail in CI

**Cause:** Environment differences — missing credentials, different Go version, race conditions in parallel tests.

**Fix:**
- Ensure CI has all required environment variables.
- Pin the Go version in CI.
- Use unique resource names with `acctest.RandomWithPrefix`.
- Add proper test cleanup with `CheckDestroy`.

### Binary not updating after `go install`

**Cause:** Build cache or wrong `GOBIN`.

**Fix:**

```bash
# Force a clean build
go clean -cache
go install .

# Verify the binary timestamp
ls -la ~/go/bin/terraform-provider-comfyui
```

---

## Environment Variable Reference

| Variable | Purpose | Example |
|---|---|---|
| `TF_ACC` | Enable acceptance tests | `TF_ACC=1` |
| `TF_LOG` | Set Terraform log level (all) | `TF_LOG=DEBUG` |
| `TF_LOG_PROVIDER` | Set log level for providers only | `TF_LOG_PROVIDER=TRACE` |
| `TF_LOG_PATH` | Write logs to a file | `TF_LOG_PATH=./tf.log` |
| `TF_REATTACH_PROVIDERS` | Connect Terraform to debug provider | *(JSON blob from -debug output)* |
| `TF_CLI_CONFIG_FILE` | Override `.terraformrc` location | `TF_CLI_CONFIG_FILE=./dev.tfrc` |

---

## Example: Complete `.terraformrc` for Development

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/sbuglione/comfyui" = "/home/sbuglione/go/bin"
  }
  direct {}
}
```

Store this at `~/.terraformrc`. To use a project-local config instead:

```bash
export TF_CLI_CONFIG_FILE="$(pwd)/dev.tfrc"
terraform plan
```

---

## Example: Complete Makefile for Development

```makefile
BINARY_NAME=terraform-provider-comfyui
GOBIN=$(shell go env GOPATH)/bin

.PHONY: build install test testacc lint clean debug

build:
	go build -o $(BINARY_NAME) .

install:
	go install .

test:
	go test ./internal/provider/... -v -race

testacc:
	TF_ACC=1 go test ./internal/provider/... -v -timeout 120m

lint:
	golangci-lint run ./...

clean:
	go clean -cache
	rm -f $(BINARY_NAME)

debug:
	go build -gcflags="all=-N -l" -o $(BINARY_NAME) .
	./$(BINARY_NAME) -debug
```

Usage:

```bash
make install     # Build and install to GOPATH/bin
make test        # Run unit tests
make testacc     # Run acceptance tests
make debug       # Start provider in debug mode
```

---

## References

- Debugging Providers: <https://developer.hashicorp.com/terraform/plugin/debugging>
- Plugin Framework Logging: <https://developer.hashicorp.com/terraform/plugin/log/writing>
- `tflog` Package: <https://pkg.go.dev/github.com/hashicorp/terraform-plugin-log/tflog>
- Delve Debugger: <https://github.com/go-delve/delve>
- Terraform CLI Configuration: <https://developer.hashicorp.com/terraform/cli/config/config-file>
- Go Build Flags: <https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies>
