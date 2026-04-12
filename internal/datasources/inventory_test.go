package datasources

import (
	"context"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/inventory"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestInventoryDataSourceSchema_ExposesKindsAndInventories(t *testing.T) {
	ds := NewInventoryDataSource().(*InventoryDataSource)
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	kindsAttr, ok := resp.Schema.Attributes["kinds"].(datasourceschema.ListAttribute)
	if !ok {
		t.Fatalf("expected kinds to be a list attribute, got %T", resp.Schema.Attributes["kinds"])
	}
	if !kindsAttr.Optional {
		t.Fatal("expected kinds to be optional")
	}

	inventoriesAttr, ok := resp.Schema.Attributes["inventories"].(datasourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected inventories to be a list nested attribute, got %T", resp.Schema.Attributes["inventories"])
	}
	if _, ok := inventoriesAttr.NestedObject.Attributes["kind"].(datasourceschema.StringAttribute); !ok {
		t.Fatalf("expected inventories.kind to be a string attribute, got %T", inventoriesAttr.NestedObject.Attributes["kind"])
	}
	if _, ok := inventoriesAttr.NestedObject.Attributes["values"].(datasourceschema.ListAttribute); !ok {
		t.Fatalf("expected inventories.values to be a list attribute, got %T", inventoriesAttr.NestedObject.Attributes["values"])
	}
	if _, ok := inventoriesAttr.NestedObject.Attributes["value_count"].(datasourceschema.Int64Attribute); !ok {
		t.Fatalf("expected inventories.value_count to be an int64 attribute, got %T", inventoriesAttr.NestedObject.Attributes["value_count"])
	}
}

func TestInventoryStateFromKinds_ReturnsSortedInventoryValues(t *testing.T) {
	state, diags := inventoryStateFromKinds(context.Background(), inventoryInventoryService{
		values: map[inventory.Kind][]string{
			inventory.KindCheckpoints: {"zeta.safetensors", "alpha.safetensors"},
			inventory.KindLoras:       {"detailer.safetensors"},
		},
	}, []inventory.Kind{inventory.KindCheckpoints, inventory.KindLoras})
	if diags.HasError() {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}

	if len(state.Inventories) != 2 {
		t.Fatalf("expected two inventories, got %#v", state.Inventories)
	}
	if state.Inventories[0].Kind.ValueString() != string(inventory.KindCheckpoints) {
		t.Fatalf("expected checkpoints inventory first, got %#v", state.Inventories[0])
	}
	if state.Inventories[0].ValueCount.ValueInt64() != 2 {
		t.Fatalf("expected checkpoints inventory count=2, got %#v", state.Inventories[0])
	}
	var checkpointValues []string
	if diags := state.Inventories[0].Values.ElementsAs(context.Background(), &checkpointValues, false); diags.HasError() {
		t.Fatalf("failed to read checkpoint values: %v", diags)
	}
	if len(checkpointValues) != 2 || checkpointValues[0] != "alpha.safetensors" || checkpointValues[1] != "zeta.safetensors" {
		t.Fatalf("expected sorted checkpoint values, got %#v", checkpointValues)
	}
}

func TestInventoryStateFromKinds_ReportsLookupFailures(t *testing.T) {
	_, diags := inventoryStateFromKinds(context.Background(), inventoryInventoryService{
		err: map[inventory.Kind]error{
			inventory.KindCheckpoints: context.DeadlineExceeded,
		},
	}, []inventory.Kind{inventory.KindCheckpoints})
	if !diags.HasError() {
		t.Fatal("expected diagnostics when inventory lookup fails")
	}
}

type inventoryInventoryService struct {
	values map[inventory.Kind][]string
	err    map[inventory.Kind]error
}

func (s inventoryInventoryService) GetInventory(_ context.Context, kind inventory.Kind) ([]string, error) {
	if err := s.err[kind]; err != nil {
		return nil, err
	}
	return append([]string(nil), s.values[kind]...), nil
}
