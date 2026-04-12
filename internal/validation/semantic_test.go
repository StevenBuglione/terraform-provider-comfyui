package validation

import (
	"strings"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/artifacts"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

func testNodeInfo() map[string]client.NodeInfo {
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
		"LatentSource": {
			Output:     []string{"LATENT"},
			OutputName: []string{"LATENT"},
		},
		"FlexibleConsumer": {
			Input: client.NodeInputInfo{
				Required: map[string]interface{}{
					"value": []interface{}{"*"},
				},
			},
			InputOrder: map[string][]string{
				"required": {"value"},
			},
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

func mustParsePrompt(t *testing.T, raw string) *artifacts.Prompt {
	t.Helper()
	prompt, err := artifacts.ParsePromptJSON(raw)
	if err != nil {
		t.Fatalf("ParsePromptJSON returned error: %v", err)
	}
	return prompt
}

func requireErrorContaining(t *testing.T, report Report, want string) {
	t.Helper()
	for _, err := range report.Errors {
		if strings.Contains(err, want) {
			return
		}
	}
	t.Fatalf("expected error containing %q, got %v", want, report.Errors)
}

func TestValidatePrompt_AllowsValidPromptAndHiddenInputs(t *testing.T) {
	report := ValidatePrompt(
		mustParsePrompt(t, `{
		  "1": {
		    "class_type": "LoadImage",
		    "inputs": {
		      "image": "input.png"
		    }
		  },
		  "2": {
		    "class_type": "SaveImage",
		    "inputs": {
		      "images": ["1", 0],
		      "filename_prefix": "ComfyUI"
		    }
		  }
		}`),
		testNodeInfo(),
		Options{RequireOutputNode: true},
	)

	if !report.Valid {
		t.Fatalf("expected prompt to be valid, got errors %v", report.Errors)
	}
	if report.ErrorCount != 0 || report.WarningCount != 0 {
		t.Fatalf("expected zero findings, got %#v", report)
	}
	if report.ValidatedNodeCount != 2 {
		t.Fatalf("expected validated_node_count=2, got %d", report.ValidatedNodeCount)
	}
}

func TestValidatePrompt_RejectsMissingNodeType(t *testing.T) {
	report := ValidatePrompt(
		mustParsePrompt(t, `{
		  "1": {
		    "class_type": "DoesNotExist",
		    "inputs": {}
		  }
		}`),
		testNodeInfo(),
		Options{},
	)

	requireErrorContaining(t, report, `node "1" uses unknown class_type "DoesNotExist"`)
}

func TestValidatePrompt_RejectsMissingRequiredInput(t *testing.T) {
	report := ValidatePrompt(
		mustParsePrompt(t, `{
		  "1": {
		    "class_type": "SaveImage",
		    "inputs": {
		      "filename_prefix": "ComfyUI"
		    }
		  }
		}`),
		testNodeInfo(),
		Options{},
	)

	requireErrorContaining(t, report, `node "1" (SaveImage) is missing required input "images"`)
}

func TestValidatePrompt_RejectsUnknownInputName(t *testing.T) {
	report := ValidatePrompt(
		mustParsePrompt(t, `{
		  "1": {
		    "class_type": "SaveImage",
		    "inputs": {
		      "images": ["2", 0],
		      "bogus": true
		    }
		  },
		  "2": {
		    "class_type": "LoadImage",
		    "inputs": {
		      "image": "input.png"
		    }
		  }
		}`),
		testNodeInfo(),
		Options{},
	)

	requireErrorContaining(t, report, `node "1" (SaveImage) uses unexpected input "bogus"`)
}

func TestValidatePrompt_RejectsMissingLinkedSourceNode(t *testing.T) {
	report := ValidatePrompt(
		mustParsePrompt(t, `{
		  "1": {
		    "class_type": "SaveImage",
		    "inputs": {
		      "images": ["99", 0]
		    }
		  }
		}`),
		testNodeInfo(),
		Options{},
	)

	requireErrorContaining(t, report, `node "1" (SaveImage) input "images" references missing source node "99"`)
}

func TestValidatePrompt_RejectsBadOutputSlot(t *testing.T) {
	report := ValidatePrompt(
		mustParsePrompt(t, `{
		  "1": {
		    "class_type": "LoadImage",
		    "inputs": {
		      "image": "input.png"
		    }
		  },
		  "2": {
		    "class_type": "SaveImage",
		    "inputs": {
		      "images": ["1", 4]
		    }
		  }
		}`),
		testNodeInfo(),
		Options{},
	)

	requireErrorContaining(t, report, `node "2" (SaveImage) input "images" references output slot 4 on node "1" (LoadImage), but only 1 outputs exist`)
}

func TestValidatePrompt_RejectsTypeMismatch(t *testing.T) {
	report := ValidatePrompt(
		mustParsePrompt(t, `{
		  "1": {
		    "class_type": "LatentSource",
		    "inputs": {}
		  },
		  "2": {
		    "class_type": "SaveImage",
		    "inputs": {
		      "images": ["1", 0]
		    }
		  }
		}`),
		testNodeInfo(),
		Options{},
	)

	requireErrorContaining(t, report, `node "2" (SaveImage) input "images" expects type "IMAGE" but linked output is "LATENT"`)
}

func TestValidatePrompt_AllowsWildcardTypeCompatibility(t *testing.T) {
	report := ValidatePrompt(
		mustParsePrompt(t, `{
		  "1": {
		    "class_type": "LoadImage",
		    "inputs": {
		      "image": "input.png"
		    }
		  },
		  "2": {
		    "class_type": "FlexibleConsumer",
		    "inputs": {
		      "value": ["1", 0]
		    }
		  }
		}`),
		testNodeInfo(),
		Options{},
	)

	if !report.Valid {
		t.Fatalf("expected wildcard input type to be compatible, got %v", report.Errors)
	}
}

func TestValidatePrompt_RequiresNativeOutputNodeWhenRequested(t *testing.T) {
	report := ValidatePrompt(
		mustParsePrompt(t, `{
		  "1": {
		    "class_type": "LoadImage",
		    "inputs": {
		      "image": "input.png"
		    }
		  }
		}`),
		testNodeInfo(),
		Options{RequireOutputNode: true},
	)

	requireErrorContaining(t, report, "prompt does not include any node marked output_node=true")
}

func TestValidatePrompt_ExecutableModeRequiresOutputNode(t *testing.T) {
	report := ValidatePrompt(
		mustParsePrompt(t, `{
		  "1": {
		    "class_type": "LoadImage",
		    "inputs": {
		      "image": "input.png"
		    }
		  }
		}`),
		testNodeInfo(),
		Options{Mode: ValidationModeExecutableWorkflow},
	)

	requireErrorContaining(t, report, "prompt does not include any node marked output_node=true")
}

func TestValidatePrompt_FragmentModeAllowsMissingOutputNode(t *testing.T) {
	report := ValidatePrompt(
		mustParsePrompt(t, `{
		  "1": {
		    "class_type": "LoadImage",
		    "inputs": {
		      "image": "input.png"
		    }
		  }
		}`),
		testNodeInfo(),
		Options{Mode: ValidationModeFragment},
	)

	if !report.Valid {
		t.Fatalf("expected fragment validation to allow missing output node, got %v", report.Errors)
	}
}
