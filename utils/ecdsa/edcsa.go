package ecdsa

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	pkcs8enc "github.com/fluffy-bunny/fluffycore/utils/pkcs8enc"
)

// generateKeyFromPassword derives a key from password using OpenSSL's EVP_BytesToKey (MD5-based).
func generateKeyFromPassword(password, salt []byte, keyLen int) []byte {
	var result []byte
	hash := md5.New()
	hash.Write(password)
	hash.Write(salt)
	result = hash.Sum(nil)
	for len(result) < keyLen {
		hash.Reset()
		hash.Write(result)
		hash.Write(password)
		hash.Write(salt)
		result = append(result, hash.Sum(nil)...)
	}
	return result[:keyLen]
}

// isEncryptedPEMBlock checks if a PEM block is encrypted by looking at headers.
func isEncryptedPEMBlock(b *pem.Block) bool {
	procType, ok := b.Headers["Proc-Type"]
	if !ok {
		return false
	}
	return strings.Contains(procType, "ENCRYPTED")
}

// EncryptPemBlock encrypts a PEM block in-place using AES-256-CBC with OpenSSL-compatible key derivation.
//
// Deprecated: the legacy OpenSSL PEM format derives the key with one MD5 pass
// (EVP_BytesToKey) and is trivially brute-forced. Encode now writes encrypted PKCS#8
// (utils/pkcs8enc). Kept, with DecryptPemBlock, to read existing keys.
func EncryptPemBlock(block *pem.Block, password string, alg x509.PEMCipher) error {
	if len(password) > 0 {
		// Generate random IV (AES block size = 16 bytes)
		iv := make([]byte, aes.BlockSize)
		if _, err := rand.Read(iv); err != nil {
			return fmt.Errorf("failed to generate IV: %w", err)
		}

		// Derive key from password + IV salt using EVP_BytesToKey
		key := generateKeyFromPassword([]byte(password), iv[:8], 32) // AES-256 = 32 bytes

		// PKCS7 pad the data
		blockSize := aes.BlockSize
		paddingLen := blockSize - len(block.Bytes)%blockSize
		padded := make([]byte, len(block.Bytes)+paddingLen)
		copy(padded, block.Bytes)
		for i := len(block.Bytes); i < len(padded); i++ {
			padded[i] = byte(paddingLen)
		}

		// Encrypt with AES-256-CBC
		blockCipher, err := aes.NewCipher(key)
		if err != nil {
			return fmt.Errorf("failed to create cipher: %w", err)
		}
		mode := cipher.NewCBCEncrypter(blockCipher, iv)
		encrypted := make([]byte, len(padded))
		mode.CryptBlocks(encrypted, padded)

		// Set PEM headers for OpenSSL compatibility
		if block.Headers == nil {
			block.Headers = make(map[string]string)
		}
		block.Headers["Proc-Type"] = "4,ENCRYPTED"
		block.Headers["DEK-Info"] = "AES-256-CBC," + strings.ToUpper(hex.EncodeToString(iv))
		block.Bytes = encrypted
	}
	return nil
}

// DecryptPemBlock decrypts a PEM block in-place using AES-256-CBC with OpenSSL-compatible key derivation.
func DecryptPemBlock(block *pem.Block, password string) (err error) {
	if isEncryptedPEMBlock(block) {
		dekInfo, ok := block.Headers["DEK-Info"]
		if !ok {
			return fmt.Errorf("encrypted PEM block missing DEK-Info header")
		}
		parts := strings.SplitN(dekInfo, ",", 2)
		if len(parts) != 2 {
			return fmt.Errorf("malformed DEK-Info header")
		}
		cipherName := parts[0]
		if cipherName != "AES-256-CBC" {
			return fmt.Errorf("unsupported cipher: %s", cipherName)
		}
		iv, err := hex.DecodeString(parts[1])
		if err != nil {
			return fmt.Errorf("invalid IV in DEK-Info: %w", err)
		}
		// Both used to be unchecked: a short IV panicked on iv[:8], and a wrong-length IV or a
		// ciphertext that isn't whole AES blocks panicked inside the CBC decrypter.
		if len(iv) != aes.BlockSize {
			return fmt.Errorf("invalid IV length: expected %d, got %d", aes.BlockSize, len(iv))
		}
		if len(block.Bytes) == 0 || len(block.Bytes)%aes.BlockSize != 0 {
			return fmt.Errorf("encrypted data length %d is not a multiple of the AES block size", len(block.Bytes))
		}

		key := generateKeyFromPassword([]byte(password), iv[:8], 32)

		blockCipher, err := aes.NewCipher(key)
		if err != nil {
			return fmt.Errorf("failed to create cipher: %w", err)
		}
		mode := cipher.NewCBCDecrypter(blockCipher, iv)
		decrypted := make([]byte, len(block.Bytes))
		mode.CryptBlocks(decrypted, block.Bytes)

		// Remove PKCS7 padding
		if len(decrypted) == 0 {
			return fmt.Errorf("decrypted data is empty")
		}
		paddingLen := int(decrypted[len(decrypted)-1])
		if paddingLen == 0 || paddingLen > aes.BlockSize || paddingLen > len(decrypted) {
			return fmt.Errorf("invalid PKCS7 padding length: %d", paddingLen)
		}
		for i := 0; i < paddingLen; i++ {
			if decrypted[len(decrypted)-1-i] != byte(paddingLen) {
				return fmt.Errorf("invalid PKCS7 padding at byte %d", i)
			}
		}
		decrypted = decrypted[:len(decrypted)-paddingLen]

		delete(block.Headers, "Proc-Type")
		delete(block.Headers, "DEK-Info")
		block.Bytes = decrypted
	}
	return nil
}

