package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	structs "github.com/fatih/structs"
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_async "github.com/fluffy-bunny/fluffycore/async"
	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
	fluffycore_contract_endpoint "github.com/fluffy-bunny/fluffycore/contracts/endpoint"
	fluffycore_contracts_health "github.com/fluffy-bunny/fluffycore/contracts/health"
	fluffycore_contract_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	fluffycore_contracts_tasks "github.com/fluffy-bunny/fluffycore/contracts/tasks"
	services_health "github.com/fluffy-bunny/fluffycore/internal/services/health"
	fluffycore_middleware "github.com/fluffy-bunny/fluffycore/middleware"
	fluffycore_services_common "github.com/fluffy-bunny/fluffycore/services/common"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	viperEx "github.com/fluffy-bunny/viperEx"
	grpc_gateway_runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	async "github.com/reugn/async"
	zerolog "github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	pkgerrors "github.com/rs/zerolog/pkgerrors"
	viper "github.com/spf13/viper"
	grpc "google.golang.org/grpc"
	insecure "google.golang.org/grpc/credentials/insecure"
	grpc_health "google.golang.org/grpc/health/grpc_health_v1"
	keepalive "google.golang.org/grpc/keepalive"
	grpc_reflection "google.golang.org/grpc/reflection"
)

type ServerInstance struct {
	Server *grpc.Server
	Future async.Future[fluffycore_async.AsyncResponse]

	ServerGRPCGatewayMux *http.Server
	FutureGRPCGatewayMux async.Future[fluffycore_async.AsyncResponse]

	Endpoints     []interface{}
	RootContainer di.Container
	logSetupOnce  sync.Once
	Scheduler     fluffycore_contracts_tasks.ISingletonScheduler
}
type Runtime struct {
	ServerInstances *ServerInstance
	waitChannel     chan os.Signal
}

// NewRuntime returns an instance of a new Runtime
func NewRuntime() *Runtime {
	return &Runtime{
		waitChannel:     make(chan os.Signal, 1),
		ServerInstances: &ServerInstance{},
	}
}

// Stop ...
func (s *Runtime) Stop() {
	s.waitChannel <- os.Interrupt
}

