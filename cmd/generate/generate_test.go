package main

import (
	"math"
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
