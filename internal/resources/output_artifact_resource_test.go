package resources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOutputArtifactResource_DownloadsAndWritesLocalFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "downloads", "image.png")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("png-bytes"))
		case http.MethodHead:
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	}))
	defer server.Close()

	r := &OutputArtifactResource{client: newUploadedFileTestClient(server)}
	model := OutputArtifactModel{
		Filename:  types.StringValue("image.png"),
		Subfolder: types.StringValue("gallery"),
		Type:      types.StringValue("output"),
		Path:      types.StringValue(path),
	}

	if err := r.download(context.Background(), &model, ""); err != nil {
		t.Fatalf("download returned error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(content) != "png-bytes" {
		t.Fatalf("unexpected downloaded content: %q", string(content))
	}
	if model.ContentLength.ValueInt64() != int64(len("png-bytes")) {
		t.Fatalf("expected content_length=%d, got %d", len("png-bytes"), model.ContentLength.ValueInt64())
	}
}

func TestOutputArtifactResource_ReadRedownloadsMissingLocalFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "downloads", "image.png")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("png-bytes"))
		case http.MethodHead:
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	}))
	defer server.Close()

	r := &OutputArtifactResource{client: newUploadedFileTestClient(server)}
	model := OutputArtifactModel{
		Filename: types.StringValue("image.png"),
		Type:     types.StringValue("output"),
		Path:     types.StringValue(path),
	}

	exists, err := r.refresh(context.Background(), &model)
	if err != nil {
		t.Fatalf("refresh returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected remote file to exist")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected local file to be restored, got %v", err)
	}
}

func TestOutputArtifactResource_UpdateCleansPreviousPath(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.png")
	newPath := filepath.Join(dir, "nested", "new.png")
	if err := os.WriteFile(oldPath, []byte("old-bytes"), 0644); err != nil {
		t.Fatalf("failed to seed old file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("new-bytes"))
		case http.MethodHead:
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	}))
	defer server.Close()

	r := &OutputArtifactResource{client: newUploadedFileTestClient(server)}
	model := OutputArtifactModel{
		Filename: types.StringValue("image.png"),
		Type:     types.StringValue("output"),
		Path:     types.StringValue(newPath),
	}

	if err := r.download(context.Background(), &model, oldPath); err != nil {
		t.Fatalf("download returned error: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old path to be removed, got %v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected new path to exist, got %v", err)
	}
}
