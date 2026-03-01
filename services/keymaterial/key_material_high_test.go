package keymaterial

import (
	"encoding/pem"
	"testing"
)

// TestDecryptPEM_InvalidPEM verifies DecryptPEM returns error for invalid PEM.
func TestDecryptPEM_InvalidPEM(t *testing.T) {
	_, err := DecryptPEM([]byte("not a pem block"), []byte("password"))
	if err == nil {
		t.Fatal("expected error for invalid PEM data")
	}
}

// TestDecryptPEM_Unencrypted verifies DecryptPEM returns unencrypted PEM as-is.
func TestDecryptPEM_Unencrypted(t *testing.T) {
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "TEST KEY",
		Bytes: []byte("test data here"),
	})
	result, err := DecryptPEM(pemData, []byte("password"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return original PEM unchanged
	block, _ := pem.Decode(result)
	if block == nil {
		t.Fatal("expected valid PEM block in result")
	}
	if string(block.Bytes) != "test data here" {
		t.Fatalf("expected original data, got %q", string(block.Bytes))
	}
}

// TestDecryptPEM_UnsupportedCipher verifies DecryptPEM rejects non-AES-256-CBC ciphers.
func TestDecryptPEM_UnsupportedCipher(t *testing.T) {
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "TEST KEY",
		Bytes: []byte("fake encrypted data"),
		Headers: map[string]string{
			"Proc-Type": "4,ENCRYPTED",
			"DEK-Info":  "DES-CBC,AABBCCDD",
		},
	})
	_, err := DecryptPEM(pemData, []byte("password"))
	if err == nil {
		t.Fatal("expected error for unsupported cipher")
	}
}

// TestDecryptPEM_InvalidIV verifies DecryptPEM rejects malformed IV.
func TestDecryptPEM_InvalidIV(t *testing.T) {
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "TEST KEY",
		Bytes: []byte("fake encrypted data"),
		Headers: map[string]string{
			"Proc-Type": "4,ENCRYPTED",
			"DEK-Info":  "AES-256-CBC,ZZZZ",
		},
	})
	_, err := DecryptPEM(pemData, []byte("password"))
	if err == nil {
		t.Fatal("expected error for invalid IV")
	}
}

// TestDecryptPEM_BadPadding verifies that invalid encrypted data
// is detected rather than causing a panic.
func TestDecryptPEM_BadPadding(t *testing.T) {
	// Test 1: empty encrypted data should be caught
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "TEST KEY",
		Bytes: []byte{}, // empty
		Headers: map[string]string{
			"Proc-Type": "4,ENCRYPTED",
			"DEK-Info":  "AES-256-CBC,00112233445566778899AABBCCDDEEFF",
		},
	})
	_, err := DecryptPEM(pemData, []byte("password"))
	if err == nil {
		t.Fatal("expected error for empty encrypted data")
	}

	// Test 2: data not a multiple of block size should be caught
	pemData = pem.EncodeToMemory(&pem.Block{
		Type:  "TEST KEY",
		Bytes: []byte("odd-length-data"), // 15 bytes, not multiple of 16
		Headers: map[string]string{
			"Proc-Type": "4,ENCRYPTED",
			"DEK-Info":  "AES-256-CBC,00112233445566778899AABBCCDDEEFF",
		},
	})
	_, err = DecryptPEM(pemData, []byte("password"))
	if err == nil {
		t.Fatal("expected error for non-aligned encrypted data")
	}
}
