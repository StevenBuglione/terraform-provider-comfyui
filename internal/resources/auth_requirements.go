package resources

import (
	"github.com/StevenBuglione/terraform-provider-comfyui/internal/nodeschema"
)

// AuthRequirement represents a normalized auth family requirement
type AuthRequirement struct {
	// Family is the normalized auth family name (e.g., "comfy_org")
	Family string

	// RequiredFields lists the hidden auth field names required by this family
	RequiredFields []string

	// TriggeringNodes lists the node class_type values that triggered this requirement
	TriggeringNodes []string
}

// ExtractAuthRequirements inspects an assembled prompt and returns auth requirements
// based on the hidden auth fields declared in the node schemas.
//
// It processes the prompt map, extracts class_type values, looks up their schemas,
// examines hidden inputs, normalizes them into auth families, and deduplicates.
//
// Returns a slice of AuthRequirement, one per unique auth family.
func ExtractAuthRequirements(prompt map[string]interface{}) ([]AuthRequirement, error) {
	// Map from auth family name to requirement being built
	familyMap := make(map[string]*AuthRequirement)

	// Iterate over all nodes in the prompt
	for _, nodeData := range prompt {
		// Safely extract the node map
		nodeMap, ok := nodeData.(map[string]interface{})
		if !ok {
			// Skip malformed nodes
			continue
		}

		// Extract class_type
		classTypeRaw, ok := nodeMap["class_type"]
		if !ok {
			// Skip nodes without class_type
			continue
		}

		classType, ok := classTypeRaw.(string)
		if !ok {
			// Skip if class_type is not a string
			continue
		}

		// Look up the node schema
		schema, found := nodeschema.LookupGeneratedNodeSchema(classType)
		if !found {
			// Node schema not found; skip
			continue
		}

		// Check if this node has hidden auth inputs
		if len(schema.HiddenInputs) == 0 {
			continue
		}

		// Extract auth fields from hidden inputs
		authFields := extractAuthFields(schema.HiddenInputs)
		if len(authFields) == 0 {
			// No auth fields found
			continue
		}

		// Normalize auth fields into families
		family, fields := normalizeAuthFamily(authFields)
		if family == "" {
			// No recognized auth family
			continue
		}

		// Add or update the family requirement
		req, exists := familyMap[family]
		if !exists {
			req = &AuthRequirement{
				Family:          family,
				RequiredFields:  fields,
				TriggeringNodes: []string{},
			}
			familyMap[family] = req
		}

		// Add this node to the triggering nodes
		if !stringInSlice(req.TriggeringNodes, classType) {
			req.TriggeringNodes = append(req.TriggeringNodes, classType)
		}
	}

	// Convert map to slice
	result := make([]AuthRequirement, 0, len(familyMap))
	for _, req := range familyMap {
		result = append(result, *req)
	}

	return result, nil
}

// extractAuthFields filters hidden inputs for auth-related fields
func extractAuthFields(hiddenInputs []nodeschema.GeneratedNodeSchemaHiddenInput) []string {
	var authFields []string
	for _, input := range hiddenInputs {
		// Check if this is an auth field based on naming patterns
		if isAuthField(input.Name) {
			authFields = append(authFields, input.Name)
		}
	}
	return authFields
}

func stringInSlice(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isAuthField determines if a field name represents an auth field
func isAuthField(fieldName string) bool {
	// Currently, we recognize fields ending with _comfy_org
	// This can be extended for future auth families
	switch fieldName {
	case "auth_token_comfy_org", "api_key_comfy_org":
		return true
	default:
		return false
	}
}

// normalizeAuthFamily takes a list of auth field names and returns
// the normalized family name and the canonical list of required fields
func normalizeAuthFamily(authFields []string) (family string, fields []string) {
	// Check if any field belongs to the comfy_org family
	hasComfyOrg := false
	for _, field := range authFields {
		if field == "auth_token_comfy_org" || field == "api_key_comfy_org" {
			hasComfyOrg = true
			break
		}
	}

	if hasComfyOrg {
		// Return the comfy_org family with its canonical field list
		return "comfy_org", []string{"auth_token_comfy_org", "api_key_comfy_org"}
	}

	// No recognized family
	return "", nil
}
