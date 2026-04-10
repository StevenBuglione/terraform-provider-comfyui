package artifacts

import "testing"

func TestParsePromptJSON_ParsesNodesAndMeta(t *testing.T) {
	raw := `{
	  "1": {
	    "class_type": "CheckpointLoaderSimple",
	    "inputs": {
	      "ckpt_name": "sd_xl_base_1.0.safetensors"
	    },
	    "_meta": {
	      "title": "Base Model"
	    }
	  },
	  "2": {
	    "class_type": "SaveImage",
	    "inputs": {
	      "filename_prefix": "roundtrip-test",
	      "images": ["1", 0]
	    }
	  }
	}`

	prompt, err := ParsePromptJSON(raw)
	if err != nil {
		t.Fatalf("ParsePromptJSON returned error: %v", err)
	}

	if len(prompt.Nodes) != 2 {
		t.Fatalf("expected 2 prompt nodes, got %d", len(prompt.Nodes))
	}

	node1, ok := prompt.Nodes["1"]
	if !ok {
		t.Fatal(`expected node "1" to be present`)
	}

	if node1.ClassType != "CheckpointLoaderSimple" {
		t.Fatalf("expected class_type CheckpointLoaderSimple, got %q", node1.ClassType)
	}

	if got := node1.Inputs["ckpt_name"]; got != "sd_xl_base_1.0.safetensors" {
		t.Fatalf("expected ckpt_name to round-trip, got %#v", got)
	}

	if node1.Meta["title"] != "Base Model" {
		t.Fatalf("expected _meta.title to round-trip, got %#v", node1.Meta["title"])
	}
}
