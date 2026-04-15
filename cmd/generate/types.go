package main

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

type NodeUIHints struct {
	Version        string                `json:"version"`
	ExtractedAt    string                `json:"extracted_at"`
	ComfyUICommit  string                `json:"comfyui_commit_sha"`
	ComfyUIVersion string                `json:"comfyui_version"`
	TotalNodes     int                   `json:"total_nodes"`
	FailedNodes    []NodeUIHintFailure   `json:"failed_nodes"`
	Nodes          map[string]NodeUIHint `json:"nodes"`
}

type NodeUIHintFailure struct {
	NodeType string `json:"node_type"`
	Message  string `json:"message"`
}

type NodeUIHint struct {
	NodeType       string                  `json:"node_type"`
	DisplayName    string                  `json:"display_name"`
	MinWidth       *float64                `json:"min_width"`
	MinHeight      *float64                `json:"min_height"`
	ComputedWidth  *float64                `json:"computed_width"`
	ComputedHeight *float64                `json:"computed_height"`
	Widgets        map[string]WidgetUIHint `json:"widgets"`
}

type WidgetUIHint struct {
	WidgetType     string   `json:"widget_type"`
	HasComputeSize bool     `json:"has_compute_size"`
	ComputedWidth  *float64 `json:"computed_width"`
	ComputedHeight *float64 `json:"computed_height"`
	MinNodeWidth   *float64 `json:"min_node_width"`
	MinNodeHeight  *float64 `json:"min_node_height"`
}

type NodeUIHintsTemplateData struct {
	Version        string
	ExtractedAt    string
	ComfyUICommit  string
	ComfyUIVersion string
	TotalNodes     int
	Nodes          []NodeUIHintTemplateNode
}

type NodeUIHintTemplateNode struct {
	NodeType       string
	MinWidth       float64
	MinHeight      float64
	ComputedWidth  float64
	ComputedHeight float64
	Widgets        []WidgetUIHintTemplateNode
}

type WidgetUIHintTemplateNode struct {
	Name           string
	WidgetType     string
	HasComputeSize bool
	ComputedWidth  float64
	ComputedHeight float64
	MinNodeWidth   float64
	MinNodeHeight  float64
}

type NodeSchemaTemplateData struct {
	Version        string
	ExtractedAt    string
	ComfyUIVersion string
	TotalNodes     int
	Nodes          []GeneratedNodeSchema
}

type GeneratedNodeSchema struct {
	NodeType       string
	TerraformType  string
	DisplayName    string
	Description    string
	Category       string
	OutputNode     bool
	Deprecated     bool
	Experimental   bool
	RequiredInputs []GeneratedNodeSchemaInput
	OptionalInputs []GeneratedNodeSchemaInput
	HiddenInputs   []GeneratedNodeSchemaHiddenInput
	Outputs        []GeneratedNodeSchemaOutput
}

type GeneratedNodeSchemaInput struct {
	Name                         string
	Type                         string
	Required                     bool
	IsLinkType                   bool
	ValidationKind               string
	InventoryKind                string
	SupportsStrictPlanValidation bool
	DefaultValue                 string
	HasDefaultValue              bool
	MinValue                     string
	HasMinValue                  bool
	MaxValue                     string
	HasMaxValue                  bool
	StepValue                    string
	HasStepValue                 bool
	EnumValues                   []string
	Multiline                    bool
	DynamicOptions               bool
	DynamicOptionsSource         string
	DynamicComboOptions          []GeneratedDynamicComboOption
	Tooltip                      string
	DisplayName                  string
}

type GeneratedDynamicComboOption struct {
	Key    string
	Inputs []GeneratedNodeSchemaInput
}

type GeneratedNodeSchemaHiddenInput struct {
	Name string
	Type string
}

type GeneratedNodeSchemaOutput struct {
	Name      string
	Type      string
	SlotIndex int
	IsList    bool
}

// Primitive ComfyUI types that map to specific Terraform attribute types.
var primitiveTypes = map[string]string{
	"INT":     "int64",
	"FLOAT":   "float64",
	"STRING":  "string",
	"BOOLEAN": "bool",
	"COMBO":   "string",
}

// isPrimitiveType returns true if the ComfyUI type maps to a non-string Terraform type.
func isPrimitiveType(t string) bool {
	_, ok := primitiveTypes[t]
	return ok
}

// tfAttributeType returns the Terraform schema attribute constructor for a ComfyUI type.
func tfAttributeType(t string) string {
	switch t {
	case "INT":
		return "schema.Int64Attribute"
	case "FLOAT":
		return "schema.Float64Attribute"
	case "BOOLEAN":
		return "schema.BoolAttribute"
	default:
		return "schema.StringAttribute"
	}
}

func isDynamicComboType(t string) bool {
	return strings.HasPrefix(t, "COMFY_DYNAMICCOMBO")
}

// goFieldType returns the Go types.* type for a ComfyUI type.
func goFieldType(t string) string {
	if isDynamicComboType(t) {
		return "types.Object"
	}
	switch t {
	case "INT":
		return "types.Int64"
	case "FLOAT":
		return "types.Float64"
	case "BOOLEAN":
		return "types.Bool"
	default:
		return "types.String"
	}
}

// clampInt64 clamps a float64 value to the int64 range.
func clampInt64(v float64) int64 {
	if v >= float64(math.MaxInt64) {
		return math.MaxInt64
	}
	if v <= float64(math.MinInt64) {
		return math.MinInt64
	}
	return int64(v)
}

var nonAlphanumRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

var terraformReservedRootNames = map[string]bool{
	"count":       true,
	"for_each":    true,
	"depends_on":  true,
	"lifecycle":   true,
	"provider":    true,
	"provisioner": true,
	"connection":  true,
}

// sanitizeName converts any input name to a valid Go/Terraform snake_case identifier.
func sanitizeName(name string) string {
	// Replace non-alphanumeric chars with underscores
	s := nonAlphanumRe.ReplaceAllString(name, "_")
	// Trim leading/trailing underscores
	s = strings.Trim(s, "_")
	// Lowercase
	s = strings.ToLower(s)
	// If it starts with a digit, prefix with underscore
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		s = "_" + s
	}
	if s == "" {
		s = "unnamed"
	}
	if terraformReservedRootNames[s] {
		s += "_value"
	}
	return s
}

// toSnakeCase converts a PascalCase or camelCase string to snake_case.
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := rune(s[i-1])
				if unicode.IsLower(prev) || unicode.IsDigit(prev) {
					result = append(result, '_')
				} else if unicode.IsUpper(prev) && i+1 < len(s) && unicode.IsLower(rune(s[i+1])) {
					result = append(result, '_')
				}
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// toPascalCase converts a snake_case string to PascalCase.
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	var result strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		// Handle common abbreviations
		upper := strings.ToUpper(p)
		switch upper {
		case "ID", "API", "URL", "HTTP", "IO", "IP", "JSON", "XML", "SQL", "CLI", "UI", "UUID", "VAE", "CLIP", "GPU", "CPU", "RAM":
			result.WriteString(upper)
		default:
			result.WriteString(strings.ToUpper(p[:1]) + p[1:])
		}
	}
	r := result.String()
	// If the result starts with a digit, prefix with X
	if len(r) > 0 && r[0] >= '0' && r[0] <= '9' {
		r = "X" + r
	}
	return r
}

// goReservedWords that can't be used as Go identifiers.
var goReservedWords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// safeGoName returns a Go-safe identifier, appending _ if it's a reserved word.
func safeGoName(name string) string {
	if goReservedWords[strings.ToLower(name)] {
		return name + "_"
	}
	return name
}
