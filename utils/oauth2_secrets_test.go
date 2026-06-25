package utils

import (
	"strings"
	"testing"

	argon2id "github.com/alexedwards/argon2id"
	"github.com/stretchr/testify/require"
)

func TestOAuth2Secrets_Generate_Structure(t *testing.T) {
	s := NewOAuth2Secrets()
	cs, err := s.Generate(nil)
	require.NoError(t, err)
	require.NotEmpty(t, cs.Secret)
	require.NotEmpty(t, cs.HashSHA256)

	// SHA-256 hash must always be present and well-formed.
	require.True(t, strings.HasPrefix(cs.HashSHA256, PrefixSHA256), "hash must carry $sha256$ prefix")
	digest := strings.TrimPrefix(cs.HashSHA256, PrefixSHA256)
	require.Len(t, digest, 64)
	require.True(t, isHex(digest), "hash digest must be hex-encoded")

	// Optional fields are empty when the corresponding option is not set.
	require.Empty(t, cs.HashHMAC, "HashHMAC must be empty when no HMAC key is configured")
	require.Empty(t, cs.HashArgon2id, "HashArgon2id must be empty when Argon2idParams is nil")
}

func TestOAuth2Secrets_Generate_AllHashes(t *testing.T) {
	s := NewOAuth2Secrets(
		WithHMACKey([]byte("server-pepper")),
		WithArgon2idParams(fastArgon2idParams()),
	)
	cs, err := s.Generate(nil)
	require.NoError(t, err)

	require.True(t, strings.HasPrefix(cs.HashSHA256, PrefixSHA256))
	require.True(t, strings.HasPrefix(cs.HashHMAC, PrefixHMACSHA256))
	require.True(t, strings.HasPrefix(cs.HashArgon2id, PrefixArgon2id))

	// All three must verify against the same plaintext.
	require.True(t, s.VerifySHA256(cs.Secret, cs.HashSHA256))
	require.True(t, s.VerifyHMAC(cs.Secret, cs.HashHMAC))
	require.True(t, s.VerifyArgon2id(cs.Secret, cs.HashArgon2id))
}

func TestOAuth2Secrets_Generate_MinEntropy(t *testing.T) {
	s := NewOAuth2Secrets()
	cs, err := s.Generate(nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(cs.Secret), 32, "secret must be at least 32 characters")
	require.Greater(t, shannonEntropy(cs.Secret), 3.5, "generated secret must exceed entropy floor")
}

func TestOAuth2Secrets_Generate_Unique(t *testing.T) {
	s := NewOAuth2Secrets()
	cs1, err := s.Generate(nil)
	require.NoError(t, err)
	cs2, err := s.Generate(nil)
	require.NoError(t, err)

	require.NotEqual(t, cs1.Secret, cs2.Secret)
	require.NotEqual(t, cs1.HashSHA256, cs2.HashSHA256)
}

func TestOAuth2Secrets_Generate_WithProvidedSecret(t *testing.T) {
	s := NewOAuth2Secrets(WithHMACKey([]byte("pepper")), WithArgon2idParams(fastArgon2idParams()))
	provided := "Tr0ub4dor&3-correct-horse-battery-staple-xkcd936"
	cs, err := s.Generate(&provided)
	require.NoError(t, err)
	require.Equal(t, provided, cs.Secret, "returned secret must equal the provided one")
	require.True(t, strings.HasPrefix(cs.HashSHA256, PrefixSHA256))
	require.True(t, strings.HasPrefix(cs.HashHMAC, PrefixHMACSHA256))
	require.True(t, strings.HasPrefix(cs.HashArgon2id, PrefixArgon2id))
	require.True(t, s.VerifySHA256(cs.Secret, cs.HashSHA256))
	require.True(t, s.VerifyHMAC(cs.Secret, cs.HashHMAC))
	require.True(t, s.VerifyArgon2id(cs.Secret, cs.HashArgon2id))
}

