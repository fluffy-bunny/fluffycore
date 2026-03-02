package nats_token

import (
	"testing"
)

// TestCreateNATSConnectTokenClientCredentials_Success verifies normal token creation.
func TestCreateNATSConnectTokenClientCredentials_Success(t *testing.T) {
	token, err := CreateNATSConnectTokenClientCredentials(&CreateNATSConnectTokenClientCredentialsRequest{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Account:      "test-account",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Verify round-trip
	decoded, err := DecodeNATSConnectTokenClientCredentials(token)
	if err != nil {
		t.Fatalf("failed to decode token: %v", err)
	}
	if decoded.ClientID != "test-client" {
		t.Fatalf("expected ClientID 'test-client', got '%s'", decoded.ClientID)
	}
	if decoded.ClientSecret != "test-secret" {
		t.Fatalf("expected ClientSecret 'test-secret', got '%s'", decoded.ClientSecret)
	}
	if decoded.Account != "test-account" {
		t.Fatalf("expected Account 'test-account', got '%s'", decoded.Account)
	}
}

// TestCreateNatsConnectionWithClientCredentials_BadURL verifies error propagation
// when connection to NATS server fails.
func TestCreateNatsConnectionWithClientCredentials_BadURL(t *testing.T) {
	_, err := CreateNatsConnectionWithClientCredentials(&NATSConnectTokenClientCredentialsRequest{
		NATSUrl:      "nats://invalid-host-that-does-not-exist:4222",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Account:      "test-account",
	})
	// Should fail to connect (not panic or silently succeed)
	if err == nil {
		t.Fatal("expected error connecting to invalid NATS URL")
	}
}
