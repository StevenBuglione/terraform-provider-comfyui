package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
)

// --- JSON structures matching node_specs.json ---

type NodeSpecs struct {
	Version  string `json:"version"`
	Nodes    []Node `json:"nodes"`
	Total    int    `json:"total_nodes"`
}

type Node struct {
	NodeID                string   `json:"node_id"`
	ClassName             string   `json:"class_name"`
	DisplayName           *string  `json:"display_name"`
	Description           *string  `json:"description"`
	Category              string   `json:"category"`
	FunctionName          string   `json:"function_name"`
	IsOutputNode          bool     `json:"is_output_node"`
	IsDeprecated          bool     `json:"is_deprecated"`
	IsExperimental        bool     `json:"is_experimental"`
	Inputs                []Input  `json:"inputs"`
	Outputs               []Output `json:"outputs"`
	TerraformResourceName string   `json:"terraform_resource_name"`
}

type Input struct {
	Name           string        `json:"name"`
	Type           string        `json:"type"`
	Required       bool          `json:"required"`
	IsLinkType     bool          `json:"is_link_type"`
	Default        interface{}   `json:"default"`
	Min            *float64      `json:"min"`
	Max            *float64      `json:"max"`
	Step           *float64      `json:"step"`
	RawOptions     []interface{} `json:"options"`
	Multiline      *bool         `json:"multiline"`
	DynamicOptions *bool         `json:"dynamic_options"`
}

// StringOptions converts RawOptions to []string, handling int/float/bool values.
func (inp Input) StringOptions() []string {
	if len(inp.RawOptions) == 0 {
		return nil
	}
	opts := make([]string, 0, len(inp.RawOptions))
	for _, o := range inp.RawOptions {
		opts = append(opts, fmt.Sprintf("%v", o))
	}
	return opts
}

type Output struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	SlotIndex int    `json:"slot_index"`
}

// --- Template data structures ---

type ResourceData struct {
	StructName               string
	TFResourceSuffix         string
	ClassName                string
	Description              string
	Fields                   []FieldData
	Attributes               []AttrData
	Outputs                  []OutputRef
	HasOutputs               bool
	NeedsInt64Validator      bool
	NeedsFloat64Validator    bool
	NeedsStringValidator     bool
	NeedsBoolPlanModifier    bool
	NeedsInt64PlanModifier   bool
	NeedsFloat64PlanModifier bool
}

type FieldData struct {
	GoName string
	GoType string
	TFName string
}

type AttrData struct {
	Definition string
}

type OutputRef struct {
	GoName    string
	SlotIndex int
}

type RegistryData struct {
	Constructors []string
}

