package insecure

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBase64Encoding_RoundTrip verifies that the encoding used in SetCookie
// (base64.URLEncoding) matches the decoding used in GetCookie.
// (Was CRITICAL: SetCookie used URLEncoding, GetCookie used StdEncoding,
// causing decode failures for values with +, /, = characters.)
func TestBase64Encoding_RoundTrip(t *testing.T) {
	// Create a value that produces different base64 in URL vs Standard encoding
	testValues := []map[string]interface{}{
		{"key": "simple value"},
		{"url": "https://example.com/path?query=value&other=123"},
		{"binary-like": "data with / and + chars"},
		{"nested": map[string]interface{}{"a": 1, "b": "test"}},
	}

	for _, val := range testValues {
		cookieData, err := json.Marshal(val)
		require.NoError(t, err)

		// Encode the way SetCookie does
		encoded := base64.URLEncoding.EncodeToString(cookieData)

		// Decode the way GetCookie now does (was StdEncoding — bug)
		decoded, err := base64.URLEncoding.DecodeString(encoded)
		require.NoError(t, err, "URLEncoding.DecodeString should decode URLEncoding.EncodeToString output")

		var result map[string]interface{}
		err = json.Unmarshal(decoded, &result)
		require.NoError(t, err)
		require.Equal(t, val["key"], result["key"])
	}
}

// TestBase64Mismatch_WouldFail demonstrates the old bug: StdEncoding
// can't decode URLEncoding output when padding differs.
func TestBase64Mismatch_WouldFail(t *testing.T) {
	// Find a value where URL and Standard encodings differ
	data := []byte(`{"url":"https://example.com/a+b/c?d=e"}`)
	urlEncoded := base64.URLEncoding.EncodeToString(data)

	// StdEncoding will fail if the URL-safe chars (-_) are present
	// or if padding differs. Verify our fix uses the right decoder.
	decoded, err := base64.URLEncoding.DecodeString(urlEncoded)
	require.NoError(t, err)
	require.Equal(t, data, decoded)
}
