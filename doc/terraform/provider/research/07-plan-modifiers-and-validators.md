# 07 — Plan Modifiers and Validators (Plugin Framework)

> Research for the AI coding harness building `terraform-provider-comfyui`.  
> All code uses the **Plugin Framework** (`terraform-plugin-framework`), never SDKv2.

## 1. Plan Modifiers — What and When

Plan modifiers run during the **plan phase** (before apply). They modify the planned value of a single attribute. Uses: preserve state for computed fields, force replacement, set defaults, normalize values. Declared per-attribute via `PlanModifiers`. They execute in order listed.

## 2. Built-in Plan Modifiers

Every type has a `*planmodifier` package with `UseStateForUnknown()`, `RequiresReplace()`, and `RequiresReplaceIfConfigured()`. Packages: `stringplanmodifier`, `int64planmodifier`, `boolplanmodifier`, `float64planmodifier`, `listplanmodifier`, `setplanmodifier`, `mapplanmodifier`, `objectplanmodifier`.

```go
import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// String — keeps state when computed, forces recreate on change, or only when explicitly configured
schema.StringAttribute{
	Computed: true, Optional: true,
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.UseStateForUnknown(),
		stringplanmodifier.RequiresReplace(),
		stringplanmodifier.RequiresReplaceIfConfigured(),
	},
}
// Int64
schema.Int64Attribute{Computed: true, PlanModifiers: []planmodifier.Int64{
	int64planmodifier.UseStateForUnknown(), int64planmodifier.RequiresReplace(),
}}
// Bool
schema.BoolAttribute{Computed: true, PlanModifiers: []planmodifier.Bool{
	boolplanmodifier.UseStateForUnknown(), boolplanmodifier.RequiresReplace(),
}}
// Float64
schema.Float64Attribute{Computed: true, PlanModifiers: []planmodifier.Float64{
	float64planmodifier.UseStateForUnknown(), float64planmodifier.RequiresReplace(),
}}
// List
schema.ListAttribute{ElementType: types.StringType, Optional: true, Computed: true,
	PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown(), listplanmodifier.RequiresReplace()},
}
// Set
schema.SetAttribute{ElementType: types.StringType, Optional: true,
	PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown(), setplanmodifier.RequiresReplace()},
}
// Map
schema.MapAttribute{ElementType: types.StringType, Optional: true,
	PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown(), mapplanmodifier.RequiresReplace()},
}
// Object
schema.ObjectAttribute{Optional: true,
	PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown(), objectplanmodifier.RequiresReplace()},
}
```

## 3. Custom Plan Modifiers

Implement `planmodifier.String` (or equivalent for other types):

```go
type String interface {
	Description(ctx context.Context) string
	MarkdownDescription(ctx context.Context) string
	PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse)
}
```

**Request fields:** `Config`, `ConfigValue`, `Path`, `PathExpression`, `Plan`, `PlanValue`, `State`, `StateValue`.  
**Response fields:** `PlanValue` (set to override), `RequiresReplace` (bool), `Diagnostics`.

### 3.1 Custom Modifier — Default Value When Null

```go
package planmodifiers

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type stringDefaultValue struct{ defaultVal string }

func StringDefaultValue(val string) planmodifier.String { return &stringDefaultValue{defaultVal: val} }

func (m *stringDefaultValue) Description(_ context.Context) string          { return "Sets default when null." }
func (m *stringDefaultValue) MarkdownDescription(_ context.Context) string  { return "Sets default when null." }
func (m *stringDefaultValue) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() {
		resp.PlanValue = types.StringValue(m.defaultVal)
	}
}
```

Usage: `PlanModifiers: []planmodifier.String{planmodifiers.StringDefaultValue("default-name")}` on an `Optional: true, Computed: true` attribute.

### 3.2 Custom Modifier — Normalize to Lowercase

