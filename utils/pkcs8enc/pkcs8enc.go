// Package pkcs8enc encrypts and decrypts PKCS#8 private keys with PBES2 (RFC 8018):
// PBKDF2-HMAC-SHA256 key derivation and AES-256-CBC, in the standard
// "ENCRYPTED PRIVATE KEY" PEM form that OpenSSL and other tools read.
//
// It replaces OpenSSL's legacy "Proc-Type: 4,ENCRYPTED" PEM encryption for new keys. That
// format derives the key with a single unsalted-iteration MD5 pass (EVP_BytesToKey), so a
// leaked key file can be brute-forced at billions of guesses per second.
package pkcs8enc

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"hash"
)

// PEMType is the PEM block type of an encrypted PKCS#8 key.
const PEMType = "ENCRYPTED PRIVATE KEY"

// Iterations is the PBKDF2 iteration count used when encrypting (OWASP's current
// recommendation for PBKDF2-HMAC-SHA256).
const Iterations = 600_000

// maxIterations bounds what Decrypt will run, so a crafted file can't pin the CPU.
const maxIterations = 10_000_000

var (
	oidPBES2      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 13}
	oidPBKDF2     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 12}
	oidHMACSHA256 = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 9}
	oidAES256CBC  = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 42}
)

type encryptedPrivateKeyInfo struct {
	Algorithm     pkix.AlgorithmIdentifier
	EncryptedData []byte
}

type pbes2Params struct {
	KeyDerivationFunc pkix.AlgorithmIdentifier
	EncryptionScheme  pkix.AlgorithmIdentifier
}

type pbkdf2Params struct {
	Salt           []byte
	IterationCount int
	KeyLength      int                      `asn1:"optional"`
	PRF            pkix.AlgorithmIdentifier `asn1:"optional"`
}

// Encrypt returns pkcs8DER (as from x509.MarshalPKCS8PrivateKey) encrypted under password,
// as an "ENCRYPTED PRIVATE KEY" PEM block.
func Encrypt(pkcs8DER []byte, password string) (*pem.Block, error) {
	if password == "" {
		return nil, errors.New("pkcs8enc: empty password")
	}
	salt := make([]byte, 16)
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, Iterations, 32)
	if err != nil {
		return nil, err
	}

	padLen := aes.BlockSize - len(pkcs8DER)%aes.BlockSize
	padded := append(append([]byte{}, pkcs8DER...), bytes.Repeat([]byte{byte(padLen)}, padLen)...)
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	encrypted := make([]byte, len(padded))
	cipher.NewCBCEncrypter(blockCipher, iv).CryptBlocks(encrypted, padded)

	kdfParams, err := asn1.Marshal(pbkdf2Params{
		Salt:           salt,
		IterationCount: Iterations,
		PRF:            pkix.AlgorithmIdentifier{Algorithm: oidHMACSHA256, Parameters: asn1.NullRawValue},
	})
	if err != nil {
		return nil, err
	}
	ivParam, err := asn1.Marshal(iv)
	if err != nil {
		return nil, err
	}
	schemeParams, err := asn1.Marshal(pbes2Params{
		KeyDerivationFunc: pkix.AlgorithmIdentifier{Algorithm: oidPBKDF2, Parameters: asn1.RawValue{FullBytes: kdfParams}},
		EncryptionScheme:  pkix.AlgorithmIdentifier{Algorithm: oidAES256CBC, Parameters: asn1.RawValue{FullBytes: ivParam}},
	})
	if err != nil {
		return nil, err
	}
	der, err := asn1.Marshal(encryptedPrivateKeyInfo{
		Algorithm:     pkix.AlgorithmIdentifier{Algorithm: oidPBES2, Parameters: asn1.RawValue{FullBytes: schemeParams}},
		EncryptedData: encrypted,
	})
	if err != nil {
		return nil, err
	}
	return &pem.Block{Type: PEMType, Bytes: der}, nil
}

