package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOAuth2Secrets_Generate_Structure(t *testing.T) {
	s := NewOAuth2Secrets()
	cs, err := s.Generate()
	require.NoError(t, err)
	require.NotEmpty(t, cs.Secret)
	require.NotEmpty(t, cs.Hash)

	// Hash must be tagged "$sha256$" + 64 hex chars (SHA-256 = 32 bytes)
	require.True(t, strings.HasPrefix(cs.Hash, PrefixSHA256), "hash must carry $sha256$ prefix")
	digest := strings.TrimPrefix(cs.Hash, PrefixSHA256)
	require.Len(t, digest, 64)
	require.True(t, isHex(digest), "hash digest must be hex-encoded")
}

func TestOAuth2Secrets_Generate_MinEntropy(t *testing.T) {
	s := NewOAuth2Secrets()
	cs, err := s.Generate()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(cs.Secret), 32, "secret must be at least 32 characters")
	require.Greater(t, shannonEntropy(cs.Secret), 3.5, "generated secret must exceed entropy floor")
}

func TestOAuth2Secrets_Generate_Unique(t *testing.T) {
	s := NewOAuth2Secrets()
	cs1, err := s.Generate()
	require.NoError(t, err)
	cs2, err := s.Generate()
	require.NoError(t, err)

	require.NotEqual(t, cs1.Secret, cs2.Secret)
	require.NotEqual(t, cs1.Hash, cs2.Hash)
}

func TestOAuth2Secrets_HashSHA256_RoundTrip(t *testing.T) {
	s := NewOAuth2Secrets()
	cs, err := s.Generate()
	require.NoError(t, err)

	hash, err := s.HashSHA256(cs.Secret)
	require.NoError(t, err)
	require.Equal(t, cs.Hash, hash)
}

func TestOAuth2Secrets_HashSHA256_TooShort(t *testing.T) {
	s := NewOAuth2Secrets()
	_, err := s.HashSHA256("tooshort")
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least 32 characters")
}

func TestOAuth2Secrets_HashSHA256_LowEntropy(t *testing.T) {
	s := NewOAuth2Secrets()
	// 32 chars but all the same byte — entropy ≈ 0
	_, err := s.HashSHA256(strings.Repeat("a", 32))
	require.Error(t, err)
	require.Contains(t, err.Error(), "entropy")
}

func TestOAuth2Secrets_HashSHA256_PlainString(t *testing.T) {
	s := NewOAuth2Secrets()
	// A user-provided secret that is not base64url — must work if it meets entropy
	secret := "Tr0ub4dor&3-correct-horse-battery-staple-xkcd936"
	h1, err := s.HashSHA256(secret)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(h1, PrefixSHA256))
	require.Len(t, strings.TrimPrefix(h1, PrefixSHA256), 64)
	// Deterministic
	h2, err := s.HashSHA256(secret)
	require.NoError(t, err)
	require.Equal(t, h1, h2)
}

func TestOAuth2Secrets_VerifySHA256_RejectsUntaggedHash(t *testing.T) {
	s := NewOAuth2Secrets()
	cs, err := s.Generate()
	require.NoError(t, err)

	// Strip the algorithm prefix — verification must fail because the algorithm
	// can no longer be identified.
	untagged := strings.TrimPrefix(cs.Hash, PrefixSHA256)
	require.False(t, s.VerifySHA256(cs.Secret, untagged))
}

func TestOAuth2Secrets_VerifySHA256_Valid(t *testing.T) {
	s := NewOAuth2Secrets()
	cs, err := s.Generate()
	require.NoError(t, err)

	require.True(t, s.VerifySHA256(cs.Secret, cs.Hash))
}

func TestOAuth2Secrets_VerifySHA256_WrongSecret(t *testing.T) {
	s := NewOAuth2Secrets()
	cs, err := s.Generate()
	require.NoError(t, err)
	cs2, err := s.Generate()
	require.NoError(t, err)

	require.False(t, s.VerifySHA256(cs2.Secret, cs.Hash))
}

func TestOAuth2Secrets_VerifySHA256_TamperedHash(t *testing.T) {
	s := NewOAuth2Secrets()
	cs, err := s.Generate()
	require.NoError(t, err)

	tampered := PrefixSHA256 + strings.Repeat("a", 64)
	require.False(t, s.VerifySHA256(cs.Secret, tampered))
}

func TestOAuth2Secrets_VerifySHA256_TooShort(t *testing.T) {
	s := NewOAuth2Secrets()
	// Any secret that fails entropy validation must return false, not panic
	require.False(t, s.VerifySHA256("short", "somehash"))
}

func TestOAuth2Secrets_VerifySHA256_LowEntropy(t *testing.T) {
	s := NewOAuth2Secrets()
	require.False(t, s.VerifySHA256(strings.Repeat("x", 32), "somehash"))
}

// --- HMAC-SHA-256 ---

func TestOAuth2Secrets_HashHMAC_RoundTrip(t *testing.T) {
	s := NewOAuth2SecretsWithHMACKey([]byte("super-secret-hmac-key-for-testing"))
	cs, err := s.Generate()
	require.NoError(t, err)

	hash, err := s.HashHMAC(cs.Secret)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(hash, PrefixHMACSHA256), "hash must carry $hmac-sha256$ prefix")
	digest := strings.TrimPrefix(hash, PrefixHMACSHA256)
	require.Len(t, digest, 64)
	require.True(t, isHex(digest))
}

