package grpcclient

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestContextWithTimeout_ExplicitDuration verifies that when an explicit
// duration is provided, it works even if defaultGrpcCallTimeoutInSeconds is nil.
// (Was CRITICAL: nil pointer dereference on *defaultGrpcCallTimeoutInSeconds
// before the len(duration) > 0 check.)
func TestContextWithTimeout_ExplicitDuration_NilDefault(t *testing.T) {
	// Ensure the global default is nil
	old := defaultGrpcCallTimeoutInSeconds
	defaultGrpcCallTimeoutInSeconds = nil
	defer func() { defaultGrpcCallTimeoutInSeconds = old }()

	// This should NOT panic — explicit duration provided
	ctx, cancel := ContextWithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NotNil(t, ctx)
}

// TestContextWithTimeout_DefaultTimeout verifies that the default timeout
// is used when no explicit duration is provided.
func TestContextWithTimeout_DefaultTimeout(t *testing.T) {
	old := defaultGrpcCallTimeoutInSeconds
	timeout := 10
	defaultGrpcCallTimeoutInSeconds = &timeout
	defer func() { defaultGrpcCallTimeoutInSeconds = old }()

	ctx, cancel := ContextWithTimeout(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	require.True(t, ok)
	// Deadline should be roughly 10 seconds from now
	remaining := time.Until(deadline)
	require.Greater(t, remaining.Seconds(), 9.0)
	require.Less(t, remaining.Seconds(), 11.0)
}

// TestContextWithTimeout_NilDefault_NoDuration verifies the panic when
// no default and no duration are provided.
func TestContextWithTimeout_NilDefault_NoDuration(t *testing.T) {
	old := defaultGrpcCallTimeoutInSeconds
	defaultGrpcCallTimeoutInSeconds = nil
	defer func() { defaultGrpcCallTimeoutInSeconds = old }()

	require.Panics(t, func() {
		ContextWithTimeout(context.Background())
	})
}