```go
package planmodifiers

import (
	"context"
	"strings"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type lowercaseModifier struct{}

func ToLowercase() planmodifier.String { return &lowercaseModifier{} }

func (m *lowercaseModifier) Description(_ context.Context) string         { return "Normalizes to lowercase." }
func (m *lowercaseModifier) MarkdownDescription(_ context.Context) string { return "Normalizes to lowercase." }
func (m *lowercaseModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() { return }
	resp.PlanValue = types.StringValue(strings.ToLower(req.PlanValue.ValueString()))
}
```

## 4. Resource-Level Plan Modification

For cross-attribute logic, implement `resource.ResourceWithModifyPlan`:

```go
var _ resource.ResourceWithModifyPlan = &exampleResource{}

func (r *exampleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() { return } // destroy — nothing to do

	var plan struct {
		Protocol types.String `tfsdk:"protocol"`
		Port     types.Int64  `tfsdk:"port"`
	}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() { return }

	// Cross-attribute default: default port for TCP.
	if plan.Protocol.ValueString() == "tcp" && plan.Port.IsNull() {
		plan.Port = types.Int64Value(8080)
		resp.Diagnostics.Append(req.Plan.Set(ctx, &plan)...)
	}
}
```

Use **attribute-level** modifiers for single-attribute concerns (defaults, replace, normalize). Use **resource-level** `ModifyPlan` when attribute A's plan depends on attribute B.

## 5. Default Values via Plan Modifiers vs Schema Defaults

The `Default` schema field (`stringdefault.StaticString(...)`) runs **before** plan modifiers and suits static values. Use plan modifier defaults for dynamic logic.

```go
import "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"

// Static default — simpler, preferred for constants
schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("us-east-1")}

// Dynamic default — use a custom plan modifier (see §3.1)
schema.StringAttribute{Optional: true, Computed: true,
	PlanModifiers: []planmodifier.String{planmodifiers.StringDefaultValue("us-east-1")},
}
```

The attribute **must** be `Computed: true` (in addition to `Optional: true`) so Terraform knows the provider may supply the value.

## 6. Validators — What and When

Validators run during the **plan phase** to check configuration. They never mutate the plan — they only return diagnostics. Declared per-attribute via `Validators`, or at resource level for cross-attribute checks.

## 7. Built-in Validators

Install: `go get github.com/hashicorp/terraform-plugin-framework-validators`

```go
import (
	"regexp"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// String validators
schema.StringAttribute{Required: true, Validators: []validator.String{
	stringvalidator.LengthBetween(1, 255),
	stringvalidator.LengthAtLeast(1),
	stringvalidator.LengthAtMost(255),
	stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z][a-z0-9_-]*$`),
		"must start with a letter, contain only lowercase alphanumerics/hyphens/underscores"),
	stringvalidator.OneOf("small", "medium", "large"),
	stringvalidator.NoneOf("admin", "root"),
}}
// Int64 validators
schema.Int64Attribute{Required: true, Validators: []validator.Int64{
	int64validator.Between(1, 65535), int64validator.AtLeast(1), int64validator.AtMost(65535),
}}
// Float64 validators
schema.Float64Attribute{Required: true, Validators: []validator.Float64{float64validator.Between(0.0, 1.0)}}
// List validators
schema.ListAttribute{ElementType: types.StringType, Required: true, Validators: []validator.List{
	listvalidator.SizeAtLeast(1), listvalidator.SizeAtMost(10), listvalidator.SizeBetween(1, 10),
}}
```

## 8. Custom Validators

Implement `validator.String` (or equivalent for other types):

```go
type String interface {
	Description(ctx context.Context) string
	MarkdownDescription(ctx context.Context) string
	ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse)
}
```

**Request fields:** `Config`, `ConfigValue`, `Path`, `PathExpression`.  
**Response fields:** `Diagnostics`.

### 8.1 Valid URL Validator

```go
package validators

