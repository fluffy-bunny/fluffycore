package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_async "github.com/fluffy-bunny/fluffycore/async"
	fluffycore_contracts_ddprofiler "github.com/fluffy-bunny/fluffycore/contracts/ddprofiler"
	fluffycore_contracts_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	fluffycore_contracts_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/contracts/middleware/auth/jwt"
	fluffycore_contracts_otel "github.com/fluffy-bunny/fluffycore/contracts/otel"
	fluffycore_contracts_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	fluffycore_contracts_tasks "github.com/fluffy-bunny/fluffycore/contracts/tasks"
	internal_auth "github.com/fluffy-bunny/fluffycore/example/internal/auth"
	contracts_config "github.com/fluffy-bunny/fluffycore/example/internal/contracts/config"
	services_greeter "github.com/fluffy-bunny/fluffycore/example/internal/services/greeter"
	services_health "github.com/fluffy-bunny/fluffycore/example/internal/services/health"
	services_mystream "github.com/fluffy-bunny/fluffycore/example/internal/services/mystream"
	services_somedisposable "github.com/fluffy-bunny/fluffycore/example/internal/services/somedisposable"
	internal_version "github.com/fluffy-bunny/fluffycore/example/internal/version"
	fluffycore_middleware_auth_jwt "github.com/fluffy-bunny/fluffycore/middleware/auth/jwt"
	fluffycore_middleware_claimsprincipal "github.com/fluffy-bunny/fluffycore/middleware/claimsprincipal"
	fluffycore_middleware_correlation "github.com/fluffy-bunny/fluffycore/middleware/correlation"
	fluffycore_middleware_dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_middleware_logging "github.com/fluffy-bunny/fluffycore/middleware/logging"
	mocks_contracts_oauth2 "github.com/fluffy-bunny/fluffycore/mocks/contracts/oauth2"
	mocks_oauth2_echo "github.com/fluffy-bunny/fluffycore/mocks/oauth2/echo"
	fluffycore_services_ddprofiler "github.com/fluffy-bunny/fluffycore/services/ddprofiler"
	fluffycore_utils_redact "github.com/fluffy-bunny/fluffycore/utils/redact"
	status "github.com/gogo/status"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	madflojo_tasks "github.com/madflojo/tasks"
	async "github.com/reugn/async"
	zerolog "github.com/rs/zerolog"
	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	otel "go.opentelemetry.io/otel"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
)

type (
	startup struct {
		*OTELContainer // embedded  OTEL
		fluffycore_contracts_runtime.UnimplementedStartup
		RootContainer di.Container

		configOptions *fluffycore_contracts_runtime.ConfigOptions
		config        *contracts_config.Config

		mockOAuth2Server       *mocks_oauth2_echo.MockOAuth2Service
		mockOAuth2ServerFuture async.Future[fluffycore_async.AsyncResponse]
		ddProfiler             fluffycore_contracts_ddprofiler.IDataDogProfiler
	}
)

