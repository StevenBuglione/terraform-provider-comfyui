package artifacts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseWorkspaceJSON_PreservesEditorFields(t *testing.T) {
	raw := `{
	  "id": "workspace-1",
	  "revision": 2,
	  "last_node_id": 7,
	  "last_link_id": 11,
	  "nodes": [
	    {
	      "id": 1,
	      "type": "KSampler",
	      "pos": [10, 20],
	      "size": [240, 262],
	      "widgets_values": [123, "randomize", 20]
	    }
	  ],
	  "links": [
	    [11, 1, 0, 2, 0, "LATENT"]
	  ],
	  "groups": [
	    {
	      "id": 3,
	      "title": "Sampler",
	      "bounding": [0, 0, 400, 300],
	      "color": "#3f789e",
	      "font_size": 24
	    }
	  ],
	  "definitions": {
	    "subgraphs": [
	      {
	        "id": "subgraph-1"
	      }
	    ]
	  },
	  "extra": {
	    "frontendVersion": "1.37.10"
	  },
	  "version": 0.4
	}`

	workspace, err := ParseWorkspaceJSON(raw)
	if err != nil {
		t.Fatalf("ParseWorkspaceJSON returned error: %v", err)
	}

	if workspace.ID != "workspace-1" {
		t.Fatalf("expected id workspace-1, got %q", workspace.ID)
	}

	if workspace.Revision != 2 {
		t.Fatalf("expected revision 2, got %d", workspace.Revision)
	}

	if workspace.LastNodeID != 7 {
		t.Fatalf("expected last_node_id 7, got %d", workspace.LastNodeID)
	}

	if len(workspace.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(workspace.Nodes))
	}

	if workspace.Nodes[0].Type != "KSampler" {
		t.Fatalf("expected node type KSampler, got %q", workspace.Nodes[0].Type)
	}

	if len(workspace.Groups) != 1 || workspace.Groups[0].Title != "Sampler" {
		t.Fatalf("expected group title Sampler, got %#v", workspace.Groups)
	}

	if workspace.Extra["frontendVersion"] != "1.37.10" {
		t.Fatalf("expected extra.frontendVersion to round-trip, got %#v", workspace.Extra["frontendVersion"])
	}

	if len(workspace.Definitions.Subgraphs) != 1 || workspace.Definitions.Subgraphs[0].ID != "subgraph-1" {
		t.Fatalf("expected definitions.subgraphs to round-trip, got %#v", workspace.Definitions.Subgraphs)
	}
}

func TestWorkspaceJSON_MarshalsLinksInNativeArrayFormat(t *testing.T) {
	workspace, err := ParseWorkspaceJSON(`{
	  "nodes": [],
	  "links": [
	    [11, 1, 0, 2, 0, "LATENT"]
	  ]
	}`)
	if err != nil {
		t.Fatalf("ParseWorkspaceJSON returned error: %v", err)
	}

	raw, err := workspace.JSON()
	if err != nil {
		t.Fatalf("workspace.JSON returned error: %v", err)
	}

	if contains(raw, `"origin_id"`) {
		t.Fatalf("expected workspace JSON to preserve native array-style links, got %s", raw)
	}
	if !contains(raw, `"LATENT"`) {
		t.Fatalf("expected workspace JSON to preserve native array-style link payload, got %s", raw)
	}
}

func TestWorkspaceJSON_DoesNotInjectEmptyDefinitions(t *testing.T) {
	workspace, err := ParseWorkspaceJSON(`{
	  "id": "workspace-plain",
	  "nodes": [],
	  "groups": []
	}`)
	if err != nil {
		t.Fatalf("ParseWorkspaceJSON returned error: %v", err)
	}

	raw, err := workspace.JSON()
	if err != nil {
		t.Fatalf("workspace.JSON returned error: %v", err)
	}

	if contains(raw, `"definitions"`) {
		t.Fatalf("expected workspace JSON to omit empty definitions, got %s", raw)
	}
}

func TestParseWorkspaceJSON_InitializesExtraMapWhenOmitted(t *testing.T) {
	workspace, err := ParseWorkspaceJSON(`{
	  "id": "workspace-plain",
	  "nodes": [],
	  "groups": []
	}`)
	if err != nil {
		t.Fatalf("ParseWorkspaceJSON returned error: %v", err)
	}

	if workspace.Extra == nil {
		t.Fatal("expected Extra to be initialized even when omitted from input")
	}
}

func TestParseWorkspaceJSON_RoundTripsUpstreamBlueprintFields(t *testing.T) {
	fixturePath := filepath.Join("..", "testdata", "blueprints", "brightness-and-contrast.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read blueprint fixture: %v", err)
	}

	workspace, err := ParseWorkspaceJSON(string(raw))
	if err != nil {
		t.Fatalf("ParseWorkspaceJSON returned error: %v", err)
	}

	if len(workspace.Nodes) != 1 {
		t.Fatalf("expected 1 top-level node, got %d", len(workspace.Nodes))
	}
	if workspace.Nodes[0].Title != "Brightness and Contrast" {
		t.Fatalf("expected top-level node title to round-trip, got %q", workspace.Nodes[0].Title)
	}
	if len(workspace.Definitions.Subgraphs) != 1 {
		t.Fatalf("expected 1 subgraph definition, got %d", len(workspace.Definitions.Subgraphs))
	}

	normalized, err := workspace.JSON()
	if err != nil {
		t.Fatalf("workspace.JSON returned error: %v", err)
	}

	for _, needle := range []string{
		`"flags": {}`,
		`"order": 2`,
		`"mode": 0`,
		`"title": "Brightness and Contrast"`,
		`"localized_name": "images.image0"`,
		`"shape": 7`,
		`"state": {`,
		`"config": {}`,
		`"inputNode": {`,
		`"outputNode": {`,
		`"widgets": []`,
		`"category": "Image Tools/Color adjust"`,
	} {
		if !strings.Contains(normalized, needle) {
			t.Fatalf("expected normalized blueprint JSON to preserve %s, got %s", needle, normalized)
		}
	}
}

func TestParseWorkspaceJSON_PreservesTopLevelBlueprintConfig(t *testing.T) {
	fixturePath := filepath.Join("..", "testdata", "blueprints", "text-to-image-zimage-turbo.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read blueprint fixture: %v", err)
	}

	workspace, err := ParseWorkspaceJSON(string(raw))
	if err != nil {
		t.Fatalf("ParseWorkspaceJSON returned error: %v", err)
	}

	if workspace.Config == nil {
		t.Fatal("expected top-level config object to be preserved")
	}

	normalized, err := workspace.JSON()
	if err != nil {
		t.Fatalf("workspace.JSON returned error: %v", err)
	}

	for _, needle := range []string{
		`"config": {}`,
		`"proxyWidgets": [`,
		`"slot_index": 0`,
		`"flags": {}`,
		`"workflowRendererVersion": "LG"`,
	} {
		if !strings.Contains(normalized, needle) {
			t.Fatalf("expected normalized blueprint JSON to preserve %s, got %s", needle, normalized)
		}
	}
}
