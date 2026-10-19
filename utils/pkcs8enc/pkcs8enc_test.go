package pkcs8enc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"
)

func encryptedKey(t *testing.T) (*pem.Block, []byte) {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(k)
	require.NoError(t, err)
	b, err := Encrypt(der, "s3cret")
	require.NoError(t, err)
	return b, der
}

func TestRoundTrip(t *testing.T) {
	b, der := encryptedKey(t)
	require.Equal(t, PEMType, b.Type)
	got, err := Decrypt(b, "s3cret")
	require.NoError(t, err)
	require.Equal(t, der, got)

	_, err = Decrypt(b, "wrong")
	require.Error(t, err)
	_, err = Encrypt(der, "")
	require.Error(t, err, "an empty password is not encryption")
}

// Malformed input must be an error, never a panic.
func TestDecrypt_MalformedInput(t *testing.T) {
	b, _ := encryptedKey(t)
	for name, bytes := range map[string][]byte{
		"empty":     {},
		"garbage":   []byte("not asn1 at all"),
		"truncated": b.Bytes[:len(b.Bytes)/2],
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Decrypt(&pem.Block{Type: PEMType, Bytes: bytes}, "s3cret")
			require.Error(t, err)
		})
	}
	// Ciphertext cut to a non-multiple of the block size.
	var info encryptedPrivateKeyInfo
	_, err := asn1Unmarshal(b.Bytes, &info)
	require.NoError(t, err)
	info.EncryptedData = info.EncryptedData[:len(info.EncryptedData)-3]
	_, err = Decrypt(&pem.Block{Type: PEMType, Bytes: mustMarshal(t, info)}, "s3cret")
	require.Error(t, err)

	_, err = Decrypt(&pem.Block{Type: "PRIVATE KEY", Bytes: b.Bytes}, "s3cret")
	require.Error(t, err, "wrong PEM type")
}
