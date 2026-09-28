package greeter

import (
	"context"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_GRPCClientFactory "github.com/fluffy-bunny/fluffycore/contracts/GRPCClientFactory"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	endpoint "github.com/fluffy-bunny/fluffycore/contracts/endpoint"
	nats_micro_service "github.com/fluffy-bunny/fluffycore/contracts/nats_micro_service"
	contracts_config "github.com/fluffy-bunny/fluffycore/example/internal/contracts/config"
	fluffycore_contracts_secretstore "github.com/fluffy-bunny/fluffycore/example/internal/contracts/secretstore"
	fluffycore_contracts_somedisposable "github.com/fluffy-bunny/fluffycore/example/internal/contracts/somedisposable"
	fluffycore_grpcclient "github.com/fluffy-bunny/fluffycore/grpcclient"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	proto_hellowworld_models "github.com/fluffy-bunny/fluffycore/proto/helloworld/models"
	fluffycore_servertls "github.com/fluffy-bunny/fluffycore/runtime/servertls"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	status "github.com/gogo/status"
	grpc_gateway_runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	nats "github.com/nats-io/nats.go"
	micro "github.com/nats-io/nats.go/micro"
	zerolog "github.com/rs/zerolog"
	otel "go.opentelemetry.io/otel"
	attribute "go.opentelemetry.io/otel/attribute"
	trace "go.opentelemetry.io/otel/trace"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
)

type (
	service struct {
		proto_helloworld.GreeterFluffyCoreServer

		config               *contracts_config.Config
		scopedSomeDisposable fluffycore_contracts_somedisposable.IScopedSomeDisposable
		grpcClientFactory    fluffycore_contracts_GRPCClientFactory.IGRPCClientFactory
		secretStore          fluffycore_contracts_secretstore.ISecretStore
		claimsPrincipal      fluffycore_contracts_common.IClaimsPrincipal
	}
	registrationServer struct {
		proto_helloworld.GreeterFluffyCoreServer
	}
)

var stemService = (*service)(nil)
var tracer = otel.Tracer("grpc-example")

var _ proto_helloworld.IFluffyCoreGreeterServer = stemService
var _ endpoint.IEndpointRegistration = (*registrationServer)(nil)
var _ endpoint.INATSEndpointRegistration = (*registrationServer)(nil)

func (s *registrationServer) RegisterHandler(gwmux *grpc_gateway_runtime.ServeMux, conn *grpc.ClientConn) {
	proto_helloworld.RegisterGreeterHandler(context.Background(), gwmux, conn)
}
func (s *registrationServer) RegisterFluffyCoreNATSHandler(ctx context.Context, natsCon *nats.Conn, conn *grpc.ClientConn, option *nats_micro_service.NATSMicroServiceRegisrationOption) (micro.Service, error) {
	return proto_helloworld.RegisterGreeterNATSHandler(ctx, natsCon, conn, option)
}

func (s *service) Ctor(
	config *contracts_config.Config,
	grpcClientFactory fluffycore_contracts_GRPCClientFactory.IGRPCClientFactory,
	scopedSomeDisposable fluffycore_contracts_somedisposable.IScopedSomeDisposable,
	secretStore fluffycore_contracts_secretstore.ISecretStore,
	claimsPrincipal fluffycore_contracts_common.IClaimsPrincipal) (proto_helloworld.IFluffyCoreGreeterServer, error) {

	return &service{
		config:               config,
		scopedSomeDisposable: scopedSomeDisposable,
		grpcClientFactory:    grpcClientFactory,
		secretStore:          secretStore,
		claimsPrincipal:      claimsPrincipal,
	}, nil
}
func AddGreeterService(builder di.ContainerBuilder) {
	proto_helloworld.AddGreeterServerWithExternalRegistration(builder,
		stemService.Ctor,
		func() endpoint.IEndpointRegistration {
			return &registrationServer{}
		})
	natsEnabled := fluffycore_utils.BoolEnv("NATS_ENABLED", false)
	if natsEnabled {
		// need a nats server with an auth callout for this to work
		proto_helloworld.AddSingletonGreeterFluffyCoreServerNATSMicroRegistration(builder)
	}

}
func (s *service) SayHelloAuth(ctx context.Context, request *proto_hellowworld_models.HelloRequest) (*proto_hellowworld_models.HelloReply, error) {
	return s.SayHello(ctx, request)
}