func TestOAuth2Secrets_VerifyHMAC_WrongAlgorithmPrefix(t *testing.T) {
	s := NewOAuth2SecretsWithHMACKey([]byte("my-key"))
	cs, err := s.Generate()
	require.NoError(t, err)

	// A SHA-256 hash must not validate against the HMAC verifier and vice versa.
	plainHash, err := s.HashSHA256(cs.Secret)
	require.NoError(t, err)
	require.False(t, s.VerifyHMAC(cs.Secret, plainHash))

	hmacHash, err := s.HashHMAC(cs.Secret)
	require.NoError(t, err)
	require.False(t, s.VerifySHA256(cs.Secret, hmacHash))
}

func TestOAuth2Secrets_HashHMAC_EmptyKey(t *testing.T) {
	s := NewOAuth2Secrets() // no key
	cs, err := s.Generate()
	require.NoError(t, err)

	_, err = s.HashHMAC(cs.Secret)
	require.Error(t, err)
}

func TestOAuth2Secrets_HashHMAC_DifferentKeys_DifferentHash(t *testing.T) {
	cs, err := NewOAuth2Secrets().Generate()
	require.NoError(t, err)

	h1, err := NewOAuth2SecretsWithHMACKey([]byte("key-one")).HashHMAC(cs.Secret)
	require.NoError(t, err)
	h2, err := NewOAuth2SecretsWithHMACKey([]byte("key-two")).HashHMAC(cs.Secret)
	require.NoError(t, err)

	require.NotEqual(t, h1, h2)
}

func TestOAuth2Secrets_HashHMAC_DifferentFromSHA256(t *testing.T) {
	s := NewOAuth2SecretsWithHMACKey([]byte("some-key"))
	cs, err := s.Generate()
	require.NoError(t, err)

	hmacHash, err := s.HashHMAC(cs.Secret)
	require.NoError(t, err)

	require.NotEqual(t, cs.Hash, hmacHash, "HMAC hash must differ from plain SHA-256 hash")
}

func TestOAuth2Secrets_VerifyHMAC_Valid(t *testing.T) {
	s := NewOAuth2SecretsWithHMACKey([]byte("my-hmac-key"))
	cs, err := s.Generate()
	require.NoError(t, err)

	hash, err := s.HashHMAC(cs.Secret)
	require.NoError(t, err)

	require.True(t, s.VerifyHMAC(cs.Secret, hash))
}

func TestOAuth2Secrets_VerifyHMAC_WrongKey(t *testing.T) {
	signer := NewOAuth2SecretsWithHMACKey([]byte("correct-key"))
	verifier := NewOAuth2SecretsWithHMACKey([]byte("wrong-key"))
	cs, err := signer.Generate()
	require.NoError(t, err)

	hash, err := signer.HashHMAC(cs.Secret)
	require.NoError(t, err)

	require.False(t, verifier.VerifyHMAC(cs.Secret, hash))
}

func TestOAuth2Secrets_VerifyHMAC_WrongSecret(t *testing.T) {
	s := NewOAuth2SecretsWithHMACKey([]byte("my-key"))
	cs, err := s.Generate()
	require.NoError(t, err)
	cs2, err := s.Generate()
	require.NoError(t, err)

	hash, err := s.HashHMAC(cs.Secret)
	require.NoError(t, err)

	require.False(t, s.VerifyHMAC(cs2.Secret, hash))
}

func TestOAuth2Secrets_Verify_AutoDispatch(t *testing.T) {
	s := NewOAuth2SecretsWithHMACKey([]byte("k"))
	cs, err := s.Generate()
	require.NoError(t, err)
	hmacHash, err := s.HashHMAC(cs.Secret)
	require.NoError(t, err)

	require.True(t, s.Verify(cs.Secret, cs.Hash), "auto-dispatch must validate $sha256$ hash")
	require.True(t, s.Verify(cs.Secret, hmacHash), "auto-dispatch must validate $hmac-sha256$ hash")
	require.False(t, s.Verify(cs.Secret, "plain-untagged"), "unknown prefix must fail")
}

func TestDetectHashAlgorithm(t *testing.T) {
	s := NewOAuth2SecretsWithHMACKey([]byte("k"))
	cs, err := s.Generate()
	require.NoError(t, err)
	hmacHash, err := s.HashHMAC(cs.Secret)
	require.NoError(t, err)

	require.Equal(t, HashAlgorithmSHA256, DetectHashAlgorithm(cs.Hash))
	require.Equal(t, HashAlgorithmHMACSHA256, DetectHashAlgorithm(hmacHash))
	require.Equal(t, HashAlgorithmUnknown, DetectHashAlgorithm("plain-untagged-hash"))
	require.Equal(t, HashAlgorithmUnknown, DetectHashAlgorithm(""))
	require.Equal(t, HashAlgorithmUnknown, DetectHashAlgorithm("$argon2id$v=19$..."))

	// Method form delegates to the package-level helper.
	require.Equal(t, HashAlgorithmSHA256, s.DetectAlgorithm(cs.Hash))

	require.Equal(t, "sha256", HashAlgorithmSHA256.String())
	require.Equal(t, "hmac-sha256", HashAlgorithmHMACSHA256.String())
	require.Equal(t, "unknown", HashAlgorithmUnknown.String())
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
