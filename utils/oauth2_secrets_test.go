package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateClientSecret_Structure(t *testing.T) {
	cs, err := GenerateClientSecret()
	require.NoError(t, err)
	require.NotEmpty(t, cs.Secret)
	require.NotEmpty(t, cs.Hash)

	// Hash must be 64 hex chars (SHA-256 = 32 bytes)
	require.Len(t, cs.Hash, 64)
	require.True(t, isHex(cs.Hash), "hash must be hex-encoded")
}

func TestGenerateClientSecret_MinEntropy(t *testing.T) {
	cs, err := GenerateClientSecret()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(cs.Secret), 32, "secret must be at least 32 characters")
	require.Greater(t, shannonEntropy(cs.Secret), 3.5, "generated secret must exceed entropy floor")
}

func TestGenerateClientSecret_Unique(t *testing.T) {
	cs1, err := GenerateClientSecret()
	require.NoError(t, err)
	cs2, err := GenerateClientSecret()
	require.NoError(t, err)

	require.NotEqual(t, cs1.Secret, cs2.Secret)
	require.NotEqual(t, cs1.Hash, cs2.Hash)
}

func TestHashClientSecret_RoundTrip(t *testing.T) {
	cs, err := GenerateClientSecret()
	require.NoError(t, err)

	hash, err := HashClientSecret(cs.Secret)
	require.NoError(t, err)
	require.Equal(t, cs.Hash, hash)
}

func TestHashClientSecret_TooShort(t *testing.T) {
	_, err := HashClientSecret("tooshort")
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least 32 characters")
}

func TestHashClientSecret_LowEntropy(t *testing.T) {
	// 32 chars but all the same byte — entropy ≈ 0
	_, err := HashClientSecret(strings.Repeat("a", 32))
	require.Error(t, err)
	require.Contains(t, err.Error(), "entropy")
}

func TestHashClientSecret_PlainString(t *testing.T) {
	// A user-provided secret that is not base64url — must work if it meets entropy
	secret := "Tr0ub4dor&3-correct-horse-battery-staple-xkcd936"
	h1, err := HashClientSecret(secret)
	require.NoError(t, err)
	require.Len(t, h1, 64)
	// Deterministic
	h2, err := HashClientSecret(secret)
	require.NoError(t, err)
	require.Equal(t, h1, h2)
}

func TestVerifyClientSecret_Valid(t *testing.T) {
	cs, err := GenerateClientSecret()
	require.NoError(t, err)

	ok := VerifyClientSecret(cs.Secret, cs.Hash)
	require.True(t, ok)
}

func TestVerifyClientSecret_WrongSecret(t *testing.T) {
	cs, err := GenerateClientSecret()
	require.NoError(t, err)
	cs2, err := GenerateClientSecret()
	require.NoError(t, err)

	ok := VerifyClientSecret(cs2.Secret, cs.Hash)
	require.False(t, ok)
}

func TestVerifyClientSecret_TamperedHash(t *testing.T) {
	cs, err := GenerateClientSecret()
	require.NoError(t, err)

	tampered := strings.Repeat("a", 64)
	ok := VerifyClientSecret(cs.Secret, tampered)
	require.False(t, ok)
}

func TestVerifyClientSecret_TooShort(t *testing.T) {
	// Any secret that fails entropy validation must return false, not panic
	ok := VerifyClientSecret("short", "somehash")
	require.False(t, ok)
}

func TestVerifyClientSecret_LowEntropy(t *testing.T) {
	ok := VerifyClientSecret(strings.Repeat("x", 32), "somehash")
	require.False(t, ok)
}

// --- HMAC-SHA-256 ---

func TestHashClientSecretHMAC_RoundTrip(t *testing.T) {
	key := []byte("super-secret-hmac-key-for-testing")
	cs, err := GenerateClientSecret()
	require.NoError(t, err)

	hash, err := HashClientSecretHMAC(cs.Secret, key)
	require.NoError(t, err)
	require.Len(t, hash, 64)
	require.True(t, isHex(hash))
}

func TestHashClientSecretHMAC_EmptyKey(t *testing.T) {
	cs, err := GenerateClientSecret()
	require.NoError(t, err)

	_, err = HashClientSecretHMAC(cs.Secret, []byte{})
	require.Error(t, err)
}

func TestHashClientSecretHMAC_DifferentKeys_DifferentHash(t *testing.T) {
	key1 := []byte("key-one")
	key2 := []byte("key-two")
	cs, err := GenerateClientSecret()
	require.NoError(t, err)

	h1, err := HashClientSecretHMAC(cs.Secret, key1)
	require.NoError(t, err)
	h2, err := HashClientSecretHMAC(cs.Secret, key2)
	require.NoError(t, err)

	require.NotEqual(t, h1, h2)
}

func TestHashClientSecretHMAC_DifferentFromSHA256(t *testing.T) {
	key := []byte("some-key")
	cs, err := GenerateClientSecret()
	require.NoError(t, err)

	hmacHash, err := HashClientSecretHMAC(cs.Secret, key)
	require.NoError(t, err)

	require.NotEqual(t, cs.Hash, hmacHash, "HMAC hash must differ from plain SHA-256 hash")
}

func TestVerifyClientSecretHMAC_Valid(t *testing.T) {
	key := []byte("my-hmac-key")
	cs, err := GenerateClientSecret()
	require.NoError(t, err)

	hash, err := HashClientSecretHMAC(cs.Secret, key)
	require.NoError(t, err)

	ok := VerifyClientSecretHMAC(cs.Secret, hash, key)
	require.True(t, ok)
}

func TestVerifyClientSecretHMAC_WrongKey(t *testing.T) {
	key := []byte("correct-key")
	wrongKey := []byte("wrong-key")
	cs, err := GenerateClientSecret()
	require.NoError(t, err)

	hash, err := HashClientSecretHMAC(cs.Secret, key)
	require.NoError(t, err)

	ok := VerifyClientSecretHMAC(cs.Secret, hash, wrongKey)
	require.False(t, ok)
}

func TestVerifyClientSecretHMAC_WrongSecret(t *testing.T) {
	key := []byte("my-key")
	cs, err := GenerateClientSecret()
	require.NoError(t, err)
	cs2, err := GenerateClientSecret()
	require.NoError(t, err)

	hash, err := HashClientSecretHMAC(cs.Secret, key)
	require.NoError(t, err)

	ok := VerifyClientSecretHMAC(cs2.Secret, hash, key)
	require.False(t, ok)
}

// isHex returns true if s contains only valid lowercase hex characters.
func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
