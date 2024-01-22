package endpoint

import (
	grpc_gateway_runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type (
	IEndpointRegistration interface {
		RegisterFluffyCoreGRPCService(s *grpc.Server)
		RegisterFluffyCoreHandler(gwmux *grpc_gateway_runtime.ServeMux, conn *grpc.ClientConn)
	}
)
