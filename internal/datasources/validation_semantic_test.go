package datasources

import (
	"context"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func validationTestNodeInfo() map[string]client.NodeInfo {
	return map[string]client.NodeInfo{
		"LoadImage": {
			Input: client.NodeInputInfo{
				Required: map[string]interface{}{
					"image": []interface{}{"STRING"},
				},
			},
			InputOrder: map[string][]string{
				"required": {"image"},
			},
			Output:     []string{"IMAGE"},
			OutputName: []string{"IMAGE"},
		},
		"SaveImage": {
			Input: client.NodeInputInfo{
				Required: map[string]interface{}{
					"images": []interface{}{"IMAGE"},
				},
				Optional: map[string]interface{}{
					"filename_prefix": []interface{}{"STRING"},
				},
				Hidden: map[string]interface{}{
					"prompt":        "PROMPT",
					"extra_pnginfo": "EXTRA_PNGINFO",
					"unique_id":     "UNIQUE_ID",
				},
			},
			InputOrder: map[string][]string{
				"required": {"images"},
				"optional": {"filename_prefix"},
			},
			OutputNode: true,
		},
	}
}

func TestPromptValidationStateFromInput_ReturnsValidationContract(t *testing.T) {
	state, err := promptValidationStateFromInput("", `{
	  "1": {
	    "class_type": "SaveImage",
	    "inputs": {
	      "filename_prefix": "ComfyUI"
	    }
	  }
	}`, validationTestNodeInfo())
	if err != nil {
		t.Fatalf("promptValidationStateFromInput returned error: %v", err)
	}

	if state.Valid.ValueBool() {
		t.Fatal("expected prompt validation to fail")
	}
	if state.ErrorCount.ValueInt64() != 1 {
		t.Fatalf("expected one validation error, got %d", state.ErrorCount.ValueInt64())
	}
	var errors []string
	if diags := state.Errors.ElementsAs(context.Background(), &errors, false); diags.HasError() {
		t.Fatalf("failed to read errors list: %v", diags)
	}
	if len(errors) != 1 {
		t.Fatalf("expected one error string, got %#v", errors)
	}
	var warnings []string
	if diags := state.Warnings.ElementsAs(context.Background(), &warnings, false); diags.HasError() {
		t.Fatalf("failed to read warnings list: %v", diags)
	}
	if state.WarningCount.ValueInt64() != 0 || len(warnings) != 0 {
		t.Fatalf("expected no warnings, got count=%d warnings=%#v", state.WarningCount.ValueInt64(), warnings)
	}
	if state.ValidatedNodeCount.ValueInt64() != 1 {
		t.Fatalf("expected validated_node_count=1, got %d", state.ValidatedNodeCount.ValueInt64())
	}
	if state.NormalizedJSON.ValueString() == "" {
		t.Fatal("expected normalized_json to be populated")
	}
}

func TestWorkspaceValidationStateFromInput_ReturnsTranslationAndValidation(t *testing.T) {
	state, err := workspaceValidationStateFromInput("", `{
	  "nodes": [
	    {
	      "id": 1,
	      "type": "LoadImage",
	      "inputs": [
	        {
	          "name": "image",
	          "type": "STRING",
	          "widget": {"name": "image"},
	          "link": null
	        }
	      ],
	      "outputs": [
	        {
	          "name": "IMAGE",
	          "type": "IMAGE",
	          "links": [1]
	        }
	      ],
	      "widgets_values": ["input.png"]
	    },
	    {
	      "id": 2,
	      "type": "SaveImage",
	      "inputs": [
	        {
	          "name": "images",
	          "type": "IMAGE",
	          "link": 1
	        },
	        {
	          "name": "filename_prefix",
	          "type": "STRING",
	          "widget": {"name": "filename_prefix"},
	          "link": null
	        }
	      ],
	      "widgets_values": ["ComfyUI"]
	    }
	  ],
	  "links": [
	    [1, 1, 0, 2, 0, "IMAGE"]
	  ]
	}`, validationTestNodeInfo())
	if err != nil {
		t.Fatalf("workspaceValidationStateFromInput returned error: %v", err)
	}

	if !state.Valid.ValueBool() {
		t.Fatalf("expected translated workspace to validate, got %v", state.Errors)
	}
	if state.TranslatedPromptJSON.ValueString() == "" {
		t.Fatal("expected translated_prompt_json to be populated")
	}
	if state.TranslationFidelity.ValueString() == "" {
		t.Fatal("expected translation_fidelity to be populated")
	}
	if state.TranslationPreservedFields.IsNull() || state.TranslationSynthesizedFields.IsNull() ||
		state.TranslationUnsupportedFields.IsNull() || state.TranslationNotes.IsNull() {
		t.Fatal("expected structured translation fields to be populated")
	}
}

func TestPromptValidationStateFromInput_AllowsPartialPromptWithoutOutputNode(t *testing.T) {
	state, err := promptValidationStateFromInput("", `{
	  "1": {
	    "class_type": "LoadImage",
	    "inputs": {
	      "image": "input.png"
	    }
	  }
	}`, validationTestNodeInfo())
	if err != nil {
		t.Fatalf("promptValidationStateFromInput returned error: %v", err)
	}

	if !state.Valid.ValueBool() {
		t.Fatalf("expected partial prompt without output node to remain valid, got %v", state.Errors)
	}
}

func TestPromptValidationDataSourceSchema_ValidatesPathAndJSONSelection(t *testing.T) {
	ds := NewPromptValidationDataSource().(*PromptValidationDataSource)
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	pathAttr, ok := resp.Schema.Attributes["path"].(datasourceschema.StringAttribute)
	if !ok {
		t.Fatalf("expected path to be a string attribute, got %#v", resp.Schema.Attributes["path"])
	}
	if len(pathAttr.Validators) != 2 {
		t.Fatalf("expected path to have 2 validators, got %d", len(pathAttr.Validators))
	}

	jsonAttr, ok := resp.Schema.Attributes["json"].(datasourceschema.StringAttribute)
	if !ok {
		t.Fatalf("expected json to be a string attribute, got %#v", resp.Schema.Attributes["json"])
	}
	if len(jsonAttr.Validators) != 2 {
		t.Fatalf("expected json to have 2 validators, got %d", len(jsonAttr.Validators))
	}
}

func TestWorkspaceValidationDataSourceSchema_ValidatesPathAndJSONSelection(t *testing.T) {
	ds := NewWorkspaceValidationDataSource().(*WorkspaceValidationDataSource)
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	pathAttr, ok := resp.Schema.Attributes["path"].(datasourceschema.StringAttribute)
	if !ok {
		t.Fatalf("expected path to be a string attribute, got %#v", resp.Schema.Attributes["path"])
	}
	if len(pathAttr.Validators) != 2 {
		t.Fatalf("expected path to have 2 validators, got %d", len(pathAttr.Validators))
	}

	jsonAttr, ok := resp.Schema.Attributes["json"].(datasourceschema.StringAttribute)
	if !ok {
		t.Fatalf("expected json to be a string attribute, got %#v", resp.Schema.Attributes["json"])
	}
	if len(jsonAttr.Validators) != 2 {
		t.Fatalf("expected json to have 2 validators, got %d", len(jsonAttr.Validators))
	}
}
