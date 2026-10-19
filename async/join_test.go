package async

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func joinWithin(t *testing.T, pr *PromiseResponse[*FutureResponse]) (*FutureResponse, error) {
	t.Helper()
	type result struct {
		v   *FutureResponse
		err error
	}
	ch := make(chan result, 1)
	go func() { v, err := pr.Future.Join(); ch <- result{v, err} }()
	select {
	case r := <-ch:
		return r.v, r.err
	case <-time.After(5 * time.Second):
		t.Fatal("Join deadlocked")
		return nil, nil
	}
}

// Join without polling IsComplete first used to block forever.
func TestExecuteAsync_JoinWithoutIsComplete(t *testing.T) {
	v, err := joinWithin(t, ExecuteAsync(func() (interface{}, error) { return 42, nil }))
	require.NoError(t, err)
	require.Equal(t, 42, v.Value)

	boom := errors.New("boom")
	_, err = joinWithin(t, ExecuteAsync(func() (interface{}, error) { return nil, boom }))
	require.ErrorIs(t, err, boom)

	_, err = joinWithin(t, ExecuteAsync(func() (interface{}, error) { panic("oops") }))
	require.ErrorContains(t, err, "panic: oops")
}

// IsComplete used to consume a one-shot signal: after the first true it said false again.
func TestExecuteAsync_IsCompleteStaysTrue(t *testing.T) {
	pr := ExecuteAsync(func() (interface{}, error) { return "x", nil })
	_, err := joinWithin(t, pr)
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		require.True(t, pr.IsComplete())
	}
}
