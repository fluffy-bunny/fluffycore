package greeter

import (
	"context"

	"github.com/dozm/di"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
)

type (
	service struct {
		proto_helloworld.UnimplementedGreeterServer
	}
)

func (s *service) SayHello(ctx context.Context, request *proto_helloworld.HelloRequest) (*proto_helloworld.HelloReply, error) {
	return nil, status.Error(codes.Unimplemented, "method SayHello not implemented")
}
func AddGreeterService(cb di.ContainerBuilder) {
	di.AddSingleton[proto_helloworld.IGreeterServer](cb, func() proto_helloworld.IGreeterServer {
		return &service{}
	})
}
