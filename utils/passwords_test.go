package utils

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

// GeneratePassword used to return an xid (timestamp + machine id + pid + counter): two
// passwords generated back to back shared almost all their characters.
func TestGeneratePassword_IsFullEntropyRandom(t *testing.T) {
	const n = 200
	seen := make(map[string]bool, n)
	var prev []byte
	for i := 0; i < n; i++ {
		pw, err := GeneratePassword()
		require.NoError(t, err)
		require.False(t, seen[pw], "duplicate password")
		seen[pw] = true

		raw, err := base64.RawURLEncoding.DecodeString(pw)
		require.NoError(t, err, "password is URL-safe base64")
		require.Len(t, raw, generatedPasswordBytes, "256 bits of entropy")

		if prev != nil {
			// An xid-style value shares a long prefix with the previous one; random bytes
			// share on average 1/256 of positions.
			same := 0
			for j := range raw {
				if raw[j] == prev[j] {
					same++
				}
			}
			require.Less(t, same, 8, "consecutive passwords must not be correlated")
		}
		prev = raw
	}
}

func TestGeneratePasswordHashSet_RoundTrips(t *testing.T) {
	set, err := GeneratePasswordHashSet(nil)
	require.NoError(t, err)
	ok, err := ComparePasswordHash(set.Password, set.Hash)
	require.NoError(t, err)
	require.True(t, ok)

	provided := "my-own-secret"
	set, err = GeneratePasswordHashSet(&provided)
	require.NoError(t, err)
	require.Equal(t, provided, set.Password)
}
