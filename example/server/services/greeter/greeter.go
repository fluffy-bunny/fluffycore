package greeter

import (
	"context"

	"github.com/dozm/di"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	"github.com/gogo/status"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
)

type (
	service struct {
		proto_helloworld.UnimplementedGreeterServer
	}
)

func (s *service) SayHello(ctx context.Context, request *proto_helloworld.HelloRequest) (*proto_helloworld.HelloReply, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Msg("SayHello")
	return nil, status.Error(codes.Unimplemented, "method SayHello not implemented")
}
func AddGreeterService(builder di.ContainerBuilder) {
	proto_helloworld.AddGreeterServer[proto_helloworld.IGreeterServer](builder, func() proto_helloworld.IGreeterServer {
		return &service{}
	})
}
