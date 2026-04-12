package artifacts

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/nodeschema"
)

type TerraformIR struct {
	Resources []TerraformResource `json:"resources"`
	Workflow  TerraformWorkflow   `json:"workflow"`
}

type TerraformResource struct {
	Type       string               `json:"type"`
	Name       string               `json:"name"`
	NodeID     string               `json:"node_id"`
	ClassType  string               `json:"class_type"`
	Attributes []TerraformAttribute `json:"attributes"`
}

type TerraformAttribute struct {
	Name       string      `json:"name"`
	Literal    interface{} `json:"literal,omitempty"`
	Expression string      `json:"expression,omitempty"`
}

type TerraformWorkflow struct {
	Type              string   `json:"type"`
	Name              string   `json:"name"`
	NodeIDExpressions []string `json:"node_id_expressions"`
}

func BuildTerraformIRFromPrompt(prompt *Prompt) (*TerraformIR, *TranslationReport, error) {
	if prompt == nil {
		return nil, nil, fmt.Errorf("prompt is nil")
	}
	if len(prompt.Nodes) == 0 {
		return nil, nil, fmt.Errorf("prompt must contain at least one node")
	}

	nodeIDs := sortedPromptNodeIDs(prompt.Nodes)
	resourceNames := make(map[string]string, len(nodeIDs))
	nodeSchemas := make(map[string]nodeschema.GeneratedNodeSchema, len(nodeIDs))
	ir := &TerraformIR{
		Resources: make([]TerraformResource, 0, len(nodeIDs)),
		Workflow: TerraformWorkflow{
			Type:              "comfyui_workflow",
			Name:              "workflow",
			NodeIDExpressions: make([]string, 0, len(nodeIDs)),
		},
	}
	report := NewTranslationReport()

	for _, nodeID := range nodeIDs {
		node := prompt.Nodes[nodeID]
		schema, ok := nodeschema.LookupGeneratedNodeSchema(node.ClassType)
		if !ok {
			return nil, nil, fmt.Errorf("no generated node schema exists for %q", node.ClassType)
		}
		nodeSchemas[nodeID] = schema
		resourceName := terraformResourceNameForNodeID(nodeID)
		resourceNames[nodeID] = resourceName

		attributes, err := terraformAttributesForNode(node, schema, resourceNames, nodeSchemas)
		if err != nil {
			return nil, nil, err
		}

		ir.Resources = append(ir.Resources, TerraformResource{
			Type:       schema.TerraformType,
			Name:       resourceName,
			NodeID:     nodeID,
			ClassType:  node.ClassType,
			Attributes: attributes,
		})
		ir.Workflow.NodeIDExpressions = append(ir.Workflow.NodeIDExpressions, fmt.Sprintf("%s.%s.id", schema.TerraformType, resourceName))
		report.AddPreservedField(fmt.Sprintf("prompt.%s.class_type", nodeID))
		report.AddSynthesizedField(fmt.Sprintf("terraform.resources[%s].name", nodeID))
	}

	sort.Slice(ir.Resources, func(i, j int) bool {
		left, _ := strconv.Atoi(ir.Resources[i].NodeID)
		right, _ := strconv.Atoi(ir.Resources[j].NodeID)
		return left < right
	})

	return ir, report, nil
}

func terraformAttributesForNode(node PromptNode, schema nodeschema.GeneratedNodeSchema, resourceNames map[string]string, schemas map[string]nodeschema.GeneratedNodeSchema) ([]TerraformAttribute, error) {
	orderedNames := orderedTerraformInputNames(node, schema)
	attributes := make([]TerraformAttribute, 0, len(orderedNames))
	for _, inputName := range orderedNames {
		value, ok := node.Inputs[inputName]
		if !ok {
			continue
		}

		if sourceNodeID, sourceSlot, linked := promptLinkValue(value); linked {
			sourceSchema, ok := schemas[sourceNodeID]
			if !ok {
				return nil, fmt.Errorf("missing schema for linked source node %q", sourceNodeID)
			}
			if sourceSlot < 0 || sourceSlot >= len(sourceSchema.Outputs) {
				return nil, fmt.Errorf("linked source node %q slot %d out of range", sourceNodeID, sourceSlot)
			}
			outputAttr := sanitizeTerraformAttributeName(sourceSchema.Outputs[sourceSlot].Name) + "_output"
			attributes = append(attributes, TerraformAttribute{
				Name:       sanitizeTerraformAttributeName(inputName),
				Expression: fmt.Sprintf("%s.%s.%s", sourceSchema.TerraformType, resourceNames[sourceNodeID], outputAttr),
			})
			continue
		}

		attributes = append(attributes, TerraformAttribute{
			Name:    sanitizeTerraformAttributeName(inputName),
			Literal: value,
		})
	}
	return attributes, nil
}

func orderedTerraformInputNames(node PromptNode, schema nodeschema.GeneratedNodeSchema) []string {
	ordered := make([]string, 0, len(node.Inputs))
	seen := make(map[string]bool, len(node.Inputs))
	for _, input := range append(append([]nodeschema.GeneratedNodeSchemaInput{}, schema.RequiredInputs...), schema.OptionalInputs...) {
		if _, ok := node.Inputs[input.Name]; ok {
			ordered = append(ordered, input.Name)
			seen[input.Name] = true
		}
	}
	extras := make([]string, 0)
	for inputName := range node.Inputs {
		if !seen[inputName] {
			extras = append(extras, inputName)
		}
	}
	sort.Strings(extras)
	return append(ordered, extras...)
}

func terraformResourceNameForNodeID(nodeID string) string {
	sanitized := sanitizeTerraformAttributeName(nodeID)
	if sanitized == "" {
		sanitized = "node"
	}
	if sanitized[0] >= '0' && sanitized[0] <= '9' {
		return "node_" + sanitized
	}
	return sanitized
}

func sanitizeTerraformAttributeName(name string) string {
	var builder strings.Builder
	for _, r := range name {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r + ('a' - 'A'))
		default:
			builder.WriteRune('_')
		}
	}
	result := strings.Trim(builder.String(), "_")
	result = strings.ReplaceAll(result, "__", "_")
	return result
}

func (ir *TerraformIR) JSON() (string, error) {
	if ir == nil {
		return "", fmt.Errorf("terraform ir is nil")
	}
	data, err := json.MarshalIndent(ir, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal terraform ir: %w", err)
	}
	return string(data), nil
}
