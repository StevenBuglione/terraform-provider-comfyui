package artifacts

import "testing"

func TestBuildTerraformIRFromPrompt_CreatesNodeResourcesAndWorkflow(t *testing.T) {
	prompt := mustParsePromptForTerraformIRTest(t, `{
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
	}`)

	ir, report, err := BuildTerraformIRFromPrompt(prompt)
	if err != nil {
		t.Fatalf("BuildTerraformIRFromPrompt returned error: %v", err)
	}
	if len(ir.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(ir.Resources))
	}
	if ir.Resources[0].Type != "comfyui_load_image" || ir.Resources[0].Name != "node_1" {
		t.Fatalf("unexpected first resource: %#v", ir.Resources[0])
	}
	if ir.Resources[1].Type != "comfyui_save_image" || ir.Resources[1].Name != "node_2" {
		t.Fatalf("unexpected second resource: %#v", ir.Resources[1])
	}
	if got := ir.Resources[1].Attributes[0].Expression; got != "comfyui_load_image.node_1.image_output" {
		t.Fatalf("expected link input to render as expression, got %#v", ir.Resources[1].Attributes[0])
	}
	if ir.Workflow.Type != "comfyui_workflow" || ir.Workflow.Name != "workflow" {
		t.Fatalf("unexpected workflow block: %#v", ir.Workflow)
	}
	if len(ir.Workflow.NodeIDExpressions) != 2 {
		t.Fatalf("expected 2 workflow node id expressions, got %#v", ir.Workflow.NodeIDExpressions)
	}
	if report.Fidelity() != "synthetic" {
		t.Fatalf("expected synthetic fidelity due to synthesized Terraform labels, got %q", report.Fidelity())
	}
}

func mustParsePromptForTerraformIRTest(t *testing.T, raw string) *Prompt {
	t.Helper()
	prompt, err := ParsePromptJSON(raw)
	if err != nil {
		t.Fatalf("ParsePromptJSON returned error: %v", err)
	}
	return prompt
}
