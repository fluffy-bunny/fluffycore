package certgen

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewBundle_ServerVerifiesAgainstCA_ClientVerifiesAgainstCA(t *testing.T) {
	bundle, err := NewBundle(BundleOptions{
		ServerCommonName: "localhost",
		ServerHosts:      []string{"localhost", "127.0.0.1"},
		ClientCommonName: "test-client",
		ClientURIs:       []string{"spiffe://cluster.local/ns/foo/sa/bar"},
		Org:              "fluffycore-test",
	})
	require.NoError(t, err)

	require.NoError(t, bundle.Server.Cert.CheckSignatureFrom(bundle.CA.Cert))
	require.NoError(t, bundle.Client.Cert.CheckSignatureFrom(bundle.CA.Cert))
	require.Contains(t, bundle.Server.Cert.DNSNames, "localhost")
	require.Len(t, bundle.Client.Cert.URIs, 1)
	require.Equal(t, "spiffe://cluster.local/ns/foo/sa/bar", bundle.Client.Cert.URIs[0].String())
}

func TestNewBundle_InvalidClientURI_Errors(t *testing.T) {
	_, err := NewBundle(BundleOptions{
		ServerCommonName: "localhost",
		ClientCommonName: "test-client",
		ClientURIs:       []string{"://not-a-valid-uri"},
	})
	require.Error(t, err)
}

func TestBundle_WriteFiles_WritesAllExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	bundle, err := NewBundle(BundleOptions{ServerCommonName: "localhost", ClientCommonName: "test-client"})
	require.NoError(t, err)
	require.NoError(t, bundle.WriteFiles(dir, false))

	for _, f := range bundleFileNames {
		info, err := os.Stat(filepath.Join(dir, f))
		require.NoError(t, err, f)
		require.Greater(t, info.Size(), int64(0), f)
	}
}

func TestBundle_WriteFiles_RefusesToOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	bundle, err := NewBundle(BundleOptions{ServerCommonName: "localhost", ClientCommonName: "test-client"})
	require.NoError(t, err)
	require.NoError(t, bundle.WriteFiles(dir, false))
	require.Error(t, bundle.WriteFiles(dir, false))
	require.NoError(t, bundle.WriteFiles(dir, true))
}

// TestNewBundle_Forever_CAOutlivesLeaves regression-tests a real bug: 5x a
// ~100-year (Forever) leaf validity overflows time.Duration's int64
// nanosecond range (anything past ~58 years, when multiplied by 5, wraps).
// NewCA's own "<=0 means default" fallback silently swallowed the overflow
// and produced a CA that expired in 5 years -- years before the leaves it
// signed, which would have made every leaf unverifiable once the CA expired.
func TestNewBundle_Forever_CAOutlivesLeaves(t *testing.T) {
	bundle, err := NewBundle(BundleOptions{
		ServerCommonName: "localhost",
		ClientCommonName: "test-client",
		ValidFor:         Forever,
	})
	require.NoError(t, err)

	require.True(t, bundle.CA.Cert.NotAfter.After(bundle.Server.Cert.NotAfter) || bundle.CA.Cert.NotAfter.Equal(bundle.Server.Cert.NotAfter),
		"the CA must not expire before the leaves it signs: CA NotAfter=%s, server NotAfter=%s", bundle.CA.Cert.NotAfter, bundle.Server.Cert.NotAfter)
	require.Greater(t, bundle.CA.Cert.NotAfter.Sub(time.Now()), 50*365*24*time.Hour,
		"the CA must not have silently fallen back to its short (5 year) default")
}

func TestCAValidForFrom_NoOverflowDoesMultiply(t *testing.T) {
	require.Equal(t, 5*24*time.Hour, caValidForFrom(24*time.Hour))
}

func TestCAValidForFrom_OverflowFallsBackToLeafDuration(t *testing.T) {
	require.Equal(t, Forever, caValidForFrom(Forever))
}
