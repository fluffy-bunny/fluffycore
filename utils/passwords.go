package utils

import (
	"crypto/rand"
	"encoding/base64"

	argon2id "github.com/alexedwards/argon2id"
)

type (
	PasswordHashSet struct {
		Hash     string `json:"hash"`
		Password string `json:"password"`
	}
)

// GeneratePasswordHash creates an argon2id hash of the given password.
func GeneratePasswordHash(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hash, nil
}

// ComparePasswordHash compares a plaintext password against an argon2id hash.
func ComparePasswordHash(password string, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

// generatedPasswordBytes is the entropy of a GeneratePassword result: 256 bits.
const generatedPasswordBytes = 32

// GeneratePassword returns a random password: 32 bytes from crypto/rand, base64url-encoded
// without padding (43 URL-safe characters).
//
// It previously returned an xid, which is a 4-byte timestamp, 3-byte machine id, 2-byte pid
// and 3-byte counter -- given one generated password (or just the host and approximate
// time), others are trivially guessable.
func GeneratePassword() (string, error) {
	b := make([]byte, generatedPasswordBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GeneratePasswordHashSet generates a password (or uses the provided secret) and returns
// both the password and its argon2id hash.
func GeneratePasswordHashSet(secret *string) (*PasswordHashSet, error) {
	var pass string
	var err error
	if !IsEmptyOrNil(secret) && !IsEmptyOrNil(*secret) {
		pass = *secret
	} else {
		pass, err = GeneratePassword()
		if err != nil {
			return nil, err
		}
	}

	hash, err := GeneratePasswordHash(pass)
	if err != nil {
		return nil, err
	}
	return &PasswordHashSet{
		Hash:     hash,
		Password: pass,
	}, nil

}
