package cache

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Starve ttlcache's expiry goroutine across the moment an entry expires (as a loaded CI runner
// does), re-set the key, and check the fresh entry survives its own TTL. Without dropExpired the
// stale queue entry's later eviction deleted the fresh item by key.
func TestSetAfterExpiryWithLateCleanerKeepsFreshEntry(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1)) // one P, so the busy wait below blocks the cleaner
	for name, set := range map[string]func(*service, string){
		"SetWithTTL": func(c *service, v string) { _ = c.SetWithTTL("k", v, 40*time.Millisecond) },
		"Set": func(c *service, v string) {
			_ = c.SetTTL(40 * time.Millisecond)
			_ = c.Set("k", v)
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := NewMemoryCache()
			defer c.Close()
			set(c, "v1")
			for deadline := time.Now().Add(45 * time.Millisecond); time.Now().Before(deadline); {
				// busy: no yield
			}
			_, err := c.Get("k")
			require.Error(t, err, "the entry has expired")

			set(c, "v2")                      // what a re-resolve does
			time.Sleep(10 * time.Millisecond) // the cleaner runs now
			v, err := c.Get("k")
			require.NoError(t, err, "a 10ms-old entry with a 40ms TTL must still be there")
			require.Equal(t, "v2", v)
		})
	}
}
