package gencert

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"testing"

	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
	fluffycore_servertls "github.com/fluffy-bunny/fluffycore/runtime/servertls"
	"github.com/stretchr/testify/require"
)

// resetFlags restores the package-level flag vars gencert's run() reads, so
// tests don't leak state into one another (cobra normally owns this via flag
// parsing, but these tests call run() directly).
func resetFlags(t *testing.T, dir string) {
	t.Helper()
	outDir = dir
	hosts = []string{"localhost", "127.0.0.1"}
	serverCN = "localhost"
	clientCN = "fluffycore-test-client"
	clientURIs = nil
	org = "fluffycore-test"
	validDays = 1
	force = false
}

func TestRun_WritesAllExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	resetFlags(t, dir)

	require.NoError(t, run())

	for _, f := range []string{"ca.pem", "ca-key.pem", "server-cert.pem", "server-key.pem", "client-cert.pem", "client-key.pem"} {
		info, err := os.Stat(filepath.Join(dir, f))
		require.NoError(t, err, f)
		require.Greater(t, info.Size(), int64(0), f)
	}
}

func TestRun_RefusesToOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	resetFlags(t, dir)
	require.NoError(t, run())

	resetFlags(t, dir) // force stays false
	require.Error(t, run())
}

func TestRun_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	resetFlags(t, dir)
	require.NoError(t, run())

	resetFlags(t, dir)
	force = true
	require.NoError(t, run())
}

// TestRun_GeneratedBundle_EndToEndMutualTLSHandshake is the "client to server
// test" this tool exists for: it runs gencert's own run(), builds the exact
// *tls.Config the fluffycore gRPC server would build from the resulting files
// via runtime/servertls, and performs a real TCP+TLS handshake against it using
// the generated client certificate -- proving the whole bundle actually works
// for mutual TLS, not just that the files parse.
func TestRun_GeneratedBundle_EndToEndMutualTLSHandshake(t *testing.T) {
	dir := t.TempDir()
	resetFlags(t, dir)
	require.NoError(t, run())

	serverTLSConfig, err := fluffycore_servertls.BuildServerTLSConfig(&fluffycore_contracts_config.CoreConfig{
		TLSEnabled:      true,
		TLSCertFile:     filepath.Join(dir, "server-cert.pem"),
		TLSKeyFile:      filepath.Join(dir, "server-key.pem"),
		TLSClientCAFile: filepath.Join(dir, "ca.pem"),
		TLSClientAuth:   fluffycore_servertls.ClientAuthRequire, // force verification for this test, not just "if given"
	})
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()
		tlsConn := tls.Server(conn, serverTLSConfig)
		serverDone <- tlsConn.Handshake()
	}()

	clientCert, err := tls.LoadX509KeyPair(filepath.Join(dir, "client-cert.pem"), filepath.Join(dir, "client-key.pem"))
	require.NoError(t, err)
	caPEM, err := os.ReadFile(filepath.Join(dir, "ca.pem"))
	require.NoError(t, err)
	rootPool := x509.NewCertPool()
	require.True(t, rootPool.AppendCertsFromPEM(caPEM))

	clientConn, err := tls.Dial("tcp", listener.Addr().String(), &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      rootPool,
		ServerName:   "localhost",
	})
	require.NoError(t, err, "client must be able to complete the TLS handshake using the generated bundle")
	defer clientConn.Close()

	require.NoError(t, <-serverDone, "server must accept and verify the client certificate")

	state := clientConn.ConnectionState()
	require.NotEmpty(t, state.PeerCertificates)
	require.Equal(t, serverCN, state.PeerCertificates[0].Subject.CommonName)
}

// TestRun_GeneratedBundle_RejectsUnknownClientCert proves the flip side: a
// client presenting a certificate NOT signed by the generated CA must be
// rejected by the server, confirming verification is actually happening and
// isn't a rubber stamp.
func TestRun_GeneratedBundle_RejectsUnknownClientCert(t *testing.T) {
	dir := t.TempDir()
	resetFlags(t, dir)
	require.NoError(t, run())

	// A second, unrelated bundle -- its client cert is signed by a different CA.
	otherDir := t.TempDir()
	resetFlags(t, otherDir)
	require.NoError(t, run())

	serverTLSConfig, err := fluffycore_servertls.BuildServerTLSConfig(&fluffycore_contracts_config.CoreConfig{
		TLSEnabled:      true,
		TLSCertFile:     filepath.Join(dir, "server-cert.pem"),
		TLSKeyFile:      filepath.Join(dir, "server-key.pem"),
		TLSClientCAFile: filepath.Join(dir, "ca.pem"),
		TLSClientAuth:   fluffycore_servertls.ClientAuthRequire,
	})
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()
		tlsConn := tls.Server(conn, serverTLSConfig)
		serverDone <- tlsConn.Handshake()
	}()

	unknownClientCert, err := tls.LoadX509KeyPair(filepath.Join(otherDir, "client-cert.pem"), filepath.Join(otherDir, "client-key.pem"))
	require.NoError(t, err)
	caPEM, err := os.ReadFile(filepath.Join(dir, "ca.pem"))
	require.NoError(t, err)
	rootPool := x509.NewCertPool()
	require.True(t, rootPool.AppendCertsFromPEM(caPEM))

	// Force TLS 1.2: under TLS 1.3, Go's client-side Handshake()/Dial() can
	// return successfully before a server-side client-cert rejection alert
	// (sent as part of the server's post-handshake processing) has been
	// observed by the client. TLS 1.2's handshake is fully synchronous, so the
	// server's rejection is guaranteed to surface on the client's blocking
	// Dial() call -- exactly what this test needs to assert deterministically.
	_, dialErr := tls.Dial("tcp", listener.Addr().String(), &tls.Config{
		Certificates: []tls.Certificate{unknownClientCert},
		RootCAs:      rootPool,
		ServerName:   "localhost",
		MaxVersion:   tls.VersionTLS12,
	})
	require.Error(t, dialErr, "a client certificate from an unrelated CA must be rejected")
	require.Error(t, <-serverDone)
}
