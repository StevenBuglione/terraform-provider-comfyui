package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteManagedArtifactFile_CreatesDirectoriesAndReturnsHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "prompt.json")
	content := `{"hello":"world"}`

	sha, err := writeManagedArtifactFile(path, content)
	if err != nil {
		t.Fatalf("writeManagedArtifactFile returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written artifact: %v", err)
	}
	if string(data) != content {
		t.Fatalf("expected content to round-trip, got %q", string(data))
	}

	expected := sha256.Sum256([]byte(content))
	if sha != hex.EncodeToString(expected[:]) {
		t.Fatalf("expected sha256 %s, got %s", hex.EncodeToString(expected[:]), sha)
	}
}

func TestReadManagedArtifactFile_ReturnsContentAndHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.json")
	content := `{"nodes":[]}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to seed artifact file: %v", err)
	}

	readContent, sha, err := readManagedArtifactFile(path)
	if err != nil {
		t.Fatalf("readManagedArtifactFile returned error: %v", err)
	}

	if readContent != content {
		t.Fatalf("expected content %q, got %q", content, readContent)
	}
	if sha == "" {
		t.Fatal("expected sha to be populated")
	}
}

func TestCleanupPreviousArtifactFile_RemovesOldPath(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.json")
	newPath := filepath.Join(dir, "new.json")
	if err := os.WriteFile(oldPath, []byte(`{"old":true}`), 0644); err != nil {
		t.Fatalf("failed to seed old artifact: %v", err)
	}

	if err := cleanupPreviousArtifactFile(oldPath, newPath); err != nil {
		t.Fatalf("cleanupPreviousArtifactFile returned error: %v", err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old artifact to be removed, stat err=%v", err)
	}
}
