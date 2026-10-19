package pkcs8enc

import (
	"encoding/asn1"
	"testing"

	"github.com/stretchr/testify/require"
)

func asn1Unmarshal(b []byte, v any) ([]byte, error) { return asn1.Unmarshal(b, v) }

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := asn1.Marshal(v)
	require.NoError(t, err)
	return b
}
