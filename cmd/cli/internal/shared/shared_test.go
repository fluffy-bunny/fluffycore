package shared

import (
	"testing"

	require "github.com/stretchr/testify/require"
)

type MyStringType string

func TestNilStrPtr(t *testing.T) {
	var str string
	var nilStrPtr *string

	// Test with a non-nil string
	str = "test"
	result := NilStrPtr(str)
	require.NotNil(t, result, "Expected non-nil pointer for non-empty string")
	require.Equal(t, str, *result, "Expected pointer to point to the original string")

	res0 := NilStrPtr(&str)
	require.NotNil(t, res0, "Expected non-nil pointer for non-empty string")
	require.Equal(t, *result, *res0, "Expected pointers to point to the same value")

	// Test with a nil string
	result = NilStrPtr(nilStrPtr)
	require.Nil(t, result, "Expected nil pointer for nil string")

	myString := MyStringType("example")
	result = NilStrPtr(myString)
	require.NotNil(t, result, "Expected non-nil pointer for MyStringType")

	res1 := NilStrPtr(&myString)
	require.NotNil(t, res1, "Expected non-nil pointer for MyStringType")
	require.Equal(t, *result, *res1, "Expected pointers to point to the same value")

	res2 := NilStrPtr(result)
	require.NotNil(t, res2, "Expected non-nil pointer for MyStringType")
	require.Equal(t, *result, *res2, "Expected pointers to point to the same value")
}
