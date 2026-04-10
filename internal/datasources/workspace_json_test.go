package datasources

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestWorkspaceJSONStateFromInput_ParsesRawJSON(t *testing.T) {
	state, err := workspaceJSONStateFromInput("", `{
	  "id": "workspace-1",
	  "nodes": [
	    {
	      "id": 1,
	      "type": "KSampler"
	    }
	  ],
	  "groups": [
	    {
	      "id": 3,
	      "title": "Sampler"
	    }
	  ],
	  "definitions": {
	    "subgraphs": [
	      {
	        "id": "subgraph-1"
	      }
	    ]
	  }
	}`)
	if err != nil {
		t.Fatalf("workspaceJSONStateFromInput returned error: %v", err)
	}

	if state.NodeCount.ValueInt64() != 1 {
		t.Fatalf("expected node_count=1, got %d", state.NodeCount.ValueInt64())
	}
	if state.GroupCount.ValueInt64() != 1 {
		t.Fatalf("expected group_count=1, got %d", state.GroupCount.ValueInt64())
	}
	if state.SubgraphCount.ValueInt64() != 1 {
		t.Fatalf("expected subgraph_count=1, got %d", state.SubgraphCount.ValueInt64())
	}
}

func TestWorkspaceJSONStateFromInput_LoadsPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.json")
	if err := os.WriteFile(path, []byte(`{
	  "id": "workspace-2",
	  "nodes": [],
	  "groups": []
	}`), 0644); err != nil {
		t.Fatalf("failed to write test workspace file: %v", err)
	}

	state, err := workspaceJSONStateFromInput(path, "")
	if err != nil {
		t.Fatalf("workspaceJSONStateFromInput returned error: %v", err)
	}

	if state.Path.ValueString() != path {
		t.Fatalf("expected path to round-trip, got %q", state.Path.ValueString())
	}
	if state.NormalizedJSON.ValueString() == "" {
		t.Fatal("expected normalized_json to be populated")
	}
}

func TestWorkspaceJSONStateFromInput_RejectsBothPathAndJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.json")
	if err := os.WriteFile(path, []byte(`{"id":"workspace-1","nodes":[],"groups":[]}`), 0644); err != nil {
		t.Fatalf("failed to write test workspace file: %v", err)
	}

	_, err := workspaceJSONStateFromInput(path, `{"id":"workspace-2","nodes":[],"groups":[]}`)
	if err == nil {
		t.Fatal("expected providing both path and json to return an error")
	}
}

func TestWorkspaceJSONDataSourceSchema_ValidatesPathAndJSONSelection(t *testing.T) {
	ds := NewWorkspaceJSONDataSource().(*WorkspaceJSONDataSource)
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
