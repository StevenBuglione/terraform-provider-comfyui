package datasources

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestPromptJSONStateFromInput_ParsesRawJSON(t *testing.T) {
	state, err := promptJSONStateFromInput("", `{
	  "1": {
	    "class_type": "CheckpointLoaderSimple",
	    "inputs": {
	      "ckpt_name": "sd_xl_base_1.0.safetensors"
	    }
	  },
	  "2": {
	    "class_type": "SaveImage",
	    "inputs": {}
	  }
	}`)
	if err != nil {
		t.Fatalf("promptJSONStateFromInput returned error: %v", err)
	}

	if state.NodeCount.ValueInt64() != 2 {
		t.Fatalf("expected node_count=2, got %d", state.NodeCount.ValueInt64())
	}
	if state.NormalizedJSON.ValueString() == "" {
		t.Fatal("expected normalized_json to be populated")
	}
}

func TestPromptJSONStateFromInput_LoadsPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.json")
	if err := os.WriteFile(path, []byte(`{
	  "1": {
	    "class_type": "KSampler",
	    "inputs": {}
	  }
	}`), 0644); err != nil {
		t.Fatalf("failed to write test prompt file: %v", err)
	}

	state, err := promptJSONStateFromInput(path, "")
	if err != nil {
		t.Fatalf("promptJSONStateFromInput returned error: %v", err)
	}

	if state.Path.ValueString() != path {
		t.Fatalf("expected path to round-trip, got %q", state.Path.ValueString())
	}
	if state.NodeCount.ValueInt64() != 1 {
		t.Fatalf("expected node_count=1, got %d", state.NodeCount.ValueInt64())
	}
}

func TestPromptJSONStateFromInput_RejectsBothPathAndJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.json")
	if err := os.WriteFile(path, []byte(`{"1":{"class_type":"KSampler","inputs":{}}}`), 0644); err != nil {
		t.Fatalf("failed to write test prompt file: %v", err)
	}

	_, err := promptJSONStateFromInput(path, `{"1":{"class_type":"SaveImage","inputs":{}}}`)
	if err == nil {
		t.Fatal("expected providing both path and json to return an error")
	}
}

func TestPromptJSONDataSourceSchema_ValidatesPathAndJSONSelection(t *testing.T) {
	ds := NewPromptJSONDataSource().(*PromptJSONDataSource)
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
