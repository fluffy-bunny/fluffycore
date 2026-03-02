package async

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestExecuteAsync_PanicRecovery verifies that recover() works correctly
// inside the goroutine (was CRITICAL: recover() was called outside defer,
// making it a no-op that never caught panics).
func TestExecuteAsync_PanicRecovery(t *testing.T) {
	pr := ExecuteAsync(func() (interface{}, error) {
		panic("test panic")
	})

	// Wait for completion
	for {
		if pr.IsComplete() {
			break
		}
	}

	future := pr.Future
	v, err := future.Join()
	// The panic should be caught and returned as an error.
	// The error may be in err (from Map) or in v.Err (from FutureResponse).
	if err != nil {
		require.Contains(t, err.Error(), "panic: test panic")
	} else {
		require.NotNil(t, v)
		require.Error(t, v.Err)
		require.Contains(t, v.Err.Error(), "panic: test panic")
		require.Nil(t, v.Value)
	}
}

// TestExecuteAsync_NormalExecution verifies normal (non-panic) execution
// still works after the defer/recover fix.
func TestExecuteAsync_NormalExecution(t *testing.T) {
	pr := ExecuteAsync(func() (interface{}, error) {
		return "hello", nil
	})

	for {
		if pr.IsComplete() {
			break
		}
	}

	future := pr.Future
	v, err := future.Join()
	require.NoError(t, err)
	require.Equal(t, "hello", v.Value)
}

// TestExecuteAsync_NilPanic verifies that nil panics are caught too.
func TestExecuteAsync_NilPanic(t *testing.T) {
	pr := ExecuteAsync(func() (interface{}, error) {
		panic(nil)
	})

	for {
		if pr.IsComplete() {
			break
		}
	}

	future := pr.Future
	v, err := future.Join()
	// nil panics should still result in an error
	if err != nil {
		require.Error(t, err)
	} else {
		require.NotNil(t, v)
		require.Error(t, v.Err)
		require.Nil(t, v.Value)
	}
}
