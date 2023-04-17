package greeter

import (
	"context"

	"github.com/dozm/di"
	contracts_config "github.com/fluffy-bunny/fluffycore/example/internal/contracts/config"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	"github.com/gogo/status"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
)

type (
	service struct {
		proto_helloworld.UnimplementedGreeterServer
		config *contracts_config.Config
	}
)

func (s *service) SayHello(ctx context.Context, request *proto_helloworld.HelloRequest) (*proto_helloworld.HelloReply, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Msg("SayHello")
	return nil, status.Errorf(codes.Unimplemented, "method SayHello not implemented for %s", s.config.CoreConfig.ApplicationName)
}
func AddGreeterService(builder di.ContainerBuilder) {
	proto_helloworld.AddGreeterServer[proto_helloworld.IGreeterServer](builder, func(config *contracts_config.Config) proto_helloworld.IGreeterServer {
		return &service{
			config: config,
		}
	})
}