func TestOAuth2Secrets_Generate_ProvidedSecretWriteback(t *testing.T) {
	s := NewOAuth2Secrets()
	// Empty string → generate random; the pointer should be updated.
	placeholder := ""
	cs, err := s.Generate(&placeholder)
	require.NoError(t, err)
	require.NotEmpty(t, placeholder, "Generate must write the new secret back through the pointer")
	require.Equal(t, placeholder, cs.Secret)
}

func TestOAuth2Secrets_Generate_NilPointerWriteback(t *testing.T) {
	s := NewOAuth2Secrets()
	// nil pointer → generate random; must not panic.
	cs, err := s.Generate(nil)
	require.NoError(t, err)
	require.NotEmpty(t, cs.Secret)
}

func TestOAuth2Secrets_Generate_ProvidedSecretLowEntropy(t *testing.T) {
	s := NewOAuth2Secrets()
	bad := strings.Repeat("a", 32) // 32 chars, zero entropy
	_, err := s.Generate(&bad)
	require.Error(t, err)
	require.Contains(t, err.Error(), "entropy")
}

func TestOAuth2Secrets_Generate_ProvidedSecretTooShort(t *testing.T) {
	s := NewOAuth2Secrets()
	bad := "tooshort"
	_, err := s.Generate(&bad)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least 32 characters")
}

func TestOAuth2Secrets_HashSHA256_RoundTrip(t *testing.T) {
	s := NewOAuth2Secrets()
	cs, err := s.Generate(nil)
	require.NoError(t, err)

	hash, err := s.HashSHA256(cs.Secret)
	require.NoError(t, err)
	require.Equal(t, cs.HashSHA256, hash)
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
	cs, err := s.Generate(nil)
	require.NoError(t, err)

	// Strip the algorithm prefix — verification must fail because the algorithm
	// can no longer be identified.
	untagged := strings.TrimPrefix(cs.HashSHA256, PrefixSHA256)
	require.False(t, s.VerifySHA256(cs.Secret, untagged))
}

func TestOAuth2Secrets_VerifySHA256_Valid(t *testing.T) {
	s := NewOAuth2Secrets()
	cs, err := s.Generate(nil)
	require.NoError(t, err)

	require.True(t, s.VerifySHA256(cs.Secret, cs.HashSHA256))
}

func TestOAuth2Secrets_VerifySHA256_WrongSecret(t *testing.T) {
	s := NewOAuth2Secrets()
	cs, err := s.Generate(nil)
	require.NoError(t, err)
	cs2, err := s.Generate(nil)
	require.NoError(t, err)

	require.False(t, s.VerifySHA256(cs2.Secret, cs.HashSHA256))
}

func TestOAuth2Secrets_VerifySHA256_TamperedHash(t *testing.T) {
	s := NewOAuth2Secrets()
	cs, err := s.Generate(nil)
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
	s := NewOAuth2Secrets(WithHMACKey([]byte("super-secret-hmac-key-for-testing")))
	cs, err := s.Generate(nil)
	require.NoError(t, err)

	hash, err := s.HashHMAC(cs.Secret)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(hash, PrefixHMACSHA256), "hash must carry $hmac-sha256$ prefix")
	digest := strings.TrimPrefix(hash, PrefixHMACSHA256)
	require.Len(t, digest, 64)
	require.True(t, isHex(digest))
}

func TestOAuth2Secrets_VerifyHMAC_WrongAlgorithmPrefix(t *testing.T) {
	s := NewOAuth2Secrets(WithHMACKey([]byte("my-key")))
	cs, err := s.Generate(nil)
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
	cs, err := s.Generate(nil)
	require.NoError(t, err)

	_, err = s.HashHMAC(cs.Secret)
	require.Error(t, err)
}

