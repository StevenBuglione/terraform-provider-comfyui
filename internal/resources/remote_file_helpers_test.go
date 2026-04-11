package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestRemoteFileSHA256_ComputesHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	got, err := localFileSHA256(path)
	if err != nil {
		t.Fatalf("localFileSHA256 returned error: %v", err)
	}

	want := sha256.Sum256([]byte("hello"))
	if got != hex.EncodeToString(want[:]) {
		t.Fatalf("expected sha256 %s, got %s", hex.EncodeToString(want[:]), got)
	}
}

func TestRemoteFileWriteManagedBinaryFile_ReturnsAbsolutePathHashAndLength(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "download.png")
	absPath, sha, length, err := writeManagedBinaryFile(path, []byte("png-bytes"))
	if err != nil {
		t.Fatalf("writeManagedBinaryFile returned error: %v", err)
	}

	if !filepath.IsAbs(absPath) {
		t.Fatalf("expected absolute path, got %q", absPath)
	}
	if length != int64(len("png-bytes")) {
		t.Fatalf("expected content_length=%d, got %d", len("png-bytes"), length)
	}
	if sha == "" {
		t.Fatal("expected sha256 to be populated")
	}
}

func TestRemoteFileDeleteWarning_AddsDiagnostic(t *testing.T) {
	var diags diag.Diagnostics
	addRemoteDeleteWarning(&diags, "input/fixtures/example.png")
	if len(diags) != 1 || diags[0].Severity() != diag.SeverityWarning {
		t.Fatalf("expected one warning diagnostic, got %#v", diags)
	}
}