// Decrypt returns the PKCS#8 DER inside an "ENCRYPTED PRIVATE KEY" block. It supports
// PBES2 with PBKDF2 (HMAC-SHA256, or the RFC 8018 default HMAC-SHA1) and AES-256-CBC, and
// returns an error -- never panics -- on malformed input or a wrong password.
func Decrypt(block *pem.Block, password string) ([]byte, error) {
	if block.Type != PEMType {
		return nil, fmt.Errorf("pkcs8enc: PEM type %q, want %q", block.Type, PEMType)
	}
	var info encryptedPrivateKeyInfo
	if rest, err := asn1.Unmarshal(block.Bytes, &info); err != nil || len(rest) != 0 {
		return nil, errors.New("pkcs8enc: malformed EncryptedPrivateKeyInfo")
	}
	if !info.Algorithm.Algorithm.Equal(oidPBES2) {
		return nil, fmt.Errorf("pkcs8enc: unsupported encryption algorithm %v", info.Algorithm.Algorithm)
	}
	var scheme pbes2Params
	if _, err := asn1.Unmarshal(info.Algorithm.Parameters.FullBytes, &scheme); err != nil {
		return nil, errors.New("pkcs8enc: malformed PBES2 parameters")
	}
	if !scheme.KeyDerivationFunc.Algorithm.Equal(oidPBKDF2) {
		return nil, fmt.Errorf("pkcs8enc: unsupported key derivation %v", scheme.KeyDerivationFunc.Algorithm)
	}
	var kdf pbkdf2Params
	if _, err := asn1.Unmarshal(scheme.KeyDerivationFunc.Parameters.FullBytes, &kdf); err != nil {
		return nil, errors.New("pkcs8enc: malformed PBKDF2 parameters")
	}
	if kdf.IterationCount < 1 || kdf.IterationCount > maxIterations {
		return nil, fmt.Errorf("pkcs8enc: PBKDF2 iteration count %d out of range", kdf.IterationCount)
	}
	if !scheme.EncryptionScheme.Algorithm.Equal(oidAES256CBC) {
		return nil, fmt.Errorf("pkcs8enc: unsupported cipher %v", scheme.EncryptionScheme.Algorithm)
	}
	var iv []byte
	if _, err := asn1.Unmarshal(scheme.EncryptionScheme.Parameters.FullBytes, &iv); err != nil || len(iv) != aes.BlockSize {
		return nil, errors.New("pkcs8enc: malformed AES-256-CBC IV")
	}
	if len(info.EncryptedData) == 0 || len(info.EncryptedData)%aes.BlockSize != 0 {
		return nil, errors.New("pkcs8enc: encrypted data is not a whole number of AES blocks")
	}

	var key []byte
	var err error
	switch {
	case len(kdf.PRF.Algorithm) == 0: // absent: RFC 8018's default, hmacWithSHA1
		key, err = pbkdf2.Key(sha1New, password, kdf.Salt, kdf.IterationCount, 32)
	case kdf.PRF.Algorithm.Equal(oidHMACSHA256):
		key, err = pbkdf2.Key(sha256.New, password, kdf.Salt, kdf.IterationCount, 32)
	default:
		return nil, fmt.Errorf("pkcs8enc: unsupported PBKDF2 PRF %v", kdf.PRF.Algorithm)
	}
	if err != nil {
		return nil, err
	}

	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	decrypted := make([]byte, len(info.EncryptedData))
	cipher.NewCBCDecrypter(blockCipher, iv).CryptBlocks(decrypted, info.EncryptedData)
	padLen := int(decrypted[len(decrypted)-1])
	if padLen == 0 || padLen > aes.BlockSize || !bytes.Equal(decrypted[len(decrypted)-padLen:], bytes.Repeat([]byte{byte(padLen)}, padLen)) {
		return nil, errors.New("pkcs8enc: decryption failed (wrong password?)")
	}
	return decrypted[:len(decrypted)-padLen], nil
}

func sha1New() hash.Hash { return sha1.New() }
