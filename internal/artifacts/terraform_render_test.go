package artifacts

import (
	"strings"
	"testing"
)

func TestRenderTerraformHCL_RendersDeterministicWorkflow(t *testing.T) {
	ir := &TerraformIR{
		Resources: []TerraformResource{
			{
				Type: "comfyui_load_image",
				Name: "node_1",
				Attributes: []TerraformAttribute{
					{Name: "image", Literal: "input.png"},
				},
			},
			{
				Type: "comfyui_save_image",
				Name: "node_2",
				Attributes: []TerraformAttribute{
					{Name: "images", Expression: "comfyui_load_image.node_1.image_output"},
					{Name: "filename_prefix", Literal: "ComfyUI"},
				},
			},
		},
		Workflow: TerraformWorkflow{
			Type:              "comfyui_workflow",
			Name:              "workflow",
			NodeIDExpressions: []string{"comfyui_load_image.node_1.id", "comfyui_save_image.node_2.id"},
		},
	}

	hcl, err := RenderTerraformHCL(ir)
	if err != nil {
		t.Fatalf("RenderTerraformHCL returned error: %v", err)
	}
	for _, want := range []string{
		`resource "comfyui_load_image" "node_1"`,
		`image = "input.png"`,
		`resource "comfyui_save_image" "node_2"`,
		`images = comfyui_load_image.node_1.image_output`,
		`resource "comfyui_workflow" "workflow"`,
		`node_ids = [`,
	} {
		if !strings.Contains(hcl, want) {
			t.Fatalf("expected HCL to contain %q, got:\n%s", want, hcl)
		}
	}
}
