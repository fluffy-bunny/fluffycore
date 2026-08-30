package servertls

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
	"time"

	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
	"github.com/fluffy-bunny/fluffycore/utils/certgen"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path
}

func genServerCertFiles(t *testing.T, dir string) (certFile, keyFile, caFile string, ca *certgen.CA) {
	t.Helper()
	ca, err := certgen.NewCA(certgen.CAOptions{CommonName: "test-ca"})
	require.NoError(t, err)
	server, err := certgen.NewLeafCert(ca, certgen.CertOptions{CommonName: "localhost", DNSNames: []string{"localhost"}, IsServer: true})
	require.NoError(t, err)

	certFile = writeFile(t, dir, "server-cert.pem", server.CertPEM)
	keyFile = writeFile(t, dir, "server-key.pem", server.KeyPEM)
	caFile = writeFile(t, dir, "ca.pem", ca.CertPEM)
	return
}

func TestBuildServerTLSConfig_Disabled_ReturnsNil(t *testing.T) {
	cfg := &fluffycore_contracts_config.CoreConfig{TLSEnabled: false}
	tlsConfig, err := BuildServerTLSConfig(cfg)
	require.NoError(t, err)
	require.Nil(t, tlsConfig)
}

func TestBuildServerTLSConfig_EnabledWithoutCertFiles_Errors(t *testing.T) {
	cfg := &fluffycore_contracts_config.CoreConfig{TLSEnabled: true}
	_, err := BuildServerTLSConfig(cfg)
	require.Error(t, err)
}

func TestBuildServerTLSConfig_PlainTLS_NoClientCA(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _, _ := genServerCertFiles(t, dir)

	cfg := &fluffycore_contracts_config.CoreConfig{
		TLSEnabled:  true,
		TLSCertFile: certFile,
		TLSKeyFile:  keyFile,
	}
	tlsConfig, err := BuildServerTLSConfig(cfg)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig)

	cert, err := tlsConfig.GetCertificate(nil)
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.Nil(t, tlsConfig.GetConfigForClient, "no client CA configured -- must not request/verify a client cert")
}

func TestBuildServerTLSConfig_MutualTLS_DefaultsToVerifyIfGiven(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, caFile, _ := genServerCertFiles(t, dir)

	cfg := &fluffycore_contracts_config.CoreConfig{
		TLSEnabled:      true,
		TLSCertFile:     certFile,
		TLSKeyFile:      keyFile,
		TLSClientCAFile: caFile,
	}
	tlsConfig, err := BuildServerTLSConfig(cfg)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig.GetConfigForClient)

	clientConfig, err := tlsConfig.GetConfigForClient(nil)
	require.NoError(t, err)
	require.NotNil(t, clientConfig.ClientCAs)
	require.Nil(t, clientConfig.GetConfigForClient, "must not recurse")
}

func TestResolveConfig_CertsDirImpliesEnabled_AndDerivesPaths(t *testing.T) {
	cfg := &fluffycore_contracts_config.CoreConfig{TLSCertsDir: "/certs"}
	resolved := resolveConfig(cfg)
	require.True(t, resolved.enabled, "setting TLSCertsDir alone must imply TLS is enabled")
	require.Equal(t, filepath.Join("/certs", "server-cert.pem"), resolved.certFile)
	require.Equal(t, filepath.Join("/certs", "server-key.pem"), resolved.keyFile)
	require.Equal(t, filepath.Join("/certs", "ca.pem"), resolved.clientCAFile)
}

func TestResolveConfig_ExplicitFieldsOverrideCertsDir(t *testing.T) {
	cfg := &fluffycore_contracts_config.CoreConfig{
		TLSCertsDir:     "/certs",
		TLSCertFile:     "/explicit/cert.pem",
		TLSKeyFile:      "/explicit/key.pem",
		TLSClientCAFile: "/explicit/ca.pem",
	}
	resolved := resolveConfig(cfg)
	require.Equal(t, "/explicit/cert.pem", resolved.certFile)
	require.Equal(t, "/explicit/key.pem", resolved.keyFile)
	require.Equal(t, "/explicit/ca.pem", resolved.clientCAFile)
}

func TestResolveConfig_NeitherSet_NotEnabled(t *testing.T) {
	resolved := resolveConfig(&fluffycore_contracts_config.CoreConfig{})
	require.False(t, resolved.enabled)
}

func TestBuildServerTLSConfig_CertsDir_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	genServerCertFiles(t, dir) // writes server-cert.pem/server-key.pem/ca.pem -- exactly what TLSCertsDir expects

	cfg := &fluffycore_contracts_config.CoreConfig{TLSCertsDir: dir}
	tlsConfig, err := BuildServerTLSConfig(cfg)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig, "TLSCertsDir alone must be enough to enable TLS")

	cert, err := tlsConfig.GetCertificate(nil)
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.NotNil(t, tlsConfig.GetConfigForClient, "ca.pem is present under TLSCertsDir, so mutual TLS must be implied too")
}

