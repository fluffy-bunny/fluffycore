package grpcclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	certgen "github.com/fluffy-bunny/fluffycore/utils/certgen"
	"github.com/stretchr/testify/require"
	otel "go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	grpc "google.golang.org/grpc"
	credentials "google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	grpc_health "google.golang.org/grpc/health/grpc_health_v1"
)

type pki struct {
	caFile     string
	clientCert tls.Certificate
	serverTLS  *tls.Config
}

// newPKI returns a CA, a server config for "localhost" and a client certificate.
func newPKI(t *testing.T, requireClientCert bool) pki {
	t.Helper()
	ca, err := certgen.NewCA(certgen.CAOptions{CommonName: "test-ca"})
	require.NoError(t, err)
	server, err := certgen.NewLeafCert(ca, certgen.CertOptions{CommonName: "localhost", DNSNames: []string{"localhost"}, IsServer: true})
	require.NoError(t, err)
	client, err := certgen.NewLeafCert(ca, certgen.CertOptions{CommonName: "client"})
	require.NoError(t, err)

	caFile := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(caFile, ca.CertPEM, 0o600))
	serverCert, err := tls.X509KeyPair(server.CertPEM, server.KeyPEM)
	require.NoError(t, err)
	clientCert, err := tls.X509KeyPair(client.CertPEM, client.KeyPEM)
	require.NoError(t, err)
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ca.CertPEM)
	serverTLS := &tls.Config{Certificates: []tls.Certificate{serverCert}, ClientCAs: pool}
	if requireClientCert {
		serverTLS.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return pki{caFile: caFile, clientCert: clientCert, serverTLS: serverTLS}
}

func serveTLS(t *testing.T, cfg *tls.Config) int {
	t.Helper()
	srv := grpc.NewServer(grpc.Creds(credentials.NewTLS(cfg)))
	grpc_health.RegisterHealthServer(srv, health.NewServer())
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis.Addr().(*net.TCPAddr).Port
}

func checkHealth(t *testing.T, opts ...GRPCClientOption) error {
	t.Helper()
	c, err := NewGRPCClient(opts...)
	require.NoError(t, err)
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = grpc_health.NewHealthClient(c.GetConnection()).Check(ctx, &grpc_health.HealthCheckRequest{})
	return err
}

// WithCertBundle alone used to be ignored (insecure is the default): it dialed plaintext.
func TestWithCertBundle_ImpliesTLS(t *testing.T) {
	p := newPKI(t, false)
	port := serveTLS(t, p.serverTLS)
	require.NoError(t, checkHealth(t, WithHost("localhost"), WithPort(port), WithCertBundle(p.caFile)))
}

// With a bundle, WithClientCert used to be dropped, so mTLS could never work.
func TestWithCertBundleAndClientCert_MutualTLS(t *testing.T) {
	p := newPKI(t, true)
	port := serveTLS(t, p.serverTLS)
	require.NoError(t, checkHealth(t, WithHost("localhost"), WithPort(port), WithCertBundle(p.caFile), WithClientCert(p.clientCert)))
	require.Error(t, checkHealth(t, WithHost("localhost"), WithPort(port), WithCertBundle(p.caFile)), "no client cert: the server must refuse")
}

func TestExplicitInsecureWithTLSOptions_IsAnError(t *testing.T) {
	p := newPKI(t, false)
	_, err := NewGRPCClient(WithHost("localhost"), WithPort(1), WithInsecure(true), WithCertBundle(p.caFile))
	require.Error(t, err)
}

// TLS without a bundle used to trust an empty pool; it must use the system roots.
func TestTLSConfig_NoBundleUsesSystemRoots(t *testing.T) {
	c := &GRPCClient{}
	cfg, err := c.tlsConfig("example.com")
	require.NoError(t, err)
	require.Nil(t, cfg.RootCAs, "nil RootCAs means the system roots")
}

// Enabling DataDog used to silently disable OTEL tracing.
func TestTracing_OTELStillOnWithDataDog(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	p := newPKI(t, false)
	port := serveTLS(t, p.serverTLS)
	require.NoError(t, checkHealth(t, WithHost("localhost"), WithPort(port), WithCertBundle(p.caFile),
		WithOTELTracer(true), WithDataDogTracer(true)))
	require.NotEmpty(t, recorder.Ended(), "the OTEL client span must still be recorded")
}
