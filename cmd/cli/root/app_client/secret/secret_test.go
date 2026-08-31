package secret

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClient_MissingCertFile_Errors(t *testing.T) {
	certFile, keyFile, caFile = "/does/not/exist-cert.pem", "/does/not/exist-key.pem", "/does/not/exist-ca.pem"
	_, _, err := newClient()
	require.Error(t, err)
	require.Contains(t, err.Error(), "load client cert/key")
}
