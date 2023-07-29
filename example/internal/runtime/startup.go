package runtime

import (
	"context"
	"fmt"
	"net/http"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_async "github.com/fluffy-bunny/fluffycore/async"
	fluffycore_contracts_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	fluffycore_contracts_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	contracts_config "github.com/fluffy-bunny/fluffycore/example/internal/contracts/config"
	services_greeter "github.com/fluffy-bunny/fluffycore/example/internal/services/greeter"
	services_health "github.com/fluffy-bunny/fluffycore/example/internal/services/health"
	services_mystream "github.com/fluffy-bunny/fluffycore/example/internal/services/mystream"
	services_somedisposable "github.com/fluffy-bunny/fluffycore/example/internal/services/somedisposable"
	fluffycore_middleware_dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_middleware_logging "github.com/fluffy-bunny/fluffycore/middleware/logging"
	mocks_oauth2_echo "github.com/fluffy-bunny/fluffycore/mocks/oauth2/echo"
	fluffycore_utils_redact "github.com/fluffy-bunny/fluffycore/utils/redact"
	echo "github.com/labstack/echo/v4"
	async "github.com/reugn/async"
	log "github.com/rs/zerolog/log"
)

type (
	startup struct {
		fluffycore_contracts_runtime.UnimplementedStartup
		configOptions          *fluffycore_contracts_runtime.ConfigOptions
		config                 *contracts_config.Config
		mockOAuth2Server       *echo.Echo
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
func (s *startup) ConfigureServices(builder di.ContainerBuilder) {

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
}
func (s *startup) Configure(rootContainer di.Container, unaryServerInterceptorBuilder fluffycore_contracts_middleware.IUnaryServerInterceptorBuilder, streamServerInterceptorBuilder fluffycore_contracts_middleware.IStreamServerInterceptorBuilder) {

	// puts a zerlog logger into the request context
	unaryServerInterceptorBuilder.Use(fluffycore_middleware_logging.EnsureContextLoggingUnaryServerInterceptor())
	streamServerInterceptorBuilder.Use(fluffycore_middleware_logging.EnsureContextLoggingStreamServerInterceptor())

	// dicontext is responsible of create a scoped context for each request.
	unaryServerInterceptorBuilder.Use(fluffycore_middleware_dicontext.UnaryServerInterceptor(rootContainer))
	streamServerInterceptorBuilder.Use(fluffycore_middleware_dicontext.StreamServerInterceptor(rootContainer))

}

// OnPreServerStartup ...
func (s *startup) OnPreServerStartup() error {
	s.mockOAuth2Server = mocks_oauth2_echo.NewOAuth2TestServer()
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
func (s *startup) OnPreServerShutdown() {
	ctx := context.Background()
	log.Info().Msg("mockOAuth2Server shutting down")
	s.mockOAuth2Server.Shutdown(ctx)
	log.Info().Msg("mockOAuth2Server shutdown complete")
	s.mockOAuth2ServerFuture.Join()
	log.Info().Msg("mockOAuth2Server shutdown complete")
}
