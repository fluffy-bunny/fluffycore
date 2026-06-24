package utils

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration is a time.Duration that JSON-unmarshals from a human-readable string
// ("60s", "5m", "24h") as well as from a raw integer (nanoseconds).
type Duration time.Duration

// UnmarshalText implements encoding.TextUnmarshaler.
// mapstructure (used by fluffycore for env/config decoding) calls this when
// it encounters a string value destined for a custom type.
func (d *Duration) UnmarshalText(b []byte) error {
	pd, err := time.ParseDuration(string(b))
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", string(b), err)
	}
	*d = Duration(pd)
	return nil
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	// Try string form first: "5m", "24h", "60s" …
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		return d.UnmarshalText([]byte(s))
	}
	// Fall back to integer nanoseconds for backward-compat.
	var n int64
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("cacheTTL must be a duration string (e.g. \"5m\") or integer nanoseconds: %w", err)
	}
	*d = Duration(n)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}
