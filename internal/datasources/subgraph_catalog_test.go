package datasources

import (
	"context"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

func TestSubgraphCatalogStateFromEntries_SortsAndFlattensCatalog(t *testing.T) {
	state, err := subgraphCatalogStateFromEntries(map[string]client.GlobalSubgraphCatalogEntry{
		"z-entry": {
			Source: "custom_node",
			Name:   "Custom Pack",
			Info: client.GlobalSubgraphInfo{
				NodePack: "custom_nodes.example",
			},
		},
		"a-entry": {
			Source: "templates",
			Name:   "Brightness and Contrast",
			Info: client.GlobalSubgraphInfo{
				NodePack: "comfyui",
			},
		},
	})
	if err != nil {
		t.Fatalf("subgraphCatalogStateFromEntries returned error: %v", err)
	}

	if state.EntryCount.ValueInt64() != 2 {
		t.Fatalf("expected entry_count=2, got %d", state.EntryCount.ValueInt64())
	}

	var entries []subgraphCatalogEntryModel
	if diags := state.Entries.ElementsAs(context.Background(), &entries, false); diags.HasError() {
		t.Fatalf("failed to decode entries list: %v", diags)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].ID.ValueString() != "a-entry" {
		t.Fatalf("expected entries to be sorted by id, got first id %q", entries[0].ID.ValueString())
	}
	if entries[0].NodePack.ValueString() != "comfyui" {
		t.Fatalf("expected node_pack comfyui, got %q", entries[0].NodePack.ValueString())
	}
	if entries[1].Source.ValueString() != "custom_node" {
		t.Fatalf("expected custom_node source for second entry, got %q", entries[1].Source.ValueString())
	}
}
