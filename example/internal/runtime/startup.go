package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_async "github.com/fluffy-bunny/fluffycore/async"
	fluffycore_contracts_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	fluffycore_contracts_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/contracts/middleware/auth/jwt"
	fluffycore_contracts_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	internal_auth "github.com/fluffy-bunny/fluffycore/example/internal/auth"
	contracts_config "github.com/fluffy-bunny/fluffycore/example/internal/contracts/config"
	services_greeter "github.com/fluffy-bunny/fluffycore/example/internal/services/greeter"
	services_health "github.com/fluffy-bunny/fluffycore/example/internal/services/health"
	services_mystream "github.com/fluffy-bunny/fluffycore/example/internal/services/mystream"
	services_somedisposable "github.com/fluffy-bunny/fluffycore/example/internal/services/somedisposable"
	fluffycore_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/middleware/auth/jwt"
	fluffycore_middleware_claimsprincipal "github.com/fluffy-bunny/fluffycore/middleware/claimsprincipal"
	fluffycore_middleware_dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_middleware_logging "github.com/fluffy-bunny/fluffycore/middleware/logging"
	mocks_contracts_oauth2 "github.com/fluffy-bunny/fluffycore/mocks/contracts/oauth2"
	mocks_oauth2_echo "github.com/fluffy-bunny/fluffycore/mocks/oauth2/echo"
	fluffycore_utils_redact "github.com/fluffy-bunny/fluffycore/utils/redact"
	"github.com/gogo/status"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	async "github.com/reugn/async"
	zerolog "github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
)

type (
	startup struct {
		fluffycore_contracts_runtime.UnimplementedStartup
		configOptions *fluffycore_contracts_runtime.ConfigOptions
		config        *contracts_config.Config

		mockOAuth2Server       *mocks_oauth2_echo.MockOAuth2Service
		mockOAuth2ServerFuture async.Future[interface{}]
	}
)

func NewStartup() fluffycore_contracts_runtime.IStartup {
	return &startup{}
}

