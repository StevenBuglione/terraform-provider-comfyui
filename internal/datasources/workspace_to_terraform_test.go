package datasources

import "testing"

func TestWorkspaceToTerraformStateFromInput_ReturnsIRAndHCL(t *testing.T) {
	state, err := workspaceToTerraformStateFromInput("", `{
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
	        }
	      ]
	    }
	  ],
	  "links": [
	    [1, 1, 0, 2, 0, "IMAGE"]
	  ]
	}`)
	if err != nil {
		t.Fatalf("workspaceToTerraformStateFromInput returned error: %v", err)
	}
	if state.TerraformHCL.ValueString() == "" {
		t.Fatal("expected terraform_hcl to be populated")
	}
	if state.TranslatedPromptJSON.ValueString() == "" {
		t.Fatal("expected translated prompt json to be populated")
	}
}
