package ecdsa

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDecodePrivatePem_InvalidPEM verifies that DecodePrivatePem returns
// an error for invalid PEM data instead of panicking with nil dereference.
// (Was CRITICAL: pem.Decode returning nil was not checked.)
func TestDecodePrivatePem_InvalidPEM(t *testing.T) {
	privateKey, publicKey, err := DecodePrivatePem("", "not a valid PEM")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode PEM block")
	require.Nil(t, privateKey)
	require.Nil(t, publicKey)
}

// TestDecodePrivatePem_InvalidKey verifies that DecodePrivatePem returns
// an error for valid PEM but invalid EC key data instead of panicking.
// (Was CRITICAL: x509.ParseECPrivateKey error was silently ignored.)
func TestDecodePrivatePem_InvalidKey(t *testing.T) {
	fakePEM := "-----BEGIN EC PRIVATE KEY-----\nYm9ndXMgZGF0YQ==\n-----END EC PRIVATE KEY-----"
	privateKey, publicKey, err := DecodePrivatePem("", fakePEM)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse EC private key")
	require.Nil(t, privateKey)
	require.Nil(t, publicKey)
}

// TestDecodePrivatePem_ValidKey verifies normal operation with a valid key.
func TestDecodePrivatePem_ValidKey(t *testing.T) {
	privateKey, privateEncoded, _, err := GenerateECDSAPublicPrivateKeySet("")
	require.NoError(t, err)
	require.NotNil(t, privateKey)

	decodedPriv, decodedPub, err := DecodePrivatePem("", privateEncoded)
	require.NoError(t, err)
	require.NotNil(t, decodedPriv)
	require.NotNil(t, decodedPub)
}

// TestDecode_InvalidPrivatePEM verifies decode handles invalid private PEM.
func TestDecode_InvalidPrivatePEM(t *testing.T) {
	priv, pub, err := decode("", "not valid", "also not valid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode private PEM block")
	require.Nil(t, priv)
	require.Nil(t, pub)
}

// TestDecode_InvalidPublicPEM verifies decode handles invalid public PEM.
func TestDecode_InvalidPublicPEM(t *testing.T) {
	// Generate valid private key
	_, privEncoded, _, err := GenerateECDSAPublicPrivateKeySet("")
	require.NoError(t, err)

	priv, pub, err := decode("", privEncoded, "not valid public pem")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode public PEM block")
	require.Nil(t, priv)
	require.Nil(t, pub)
}

// TestRoundTrip_EncodeDecodePrivatePem verifies encode/decode round-trip.
func TestRoundTrip_EncodeDecodePrivatePem(t *testing.T) {
	privateKey, privEncoded, pubEncoded, err := GenerateECDSAPublicPrivateKeySet("")
	require.NoError(t, err)

	decodedPriv, decodedPub, err := decode("", privEncoded, pubEncoded)
	require.NoError(t, err)
	require.True(t, privateKey.Equal(decodedPriv))
	require.True(t, privateKey.PublicKey.Equal(decodedPub))
}

// TestRoundTrip_WithPassword verifies encode/decode with password.
func TestRoundTrip_WithPassword(t *testing.T) {
	_, privEncoded, _, err := GenerateECDSAPublicPrivateKeySet("testpassword")
	require.NoError(t, err)

	decodedPriv, decodedPub, err := DecodePrivatePem("testpassword", privEncoded)
	require.NoError(t, err)
	require.NotNil(t, decodedPriv)
	require.NotNil(t, decodedPub)
}
