package resources

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSubgraphStateFromInput_NormalizesEditorNativeJSON(t *testing.T) {
	state, err := subgraphStateFromInput("", `{
	  "id": "workspace-1",
	  "config": {},
	  "nodes": [
	    {
	      "id": 1,
	      "type": "subgraph-node",
	      "flags": {},
	      "order": 0,
	      "mode": 0,
	      "title": "Example Subgraph"
	    }
	  ],
	  "links": [],
	  "groups": [],
	  "definitions": {
	    "subgraphs": [
	      {
	        "id": "subgraph-1",
	        "name": "Example Subgraph",
	        "config": {},
	        "widgets": [],
	        "category": "Examples"
	      }
	    ]
	  },
	  "extra": {}
	}`)
	if err != nil {
		t.Fatalf("subgraphStateFromInput returned error: %v", err)
	}

	if state.WorkspaceID.ValueString() != "workspace-1" {
		t.Fatalf("expected workspace_id to round-trip, got %q", state.WorkspaceID.ValueString())
	}
	if state.DefinitionCount.ValueInt64() != 1 {
		t.Fatalf("expected definition_count=1, got %d", state.DefinitionCount.ValueInt64())
	}
	if !strings.Contains(state.NormalizedJSON.ValueString(), `"category": "Examples"`) {
		t.Fatalf("expected normalized JSON to preserve category, got %s", state.NormalizedJSON.ValueString())
	}
}

func TestSubgraphResource_MaterializeRefreshDeleteAndImportLocalFile(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old", "subgraph.json")
	newPath := filepath.Join(dir, "new", "subgraph.json")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0o755); err != nil {
		t.Fatalf("failed to create old path directory: %v", err)
	}
	if err := os.WriteFile(oldPath, []byte(`{"id":"old-workspace","nodes":[],"links":[],"groups":[]}`), 0o644); err != nil {
		t.Fatalf("failed to seed old subgraph file: %v", err)
	}

	r := &SubgraphResource{}
	state := SubgraphResourceModel{
		Path: types.StringValue(newPath),
		JSON: types.StringValue(`{
		  "id": "workspace-2",
		  "config": {},
		  "nodes": [
		    {
		      "id": 1,
		      "type": "subgraph-node",
		      "title": "Materialized Subgraph"
		    }
		  ],
		  "links": [],
		  "groups": [],
		  "definitions": {
		    "subgraphs": [
		      {
		        "id": "subgraph-2",
		        "name": "Materialized Subgraph",
		        "config": {},
		        "widgets": []
		      }
		    ]
		  },
		  "extra": {}
		}`),
	}

	if err := r.materialize(&state, oldPath); err != nil {
		t.Fatalf("materialize returned error: %v", err)
	}
	if state.SHA256.ValueString() == "" {
		t.Fatal("expected sha256 to be populated after materialize")
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected new subgraph file to exist, got %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old subgraph path to be cleaned up, got %v", err)
	}

	refreshed, err := r.refresh(&state)
	if err != nil {
		t.Fatalf("refresh returned error: %v", err)
	}
	if !refreshed {
		t.Fatal("expected refresh to keep resource present")
	}

	imported, err := subgraphStateFromFile(newPath)
	if err != nil {
		t.Fatalf("subgraphStateFromFile returned error: %v", err)
	}
	if imported.Path.ValueString() != newPath {
		t.Fatalf("expected import path %q, got %q", newPath, imported.Path.ValueString())
	}
	if imported.DefinitionCount.ValueInt64() != 1 {
		t.Fatalf("expected imported definition_count=1, got %d", imported.DefinitionCount.ValueInt64())
	}

	if err := r.remove(newPath); err != nil {
		t.Fatalf("remove returned error: %v", err)
	}
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Fatalf("expected subgraph file to be deleted, got %v", err)
	}
}
