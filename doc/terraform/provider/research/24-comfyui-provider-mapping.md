# 24 — ComfyUI Provider Mapping: API to Terraform Resources, Data Sources, and Functions

## Purpose

This document defines the complete mapping from ComfyUI's REST/WebSocket API surface
to Terraform provider primitives — resources, data sources, and provider-defined
functions. Every API endpoint is accounted for. Each Terraform concept includes its
schema, CRUD lifecycle, and concrete HCL + Go code examples so an implementation
agent can produce the provider without ambiguity.

---

## 1. ComfyUI API Overview

ComfyUI is a node-based generative-AI workflow tool. Its backend exposes a set of
REST endpoints and a WebSocket channel for real-time status. The table below is the
authoritative list referenced throughout this document.

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/prompt` | Submit (queue) a workflow for execution |
| GET | `/prompt` | Get current queue information |
| GET | `/history/{prompt_id}` | Retrieve execution history and outputs |
| GET | `/view` | Retrieve a generated output file (image, etc.) |
| POST | `/upload/image` | Upload an image to use as workflow input |
| POST | `/upload/mask` | Upload a mask image |
| GET | `/queue` | Queue state — running + pending items |
| POST | `/interrupt` | Interrupt the currently running workflow |
| GET | `/system_stats` | System health: OS, GPU, VRAM, ComfyUI version |
| GET | `/object_info` | Available node definitions and their specs |
| WS | `/ws` | Real-time execution progress events |

### 1.1 Key API Behaviours

* **POST /prompt** returns `{ "prompt_id": "<uuid>" }`. The workflow JSON is sent
  inside a `prompt` key. An optional `client_id` associates WebSocket messages.
* **GET /history/{prompt_id}** returns a map keyed by `prompt_id`. Each entry
  contains `status` (`{ status_str, completed, messages }`) and `outputs` (keyed
  by node id).
* **GET /view** accepts query parameters `filename`, `subfolder`, and `type`
  (`output` | `input` | `temp`). It returns raw binary data (e.g. a PNG).
* **POST /upload/image** is `multipart/form-data` with fields `image` (file),
  `subfolder` (optional), `overwrite` (bool), and `type` (`input` | `temp`).
* **GET /object_info** can be called bare (returns all nodes) or as
  `/object_info/{node_name}` (single node).
* **WebSocket /ws?clientId={id}** pushes JSON frames:
  `execution_start`, `execution_cached`, `executing`, `progress`,
  `executed`, `execution_error`, `execution_interrupted`.

---

## 2. Provider Configuration

### 2.1 HCL Schema

```hcl
terraform {
  required_providers {
    comfyui = {
      source  = "registry.terraform.io/sbuglione/comfyui"
      version = "~> 0.1"
    }
  }
}

provider "comfyui" {
  # Base URL of the ComfyUI server (required).
  endpoint = "http://localhost:8188"

  # Optional API key — used when ComfyUI is behind an auth proxy.
  api_key = var.comfyui_api_key

  # HTTP request timeout in seconds. Workflow executions can be long.
  timeout = 300

  # Optional client ID for WebSocket correlation.
  # If omitted the provider generates a UUID at configure time.
  client_id = "terraform-run"
}
```

### 2.2 Go Provider Schema (Framework)

```go
// internal/provider/provider.go

type ComfyUIProviderModel struct {
    Endpoint types.String `tfsdk:"endpoint"`
    APIKey   types.String `tfsdk:"api_key"`
    Timeout  types.Int64  `tfsdk:"timeout"`
    ClientID types.String `tfsdk:"client_id"`
}

func (p *ComfyUIProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Provider for managing ComfyUI workflows and executions.",
        Attributes: map[string]schema.Attribute{
            "endpoint": schema.StringAttribute{
                Description: "Base URL of the ComfyUI server.",
                Required:    true,
            },
            "api_key": schema.StringAttribute{
                Description: "API key for authenticated setups.",
                Optional:    true,
                Sensitive:   true,
            },
            "timeout": schema.Int64Attribute{
                Description: "HTTP request timeout in seconds.",
                Optional:    true,
            },
            "client_id": schema.StringAttribute{
                Description: "Client ID for WebSocket correlation.",
                Optional:    true,
            },
        },
    }
}
```

### 2.3 HTTP Client Initialisation

```go
// internal/provider/provider.go — Configure method

