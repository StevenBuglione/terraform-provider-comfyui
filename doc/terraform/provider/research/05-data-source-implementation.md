# Data Source Implementation in the Terraform Plugin Framework

Data sources let Terraform read external data without managing its lifecycle.
They implement the `datasource.DataSource` interface: `Metadata`, `Schema`, `Read`.

---

## 1. The `datasource.DataSource` Interface
```go
package datasource

import "context"

type DataSource interface {
    Metadata(ctx context.Context, req MetadataRequest, resp *MetadataResponse)
    Schema(ctx context.Context, req SchemaRequest, resp *SchemaResponse)
    Read(ctx context.Context, req ReadRequest, resp *ReadResponse)
}
```

**Metadata** — sets the type name used in `data "<type_name>" "<label>"` HCL blocks.
**Schema** — declares attributes (inputs and outputs), their types, and constraints.
**Read** — called every plan/apply to fetch fresh data from the external system.

```go
func (d *serverDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_server"
}
```

> **Important:** Data sources use `datasource/schema`, not `resource/schema`.

---

## 2. How Data Sources Differ from Resources

| Aspect              | Resource                               | Data Source                     |
|---------------------|----------------------------------------|---------------------------------|
| HCL block           | `resource "type" "label" {}`           | `data "type" "label" {}`        |
| Interface methods   | Metadata, Schema, Create, Read, Update, Delete | Metadata, Schema, Read   |
| State lifecycle     | Managed — tracks create/update/destroy | None — re-read every run        |
| Import support      | Yes (`ImportState`)                    | No                              |
| Purpose             | Manage infrastructure                  | Look up existing infrastructure |

- **Read-only:** Data sources never create, modify, or destroy external objects.
- **No state lifecycle:** Each plan/apply re-reads; no drift detection.
- **No import:** `terraform import` does not apply to data sources.

---

## 3. Data Source Model Struct with `tfsdk` Tags

All fields must use Plugin Framework types (`types.String`, `types.Int64`, etc.).
Tag values must exactly match schema attribute names. Lookup attributes are
`Required: true`; output attributes are `Computed: true`.

```go
package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type serverDataSourceModel struct {
    Name    types.String `tfsdk:"name"`       // Required — lookup key
    ID      types.String `tfsdk:"id"`         // Computed — from API
    Status  types.String `tfsdk:"status"`     // Computed
    Address types.String `tfsdk:"address"`    // Computed
    Port    types.Int64  `tfsdk:"port"`       // Computed
    GPU     types.Bool   `tfsdk:"gpu_enabled"` // Computed
}
```

Framework types carry null/unknown semantics that bare Go primitives cannot
represent, which is essential for correct Terraform plan behaviour.

---

## 4. Read Method Implementation

Four steps: **read config → call API → populate model → set state**.

```go
package provider

import (
    "context"
    "fmt"

    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/hashicorp/terraform-plugin-log/tflog"
)

func (d *serverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var model serverDataSourceModel
    diags := req.Config.Get(ctx, &model)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    tflog.Debug(ctx, "Reading server data source", map[string]any{"name": model.Name.ValueString()})

    server, err := d.client.GetServerByName(model.Name.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Unable to Read Server",
            fmt.Sprintf("Could not find server %q: %s", model.Name.ValueString(), err.Error()))
        return
    }

    model.ID = types.StringValue(server.ID)
    model.Status = types.StringValue(server.Status)
    model.Address = types.StringValue(server.Address)
    model.Port = types.Int64Value(int64(server.Port))
    model.GPU = types.BoolValue(server.GPUEnabled)

    diags = resp.State.Set(ctx, &model)
    resp.Diagnostics.Append(diags...)
}
```

**Error handling rules:**
- `AddError(summary, detail)` — halts this data source.
- `AddWarning(summary, detail)` — non-fatal.
- Always check `HasError()` after `Append` and return early.

---

## 5. Common Patterns

### 5.1 Lookup by ID

```go
// Schema: "id": schema.StringAttribute{ Required: true }
server, err := d.client.GetServer(model.ID.ValueString())
if err != nil {
    resp.Diagnostics.AddError("Server not found",
        fmt.Sprintf("No server with ID %q: %s", model.ID.ValueString(), err.Error()))
    return
}
model.Name = types.StringValue(server.Name)
model.Status = types.StringValue(server.Status)
```

### 5.2 Lookup by Name

```go
// Schema: "name": schema.StringAttribute{ Required: true }
server, err := d.client.GetServerByName(model.Name.ValueString())
if err != nil {
    resp.Diagnostics.AddError("Server not found",
        fmt.Sprintf("No server named %q: %s", model.Name.ValueString(), err.Error()))
    return
}
model.ID = types.StringValue(server.ID)
```

### 5.3 List Filtering

Use `ListNestedAttribute` for plural data sources returning filtered collections.