// Wait for someone to call stop
func (s *Runtime) Wait() {
	signal.Notify(
		s.waitChannel,
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)
	<-s.waitChannel
}
func (s *Runtime) StartWithListenter(lis net.Listener, startup fluffycore_contract_runtime.IStartup) {
	ctx := context.Background()
	var err error

	// start the pprof web server
	pProfServer := NewPProfServer()
	pProfServer.Start()
	defer func() {
		pProfServer.Stop()
	}()

	// start the go profiler
	pprof := NewPProf()
	pprof.Start()
	defer func() {
		pprof.Stop()
	}()

	control := NewControl(s)
	control.Start()
	defer func() {
		control.Stop()
	}()
	logFormat := fluffycore_utils.StringEnv("LOG_FORMAT", "json")
	logFileName := fluffycore_utils.StringEnv("LOG_FILE", "stderr")

	var logFile *os.File
	// validate log destination
	var target io.Writer
	switch logFileName {
	case "stderr":
		target = os.Stderr
	case "stdout":
		target = os.Stdout
	default:
		// Open the log file

		logFileName = fixPath(logFileName)
		if logFile, err = os.Create(logFileName); err != nil {
			log.Fatal().Err(err).Msg("Creating log file")
		}

		// Pass the ioWriter to the logger
		target = logFile
	}

	logLevel := fluffycore_utils.StringEnv("LOG_LEVEL", "info")
	prettyLog := fluffycore_utils.BoolEnv("PRETTY_LOG", false)

	if prettyLog || logFormat == "pretty" {
		target = zerolog.ConsoleWriter{Out: target}
	}
	log.Logger = log.Output(target)
	ctx = log.Logger.With().Caller().Logger().WithContext(ctx)

	// do once
	// race condition here with zerolog under test
	s.ServerInstances.logSetupOnce.Do(func() {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		switch strings.ToLower(logLevel) {
		case "debug":
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		case "info":
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		case "warn":
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case "error":
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		case "fatal":
			zerolog.SetGlobalLevel(zerolog.FatalLevel)
		case "panic":
			zerolog.SetGlobalLevel(zerolog.PanicLevel)
		case "trace":
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
		}

	})
	builder := di.Builder()
	builder.ConfigureOptions(func(o *di.Options) {
		o.ValidateScopes = true
		o.ValidateOnBuild = true
	})
	// default health service
	services_health.AddHealthService(builder)
	fluffycore_services_common.AddCommonServices(builder)

	configOptions := startup.GetConfigOptions()
	if configOptions == nil {
		panic("configOptions is nil")
	}
	if configOptions.Destination == nil {
		log.Info().Msg("configOptions.Destination is nil, use default")
		configOptions.Destination = &fluffycore_contracts_config.CoreConfig{}
	}
	err = LoadConfig(configOptions)
	if err != nil {
		panic(err)
	}
	// configOptions.Destination contains *fluffycore_contracts_config.CoreConfig
	configBytes, err := json.Marshal(configOptions.Destination)
	if err != nil {
		panic(err)
	}
	var coreConfig *fluffycore_contracts_config.CoreConfig
	err = json.Unmarshal(configBytes, &coreConfig)
	if err != nil {
		panic(err)
	}
	di.AddInstance[*fluffycore_contracts_config.CoreConfig](builder, coreConfig)

	si := &ServerInstance{}
	startup.ConfigureServices(ctx, builder)
	si.RootContainer = builder.Build()
	defer func() {
		// Dispose root
		si.RootContainer.(di.Disposable).Dispose()
	}()
	log.Info().Msg("starting up the ISingletonScheduler")
	si.Scheduler = di.Get[fluffycore_contracts_tasks.ISingletonScheduler](si.RootContainer)
	err = si.Scheduler.Start()
	if err != nil {
		panic(err)
	}
	unaryServerInterceptorBuilder := fluffycore_middleware.NewUnaryServerInterceptorBuilder()
	streamServerInterceptorBuilder := fluffycore_middleware.NewStreamServerInterceptorBuilder()
	startup.SetRootContainer(si.RootContainer)
	var serverOpts []grpc.ServerOption
	serverOpts = append(serverOpts, startup.ConfigureServerOpts(ctx)...)
	startup.Configure(ctx, si.RootContainer, unaryServerInterceptorBuilder, streamServerInterceptorBuilder)

	serverOpts = append(serverOpts, grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle: 5 * time.Minute, // <--- This fixes it!
	}))
	unaryInterceptors := unaryServerInterceptorBuilder.GetUnaryServerInterceptors()
	if len(unaryInterceptors) != 0 {
		serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(unaryInterceptors...))
	}
	streamInterceptors := streamServerInterceptorBuilder.GetStreamServerInterceptors()
	if len(streamInterceptors) != 0 {
		serverOpts = append(serverOpts, grpc.ChainStreamInterceptor(streamInterceptors...))
	}
	grpcServer := grpc.NewServer(
		serverOpts...,
	)
	enableGRPCReflection := fluffycore_utils.BoolEnv("ENABLE_GRPC_SERVER_REFLECTION", false)
	if enableGRPCReflection {
		log.Info().Msg("Enabling GRPC Server Reflection")
		grpc_reflection.Register(grpcServer)
	}
	if grpcServer == nil {
		panic("server is nil")
	}
	endpoints := di.Get[[]fluffycore_contract_endpoint.IEndpointRegistration](si.RootContainer)
	for _, endpoint := range endpoints {
		endpoint.RegisterFluffyCoreGRPCService(grpcServer)
	}

	healthServer := di.Get[fluffycore_contracts_health.IHealthServer](si.RootContainer)
	grpc_health.RegisterHealthServer(grpcServer, healthServer)

	err = startup.OnPreServerStartup(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("OnPreServerStartup failed")
		panic(err)
	}
	if lis == nil {
		if coreConfig.PORT == 0 {
			panic("port is not set")
		}
		lis, err = net.Listen("tcp", fmt.Sprintf(":%d", coreConfig.PORT))
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to listen")
		}
	}

	future := asyncServeGRPC(ctx, grpcServer, lis)
	si.Server = grpcServer
	si.Future = future

	if coreConfig.GRPCGateWayEnabled {
		// Create a client connection to the gRPC server we just started
		// This is where the gRPC-Gateway proxies the requests
		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}

		endpoint := fmt.Sprintf("0.0.0.0:%d", coreConfig.PORT)
		// Create a client connection to the gRPC server we just started
		// This is where the gRPC-Gateway proxies the requests
		conn, err := grpc.NewClient(endpoint, opts...)

		if err != nil {
			log.Fatal().Err(err).Msg("Failed to dial server")
		}
		defer func() {
			if err != nil {
				if cerr := conn.Close(); cerr != nil {
					log.Info().Msgf("Failed to close conn to %s: %v", endpoint, cerr)
				}
				return
			}
			go func() {
				<-ctx.Done()
				if cerr := conn.Close(); cerr != nil {
					log.Info().Msgf("Failed to close conn to %s: %v", endpoint, cerr)
				}
			}()
		}()
		// the framework already is putting in the metadata like authorization when it forwards the request
		// the POST request has
		// --header 'Authorization: Bearer {{token}}
		// which gets put into a grpc metadata header
		// https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/customizing_your_gateway/
		serveMuxOptions := []grpc_gateway_runtime.ServeMuxOption{
			//grpc_gateway_runtime.WithIncomingHeaderMatcher(CustomIncomingHeaderMatcher),
			grpc_gateway_runtime.WithErrorHandler(FluffyGRPCGatewayDefaultHTTPErrorHandler),
			grpc_gateway_runtime.WithHealthzEndpoint(grpc_health.NewHealthClient(conn)),
		}

		gwmux := grpc_gateway_runtime.NewServeMux(
			serveMuxOptions...,
		)
		for _, endpoint := range endpoints {
			endpoint.RegisterFluffyCoreHandler(gwmux, conn)
		}

		gwServer := &http.Server{
			Addr:    fmt.Sprintf(":%d", coreConfig.RESTPort),
			Handler: gwmux,
		}
		si.ServerGRPCGatewayMux = gwServer
		future := asyncServeGRPCGatewayMux(gwServer)
		si.FutureGRPCGatewayMux = future
	}
	s.Wait()
	log.Info().Msg("Interupt triggered")
	startup.OnPreServerShutdown(ctx)
	if si.ServerGRPCGatewayMux != nil {
		si.ServerGRPCGatewayMux.Shutdown(context.Background())
	}
	si.Scheduler.Stop()
	si.Server.GracefulStop()
	startup.OnPostServerShutdown(ctx)
	if si.FutureGRPCGatewayMux != nil {
		si.FutureGRPCGatewayMux.Join()
	}
	si.Future.Join()

}
func FluffyGRPCGatewayDefaultHTTPErrorHandler(ctx context.Context, mux *grpc_gateway_runtime.ServeMux, marshaler grpc_gateway_runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	// we need to set the metadata so we don't get a nonsense log that it doesn't exist
	var metadata grpc_gateway_runtime.ServerMetadata
	ctx = grpc_gateway_runtime.NewServerMetadataContext(ctx, metadata)
	grpc_gateway_runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
}
func LoadConfig(configOptions *fluffycore_contract_runtime.ConfigOptions) error {
	v := viper.NewWithOptions(viper.KeyDelimiter("__"))
	var err error
	v.SetConfigType("json")
	if !fluffycore_utils.IsEmptyOrNil(configOptions.EnvPrefix) {
		v.SetEnvPrefix(configOptions.EnvPrefix)
	}
	// Environment Variables override everything.
	v.AutomaticEnv()

	// 1. Read in as buffer to set a default baseline.
	rootConfigMap := make(map[string]interface{})
	err = json.Unmarshal(configOptions.RootConfig, &rootConfigMap)
	if err != nil {
		log.Error().Err(err).Msg("ConfigDefaultJSON did not unmarshal")
		return err
	}

	if _, ok := rootConfigMap["APPLICATION_ENVIRONMENT"]; !ok {
		rootConfigMap["APPLICATION_ENVIRONMENT"] = "in-enviroment"
	}
	if _, ok := rootConfigMap["APPLICATION_NAME"]; !ok {
		rootConfigMap["APPLICATION_NAME"] = "in-enviroment"
	}
	if _, ok := rootConfigMap["PORT"]; !ok {
		rootConfigMap["PORT"] = 0
	}
	if _, ok := rootConfigMap["GRPC_GATEWAY_ENABLED"]; !ok {
		rootConfigMap["GRPC_GATEWAY_ENABLED"] = true
	}
	if _, ok := rootConfigMap["REST_PORT"]; !ok {
		rootConfigMap["REST_PORT"] = 0
	}
	if _, ok := rootConfigMap["PRETTY_LOG"]; !ok {
		rootConfigMap["PRETTY_LOG"] = true
	}
	if _, ok := rootConfigMap["LOG_LEVEL"]; !ok {
		rootConfigMap["LOG_LEVEL"] = "info"
	}
	if _, ok := rootConfigMap["ddProfilerConfig"]; !ok {
		rootConfigMap["ddProfilerConfig"] = map[string]interface{}{
			"enabled":                false,
			"serviceName":            "in-enviroment",
			"applicationEnvironment": "in-enviroment",
			"version":                "in-enviroment",
		}
	}
	rootConfig, err := json.Marshal(rootConfigMap)
	if err != nil {
		log.Error().Err(err).Msg("ConfigDefaultJSON did not marshal")
		return err
	}
	configOptions.RootConfig = rootConfig
	err = v.ReadConfig(bytes.NewBuffer(configOptions.RootConfig))
	if err != nil {
		log.Err(err).Msg("ConfigDefaultYaml did not read in")
		return err
	}

	environment := os.Getenv("APPLICATION_ENVIRONMENT")

	if len(environment) > 0 && len(configOptions.ConfigPath) != 0 {
		v.AddConfigPath(configOptions.ConfigPath)

		configFile := "appsettings." + environment + ".json"
		configPath := path.Join(configOptions.ConfigPath, configFile)
		err = ValidateConfigPath(configPath)
		if err == nil {
			v.SetConfigFile(configPath)
			err = v.MergeInConfig()
			if err != nil {
				return err
			}
			log.Info().Str("configPath", configPath).Msg("Merging in config")
		} else {
			log.Info().Str("configPath", configPath).Msg("Config file not present")
		}
	}

	// we need to do a viper Unmarshal because that is the only way we get the
	// ENV variables to come in
	err = v.Unmarshal(configOptions.Destination)
	if err != nil {
		return err
	}
	// we do all settings here, becuase a v.AllSettings will NOT bring in the ENV variables
	structs.DefaultTagName = "mapstructure"
	allSettings := structs.Map(configOptions.Destination)

	// normal viper stuff
	myViperEx, err := viperEx.New(allSettings,
		viperEx.WithEnvPrefix(configOptions.EnvPrefix),
		viperEx.WithDelimiter("__"))
	if err != nil {
		return err
	}
	myViperEx.UpdateFromEnv()
	err = myViperEx.Unmarshal(configOptions.Destination)
	return err
}

