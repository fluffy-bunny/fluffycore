package runtime

import (
	"context"

	fluffycore_contracts_otel "github.com/fluffy-bunny/fluffycore/contracts/otel"
	fluffycore_contracts_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	internal_auth "github.com/fluffy-bunny/fluffycore/example/internal/auth"
	fluffycore_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/middleware/auth/jwt"
	fluffycore_middleware_claimsprincipal "github.com/fluffy-bunny/fluffycore/middleware/claimsprincipal"
	fluffycore_middleware_correlation "github.com/fluffy-bunny/fluffycore/middleware/correlation"
	fluffycore_middleware_dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_middleware_logging "github.com/fluffy-bunny/fluffycore/middleware/logging"
	status "github.com/gogo/status"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	zerolog "github.com/rs/zerolog"
	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	otel "go.opentelemetry.io/otel"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
)

type (
	FluffyCoreOTELStartup struct {
		*OTELContainer // embedded  OTEL
		fluffycore_contracts_runtime.UnimplementedStartup
	}
)

func NewFluffyCoreOTELStartup() *FluffyCoreOTELStartup {
	obj := &FluffyCoreOTELStartup{}
	obj.OTELContainer = NewOTELContainer()
	return obj
}
func (s *FluffyCoreOTELStartup) SetConfig(config *fluffycore_contracts_otel.OTELConfig) {
	s.OTELContainer.Config = config
}
func (s *FluffyCoreOTELStartup) ConfigureServerOpts(ctx context.Context) []grpc.ServerOption {
	log := zerolog.Ctx(ctx).With().Str("method", "Configure").Logger()

	// initialized the OTEL stuff before we make our intercepters.
	s.OTELContainer.Init(ctx)
	var serverOpts []grpc.ServerOption
	otelOpts := []otelgrpc.Option{
		otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
		otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
	}
	log.Info().Msg("adding OTEL serverOpts")
	serverOpts = append(serverOpts, grpc.StatsHandler(otelgrpc.NewServerHandler(otelOpts...)))

	log.Info().Msg("adding ChainUnaryInterceptor: fluffycore_middleware_logging.EnsureContextLoggingUnaryServerInterceptor")
	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(fluffycore_middleware_logging.EnsureContextLoggingUnaryServerInterceptor()))

	log.Info().Msg("adding ChainStreamInterceptor: fluffycore_middleware_logging.EnsureContextLoggingStreamServerInterceptor")
	serverOpts = append(serverOpts, grpc.ChainStreamInterceptor(fluffycore_middleware_logging.EnsureContextLoggingStreamServerInterceptor()))

	// log correlation and spans
	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(fluffycore_middleware_correlation.EnsureOTELCorrelationIDUnaryServerInterceptor()))
	// dicontext is responsible of create a scoped context for each request.
	log.Info().Msg("adding ChainUnaryInterceptor: fluffycore_middleware_dicontext.UnaryServerInterceptor")
	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(fluffycore_middleware_dicontext.UnaryServerInterceptor(s.RootContainer)))
	log.Info().Msg("adding ChainStreamInterceptor: fluffycore_middleware_dicontext.StreamServerInterceptor")
	serverOpts = append(serverOpts, grpc.ChainStreamInterceptor(fluffycore_middleware_dicontext.StreamServerInterceptor(s.RootContainer)))

	// auth
	log.Info().Msg("adding ChainUnaryInterceptor: fluffycore_middleware_auth_jwt.UnaryServerInterceptor")
	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(fluffycore_middleware_auth_jwt.UnaryServerInterceptor(s.RootContainer)))

	// Here the gating happens
	grpcEntrypointClaimsMap := internal_auth.BuildGrpcEntrypointPermissionsClaimsMap()
	// claims principal
	log.Info().Msg("adding unaryServerInterceptorBuilder: fluffycore_middleware_claimsprincipal.UnaryServerInterceptor")
	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(fluffycore_middleware_claimsprincipal.FinalAuthVerificationMiddlewareUsingClaimsMapWithZeroTrustV2(grpcEntrypointClaimsMap)))

	// last is the recovery middleware
	customFunc := func(p interface{}) (err error) {
		return status.Errorf(codes.Unknown, "panic triggered: %v", p)
	}
	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(customFunc),
	}
	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(grpc_recovery.UnaryServerInterceptor(opts...)))

	return serverOpts
}

func (s *FluffyCoreOTELStartup) OnPreServerStartup(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().Str("method", "OnPreServerStartup").Logger()

	ctxOTEL := log.WithContext(context.Background())
	err := s.OTELContainer.Start(ctxOTEL)
	if err != nil {
		log.Error().Err(err).Msg("failed to Start OTELContainer")
		return err
	}
	return nil
}
func (s *FluffyCoreOTELStartup) OnPreServerShutdown(ctx context.Context) {
	log := zerolog.Ctx(ctx).With().Str("method", "OnPreServerShutdown").Logger()

	log.Info().Msg("OTELContainer stopping")
	s.OTELContainer.Stop(ctx)

}