func (p *ComfyUIProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var config ComfyUIProviderModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
    if resp.Diagnostics.HasError() {
        return
    }

    timeout := 300 * time.Second
    if !config.Timeout.IsNull() {
        timeout = time.Duration(config.Timeout.ValueInt64()) * time.Second
    }

    clientID := uuid.NewString()
    if !config.ClientID.IsNull() {
        clientID = config.ClientID.ValueString()
    }

    client := &comfyui.Client{
        BaseURL:    config.Endpoint.ValueString(),
        APIKey:     config.APIKey.ValueString(),
        HTTPClient: &http.Client{Timeout: timeout},
        ClientID:   clientID,
    }

    resp.DataSourceData = client
    resp.ResourceData   = client
}
```

---

## 3. Proposed Resources

### 3.1 `comfyui_workflow` — Workflow Template

Represents a workflow definition stored and version-managed by the provider. This
resource does **not** execute the workflow; it only manages the template.

#### Schema

| Attribute | Type | Mode | Description |
|-----------|------|------|-------------|
| `id` | String | Computed | Provider-assigned UUID |
| `name` | String | Required | Human-readable name |
| `workflow_json` | String | Required | Full ComfyUI workflow graph as JSON |
| `description` | String | Optional | Free-text description |
| `tags` | Set of String | Optional | Categorisation tags |
| `last_updated` | String | Computed | RFC 3339 timestamp |

#### CRUD Lifecycle

| Op | Implementation |
|----|----------------|
| Create | Generate UUID, persist workflow JSON to provider state |
| Read | Return stored state (purely local — no API call) |
| Update | Replace JSON / metadata, bump `last_updated` |
| Delete | Remove from state |

> **Note:** If a future ComfyUI release adds server-side workflow storage, this
> resource would switch to using that API.

#### HCL Example

```hcl
resource "comfyui_workflow" "txt2img" {
  name        = "txt2img-sdxl"
  description = "Text-to-image pipeline using SDXL"
  tags        = ["sdxl", "txt2img"]

  workflow_json = file("${path.module}/workflows/txt2img_sdxl.json")
}
```

---

### 3.2 `comfyui_workflow_execution` — Run a Workflow

Represents a single, one-shot execution of a workflow. Creating this resource
queues the prompt; the provider waits for completion before marking the resource
as created.

#### Schema

| Attribute | Type | Mode | Description |
|-----------|------|------|-------------|
| `id` | String | Computed | Same as `prompt_id` |
| `workflow_json` | String | Required | Workflow graph JSON |
| `prompt_id` | String | Computed | UUID returned by POST /prompt |
| `status` | String | Computed | `running`, `completed`, `error`, `interrupted` |
| `outputs` | Map of String | Computed | Node-ID → serialised output JSON |
| `execution_time_ms` | Int64 | Computed | Wall-clock duration |
| `wait_for_completion` | Bool | Optional | Default `true`; if `false`, Create returns immediately |

#### CRUD Lifecycle

| Op | Implementation |
|----|----------------|
| Create | POST `/prompt` with `workflow_json`. If `wait_for_completion`, open WebSocket `/ws?clientId={id}` and block until `executed` or `execution_error`. Then GET `/history/{prompt_id}` to populate `outputs`. |
| Read | GET `/history/{prompt_id}`. Update `status` and `outputs`. |
| Update | Not supported. ForceNew on `workflow_json`. |
| Delete | No-op or POST `/interrupt` if still running. Remove from state. |

#### Go Skeleton

```go
// internal/resources/workflow_execution.go

func (r *WorkflowExecutionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan WorkflowExecutionModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }

    client := r.client

    // 1. Queue the prompt
    promptResp, err := client.QueuePrompt(ctx, plan.WorkflowJSON.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Failed to queue prompt", err.Error())
        return
    }

    plan.PromptID = types.StringValue(promptResp.PromptID)
    plan.Status   = types.StringValue("running")

    // 2. Optionally wait via WebSocket
    if plan.WaitForCompletion.ValueBool() {
        result, err := client.WaitForExecution(ctx, promptResp.PromptID)
        if err != nil {
            resp.Diagnostics.AddError("Execution failed", err.Error())
            return
        }
        plan.Status          = types.StringValue(result.Status)
        plan.Outputs         = flattenOutputs(result.Outputs)
        plan.ExecutionTimeMs = types.Int64Value(result.DurationMs)
    }

    plan.ID = plan.PromptID
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

