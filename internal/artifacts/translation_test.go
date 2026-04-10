package artifacts

import (
	"strings"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

func TestTranslationReportFidelityTransitions(t *testing.T) {
	report := NewTranslationReport()

	if got := report.Fidelity(); got != "lossless" {
		t.Fatalf("expected empty report fidelity to be lossless, got %q", got)
	}

	report.AddPreservedField("prompt.1.class_type")
	if got := report.Fidelity(); got != "lossless" {
		t.Fatalf("expected preserved-only report fidelity to stay lossless, got %q", got)
	}

	report.AddSynthesizedField("workspace.nodes[0].pos")
	if got := report.Fidelity(); got != "synthetic" {
		t.Fatalf("expected synthesized report fidelity to be synthetic, got %q", got)
	}

	report.AddUnsupportedField("workspace.definitions")
	if got := report.Fidelity(); got != "lossy" {
		t.Fatalf("expected unsupported field to make fidelity lossy, got %q", got)
	}
}

func TestTranslateWorkspaceToPrompt_ConvertsWidgetsAndLinks(t *testing.T) {
	workspace, err := ParseWorkspaceJSON(`{
	  "nodes": [
	    {
	      "id": 1,
	      "type": "CheckpointLoaderSimple",
	      "inputs": [
	        {
	          "name": "ckpt_name",
	          "type": "COMBO",
	          "widget": {"name": "ckpt_name"},
	          "link": null
	        }
	      ],
	      "outputs": [
	        {
	          "name": "MODEL",
	          "type": "MODEL",
	          "links": [10]
	        }
	      ],
	      "widgets_values": ["sd_xl_base_1.0.safetensors"],
	      "pos": [0, 0]
	    },
	    {
	      "id": 2,
	      "type": "KSampler",
	      "inputs": [
	        {
	          "name": "model",
	          "type": "MODEL",
	          "link": 10
	        },
	        {
	          "name": "seed",
	          "type": "INT",
	          "widget": {"name": "seed"},
	          "link": null
	        }
	      ],
	      "outputs": [],
	      "widgets_values": [123],
	      "pos": [300, 0]
	    }
	  ],
	  "links": [
	    {"id": 10, "origin_id": 1, "origin_slot": 0, "target_id": 2, "target_slot": 0, "type": "MODEL"}
	  ]
	}`)
	if err != nil {
		t.Fatalf("ParseWorkspaceJSON returned error: %v", err)
	}

	prompt, report, err := TranslateWorkspaceToPrompt(workspace)
	if err != nil {
		t.Fatalf("TranslateWorkspaceToPrompt returned error: %v", err)
	}

	if len(prompt.Nodes) != 2 {
		t.Fatalf("expected 2 prompt nodes, got %d", len(prompt.Nodes))
	}

	loader := prompt.Nodes["1"]
	if loader.Inputs["ckpt_name"] != "sd_xl_base_1.0.safetensors" {
		t.Fatalf("expected ckpt_name widget value to round-trip, got %#v", loader.Inputs["ckpt_name"])
	}

	sampler := prompt.Nodes["2"]
	modelRef, ok := sampler.Inputs["model"].([]interface{})
	if !ok || len(modelRef) != 2 || modelRef[0] != "1" || modelRef[1] != 0 {
		t.Fatalf("expected model link to become [\"1\", 0], got %#v", sampler.Inputs["model"])
	}
	if sampler.Inputs["seed"] != float64(123) {
		t.Fatalf("expected seed widget value to round-trip, got %#v", sampler.Inputs["seed"])
	}
	if report.Fidelity() != "lossy" {
		t.Fatalf("expected workspace->prompt fidelity to be lossy because editor layout is dropped, got %q", report.Fidelity())
	}
}

func TestTranslatePromptToWorkspace_CreatesSyntheticLayout(t *testing.T) {
	prompt, err := ParsePromptJSON(`{
	  "1": {
	    "class_type": "CheckpointLoaderSimple",
	    "inputs": {
	      "ckpt_name": "sd_xl_base_1.0.safetensors"
	    }
	  },
	  "2": {
	    "class_type": "KSampler",
	    "inputs": {
	      "model": ["1", 0],
	      "seed": 123
	    }
	  }
	}`)
	if err != nil {
		t.Fatalf("ParsePromptJSON returned error: %v", err)
	}

	workspace, report, err := TranslatePromptToWorkspace("translated", prompt, map[string]client.NodeInfo{
		"CheckpointLoaderSimple": {
			Input:      client.NodeInputInfo{Required: map[string]interface{}{"ckpt_name": []interface{}{"COMBO"}}},
			InputOrder: map[string][]string{"required": {"ckpt_name"}},
			Output:     []string{"MODEL"},
			OutputName: []string{"MODEL"},
		},
		"KSampler": {
			Input: client.NodeInputInfo{Required: map[string]interface{}{
				"model": []interface{}{"MODEL"},
				"seed":  []interface{}{"INT"},
			}},
			InputOrder: map[string][]string{"required": {"model", "seed"}},
			Output:     []string{"LATENT"},
			OutputName: []string{"LATENT"},
		},
	})
	if err != nil {
		t.Fatalf("TranslatePromptToWorkspace returned error: %v", err)
	}

	if workspace.Name != "translated" {
		t.Fatalf("expected translated workspace name, got %q", workspace.Name)
	}
	if len(workspace.Nodes) != 2 {
		t.Fatalf("expected 2 workspace nodes, got %d", len(workspace.Nodes))
	}
	if len(workspace.Links) != 1 {
		t.Fatalf("expected 1 workspace link, got %d", len(workspace.Links))
	}
	if report.Fidelity() != "synthetic" {
		t.Fatalf("expected prompt->workspace fidelity to be synthetic, got %q", report.Fidelity())
	}
}

func TestTranslatePromptToWorkspace_PreservesNonAlphabeticalInputOrderAndTargetSlots(t *testing.T) {
	prompt, err := ParsePromptJSON(`{
	  "1": {
	    "class_type": "CheckpointLoaderSimple",
	    "inputs": {
	      "ckpt_name": "sd_xl_base_1.0.safetensors"
	    }
	  },
	  "2": {
	    "class_type": "CLIPTextEncode",
	    "inputs": {
	      "clip": ["1", 0],
	      "text": "positive"
	    }
	  },
	  "3": {
	    "class_type": "CLIPTextEncode",
	    "inputs": {
	      "clip": ["1", 0],
	      "text": "negative"
	    }
	  },
	  "4": {
	    "class_type": "KSampler",
	    "inputs": {
	      "model": ["1", 0],
	      "positive": ["2", 0],
	      "negative": ["3", 0],
	      "latent_image": "latent-placeholder",
	      "seed": 123
	    }
	  }
	}`)
	if err != nil {
		t.Fatalf("ParsePromptJSON returned error: %v", err)
	}

	workspace, _, err := TranslatePromptToWorkspace("translated", prompt, map[string]client.NodeInfo{
		"CheckpointLoaderSimple": {
			Input:      client.NodeInputInfo{Required: map[string]interface{}{"ckpt_name": []interface{}{"COMBO"}}},
			InputOrder: map[string][]string{"required": {"ckpt_name"}},
			Output:     []string{"MODEL"},
			OutputName: []string{"MODEL"},
		},
		"CLIPTextEncode": {
			Input: client.NodeInputInfo{Required: map[string]interface{}{
				"clip": []interface{}{"CLIP"},
				"text": []interface{}{"STRING"},
			}},
			InputOrder: map[string][]string{"required": {"clip", "text"}},
			Output:     []string{"CONDITIONING"},
			OutputName: []string{"CONDITIONING"},
		},
		"KSampler": {
			Input: client.NodeInputInfo{Required: map[string]interface{}{
				"model":        []interface{}{"MODEL"},
				"positive":     []interface{}{"CONDITIONING"},
				"negative":     []interface{}{"CONDITIONING"},
				"latent_image": []interface{}{"LATENT"},
				"seed":         []interface{}{"INT"},
			}},
			InputOrder: map[string][]string{"required": {"model", "positive", "negative", "latent_image", "seed"}},
			Output:     []string{"LATENT"},
			OutputName: []string{"LATENT"},
		},
	})
	if err != nil {
		t.Fatalf("TranslatePromptToWorkspace returned error: %v", err)
	}

	var sampler WorkspaceNode
	for _, node := range workspace.Nodes {
		if node.Type == "KSampler" {
			sampler = node
			break
		}
	}

	if len(sampler.Inputs) < 5 {
		t.Fatalf("expected KSampler inputs to preserve schema order, got %#v", sampler.Inputs)
	}
	if sampler.Inputs[0].Name != "model" || sampler.Inputs[1].Name != "positive" || sampler.Inputs[2].Name != "negative" {
		t.Fatalf("expected KSampler inputs to preserve schema order, got %#v", sampler.Inputs)
	}

	targetSlots := map[int]int{}
	for _, link := range workspace.Links {
		if link.TargetID == sampler.ID {
			targetSlots[link.OriginID] = link.TargetSlot
		}
	}
	if targetSlots[1] != 0 || targetSlots[2] != 1 || targetSlots[3] != 2 {
		t.Fatalf("expected model/positive/negative target slots 0/1/2, got %#v", targetSlots)
	}
}

func TestTranslatePromptToWorkspace_PreservesSchemaSlotsForSparseInputs(t *testing.T) {
	prompt, err := ParsePromptJSON(`{
	  "1": {
	    "class_type": "CheckpointLoaderSimple",
	    "inputs": {
	      "ckpt_name": "sd_xl_base_1.0.safetensors"
	    }
	  },
	  "2": {
	    "class_type": "CLIPTextEncode",
	    "inputs": {
	      "clip": ["1", 0],
	      "text": "negative"
	    }
	  },
	  "3": {
	    "class_type": "KSampler",
	    "inputs": {
	      "model": ["1", 0],
	      "negative": ["2", 0],
	      "seed": 123
	    }
	  }
	}`)
	if err != nil {
		t.Fatalf("ParsePromptJSON returned error: %v", err)
	}

	workspace, _, err := TranslatePromptToWorkspace("translated", prompt, map[string]client.NodeInfo{
		"CheckpointLoaderSimple": {
			Input:      client.NodeInputInfo{Required: map[string]interface{}{"ckpt_name": []interface{}{"COMBO"}}},
			InputOrder: map[string][]string{"required": {"ckpt_name"}},
			Output:     []string{"MODEL"},
			OutputName: []string{"MODEL"},
		},
		"CLIPTextEncode": {
			Input: client.NodeInputInfo{Required: map[string]interface{}{
				"clip": []interface{}{"CLIP"},
				"text": []interface{}{"STRING"},
			}},
			InputOrder: map[string][]string{"required": {"clip", "text"}},
			Output:     []string{"CONDITIONING"},
			OutputName: []string{"CONDITIONING"},
		},
		"KSampler": {
			Input: client.NodeInputInfo{Required: map[string]interface{}{
				"model":        []interface{}{"MODEL"},
				"positive":     []interface{}{"CONDITIONING"},
				"negative":     []interface{}{"CONDITIONING"},
				"latent_image": []interface{}{"LATENT"},
				"seed":         []interface{}{"INT"},
			}},
			InputOrder: map[string][]string{"required": {"model", "positive", "negative", "latent_image", "seed"}},
			Output:     []string{"LATENT"},
			OutputName: []string{"LATENT"},
		},
	})
	if err != nil {
		t.Fatalf("TranslatePromptToWorkspace returned error: %v", err)
	}

	var sampler WorkspaceNode
	for _, node := range workspace.Nodes {
		if node.Type == "KSampler" {
			sampler = node
			break
		}
	}

	if len(sampler.Inputs) != 3 {
		t.Fatalf("expected sparse KSampler inputs to stay sparse, got %#v", sampler.Inputs)
	}
	if sampler.Inputs[0].Name != "model" || sampler.Inputs[1].Name != "negative" || sampler.Inputs[2].Name != "seed" {
		t.Fatalf("expected sparse KSampler inputs to preserve prompt input order, got %#v", sampler.Inputs)
	}

	targetSlots := map[int]int{}
	for _, link := range workspace.Links {
		if link.TargetID == sampler.ID {
			targetSlots[link.OriginID] = link.TargetSlot
		}
	}
	if targetSlots[1] != 0 || targetSlots[2] != 2 {
		t.Fatalf("expected model/negative target slots 0/2, got %#v", targetSlots)
	}
}

func TestPromptWorkspacePrompt_RoundTripsSparseInputs(t *testing.T) {
	original, err := ParsePromptJSON(`{
	  "1": {
	    "class_type": "CheckpointLoaderSimple",
	    "inputs": {
	      "ckpt_name": "sd_xl_base_1.0.safetensors"
	    }
	  },
	  "2": {
	    "class_type": "CLIPTextEncode",
	    "inputs": {
	      "clip": ["1", 0],
	      "text": "negative"
	    }
	  },
	  "3": {
	    "class_type": "KSampler",
	    "inputs": {
	      "model": ["1", 0],
	      "negative": ["2", 0],
	      "seed": 123
	    }
	  }
	}`)
	if err != nil {
		t.Fatalf("ParsePromptJSON returned error: %v", err)
	}

	nodeInfo := map[string]client.NodeInfo{
		"CheckpointLoaderSimple": {
			Input:      client.NodeInputInfo{Required: map[string]interface{}{"ckpt_name": []interface{}{"COMBO"}}},
			InputOrder: map[string][]string{"required": {"ckpt_name"}},
			Output:     []string{"MODEL"},
			OutputName: []string{"MODEL"},
		},
		"CLIPTextEncode": {
			Input: client.NodeInputInfo{Required: map[string]interface{}{
				"clip": []interface{}{"CLIP"},
				"text": []interface{}{"STRING"},
			}},
			InputOrder: map[string][]string{"required": {"clip", "text"}},
			Output:     []string{"CONDITIONING"},
			OutputName: []string{"CONDITIONING"},
		},
		"KSampler": {
			Input: client.NodeInputInfo{Required: map[string]interface{}{
				"model":        []interface{}{"MODEL"},
				"positive":     []interface{}{"CONDITIONING"},
				"negative":     []interface{}{"CONDITIONING"},
				"latent_image": []interface{}{"LATENT"},
				"seed":         []interface{}{"INT"},
			}},
			InputOrder: map[string][]string{"required": {"model", "positive", "negative", "latent_image", "seed"}},
			Output:     []string{"LATENT"},
			OutputName: []string{"LATENT"},
		},
	}

	workspace, _, err := TranslatePromptToWorkspace("translated", original, nodeInfo)
	if err != nil {
		t.Fatalf("TranslatePromptToWorkspace returned error: %v", err)
	}

	roundTripped, _, err := TranslateWorkspaceToPrompt(workspace)
	if err != nil {
		t.Fatalf("TranslateWorkspaceToPrompt returned error: %v", err)
	}

	if roundTripped.Nodes["3"].Inputs["model"].([]interface{})[0] != "1" {
		t.Fatalf("expected model link to round-trip, got %#v", roundTripped.Nodes["3"].Inputs["model"])
	}
	if roundTripped.Nodes["3"].Inputs["negative"].([]interface{})[0] != "2" {
		t.Fatalf("expected negative link to round-trip, got %#v", roundTripped.Nodes["3"].Inputs["negative"])
	}
	if roundTripped.Nodes["3"].Inputs["seed"] != float64(123) {
		t.Fatalf("expected seed widget to round-trip, got %#v", roundTripped.Nodes["3"].Inputs["seed"])
	}
	if _, ok := roundTripped.Nodes["3"].Inputs["positive"]; ok {
		t.Fatalf("expected absent sparse input positive to remain absent, got %#v", roundTripped.Nodes["3"].Inputs)
	}
}

func contains(value string, needle string) bool {
	return strings.Contains(value, needle)
}
