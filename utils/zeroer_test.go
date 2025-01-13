package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestZeroerFunc(t *testing.T) {

	type MYFUNC func() error
	var myfunc MYFUNC
	require.True(t, IsNil(myfunc))
	require.True(t, IsEmptyOrNil(myfunc))

	myfunc = func() error {
		return nil
	}
	require.False(t, IsNil(myfunc))
	require.False(t, IsEmptyOrNil(myfunc))

}

func TestZeroerStruct(t *testing.T) {

	type MYSTRUCT struct {
		Name string
	}
	var mystruct *MYSTRUCT
	require.True(t, IsNil(mystruct))
	require.True(t, IsEmptyOrNil(mystruct))

	mystruct = &MYSTRUCT{
		Name: "test",
	}
	require.False(t, IsNil(mystruct))
	require.False(t, IsEmptyOrNil(mystruct))

}
