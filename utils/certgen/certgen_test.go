package certgen

import (
	"crypto/x509"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewCA_SelfSigned(t *testing.T) {
	ca, err := NewCA(CAOptions{CommonName: "test-ca", Org: "fluffycore-test"})
	require.NoError(t, err)
	require.NotNil(t, ca.Cert)
	require.True(t, ca.Cert.IsCA)
	require.NotEmpty(t, ca.CertPEM)
	require.NotEmpty(t, ca.KeyPEM)

	// A self-signed CA verifies against its own cert.
	require.NoError(t, ca.Cert.CheckSignatureFrom(ca.Cert))
}

func TestNewLeafCert_ServerCert_VerifiesAgainstCA(t *testing.T) {
	ca, err := NewCA(CAOptions{CommonName: "test-ca"})
	require.NoError(t, err)

	serverURI, err := url.Parse("spiffe://cluster.local/ns/test/sa/server")
	require.NoError(t, err)

	leaf, err := NewLeafCert(ca, CertOptions{
		CommonName:  "server.local",
		DNSNames:    []string{"localhost", "server.local"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		URIs:        []*url.URL{serverURI},
		IsServer:    true,
	})
	require.NoError(t, err)
	require.NoError(t, leaf.Cert.CheckSignatureFrom(ca.Cert))
	require.Contains(t, leaf.Cert.DNSNames, "localhost")
	require.Len(t, leaf.Cert.URIs, 1)
	require.Equal(t, "spiffe://cluster.local/ns/test/sa/server", leaf.Cert.URIs[0].String())
}

func TestNewLeafCert_ClientCert_ExtKeyUsage(t *testing.T) {
	ca, err := NewCA(CAOptions{CommonName: "test-ca"})
	require.NoError(t, err)

	leaf, err := NewLeafCert(ca, CertOptions{CommonName: "test-client"})
	require.NoError(t, err)
	require.NoError(t, leaf.Cert.CheckSignatureFrom(ca.Cert))
	require.Contains(t, leaf.Cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth)
}

func TestNewLeafCert_HonorsValidFor(t *testing.T) {
	ca, err := NewCA(CAOptions{CommonName: "test-ca"})
	require.NoError(t, err)

	leaf, err := NewLeafCert(ca, CertOptions{CommonName: "short-lived", ValidFor: time.Hour})
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(time.Hour), leaf.Cert.NotAfter, time.Minute)
}

func TestNewLeafCert_DifferentCAsDoNotCrossVerify(t *testing.T) {
	ca1, err := NewCA(CAOptions{CommonName: "ca-1"})
	require.NoError(t, err)
	ca2, err := NewCA(CAOptions{CommonName: "ca-2"})
	require.NoError(t, err)

	leaf, err := NewLeafCert(ca1, CertOptions{CommonName: "client"})
	require.NoError(t, err)

	require.Error(t, leaf.Cert.CheckSignatureFrom(ca2.Cert))
}
