package mongostore

import (
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
)

// TestVerifyID_ValidXID verifies that VerifyID accepts valid xid IDs.
// (Was CRITICAL: VerifyID used hex.DecodeString but xid uses custom
// base32 encoding, so it always rejected valid IDs.)
func TestVerifyID_ValidXID(t *testing.T) {
	id := xid.New().String()
	require.True(t, VerifyID(id), "valid xid ID should pass verification: %s", id)
}

// TestVerifyID_MultipleValidXIDs verifies multiple generated IDs.
func TestVerifyID_MultipleValidXIDs(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := xid.New().String()
		require.True(t, VerifyID(id), "valid xid ID should pass: %s", id)
	}
}

// TestVerifyID_InvalidIDs verifies that VerifyID rejects invalid IDs.
func TestVerifyID_InvalidIDs(t *testing.T) {
	testCases := []struct {
		name string
		id   string
	}{
		{"empty string", ""},
		{"too short", "abc"},
		{"random garbage", "!!!not-an-id!!!"},
		{"hex string (old broken behavior)", "deadbeefdeadbeefdeadbeef"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.False(t, VerifyID(tc.id), "invalid ID should fail: %s", tc.id)
		})
	}
}

// TestNewId_IsValid verifies NewId produces valid IDs that pass VerifyID.
func TestNewId_IsValid(t *testing.T) {
	id := NewId()
	require.True(t, VerifyID(id))
}