func main() {
	start := time.Now()

	specsPath := "scripts/extract/node_specs.json"
	outputDir := "internal/resources/generated"

	log.Printf("Reading node specs from %s", specsPath)
	data, err := os.ReadFile(specsPath)
	if err != nil {
		log.Fatalf("Failed to read specs: %v", err)
	}

	var specs NodeSpecs
	if err := json.Unmarshal(data, &specs); err != nil {
		log.Fatalf("Failed to parse specs: %v", err)
	}
	log.Printf("Loaded %d nodes", len(specs.Nodes))

	// Parse templates
	resTmpl, err := template.New("resource").Parse(resourceTemplate)
	if err != nil {
		log.Fatalf("Failed to parse resource template: %v", err)
	}
	regTmpl, err := template.New("registry").Parse(registryTemplate)
	if err != nil {
		log.Fatalf("Failed to parse registry template: %v", err)
	}

	// Clean output directory
	entries, _ := os.ReadDir(outputDir)
	for _, e := range entries {
		os.Remove(filepath.Join(outputDir, e.Name()))
	}

	var constructors []string
	var warnings []string
	generated := 0

	for _, node := range specs.Nodes {
		rd, warns := buildResourceData(node)
		warnings = append(warnings, warns...)

		var buf bytes.Buffer
		if err := resTmpl.Execute(&buf, rd); err != nil {
			log.Printf("WARNING: template exec failed for %s: %v", node.NodeID, err)
			warnings = append(warnings, fmt.Sprintf("template exec failed for %s: %v", node.NodeID, err))
			continue
		}

		// gofmt the output
		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			log.Printf("WARNING: gofmt failed for %s: %v", node.NodeID, err)
			// Write unformatted for debugging
			formatted = buf.Bytes()
		}

		// File name: resource_<suffix>.go
		filename := fmt.Sprintf("resource_%s.go", rd.TFResourceSuffix)
		outPath := filepath.Join(outputDir, filename)

		if err := os.WriteFile(outPath, formatted, 0644); err != nil {
			log.Fatalf("Failed to write %s: %v", outPath, err)
		}

		constructors = append(constructors, fmt.Sprintf("New%sResource", rd.StructName))
		generated++
	}

	// Sort constructors for deterministic output
	sort.Strings(constructors)

	// Generate registry
	regData := RegistryData{Constructors: constructors}
	var regBuf bytes.Buffer
	if err := regTmpl.Execute(&regBuf, regData); err != nil {
		log.Fatalf("Failed to execute registry template: %v", err)
	}
	regFormatted, err := format.Source(regBuf.Bytes())
	if err != nil {
		log.Printf("WARNING: gofmt failed for registry: %v", err)
		regFormatted = regBuf.Bytes()
	}
	regPath := filepath.Join(outputDir, "registry.go")
	if err := os.WriteFile(regPath, regFormatted, 0644); err != nil {
		log.Fatalf("Failed to write registry: %v", err)
	}

	elapsed := time.Since(start)

	fmt.Println("════════════════════════════════════════════")
	fmt.Println("  ComfyUI Terraform Provider — Code Generator")
	fmt.Println("════════════════════════════════════════════")
	fmt.Printf("  Resources generated:  %d\n", generated)
	fmt.Printf("  Files created:        %d (+ 1 registry)\n", generated)
	fmt.Printf("  Warnings:             %d\n", len(warnings))
	fmt.Printf("  Output directory:     %s\n", outputDir)
	fmt.Printf("  Elapsed:              %s\n", elapsed.Round(time.Millisecond))
	fmt.Println("════════════════════════════════════════════")

	for _, w := range warnings {
		fmt.Printf("  ⚠  %s\n", w)
	}
}

func buildResourceData(node Node) (ResourceData, []string) {
	var warnings []string

	rawSuffix := strings.TrimPrefix(node.TerraformResourceName, "comfyui_")
	suffix := sanitizeName(rawSuffix)
	structName := toPascalCase(suffix)

	desc := fmt.Sprintf("ComfyUI %s node", node.ClassName)
	if node.DisplayName != nil && *node.DisplayName != "" {
		desc = fmt.Sprintf("ComfyUI %s node — %s", node.ClassName, *node.DisplayName)
	}
	if node.Description != nil && *node.Description != "" {
		desc = *node.Description
	}
	if node.Category != "" {
		desc += " [" + node.Category + "]"
	}
	if node.IsDeprecated {
		desc = "(DEPRECATED) " + desc
	}
	if node.IsExperimental {
		desc = "(EXPERIMENTAL) " + desc
	}

	rd := ResourceData{
		StructName:       structName,
		TFResourceSuffix: suffix,
		ClassName:        node.ClassName,
		Description:      desc,
		HasOutputs:       len(node.Outputs) > 0,
	}

	// Track used TF names to avoid duplicates within a resource
	usedNames := map[string]bool{"id": true, "node_id": true}

	// Process inputs
	for _, inp := range node.Inputs {
		tfName := sanitizeName(inp.Name)
		// Deduplicate
		origTF := tfName
		for i := 2; usedNames[tfName]; i++ {
			tfName = fmt.Sprintf("%s_%d", origTF, i)
		}
		usedNames[tfName] = true

		goName := safeGoName(toPascalCase(tfName))
		goType := goFieldType(inp.Type)

		rd.Fields = append(rd.Fields, FieldData{
			GoName: goName,
			GoType: goType,
			TFName: tfName,
		})

		attrDef := buildInputAttribute(inp, tfName, &rd)
		rd.Attributes = append(rd.Attributes, AttrData{Definition: attrDef})
	}

	// Process outputs
	for _, out := range node.Outputs {
		outTFName := sanitizeName(out.Name) + "_output"
		origTF := outTFName
		for i := 2; usedNames[outTFName]; i++ {
			outTFName = fmt.Sprintf("%s_%d", origTF, i)
		}
		usedNames[outTFName] = true

		goName := safeGoName(toPascalCase(outTFName))

		rd.Fields = append(rd.Fields, FieldData{
			GoName: goName,
			GoType: "types.String",
			TFName: outTFName,
		})

		rd.Outputs = append(rd.Outputs, OutputRef{
			GoName:    goName,
			SlotIndex: out.SlotIndex,
		})

		attrDef := buildOutputAttribute(out, outTFName)
		rd.Attributes = append(rd.Attributes, AttrData{Definition: attrDef})
	}

	return rd, warnings
}

