package servertls_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
	servertls "github.com/fluffy-bunny/fluffycore/runtime/servertls"
	certgen "github.com/fluffy-bunny/fluffycore/utils/certgen"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
	credentials "google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	grpc_health "google.golang.org/grpc/health/grpc_health_v1"
)

func TestLoopbackTarget(t *testing.T) {
	for addr, want := range map[net.Addr]string{
		&net.TCPAddr{IP: net.IPv4zero, Port: 50051}:         "localhost:50051", // used to dial 0.0.0.0
		&net.TCPAddr{IP: net.IPv6unspecified, Port: 50051}:  "localhost:50051",
		&net.TCPAddr{Port: 7}:                               "localhost:7",
		&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9}: "127.0.0.1:9",
		&net.TCPAddr{IP: net.ParseIP("::1"), Port: 9}:       "[::1]:9",
		&net.UnixAddr{Name: "/tmp/grpc.sock", Net: "unix"}:  "unix:/tmp/grpc.sock",
	} {
		require.Equal(t, want, servertls.LoopbackTarget(addr, 1234), "%v", addr)
	}
	require.Equal(t, "localhost:1234", servertls.LoopbackTarget(fakeAddr{}, 1234), "unknown listener types fall back")
}

type fakeAddr struct{}

func (fakeAddr) Network() string { return "bufconn" }
func (fakeAddr) String() string  { return "bufconn" }

// writeServerCert writes a CA-signed server cert for names under dir (the tlsCertsDir layout).
func writeServerCert(t *testing.T, dir string, names ...string) {
	t.Helper()
	ca, err := certgen.NewCA(certgen.CAOptions{CommonName: "ca-" + names[0]})
	require.NoError(t, err)
	server, err := certgen.NewLeafCert(ca, certgen.CertOptions{CommonName: names[0], DNSNames: names, IsServer: true})
	require.NoError(t, err)
	for name, data := range map[string][]byte{"server-cert.pem": server.CertPEM, "server-key.pem": server.KeyPEM, "ca.pem": ca.CertPEM} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0o600))
	}
}

func startTLSServer(t *testing.T, cfg *fluffycore_contracts_config.CoreConfig) net.Addr {
	t.Helper()
	tlsConfig, err := servertls.BuildServerTLSConfig(cfg)
	require.NoError(t, err)
	srv := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	grpc_health.RegisterHealthServer(srv, health.NewServer())
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis.Addr()
}

func check(t *testing.T, target string, creds credentials.TransportCredentials) error {
	t.Helper()
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(creds))
	require.NoError(t, err)
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = grpc_health.NewHealthClient(conn).Check(ctx, &grpc_health.HealthCheckRequest{}, grpc.WaitForReady(false))
	return err
}

func TestLoopbackClientCredentials_PlaintextWhenServerNotTLS(t *testing.T) {
	creds, err := servertls.LoopbackClientCredentials(&fluffycore_contracts_config.CoreConfig{TLSCertsDir: t.TempDir()}, false)
	require.NoError(t, err)
	require.Equal(t, "insecure", creds.Info().SecurityProtocol)
}

// The gateway/healthz/NATS bridge dial used to be plaintext, so it failed against a TLS
// server. It must work -- and keep working when the certificate is rotated on disk.
func TestLoopbackClientCredentials_TLSAndRotation(t *testing.T) {
	dir := t.TempDir()
	cfg := &fluffycore_contracts_config.CoreConfig{TLSCertsDir: dir}
	writeServerCert(t, dir, "localhost")
	addr := startTLSServer(t, cfg)

	creds, err := servertls.LoopbackClientCredentials(cfg, true)
	require.NoError(t, err)
	require.NoError(t, check(t, servertls.LoopbackTarget(addr, 0), creds))

	// Rotate to a certificate from a different CA; a server now serving it is still trusted
	// by the SAME credentials, because the trust file is re-read at each handshake.
	writeServerCert(t, dir, "localhost")
	rotated := startTLSServer(t, cfg)
	require.NoError(t, check(t, servertls.LoopbackTarget(rotated, 0), creds))

	// A server whose certificate isn't the one on disk is not trusted.
	other := t.TempDir()
	writeServerCert(t, other, "localhost")
	stranger := startTLSServer(t, &fluffycore_contracts_config.CoreConfig{TLSCertsDir: other})
	require.Error(t, check(t, servertls.LoopbackTarget(stranger, 0), creds))
}

// The certificate must cover "localhost".
func TestLoopbackClientCredentials_RequiresLocalhostName(t *testing.T) {
	dir := t.TempDir()
	cfg := &fluffycore_contracts_config.CoreConfig{TLSCertsDir: dir}
	writeServerCert(t, dir, "api.example.com")
	addr := startTLSServer(t, cfg)
	creds, err := servertls.LoopbackClientCredentials(cfg, true)
	require.NoError(t, err)
	require.Error(t, check(t, servertls.LoopbackTarget(addr, 0), creds))
}

func TestRequiresClientCert(t *testing.T) {
	dir := t.TempDir()
	require.False(t, servertls.RequiresClientCert(&fluffycore_contracts_config.CoreConfig{TLSCertsDir: dir}))
	require.True(t, servertls.RequiresClientCert(&fluffycore_contracts_config.CoreConfig{TLSCertsDir: dir, TLSClientAuth: servertls.ClientAuthRequire}))
	require.False(t, servertls.RequiresClientCert(nil))
}