// parsePrivateBlock reads an EC private key from an encrypted PKCS#8 block (what Encode
// writes now), a legacy OpenSSL-encrypted block, or an unencrypted SEC1 block.
func parsePrivateBlock(block *pem.Block, password string) (*ecdsa.PrivateKey, error) {
	if block.Type == pkcs8enc.PEMType {
		der, err := pkcs8enc.Decrypt(block, password)
		if err != nil {
			return nil, fmt.Errorf("ecdsa: %w", err)
		}
		key, err := x509.ParsePKCS8PrivateKey(der)
		if err != nil {
			return nil, fmt.Errorf("ecdsa: failed to parse PKCS#8 private key: %w", err)
		}
		ecKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("ecdsa: private key is %T, not ECDSA", key)
		}
		return ecKey, nil
	}
	if password != "" {
		if err := DecryptPemBlock(block, password); err != nil {
			return nil, err
		}
	}
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("ecdsa: failed to parse EC private key: %w", err)
	}
	return privateKey, nil
}

func DecodePrivatePem(password string, pemEncoded string) (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemEncoded))
	if block == nil {
		return nil, nil, errors.New("ecdsa: failed to decode PEM block")
	}
	privateKey, err := parsePrivateBlock(block, password)
	if err != nil {
		return nil, nil, err
	}
	publicKey := &privateKey.PublicKey

	return privateKey, publicKey, nil
}

func decode(password string, pemEncoded string, pemEncodedPub string) (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemEncoded))
	if block == nil {
		return nil, nil, errors.New("ecdsa: failed to decode private PEM block")
	}
	privateKey, err := parsePrivateBlock(block, password)
	if err != nil {
		return nil, nil, err
	}

	blockPub, _ := pem.Decode([]byte(pemEncodedPub))
	if blockPub == nil {
		return nil, nil, errors.New("ecdsa: failed to decode public PEM block")
	}
	genericPublicKey, err := x509.ParsePKIXPublicKey(blockPub.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("ecdsa: failed to parse public key: %w", err)
	}
	publicKey, ok := genericPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, nil, errors.New("ecdsa: public key is not an ECDSA key")
	}

	return privateKey, publicKey, nil
}
func Encode(password string, privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) (string, string, error) {
	var block *pem.Block
	if password != "" {
		// Encrypted PKCS#8 (PBES2, PBKDF2-HMAC-SHA256). This used to be the legacy OpenSSL
		// PEM encryption, whose single-MD5 key derivation is trivially brute-forced.
		pkcs8DER, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return "", "", err
		}
		if block, err = pkcs8enc.Encrypt(pkcs8DER, password); err != nil {
			return "", "", err
		}
	} else {
		x509Encoded, err := x509.MarshalECPrivateKey(privateKey)
		if err != nil {
			return "", "", err
		}
		block = &pem.Block{Type: "EC PRIVATE KEY", Bytes: x509Encoded}
	}
	pemEncoded := pem.EncodeToMemory(block)

	x509EncodedPub, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", "", err
	}
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "EC  PUBLIC KEY", Bytes: x509EncodedPub})

	return string(pemEncoded), string(pemEncodedPub), nil
}

func pubBytes(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(elliptic.P256(), pub.X, pub.Y)
}

func GenerateECDSAPublicPrivateKeySet(password string) (privateKey *ecdsa.PrivateKey, privateEncoded string, publicEncoded string, err error) {
	privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", "", err
	}

	privateEncoded, publicEncoded, err = Encode(password, privateKey, &privateKey.PublicKey)
	return
}
