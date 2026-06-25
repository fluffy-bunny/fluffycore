package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"

	argon2id "github.com/alexedwards/argon2id"
)

const (
	// minSecretLen is the minimum character length for any client secret.
	minSecretLen = 32
	// minEntropyBitsPerChar is the Shannon entropy floor (bits/char).
	// Rejects low-entropy secrets such as repeated characters or dictionary words.
	minEntropyBitsPerChar = 3.5

	// PHC-style algorithm identifiers prefixed to every stored hash so callers
	// can detect the algorithm without out-of-band metadata (cf. argon2's
	// "$argon2id$..." format).
	PrefixSHA256     = "$sha256$"
	PrefixHMACSHA256 = "$hmac-sha256$"
	// PrefixArgon2id is the PHC prefix emitted by the argon2id library itself.
	PrefixArgon2id = "$argon2id$"
)

// ClientSecret holds a generated plaintext secret and its algorithm-tagged hash.
type ClientSecret struct {
	// Secret is the base64url-encoded plaintext. Distribute this to the client once.
	Secret string `json:"secret"`
	// Hash is the algorithm-tagged digest (e.g. "$sha256$<hex>"). Store this server-side.
	Hash string `json:"hash"`
}

// HashAlgorithm identifies which hashing algorithm produced a stored hash so
// callers can dispatch to the correct verifier.
type HashAlgorithm int

const (
	// HashAlgorithmUnknown means the hash carries no recognized prefix.
	HashAlgorithmUnknown HashAlgorithm = iota
	// HashAlgorithmSHA256 — verify with OAuth2Secrets.VerifySHA256.
	HashAlgorithmSHA256
	// HashAlgorithmHMACSHA256 — verify with OAuth2Secrets.VerifyHMAC.
	HashAlgorithmHMACSHA256
	// HashAlgorithmArgon2id — verify with OAuth2Secrets.VerifyArgon2id.
	HashAlgorithmArgon2id
)

// String returns the human-readable algorithm name.
func (a HashAlgorithm) String() string {
	switch a {
	case HashAlgorithmSHA256:
		return "sha256"
	case HashAlgorithmHMACSHA256:
		return "hmac-sha256"
	case HashAlgorithmArgon2id:
		return "argon2id"
	default:
		return "unknown"
	}
}

// DetectHashAlgorithm inspects a stored hash and returns the algorithm tag
// that produced it. HashAlgorithmUnknown means the hash is malformed or from
// an unsupported algorithm and cannot be verified by this package.
func DetectHashAlgorithm(hash string) HashAlgorithm {
	switch {
	case strings.HasPrefix(hash, PrefixSHA256):
		return HashAlgorithmSHA256
	case strings.HasPrefix(hash, PrefixHMACSHA256):
		return HashAlgorithmHMACSHA256
	case strings.HasPrefix(hash, PrefixArgon2id):
		return HashAlgorithmArgon2id
	default:
		return HashAlgorithmUnknown
	}
}

// OAuth2Secrets generates and validates OAuth2 client secrets.
// The zero value is usable for SHA-256 operations; populate HMACKey to enable
// the HMAC-SHA-256 variant, or Argon2idParams to enable the argon2id variant.
type OAuth2Secrets struct {
	// HMACKey is required by HashHMAC / VerifyHMAC. May be nil for SHA-256-only use.
	HMACKey []byte
	// Argon2idParams controls argon2id cost. nil uses argon2id.DefaultParams.
	Argon2idParams *argon2id.Params
}

type OAuth2SecretsOption func(*OAuth2Secrets)

// WithHMACKey configures the HMAC key for HashHMAC / VerifyHMAC.
func WithHMACKey(key []byte) OAuth2SecretsOption {
	return func(o *OAuth2Secrets) {
		o.HMACKey = key
	}
}

// WithArgon2idParams configures the argon2id cost parameters. Pass nil to use argon2id.DefaultParams.
func WithArgon2idParams(params *argon2id.Params) OAuth2SecretsOption {
	return func(o *OAuth2Secrets) {
		o.Argon2idParams = params
	}
}