func NewStartup() fluffycore_contracts_runtime.IStartup {
	return &startup{
		OTELContainer: NewOTELContainer(),
	}
}
func (s *startup) SetRootContainer(container di.Container) {
	s.RootContainer = container
}
func (s *startup) GetConfigOptions() *fluffycore_contracts_runtime.ConfigOptions {
	s.config = &contracts_config.Config{}
	s.configOptions = &fluffycore_contracts_runtime.ConfigOptions{
		Destination: s.config,
		RootConfig:  contracts_config.ConfigDefaultJSON,
		EnvPrefix:   "EXAMPLE",
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
	config := s.configOptions.Destination.(*contracts_config.Config)
	if config.OTELConfig == nil {
		config.OTELConfig = &fluffycore_contracts_otel.OTELConfig{}
	}
	s.OTELContainer.Config = config.OTELConfig

	config.OTELConfig.ServiceName = config.ApplicationName

	config.DDConfig.ApplicationEnvironment = config.ApplicationEnvironment
	config.DDConfig.ServiceName = config.ApplicationName
	config.DDConfig.Version = internal_version.Version()
	fluffycore_contracts_ddprofiler.AddDDConfig(builder, config.DDConfig)
	contracts_config.AddConfig(builder, config)

	fluffycore_services_ddprofiler.AddSingletonIProfiler(builder)
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
func (s *startup) ConfigureServerOpts(ctx context.Context) []grpc.ServerOption {
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
func (s *startup) Configure2(ctx context.Context, rootContainer di.Container, unaryServerInterceptorBuilder fluffycore_contracts_middleware.IUnaryServerInterceptorBuilder, streamServerInterceptorBuilder fluffycore_contracts_middleware.IStreamServerInterceptorBuilder) {
	log := zerolog.Ctx(ctx).With().Str("method", "Configure").Logger()

	// puts a zerlog logger into the request context
	log.Info().Msg("adding unaryServerInterceptorBuilder: fluffycore_middleware_logging.EnsureContextLoggingUnaryServerInterceptor")
	unaryServerInterceptorBuilder.Use(fluffycore_middleware_logging.EnsureContextLoggingUnaryServerInterceptor())
	log.Info().Msg("adding streamServerInterceptorBuilder: fluffycore_middleware_logging.EnsureContextLoggingStreamServerInterceptor")
	streamServerInterceptorBuilder.Use(fluffycore_middleware_logging.EnsureContextLoggingStreamServerInterceptor())

	// log correlation and spans
	unaryServerInterceptorBuilder.Use(fluffycore_middleware_correlation.EnsureOTELCorrelationIDUnaryServerInterceptor())
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

type taskTracker struct {
	ID    string
	Count int
}

// OnPreServerStartup ...
func (s *startup) OnPreServerStartup(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().Str("method", "OnPreServerStartup").Logger()

	ctxOTEL := log.WithContext(context.Background())
	err := s.OTELContainer.Start(ctxOTEL)
	if err != nil {
		log.Error().Err(err).Msg("failed to Start OTELContainer")
		return err
	}

	log.Info().Msg("starting up the ISingletonScheduler")
	singletonScheduler := di.Get[fluffycore_contracts_tasks.ISingletonScheduler](s.RootContainer)

	myTaskTracker := &taskTracker{}
	taskID, err := singletonScheduler.Add(&madflojo_tasks.Task{
		Interval: 5 * time.Second,
		TaskFunc: func() error {
			// Put your logic here
			if myTaskTracker.Count > 2 {
				log.Info().Interface("myTaskTracker", myTaskTracker).Msg("I am a task and I am stopping myself")
				singletonScheduler.Del(myTaskTracker.ID)
			} else {
				myTaskTracker.Count++
				log.Info().Interface("myTaskTracker", myTaskTracker).Msg("I am a task and I am running every 5 seconds")
			}
			return nil
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to add task")
		return err
	}
	myTaskTracker.ID = taskID

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
	s.mockOAuth2ServerFuture = fluffycore_async.ExecuteWithPromiseAsync(func(promise async.Promise[fluffycore_async.AsyncResponse]) {
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

	// TODO: is there an OTEL profiler
	s.ddProfiler, err = di.TryGet[fluffycore_contracts_ddprofiler.IDataDogProfiler](s.RootContainer)
	if err == nil {
		s.ddProfiler.Start(ctx)
	}
	return nil
}

// OnPreServerShutdown ...
func (s *startup) OnPreServerShutdown(ctx context.Context) {
	log := zerolog.Ctx(ctx).With().Str("method", "OnPreServerShutdown").Logger()
	log.Info().Msg("mockOAuth2Server shutting down")
	s.mockOAuth2Server.Shutdown(ctx)
	s.mockOAuth2ServerFuture.Join()
	log.Info().Msg("mockOAuth2Server shutdown complete")
	log.Info().Msg("Stopping Datadog Tracer and Profiler")
	s.ddProfiler.Stop(ctx)

	log.Info().Msg("Datadog Tracer and Profiler stopped")
	s.OTELContainer.Stop(ctx)

}
