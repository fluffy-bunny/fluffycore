// Code generated by protoc-gen-go-fluffycore-nats. DO NOT EDIT.

package helloworld

import (
	context "context"
	fmt "fmt"
	fluffy_dozm_di "github.com/fluffy-bunny/fluffy-dozm-di"
	endpoint "github.com/fluffy-bunny/fluffycore/contracts/endpoint"
	nats_micro_service "github.com/fluffy-bunny/fluffycore/contracts/nats_micro_service"
	nats_micro_service1 "github.com/fluffy-bunny/fluffycore/nats/nats_micro_service"
	utils "github.com/fluffy-bunny/fluffycore/utils"
	nats_go "github.com/nats-io/nats.go"
	micro "github.com/nats-io/nats.go/micro"
	grpc "google.golang.org/grpc"
	protojson "google.golang.org/protobuf/encoding/protojson"
	reflect "reflect"
	strings "strings"
	time "time"
)

type GreeterFluffyCoreServerNATSMicroRegistration struct {
}

var stemServiceGreeterFluffyCoreServerNATSMicroRegistration = (*GreeterFluffyCoreServerNATSMicroRegistration)(nil)
var _ endpoint.INATSEndpointRegistration = stemServiceGreeterFluffyCoreServerNATSMicroRegistration

func AddSingletonGreeterFluffyCoreServerNATSMicroRegistration(cb fluffy_dozm_di.ContainerBuilder) {
	fluffy_dozm_di.AddSingleton[endpoint.INATSEndpointRegistration](cb, stemServiceGreeterFluffyCoreServerNATSMicroRegistration.Ctor)
}

func (s *GreeterFluffyCoreServerNATSMicroRegistration) Ctor() (endpoint.INATSEndpointRegistration, error) {
	return &GreeterFluffyCoreServerNATSMicroRegistration{}, nil
}

func (s *GreeterFluffyCoreServerNATSMicroRegistration) RegisterFluffyCoreNATSHandler(ctx context.Context, natsConn *nats_go.Conn, conn *grpc.ClientConn, option *nats_micro_service.NATSMicroServiceRegisrationOption) (micro.Service, error) {
	return RegisterGreeterNATSHandler(ctx, natsConn, conn, option)
}

func RegisterGreeterNATSHandler(ctx context.Context, natsCon *nats_go.Conn, conn *grpc.ClientConn, option *nats_micro_service.NATSMicroServiceRegisrationOption) (micro.Service, error) {
	client := NewGreeterClient(conn)
	return RegisterGreeterNATSHandlerClient(ctx, natsCon, client, option)
}

func RegisterGreeterNATSHandlerClient(ctx context.Context, nc *nats_go.Conn, client GreeterClient, option *nats_micro_service.NATSMicroServiceRegisrationOption) (micro.Service, error) {
	defaultConfig := &micro.Config{
		Name:        "Greeter",
		Version:     "0.0.1",
		Description: "The Greeter nats micro service",
	}

	for _, option := range option.NATSMicroConfigOptions {
		option(defaultConfig)
	}

	svc, err := micro.AddService(nc, *defaultConfig)
	if err != nil {
		return nil, err
	}

	pkgPath := reflect.TypeOf((*GreeterServer)(nil)).Elem().PkgPath()
	fullPath := fmt.Sprintf("%s/%s", pkgPath, "Greeter")
	groupName := strings.ReplaceAll(
		fullPath,
		"/",
		".",
	)

	if utils.IsNotEmptyOrNil(option.GroupName) {
		groupName = option.GroupName
	}

	m := svc.AddGroup(groupName)
	m.AddEndpoint("SayHello",
		micro.HandlerFunc(func(req micro.Request) {
			nats_micro_service1.HandleRequest(
				req,
				func(r *HelloRequest) error {
					return protojson.Unmarshal(req.Data(), r)
				},
				func(ctx context.Context, request *HelloRequest) (*HelloReply, error) {
					return client.SayHello(ctx, request)
				},
			)
		}),
		micro.WithEndpointMetadata(map[string]string{
			"description":     "SayHello",
			"format":          "application/json",
			"request_schema":  utils.SchemaFor(&HelloRequest{}),
			"response_schema": utils.SchemaFor(&HelloReply{}),
		}))

	m.AddEndpoint("SayHelloAuth",
		micro.HandlerFunc(func(req micro.Request) {
			nats_micro_service1.HandleRequest(
				req,
				func(r *HelloRequest) error {
					return protojson.Unmarshal(req.Data(), r)
				},
				func(ctx context.Context, request *HelloRequest) (*HelloReply, error) {
					return client.SayHelloAuth(ctx, request)
				},
			)
		}),
		micro.WithEndpointMetadata(map[string]string{
			"description":     "SayHelloAuth",
			"format":          "application/json",
			"request_schema":  utils.SchemaFor(&HelloRequest{}),
			"response_schema": utils.SchemaFor(&HelloReply{}),
		}))

	m.AddEndpoint("SayHelloDownstream",
		micro.HandlerFunc(func(req micro.Request) {
			nats_micro_service1.HandleRequest(
				req,
				func(r *HelloRequest) error {
					return protojson.Unmarshal(req.Data(), r)
				},
				func(ctx context.Context, request *HelloRequest) (*HelloReply, error) {
					return client.SayHelloDownstream(ctx, request)
				},
			)
		}),
		micro.WithEndpointMetadata(map[string]string{
			"description":     "SayHelloDownstream",
			"format":          "application/json",
			"request_schema":  utils.SchemaFor(&HelloRequest{}),
			"response_schema": utils.SchemaFor(&HelloReply{}),
		}))

	return svc, nil
}

