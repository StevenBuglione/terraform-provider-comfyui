package nodeschema

func All() []GeneratedNodeSchema {
	result := make([]GeneratedNodeSchema, 0, len(generatedNodeSchemas))
	for _, schema := range generatedNodeSchemas {
		result = append(result, schema)
	}
	return result
}

// RegisterForTest inserts schema into generatedNodeSchemas under key.
// Intended for use in tests only; do not call from production code.
func RegisterForTest(key string, schema GeneratedNodeSchema) {
	generatedNodeSchemas[key] = schema
}