#### HCL Example

```hcl
resource "comfyui_workflow_execution" "run" {
  workflow_json = comfyui_workflow.txt2img.workflow_json

  wait_for_completion = true
}

output "prompt_id" {
  value = comfyui_workflow_execution.run.prompt_id
}

output "generated_images" {
  value = comfyui_workflow_execution.run.outputs
}
```

#### Ephemeral Resource Consideration

Terraform 1.10 introduced **ephemeral resources** — resources that exist only for
the duration of a plan/apply and are never persisted in state. A workflow execution
is a strong candidate for ephemeral treatment because:

* Executions are inherently one-shot; re-reading stale history has limited value.
* Outputs (images) are consumed downstream, not managed long-term.

If implementing as ephemeral, use `ephemeral "comfyui_workflow_execution"` with the
same schema minus `id`. The `Open` method replaces `Create`; there is no `Read`.

---

### 3.3 `comfyui_uploaded_image` — Manage Uploaded Images

#### Schema

| Attribute | Type | Mode | Description |
|-----------|------|------|-------------|
| `id` | String | Computed | `{type}/{subfolder}/{filename}` |
| `file_path` | String | Required | Local path to the image file |
| `file_hash` | String | Computed | SHA-256 of the uploaded content |
| `filename` | String | Computed | Server-assigned filename |
| `subfolder` | String | Optional | Target subfolder on server |
| `overwrite` | Bool | Optional | Default `true` |
| `type` | String | Optional | `input` (default) or `temp` |

#### CRUD Lifecycle

| Op | Implementation |
|----|----------------|
| Create | POST `/upload/image` (multipart). Parse response for `name`, `subfolder`, `type`. |
| Read | GET `/view?filename={}&subfolder={}&type={}`. Compare hash. |
| Update | Re-upload if `file_hash` changed (ForceNew on `file_path`). |
| Delete | No direct API. Best-effort: mark removed from state. |

#### HCL Example

```hcl
resource "comfyui_uploaded_image" "init_image" {
  file_path = "${path.module}/images/sketch.png"
  subfolder = "terraform"
  overwrite = true
}
```

---

### 3.4 `comfyui_uploaded_mask` — Manage Uploaded Masks

Identical schema and lifecycle to `comfyui_uploaded_image` but hits
POST `/upload/mask`. Implement as a thin wrapper.

```hcl
resource "comfyui_uploaded_mask" "inpaint_mask" {
  file_path = "${path.module}/masks/face_region.png"
  subfolder = "terraform"
}
```

---

## 4. Proposed Data Sources

### 4.1 `comfyui_system_stats`

Reads GET `/system_stats`. Useful for pre-flight checks and conditional logic.

#### Schema

| Attribute | Type | Description |
|-----------|------|-------------|
| `os` | String | Operating system of the ComfyUI host |
| `gpu_name` | String | Primary GPU device name |
| `gpu_vram_total_mb` | Int64 | Total VRAM in megabytes |
| `gpu_vram_free_mb` | Int64 | Free VRAM in megabytes |
| `comfyui_version` | String | Server version string |
| `python_version` | String | Python runtime version |
| `torch_version` | String | PyTorch version |
| `devices` | List of Object | All GPU devices with individual stats |

#### API Response Shape (reference)

```json
{
  "system": {
    "os": "posix",
    "python_version": "3.11.7",
    "embedded_python": false
  },
  "devices": [
    {
      "name": "NVIDIA GeForce RTX 4090",
      "type": "cuda",
      "index": 0,
      "vram_total": 25769803776,
      "vram_free": 22548578304,
      "torch_vram_total": 25769803776,
      "torch_vram_free": 22548578304
    }
  ]
}
```

#### HCL Example

```hcl
data "comfyui_system_stats" "server" {}

output "gpu" {
  value = data.comfyui_system_stats.server.gpu_name
}

# Gate heavy workflows on available VRAM
locals {
  use_sdxl = data.comfyui_system_stats.server.gpu_vram_free_mb >= 10240
}
```

