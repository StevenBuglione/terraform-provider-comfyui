package version

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/datasources"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/nodeschema"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/provider"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/resources"
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/resources/generated"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	providerfw "github.com/hashicorp/terraform-plugin-framework/provider"
)

// TestVersionConsistency validates that version metadata is consistent across
// the repository, ensuring the provider version line stays aligned with the
// generated ComfyUI version pin.
func TestVersionConsistency(t *testing.T) {
	// 1. Load ComfyUI version from node_specs.json
	nodeSpecsPath := filepath.Join("..", "..", "scripts", "extract", "node_specs.json")
	nodeSpecsData, err := os.ReadFile(nodeSpecsPath)
	if err != nil {
		t.Fatalf("failed to read node_specs.json: %v", err)
	}

	var nodeSpecs struct {
		Version        string `json:"version"`
		ComfyUIVersion string `json:"comfyui_version"`
		ExtractedAt    string `json:"extracted_at"`
		TotalNodes     int    `json:"total_nodes"`
	}

	if err := json.Unmarshal(nodeSpecsData, &nodeSpecs); err != nil {
		t.Fatalf("failed to parse node_specs.json: %v", err)
	}

	// Strip leading 'v' for comparison
	expectedVersion := strings.TrimPrefix(nodeSpecs.ComfyUIVersion, "v")
	if expectedVersion == "" {
		t.Fatal("node_specs.json comfyui_version is empty")
	}

	// 2. Verify generated constant matches node_specs.json
	generatedVersion := strings.TrimPrefix(generated.ComfyUIVersion, "v")
	if generatedVersion != expectedVersion {
		t.Errorf("generated.ComfyUIVersion (%q) does not match node_specs.json comfyui_version (%q)",
			generatedVersion, expectedVersion)
	}

	// 2a. Verify node_ui_hints_generated constant matches node_specs.json
	nodeUIHintsVersion := strings.TrimPrefix(resources.GeneratedNodeUIHintsComfyUIVersion, "v")
	if nodeUIHintsVersion != expectedVersion {
		t.Errorf("resources.GeneratedNodeUIHintsComfyUIVersion (%q) does not match node_specs.json comfyui_version (%q)",
			nodeUIHintsVersion, expectedVersion)
	}

	// 2b. Verify nodeschema/generated constant matches node_specs.json
	nodeschemaVersion := strings.TrimPrefix(nodeschema.GeneratedNodeSchemaComfyUIVersion, "v")
	if nodeschemaVersion != expectedVersion {
		t.Errorf("nodeschema.GeneratedNodeSchemaComfyUIVersion (%q) does not match node_specs.json comfyui_version (%q)",
			nodeschemaVersion, expectedVersion)
	}

	// 3. Verify node count matches
	if generated.NodeCount != nodeSpecs.TotalNodes {
		t.Errorf("generated.NodeCount (%d) does not match node_specs.json total_nodes (%d)",
			generated.NodeCount, nodeSpecs.TotalNodes)
	}

	// 4. Verify extracted timestamp matches
	if generated.ExtractedAt != nodeSpecs.ExtractedAt {
		t.Errorf("generated.ExtractedAt (%q) does not match node_specs.json extracted_at (%q)",
			generated.ExtractedAt, nodeSpecs.ExtractedAt)
	}

	// 5. Verify version format is valid three-part SemVer
	parts := strings.Split(expectedVersion, ".")
	if len(parts) != 3 {
		t.Errorf("ComfyUI version %q is not valid three-part SemVer (major.minor.patch)", expectedVersion)
	}

	// 6. Extract major.minor for constraint validation
	if len(parts) >= 2 {
		majorMinor := parts[0] + "." + parts[1]
		expectedConstraint := "~> " + majorMinor

		// Check README.md contains the version constraint
		readmePath := filepath.Join("..", "..", "README.md")
		readmeData, err := os.ReadFile(readmePath)
		if err != nil {
			t.Logf("WARNING: failed to read README.md: %v", err)
		} else if !strings.Contains(string(readmeData), expectedConstraint) {
			t.Errorf("README.md does not contain expected version constraint %q", expectedConstraint)
		}

		// Check docs/index.md contains the version constraint
		docsIndexPath := filepath.Join("..", "..", "docs", "index.md")
		docsIndexData, err := os.ReadFile(docsIndexPath)
		if err != nil {
			t.Logf("WARNING: failed to read docs/index.md: %v", err)
		} else if !strings.Contains(string(docsIndexData), expectedConstraint) {
			t.Errorf("docs/index.md does not contain expected version constraint %q", expectedConstraint)
		}
	}

	t.Logf("✓ Version consistency validated: ComfyUI %s, %d nodes, extracted %s",
		nodeSpecs.ComfyUIVersion, nodeSpecs.TotalNodes, nodeSpecs.ExtractedAt)
}

