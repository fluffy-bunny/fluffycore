package singletonenvironmentservices

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// Looking up one variable used to make every later GetEnvironmentMap return only that one.
func TestGetEnvironmentMap_NotPartialAfterSingleLookup(t *testing.T) {
	t.Setenv("FLUFFY_ENV_TEST_A", "a")
	t.Setenv("FLUFFY_ENV_TEST_B", "b")
	ctx := context.Background()
	s := &service{}

	v, ok, err := s.GetEnvironmentVariable(ctx, "FLUFFY_ENV_TEST_A")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "a", v)

	m, err := s.GetEnvironmentMap(ctx)
	require.NoError(t, err)
	require.Equal(t, "b", m["FLUFFY_ENV_TEST_B"])
	require.GreaterOrEqual(t, len(m), len(os.Environ())-5) // Windows has a few "=C:"-style entries we skip

	// Still whole after a clear + single lookup.
	require.NoError(t, s.ClearCache(ctx))
	_, _, _ = s.GetEnvironmentVariable(ctx, "FLUFFY_ENV_TEST_A")
	m, err = s.GetEnvironmentMap(ctx)
	require.NoError(t, err)
	require.Equal(t, "b", m["FLUFFY_ENV_TEST_B"])
}