#### Go Implementation Sketch

```go
func (d *SystemStatsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    client := d.client

    stats, err := client.GetSystemStats(ctx)
    if err != nil {
        resp.Diagnostics.AddError("Failed to read system stats", err.Error())
        return
    }

    var state SystemStatsModel
    state.ID              = types.StringValue("system_stats")
    state.OS              = types.StringValue(stats.System.OS)
    state.PythonVersion   = types.StringValue(stats.System.PythonVersion)
    state.ComfyUIVersion  = types.StringValue(stats.System.ComfyUIVersion)

    if len(stats.Devices) > 0 {
        primary := stats.Devices[0]
        state.GPUName        = types.StringValue(primary.Name)
        state.GPUVRAMTotalMB = types.Int64Value(primary.VRAMTotal / (1024 * 1024))
        state.GPUVRAMFreeMB  = types.Int64Value(primary.VRAMFree / (1024 * 1024))
    }

    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

---

### 4.2 `comfyui_queue`

Reads GET `/queue`. Returns counts and details of running/pending prompts.

#### Schema

| Attribute | Type | Description |
|-----------|------|-------------|
| `running_count` | Int64 | Number of currently executing prompts |
| `pending_count` | Int64 | Number of queued prompts waiting |
| `running_prompts` | List of Object | Details of running entries |
| `pending_prompts` | List of Object | Details of pending entries |

#### HCL Example

```hcl
data "comfyui_queue" "current" {}

# Use a precondition to block if the queue is overloaded
resource "comfyui_workflow_execution" "run" {
  workflow_json = file("workflow.json")

  lifecycle {
    precondition {
      condition     = data.comfyui_queue.current.pending_count < 10
      error_message = "Queue has too many pending items; aborting."
    }
  }
}
```

---

### 4.3 `comfyui_node_info`

Reads GET `/object_info` or GET `/object_info/{node_name}`.

#### Schema

| Attribute | Type | Description |
|-----------|------|-------------|
| `node_name` | String (Optional) | Filter to a specific node class |
| `nodes` | List of Object | Available node definitions |

Each node object:

| Field | Type | Description |
|-------|------|-------------|
| `name` | String | Node class name (e.g. `KSampler`) |
| `display_name` | String | Human-friendly label |
| `category` | String | Menu category path |
| `input_required` | Map | Required input specs |
| `input_optional` | Map | Optional input specs |
| `output_types` | List of String | Output slot types |
| `output_names` | List of String | Output slot names |

#### HCL Example

```hcl
data "comfyui_node_info" "sampler" {
  node_name = "KSampler"
}

output "sampler_inputs" {
  value = data.comfyui_node_info.sampler.nodes[0].input_required
}

# List all available nodes (no filter)
data "comfyui_node_info" "all" {}

output "node_count" {
  value = length(data.comfyui_node_info.all.nodes)
}
```

---

### 4.4 `comfyui_workflow_history`

Reads GET `/history/{prompt_id}`.

#### Schema

| Attribute | Type | Description |
|-----------|------|-------------|
| `prompt_id` | String (Required) | UUID of the execution to look up |
| `status` | String | `completed`, `error`, `interrupted` |
| `outputs` | Map of String | Node-ID → serialised output JSON |
| `execution_time_ms` | Int64 | Duration of the execution |
| `messages` | List of Object | Status messages from the run |

#### HCL Example

```hcl
data "comfyui_workflow_history" "last_run" {
  prompt_id = comfyui_workflow_execution.run.prompt_id
}

output "run_status" {
  value = data.comfyui_workflow_history.last_run.status
}
```

---

### 4.5 `comfyui_output`

Reads GET `/view` to fetch a specific generated file.

#### Schema

| Attribute | Type | Description |
|-----------|------|-------------|
| `filename` | String (Required) | Name of the output file |
| `subfolder` | String | Subfolder on the server |
| `type` | String | `output` (default), `input`, or `temp` |
| `content_base64` | String (Computed) | Base64-encoded file content |
| `content_length` | Int64 (Computed) | Size in bytes |

#### HCL Example

```hcl
data "comfyui_output" "generated_image" {
  filename  = "ComfyUI_00001_.png"
  subfolder = ""
  type      = "output"
}

