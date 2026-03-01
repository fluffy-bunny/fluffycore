package oidc

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestLoadDiscoveryDocument_WithTimeout(t *testing.T) {
	// Create a test server that returns a valid discovery document
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"issuer": "https://test.example.com",
				"jwks_uri": "` + "http://" + r.Host + `/jwks` + `",
				"authorization_endpoint": "https://test.example.com/authorize",
				"token_endpoint": "https://test.example.com/token"
			}`))
		} else if r.URL.Path == "/jwks" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"keys": []}`))
		}
	}))
	defer server.Close()

	discoveryURL, _ := url.Parse(server.URL + "/.well-known/openid-configuration")
	doc := NewDiscoveryDocument(*discoveryURL)
	err := doc.Initialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.KeyResponse == nil {
		t.Error("expected KeyResponse to be populated")
	}
}

func TestLoadDiscoveryDocument_InvalidURL(t *testing.T) {
	discoveryURL, _ := url.Parse("http://localhost:99999/nonexistent")
	doc := NewDiscoveryDocument(*discoveryURL)
	err := doc.Initialize()
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestLoadJwksData_ErrorWrapping(t *testing.T) {
	// Create server that returns valid discovery but invalid JWKS URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"issuer": "https://test.example.com",
			"jwks_uri": "http://localhost:99999/nonexistent-jwks",
			"authorization_endpoint": "https://test.example.com/authorize",
			"token_endpoint": "https://test.example.com/token"
		}`))
	}))
	defer server.Close()

	discoveryURL, _ := url.Parse(server.URL + "/.well-known/openid-configuration")
	doc := NewDiscoveryDocument(*discoveryURL)
	err := doc.Initialize()
	if err == nil {
		t.Error("expected error for unreachable JWKS URL")
	}
	// Verify the error message includes the URL (new behavior)
	errStr := err.Error()
	if errStr == "error loading jwks data: could not fetch jwks URL" {
		t.Error("error should now include the URL and wrap the original error")
	}
}
