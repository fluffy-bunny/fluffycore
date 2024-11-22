package utils

import "github.com/invopop/jsonschema"

func SchemaFor[T any](t T) string {
	schema := jsonschema.Reflect(t)
	data, _ := schema.MarshalJSON()
	return string(data)
}
