package utils

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// Arrays used to panic: reflect.Value.IsNil on an array.
func TestIsNil_Arrays(t *testing.T) {
	require.False(t, IsNil([2]int{}))
	require.True(t, IsEmptyOrNil([2]int{}))
	require.False(t, IsEmptyOrNil([2]int{0, 1}))
	require.True(t, IsEmptyOrNil([0]string{}))
}

func TestIsNil_Basics(t *testing.T) {
	var p *int
	var m map[string]int
	require.True(t, IsNil(nil))
	require.True(t, IsNil(p))
	require.True(t, IsNil(m))
	require.False(t, IsNil(0))
	require.True(t, IsEmptyOrNil(""))
	require.True(t, IsEmptyOrNil([]int{}))
	require.False(t, IsEmptyOrNil("x"))
}

func TestIsZero_UnexportedFields(t *testing.T) {
	type s struct {
		a int
		B string
	}
	require.True(t, IsZero(reflect.ValueOf(s{})))
	require.False(t, IsZero(reflect.ValueOf(s{a: 1})))
}
