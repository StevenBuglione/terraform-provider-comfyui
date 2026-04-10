package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

func writeManagedArtifactFile(path string, content string) (string, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("writing file %s: %w", path, err)
	}

	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:]), nil
}

func readManagedArtifactFile(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("reading file %s: %w", path, err)
	}

	content := string(data)
	sum := sha256.Sum256(data)
	return content, hex.EncodeToString(sum[:]), nil
}

func cleanupPreviousArtifactFile(oldPath string, newPath string) error {
	if oldPath == "" || oldPath == newPath {
		return nil
	}
	if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing previous file %s: %w", oldPath, err)
	}
	return nil
}
