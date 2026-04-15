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
	"strconv"
	"strings"
	"text/template"
	"time"
)

// --- JSON structures matching node_specs.json ---

type NodeSpecs struct {
	Version        string `json:"version"`
	ComfyUIVersion string `json:"comfyui_version"`
	ExtractedAt    string `json:"extracted_at"`
	Nodes          []Node `json:"nodes"`
	Total          int    `json:"total_nodes"`
}

type Node struct {
	NodeID                string        `json:"node_id"`
	ClassName             string        `json:"class_name"`
	DisplayName           *string       `json:"display_name"`
	Description           *string       `json:"description"`
	Category              string        `json:"category"`
	FunctionName          string        `json:"function_name"`
	IsOutputNode          bool          `json:"is_output_node"`
	IsDeprecated          bool          `json:"is_deprecated"`
	IsExperimental        bool          `json:"is_experimental"`
	Inputs                []Input       `json:"inputs"`
	HiddenInputs          []HiddenInput `json:"hidden_inputs"`
	Outputs               []Output      `json:"outputs"`
	Source                SourceInfo    `json:"source"`
	TerraformResourceName string        `json:"terraform_resource_name"`
}

type Input struct {
	Name                         string               `json:"name"`
	Type                         string               `json:"type"`
	Required                     bool                 `json:"required"`
	IsLinkType                   bool                 `json:"is_link_type"`
	ValidationKind               string               `json:"validation_kind"`
	InventoryKind                string               `json:"inventory_kind"`
	SupportsStrictPlanValidation bool                 `json:"supports_strict_plan_validation"`
	Default                      interface{}          `json:"default"`
	Min                          *NumberValue         `json:"min"`
	Max                          *NumberValue         `json:"max"`
	Step                         *NumberValue         `json:"step"`
	RawOptions                   []interface{}        `json:"options"`
	Multiline                    *bool                `json:"multiline"`
	DynamicOptions               *bool                `json:"dynamic_options"`
	DynamicOptionsSource         *string              `json:"dynamic_options_source"`
	DynamicComboOptions          []DynamicComboOption `json:"dynamic_combo_options"`
	Tooltip                      *string              `json:"tooltip"`
	DisplayName                  *string              `json:"display_name"`

	mergedDefaultValues        []string
	mergedRangeHints           []string
	mergedStepValues           []string
	mergedDynamicOptionSources []string
}

type DynamicComboOption struct {
	Key    string  `json:"key"`
	Inputs []Input `json:"inputs"`
}

type NumberValue struct {
	Raw     string
	Float64 float64
}

func (n *NumberValue) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		return nil
	}
	n.Raw = raw
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return err
	}
	n.Float64 = value
	return nil
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
	IsList    bool   `json:"is_list"`
}

type HiddenInput struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type SourceInfo struct {
	File       string `json:"file"`
	Pattern    string `json:"pattern"`
	LineNumber *int   `json:"line_number"`
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
	Constructors   []string
	ComfyUIVersion string
	NodeCount      int
	ExtractedAt    string
}

