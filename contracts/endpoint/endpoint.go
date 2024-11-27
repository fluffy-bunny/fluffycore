package endpoint

import (
	"context"

	nats_micro_service "github.com/fluffy-bunny/fluffycore/contracts/nats_micro_service"
	grpc_gateway_runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	nats "github.com/nats-io/nats.go"
	micro "github.com/nats-io/nats.go/micro"
	grpc "google.golang.org/grpc"
)

type (
	IEndpointRegistration interface {
		RegisterFluffyCoreGRPCService(s *grpc.Server)
		RegisterFluffyCoreHandler(gwmux *grpc_gateway_runtime.ServeMux, conn *grpc.ClientConn)
	}
	INATSEndpointRegistration interface {
		RegisterFluffyCoreNATSHandler(ctx context.Context, natsCon *nats.Conn, conn *grpc.ClientConn, option *nats_micro_service.NATSMicroServiceRegisrationOption) (micro.Service, error)
	}
)