// NewOAuth2Secrets returns an OAuth2Secrets with no HMAC key configured
// (SHA-256 operations only).
func NewOAuth2Secrets(opts ...OAuth2SecretsOption) *OAuth2Secrets {
	o := &OAuth2Secrets{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Generate creates a cryptographically random OAuth2 client secret from 32
// random bytes, returning both the base64url-encoded plaintext and its
// "$sha256$<hex>" tagged hash.
func (s *OAuth2Secrets) Generate() (*ClientSecret, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	secret := base64.RawURLEncoding.EncodeToString(raw)
	hash, err := s.HashSHA256(secret)
	if err != nil {
		return nil, err
	}
	return &ClientSecret{Secret: secret, Hash: hash}, nil
}

// HashSHA256 returns a "$sha256$<hex>" tagged digest of secret.
// It accepts any plaintext string (user-supplied or generated) and enforces:
//   - at least 32 characters
//   - Shannon entropy ≥ 3.5 bits/char (rejects repeated or predictable patterns)
func (s *OAuth2Secrets) HashSHA256(secret string) (string, error) {
	if err := validateSecretEntropy(secret); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(secret))
	return PrefixSHA256 + hex.EncodeToString(sum[:]), nil
}

// VerifySHA256 compares a plaintext secret against a "$sha256$<hex>" tagged
// hash using constant-time comparison. Returns false on any validation error
// or algorithm mismatch.
func (s *OAuth2Secrets) VerifySHA256(secret, hash string) bool {
	if !strings.HasPrefix(hash, PrefixSHA256) {
		return false
	}
	computed, err := s.HashSHA256(secret)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
}

// HashHMAC returns a "$hmac-sha256$<hex>" tagged HMAC-SHA-256 of secret using
// the receiver's HMACKey. Applies the same entropy validation as HashSHA256.
func (s *OAuth2Secrets) HashHMAC(secret string) (string, error) {
	if len(s.HMACKey) == 0 {
		return "", errors.New("HMAC key must not be empty")
	}
	if err := validateSecretEntropy(secret); err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, s.HMACKey)
	mac.Write([]byte(secret))
	return PrefixHMACSHA256 + hex.EncodeToString(mac.Sum(nil)), nil
}

// VerifyHMAC compares a plaintext secret against a "$hmac-sha256$<hex>" tagged
// hash using constant-time comparison.
func (s *OAuth2Secrets) VerifyHMAC(secret, hash string) bool {
	if !strings.HasPrefix(hash, PrefixHMACSHA256) {
		return false
	}
	computed, err := s.HashHMAC(secret)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
}

// HashArgon2id returns an argon2id PHC string (e.g. "$argon2id$v=19$...") for
// secret using the receiver's Argon2idParams (nil → argon2id.DefaultParams).
// Because argon2id salts each hash randomly the output is non-deterministic;
// use VerifyArgon2id to validate against a stored hash.
func (s *OAuth2Secrets) HashArgon2id(secret string) (string, error) {
	if err := validateSecretEntropy(secret); err != nil {
		return "", err
	}
	params := s.Argon2idParams
	if params == nil {
		params = argon2id.DefaultParams
	}
	return argon2id.CreateHash(secret, params)
}

// VerifyArgon2id compares a plaintext secret against a stored argon2id PHC
// hash. Returns false on prefix mismatch or any internal error.
func (s *OAuth2Secrets) VerifyArgon2id(secret, hash string) bool {
	if !strings.HasPrefix(hash, PrefixArgon2id) {
		return false
	}
	match, err := argon2id.ComparePasswordAndHash(secret, hash)
	if err != nil {
		return false
	}
	return match
}

// Verify auto-dispatches on the hash's algorithm prefix. Returns false for an
// unknown prefix or when the required key material (e.g. HMACKey) is absent.
func (s *OAuth2Secrets) Verify(secret, hash string) bool {
	switch DetectHashAlgorithm(hash) {
	case HashAlgorithmSHA256:
		return s.VerifySHA256(secret, hash)
	case HashAlgorithmHMACSHA256:
		return s.VerifyHMAC(secret, hash)
	case HashAlgorithmArgon2id:
		return s.VerifyArgon2id(secret, hash)
	default:
		return false
	}
}

// DetectAlgorithm is a convenience wrapper around the package-level
// DetectHashAlgorithm so callers holding only an OAuth2Secrets can inspect a
// hash without importing the helper separately.
func (s *OAuth2Secrets) DetectAlgorithm(hash string) HashAlgorithm {
	return DetectHashAlgorithm(hash)
}

// validateSecretEntropy enforces minimum length and Shannon entropy on secret.
func validateSecretEntropy(secret string) error {
	if len(secret) < minSecretLen {
		return fmt.Errorf("client secret must be at least %d characters, got %d", minSecretLen, len(secret))
	}
	if e := shannonEntropy(secret); e < minEntropyBitsPerChar {
		return fmt.Errorf("client secret entropy %.2f bits/char is below the required %.2f bits/char", e, minEntropyBitsPerChar)
	}
	return nil
}

// shannonEntropy returns the Shannon entropy of s in bits per byte.
func shannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	freq := make(map[byte]int)
	for i := 0; i < len(s); i++ {
		freq[s[i]]++
	}
	n := float64(len(s))
	var entropy float64
	for _, count := range freq {
		p := float64(count) / n
		entropy -= p * math.Log2(p)
	}
	return entropy
}
