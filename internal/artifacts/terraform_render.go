package artifacts

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func RenderTerraformHCL(ir *TerraformIR) (string, error) {
	if ir == nil {
		return "", fmt.Errorf("terraform ir is nil")
	}

	var builder strings.Builder
	for index, resource := range ir.Resources {
		if index > 0 {
			builder.WriteString("\n")
		}
		renderTerraformResource(&builder, resource)
	}
	if len(ir.Resources) > 0 {
		builder.WriteString("\n")
	}
	renderTerraformWorkflow(&builder, ir.Workflow)
	return builder.String(), nil
}

func renderTerraformResource(builder *strings.Builder, resource TerraformResource) {
	fmt.Fprintf(builder, "resource %q %q {\n", resource.Type, resource.Name)
	for _, attr := range sortedTerraformAttributes(resource.Attributes) {
		if attr.Expression != "" {
			fmt.Fprintf(builder, "  %s = %s\n", attr.Name, attr.Expression)
			continue
		}
		fmt.Fprintf(builder, "  %s = %s\n", attr.Name, renderTerraformLiteral(attr.Literal))
	}
	builder.WriteString("}\n")
}

func renderTerraformWorkflow(builder *strings.Builder, workflow TerraformWorkflow) {
	fmt.Fprintf(builder, "resource %q %q {\n", workflow.Type, workflow.Name)
	builder.WriteString("  node_ids = [\n")
	for _, expr := range workflow.NodeIDExpressions {
		fmt.Fprintf(builder, "    %s,\n", expr)
	}
	builder.WriteString("  ]\n")
	builder.WriteString("}\n")
}

func sortedTerraformAttributes(attributes []TerraformAttribute) []TerraformAttribute {
	sorted := append([]TerraformAttribute(nil), attributes...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

func renderTerraformLiteral(value interface{}) string {
	switch v := value.(type) {
	case string:
		data, _ := json.Marshal(v)
		return string(data)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case json.Number:
		return v.String()
	case float64:
		data, _ := json.Marshal(v)
		return string(data)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case nil:
		return "null"
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}
