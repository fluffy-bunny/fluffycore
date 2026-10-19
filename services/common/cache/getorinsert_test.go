package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetOrInsertE_ReturnsAdderError(t *testing.T) {
	c := NewMemoryCache()
	boom := errors.New("backend down")
	_, err := c.GetOrInsertE("k", func() (interface{}, time.Duration, error) { return nil, 0, boom })
	require.ErrorIs(t, err, boom, "the adder's error used to be swallowed")
	_, err = c.Get("k")
	require.Error(t, err, "a failed adder must not cache anything")
}

// Concurrent misses on one key used to run the adder once per caller.
func TestGetOrInsertE_ConcurrentMissesRunAdderOnce(t *testing.T) {
	c := NewMemoryCache()
	var calls atomic.Int32
	release := make(chan struct{})
	adder := func() (interface{}, time.Duration, error) {
		calls.Add(1)
		<-release // hold the first caller so the others pile up behind it
		return "v", time.Minute, nil
	}

	const n = 50
	var wg sync.WaitGroup
	results := make([]interface{}, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], _ = c.GetOrInsertE("hot", adder)
		}(i)
	}
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()

	require.EqualValues(t, 1, calls.Load())
	for _, r := range results {
		require.Equal(t, "v", r)
	}
}
