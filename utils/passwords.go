package utils

import (
	argon2id "github.com/alexedwards/argon2id"
	xid "github.com/rs/xid"
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

const special string = "!@#$%&*"

func GeneratePassword() (string, error) {
	guid := xid.New().String()
	return guid, nil

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
