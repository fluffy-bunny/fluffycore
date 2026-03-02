package utils

import "github.com/invopop/jsonschema"

// SchemaFor generates a JSON Schema string for the given type using reflection.
func SchemaFor[T any](t T) string {
	schema := jsonschema.Reflect(t)
	data, err := schema.MarshalJSON()
	if err != nil {
		return "{}"
	}
	return string(data)
}
