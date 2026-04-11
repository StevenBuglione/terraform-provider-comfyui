package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUploadedMaskResource_UploadsWithOriginalReference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mask.png")
	if err := os.WriteFile(path, []byte("mask-bytes"), 0644); err != nil {
		t.Fatalf("failed to write mask file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upload/mask" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}
		if got := r.FormValue("original_ref"); got != `{"filename":"base.png","subfolder":"gallery","type":"output"}` {
			t.Fatalf("unexpected original_ref payload: %q", got)
		}

		if err := json.NewEncoder(w).Encode(client.UploadResponse{
			Name:      "mask.png",
			Subfolder: "",
			Type:      "input",
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	r := &UploadedMaskResource{client: newUploadedFileTestClient(server)}
	model := UploadedMaskModel{
		FilePath:          types.StringValue(path),
		Type:              types.StringValue("input"),
		Overwrite:         types.BoolValue(true),
		OriginalFilename:  types.StringValue("base.png"),
		OriginalSubfolder: types.StringValue("gallery"),
		OriginalType:      types.StringValue("output"),
	}

	if err := r.upload(context.Background(), &model); err != nil {
		t.Fatalf("upload returned error: %v", err)
	}
	if model.ID.ValueString() != "input/mask.png" {
		t.Fatalf("expected id=input/mask.png, got %q", model.ID.ValueString())
	}
}