func TestOAuth2Secrets_HashHMAC_DifferentKeys_DifferentHash(t *testing.T) {
	cs, err := NewOAuth2Secrets().Generate(nil)
	require.NoError(t, err)

	h1, err := NewOAuth2Secrets(WithHMACKey([]byte("key-one"))).HashHMAC(cs.Secret)
	require.NoError(t, err)
	h2, err := NewOAuth2Secrets(WithHMACKey([]byte("key-two"))).HashHMAC(cs.Secret)
	require.NoError(t, err)

	require.NotEqual(t, h1, h2)
}

func TestOAuth2Secrets_HashHMAC_DifferentFromSHA256(t *testing.T) {
	s := NewOAuth2Secrets(WithHMACKey([]byte("some-key")))
	cs, err := s.Generate(nil)
	require.NoError(t, err)

	hmacHash, err := s.HashHMAC(cs.Secret)
	require.NoError(t, err)

	require.NotEqual(t, cs.HashSHA256, hmacHash, "HMAC hash must differ from plain SHA-256 hash")
}

func TestOAuth2Secrets_VerifyHMAC_Valid(t *testing.T) {
	s := NewOAuth2Secrets(WithHMACKey([]byte("my-hmac-key")))
	cs, err := s.Generate(nil)
	require.NoError(t, err)

	hash, err := s.HashHMAC(cs.Secret)
	require.NoError(t, err)

	require.True(t, s.VerifyHMAC(cs.Secret, hash))
}

func TestOAuth2Secrets_VerifyHMAC_WrongKey(t *testing.T) {
	signer := NewOAuth2Secrets(WithHMACKey([]byte("correct-key")))
	verifier := NewOAuth2Secrets(WithHMACKey([]byte("wrong-key")))
	cs, err := signer.Generate(nil)
	require.NoError(t, err)

	hash, err := signer.HashHMAC(cs.Secret)
	require.NoError(t, err)

	require.False(t, verifier.VerifyHMAC(cs.Secret, hash))
}

func TestOAuth2Secrets_VerifyHMAC_WrongSecret(t *testing.T) {
	s := NewOAuth2Secrets(WithHMACKey([]byte("my-key")))
	cs, err := s.Generate(nil)
	require.NoError(t, err)
	cs2, err := s.Generate(nil)
	require.NoError(t, err)

	hash, err := s.HashHMAC(cs.Secret)
	require.NoError(t, err)

	require.False(t, s.VerifyHMAC(cs2.Secret, hash))
}

func TestOAuth2Secrets_Verify_AutoDispatch(t *testing.T) {
	s := NewOAuth2Secrets(WithHMACKey([]byte("k")))
	cs, err := s.Generate(nil)
	require.NoError(t, err)
	hmacHash, err := s.HashHMAC(cs.Secret)
	require.NoError(t, err)
	argonHash, err := NewOAuth2Secrets(WithArgon2idParams(fastArgon2idParams())).HashArgon2id(cs.Secret)
	require.NoError(t, err)

	require.True(t, s.Verify(cs.Secret, cs.HashSHA256), "auto-dispatch must validate $sha256$ hash")
	require.True(t, s.Verify(cs.Secret, hmacHash), "auto-dispatch must validate $hmac-sha256$ hash")
	require.True(t, s.Verify(cs.Secret, argonHash), "auto-dispatch must validate $argon2id$ hash")
	require.False(t, s.Verify(cs.Secret, "plain-untagged"), "unknown prefix must fail")
}