import (
	"context"
	"net/url"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type validURLValidator struct{}
func ValidURL() validator.String { return &validURLValidator{} }

func (v *validURLValidator) Description(_ context.Context) string         { return "Value must be a valid URL with scheme and host." }
func (v *validURLValidator) MarkdownDescription(_ context.Context) string { return "Value must be a valid URL with scheme and host." }
func (v *validURLValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() { return }
	val := req.ConfigValue.ValueString()
	u, err := url.Parse(val)
	if err != nil || u.Scheme == "" || u.Host == "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid URL",
			"The value "+val+" is not a valid URL. A scheme (e.g. https) and host are required.")
	}
}
```

### 8.2 Custom Pattern Validator

```go
package validators

import (
	"context"
	"fmt"
	"regexp"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type patternValidator struct{ pattern *regexp.Regexp; message string }
func MatchesPattern(p *regexp.Regexp, msg string) validator.String { return &patternValidator{pattern: p, message: msg} }

func (v *patternValidator) Description(_ context.Context) string         { return fmt.Sprintf("Must match: %s", v.pattern) }
func (v *patternValidator) MarkdownDescription(_ context.Context) string { return fmt.Sprintf("Must match: `%s`", v.pattern) }
func (v *patternValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() { return }
	val := req.ConfigValue.ValueString()
	if !v.pattern.MatchString(val) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Value", fmt.Sprintf("%s: got %q", v.message, val))
	}
}
```

Usage: `Validators: []validator.String{validators.MatchesPattern(regexp.MustCompile(`+"`"+`^wf-[a-z0-9]+$`+"`"+`), "must be a workflow ID starting with 'wf-'")}`

## 9. Cross-Attribute Validation

Implement `resource.ResourceWithConfigValidators`:

```go
type ResourceWithConfigValidators interface {
	Resource
	ConfigValidators(ctx context.Context) []resource.ConfigValidator
}
```

Each `resource.ConfigValidator`:

```go
type ConfigValidator interface {
	Description(ctx context.Context) string
	MarkdownDescription(ctx context.Context) string
	ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse)
}
```

### 9.1 Port Required When Protocol Is TCP

```go
package validators

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type portRequiredForTCP struct{}
func PortRequiredForTCP() resource.ConfigValidator { return &portRequiredForTCP{} }

func (v *portRequiredForTCP) Description(_ context.Context) string         { return "port required when protocol is tcp" }
func (v *portRequiredForTCP) MarkdownDescription(_ context.Context) string { return "`port` required when `protocol` is `tcp`" }
func (v *portRequiredForTCP) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg struct {
		Protocol types.String `tfsdk:"protocol"`
		Port     types.Int64  `tfsdk:"port"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() { return }
	if cfg.Protocol.IsNull() || cfg.Protocol.IsUnknown() { return }
	if cfg.Protocol.ValueString() == "tcp" && cfg.Port.IsNull() {
		resp.Diagnostics.AddError("Missing Required Attribute", `port must be set when protocol is "tcp".`)
	}
}
```

Wire it in:

```go
var _ resource.ResourceWithConfigValidators = &connectionResource{}

func (r *connectionResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{validators.PortRequiredForTCP()}
}
```

## 10. Import Path Reference

| Purpose | Package |
|---|---|
| Plan modifier interfaces | `github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier` |
| Built-in `*planmodifier` | `.../resource/schema/{string,int64,bool,float64,list,set,map,object}planmodifier` |
| Schema defaults | `.../resource/schema/{string,int64,bool,float64}default` |
| Validator interfaces | `github.com/hashicorp/terraform-plugin-framework/schema/validator` |
| Built-in validators | `github.com/hashicorp/terraform-plugin-framework-validators/{string,int64,float64,list}validator` |

## 11. References

- <https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification>
- <https://developer.hashicorp.com/terraform/plugin/framework/validation>
- <https://developer.hashicorp.com/terraform/plugin/framework/resources/default>
- <https://github.com/hashicorp/terraform-plugin-framework-validators>
