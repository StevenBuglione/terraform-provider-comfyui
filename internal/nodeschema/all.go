package nodeschema

func All() []GeneratedNodeSchema {
	result := make([]GeneratedNodeSchema, 0, len(generatedNodeSchemas))
	for _, schema := range generatedNodeSchemas {
		result = append(result, schema)
	}
	return result
}