// TestProviderSchemaVersionAlignment validates that the provider schema description
// includes the correct ComfyUI version and node count from generated constants.
func TestProviderSchemaVersionAlignment(t *testing.T) {
	// Create a provider instance with a test version
	testProviderVersion := "0.18.5-test"
	p := provider.New(testProviderVersion)()

	// Get the schema
	var schemaResp providerfw.SchemaResponse
	p.Schema(context.Background(), providerfw.SchemaRequest{}, &schemaResp)

	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("provider schema has diagnostics: %v", schemaResp.Diagnostics)
	}

	desc := schemaResp.Schema.Description

	// Verify schema description contains the ComfyUI version
	if !strings.Contains(desc, generated.ComfyUIVersion) {
		t.Errorf("Provider schema description does not contain ComfyUI version %q: %s",
			generated.ComfyUIVersion, desc)
	}

	// Verify schema description contains the node count
	expectedNodeCountPhrase := fmt.Sprintf("%d node resources", generated.NodeCount)
	if !strings.Contains(desc, expectedNodeCountPhrase) {
		t.Errorf("Provider schema description does not contain expected node count phrase %q: %s",
			expectedNodeCountPhrase, desc)
	}

	t.Logf("✓ Provider schema version alignment validated: description includes %s and %d nodes",
		generated.ComfyUIVersion, generated.NodeCount)
}

// TestProviderInfoDataSourceAlignment validates that the provider_info data source
// exposes the correct version metadata attributes.
func TestProviderInfoDataSourceAlignment(t *testing.T) {
	// Create a provider info data source instance
	testProviderVersion := "0.18.5-test"
	ds := datasources.NewProviderInfoDataSource(testProviderVersion)()

	// Get the schema
	var schemaResp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("provider_info data source schema has diagnostics: %v", schemaResp.Diagnostics)
	}

	// Verify all four expected attributes are present
	requiredAttrs := []string{"provider_version", "comfyui_version", "node_count", "extracted_at"}
	for _, attr := range requiredAttrs {
		if _, ok := schemaResp.Schema.Attributes[attr]; !ok {
			t.Errorf("provider_info data source schema missing required attribute %q", attr)
		}
	}

	// Verify the schema description mentions version and compatibility
	desc := schemaResp.Schema.Description
	if !strings.Contains(desc, "version") || !strings.Contains(desc, "compatibility") {
		t.Errorf("provider_info schema description should mention version and compatibility: %s", desc)
	}

	t.Logf("✓ Provider info data source alignment validated: schema exposes all required version metadata attributes")
}

// TestVersionLineAlignment validates that the compatibility line policy is
// enforced for the current ComfyUI pin.
func TestVersionLineAlignment(t *testing.T) {
	// Load ComfyUI version from generated constant
	comfyUIVersion := strings.TrimPrefix(generated.ComfyUIVersion, "v")
	parts := strings.Split(comfyUIVersion, ".")
	if len(parts) != 3 {
		t.Fatalf("ComfyUI version %q is not valid three-part SemVer", comfyUIVersion)
	}

	major := parts[0]
	minor := parts[1]
	patch := parts[2]

	// For the 0.18.x line, verify:
	// - major is 0
	// - minor is 18
	// - patch is 5 (the upstream pin)
	if major != "0" {
		t.Errorf("Expected major version 0, got %s", major)
	}

	if minor != "18" {
		t.Errorf("Expected minor version 18 for this compatibility line, got %s", minor)
	}

	if patch != "5" {
		t.Errorf("Expected patch version 5 (the ComfyUI v0.18.5 pin), got %s", patch)
	}

	t.Logf("✓ Version line alignment validated: 0.%s.x line pinned to ComfyUI v%s", minor, comfyUIVersion)
}
