package ecdsa

import (
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	pkcs8enc "github.com/fluffy-bunny/fluffycore/utils/pkcs8enc"
	"github.com/stretchr/testify/require"
)

// With a password, Encode used to write legacy OpenSSL PEM encryption (single-MD5 KDF).
func TestEncode_WithPasswordWritesEncryptedPKCS8(t *testing.T) {
	key, priv, pub, err := GenerateECDSAPublicPrivateKeySet("pw")
	require.NoError(t, err)
	block, _ := pem.Decode([]byte(priv))
	require.Equal(t, pkcs8enc.PEMType, block.Type)
	require.NotContains(t, priv, "Proc-Type")

	got, _, err := decode("pw", priv, pub)
	require.NoError(t, err)
	require.True(t, got.Equal(key))
	_, _, err = DecodePrivatePem("wrong", priv)
	require.Error(t, err)
}

// Keys encrypted the old way must still load.
func TestDecodePrivatePem_LegacyEncryptedStillReadable(t *testing.T) {
	key, _, _, err := GenerateECDSAPublicPrivateKeySet("")
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	block := &pem.Block{Type: "EC PRIVATE KEY", Bytes: der}
	require.NoError(t, EncryptPemBlock(block, "pw", x509.PEMCipherAES256))

	got, _, err := DecodePrivatePem("pw", string(pem.EncodeToMemory(block)))
	require.NoError(t, err)
	require.True(t, got.Equal(key))
}

// These used to panic: iv[:8] on a short IV, and the CBC decrypter on a bad IV/data length.
func TestDecryptPemBlock_MalformedDoesNotPanic(t *testing.T) {
	for name, block := range map[string]*pem.Block{
		"short IV":      {Headers: map[string]string{"Proc-Type": "4,ENCRYPTED", "DEK-Info": "AES-256-CBC,0011"}, Bytes: make([]byte, 16)},
		"long IV":       {Headers: map[string]string{"Proc-Type": "4,ENCRYPTED", "DEK-Info": "AES-256-CBC," + strings.Repeat("00", 20)}, Bytes: make([]byte, 16)},
		"partial block": {Headers: map[string]string{"Proc-Type": "4,ENCRYPTED", "DEK-Info": "AES-256-CBC," + strings.Repeat("00", 16)}, Bytes: make([]byte, 17)},
		"empty data":    {Headers: map[string]string{"Proc-Type": "4,ENCRYPTED", "DEK-Info": "AES-256-CBC," + strings.Repeat("00", 16)}},
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, DecryptPemBlock(block, "pw"))
		})
	}
}
