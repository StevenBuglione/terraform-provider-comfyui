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

func newUploadedFileTestClient(server *httptest.Server) *client.Client {
	return &client.Client{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
	}
}

func TestUploadedImageResource_UploadsAndBuildsState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(path, []byte("image-bytes"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upload/image" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}
		if got := r.FormValue("overwrite"); got != "true" {
			t.Fatalf("expected overwrite=true, got %q", got)
		}
		if got := r.FormValue("subfolder"); got != "fixtures" {
			t.Fatalf("expected subfolder=fixtures, got %q", got)
		}

		if err := json.NewEncoder(w).Encode(client.UploadResponse{
			Name:      "remote.png",
			Subfolder: "fixtures",
			Type:      "input",
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	r := &UploadedImageResource{client: newUploadedFileTestClient(server)}
	model := UploadedImageModel{
		FilePath:  types.StringValue(path),
		Filename:  types.StringValue("requested.png"),
		Subfolder: types.StringValue("fixtures"),
		Type:      types.StringValue("input"),
		Overwrite: types.BoolValue(true),
	}

	if err := r.upload(context.Background(), &model); err != nil {
		t.Fatalf("upload returned error: %v", err)
	}
	if model.ID.ValueString() != "input/fixtures/remote.png" {
		t.Fatalf("expected id=input/fixtures/remote.png, got %q", model.ID.ValueString())
	}
	if model.URL.ValueString() == "" || model.SHA256.ValueString() == "" {
		t.Fatalf("expected url and sha256 to be populated, got %#v", model)
	}
}

func TestUploadedImageResource_ReadDetectsMissingRemoteFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	r := &UploadedImageResource{client: newUploadedFileTestClient(server)}
	model := UploadedImageModel{
		FilePath:  types.StringValue("/tmp/source.png"),
		Filename:  types.StringValue("remote.png"),
		Subfolder: types.StringValue("fixtures"),
		Type:      types.StringValue("input"),
		Overwrite: types.BoolValue(true),
		SHA256:    types.StringValue("prior-sha"),
	}

	exists, err := r.refresh(context.Background(), &model)
	if err != nil {
		t.Fatalf("refresh returned error: %v", err)
	}
	if exists {
		t.Fatal("expected refresh to report missing remote file")
	}
}