func (s *startup) GetConfigOptions() *fluffycore_contracts_runtime.ConfigOptions {
	s.config = &contracts_config.Config{}
	s.configOptions = &fluffycore_contracts_runtime.ConfigOptions{
		Destination: s.config,
		RootConfig:  contracts_config.ConfigDefaultJSON,
	}
	return s.configOptions
}
func (s *startup) ConfigureServices(ctx context.Context, builder di.ContainerBuilder) {
	log := zerolog.Ctx(ctx).With().Str("method", "Configure").Logger()
	dst, err := fluffycore_utils_redact.CloneAndRedact(s.configOptions.Destination)
	if err != nil {
		panic(err)
	}
	log.Info().Interface("config", dst).Msg("config")
	di.AddSingleton[*contracts_config.Config](builder, func() *contracts_config.Config {
		return s.configOptions.Destination.(*contracts_config.Config)
	})
	services_health.AddHealthService(builder)
	services_greeter.AddGreeterService(builder)
	services_somedisposable.AddScopedSomeDisposable(builder)
	services_mystream.AddMyStreamService(builder)
	issuerConfigs := &fluffycore_contracts_middleware_auth_jwt.IssuerConfigs{}
	for idx := range s.config.JWTValidators.Issuers {
		issuerConfigs.IssuerConfigs = append(issuerConfigs.IssuerConfigs,
			&fluffycore_contracts_middleware_auth_jwt.IssuerConfig{
				OAuth2Config: &fluffycore_contracts_middleware_auth_jwt.OAuth2Config{
					Issuer:  s.config.JWTValidators.Issuers[idx],
					JWKSUrl: s.config.JWTValidators.JWKSURLS[idx],
				},
			})
	}
	fluffycore_middleware_auth_jwt.AddValidators(builder, issuerConfigs)
}
func (s *startup) Configure(ctx context.Context, rootContainer di.Container, unaryServerInterceptorBuilder fluffycore_contracts_middleware.IUnaryServerInterceptorBuilder, streamServerInterceptorBuilder fluffycore_contracts_middleware.IStreamServerInterceptorBuilder) {
	log := zerolog.Ctx(ctx).With().Str("method", "Configure").Logger()

	// puts a zerlog logger into the request context
	log.Info().Msg("adding unaryServerInterceptorBuilder: fluffycore_middleware_logging.EnsureContextLoggingUnaryServerInterceptor")
	unaryServerInterceptorBuilder.Use(fluffycore_middleware_logging.EnsureContextLoggingUnaryServerInterceptor())
	log.Info().Msg("adding streamServerInterceptorBuilder: fluffycore_middleware_logging.EnsureContextLoggingStreamServerInterceptor")
	streamServerInterceptorBuilder.Use(fluffycore_middleware_logging.EnsureContextLoggingStreamServerInterceptor())

	// dicontext is responsible of create a scoped context for each request.
	log.Info().Msg("adding unaryServerInterceptorBuilder: fluffycore_middleware_dicontext.UnaryServerInterceptor")
	unaryServerInterceptorBuilder.Use(fluffycore_middleware_dicontext.UnaryServerInterceptor(rootContainer))
	log.Info().Msg("adding streamServerInterceptorBuilder: fluffycore_middleware_dicontext.StreamServerInterceptor")
	streamServerInterceptorBuilder.Use(fluffycore_middleware_dicontext.StreamServerInterceptor(rootContainer))

	// auth
	log.Info().Msg("adding unaryServerInterceptorBuilder: fluffycore_middleware_auth_jwt.UnaryServerInterceptor")
	unaryServerInterceptorBuilder.Use(fluffycore_middleware_auth_jwt.UnaryServerInterceptor(rootContainer))

	// Here the gating happens
	grpcEntrypointClaimsMap := internal_auth.BuildGrpcEntrypointPermissionsClaimsMap()
	// claims principal
	log.Info().Msg("adding unaryServerInterceptorBuilder: fluffycore_middleware_claimsprincipal.UnaryServerInterceptor")
	unaryServerInterceptorBuilder.Use(fluffycore_middleware_claimsprincipal.FinalAuthVerificationMiddlewareUsingClaimsMapWithZeroTrustV2(grpcEntrypointClaimsMap))

	// last is the recovery middleware
	customFunc := func(p interface{}) (err error) {
		return status.Errorf(codes.Unknown, "panic triggered: %v", p)
	}
	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(customFunc),
	}
	unaryServerInterceptorBuilder.Use(grpc_recovery.UnaryServerInterceptor(opts...))

}

// OnPreServerStartup ...
func (s *startup) OnPreServerStartup(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().Str("method", "OnPreServerStartup").Logger()

	clientsJSON, err := os.ReadFile(s.config.ConfigFiles.ClientPath)
	var clients []mocks_contracts_oauth2.Client
	if err != nil {
		return err
	}
	err = json.Unmarshal(clientsJSON, &clients)
	if err != nil {
		return err
	}

	log.Info().Interface("clients", clients).Msg("clients")
	s.mockOAuth2Server = mocks_oauth2_echo.NewOAuth2TestServer(&mocks_contracts_oauth2.MockOAuth2Config{
		Clients: clients,
	})
	s.mockOAuth2ServerFuture = fluffycore_async.ExecuteWithPromiseAsync(func(promise async.Promise[interface{}]) {
		var err error
		defer func() {
			promise.Success(&fluffycore_async.AsyncResponse{
				Message: "End Serve - mockOAuth2Server",
				Error:   err,
			})
		}()
		log.Info().Msg("mockOAuth2Server starting up")
		err = s.mockOAuth2Server.Start(fmt.Sprintf(":%d", s.config.OAuth2Port))
		if err != nil && http.ErrServerClosed == err {
			err = nil
		}
		if err != nil {
			log.Error().Err(err).Msg("failed to start server")
		}
	})

	return nil
}

// OnPreServerShutdown ...
func (s *startup) OnPreServerShutdown(ctx context.Context) {
	log := zerolog.Ctx(ctx).With().Str("method", "OnPreServerShutdown").Logger()
	log.Info().Msg("mockOAuth2Server shutting down")
	s.mockOAuth2Server.Shutdown(ctx)
	log.Info().Msg("mockOAuth2Server shutdown complete")
	s.mockOAuth2ServerFuture.Join()
	log.Info().Msg("mockOAuth2Server shutdown complete")
}
