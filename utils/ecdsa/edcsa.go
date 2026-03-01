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

func DecodePrivatePem(password string, pemEncoded string) (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemEncoded))
	if block == nil {
		return nil, nil, errors.New("ecdsa: failed to decode PEM block")
	}
	if password != "" {
		if err := DecryptPemBlock(block, password); err != nil {
			return nil, nil, err
		}
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("ecdsa: failed to parse EC private key: %w", err)
	}
	publicKey := &privateKey.PublicKey

	return privateKey, publicKey, nil
}

func decode(password string, pemEncoded string, pemEncodedPub string) (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemEncoded))
	if block == nil {
		return nil, nil, errors.New("ecdsa: failed to decode private PEM block")
	}
	if password != "" {
		if err := DecryptPemBlock(block, password); err != nil {
			return nil, nil, err
		}
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("ecdsa: failed to parse EC private key: %w", err)
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
	x509Encoded, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", "", err
	}
	// Convert it to pem
	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: x509Encoded,
	}
	var pemEncoded []byte
	// Encrypt the pem
	if password != "" {
		err = EncryptPemBlock(block, password, x509.PEMCipherAES256)

		if err != nil {
			return "", "", err
		}
		//	DecryptedPEMBlock, _ := x509.DecryptPEMBlock(block, []byte(password))
		//	fmt.Println(string(DecryptedPEMBlock))
	}
	pemEncoded = pem.EncodeToMemory(block)

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