func TestDetectHashAlgorithm(t *testing.T) {
	s := NewOAuth2Secrets(WithHMACKey([]byte("k")))
	cs, err := s.Generate(nil)
	require.NoError(t, err)
	hmacHash, err := s.HashHMAC(cs.Secret)
	require.NoError(t, err)
	argonHash, err := NewOAuth2Secrets(WithArgon2idParams(fastArgon2idParams())).HashArgon2id(cs.Secret)
	require.NoError(t, err)

	require.Equal(t, HashAlgorithmSHA256, DetectHashAlgorithm(cs.HashSHA256))
	require.Equal(t, HashAlgorithmHMACSHA256, DetectHashAlgorithm(hmacHash))
	require.Equal(t, HashAlgorithmArgon2id, DetectHashAlgorithm(argonHash))
	require.Equal(t, HashAlgorithmUnknown, DetectHashAlgorithm("plain-untagged-hash"))
	require.Equal(t, HashAlgorithmUnknown, DetectHashAlgorithm(""))

	// Method form delegates to the package-level helper.
	require.Equal(t, HashAlgorithmSHA256, s.DetectAlgorithm(cs.HashSHA256))

	require.Equal(t, "sha256", HashAlgorithmSHA256.String())
	require.Equal(t, "hmac-sha256", HashAlgorithmHMACSHA256.String())
	require.Equal(t, "argon2id", HashAlgorithmArgon2id.String())
	require.Equal(t, "unknown", HashAlgorithmUnknown.String())
}

// --- Argon2id ---

// fastArgon2idParams returns minimal cost params suitable for unit tests.
func fastArgon2idParams() *argon2id.Params {
	return &argon2id.Params{
		Memory:      8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
}

func TestOAuth2Secrets_HashArgon2id_RoundTrip(t *testing.T) {
	s := NewOAuth2Secrets(WithArgon2idParams(fastArgon2idParams()))
	cs, err := s.Generate(nil)
	require.NoError(t, err)

	hash, err := s.HashArgon2id(cs.Secret)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(hash, PrefixArgon2id), "hash must carry $argon2id$ prefix")
	require.True(t, s.VerifyArgon2id(cs.Secret, hash))
}

func TestOAuth2Secrets_HashArgon2id_NonDeterministic(t *testing.T) {
	// argon2id salts every call — two hashes of the same secret must differ.
	s := NewOAuth2Secrets(WithArgon2idParams(fastArgon2idParams()))
	cs, err := s.Generate(nil)
	require.NoError(t, err)

	h1, err := s.HashArgon2id(cs.Secret)
	require.NoError(t, err)
	h2, err := s.HashArgon2id(cs.Secret)
	require.NoError(t, err)
	require.NotEqual(t, h1, h2, "two argon2id hashes of the same secret must differ (different salts)")
	// But both must still verify against the original secret.
	require.True(t, s.VerifyArgon2id(cs.Secret, h1))
	require.True(t, s.VerifyArgon2id(cs.Secret, h2))
}

func TestOAuth2Secrets_HashArgon2id_DefaultParams(t *testing.T) {
	// nil params → argon2id.DefaultParams; must still produce a valid hash.
	s := NewOAuth2Secrets(WithArgon2idParams(nil))
	cs, err := s.Generate(nil)
	require.NoError(t, err)

	hash, err := s.HashArgon2id(cs.Secret)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(hash, PrefixArgon2id))
	require.True(t, s.VerifyArgon2id(cs.Secret, hash))
}

func TestOAuth2Secrets_VerifyArgon2id_WrongSecret(t *testing.T) {
	s := NewOAuth2Secrets(WithArgon2idParams(fastArgon2idParams()))
	cs, err := s.Generate(nil)
	require.NoError(t, err)
	cs2, err := s.Generate(nil)
	require.NoError(t, err)

	hash, err := s.HashArgon2id(cs.Secret)
	require.NoError(t, err)

	require.False(t, s.VerifyArgon2id(cs2.Secret, hash))
}

func TestOAuth2Secrets_VerifyArgon2id_WrongPrefix(t *testing.T) {
	s := NewOAuth2Secrets(WithArgon2idParams(fastArgon2idParams()))
	cs, err := s.Generate(nil)
	require.NoError(t, err)

	// A SHA-256 hash must not validate against the argon2id verifier.
	require.False(t, s.VerifyArgon2id(cs.Secret, cs.HashSHA256))
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
