package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func localFileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func writeManagedBinaryFile(path string, content []byte) (string, string, int64, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", 0, fmt.Errorf("resolve absolute path for %s: %w", path, err)
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", 0, fmt.Errorf("create directory %s: %w", dir, err)
	}
	if err := os.WriteFile(absPath, content, 0644); err != nil {
		return "", "", 0, fmt.Errorf("write file %s: %w", absPath, err)
	}

	sum := sha256.Sum256(content)
	return absPath, hex.EncodeToString(sum[:]), int64(len(content)), nil
}

func readManagedBinaryFile(path string) ([]byte, string, int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", 0, fmt.Errorf("read file %s: %w", path, err)
	}

	sum := sha256.Sum256(data)
	return data, hex.EncodeToString(sum[:]), int64(len(data)), nil
}

func remoteFileID(fileType string, subfolder string, filename string) string {
	parts := []string{strings.Trim(fileType, "/")}
	if trimmed := strings.Trim(subfolder, "/"); trimmed != "" {
		parts = append(parts, trimmed)
	}
	parts = append(parts, strings.Trim(filename, "/"))
	return strings.Join(parts, "/")
}

func addRemoteDeleteWarning(diags *diag.Diagnostics, id string) {
	diags.AddWarning(
		"Remote file deletion not supported",
		fmt.Sprintf("ComfyUI does not expose a file-delete endpoint, so %q was removed from Terraform state only.", id),
	)
}

func boolValueOrDefault(value types.Bool, defaultValue bool) bool {
	if value.IsNull() || value.IsUnknown() {
		return defaultValue
	}
	return value.ValueBool()
}
