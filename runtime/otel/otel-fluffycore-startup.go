package otel

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
	fluffycore_contracts_otel "github.com/fluffy-bunny/fluffycore/contracts/otel"
	fluffycore_contracts_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	fluffycore_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/middleware/auth/jwt"
	fluffycore_middleware_auth_mtls "github.com/fluffy-bunny/fluffycore/middleware/auth/mtls"
	fluffycore_middleware_claimsprincipal "github.com/fluffy-bunny/fluffycore/middleware/claimsprincipal"
	fluffycore_middleware_correlation "github.com/fluffy-bunny/fluffycore/middleware/correlation"
	fluffycore_middleware_dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_middleware_logging "github.com/fluffy-bunny/fluffycore/middleware/logging"
	fluffycore_servertls "github.com/fluffy-bunny/fluffycore/runtime/servertls"
	status "github.com/gogo/status"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	zerolog "github.com/rs/zerolog"
	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	otel "go.opentelemetry.io/otel"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	credentials "google.golang.org/grpc/credentials"
)

type (
	FluffyCoreOTELStartup struct {
		*OTELContainer // embedded  OTEL
		fluffycore_contracts_runtime.UnimplementedStartup
		FluffyCoreOTELStartupConfig *FluffyCoreOTELStartupConfig
	}
	FuncAuthGetEntryPointConfigs func() map[string]contracts_common.IEntryPointConfig

	FluffyCoreOTELStartupConfig struct {
		FuncAuthGetEntryPointConfigs FuncAuthGetEntryPointConfigs
	}
)

func NewFluffyCoreOTELStartup(fluffyCoreOTELStartupConfig *FluffyCoreOTELStartupConfig) *FluffyCoreOTELStartup {
	obj := &FluffyCoreOTELStartup{
		FluffyCoreOTELStartupConfig: fluffyCoreOTELStartupConfig,
		OTELContainer:               NewOTELContainer(),
	}

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

	// TLS / mutual TLS -- opt-in via CoreConfig.TLSEnabled. See
	// runtime/servertls and middleware/auth/mtls's README for the full picture:
	// this only ever enables TLS connection-wide (and, when a client CA bundle
	// is configured, requests+verifies a client cert connection-wide); which
	// *methods* actually require a verified client cert is enforced later by
	// FinalAuthVerificationMiddlewareUsingClaimsMapWithZeroTrustV2, exactly like
	// any other permission claim.
	coreConfig, err := di.TryGet[*fluffycore_contracts_config.CoreConfig](s.RootContainer)
	if err == nil && coreConfig != nil {
		tlsConfig, tlsErr := fluffycore_servertls.BuildServerTLSConfig(coreConfig)
		if tlsErr != nil {
			log.Fatal().Err(tlsErr).Msg("failed to build gRPC server TLS config")
		}
		if tlsConfig != nil {
			log.Info().
				Bool("mutualTLS", fluffycore_servertls.IsMutualTLSEnabled(coreConfig)).
				Msg("enabling gRPC server TLS")
			serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsConfig)))
		}
	}

	// log correlation and spans
	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(fluffycore_middleware_correlation.EnsureOTELCorrelationIDUnaryServerInterceptor()))
	// FIRST FIRST: ScopedContextxxx must be called first
	// --------------------------------------------------------------------------------------------------------
	log.Info().Msg("adding ChainUnaryInterceptor: fluffycore_middleware_dicontext.UnaryServerInterceptor")
	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(fluffycore_middleware_dicontext.ScopedContextUnaryServerInterceptor(s.RootContainer)))
	log.Info().Msg("adding ChainStreamInterceptor: fluffycore_middleware_dicontext.StreamServerInterceptor")
	serverOpts = append(serverOpts, grpc.ChainStreamInterceptor(fluffycore_middleware_dicontext.ScopedContextStreamServerInterceptor(s.RootContainer)))

	log.Info().Msg("adding ChainUnaryInterceptor: fluffycore_middleware_logging.EnsureContextLoggingUnaryServerInterceptor")
	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(fluffycore_middleware_logging.EnsureContextLoggingUnaryServerInterceptor()))

	// mTLS claims -- turns a verified client certificate (if any) into claims
	// on the scoped IClaimsPrincipal. Safe to register unconditionally: with no
	// TLS peer info on the connection (TLS disabled, or a plaintext dial) it's
	// a cheap no-op.
	log.Info().Msg("adding ChainUnaryInterceptor: fluffycore_middleware_auth_mtls.UnaryServerInterceptor")
	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(fluffycore_middleware_auth_mtls.UnaryServerInterceptor()))
	log.Info().Msg("adding ChainStreamInterceptor: fluffycore_middleware_auth_mtls.StreamServerInterceptor")
	serverOpts = append(serverOpts, grpc.ChainStreamInterceptor(fluffycore_middleware_auth_mtls.StreamServerInterceptor()))

	// auth
	log.Info().Msg("adding ChainUnaryInterceptor: fluffycore_middleware_auth_jwt.UnaryServerInterceptor")
	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(fluffycore_middleware_auth_jwt.UnaryServerInterceptor(s.RootContainer)))
	log.Info().Msg("adding ChainStreamInterceptor: fluffycore_middleware_logging.EnsureContextLoggingStreamServerInterceptor")
	serverOpts = append(serverOpts, grpc.ChainStreamInterceptor(fluffycore_middleware_logging.EnsureContextLoggingStreamServerInterceptor()))

	// Here the gating happens
	//grpcEntrypointClaimsMap := internal_auth.BuildGrpcEntrypointPermissionsClaimsMap()
	grpcEntrypointClaimsMap := s.FluffyCoreOTELStartupConfig.FuncAuthGetEntryPointConfigs()
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