resource "local_file" "result" {
  content_base64 = data.comfyui_output.generated_image.content_base64
  filename       = "${path.module}/output/result.png"
}
```

---

## 5. Provider-Defined Functions (Terraform ≥ 1.8)

### 5.1 `parse_workflow_json`

Validates and normalises a ComfyUI workflow JSON string. Returns a structured
object or raises a function error on invalid input.

#### Signature

```
provider::comfyui::parse_workflow_json(json_string) → object
```

#### Go Implementation

```go
// internal/functions/parse_workflow_json.go

func (f ParseWorkflowJSONFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
    resp.Definition = function.Definition{
        Summary:     "Parse and validate a ComfyUI workflow JSON string.",
        Description: "Accepts a JSON string representing a ComfyUI workflow graph. Returns a structured object with node_count, nodes list, and validation errors.",
        Parameters: []function.Parameter{
            function.StringParameter{
                Name:        "json_string",
                Description: "The raw JSON string of a ComfyUI workflow.",
            },
        },
        Return: function.ObjectReturn{
            AttributeTypes: map[string]attr.Type{
                "valid":      types.BoolType,
                "node_count": types.Int64Type,
                "node_ids":   types.ListType{ElemType: types.StringType},
                "errors":     types.ListType{ElemType: types.StringType},
            },
        },
    }
}

func (f ParseWorkflowJSONFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
    var jsonStr string
    resp.Error = function.ConcatFuncErrors(req.Arguments.Get(ctx, &jsonStr))
    if resp.Error != nil {
        return
    }

    var workflow map[string]interface{}
    if err := json.Unmarshal([]byte(jsonStr), &workflow); err != nil {
        resp.Error = function.NewFuncError("Invalid JSON: " + err.Error())
        return
    }

    nodeIDs := make([]string, 0, len(workflow))
    for id := range workflow {
        nodeIDs = append(nodeIDs, id)
    }
    sort.Strings(nodeIDs)

    result := ParseWorkflowResult{
        Valid:     true,
        NodeCount: int64(len(nodeIDs)),
        NodeIDs:   nodeIDs,
        Errors:    []string{},
    }

    resp.Error = function.ConcatFuncErrors(resp.Result.Set(ctx, &result))
}
```

#### HCL Example

```hcl
locals {
  raw_workflow = file("${path.module}/workflows/txt2img.json")
  parsed       = provider::comfyui::parse_workflow_json(local.raw_workflow)
}

output "workflow_node_count" {
  value = local.parsed.node_count
}
```

### 5.2 `node_output_name`

Constructs a deterministic output reference string for wiring nodes.

#### Signature

```
provider::comfyui::node_output_name(node_id, output_index) → string
```

#### HCL Example

```hcl
locals {
  sampler_output = provider::comfyui::node_output_name("3", 0)
  # Returns "3:0"
}
```

---

## 6. Complete End-to-End Example

```hcl
# ──────────────────────────────────────────────
# main.tf — Full ComfyUI Terraform workflow
# ──────────────────────────────────────────────