```go
package provider

import (
    "context"

    "github.com/hashicorp/terraform-plugin-framework/attr"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

type serversDataSourceModel struct {
    StatusFilter types.String `tfsdk:"status"`
    Servers      types.List   `tfsdk:"servers"`
}

type serverItemModel struct {
    ID     types.String `tfsdk:"id"`
    Name   types.String `tfsdk:"name"`
    Status types.String `tfsdk:"status"`
}

func (d *serversDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "status": schema.StringAttribute{Optional: true},
            "servers": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id":     schema.StringAttribute{Computed: true},
                        "name":   schema.StringAttribute{Computed: true},
                        "status": schema.StringAttribute{Computed: true},
                    },
                },
            },
        },
    }
}

func (d *serversDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var model serversDataSourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
    if resp.Diagnostics.HasError() {
        return
    }

    allServers, err := d.client.ListServers()
    if err != nil {
        resp.Diagnostics.AddError("Unable to list servers", err.Error())
        return
    }

    var items []serverItemModel
    for _, s := range allServers {
        if !model.StatusFilter.IsNull() && s.Status != model.StatusFilter.ValueString() {
            continue
        }
        items = append(items, serverItemModel{
            ID: types.StringValue(s.ID), Name: types.StringValue(s.Name), Status: types.StringValue(s.Status),
        })
    }

    listValue, diags := types.ListValueFrom(ctx, types.ObjectType{
        AttrTypes: map[string]attr.Type{"id": types.StringType, "name": types.StringType, "status": types.StringType},
    }, items)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
    model.Servers = listValue
    resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
```

---

## 6. Full Working Example

A complete data source looking up a server by name, with provider registration.
### 6.1 Data Source Implementation

```go
package provider

import (
    "context"
    "fmt"

    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &serverDataSource{}
var _ datasource.DataSourceWithConfigure = &serverDataSource{}

type serverDataSource struct {
    client *ApiClient
}

type serverDataSourceModel struct {
    Name    types.String `tfsdk:"name"`
    ID      types.String `tfsdk:"id"`
    Status  types.String `tfsdk:"status"`
    Address types.String `tfsdk:"address"`
}

func NewServerDataSource() datasource.DataSource {
    return &serverDataSource{}
}

func (d *serverDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_server"
}

func (d *serverDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Look up a ComfyUI server by its unique name.",
        Attributes: map[string]schema.Attribute{
            "name":    schema.StringAttribute{Description: "The unique name of the server.", Required: true},
            "id":      schema.StringAttribute{Description: "The server's unique identifier.", Computed: true},
            "status":  schema.StringAttribute{Description: "Current status (running, stopped).", Computed: true},
            "address": schema.StringAttribute{Description: "Network address of the server.", Computed: true},
        },
    }
}

func (d *serverDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    client, ok := req.ProviderData.(*ApiClient)
    if !ok {
        resp.Diagnostics.AddError("Unexpected Configure Type", "Expected *ApiClient.")
        return
    }
    d.client = client
}

func (d *serverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var model serverDataSourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
    if resp.Diagnostics.HasError() {
        return
    }

    tflog.Debug(ctx, "Reading server data source", map[string]any{"name": model.Name.ValueString()})

    server, err := d.client.GetServerByName(model.Name.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Unable to Read Server",
            fmt.Sprintf("Could not find server %q: %s", model.Name.ValueString(), err.Error()))
        return
    }

    model.ID = types.StringValue(server.ID)
    model.Status = types.StringValue(server.Status)
    model.Address = types.StringValue(server.Address)
    resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
```

### 6.2 Mock API Client

```go
package provider

import "fmt"

type ApiClient struct{ Endpoint, Token string }
type Server struct{ ID, Name, Status, Address string }

func (c *ApiClient) GetServerByName(name string) (*Server, error) {
    if name == "my-server" {
        return &Server{ID: "srv-123", Name: "my-server", Status: "running", Address: "10.0.0.5"}, nil
    }
    return nil, fmt.Errorf("server %q not found", name)
}
```

### 6.3 Provider Registration

```go
package provider

import (
    "context"

    "github.com/hashicorp/terraform-plugin-framework/datasource"
)

func (p *comfyuiProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        NewServerDataSource,
    }
}
```

### 6.4 HCL Usage

```hcl
data "comfyui_server" "primary" {
  name = "my-server"
}

output "server_address" {
  value = data.comfyui_server.primary.address
}
```

---

## 7. When to Use Data Sources vs Resources

**Data source:** read external state, reference existing infrastructure.
Examples — current account ID, a pre-existing workflow, available GPU types.

**Resource:** manage lifecycle (create, update, delete) of infrastructure.
Examples — a ComfyUI server instance, a workflow deployment, an API key.

| Question                                       | Data Source | Resource |
|------------------------------------------------|:-----------:|:--------:|
| Does Terraform create this object?             |      No     |   Yes    |
| Does Terraform destroy this object?            |      No     |   Yes    |
| Does Terraform detect drift on this object?    |      No     |   Yes    |
| Is the object read-only from Terraform's view? |     Yes     |    No    |
| Should `terraform import` work?                |      No     |   Yes    |

---

## 8. References

- [Plugin Framework — Data Sources](https://developer.hashicorp.com/terraform/plugin/framework/data-sources)
- [Plugin Framework — Data Source Schemas](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/schemas#data-source-schemas)
- [Plugin Framework — Data Source Configure](https://developer.hashicorp.com/terraform/plugin/framework/data-sources/configure)
- [`terraform-plugin-framework` Go docs](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework)