type (
	GreeterNATSMicroClient struct {
		option    *nats_micro_service1.NATSClientOption
		groupName string
	}
)

func NewGreeterNATSMicroClient(option *nats_micro_service1.NATSClientOption) (GreeterClient, error) {
	pkgPath := reflect.TypeOf((*GreeterServer)(nil)).Elem().PkgPath()
	fullPath := fmt.Sprintf("%s/%s", pkgPath, "Greeter")
	groupName := strings.ReplaceAll(
		fullPath,
		"/",
		".",
	)
	if option.Timeout == 0 {
		option.Timeout = time.Second * 2
	}
	return &GreeterNATSMicroClient{
		option:    option,
		groupName: groupName,
	}, nil
}

// SayHello...
func (s *GreeterNATSMicroClient) SayHello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error) {
	response := &HelloReply{}
	result, err := nats_micro_service1.HandleNATSClientRequest(
		ctx,
		s.option.NC,
		fmt.Sprintf("%s.SayHello", s.groupName),
		in,
		response,
		s.option.Timeout,
	)
	return result, err
}

// SayHelloAuth...
func (s *GreeterNATSMicroClient) SayHelloAuth(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error) {
	response := &HelloReply{}
	result, err := nats_micro_service1.HandleNATSClientRequest(
		ctx,
		s.option.NC,
		fmt.Sprintf("%s.SayHelloAuth", s.groupName),
		in,
		response,
		s.option.Timeout,
	)
	return result, err
}

// SayHelloDownstream...
func (s *GreeterNATSMicroClient) SayHelloDownstream(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error) {
	response := &HelloReply{}
	result, err := nats_micro_service1.HandleNATSClientRequest(
		ctx,
		s.option.NC,
		fmt.Sprintf("%s.SayHelloDownstream", s.groupName),
		in,
		response,
		s.option.Timeout,
	)
	return result, err
}

type Greeter2FluffyCoreServerNATSMicroRegistration struct {
}

var stemServiceGreeter2FluffyCoreServerNATSMicroRegistration = (*Greeter2FluffyCoreServerNATSMicroRegistration)(nil)
var _ endpoint.INATSEndpointRegistration = stemServiceGreeter2FluffyCoreServerNATSMicroRegistration

func AddSingletonGreeter2FluffyCoreServerNATSMicroRegistration(cb fluffy_dozm_di.ContainerBuilder) {
	fluffy_dozm_di.AddSingleton[endpoint.INATSEndpointRegistration](cb, stemServiceGreeter2FluffyCoreServerNATSMicroRegistration.Ctor)
}

func (s *Greeter2FluffyCoreServerNATSMicroRegistration) Ctor() (endpoint.INATSEndpointRegistration, error) {
	return &Greeter2FluffyCoreServerNATSMicroRegistration{}, nil
}

func (s *Greeter2FluffyCoreServerNATSMicroRegistration) RegisterFluffyCoreNATSHandler(ctx context.Context, natsConn *nats_go.Conn, conn *grpc.ClientConn, option *nats_micro_service.NATSMicroServiceRegisrationOption) (micro.Service, error) {
	return RegisterGreeter2NATSHandler(ctx, natsConn, conn, option)
}

func RegisterGreeter2NATSHandler(ctx context.Context, natsCon *nats_go.Conn, conn *grpc.ClientConn, option *nats_micro_service.NATSMicroServiceRegisrationOption) (micro.Service, error) {
	client := NewGreeter2Client(conn)
	return RegisterGreeter2NATSHandlerClient(ctx, natsCon, client, option)
}