terraform {
  required_version = ">= 1.8.0"

  required_providers {
    comfyui = {
      source  = "registry.terraform.io/sbuglione/comfyui"
      version = "~> 0.1"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

# 1. Provider configuration
provider "comfyui" {
  endpoint = var.comfyui_endpoint
  timeout  = 600
}

variable "comfyui_endpoint" {
  type    = string
  default = "http://localhost:8188"
}

# 2. Pre-flight: check system stats
data "comfyui_system_stats" "server" {}

output "server_gpu" {
  value = data.comfyui_system_stats.server.gpu_name
}

# 3. Pre-flight: check queue is not saturated
data "comfyui_queue" "current" {}

# 4. Upload an input image
resource "comfyui_uploaded_image" "sketch" {
  file_path = "${path.module}/images/sketch.png"
  subfolder = "terraform-inputs"
  overwrite = true
}

# 5. Define the workflow template
resource "comfyui_workflow" "img2img" {
  name        = "img2img-pipeline"
  description = "Image-to-image with uploaded sketch"

  workflow_json = templatefile("${path.module}/workflows/img2img.json.tftpl", {
    input_image = comfyui_uploaded_image.sketch.filename
    subfolder   = comfyui_uploaded_image.sketch.subfolder
  })
}

# 6. Execute the workflow
resource "comfyui_workflow_execution" "run" {
  workflow_json       = comfyui_workflow.img2img.workflow_json
  wait_for_completion = true

  lifecycle {
    precondition {
      condition     = data.comfyui_queue.current.pending_count < 20
      error_message = "ComfyUI queue overloaded (${data.comfyui_queue.current.pending_count} pending)."
    }
  }
}

# 7. Read the execution history to confirm
data "comfyui_workflow_history" "result" {
  prompt_id = comfyui_workflow_execution.run.prompt_id
}

# 8. Download the generated image
data "comfyui_output" "image" {
  filename = "ComfyUI_00001_.png"
  type     = "output"
}

resource "local_file" "generated" {
  content_base64 = data.comfyui_output.image.content_base64
  filename       = "${path.module}/output/generated.png"
}

# 9. Outputs
output "prompt_id" {
  value = comfyui_workflow_execution.run.prompt_id
}

output "execution_status" {
  value = data.comfyui_workflow_history.result.status
}
```

---

## 7. Architecture Recommendations

### 7.1 HTTP Client Design

* Instantiate a single `*http.Client` in the provider's `Configure` method and
  pass it to every resource/data-source via `resp.ResourceData`.
* Set `Transport.MaxIdleConnsPerHost` to at least 4 — ComfyUI handles concurrent
  REST requests but serialises GPU work.
* Honour the provider-level `timeout` for all REST calls. WebSocket waits should
  use a separate, longer deadline (or no deadline if the user sets
  `wait_for_completion = true` without a timeout override).

### 7.2 WebSocket Monitoring

```go
// internal/client/ws.go

func (c *Client) WaitForExecution(ctx context.Context, promptID string) (*ExecutionResult, error) {
    wsURL := strings.Replace(c.BaseURL, "http", "ws", 1) + "/ws?clientId=" + c.ClientID

    conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
    if err != nil {
        return nil, fmt.Errorf("websocket dial: %w", err)
    }
    defer conn.Close()

    for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            return nil, fmt.Errorf("websocket read: %w", err)
        }

        var event struct {
            Type string          `json:"type"`
            Data json.RawMessage `json:"data"`
        }
        if err := json.Unmarshal(msg, &event); err != nil {
            continue
        }

        switch event.Type {
        case "executed":
            // Prompt finished — fetch full history
            return c.GetHistory(ctx, promptID)
        case "execution_error":
            return nil, fmt.Errorf("execution error: %s", string(event.Data))
        case "execution_interrupted":
            return nil, fmt.Errorf("execution interrupted")
        }
    }
}
```

### 7.3 Retry and Error Handling

* ComfyUI may restart (model loading, OOM recovery). Implement exponential
  back-off with jitter for transient HTTP 5xx / connection-refused errors.
* Limit retries to **read** operations and the initial WebSocket dial. Never
  retry POST `/prompt` — that would duplicate queued work.
* Surface structured Terraform diagnostics (`resp.Diagnostics.AddError` /
  `AddWarning`) so `terraform apply` output is actionable.

```go
func withRetry(ctx context.Context, maxAttempts int, fn func() error) error {
    for attempt := 0; attempt < maxAttempts; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }
        if !isRetryable(err) {
            return err
        }
        backoff := time.Duration(1<<attempt) * time.Second
        jitter := time.Duration(rand.Intn(500)) * time.Millisecond
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff + jitter):
        }
    }
    return fmt.Errorf("max retries (%d) exceeded", maxAttempts)
}
```

### 7.4 State Design Considerations

* `comfyui_workflow` is purely local state — no API round-trip on Read.
* `comfyui_workflow_execution` stores `prompt_id` in state. On Read, it calls
  GET `/history/{prompt_id}` and updates `status`/`outputs`. If the ComfyUI
  server has been wiped, Read should **not** error — it should mark the resource
  as needing recreation (return empty state).
* `comfyui_uploaded_image` stores the file hash. On Read, the provider
  re-downloads via GET `/view` and compares hashes. A mismatch triggers update.

---

## 8. Complete API-to-Terraform Mapping Table

| ComfyUI API Endpoint | Terraform Name | Type | Key Attributes | Notes |
|---|---|---|---|---|
| POST `/prompt` | `comfyui_workflow_execution` | Resource | `workflow_json`, `prompt_id`, `status`, `outputs` | Create queues; optionally waits |
| GET `/history/{prompt_id}` | `comfyui_workflow_history` | Data Source | `prompt_id`, `status`, `outputs`, `execution_time_ms` | Also used by execution Read |
| POST `/upload/image` | `comfyui_uploaded_image` | Resource | `file_path`, `filename`, `subfolder`, `file_hash` | Multipart upload |
| POST `/upload/mask` | `comfyui_uploaded_mask` | Resource | `file_path`, `filename`, `subfolder`, `file_hash` | Same pattern as image |
| GET `/system_stats` | `comfyui_system_stats` | Data Source | `os`, `gpu_name`, `gpu_vram_total_mb`, `comfyui_version` | Pre-flight checks |
| GET `/queue` | `comfyui_queue` | Data Source | `running_count`, `pending_count` | Queue gating |
| GET `/object_info` | `comfyui_node_info` | Data Source | `node_name`, `nodes` | Node discovery |
| GET `/view` | `comfyui_output` | Data Source | `filename`, `content_base64`, `content_length` | Binary download |
| POST `/interrupt` | (internal) | — | — | Used by execution Delete |
| WS `/ws` | (internal) | — | — | Used by execution Create |
| — | `comfyui_workflow` | Resource | `name`, `workflow_json`, `tags` | Local-only template |
| — | `parse_workflow_json` | Function | `json_string` → object | Validation helper |
| — | `node_output_name` | Function | `node_id`, `output_index` → string | Wiring helper |

---

## 9. File and Package Layout

```
terraform-provider-comfyui/
├── main.go
├── internal/
│   ├── provider/
│   │   └── provider.go              # Provider schema + Configure
│   ├── client/
│   │   ├── client.go                # HTTP client struct
│   │   ├── prompt.go                # QueuePrompt, GetHistory
│   │   ├── upload.go                # UploadImage, UploadMask
│   │   ├── system.go                # GetSystemStats
│   │   ├── queue.go                 # GetQueue
│   │   ├── nodes.go                 # GetObjectInfo
│   │   ├── view.go                  # GetView (output download)
│   │   └── ws.go                    # WebSocket helpers
│   ├── resources/
│   │   ├── workflow.go              # comfyui_workflow
│   │   ├── workflow_execution.go    # comfyui_workflow_execution
│   │   ├── uploaded_image.go        # comfyui_uploaded_image
│   │   └── uploaded_mask.go         # comfyui_uploaded_mask
│   ├── datasources/
│   │   ├── system_stats.go          # comfyui_system_stats
│   │   ├── queue.go                 # comfyui_queue
│   │   ├── node_info.go             # comfyui_node_info
│   │   ├── workflow_history.go      # comfyui_workflow_history
│   │   └── output.go               # comfyui_output
│   └── functions/
│       ├── parse_workflow_json.go   # parse_workflow_json
│       └── node_output_name.go      # node_output_name
└── docs/                            # terraform-plugin-docs generated
```

---

## 10. Open Questions

1. **Server-side workflow storage:** ComfyUI does not currently expose CRUD for
   saved workflows via API. If this is added upstream, `comfyui_workflow` should
   migrate from local-only state to API-backed.
2. **Ephemeral vs standard resource for execution:** Terraform 1.10 ephemeral
   resources are a better semantic fit. The provider should support both modes
   during the transition period.
3. **Large output handling:** Base64-encoding multi-megabyte images in state is
   expensive. Consider a `save_to_file` attribute that writes directly to disk
   and stores only the path and hash.
4. **Authentication patterns:** ComfyUI itself has no auth. The `api_key`
   attribute targets reverse-proxy setups (e.g. nginx basic-auth or token
   headers). Document which header the key is sent in (`Authorization: Bearer`).
5. **Batch executions:** Should there be a `comfyui_batch_execution` resource
   that queues N prompts and waits for all? Or should users rely on `count` /
   `for_each` on `comfyui_workflow_execution`?

---

*This document is intended for consumption by an AI coding agent. All schemas,
code samples, and mappings are authoritative for implementation purposes.*
