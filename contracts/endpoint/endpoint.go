package endpoint

import (
	grpc_gateway_runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type (
	IEndpointRegistration interface {
		Register(s *grpc.Server)
		RegisterHandler(gwmux *grpc_gateway_runtime.ServeMux, conn *grpc.ClientConn)
	}
)
