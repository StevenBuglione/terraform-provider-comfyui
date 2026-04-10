package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildIndexJSON_MultipleWorkflows(t *testing.T) {
	ids := []string{"uuid-1", "uuid-2", "uuid-3"}
	result := buildIndexJSON("landscape-collection", "All landscape generation workflows", ids)

	var manifest indexManifest
	if err := json.Unmarshal([]byte(result), &manifest); err != nil {
		t.Fatalf("failed to parse index JSON: %v", err)
	}

	if manifest.Name != "landscape-collection" {
		t.Errorf("expected name %q, got %q", "landscape-collection", manifest.Name)
	}
	if manifest.Description != "All landscape generation workflows" {
		t.Errorf("expected description %q, got %q", "All landscape generation workflows", manifest.Description)
	}
	if manifest.WorkflowCount != 3 {
		t.Errorf("expected workflow_count 3, got %d", manifest.WorkflowCount)
	}
	if len(manifest.Workflows) != 3 {
		t.Fatalf("expected 3 workflows, got %d", len(manifest.Workflows))
	}
	for i, wf := range manifest.Workflows {
		if wf.ID != ids[i] {
			t.Errorf("workflow[%d]: expected id %q, got %q", i, ids[i], wf.ID)
		}
		if wf.Index != i {
			t.Errorf("workflow[%d]: expected index %d, got %d", i, i, wf.Index)
		}
	}
}

func TestBuildIndexJSON_EmptyWorkflows(t *testing.T) {
	result := buildIndexJSON("empty-collection", "", []string{})

	var manifest indexManifest
	if err := json.Unmarshal([]byte(result), &manifest); err != nil {
		t.Fatalf("failed to parse index JSON: %v", err)
	}

	if manifest.Name != "empty-collection" {
		t.Errorf("expected name %q, got %q", "empty-collection", manifest.Name)
	}
	if manifest.WorkflowCount != 0 {
		t.Errorf("expected workflow_count 0, got %d", manifest.WorkflowCount)
	}
	if len(manifest.Workflows) != 0 {
		t.Errorf("expected 0 workflows, got %d", len(manifest.Workflows))
	}
}

func TestWriteIndexFile_CreatesDirectoryAndFile(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "nested", "output")

	ids := []string{"id-a", "id-b"}
	indexJSON := buildIndexJSON("test-collection", "test desc", ids)

	if err := writeIndexFile(subDir, indexJSON); err != nil {
		t.Fatalf("writeIndexFile failed: %v", err)
	}

	indexPath := filepath.Join(subDir, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index.json: %v", err)
	}

	var manifest indexManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("index.json is not valid JSON: %v", err)
	}

	if manifest.Name != "test-collection" {
		t.Errorf("expected name %q, got %q", "test-collection", manifest.Name)
	}
	if manifest.WorkflowCount != 2 {
		t.Errorf("expected workflow_count 2, got %d", manifest.WorkflowCount)
	}
}

func TestWorkflowCount_Computation(t *testing.T) {
	tests := []struct {
		name  string
		ids   []string
		count int
	}{
		{"zero", []string{}, 0},
		{"one", []string{"a"}, 1},
		{"five", []string{"a", "b", "c", "d", "e"}, 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildIndexJSON("col", "", tc.ids)

			var manifest indexManifest
			if err := json.Unmarshal([]byte(result), &manifest); err != nil {
				t.Fatalf("failed to parse index JSON: %v", err)
			}
			if manifest.WorkflowCount != tc.count {
				t.Errorf("expected workflow_count %d, got %d", tc.count, manifest.WorkflowCount)
			}
		})
	}
}

func TestDescriptionValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with description", "my workflows", "my workflows"},
		{"empty description", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildIndexJSON("col", tc.input, []string{"id-1"})

			var manifest indexManifest
			if err := json.Unmarshal([]byte(result), &manifest); err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			if manifest.Description != tc.expected {
				t.Errorf("expected description %q, got %q", tc.expected, manifest.Description)
			}
		})
	}
}
