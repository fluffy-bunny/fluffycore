package main

import (
	context "context"
	"flag"
	"fmt"

	"net"

	"github.com/dozm/di"
	fluffycore_contracts_endpoint "github.com/fluffy-bunny/fluffycore/contracts/endpoint"
	fluffycore_middleware "github.com/fluffy-bunny/fluffycore/middleware"

	fluffycore_middleware_dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_middleware_logging "github.com/fluffy-bunny/fluffycore/middleware/logging"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	"github.com/gogo/status"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_zerolog "github.com/philip-bui/grpc-zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

type (
	GreeterService struct {
		proto_helloworld.UnimplementedGreeterServer
	}
)

func (s *GreeterService) SayHello(ctx context.Context, request *proto_helloworld.HelloRequest) (*proto_helloworld.HelloReply, error) {
	return nil, status.Error(codes.Unimplemented, "method SayHello not implemented")
}

var rootContainer di.Container

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatal().Msgf("failed to listen: %v", err)
	}
	b := di.Builder()
	b.ConfigureOptions(func(o *di.Options) {
		o.ValidateScopes = true
		o.ValidateOnBuild = true
	})
	proto_helloworld.AddGreeterServer[proto_helloworld.IGreeterServer](b, func() proto_helloworld.IGreeterServer {
		return &GreeterService{}
	})
	rootContainer = b.Build()

	customRecoveryFunc := func(p any) (err error) {
		return status.Errorf(codes.Unknown, "panic triggered: %v", p)
	}
	optsRecovery := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(customRecoveryFunc),
	}

	grpc_zerolog.TimestampLog = false
	logger := log.With().Caller().Logger()

	unaryServerInterceptorBuilder := fluffycore_middleware.NewUnaryServerInterceptorBuilder()
	streamServerInterceptorBuilder := fluffycore_middleware.NewStreamServerInterceptorBuilder()

	unaryServerInterceptorBuilder.Use(grpc_ctxtags.UnaryServerInterceptor())

	unaryServerInterceptorBuilder.Use(grpc_zerolog.NewUnaryServerInterceptorWithLogger(&logger))
	unaryServerInterceptorBuilder.Use(fluffycore_middleware_logging.EnsureContextLoggingUnaryServerInterceptor())
	unaryServerInterceptorBuilder.Use(fluffycore_middleware_dicontext.UnaryServerInterceptor(rootContainer))
	unaryServerInterceptorBuilder.Use(grpc_recovery.UnaryServerInterceptor(optsRecovery...))

	streamServerInterceptorBuilder.Use(grpc_ctxtags.StreamServerInterceptor())
	streamServerInterceptorBuilder.Use(grpc_recovery.StreamServerInterceptor(optsRecovery...))

	s := grpc.NewServer(

		grpc.ChainUnaryInterceptor(
			unaryServerInterceptorBuilder.GetUnaryServerInterceptors()...,
		),
		grpc.ChainStreamInterceptor(
			streamServerInterceptorBuilder.GetStreamServerInterceptors()...,
		),
	)
	endpointRegistrations := di.Get[[]fluffycore_contracts_endpoint.IEndpointRegistration](rootContainer)
	for _, endpointRegistration := range endpointRegistrations {
		endpointRegistration.Register(s)
	}

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatal().Msgf("failed to serve: %v", err)
	}
}
