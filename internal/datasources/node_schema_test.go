package datasources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestNodeSchemaDataSourceSchema_ExposesStructuredContracts(t *testing.T) {
	ds := NewNodeSchemaDataSource().(*NodeSchemaDataSource)
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	requiredAttr, ok := resp.Schema.Attributes["required_inputs"].(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected required_inputs to be a list nested attribute, got %T", resp.Schema.Attributes["required_inputs"])
	}
	if _, ok := requiredAttr.NestedObject.Attributes["default_value"].(datasourceschema.StringAttribute); !ok {
		t.Fatalf("expected required_inputs.default_value to be a string attribute, got %T", requiredAttr.NestedObject.Attributes["default_value"])
	}
	if _, ok := requiredAttr.NestedObject.Attributes["enum_values"].(datasourceschema.ListAttribute); !ok {
		t.Fatalf("expected required_inputs.enum_values to be a list attribute, got %T", requiredAttr.NestedObject.Attributes["enum_values"])
	}

	optionalAttr, ok := resp.Schema.Attributes["optional_inputs"].(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected optional_inputs to be a list nested attribute, got %T", resp.Schema.Attributes["optional_inputs"])
	}
	if _, ok := optionalAttr.NestedObject.Attributes["dynamic_options_source"].(datasourceschema.StringAttribute); !ok {
		t.Fatalf("expected optional_inputs.dynamic_options_source to be a string attribute, got %T", optionalAttr.NestedObject.Attributes["dynamic_options_source"])
	}

	outputAttr, ok := resp.Schema.Attributes["outputs"].(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected outputs to be a list nested attribute, got %T", resp.Schema.Attributes["outputs"])
	}
	if _, ok := outputAttr.NestedObject.Attributes["is_list"].(datasourceschema.BoolAttribute); !ok {
		t.Fatalf("expected outputs.is_list to be a bool attribute, got %T", outputAttr.NestedObject.Attributes["is_list"])
	}
}

func TestNodeSchemaLookupGeneratedContract_ReturnsStructuredContract(t *testing.T) {
	clip, ok := lookupGeneratedNodeSchema("CLIPTextEncode")
	if !ok {
		t.Fatal("expected generated schema for CLIPTextEncode")
	}

	if clip.DisplayName != "CLIP Text Encode (Prompt)" {
		t.Fatalf("unexpected display name: %q", clip.DisplayName)
	}
	if len(clip.RequiredInputs) != 2 {
		t.Fatalf("expected 2 required inputs, got %d", len(clip.RequiredInputs))
	}
	if clip.RequiredInputs[0].Name != "text" || !clip.RequiredInputs[0].Multiline {
		t.Fatalf("expected first required input to be multiline text, got %#v", clip.RequiredInputs[0])
	}
	if clip.RequiredInputs[1].IsLinkType != true {
		t.Fatalf("expected second required input to be a link type, got %#v", clip.RequiredInputs[1])
	}
	if len(clip.Outputs) != 1 || clip.Outputs[0].Name != "CONDITIONING" || clip.Outputs[0].Type != "CONDITIONING" {
		t.Fatalf("unexpected outputs: %#v", clip.Outputs)
	}

	ksampler, ok := lookupGeneratedNodeSchema("KSampler")
	if !ok {
		t.Fatal("expected generated schema for KSampler")
	}
	if len(ksampler.OptionalInputs) != 0 {
		t.Fatalf("expected KSampler optional inputs to be empty, got %#v", ksampler.OptionalInputs)
	}

	foundSeed := false
	foundSampler := false
	for _, input := range ksampler.RequiredInputs {
		switch input.Name {
		case "seed":
			foundSeed = true
			if input.DefaultValue != "0" || input.MinValue != "0" || input.MaxValue != "18446744073709551615" {
				t.Fatalf("unexpected seed bounds/default: %#v", input)
			}
		case "sampler_name":
			foundSampler = true
			if !input.DynamicOptions || input.DynamicOptionsSource != "comfy.samplers.KSampler.SAMPLERS" {
				t.Fatalf("unexpected sampler_name metadata: %#v", input)
			}
		}
	}
	if !foundSeed {
		t.Fatal("expected seed input in KSampler schema")
	}
	if !foundSampler {
		t.Fatal("expected sampler_name input in KSampler schema")
	}
}