func RegisterGreeter2NATSHandlerClient(ctx context.Context, nc *nats_go.Conn, client Greeter2Client, option *nats_micro_service.NATSMicroServiceRegisrationOption) (micro.Service, error) {
	defaultConfig := &micro.Config{
		Name:        "Greeter2",
		Version:     "0.0.1",
		Description: "The Greeter2 nats micro service",
	}

	for _, option := range option.NATSMicroConfigOptions {
		option(defaultConfig)
	}

	svc, err := micro.AddService(nc, *defaultConfig)
	if err != nil {
		return nil, err
	}

	pkgPath := reflect.TypeOf((*Greeter2Server)(nil)).Elem().PkgPath()
	fullPath := fmt.Sprintf("%s/%s", pkgPath, "Greeter2")
	groupName := strings.ReplaceAll(
		fullPath,
		"/",
		".",
	)

	if utils.IsNotEmptyOrNil(option.GroupName) {
		groupName = option.GroupName
	}

	m := svc.AddGroup(groupName)
	m.AddEndpoint("SayHello",
		micro.HandlerFunc(func(req micro.Request) {
			nats_micro_service1.HandleRequest(
				req,
				func(r *HelloRequest) error {
					return protojson.Unmarshal(req.Data(), r)
				},
				func(ctx context.Context, request *HelloRequest) (*HelloReply2, error) {
					return client.SayHello(ctx, request)
				},
			)
		}),
		micro.WithEndpointMetadata(map[string]string{
			"description":     "SayHello",
			"format":          "application/json",
			"request_schema":  utils.SchemaFor(&HelloRequest{}),
			"response_schema": utils.SchemaFor(&HelloReply2{}),
		}))

	return svc, nil
}

type (
	Greeter2NATSMicroClient struct {
		option    *nats_micro_service1.NATSClientOption
		groupName string
	}
)

func NewGreeter2NATSMicroClient(option *nats_micro_service1.NATSClientOption) (Greeter2Client, error) {
	pkgPath := reflect.TypeOf((*Greeter2Server)(nil)).Elem().PkgPath()
	fullPath := fmt.Sprintf("%s/%s", pkgPath, "Greeter2")
	groupName := strings.ReplaceAll(
		fullPath,
		"/",
		".",
	)
	if option.Timeout == 0 {
		option.Timeout = time.Second * 2
	}
	return &Greeter2NATSMicroClient{
		option:    option,
		groupName: groupName,
	}, nil
}

// SayHello...
func (s *Greeter2NATSMicroClient) SayHello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply2, error) {
	response := &HelloReply2{}
	result, err := nats_micro_service1.HandleNATSClientRequest(
		ctx,
		s.option.NC,
		fmt.Sprintf("%s.SayHello", s.groupName),
		in,
		response,
		s.option.Timeout,
	)
	return result, err
}

type MyStreamServiceFluffyCoreServerNATSMicroRegistration struct {
}

var stemServiceMyStreamServiceFluffyCoreServerNATSMicroRegistration = (*MyStreamServiceFluffyCoreServerNATSMicroRegistration)(nil)
var _ endpoint.INATSEndpointRegistration = stemServiceMyStreamServiceFluffyCoreServerNATSMicroRegistration

func AddSingletonMyStreamServiceFluffyCoreServerNATSMicroRegistration(cb fluffy_dozm_di.ContainerBuilder) {
	fluffy_dozm_di.AddSingleton[endpoint.INATSEndpointRegistration](cb, stemServiceMyStreamServiceFluffyCoreServerNATSMicroRegistration.Ctor)
}

func (s *MyStreamServiceFluffyCoreServerNATSMicroRegistration) Ctor() (endpoint.INATSEndpointRegistration, error) {
	return &MyStreamServiceFluffyCoreServerNATSMicroRegistration{}, nil
}

func (s *MyStreamServiceFluffyCoreServerNATSMicroRegistration) RegisterFluffyCoreNATSHandler(ctx context.Context, natsConn *nats_go.Conn, conn *grpc.ClientConn, option *nats_micro_service.NATSMicroServiceRegisrationOption) (micro.Service, error) {
	return RegisterMyStreamServiceNATSHandler(ctx, natsConn, conn, option)
}

func RegisterMyStreamServiceNATSHandler(ctx context.Context, natsCon *nats_go.Conn, conn *grpc.ClientConn, option *nats_micro_service.NATSMicroServiceRegisrationOption) (micro.Service, error) {
	client := NewMyStreamServiceClient(conn)
	return RegisterMyStreamServiceNATSHandlerClient(ctx, natsCon, client, option)
}

func RegisterMyStreamServiceNATSHandlerClient(ctx context.Context, nc *nats_go.Conn, client MyStreamServiceClient, option *nats_micro_service.NATSMicroServiceRegisrationOption) (micro.Service, error) {
	defaultConfig := &micro.Config{
		Name:        "MyStreamService",
		Version:     "0.0.1",
		Description: "The MyStreamService nats micro service",
	}

	for _, option := range option.NATSMicroConfigOptions {
		option(defaultConfig)
	}

	svc, err := micro.AddService(nc, *defaultConfig)
	if err != nil {
		return nil, err
	}

	return svc, nil
}
