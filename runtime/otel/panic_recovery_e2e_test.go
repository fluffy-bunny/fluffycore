package otel

import (
	"context"
	"net"
	"testing"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	fluffycore_contracts_otel "github.com/fluffy-bunny/fluffycore/contracts/otel"
	fluffycore_services_common "github.com/fluffy-bunny/fluffycore/services/common"
	services_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	status "github.com/gogo/status"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

type panickingHealth struct {
	grpc_health.UnimplementedHealthServer
}

func (panickingHealth) Check(context.Context, *grpc_health.HealthCheckRequest) (*grpc_health.HealthCheckResponse, error) {
	panic("handler boom")
}

func (panickingHealth) Watch(*grpc_health.HealthCheckRequest, grpc_health.Health_WatchServer) error {
	panic("stream handler boom")
}

func startServerWith(t *testing.T, root di.Container, svc grpc_health.HealthServer) grpc_health.HealthClient {
	t.Helper()
	b := services_claimsprincipal.NewEntryPointClaimsBuilder()
	b.WithGrpcEntrypointPermissionsClaimsMapOpen("/grpc.health.v1.Health/Check")
	b.WithGrpcEntrypointPermissionsClaimsMapOpen("/grpc.health.v1.Health/Watch")
	startup := NewFluffyCoreOTELStartup(&FluffyCoreOTELStartupConfig{
		FuncAuthGetEntryPointConfigs: func() map[string]contracts_common.IEntryPointConfig { return b.GetEntryPointClaimsMap() },
	})
	startup.SetConfig(&fluffycore_contracts_otel.OTELConfig{ServiceName: "panic-test"})
	startup.SetRootContainer(root)

	srv := grpc.NewServer(startup.ConfigureServerOpts(zerolog.Nop().WithContext(context.Background()))...)
	grpc_health.RegisterHealthServer(srv, svc)
	lis := bufconn.Listen(1 << 20)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return grpc_health.NewHealthClient(conn)
}

func callCtx(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// A panic INSIDE AN INTERCEPTOR -- here the logging interceptor's di.Get on a service that
// was never registered -- used to be outside the recovery interceptor (registered last) and
// crashed the whole process. If this test regresses, the test binary itself dies.
func TestStandardChain_PanicInInterceptorIsRecovered(t *testing.T) {
	b := di.Builder()
	services_claimsprincipal.AddClaimsPrincipal(b) // deliberately NOT AddCommonServices: no property bag
	client := startServerWith(t, b.Build(), healthStub{})

	_, err := client.Check(callCtx(t), &grpc_health.HealthCheckRequest{})
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
	require.NotContains(t, err.Error(), "property", "panic details must not leak to the client")

	// The server is still alive and serving.
	_, err = client.Check(callCtx(t), &grpc_health.HealthCheckRequest{})
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestStandardChain_PanicInHandlerIsRecovered(t *testing.T) {
	b := di.Builder()
	fluffycore_services_common.AddCommonServices(b)
	client := startServerWith(t, b.Build(), panickingHealth{})

	_, err := client.Check(callCtx(t), &grpc_health.HealthCheckRequest{})
	require.Equal(t, codes.Internal, status.Code(err))
	require.NotContains(t, err.Error(), "boom", "panic value must not leak to the client")

	stream, err := client.Watch(callCtx(t), &grpc_health.HealthCheckRequest{})
	require.NoError(t, err)
	_, err = stream.Recv()
	require.Equal(t, codes.Internal, status.Code(err), "stream handler panics are recovered too")
}
