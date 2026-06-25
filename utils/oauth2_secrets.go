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
)

const (
	// minSecretLen is the minimum character length for any client secret.
	minSecretLen = 32
	// minEntropyBitsPerChar is the Shannon entropy floor (bits/char).
	// Rejects low-entropy secrets such as repeated characters or dictionary words.
	minEntropyBitsPerChar = 3.5
)

// ClientSecret holds a generated plaintext secret and its SHA-256 hex hash.
type ClientSecret struct {
	// Secret is the base64url-encoded plaintext. Distribute this to the client once.
	Secret string `json:"secret"`
	// Hash is the hex-encoded SHA-256 digest. Store this server-side.
	Hash string `json:"hash"`
}

// GenerateClientSecret creates a cryptographically random OAuth2 client secret
// from 32 random bytes, returning both the base64url-encoded plaintext
// and its SHA-256 hex hash.
func GenerateClientSecret() (*ClientSecret, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	secret := base64.RawURLEncoding.EncodeToString(raw)
	hash, err := HashClientSecret(secret)
	if err != nil {
		return nil, err
	}
	return &ClientSecret{Secret: secret, Hash: hash}, nil
}

// HashClientSecret returns the hex-encoded SHA-256 hash of secret.
// It accepts any plaintext string (user-supplied or generated) and enforces:
//   - at least 32 characters
//   - Shannon entropy ≥ 3.5 bits/char (rejects repeated or predictable patterns)
func HashClientSecret(secret string) (string, error) {
	if err := validateSecretEntropy(secret); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:]), nil
}

// VerifyClientSecret compares a plaintext secret against a previously stored
// hex-encoded SHA-256 hash using constant-time comparison.
// Returns false on any validation error.
func VerifyClientSecret(secret, hash string) bool {
	computed, err := HashClientSecret(secret)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
}

// HashClientSecretHMAC returns a hex-encoded HMAC-SHA-256 of secret using the
// provided key. Applies the same entropy validation as HashClientSecret.
func HashClientSecretHMAC(secret string, key []byte) (string, error) {
	if len(key) == 0 {
		return "", errors.New("HMAC key must not be empty")
	}
	if err := validateSecretEntropy(secret); err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(secret))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// VerifyClientSecretHMAC compares a plaintext secret against a stored
// hex-encoded HMAC-SHA-256 hash using constant-time comparison.
func VerifyClientSecretHMAC(secret, hash string, key []byte) bool {
	computed, err := HashClientSecretHMAC(secret, key)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
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