func (s *service) SayHelloDownstream(ctx context.Context, request *proto_hellowworld_models.HelloRequest) (*proto_hellowworld_models.HelloReply, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Msg("SayHelloDownstream")

	return &proto_hellowworld_models.HelloReply{
		Message: "Hello " + request.Name,
	}, nil
}
func (s *service) SayHello(ctx context.Context, request *proto_hellowworld_models.HelloRequest) (*proto_hellowworld_models.HelloReply, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Msg("SayHello")
	s.workHard(ctx)
	time.Sleep(50 * time.Millisecond)

	opts := s.downstreamClientOptions()

	grpcClient, err := s.grpcClientFactory.NewGrpcClient(opts...)
	if err != nil {
		log.Error().Err(err).Msg("Creating gRPC client")
		return nil, err
	}
	defer grpcClient.Close()

	cli := proto_helloworld.NewGreeterClient(grpcClient.GetConnection())
	reply, err := cli.SayHelloDownstream(ctx, request)
	return reply, err
}

// SetSecret stores a secret value. Callable with either a normal JWT bearer
// token or mutual TLS -- see example/internal/auth.BuildGrpcEntrypointPermissionsClaimsMap,
// which requires (authenticated via JWT) OR (mtls_verified) for this method.
func (s *service) SetSecret(ctx context.Context, request *proto_hellowworld_models.SetSecretRequest) (*proto_hellowworld_models.SetSecretResponse, error) {
	owner, err := s.secretOwner()
	if err != nil {
		return nil, err
	}
	s.secretStore.Set(owner, request.OrgId, request.Key, request.Value)
	return &proto_hellowworld_models.SetSecretResponse{Success: true}, nil
}

// GetSecret retrieves a previously-set secret value. Same auth requirement as
// SetSecret: JWT or mTLS.
func (s *service) GetSecret(ctx context.Context, request *proto_hellowworld_models.GetSecretRequest) (*proto_hellowworld_models.GetSecretResponse, error) {
	owner, err := s.secretOwner()
	if err != nil {
		return nil, err
	}
	value, ok := s.secretStore.Get(owner, request.OrgId, request.Key)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "no secret found for org_id=%q key=%q", request.OrgId, request.Key)
	}
	return &proto_hellowworld_models.GetSecretResponse{Value: value}, nil
}

// downstreamClientOptions dials this same server over loopback. It used to hardcode a
// plaintext localhost:50051, which fails whenever the server runs TLS -- the example's
// default -- and ignored the configured port.
func (s *service) downstreamClientOptions() []fluffycore_grpcclient.GrpcClientOption {
	opts := []fluffycore_grpcclient.GrpcClientOption{
		fluffycore_grpcclient.WithHost("localhost"),
		fluffycore_grpcclient.WithPort(s.config.PORT),
	}
	if trustFile, ok := fluffycore_servertls.LoopbackTrustFile(&s.config.CoreConfig); ok {
		// WithInsecure(false) is required: grpcclient defaults to insecure and would otherwise
		// ignore the cert bundle and dial plaintext.
		return append(opts, fluffycore_grpcclient.WithInsecure(false), fluffycore_grpcclient.WithCertBundle(trustFile))
	}
	return append(opts, fluffycore_grpcclient.WithInsecure(true))
}

// secretOwner identifies the caller from what the server itself authenticated -- never a
// request field -- so SetSecret/GetSecret callers can only see their own secrets. request.OrgId
// is just a namespace within the caller's own secrets: trusting it alone let any
// authenticated caller read or overwrite any org's secrets.
func (s *service) secretOwner() (string, error) {
	for _, c := range s.claimsPrincipal.GetClaimsByType(fluffycore_wellknown.ClaimTypeSub) {
		if c.Value != "" && c.Value != fluffycore_wellknown.AnonymousSubject {
			return "jwt:" + c.Value, nil
		}
	}
	// mTLS: the verified client certificate. The fingerprint, not the CN: any cert the CA
	// issues could carry the same CN.
	for _, c := range s.claimsPrincipal.GetClaimsByType(fluffycore_wellknown.ClaimTypeMTLSFingerprint) {
		if c.Value != "" {
			return "mtls:" + c.Value, nil
		}
	}
	return "", status.Error(codes.PermissionDenied, "no authenticated caller identity")
}

func (s *service) workHard(ctx context.Context) {
	_, span := tracer.Start(ctx, "workHard",
		trace.WithAttributes(attribute.String("extra.key", "extra.value")))
	defer span.End()

	time.Sleep(50 * time.Millisecond)
}
