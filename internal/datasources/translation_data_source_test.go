package datasources

import (
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

func TestPromptToWorkspaceStateFromInput_TranslatesPrompt(t *testing.T) {
	state, err := promptToWorkspaceStateFromInput("translated", `{
	  "1": {
	    "class_type": "CheckpointLoaderSimple",
	    "inputs": {
	      "ckpt_name": "sd_xl_base_1.0.safetensors"
	    }
	  }
	}`, map[string]client.NodeInfo{
		"CheckpointLoaderSimple": {
			Input:      client.NodeInputInfo{Required: map[string]interface{}{"ckpt_name": []interface{}{"COMBO"}}},
			InputOrder: map[string][]string{"required": {"ckpt_name"}},
			Output:     []string{"MODEL"},
			OutputName: []string{"MODEL"},
		},
	})
	if err != nil {
		t.Fatalf("promptToWorkspaceStateFromInput returned error: %v", err)
	}

	if state.Fidelity.ValueString() != "synthetic" {
		t.Fatalf("expected synthetic fidelity, got %q", state.Fidelity.ValueString())
	}
	if state.WorkspaceJSON.ValueString() == "" {
		t.Fatal("expected workspace_json to be populated")
	}
}

func TestWorkspaceToPromptStateFromInput_TranslatesWorkspace(t *testing.T) {
	state, err := workspaceToPromptStateFromInput(`{
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
	      "widgets_values": ["sd_xl_base_1.0.safetensors"],
	      "pos": [0, 0]
	    }
	  ],
	  "links": []
	}`)
	if err != nil {
		t.Fatalf("workspaceToPromptStateFromInput returned error: %v", err)
	}

	if state.Fidelity.ValueString() != "lossy" {
		t.Fatalf("expected lossy fidelity, got %q", state.Fidelity.ValueString())
	}
	if state.PromptJSON.ValueString() == "" {
		t.Fatal("expected prompt_json to be populated")
	}
}