func TestIsMutualTLSEnabled(t *testing.T) {
	require.False(t, IsMutualTLSEnabled(nil))
	require.False(t, IsMutualTLSEnabled(&fluffycore_contracts_config.CoreConfig{}))
	require.False(t, IsMutualTLSEnabled(&fluffycore_contracts_config.CoreConfig{TLSEnabled: true, TLSCertFile: "c", TLSKeyFile: "k"}), "server-only TLS is not mutual")
	require.True(t, IsMutualTLSEnabled(&fluffycore_contracts_config.CoreConfig{TLSEnabled: true, TLSClientCAFile: "ca.pem"}))
	require.True(t, IsMutualTLSEnabled(&fluffycore_contracts_config.CoreConfig{TLSCertsDir: "/certs"}),
		"mTLS derived from TLSCertsDir must be reported too, not just the raw TLSClientCAFile field")
}

func TestParseClientAuth(t *testing.T) {
	cases := []struct {
		mode         string
		clientCAFile string
		wantErr      bool
	}{
		{"", "", false},
		{"", "ca.pem", false},
		{ClientAuthNone, "", false},
		{ClientAuthRequest, "", false},
		{ClientAuthVerifyIfGiven, "", false},
		{ClientAuthRequire, "", false},
		{"bogus", "", true},
	}
	for _, tc := range cases {
		_, err := parseClientAuth(tc.mode, tc.clientCAFile)
		if tc.wantErr {
			require.Error(t, err, tc.mode)
		} else {
			require.NoError(t, err, tc.mode)
		}
	}
}

func TestReloadingCertificate_PicksUpRotatedCertOnDisk(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, _, ca := genServerCertFiles(t, dir)

	// Tiny check interval so the test doesn't have to sleep for the production default.
	reloading, err := newReloadingCertificate(certFile, keyFile, "", time.Millisecond)
	require.NoError(t, err)

	first, err := reloading.GetCertificate(nil)
	require.NoError(t, err)
	firstLeaf, err := x509.ParseCertificate(first.Certificate[0])
	require.NoError(t, err)
	require.Equal(t, "localhost", firstLeaf.Subject.CommonName)

	// Rotate: a brand new leaf cert (different key/serial) written to the same paths.
	rotated, err := certgen.NewLeafCert(ca, certgen.CertOptions{CommonName: "rotated.local", DNSNames: []string{"rotated.local"}, IsServer: true})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(certFile, rotated.CertPEM, 0o600))
	require.NoError(t, os.WriteFile(keyFile, rotated.KeyPEM, 0o600))

	// Ensure the mtime actually advances on filesystems with coarse mtime resolution.
	future := time.Now().Add(time.Second)
	require.NoError(t, os.Chtimes(certFile, future, future))
	require.NoError(t, os.Chtimes(keyFile, future, future))

	time.Sleep(5 * time.Millisecond) // clear the check-interval throttle

	second, err := reloading.GetCertificate(nil)
	require.NoError(t, err)
	secondLeaf, err := x509.ParseCertificate(second.Certificate[0])
	require.NoError(t, err)
	require.Equal(t, "rotated.local", secondLeaf.Subject.CommonName)
}

func TestReloadingCertificate_ClientCAPoolReloads(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, caFile, _ := genServerCertFiles(t, dir)

	reloading, err := newReloadingCertificate(certFile, keyFile, caFile, time.Millisecond)
	require.NoError(t, err)

	pool := reloading.clientCAPool()
	require.NotNil(t, pool)

	// A second, unrelated CA appended to the same file -- the pool must reflect it.
	ca2, err := certgen.NewCA(certgen.CAOptions{CommonName: "second-ca"})
	require.NoError(t, err)
	existing, err := os.ReadFile(caFile)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(caFile, append(existing, ca2.CertPEM...), 0o600))
	future := time.Now().Add(time.Second)
	require.NoError(t, os.Chtimes(caFile, future, future))

	time.Sleep(5 * time.Millisecond)

	pool2 := reloading.clientCAPool()
	require.NotNil(t, pool2)
	require.NotSame(t, pool, pool2, "the pool must be rebuilt (not mutated in place) on reload")

	// Both CAs' server-signed leaves must now verify against the reloaded pool.
	opts := x509.VerifyOptions{Roots: pool2, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}}
	_, err = ca2.Cert.Verify(opts)
	require.NoError(t, err, "the newly appended CA must be trusted after reload")
}
