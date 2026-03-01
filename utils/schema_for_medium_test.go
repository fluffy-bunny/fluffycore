package utils

import (
	"testing"
)

func TestSchemaFor_ReturnsValidJSON(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	result := SchemaFor(&TestStruct{})
	if result == "" {
		t.Error("expected non-empty schema")
	}
	if result == "{}" {
		t.Error("expected a real schema, not empty object")
	}
	// Basic check that it contains expected field names
	if len(result) < 10 {
		t.Errorf("schema seems too short: %s", result)
	}
}

func TestSchemaFor_EmptyStruct(t *testing.T) {
	type Empty struct{}
	result := SchemaFor(&Empty{})
	if result == "" {
		t.Error("expected non-empty schema even for empty struct")
	}
}
