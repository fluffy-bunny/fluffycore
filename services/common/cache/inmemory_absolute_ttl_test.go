package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// A regularly-read entry must still expire at its TTL. With ttlcache's default sliding
// expiry, reading this entry every 50ms kept a 150ms-TTL entry alive forever -- which let a
// revoked reference token keep authenticating from the referencetoken middleware's cache.
func TestMemoryCache_TTLIsAbsoluteNotSliding(t *testing.T) {
	c := NewMemoryCache()
	t.Cleanup(func() { _ = c.Close() })
	require.NoError(t, c.SetWithTTL("pat", "claims", 150*time.Millisecond))

	deadline := time.Now().Add(150 * time.Millisecond)
	for time.Now().Before(deadline.Add(-30 * time.Millisecond)) {
		_, err := c.Get("pat")
		require.NoError(t, err, "entry must be present before its TTL")
		time.Sleep(25 * time.Millisecond)
	}
	// Keep reading past the TTL: with sliding expiry these reads would keep it alive.
	for i := 0; i < 6; i++ {
		time.Sleep(25 * time.Millisecond)
		_, _ = c.Get("pat")
	}
	_, err := c.Get("pat")
	require.Error(t, err, "entry must expire at its TTL even though it was read throughout")
}

func TestMemoryCache_GetWithTTLDoesNotExtend(t *testing.T) {
	c := NewMemoryCache()
	t.Cleanup(func() { _ = c.Close() })
	require.NoError(t, c.SetWithTTL("k", "v", 200*time.Millisecond))
	time.Sleep(100 * time.Millisecond)
	_, _, err := c.GetWithTTL("k")
	require.NoError(t, err)
	time.Sleep(150 * time.Millisecond)
	_, err = c.Get("k")
	require.Error(t, err)
}
