package health

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

func serve(t *testing.T, svc *Service) (grpc_health.HealthClient, *grpc.Server) {
	t.Helper()
	srv := grpc.NewServer()
	grpc_health.RegisterHealthServer(srv, svc)
	lis := bufconn.Listen(1 << 20)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return grpc_health.NewHealthClient(conn), srv
}

func ctx(t *testing.T) context.Context {
	c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return c
}

// Check used to answer SERVING unconditionally, even while shutting down.
func TestCheck_ReportsNotServingAfterShutdown(t *testing.T) {
	svc := New()
	client, _ := serve(t, svc)
	resp, err := client.Check(ctx(t), &grpc_health.HealthCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, grpc_health.HealthCheckResponse_SERVING, resp.Status)

	svc.Shutdown()
	resp, err = client.Check(ctx(t), &grpc_health.HealthCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, grpc_health.HealthCheckResponse_NOT_SERVING, resp.Status)
}

// Watch used to return immediately (end of stream) instead of streaming status changes.
func TestWatch_StreamsStatusChanges(t *testing.T) {
	svc := New()
	client, _ := serve(t, svc)
	stream, err := client.Watch(ctx(t), &grpc_health.HealthCheckRequest{})
	require.NoError(t, err)
	msg, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, grpc_health.HealthCheckResponse_SERVING, msg.Status)

	svc.SetServingStatus("", grpc_health.HealthCheckResponse_NOT_SERVING)
	msg, err = stream.Recv()
	require.NoError(t, err)
	require.Equal(t, grpc_health.HealthCheckResponse_NOT_SERVING, msg.Status)
}

// An open Watch must not hold GracefulStop forever: Shutdown delivers NOT_SERVING, then
// ends the stream cleanly.
func TestShutdown_EndsWatchStreamsSoGracefulStopCompletes(t *testing.T) {
	svc := New()
	client, srv := serve(t, svc)
	stream, err := client.Watch(ctx(t), &grpc_health.HealthCheckRequest{})
	require.NoError(t, err)
	_, err = stream.Recv() // SERVING
	require.NoError(t, err)

	svc.Shutdown()
	msg, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, grpc_health.HealthCheckResponse_NOT_SERVING, msg.Status)
	_, err = stream.Recv()
	require.ErrorIs(t, err, io.EOF, "the stream should end cleanly after shutdown")

	stopped := make(chan struct{})
	go func() { srv.GracefulStop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("GracefulStop blocked by a Watch stream")
	}
	svc.Shutdown() // idempotent
}