func main() {
	start := time.Now()

	specsPath := "scripts/extract/node_specs.json"
	uiHintsPath := "scripts/extract/node_ui_hints.json"
	outputDir := "internal/resources/generated"
	uiHintsOutputPath := "internal/resources/node_ui_hints_generated.go"
	nodeSchemaOutputPath := "internal/nodeschema/generated.go"

	log.Printf("Reading node specs from %s", specsPath)
	data, err := os.ReadFile(specsPath)
	if err != nil {
		log.Fatalf("Failed to read specs: %v", err)
	}

	var specs NodeSpecs
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&specs); err != nil {
		log.Fatalf("Failed to parse specs: %v", err)
	}
	log.Printf("Loaded %d nodes", len(specs.Nodes))

	log.Printf("Reading node UI hints from %s", uiHintsPath)
	uiHintsData, err := os.ReadFile(uiHintsPath)
	if err != nil {
		log.Fatalf("Failed to read UI hints: %v", err)
	}

	var uiHints NodeUIHints
	if err := json.Unmarshal(uiHintsData, &uiHints); err != nil {
		log.Fatalf("Failed to parse UI hints: %v", err)
	}
	if len(uiHints.FailedNodes) > 0 {
		log.Fatalf("UI hints contain %d extraction failures", len(uiHints.FailedNodes))
	}
	log.Printf("Loaded %d node UI hints", len(uiHints.Nodes))

	// Parse templates
	resTmpl, err := template.New("resource").Parse(resourceTemplate)
	if err != nil {
		log.Fatalf("Failed to parse resource template: %v", err)
	}
	regTmpl, err := template.New("registry").Parse(registryTemplate)
	if err != nil {
		log.Fatalf("Failed to parse registry template: %v", err)
	}
	uiHintsTmpl, err := template.New("node_ui_hints").Parse(nodeUIHintsTemplate)
	if err != nil {
		log.Fatalf("Failed to parse node UI hints template: %v", err)
	}
	nodeSchemaTmpl, err := template.New("node_schema").Parse(nodeSchemaTemplate)
	if err != nil {
		log.Fatalf("Failed to parse node schema template: %v", err)
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

	// Generate registry with version metadata
	regData := RegistryData{
		Constructors:   constructors,
		ComfyUIVersion: specs.ComfyUIVersion,
		NodeCount:      generated,
		ExtractedAt:    specs.ExtractedAt,
	}
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

	uiHintsTemplateData := buildNodeUIHintsTemplateData(uiHints)
	var uiHintsBuf bytes.Buffer
	if err := uiHintsTmpl.Execute(&uiHintsBuf, uiHintsTemplateData); err != nil {
		log.Fatalf("Failed to execute node UI hints template: %v", err)
	}
	uiHintsFormatted, err := format.Source(uiHintsBuf.Bytes())
	if err != nil {
		log.Fatalf("Failed to format node UI hints output: %v", err)
	}
	if err := os.WriteFile(uiHintsOutputPath, uiHintsFormatted, 0644); err != nil {
		log.Fatalf("Failed to write node UI hints output: %v", err)
	}

	nodeSchemaTemplateData := buildNodeSchemaTemplateData(specs)
	var nodeSchemaBuf bytes.Buffer
	if err := nodeSchemaTmpl.Execute(&nodeSchemaBuf, nodeSchemaTemplateData); err != nil {
		log.Fatalf("Failed to execute node schema template: %v", err)
	}
	nodeSchemaFormatted, err := format.Source(nodeSchemaBuf.Bytes())
	if err != nil {
		log.Fatalf("Failed to format node schema output: %v", err)
	}
	if err := os.WriteFile(nodeSchemaOutputPath, nodeSchemaFormatted, 0644); err != nil {
		log.Fatalf("Failed to write node schema output: %v", err)
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

func buildNodeUIHintsTemplateData(hints NodeUIHints) NodeUIHintsTemplateData {
	nodes := make([]NodeUIHintTemplateNode, 0, len(hints.Nodes))

	nodeTypes := make([]string, 0, len(hints.Nodes))
	for nodeType := range hints.Nodes {
		nodeTypes = append(nodeTypes, nodeType)
	}
	sort.Strings(nodeTypes)

	for _, nodeType := range nodeTypes {
		hint := hints.Nodes[nodeType]
		widgets := make([]WidgetUIHintTemplateNode, 0, len(hint.Widgets))
		widgetNames := make([]string, 0, len(hint.Widgets))
		for widgetName := range hint.Widgets {
			widgetNames = append(widgetNames, widgetName)
		}
		sort.Strings(widgetNames)
		for _, widgetName := range widgetNames {
			widgetHint := hint.Widgets[widgetName]
			widgets = append(widgets, WidgetUIHintTemplateNode{
				Name:           widgetName,
				WidgetType:     widgetHint.WidgetType,
				HasComputeSize: widgetHint.HasComputeSize,
				ComputedWidth:  derefFloat64(widgetHint.ComputedWidth),
				ComputedHeight: derefFloat64(widgetHint.ComputedHeight),
				MinNodeWidth:   derefFloat64(widgetHint.MinNodeWidth),
				MinNodeHeight:  derefFloat64(widgetHint.MinNodeHeight),
			})
		}

		nodes = append(nodes, NodeUIHintTemplateNode{
			NodeType:       nodeType,
			MinWidth:       derefFloat64(hint.MinWidth),
			MinHeight:      derefFloat64(hint.MinHeight),
			ComputedWidth:  derefFloat64(hint.ComputedWidth),
			ComputedHeight: derefFloat64(hint.ComputedHeight),
			Widgets:        widgets,
		})
	}

	return NodeUIHintsTemplateData{
		Version:        hints.Version,
		ExtractedAt:    hints.ExtractedAt,
		ComfyUICommit:  hints.ComfyUICommit,
		ComfyUIVersion: hints.ComfyUIVersion,
		TotalNodes:     len(nodes),
		Nodes:          nodes,
	}
}

func derefFloat64(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func buildNodeSchemaTemplateData(specs NodeSpecs) NodeSchemaTemplateData {
	nodes := make([]GeneratedNodeSchema, 0, len(specs.Nodes))
	for _, node := range specs.Nodes {
		nodes = append(nodes, GeneratedNodeSchema{
			NodeType:       node.NodeID,
			TerraformType:  node.TerraformResourceName,
			DisplayName:    stringValue(node.DisplayName),
			Description:    stringValue(node.Description),
			Category:       node.Category,
			OutputNode:     node.IsOutputNode,
			Deprecated:     node.IsDeprecated,
			Experimental:   node.IsExperimental,
			RequiredInputs: buildGeneratedNodeSchemaInputs(node.Inputs, true),
			OptionalInputs: buildGeneratedNodeSchemaInputs(node.Inputs, false),
			HiddenInputs:   buildGeneratedNodeSchemaHiddenInputs(node.HiddenInputs),
			Outputs:        buildGeneratedNodeSchemaOutputs(node.Outputs),
		})
	}

	return NodeSchemaTemplateData{
		Version:        specs.Version,
		ExtractedAt:    specs.ExtractedAt,
		ComfyUIVersion: specs.ComfyUIVersion,
		TotalNodes:     len(nodes),
		Nodes:          nodes,
	}
}

func buildGeneratedNodeSchemaInputs(inputs []Input, required bool) []GeneratedNodeSchemaInput {
	filtered := make([]GeneratedNodeSchemaInput, 0)
	for _, input := range inputs {
		if input.Required != required {
			continue
		}
		filtered = append(filtered, GeneratedNodeSchemaInput{
			Name:                         input.Name,
			Type:                         input.Type,
			Required:                     input.Required,
			IsLinkType:                   input.IsLinkType,
			ValidationKind:               input.ValidationKind,
			InventoryKind:                input.InventoryKind,
			SupportsStrictPlanValidation: input.SupportsStrictPlanValidation,
			DefaultValue:                 formatOptionalValue(input.Default),
			HasDefaultValue:              input.Default != nil,
			MinValue:                     formatOptionalNumber(input.Min),
			HasMinValue:                  input.Min != nil,
			MaxValue:                     formatOptionalNumber(input.Max),
			HasMaxValue:                  input.Max != nil,
			StepValue:                    formatOptionalNumber(input.Step),
			HasStepValue:                 input.Step != nil,
			EnumValues:                   input.StringOptions(),
			Multiline:                    boolValue(input.Multiline),
			DynamicOptions:               boolValue(input.DynamicOptions),
			DynamicOptionsSource:         stringValue(input.DynamicOptionsSource),
			DynamicComboOptions:          buildGeneratedDynamicComboOptions(input.DynamicComboOptions),
			Tooltip:                      stringValue(input.Tooltip),
			DisplayName:                  stringValue(input.DisplayName),
		})
	}
	return filtered
}

func buildGeneratedDynamicComboOptions(options []DynamicComboOption) []GeneratedDynamicComboOption {
	if len(options) == 0 {
		return nil
	}

	generated := make([]GeneratedDynamicComboOption, 0, len(options))
	for _, option := range options {
		inputs := make([]GeneratedNodeSchemaInput, 0, len(option.Inputs))
		for _, input := range option.Inputs {
			inputs = append(inputs, GeneratedNodeSchemaInput{
				Name:                         input.Name,
				Type:                         input.Type,
				Required:                     input.Required,
				IsLinkType:                   input.IsLinkType,
				ValidationKind:               input.ValidationKind,
				InventoryKind:                input.InventoryKind,
				SupportsStrictPlanValidation: input.SupportsStrictPlanValidation,
				DefaultValue:                 formatOptionalValue(input.Default),
				HasDefaultValue:              input.Default != nil,
				MinValue:                     formatOptionalNumber(input.Min),
				HasMinValue:                  input.Min != nil,
				MaxValue:                     formatOptionalNumber(input.Max),
				HasMaxValue:                  input.Max != nil,
				StepValue:                    formatOptionalNumber(input.Step),
				HasStepValue:                 input.Step != nil,
				EnumValues:                   input.StringOptions(),
				Multiline:                    boolValue(input.Multiline),
				DynamicOptions:               boolValue(input.DynamicOptions),
				DynamicOptionsSource:         stringValue(input.DynamicOptionsSource),
				DynamicComboOptions:          buildGeneratedDynamicComboOptions(input.DynamicComboOptions),
				Tooltip:                      stringValue(input.Tooltip),
				DisplayName:                  stringValue(input.DisplayName),
			})
		}
		generated = append(generated, GeneratedDynamicComboOption{
			Key:    option.Key,
			Inputs: inputs,
		})
	}
	return generated
}

func buildGeneratedNodeSchemaHiddenInputs(inputs []HiddenInput) []GeneratedNodeSchemaHiddenInput {
	filtered := make([]GeneratedNodeSchemaHiddenInput, 0, len(inputs))
	for _, input := range inputs {
		filtered = append(filtered, GeneratedNodeSchemaHiddenInput(input))
	}
	return filtered
}

func buildGeneratedNodeSchemaOutputs(outputs []Output) []GeneratedNodeSchemaOutput {
	filtered := make([]GeneratedNodeSchemaOutput, 0, len(outputs))
	for _, output := range outputs {
		filtered = append(filtered, GeneratedNodeSchemaOutput(output))
	}
	return filtered
}

func buildResourceData(node Node) (ResourceData, []string) {
	var warnings []string

	rawSuffix := strings.TrimPrefix(node.TerraformResourceName, "comfyui_")
	suffix := sanitizeName(rawSuffix)
	structName := toPascalCase(suffix)

	desc := buildNodeDescription(node)
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
	return buildSchemaAttribute(inp, tfName, rd, inp.Required, true)
}

func buildSchemaAttribute(inp Input, tfName string, rd *ResourceData, required bool, emitValidators bool) string {
	if isDynamicComboType(inp.Type) {
		return buildDynamicComboAttribute(inp, tfName, rd, required, emitValidators)
	}

	var b strings.Builder

	attrType := tfAttributeType(inp.Type)

	fmt.Fprintf(&b, "%q: %s{\n", tfName, attrType)

	fmt.Fprintf(&b, "\t\t\t\tMarkdownDescription: %q,\n", buildInputDescription(inp))

	if required {
		b.WriteString("\t\t\t\tRequired: true,\n")
	} else {
		b.WriteString("\t\t\t\tOptional: true,\n")
	}

	appendInputValidators(&b, inp, rd, emitValidators)

	b.WriteString("\t\t\t},")
	return b.String()
}

func buildDynamicComboAttribute(inp Input, tfName string, rd *ResourceData, required bool, _ bool) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%q: schema.SingleNestedAttribute{\n", tfName)
	fmt.Fprintf(&b, "\t\t\t\tMarkdownDescription: %q,\n", buildDynamicComboDescription(inp))
	if required {
		b.WriteString("\t\t\t\tRequired: true,\n")
	} else {
		b.WriteString("\t\t\t\tOptional: true,\n")
	}
	b.WriteString("\t\t\t\tAttributes: map[string]schema.Attribute{\n")
	b.WriteString(buildDynamicComboSelectionAttribute())
	usedChildNames := map[string]bool{dynamicComboSelectionKey: true}
	for _, child := range mergedDynamicComboInputs(inp.DynamicComboOptions) {
		childName := nextDynamicComboChildName(child.Name, usedChildNames)
		b.WriteString(indentSchemaAttribute(buildSchemaAttribute(child, childName, rd, false, false), "\t\t\t\t\t"))
		b.WriteString("\n")
	}
	b.WriteString("\t\t\t\t},\n")
	b.WriteString("\t\t\t},")
	return b.String()
}

const dynamicComboSelectionKey = "selection"

func buildDynamicComboSelectionAttribute() string {
	var b strings.Builder

	fmt.Fprintf(&b, "%q: schema.StringAttribute{\n", dynamicComboSelectionKey)
	b.WriteString("\t\t\t\tRequired: true,\n")
	b.WriteString("\t\t\t\tMarkdownDescription: \"Selected DynamicCombo option key.\",\n")
	b.WriteString("\t\t\t},\n")
	return b.String()
}

func buildDynamicComboDescription(inp Input) string {
	return buildInputDescription(inp) + " Set `selection` to choose the active option. The nested fields below are a union across all options; the provider validates which child fields are required and allowed for the selected option."
}

func appendInputValidators(b *strings.Builder, inp Input, rd *ResourceData, emitValidators bool) {
	if !emitValidators {
		return
	}

	switch inp.Type {
	case "INT":
		if inp.Min != nil && inp.Max != nil {
			rd.NeedsInt64Validator = true
			minVal := clampInt64(inp.Min.Float64)
			maxVal := clampInt64(inp.Max.Float64)
			fmt.Fprintf(b, "\t\t\t\tValidators: []validator.Int64{\n\t\t\t\t\tint64validator.Between(%d, %d),\n\t\t\t\t},\n", minVal, maxVal)
		}
	case "FLOAT":
		if inp.Min != nil && inp.Max != nil {
			rd.NeedsFloat64Validator = true
			fmt.Fprintf(b, "\t\t\t\tValidators: []validator.Float64{\n\t\t\t\t\tfloat64validator.Between(%v, %v),\n\t\t\t\t},\n", inp.Min.Float64, inp.Max.Float64)
		}
	case "COMBO":
		opts := inp.StringOptions()
		if len(opts) > 0 && (inp.DynamicOptions == nil || !*inp.DynamicOptions) {
			rd.NeedsStringValidator = true
			b.WriteString("\t\t\t\tValidators: []validator.String{\n\t\t\t\t\tstringvalidator.OneOf(\n")
			for _, opt := range opts {
				fmt.Fprintf(b, "\t\t\t\t\t\t%q,\n", opt)
			}
			b.WriteString("\t\t\t\t\t),\n\t\t\t\t},\n")
		}
	}
}

func buildOutputAttribute(out Output, tfName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%q: schema.StringAttribute{\n", tfName)
	fmt.Fprintf(&b, "\t\t\t\tMarkdownDescription: \"Output: %s (slot %d).\",\n", out.Type, out.SlotIndex)
	b.WriteString("\t\t\t\tComputed: true,\n")
	b.WriteString("\t\t\t\tPlanModifiers: []planmodifier.String{\n\t\t\t\t\tstringplanmodifier.UseStateForUnknown(),\n\t\t\t\t},\n")
	b.WriteString("\t\t\t},")
	return b.String()
}

func mergedDynamicComboInputs(options []DynamicComboOption) []Input {
	if len(options) == 0 {
		return nil
	}

	merged := make(map[string]Input)
	order := make([]string, 0)
	for _, option := range options {
		for _, input := range option.Inputs {
			key := sanitizeName(input.Name)
			if existing, ok := merged[key]; ok {
				merged[key] = mergeDynamicComboInput(existing, input)
				continue
			}
			merged[key] = input
			order = append(order, key)
		}
	}
	sort.Strings(order)

	result := make([]Input, 0, len(order))
	for _, key := range order {
		result = append(result, merged[key])
	}
	return result
}

func mergeDynamicComboInput(existing, next Input) Input {
	merged := existing
	merged.DynamicOptions = mergeBoolPointer(existing.DynamicOptions, next.DynamicOptions)
	merged.DynamicOptionsSource = mergeStringPointer(existing.DynamicOptionsSource, next.DynamicOptionsSource)
	merged.Default = mergePreferredValue(existing.Default, next.Default)
	merged.Min = mergePreferredNumber(existing.Min, next.Min)
	merged.Max = mergePreferredNumber(existing.Max, next.Max)
	merged.Step = mergePreferredNumber(existing.Step, next.Step)
	merged.RawOptions = mergeRawOptions(existing.RawOptions, next.RawOptions)
	merged.mergedDefaultValues = mergeUniqueStrings(existing.mergedDefaultValues, defaultMetadataValues(existing), defaultMetadataValues(next))
	merged.mergedRangeHints = mergeUniqueStrings(existing.mergedRangeHints, rangeMetadataValues(existing), rangeMetadataValues(next))
	merged.mergedStepValues = mergeUniqueStrings(existing.mergedStepValues, stepMetadataValues(existing), stepMetadataValues(next))
	merged.mergedDynamicOptionSources = mergeUniqueStrings(existing.mergedDynamicOptionSources, dynamicOptionSourceMetadataValues(existing), dynamicOptionSourceMetadataValues(next))
	return merged
}

func mergeRawOptions(existing, next []interface{}) []interface{} {
	if len(existing) == 0 {
		return append([]interface{}(nil), next...)
	}
	if len(next) == 0 {
		return append([]interface{}(nil), existing...)
	}

	values := make([]interface{}, 0, len(existing)+len(next))
	seen := make(map[string]bool, len(existing)+len(next))
	appendUnique := func(options []interface{}) {
		for _, option := range options {
			key := fmt.Sprintf("%v", option)
			if seen[key] {
				continue
			}
			seen[key] = true
			values = append(values, option)
		}
	}
	appendUnique(existing)
	appendUnique(next)
	return values
}

func buildNodeDescription(node Node) string {
	desc := fmt.Sprintf("ComfyUI %s node", node.ClassName)
	if node.DisplayName != nil && *node.DisplayName != "" {
		desc = fmt.Sprintf("ComfyUI %s node — %s", node.ClassName, *node.DisplayName)
	}
	if node.Description != nil && *node.Description != "" {
		desc = strings.TrimSpace(*node.Description)
	}

	parts := []string{desc}
	if node.Category != "" {
		parts = append(parts, "["+node.Category+"]")
	}
	if hidden := formatHiddenInputs(node.HiddenInputs); hidden != "" {
		parts = append(parts, "Hidden runtime inputs: "+hidden+".")
	}
	if source := formatSourceInfo(node.Source); source != "" {
		parts = append(parts, source)
	}

	return strings.Join(parts, " ")
}

func buildInputDescription(inp Input) string {
	parts := []string{fmt.Sprintf("Input: %s.", inp.Type)}
	if inp.DisplayName != nil && *inp.DisplayName != "" && *inp.DisplayName != inp.Name {
		parts = append(parts, sentence("Display name", *inp.DisplayName))
	}
	if inp.IsLinkType {
		parts = append(parts, "Link input.")
	}
	if len(inp.mergedDefaultValues) > 1 {
		parts = append(parts, fmt.Sprintf("Defaults vary by selection: %s.", strings.Join(inp.mergedDefaultValues, ", ")))
	} else if inp.Default != nil {
		parts = append(parts, fmt.Sprintf("Default: %s.", formatValue(inp.Default)))
	}
	if len(inp.mergedRangeHints) > 1 {
		parts = append(parts, fmt.Sprintf("Allowed ranges vary by selection: %s.", strings.Join(inp.mergedRangeHints, "; ")))
	} else if rangeHint := formatRangeHint(inp.Min, inp.Max); rangeHint != "" {
		parts = append(parts, rangeHint)
	}
	if len(inp.mergedStepValues) > 1 {
		parts = append(parts, fmt.Sprintf("Steps vary by selection: %s.", strings.Join(inp.mergedStepValues, ", ")))
	} else if inp.Step != nil {
		parts = append(parts, fmt.Sprintf("Step: %s.", inp.Step.Raw))
	}
	if optionsHint := formatOptionsHint(inp.StringOptions()); optionsHint != "" {
		parts = append(parts, optionsHint)
	}
	if inp.DynamicOptions != nil && *inp.DynamicOptions {
		if len(inp.mergedDynamicOptionSources) > 0 {
			parts = append(parts, fmt.Sprintf("Dynamic options are resolved by ComfyUI at runtime from one of: %s.", strings.Join(inp.mergedDynamicOptionSources, "; ")))
		} else if inp.DynamicOptionsSource != nil && *inp.DynamicOptionsSource != "" {
			if source, ok := conciseDynamicOptionsSource(*inp.DynamicOptionsSource); ok {
				parts = append(parts, fmt.Sprintf("Dynamic options are resolved by ComfyUI at runtime from: %s.", source))
			} else {
				parts = append(parts, "Dynamic options are resolved by ComfyUI at runtime.")
			}
		} else {
			parts = append(parts, "Dynamic options are resolved by ComfyUI at runtime.")
		}
	}
	if inp.Multiline != nil && *inp.Multiline {
		parts = append(parts, "Supports multiline text.")
	}
	if inp.Tooltip != nil && *inp.Tooltip != "" {
		parts = append(parts, sentence("Tooltip", *inp.Tooltip))
	}

	return strings.Join(parts, " ")
}

func formatHiddenInputs(hiddenInputs []HiddenInput) string {
	if len(hiddenInputs) == 0 {
		return ""
	}

	values := make([]string, 0, len(hiddenInputs))
	for _, hidden := range hiddenInputs {
		values = append(values, fmt.Sprintf("%s (%s)", hidden.Name, hidden.Type))
	}

	return strings.Join(values, ", ")
}

func formatSourceInfo(source SourceInfo) string {
	location := strings.TrimSpace(source.File)
	if location != "" && source.LineNumber != nil {
		location = fmt.Sprintf("%s:%d", location, *source.LineNumber)
	}

	switch {
	case location != "" && source.Pattern != "":
		return fmt.Sprintf("Source: %s (%s).", location, source.Pattern)
	case location != "":
		return fmt.Sprintf("Source: %s.", location)
	case source.Pattern != "":
		return fmt.Sprintf("Source pattern: %s.", source.Pattern)
	default:
		return ""
	}
}

func formatRangeHint(min, max *NumberValue) string {
	switch {
	case min != nil && max != nil:
		return fmt.Sprintf("Allowed range: %s to %s.", min.Raw, max.Raw)
	case min != nil:
		return fmt.Sprintf("Minimum value: %s.", min.Raw)
	case max != nil:
		return fmt.Sprintf("Maximum value: %s.", max.Raw)
	default:
		return ""
	}
}

func formatOptionsHint(options []string) string {
	if len(options) == 0 {
		return ""
	}

	quoted := make([]string, 0, len(options))
	for _, option := range options {
		quoted = append(quoted, strconv.Quote(option))
	}
	return fmt.Sprintf("Options: %s.", strings.Join(quoted, ", "))
}

func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strconv.Quote(v)
	case float64:
		return formatNumber(v)
	case float32:
		return formatNumber(float64(v))
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func formatOptionalValue(value interface{}) string {
	if value == nil {
		return ""
	}
	return formatValue(value)
}

func formatOptionalNumber(value *NumberValue) string {
	if value == nil {
		return ""
	}
	return value.Raw
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func conciseDynamicOptionsSource(source string) (string, bool) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return "", false
	}
	if strings.ContainsAny(trimmed, "[]{}\n") || len(trimmed) > 80 {
		return "", false
	}
	return trimmed, true
}

func nextDynamicComboChildName(name string, used map[string]bool) string {
	candidate := sanitizeName(name)
	if candidate == dynamicComboSelectionKey {
		candidate += "_value"
	}

	base := candidate
	suffix := 2
	for used[candidate] {
		candidate = fmt.Sprintf("%s_%d", base, suffix)
		suffix++
	}
	used[candidate] = true
	return candidate
}

func indentSchemaAttribute(definition, indent string) string {
	if definition == "" {
		return ""
	}

	lines := strings.Split(definition, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

func mergeBoolPointer(existing, next *bool) *bool {
	switch {
	case existing == nil && next == nil:
		return nil
	case existing != nil && next == nil:
		value := *existing
		return &value
	case existing == nil && next != nil:
		value := *next
		return &value
	default:
		value := *existing || *next
		return &value
	}
}

func mergeStringPointer(existing, next *string) *string {
	if strings.TrimSpace(stringValue(next)) == "" {
		return existing
	}
	if strings.TrimSpace(stringValue(existing)) == "" {
		value := *next
		return &value
	}
	if *existing == *next {
		value := *existing
		return &value
	}
	value := *existing
	return &value
}

func mergePreferredValue(existing, next interface{}) interface{} {
	switch {
	case existing != nil:
		return existing
	default:
		return next
	}
}

func mergePreferredNumber(existing, next *NumberValue) *NumberValue {
	switch {
	case existing != nil:
		value := *existing
		return &value
	case next != nil:
		value := *next
		return &value
	default:
		return nil
	}
}

func mergeUniqueStrings(existing []string, groups ...[]string) []string {
	seen := make(map[string]bool, len(existing))
	values := make([]string, 0, len(existing))

	for _, value := range existing {
		if strings.TrimSpace(value) == "" || seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}

	for _, group := range groups {
		for _, value := range group {
			if strings.TrimSpace(value) == "" || seen[value] {
				continue
			}
			seen[value] = true
			values = append(values, value)
		}
	}

	return values
}

func defaultMetadataValues(inp Input) []string {
	if inp.Default == nil {
		return nil
	}
	return []string{formatValue(inp.Default)}
}

func rangeMetadataValues(inp Input) []string {
	if len(inp.mergedRangeHints) > 0 {
		return inp.mergedRangeHints
	}
	if hint := rangeMetadataValue(inp.Min, inp.Max); hint != "" {
		return []string{hint}
	}
	return nil
}

func rangeMetadataValue(min, max *NumberValue) string {
	switch {
	case min != nil && max != nil:
		return fmt.Sprintf("%s to %s", min.Raw, max.Raw)
	case min != nil:
		return fmt.Sprintf("min %s", min.Raw)
	case max != nil:
		return fmt.Sprintf("max %s", max.Raw)
	default:
		return ""
	}
}

func stepMetadataValues(inp Input) []string {
	if inp.Step == nil {
		return nil
	}
	return []string{inp.Step.Raw}
}

func dynamicOptionSourceMetadataValues(inp Input) []string {
	if len(inp.mergedDynamicOptionSources) > 0 {
		return inp.mergedDynamicOptionSources
	}
	if inp.DynamicOptionsSource == nil || strings.TrimSpace(*inp.DynamicOptionsSource) == "" {
		return nil
	}
	if source, ok := conciseDynamicOptionsSource(*inp.DynamicOptionsSource); ok {
		return []string{source}
	}
	return nil
}

func sentence(label, value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	if !strings.HasSuffix(text, ".") && !strings.HasSuffix(text, "!") && !strings.HasSuffix(text, "?") {
		text += "."
	}
	return fmt.Sprintf("%s: %s", label, text)
}
