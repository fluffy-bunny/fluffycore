package greeter

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	endpoint "github.com/fluffy-bunny/fluffycore/contracts/endpoint"
	contracts_config "github.com/fluffy-bunny/fluffycore/example/internal/contracts/config"
	fluffycore_contracts_somedisposable "github.com/fluffy-bunny/fluffycore/example/internal/contracts/somedisposable"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	grpc_gateway_runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	zerolog "github.com/rs/zerolog"
	"google.golang.org/grpc"
)

type (
	service struct {
		proto_helloworld.GreeterFluffyCoreServer

		config               *contracts_config.Config
		scopedSomeDisposable fluffycore_contracts_somedisposable.IScopedSomeDisposable
	}
	registrationServer struct {
		proto_helloworld.GreeterFluffyCoreServer
	}
)

var stemService = (*service)(nil)

func init() {
	var _ proto_helloworld.IFluffyCoreGreeterServer = stemService
	var _ endpoint.IEndpointRegistration = (*registrationServer)(nil)
}

func (s *registrationServer) RegisterHandler(gwmux *grpc_gateway_runtime.ServeMux, conn *grpc.ClientConn) {
	proto_helloworld.RegisterGreeterHandler(context.Background(), gwmux, conn)
}
func (s *service) Ctor(
	config *contracts_config.Config,
	scopedSomeDisposable fluffycore_contracts_somedisposable.IScopedSomeDisposable) (proto_helloworld.IFluffyCoreGreeterServer, error) {
	return &service{
		config:               config,
		scopedSomeDisposable: scopedSomeDisposable,
	}, nil
}
func AddGreeterService(builder di.ContainerBuilder) {
	proto_helloworld.AddGreeterServerWithExternalRegistration(builder,
		stemService.Ctor,
		func() endpoint.IEndpointRegistration {
			return &registrationServer{}
		})
}
func (s *service) SayHello(ctx context.Context, request *proto_helloworld.HelloRequest) (*proto_helloworld.HelloReply, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Msg("SayHello")
	return &proto_helloworld.HelloReply{
		Message: "Hello " + request.Name,
	}, nil
}
