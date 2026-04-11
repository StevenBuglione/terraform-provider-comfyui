package datasources

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

func TestSubgraphDefinitionStateFromEntry_NormalizesEditorNativeJSON(t *testing.T) {
	fixturePath := filepath.Join("..", "testdata", "blueprints", "brightness-and-contrast.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read blueprint fixture: %v", err)
	}

	state, err := subgraphDefinitionStateFromEntry("catalog-id", &client.GlobalSubgraphDefinition{
		Source: "templates",
		Name:   "Brightness and Contrast",
		Info: client.GlobalSubgraphInfo{
			NodePack: "comfyui",
		},
		Data: string(raw),
	})
	if err != nil {
		t.Fatalf("subgraphDefinitionStateFromEntry returned error: %v", err)
	}

	if state.ID.ValueString() != "catalog-id" {
		t.Fatalf("expected id to round-trip, got %q", state.ID.ValueString())
	}
	if state.DefinitionCount.ValueInt64() != 1 {
		t.Fatalf("expected definition_count=1, got %d", state.DefinitionCount.ValueInt64())
	}
	if !strings.Contains(state.NormalizedJSON.ValueString(), `"title": "Brightness and Contrast"`) {
		t.Fatalf("expected normalized JSON to preserve blueprint title, got %s", state.NormalizedJSON.ValueString())
	}

	var definitionIDs []string
	if diags := state.DefinitionIDs.ElementsAs(context.Background(), &definitionIDs, false); diags.HasError() {
		t.Fatalf("failed to decode definition ids: %v", diags)
	}
	if len(definitionIDs) != 1 || definitionIDs[0] != "916dff42-6166-4d45-b028-04eaf69fbb35" {
		t.Fatalf("expected definition id to round-trip, got %#v", definitionIDs)
	}
}

func TestSubgraphDefinitionStateFromEntry_RejectsNullEntry(t *testing.T) {
	_, err := subgraphDefinitionStateFromEntry("missing-id", nil)
	if err == nil {
		t.Fatal("expected nil entry to return an error")
	}
}

func TestSubgraphDefinitionStateFromEntry_PreservesRawInfoJSON(t *testing.T) {
	fixturePath := filepath.Join("..", "testdata", "blueprints", "brightness-and-contrast.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read blueprint fixture: %v", err)
	}

	var entry client.GlobalSubgraphDefinition
	if err := json.Unmarshal([]byte(`{"source":"templates","name":"Brightness and Contrast","info":{"node_pack":"comfyui","category":"Image Tools","experimental":true},"data":`+strconv.Quote(string(raw))+`}`), &entry); err != nil {
		t.Fatalf("unmarshal entry: %v", err)
	}

	state, err := subgraphDefinitionStateFromEntry("catalog-id", &entry)
	if err != nil {
		t.Fatalf("subgraphDefinitionStateFromEntry returned error: %v", err)
	}
	if state.InfoJSON.ValueString() != `{"node_pack":"comfyui","category":"Image Tools","experimental":true}` {
		t.Fatalf("expected raw info_json to preserve unknown fields, got %s", state.InfoJSON.ValueString())
	}
}

func TestSubgraphDefinitionStateFromEntry_PreservesNullInfoJSON(t *testing.T) {
	fixturePath := filepath.Join("..", "testdata", "blueprints", "brightness-and-contrast.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read blueprint fixture: %v", err)
	}

	var entry client.GlobalSubgraphDefinition
	if err := json.Unmarshal([]byte(`{"source":"templates","name":"Brightness and Contrast","info":null,"data":`+strconv.Quote(string(raw))+`}`), &entry); err != nil {
		t.Fatalf("unmarshal entry: %v", err)
	}

	state, err := subgraphDefinitionStateFromEntry("catalog-id", &entry)
	if err != nil {
		t.Fatalf("subgraphDefinitionStateFromEntry returned error: %v", err)
	}
	if !state.NodePack.IsNull() {
		t.Fatalf("expected node_pack to be null, got %q", state.NodePack.ValueString())
	}
	if state.InfoJSON.ValueString() != "null" {
		t.Fatalf("expected info_json to preserve null, got %s", state.InfoJSON.ValueString())
	}
}
