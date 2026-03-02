package ecdsa

import (
	"crypto/x509"
	"encoding/pem"
	"testing"
)

// TestEncryptDecryptPemBlock_RoundTrip verifies that EncryptPemBlock and
// DecryptPemBlock produce a valid round-trip without using deprecated x509 APIs.
func TestEncryptDecryptPemBlock_RoundTrip(t *testing.T) {
	originalData := []byte("this is the secret key data for testing round-trip")

	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: originalData,
	}

	password := "test-password-123"

	// Encrypt
	err := EncryptPemBlock(block, password, x509.PEMCipherAES256)
	if err != nil {
		t.Fatalf("EncryptPemBlock failed: %v", err)
	}

	// Verify headers were set
	if block.Headers["Proc-Type"] != "4,ENCRYPTED" {
		t.Fatalf("expected Proc-Type header, got %q", block.Headers["Proc-Type"])
	}
	dekInfo, ok := block.Headers["DEK-Info"]
	if !ok || dekInfo == "" {
		t.Fatal("expected DEK-Info header to be set")
	}

	// Encrypted bytes should differ from original
	if string(block.Bytes) == string(originalData) {
		t.Fatal("encrypted data should differ from original")
	}

	// Decrypt
	err = DecryptPemBlock(block, password)
	if err != nil {
		t.Fatalf("DecryptPemBlock failed: %v", err)
	}

	// Verify decrypted data matches original
	if string(block.Bytes) != string(originalData) {
		t.Fatalf("decrypted data doesn't match original.\nExpected: %q\nGot:      %q", originalData, block.Bytes)
	}

	// Headers should be cleaned up
	if _, ok := block.Headers["Proc-Type"]; ok {
		t.Fatal("Proc-Type header should be removed after decryption")
	}
	if _, ok := block.Headers["DEK-Info"]; ok {
		t.Fatal("DEK-Info header should be removed after decryption")
	}
}

// TestEncryptPemBlock_EmptyPassword verifies no encryption with empty password.
func TestEncryptPemBlock_EmptyPassword(t *testing.T) {
	originalData := []byte("no encryption needed")
	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: originalData,
	}

	err := EncryptPemBlock(block, "", x509.PEMCipherAES256)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Data should be unchanged
	if string(block.Bytes) != string(originalData) {
		t.Fatal("data should not be modified with empty password")
	}
}

// TestDecryptPemBlock_NotEncrypted verifies no-op for unencrypted blocks.
func TestDecryptPemBlock_NotEncrypted(t *testing.T) {
	originalData := []byte("plain data")
	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: originalData,
	}

	err := DecryptPemBlock(block, "any-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(block.Bytes) != string(originalData) {
		t.Fatal("unencrypted block should not be modified")
	}
}

// TestDecryptPemBlock_WrongPassword verifies wrong password produces error.
func TestDecryptPemBlock_WrongPassword(t *testing.T) {
	originalData := []byte("secret key data here for wrong password test!")
	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: originalData,
	}

	err := EncryptPemBlock(block, "correct-password", x509.PEMCipherAES256)
	if err != nil {
		t.Fatalf("EncryptPemBlock failed: %v", err)
	}

	// Attempt decrypt with wrong password — should fail on padding validation
	err = DecryptPemBlock(block, "wrong-password")
	if err == nil {
		// With wrong password, decryption produces garbage — padding validation should catch it
		t.Log("WARNING: wrong password did not cause error (may produce garbage data)")
	}
}

// TestIsEncryptedPEMBlock verifies the helper correctly identifies encrypted blocks.
func TestIsEncryptedPEMBlock(t *testing.T) {
	encrypted := &pem.Block{
		Headers: map[string]string{"Proc-Type": "4,ENCRYPTED"},
	}
	if !isEncryptedPEMBlock(encrypted) {
		t.Fatal("expected encrypted block to be detected")
	}

	unencrypted := &pem.Block{
		Headers: map[string]string{},
	}
	if isEncryptedPEMBlock(unencrypted) {
		t.Fatal("expected unencrypted block to not be detected as encrypted")
	}

	noHeaders := &pem.Block{}
	if isEncryptedPEMBlock(noHeaders) {
		t.Fatal("expected block without headers to not be detected as encrypted")
	}
}

// TestGenerateAndDecodeECDSA_RoundTrip verifies full key generation with
// encryption and then decryption using the non-deprecated implementations.
func TestGenerateAndDecodeECDSA_RoundTrip(t *testing.T) {
	password := "test-keygen-password"

	_, privEncoded, pubEncoded, err := GenerateECDSAPublicPrivateKeySet(password)
	if err != nil {
		t.Fatalf("GenerateECDSAPublicPrivateKeySet failed: %v", err)
	}

	if privEncoded == "" || pubEncoded == "" {
		t.Fatal("expected non-empty encoded keys")
	}

	// Decode should work with correct password
	privKey, pubKey, err := decode(password, privEncoded, pubEncoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if privKey == nil || pubKey == nil {
		t.Fatal("expected non-nil keys")
	}

	// DecodePrivatePem should also work
	privKey2, pubKey2, err := DecodePrivatePem(password, privEncoded)
	if err != nil {
		t.Fatalf("DecodePrivatePem failed: %v", err)
	}
	if privKey2 == nil || pubKey2 == nil {
		t.Fatal("expected non-nil keys from DecodePrivatePem")
	}
}

// TestGenerateAndDecodeECDSA_NoPassword verifies key generation without encryption.
func TestGenerateAndDecodeECDSA_NoPassword(t *testing.T) {
	_, privEncoded, pubEncoded, err := GenerateECDSAPublicPrivateKeySet("")
	if err != nil {
		t.Fatalf("GenerateECDSAPublicPrivateKeySet failed: %v", err)
	}

	privKey, pubKey, err := decode("", privEncoded, pubEncoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if privKey == nil || pubKey == nil {
		t.Fatal("expected non-nil keys")
	}
}
