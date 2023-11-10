// Code generated by protoc-gen-go-fluffycore-di. DO NOT EDIT.
// Code generated grpcGateway

package helloworld

import (
	context "context"
	fluffy_dozm_di "github.com/fluffy-bunny/fluffy-dozm-di"
	endpoint "github.com/fluffy-bunny/fluffycore/contracts/endpoint"
	dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	grpc "google.golang.org/grpc"
)

// IGreeterServer defines the grpc server
type IGreeterServer interface {
	GreeterServer
}

type UnimplementedGreeterServerEndpointRegistration struct {
}

func (UnimplementedGreeterServerEndpointRegistration) RegisterHandler(gwmux *runtime.ServeMux, conn *grpc.ClientConn) {
}

// GreeterFluffyCoreServer defines the grpc server truct
type GreeterFluffyCoreServer struct {
	UnimplementedGreeterServer
	UnimplementedGreeterServerEndpointRegistration
}

// Register the server with grpc
func (srv *GreeterFluffyCoreServer) Register(s *grpc.Server) {
	RegisterGreeterServer(s, srv)
}

// AddGreeterServerWithExternalRegistration adds the fluffycore aware grpc server and external registration service.  Mainly used for grpc-gateway
func AddGreeterServerWithExternalRegistration[T IGreeterServer](cb fluffy_dozm_di.ContainerBuilder, ctor any, register func() endpoint.IEndpointRegistration) {
	fluffy_dozm_di.AddSingleton[endpoint.IEndpointRegistration](cb, register)
	fluffy_dozm_di.AddScoped[IGreeterServer](cb, ctor)
}

// AddGreeterServer adds the fluffycore aware grpc server
func AddGreeterServer[T IGreeterServer](cb fluffy_dozm_di.ContainerBuilder, ctor any) {
	AddGreeterServerWithExternalRegistration[IGreeterServer](cb, ctor, func() endpoint.IEndpointRegistration {
		return &GreeterFluffyCoreServer{}
	})
}

// SayHello...
func (s *GreeterFluffyCoreServer) SayHello(ctx context.Context, request *HelloRequest) (*HelloReply, error) {
	requestContainer := dicontext.GetRequestContainer(ctx)
	downstreamService := fluffy_dozm_di.Get[IGreeterServer](requestContainer)
	return downstreamService.SayHello(ctx, request)
}

// IGreeter2Server defines the grpc server
type IGreeter2Server interface {
	Greeter2Server
}

type UnimplementedGreeter2ServerEndpointRegistration struct {
}

func (UnimplementedGreeter2ServerEndpointRegistration) RegisterHandler(gwmux *runtime.ServeMux, conn *grpc.ClientConn) {
}

// Greeter2FluffyCoreServer defines the grpc server truct
type Greeter2FluffyCoreServer struct {
	UnimplementedGreeter2Server
	UnimplementedGreeter2ServerEndpointRegistration
}

// Register the server with grpc
func (srv *Greeter2FluffyCoreServer) Register(s *grpc.Server) {
	RegisterGreeter2Server(s, srv)
}

// AddGreeter2ServerWithExternalRegistration adds the fluffycore aware grpc server and external registration service.  Mainly used for grpc-gateway
func AddGreeter2ServerWithExternalRegistration[T IGreeter2Server](cb fluffy_dozm_di.ContainerBuilder, ctor any, register func() endpoint.IEndpointRegistration) {
	fluffy_dozm_di.AddSingleton[endpoint.IEndpointRegistration](cb, register)
	fluffy_dozm_di.AddScoped[IGreeter2Server](cb, ctor)
}

// AddGreeter2Server adds the fluffycore aware grpc server
func AddGreeter2Server[T IGreeter2Server](cb fluffy_dozm_di.ContainerBuilder, ctor any) {
	AddGreeter2ServerWithExternalRegistration[IGreeter2Server](cb, ctor, func() endpoint.IEndpointRegistration {
		return &Greeter2FluffyCoreServer{}
	})
}

// SayHello...
func (s *Greeter2FluffyCoreServer) SayHello(ctx context.Context, request *HelloRequest) (*HelloReply2, error) {
	requestContainer := dicontext.GetRequestContainer(ctx)
	downstreamService := fluffy_dozm_di.Get[IGreeter2Server](requestContainer)
	return downstreamService.SayHello(ctx, request)
}

// IMyStreamServiceServer defines the grpc server
type IMyStreamServiceServer interface {
	MyStreamServiceServer
}

type UnimplementedMyStreamServiceServerEndpointRegistration struct {
}

func (UnimplementedMyStreamServiceServerEndpointRegistration) RegisterHandler(gwmux *runtime.ServeMux, conn *grpc.ClientConn) {
}

// MyStreamServiceFluffyCoreServer defines the grpc server truct
type MyStreamServiceFluffyCoreServer struct {
	UnimplementedMyStreamServiceServer
	UnimplementedMyStreamServiceServerEndpointRegistration
}

// Register the server with grpc
func (srv *MyStreamServiceFluffyCoreServer) Register(s *grpc.Server) {
	RegisterMyStreamServiceServer(s, srv)
}

// AddMyStreamServiceServerWithExternalRegistration adds the fluffycore aware grpc server and external registration service.  Mainly used for grpc-gateway
func AddMyStreamServiceServerWithExternalRegistration[T IMyStreamServiceServer](cb fluffy_dozm_di.ContainerBuilder, ctor any, register func() endpoint.IEndpointRegistration) {
	fluffy_dozm_di.AddSingleton[endpoint.IEndpointRegistration](cb, register)
	fluffy_dozm_di.AddScoped[IMyStreamServiceServer](cb, ctor)
}

// AddMyStreamServiceServer adds the fluffycore aware grpc server
func AddMyStreamServiceServer[T IMyStreamServiceServer](cb fluffy_dozm_di.ContainerBuilder, ctor any) {
	AddMyStreamServiceServerWithExternalRegistration[IMyStreamServiceServer](cb, ctor, func() endpoint.IEndpointRegistration {
		return &MyStreamServiceFluffyCoreServer{}
	})
}

// RequestPoints...
func (s *MyStreamServiceFluffyCoreServer) RequestPoints(request *PointsRequest, stream MyStreamService_RequestPointsServer) error {
	ctx := stream.Context()
	requestContainer := dicontext.GetRequestContainer(ctx)
	downstreamService := fluffy_dozm_di.Get[IMyStreamServiceServer](requestContainer)
	return downstreamService.RequestPoints(request, stream)
}

// StreamPoints...
func (s *MyStreamServiceFluffyCoreServer) StreamPoints(stream MyStreamService_StreamPointsServer) error {
	ctx := stream.Context()
	requestContainer := dicontext.GetRequestContainer(ctx)
	downstreamService := fluffy_dozm_di.Get[IMyStreamServiceServer](requestContainer)
	return downstreamService.StreamPoints(stream)
}