func buildInputAttribute(inp Input, tfName string, rd *ResourceData) string {
	var b strings.Builder

	attrType := tfAttributeType(inp.Type)

	b.WriteString(fmt.Sprintf("%q: %s{\n", tfName, attrType))

	// Description
	descParts := []string{fmt.Sprintf("Input: %s", inp.Type)}
	if inp.IsLinkType {
		descParts = append(descParts, "(link)")
	}
	if inp.Default != nil {
		descParts = append(descParts, fmt.Sprintf("default: %v", inp.Default))
	}
	b.WriteString(fmt.Sprintf("\t\t\t\tDescription: %q,\n", strings.Join(descParts, " ")))

	// Required/Optional
	if inp.Required {
		b.WriteString("\t\t\t\tRequired: true,\n")
	} else {
		b.WriteString("\t\t\t\tOptional: true,\n")
	}

	// Validators
	switch inp.Type {
	case "INT":
		if inp.Min != nil && inp.Max != nil {
			rd.NeedsInt64Validator = true
			minVal := clampInt64(*inp.Min)
			maxVal := clampInt64(*inp.Max)
			b.WriteString(fmt.Sprintf("\t\t\t\tValidators: []validator.Int64{\n\t\t\t\t\tint64validator.Between(%d, %d),\n\t\t\t\t},\n", minVal, maxVal))
		}
	case "FLOAT":
		if inp.Min != nil && inp.Max != nil {
			rd.NeedsFloat64Validator = true
			b.WriteString(fmt.Sprintf("\t\t\t\tValidators: []validator.Float64{\n\t\t\t\t\tfloat64validator.Between(%v, %v),\n\t\t\t\t},\n", *inp.Min, *inp.Max))
		}
	case "COMBO":
		opts := inp.StringOptions()
		if len(opts) > 0 && (inp.DynamicOptions == nil || !*inp.DynamicOptions) {
			rd.NeedsStringValidator = true
			b.WriteString("\t\t\t\tValidators: []validator.String{\n\t\t\t\t\tstringvalidator.OneOf(\n")
			for _, opt := range opts {
				b.WriteString(fmt.Sprintf("\t\t\t\t\t\t%q,\n", opt))
			}
			b.WriteString("\t\t\t\t\t),\n\t\t\t\t},\n")
		}
	}

	b.WriteString("\t\t\t},")
	return b.String()
}

func buildOutputAttribute(out Output, tfName string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%q: schema.StringAttribute{\n", tfName))
	b.WriteString(fmt.Sprintf("\t\t\t\tDescription: \"Output: %s (slot %d)\",\n", out.Type, out.SlotIndex))
	b.WriteString("\t\t\t\tComputed: true,\n")
	b.WriteString("\t\t\t\tPlanModifiers: []planmodifier.String{\n\t\t\t\t\tstringplanmodifier.UseStateForUnknown(),\n\t\t\t\t},\n")
	b.WriteString("\t\t\t},")
	return b.String()
}
