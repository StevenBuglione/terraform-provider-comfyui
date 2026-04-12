package datasources

import "testing"

func TestPromptToTerraformStateFromInput_ReturnsIRAndHCL(t *testing.T) {
	state, err := promptToTerraformStateFromInput("", `{
	  "1": {
	    "class_type": "LoadImage",
	    "inputs": {
	      "image": "input.png"
	    }
	  },
	  "2": {
	    "class_type": "SaveImage",
	    "inputs": {
	      "images": ["1", 0]
	    }
	  }
	}`)
	if err != nil {
		t.Fatalf("promptToTerraformStateFromInput returned error: %v", err)
	}
	if state.TerraformHCL.ValueString() == "" {
		t.Fatal("expected terraform_hcl to be populated")
	}
	if state.TerraformIRJSON.ValueString() == "" {
		t.Fatal("expected terraform_ir_json to be populated")
	}
	if state.Fidelity.ValueString() == "" {
		t.Fatal("expected fidelity to be populated")
	}
}
