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

const watchMethod = "/grpc.health.v1.Health/Watch"

// healthStub is a minimal health service; Watch is a server-streaming RPC, which is what
// this test needs to prove the standard chain now gates streams.
type healthStub struct {
	grpc_health.UnimplementedHealthServer
}

func (healthStub) Watch(_ *grpc_health.HealthCheckRequest, ss grpc_health.Health_WatchServer) error {
	return ss.Send(&grpc_health.HealthCheckResponse{Status: grpc_health.HealthCheckResponse_SERVING})
}

// startServer builds a real gRPC server from FluffyCoreOTELStartup.ConfigureServerOpts --
// the standard fluffycore chain -- with the given entrypoint map, over an in-memory listener.
func startServer(t *testing.T, entryPoints map[string]contracts_common.IEntryPointConfig) grpc_health.HealthClient {
	t.Helper()
	builder := di.Builder()
	fluffycore_services_common.AddCommonServices(builder)
	root := builder.Build()

	startup := NewFluffyCoreOTELStartup(&FluffyCoreOTELStartupConfig{
		FuncAuthGetEntryPointConfigs: func() map[string]contracts_common.IEntryPointConfig { return entryPoints },
	})
	startup.SetConfig(&fluffycore_contracts_otel.OTELConfig{ServiceName: "stream-auth-test"})
	startup.SetRootContainer(root)

	ctx := zerolog.Nop().WithContext(context.Background())
	srv := grpc.NewServer(startup.ConfigureServerOpts(ctx)...)
	grpc_health.RegisterHealthServer(srv, healthStub{})

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

// watch opens the stream and reads the first message; a gate rejection surfaces here.
func watch(t *testing.T, client grpc_health.HealthClient) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := client.Watch(ctx, &grpc_health.HealthCheckRequest{})
	if err != nil {
		return err
	}
	_, err = stream.Recv()
	return err
}

// Before this fix, all three of these calls succeeded: the stream chain had no JWT
// interceptor and no claims-principal gate, so zero trust never applied to streams.

func TestStandardChain_Stream_UnlistedMethodDenied(t *testing.T) {
	client := startServer(t, map[string]contracts_common.IEntryPointConfig{})
	err := watch(t, client)
	require.Error(t, err)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestStandardChain_Stream_ProtectedMethodDeniedWithoutClaim(t *testing.T) {
	b := services_claimsprincipal.NewEntryPointClaimsBuilder()
	b.WithGrpcEntrypointClams(watchMethod,
		services_claimsprincipal.NewClaimFact(contracts_common.Claim{Type: "permissions", Value: "write"}))
	client := startServer(t, b.GetEntryPointClaimsMap())
	err := watch(t, client)
	require.Error(t, err)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestStandardChain_Stream_OpenMethodAllowedAnonymously(t *testing.T) {
	b := services_claimsprincipal.NewEntryPointClaimsBuilder()
	b.WithGrpcEntrypointPermissionsClaimsMapOpen(watchMethod)
	client := startServer(t, b.GetEntryPointClaimsMap())
	require.NoError(t, watch(t, client), "an open streaming method with no token must still work")
}
