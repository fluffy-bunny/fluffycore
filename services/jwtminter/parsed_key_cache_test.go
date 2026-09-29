package jwtminter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsedKey_ParsesOncePerKey(t *testing.T) {
	s := &service{}
	parses := 0
	parse := func(v string) func() (interface{}, error) {
		return func() (interface{}, error) { parses++; return v, nil }
	}
	for i := 0; i < 5; i++ {
		k, err := s.parsedKey("ES256", "pem-1", parse("key-1"))
		require.NoError(t, err)
		require.Equal(t, "key-1", k)
	}
	require.Equal(t, 1, parses, "the PEM used to be re-parsed on every mint")

	k, err := s.parsedKey("ES256", "pem-2", parse("key-2")) // rotated
	require.NoError(t, err)
	require.Equal(t, "key-2", k)
	require.Equal(t, 2, parses)
}
