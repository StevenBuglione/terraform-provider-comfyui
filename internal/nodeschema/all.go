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
//
// NOTE: mutates a package-level map and is not safe for parallel tests.
// Callers must run without t.Parallel() and must pair each call with
// DeleteForTest via t.Cleanup to avoid cross-test residue.
func RegisterForTest(key string, schema GeneratedNodeSchema) {
	generatedNodeSchemas[key] = schema
}

// DeleteForTest removes the entry registered under key from generatedNodeSchemas.
// Intended for use in t.Cleanup callbacks to undo RegisterForTest.
func DeleteForTest(key string) {
	delete(generatedNodeSchemas, key)
}
