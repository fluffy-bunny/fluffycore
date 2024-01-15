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

// IFluffyCoreGreeterServer defines the grpc server
type IFluffyCoreGreeterServer interface {
	GreeterServer
}

type UnimplementedFluffyCoreGreeterServerEndpointRegistration struct {
}

func (UnimplementedFluffyCoreGreeterServerEndpointRegistration) RegisterHandler(gwmux *runtime.ServeMux, conn *grpc.ClientConn) {
}

// GreeterFluffyCoreServer defines the grpc server truct
type GreeterFluffyCoreServer struct {
	UnimplementedGreeterServer
	UnimplementedFluffyCoreGreeterServerEndpointRegistration
}

// Register the server with grpc
func (srv *GreeterFluffyCoreServer) Register(s *grpc.Server) {
	RegisterGreeterServer(s, srv)
}

// AddGreeterServerWithExternalRegistration adds the fluffycore aware grpc server and external registration service.  Mainly used for grpc-gateway
func AddGreeterServerWithExternalRegistration[T IFluffyCoreGreeterServer](cb fluffy_dozm_di.ContainerBuilder, ctor any, register func() endpoint.IEndpointRegistration) {
	fluffy_dozm_di.AddSingleton[endpoint.IEndpointRegistration](cb, register)
	fluffy_dozm_di.AddScoped[IFluffyCoreGreeterServer](cb, ctor)
}

// AddGreeterServer adds the fluffycore aware grpc server
func AddGreeterServer[T IFluffyCoreGreeterServer](cb fluffy_dozm_di.ContainerBuilder, ctor any) {
	AddGreeterServerWithExternalRegistration[IFluffyCoreGreeterServer](cb, ctor, func() endpoint.IEndpointRegistration {
		return &GreeterFluffyCoreServer{}
	})
}

// SayHello...
func (s *GreeterFluffyCoreServer) SayHello(ctx context.Context, request *HelloRequest) (*HelloReply, error) {
	requestContainer := dicontext.GetRequestContainer(ctx)
	downstreamService := fluffy_dozm_di.Get[IFluffyCoreGreeterServer](requestContainer)
	return downstreamService.SayHello(ctx, request)
}

// IFluffyCoreGreeter2Server defines the grpc server
type IFluffyCoreGreeter2Server interface {
	Greeter2Server
}

type UnimplementedFluffyCoreGreeter2ServerEndpointRegistration struct {
}

func (UnimplementedFluffyCoreGreeter2ServerEndpointRegistration) RegisterHandler(gwmux *runtime.ServeMux, conn *grpc.ClientConn) {
}

// Greeter2FluffyCoreServer defines the grpc server truct
type Greeter2FluffyCoreServer struct {
	UnimplementedGreeter2Server
	UnimplementedFluffyCoreGreeter2ServerEndpointRegistration
}

// Register the server with grpc
func (srv *Greeter2FluffyCoreServer) Register(s *grpc.Server) {
	RegisterGreeter2Server(s, srv)
}

// AddGreeter2ServerWithExternalRegistration adds the fluffycore aware grpc server and external registration service.  Mainly used for grpc-gateway
func AddGreeter2ServerWithExternalRegistration[T IFluffyCoreGreeter2Server](cb fluffy_dozm_di.ContainerBuilder, ctor any, register func() endpoint.IEndpointRegistration) {
	fluffy_dozm_di.AddSingleton[endpoint.IEndpointRegistration](cb, register)
	fluffy_dozm_di.AddScoped[IFluffyCoreGreeter2Server](cb, ctor)
}

// AddGreeter2Server adds the fluffycore aware grpc server
func AddGreeter2Server[T IFluffyCoreGreeter2Server](cb fluffy_dozm_di.ContainerBuilder, ctor any) {
	AddGreeter2ServerWithExternalRegistration[IFluffyCoreGreeter2Server](cb, ctor, func() endpoint.IEndpointRegistration {
		return &Greeter2FluffyCoreServer{}
	})
}

// SayHello...
func (s *Greeter2FluffyCoreServer) SayHello(ctx context.Context, request *HelloRequest) (*HelloReply2, error) {
	requestContainer := dicontext.GetRequestContainer(ctx)
	downstreamService := fluffy_dozm_di.Get[IFluffyCoreGreeter2Server](requestContainer)
	return downstreamService.SayHello(ctx, request)
}

// IFluffyCoreMyStreamServiceServer defines the grpc server
type IFluffyCoreMyStreamServiceServer interface {
	MyStreamServiceServer
}

type UnimplementedFluffyCoreMyStreamServiceServerEndpointRegistration struct {
}

func (UnimplementedFluffyCoreMyStreamServiceServerEndpointRegistration) RegisterHandler(gwmux *runtime.ServeMux, conn *grpc.ClientConn) {
}

// MyStreamServiceFluffyCoreServer defines the grpc server truct
type MyStreamServiceFluffyCoreServer struct {
	UnimplementedMyStreamServiceServer
	UnimplementedFluffyCoreMyStreamServiceServerEndpointRegistration
}

// Register the server with grpc
func (srv *MyStreamServiceFluffyCoreServer) Register(s *grpc.Server) {
	RegisterMyStreamServiceServer(s, srv)
}

// AddMyStreamServiceServerWithExternalRegistration adds the fluffycore aware grpc server and external registration service.  Mainly used for grpc-gateway
func AddMyStreamServiceServerWithExternalRegistration[T IFluffyCoreMyStreamServiceServer](cb fluffy_dozm_di.ContainerBuilder, ctor any, register func() endpoint.IEndpointRegistration) {
	fluffy_dozm_di.AddSingleton[endpoint.IEndpointRegistration](cb, register)
	fluffy_dozm_di.AddScoped[IFluffyCoreMyStreamServiceServer](cb, ctor)
}

// AddMyStreamServiceServer adds the fluffycore aware grpc server
func AddMyStreamServiceServer[T IFluffyCoreMyStreamServiceServer](cb fluffy_dozm_di.ContainerBuilder, ctor any) {
	AddMyStreamServiceServerWithExternalRegistration[IFluffyCoreMyStreamServiceServer](cb, ctor, func() endpoint.IEndpointRegistration {
		return &MyStreamServiceFluffyCoreServer{}
	})
}

// RequestPoints...
func (s *MyStreamServiceFluffyCoreServer) RequestPoints(request *PointsRequest, stream MyStreamService_RequestPointsServer) error {
	ctx := stream.Context()
	requestContainer := dicontext.GetRequestContainer(ctx)
	downstreamService := fluffy_dozm_di.Get[IFluffyCoreMyStreamServiceServer](requestContainer)
	return downstreamService.RequestPoints(request, stream)
}

// StreamPoints...
func (s *MyStreamServiceFluffyCoreServer) StreamPoints(stream MyStreamService_StreamPointsServer) error {
	ctx := stream.Context()
	requestContainer := dicontext.GetRequestContainer(ctx)
	downstreamService := fluffy_dozm_di.Get[IFluffyCoreMyStreamServiceServer](requestContainer)
	return downstreamService.StreamPoints(stream)
}