// ValidateConfigPath just makes sure, that the path provided is a file,
// that can be read
func ValidateConfigPath(configPath string) error {
	s, err := os.Stat(configPath)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", configPath)
	}
	return nil
}
func fixPath(fpath string) string {
	if fpath == "" {
		return ""
	}
	if fpath == "stdout" || fpath == "stderr" {
		return fpath
	}

	// Is it already absolute?
	if filepath.IsAbs(fpath) {
		return filepath.Clean(fpath)
	}

	// Make it absolute
	fpath, _ = filepath.Abs(fpath)

	return fpath
}
func asyncServeGRPC(ctx context.Context, grpcServer *grpc.Server, lis net.Listener) async.Future[fluffycore_async.AsyncResponse] {
	log := zerolog.Ctx(ctx).With().Logger()
	return fluffycore_async.ExecuteWithPromiseAsync(func(promise async.Promise[fluffycore_async.AsyncResponse]) {
		var err error
		log.Info().Msg("gRPC Server Starting up")

		defer func() {
			promise.Success(&fluffycore_async.AsyncResponse{
				Message: "End Serve - grpc Server",
				Error:   err,
			})
			if err != nil {
				log.Error().Err(err).Msg("gRPC Server exit")
				os.Exit(1)
			}
		}()

		if err = grpcServer.Serve(lis); err != nil {
			return
		}
		log.Info().Msg("grpc Server has shut down....")
	})
}
func asyncServeGRPCGatewayMux(httpServer *http.Server) async.Future[fluffycore_async.AsyncResponse] {
	return fluffycore_async.ExecuteWithPromiseAsync(func(promise async.Promise[fluffycore_async.AsyncResponse]) {
		var err error
		log.Info().Msg("gRPC Server Starting up")

		defer func() {
			promise.Success(&fluffycore_async.AsyncResponse{
				Message: "End Serve - http Server",
				Error:   err,
			})
			if err != nil {
				log.Error().Err(err).Msg("gRPC Server exit")
				os.Exit(1)
			}
		}()

		if err = httpServer.ListenAndServe(); err != nil {
			log.Fatal().Err(err).Msg("Failed to listen")
			return
		}
		log.Info().Msg("GRPCGatewayMux Server has shut down....")
	})
}
