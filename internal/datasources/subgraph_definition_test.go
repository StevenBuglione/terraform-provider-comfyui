package datasources

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

func TestSubgraphDefinitionStateFromEntry_NormalizesEditorNativeJSON(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "third_party", "ComfyUI", "blueprints", "Brightness and Contrast.json")
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
