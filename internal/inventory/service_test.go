package inventory

import (
	"context"
	"errors"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
)

type fakeObjectInfoReader struct {
	calls int
	info  map[string]client.NodeInfo
	err   error
}

func (f *fakeObjectInfoReader) GetObjectInfo() (map[string]client.NodeInfo, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.info, nil
}

func TestParseKind(t *testing.T) {
	if kind, ok := ParseKind("checkpoints"); !ok || kind != KindCheckpoints {
		t.Fatalf("expected checkpoints kind, got %q ok=%v", kind, ok)
	}
	if _, ok := ParseKind("not_real"); ok {
		t.Fatal("expected unknown kind to be rejected")
	}
}

func TestInventoryServiceGetInventory_CachesResults(t *testing.T) {
	reader := &fakeObjectInfoReader{
		info: map[string]client.NodeInfo{
			"CheckpointLoaderSimple": {
				Input: client.NodeInputInfo{
					Required: map[string]interface{}{
						"ckpt_name": []interface{}{
							[]interface{}{"b.safetensors", "a.safetensors"},
							map[string]interface{}{},
						},
					},
				},
			},
		},
	}
	service := NewService(reader)

	got, err := service.GetInventory(context.Background(), KindCheckpoints)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reader.calls != 1 {
		t.Fatalf("expected one object_info call, got %d", reader.calls)
	}
	if len(got) != 2 || got[0] != "a.safetensors" || got[1] != "b.safetensors" {
		t.Fatalf("unexpected inventory values: %#v", got)
	}

	gotAgain, err := service.GetInventory(context.Background(), KindCheckpoints)
	if err != nil {
		t.Fatalf("unexpected second error: %v", err)
	}
	if reader.calls != 1 {
		t.Fatalf("expected cached object_info call count of 1, got %d", reader.calls)
	}
	if len(gotAgain) != 2 || gotAgain[0] != "a.safetensors" || gotAgain[1] != "b.safetensors" {
		t.Fatalf("unexpected cached inventory values: %#v", gotAgain)
	}
}

func TestInventoryServiceGetInventory_UsesOptionsMetadata(t *testing.T) {
	reader := &fakeObjectInfoReader{
		info: map[string]client.NodeInfo{
			"BasicScheduler": {
				Input: client.NodeInputInfo{
					Required: map[string]interface{}{
						"scheduler": []interface{}{
							"COMBO",
							map[string]interface{}{
								"options": []interface{}{"karras", "simple"},
							},
						},
					},
				},
			},
		},
	}
	service := NewService(reader)

	got, err := service.GetInventory(context.Background(), Kind("schedulers"))
	if err == nil {
		t.Fatalf("expected unsupported kind error, got values %#v", got)
	}
}

func TestInventoryServiceGetInventory_ReturnsLookupErrors(t *testing.T) {
	reader := &fakeObjectInfoReader{err: errors.New("boom")}
	service := NewService(reader)

	_, err := service.GetInventory(context.Background(), KindCheckpoints)
	if err == nil || err.Error() == "" {
		t.Fatal("expected lookup error")
	}
}

func TestInventoryServiceGetInventory_RequiresRepresentativeNode(t *testing.T) {
	service := NewService(&fakeObjectInfoReader{})

	_, err := service.GetInventory(context.Background(), Kind("unsupported_kind"))
	if err == nil {
		t.Fatal("expected unsupported inventory kind error")
	}
}
