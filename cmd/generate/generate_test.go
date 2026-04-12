package main

import (
	"math"
	"strings"
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"KSampler", "k_sampler"},
		{"CLIPTextEncode", "clip_text_encode"},
		{"CheckpointLoaderSimple", "checkpoint_loader_simple"},
		{"VAEDecode", "vae_decode"},
		{"BasicScheduler", "basic_scheduler"},
		{"LumaReferenceNode", "luma_reference_node"},
		{"SaveImage", "save_image"},
		{"simple", "simple"},
		{"alreadyLower", "already_lower"},
		{"ABCDef", "abc_def"},
		{"HTTPServer", "http_server"},
		{"getHTTPResponse", "get_http_response"},
		{"XMLParser", "xml_parser"},
		{"IOReader", "io_reader"},
		{"a", "a"},
		{"AB", "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"model_name", "model_name"},
		{"some-input", "some_input"},
		{"hello world", "hello_world"},
		{"___leading___", "leading"},
		{"123start", "_123start"},
		{"UPPER_CASE", "upper_case"},
		{"special!@#chars", "special_chars"},
		{"", "unnamed"},
		{"a.b.c", "a_b_c"},
		{"already_clean", "already_clean"},
		{"count", "count_value"},
		{"for_each", "for_each_value"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"model", "Model"},
		{"seed", "Seed"},
		{"sampler_name", "SamplerName"},
		{"cfg", "Cfg"},
		{"clip_text_encode", "CLIPTextEncode"},
		{"vae_decode", "VAEDecode"},
		{"api_key", "APIKey"},
		{"url_path", "URLPath"},
		{"gpu_device", "GPUDevice"},
		{"cpu_count", "CPUCount"},
		{"io_reader", "IOReader"},
		{"json_data", "JSONData"},
		{"my_id", "MyID"},
		{"uuid_field", "UUIDField"},
		// Digit prefix gets X prepended
		{"3d_model", "X3dModel"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTfAttributeType(t *testing.T) {
	tests := []struct {
		comfyType string
		expected  string
	}{
		{"INT", "schema.Int64Attribute"},
		{"FLOAT", "schema.Float64Attribute"},
		{"BOOLEAN", "schema.BoolAttribute"},
		{"STRING", "schema.StringAttribute"},
		{"COMBO", "schema.StringAttribute"},
		{"MODEL", "schema.StringAttribute"},
		{"CLIP", "schema.StringAttribute"},
		{"IMAGE", "schema.StringAttribute"},
		{"LATENT", "schema.StringAttribute"},
		{"UNKNOWN_TYPE", "schema.StringAttribute"},
	}
	for _, tt := range tests {
		t.Run(tt.comfyType, func(t *testing.T) {
			result := tfAttributeType(tt.comfyType)
			if result != tt.expected {
				t.Errorf("tfAttributeType(%q) = %q, want %q", tt.comfyType, result, tt.expected)
			}
		})
	}
}

func TestGoFieldType(t *testing.T) {
	tests := []struct {
		comfyType string
		expected  string
	}{
		{"INT", "types.Int64"},
		{"FLOAT", "types.Float64"},
		{"BOOLEAN", "types.Bool"},
		{"STRING", "types.String"},
		{"COMBO", "types.String"},
		{"MODEL", "types.String"},
		{"CLIP", "types.String"},
		{"IMAGE", "types.String"},
	}
	for _, tt := range tests {
		t.Run(tt.comfyType, func(t *testing.T) {
			result := goFieldType(tt.comfyType)
			if result != tt.expected {
				t.Errorf("goFieldType(%q) = %q, want %q", tt.comfyType, result, tt.expected)
			}
		})
	}
}

func TestIsPrimitiveType(t *testing.T) {
	primitives := []string{"INT", "FLOAT", "STRING", "BOOLEAN", "COMBO"}
	for _, p := range primitives {
		if !isPrimitiveType(p) {
			t.Errorf("isPrimitiveType(%q) = false, want true", p)
		}
	}

	nonPrimitives := []string{"MODEL", "CLIP", "IMAGE", "LATENT", "CONDITIONING", "VAE"}
	for _, np := range nonPrimitives {
		if isPrimitiveType(np) {
			t.Errorf("isPrimitiveType(%q) = true, want false", np)
		}
	}
}

func TestSafeGoName(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		// Reserved words get underscore appended
		{"break", "break_"},
		{"case", "case_"},
		{"func", "func_"},
		{"type", "type_"},
		{"map", "map_"},
		{"range", "range_"},
		{"return", "return_"},
		{"var", "var_"},
		{"select", "select_"},
		{"interface", "interface_"},
		// Non-reserved words pass through
		{"Model", "Model"},
		{"Seed", "Seed"},
		{"custom", "custom"},
		{"sampler", "sampler"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := safeGoName(tt.input)
			if result != tt.expected {
				t.Errorf("safeGoName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestClampInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected int64
	}{
		{"zero", 0, 0},
		{"positive", 42.0, 42},
		{"negative", -100.0, -100},
		{"truncate_fraction", 3.7, 3},
		{"max_int64_overflow", float64(math.MaxInt64) * 2, math.MaxInt64},
		{"min_int64_overflow", float64(math.MinInt64) * 2, math.MinInt64},
		{"exact_max", float64(math.MaxInt64), math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampInt64(tt.input)
			if result != tt.expected {
				t.Errorf("clampInt64(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToSnakeCaseRoundtrip(t *testing.T) {
	// PascalCase -> snake_case -> PascalCase should produce a valid Go identifier
	inputs := []string{
		"CheckpointLoaderSimple",
		"SaveImage",
		"BasicScheduler",
	}
	for _, input := range inputs {
		snake := toSnakeCase(input)
		pascal := toPascalCase(snake)
		if pascal == "" {
			t.Errorf("roundtrip from %q through snake %q produced empty PascalCase", input, snake)
		}
		// The roundtrip may not be identical (e.g., abbreviation handling) but must produce a non-empty valid name
		t.Logf("%s -> %s -> %s", input, snake, pascal)
	}
}

func TestBuildResourceDataSurfacesSchemaMetadata(t *testing.T) {
	tooltip := "Controls the scale of the parallel guidance vector."
	multiline := true
	dynamicOptions := true
	nodeDescription := "Edits an image using Bria."

	node := Node{
		ClassName:             "BriaImageEditNode",
		Description:           &nodeDescription,
		Category:              "api node/image",
		TerraformResourceName: "comfyui_bria_image_edit_node",
		Inputs: []Input{
			{
				Name:     "cfg",
				Type:     "FLOAT",
				Required: true,
				Default:  1.5,
				Min:      floatPtr(0),
				Max:      floatPtr(10),
				Step:     floatPtr(0.5),
				Tooltip:  &tooltip,
			},
			{
				Name:                 "audio_encoder_name",
				Type:                 "COMBO",
				Required:             true,
				DynamicOptions:       &dynamicOptions,
				DynamicOptionsSource: stringPtr("folder_paths.get_filename_list('audio_encoders')"),
			},
			{
				Name:      "prompt",
				Type:      "STRING",
				Required:  true,
				Default:   "",
				Multiline: &multiline,
				Tooltip:   stringPtr("Instruction to edit image"),
			},
		},
		HiddenInputs: []HiddenInput{
			{Name: "auth_token_comfy_org", Type: "AUTH_TOKEN_COMFY_ORG"},
			{Name: "unique_id", Type: "UNIQUE_ID"},
		},
		Source: SourceInfo{
			File:       "comfy_api_nodes/nodes_bria.py",
			Pattern:    "v3_api",
			LineNumber: intPtr(27),
		},
	}

	rd, warnings := buildResourceData(node)
	if len(warnings) != 0 {
		t.Fatalf("buildResourceData returned warnings: %v", warnings)
	}

	if !strings.Contains(rd.Description, "Edits an image using Bria.") {
		t.Fatalf("resource description missing node description: %q", rd.Description)
	}
	if !strings.Contains(rd.Description, "[api node/image]") {
		t.Fatalf("resource description missing category: %q", rd.Description)
	}
	if !strings.Contains(rd.Description, "Hidden runtime inputs: auth_token_comfy_org (AUTH_TOKEN_COMFY_ORG), unique_id (UNIQUE_ID).") {
		t.Fatalf("resource description missing hidden inputs: %q", rd.Description)
	}
	if !strings.Contains(rd.Description, "Source: comfy_api_nodes/nodes_bria.py:27 (v3_api).") {
		t.Fatalf("resource description missing source metadata: %q", rd.Description)
	}

	attrDefs := strings.Join(extractAttrDefinitions(rd.Attributes), "\n")
	if !strings.Contains(attrDefs, "Default: 1.5.") {
		t.Fatalf("attribute descriptions missing default hint: %s", attrDefs)
	}
	if !strings.Contains(attrDefs, "Allowed range: 0 to 10.") {
		t.Fatalf("attribute descriptions missing range hint: %s", attrDefs)
	}
	if !strings.Contains(attrDefs, "Step: 0.5.") {
		t.Fatalf("attribute descriptions missing step hint: %s", attrDefs)
	}
	if !strings.Contains(attrDefs, "Tooltip: Controls the scale of the parallel guidance vector.") {
		t.Fatalf("attribute descriptions missing tooltip: %s", attrDefs)
	}
	if !strings.Contains(attrDefs, "Supports multiline text.") {
		t.Fatalf("attribute descriptions missing multiline hint: %s", attrDefs)
	}
	if !strings.Contains(attrDefs, "Dynamic options are resolved by ComfyUI at runtime from: folder_paths.get_filename_list('audio_encoders').") {
		t.Fatalf("attribute descriptions missing dynamic options source: %s", attrDefs)
	}
}

func TestBuildNodeUIHintsTemplateDataSortsNodesAndWidgets(t *testing.T) {
	hints := NodeUIHints{
		Version:        "1.0.0",
		ExtractedAt:    "2026-04-11T00:00:00Z",
		ComfyUICommit:  "abc123",
		ComfyUIVersion: "v0-test",
		Nodes: map[string]NodeUIHint{
			"ZNode": {
				MinWidth:  floatPtr(240),
				MinHeight: floatPtr(120),
				Widgets: map[string]WidgetUIHint{
					"zeta": {WidgetType: "string"},
					"alpha": {
						WidgetType:    "customtext",
						MinNodeWidth:  floatPtr(400),
						MinNodeHeight: floatPtr(200),
					},
				},
			},
			"ANode": {
				MinWidth:  floatPtr(300),
				MinHeight: floatPtr(180),
			},
		},
	}

	data := buildNodeUIHintsTemplateData(hints)
	if data.TotalNodes != 2 {
		t.Fatalf("expected 2 nodes, got %d", data.TotalNodes)
	}
	if data.Nodes[0].NodeType != "ANode" || data.Nodes[1].NodeType != "ZNode" {
		t.Fatalf("expected nodes sorted by type, got %#v", data.Nodes)
	}
	if len(data.Nodes[1].Widgets) != 2 {
		t.Fatalf("expected 2 widgets, got %d", len(data.Nodes[1].Widgets))
	}
	if data.Nodes[1].Widgets[0].Name != "alpha" || data.Nodes[1].Widgets[1].Name != "zeta" {
		t.Fatalf("expected widgets sorted by name, got %#v", data.Nodes[1].Widgets)
	}
	if data.Nodes[1].Widgets[0].MinNodeWidth != 400 || data.Nodes[1].Widgets[0].MinNodeHeight != 200 {
		t.Fatalf("expected preserved widget min sizes, got %#v", data.Nodes[1].Widgets[0])
	}
}

func TestBuildInputDescriptionCollapsesVerboseDynamicOptionSources(t *testing.T) {
	inp := Input{
		Name:                 "moderation",
		Type:                 "COMFY_DYNAMICCOMBO_V3",
		Required:             true,
		DynamicOptions:       boolPtr(true),
		DynamicOptionsSource: stringPtr("[IO.DynamicCombo.Option('false', []), IO.DynamicCombo.Option('true', [IO.Boolean.Input('prompt_content_moderation', default=False), IO.Boolean.Input('visual_input_moderation', default=False), IO.Boolean.Input('visual_output_moderation', default=True)])]"),
	}

	description := buildInputDescription(inp)
	if !strings.Contains(description, "Dynamic options are resolved by ComfyUI at runtime.") {
		t.Fatalf("expected generic dynamic options hint, got %q", description)
	}
	if strings.Contains(description, "prompt_content_moderation") {
		t.Fatalf("expected verbose dynamic options source to be omitted, got %q", description)
	}
}

func TestFormatSourceInfo(t *testing.T) {
	tests := []struct {
		name     string
		source   SourceInfo
		expected string
	}{
		{
			name: "file line and pattern",
			source: SourceInfo{
				File:       "comfy_api_nodes/nodes_bria.py",
				Pattern:    "v3_api",
				LineNumber: intPtr(27),
			},
			expected: "Source: comfy_api_nodes/nodes_bria.py:27 (v3_api).",
		},
		{
			name: "file only",
			source: SourceInfo{
				File: "nodes.py",
			},
			expected: "Source: nodes.py.",
		},
		{
			name: "pattern only",
			source: SourceInfo{
				Pattern: "v1_core",
			},
			expected: "Source pattern: v1_core.",
		},
		{
			name: "line without file falls back to pattern",
			source: SourceInfo{
				Pattern:    "v3_api",
				LineNumber: intPtr(27),
			},
			expected: "Source pattern: v3_api.",
		},
		{
			name:     "empty",
			source:   SourceInfo{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatSourceInfo(tt.source); got != tt.expected {
				t.Fatalf("formatSourceInfo() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatRangeHint(t *testing.T) {
	tests := []struct {
		name     string
		min      *float64
		max      *float64
		expected string
	}{
		{
			name:     "min and max",
			min:      floatPtr(1),
			max:      floatPtr(5),
			expected: "Allowed range: 1 to 5.",
		},
		{
			name:     "min only",
			min:      floatPtr(0),
			expected: "Minimum value: 0.",
		},
		{
			name:     "max only",
			max:      floatPtr(10),
			expected: "Maximum value: 10.",
		},
		{
			name:     "empty",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatRangeHint(tt.min, tt.max); got != tt.expected {
				t.Fatalf("formatRangeHint() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSentence(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		value    string
		expected string
	}{
		{
			name:     "adds trailing period",
			label:    "Tooltip",
			value:    "Controls guidance",
			expected: "Tooltip: Controls guidance.",
		},
		{
			name:     "preserves punctuation",
			label:    "Tooltip",
			value:    "Already punctuated!",
			expected: "Tooltip: Already punctuated!",
		},
		{
			name:     "empty",
			label:    "Tooltip",
			value:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sentence(tt.label, tt.value); got != tt.expected {
				t.Fatalf("sentence() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func extractAttrDefinitions(attrs []AttrData) []string {
	definitions := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		definitions = append(definitions, attr.Definition)
	}
	return definitions
}

func floatPtr(v float64) *float64 {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}
