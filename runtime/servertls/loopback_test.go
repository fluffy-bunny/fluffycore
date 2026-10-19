package servertls_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
	fluffycore_grpcclient "github.com/fluffy-bunny/fluffycore/grpcclient"
	servertls "github.com/fluffy-bunny/fluffycore/runtime/servertls"
	certgen "github.com/fluffy-bunny/fluffycore/utils/certgen"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
	credentials "google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	grpc_health "google.golang.org/grpc/health/grpc_health_v1"
)

func TestLoopbackTrustFile_Disabled(t *testing.T) {
	_, ok := servertls.LoopbackTrustFile(&fluffycore_contracts_config.CoreConfig{})
	require.False(t, ok)
	_, ok = servertls.LoopbackTrustFile(nil)
	require.False(t, ok)
}

// The example's SayHello dialed its own server in plaintext, which fails once TLS is on.
// Dialing with LoopbackTrustFile (as it does now) must complete a real TLS handshake.
func TestLoopbackTrustFile_DialsTLSServer(t *testing.T) {
	dir := t.TempDir()
	ca, err := certgen.NewCA(certgen.CAOptions{CommonName: "test-ca"})
	require.NoError(t, err)
	server, err := certgen.NewLeafCert(ca, certgen.CertOptions{CommonName: "localhost", DNSNames: []string{"localhost"}, IsServer: true})
	require.NoError(t, err)
	for name, data := range map[string][]byte{"server-cert.pem": server.CertPEM, "server-key.pem": server.KeyPEM, "ca.pem": ca.CertPEM} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0o600))
	}
	cfg := &fluffycore_contracts_config.CoreConfig{TLSCertsDir: dir}

	tlsConfig, err := servertls.BuildServerTLSConfig(cfg)
	require.NoError(t, err)
	srv := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	grpc_health.RegisterHealthServer(srv, health.NewServer())
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	trustFile, ok := servertls.LoopbackTrustFile(cfg)
	require.True(t, ok)
	client, err := fluffycore_grpcclient.NewGrpcClient(
		fluffycore_grpcclient.WithHost("localhost"),
		fluffycore_grpcclient.WithPort(lis.Addr().(*net.TCPAddr).Port),
		fluffycore_grpcclient.WithInsecure(false),
		fluffycore_grpcclient.WithCertBundle(trustFile),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = grpc_health.NewHealthClient(client.GetConnection()).Check(ctx, &grpc_health.HealthCheckRequest{})
	require.NoError(t, err)
}
