package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"syscall"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_async "github.com/fluffy-bunny/fluffycore/async"
	fluffycore_contracts_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	contracts_container "github.com/fluffy-bunny/fluffycore/echo/contracts/container"
	contracts_handler "github.com/fluffy-bunny/fluffycore/echo/contracts/handler"
	echo_contracts_startup "github.com/fluffy-bunny/fluffycore/echo/contracts/startup"
	middleware_container "github.com/fluffy-bunny/fluffycore/echo/middleware/container"
	middleware_logger "github.com/fluffy-bunny/fluffycore/echo/middleware/logger"
	services_contextaccessor "github.com/fluffy-bunny/fluffycore/echo/services/contextaccessor"
	services_cookies_insecure "github.com/fluffy-bunny/fluffycore/echo/services/cookies/insecure"
	services_handler "github.com/fluffy-bunny/fluffycore/echo/services/handler"
	core_echo_templates "github.com/fluffy-bunny/fluffycore/echo/templates"
	"github.com/fluffy-bunny/fluffycore/nats/nats_micro_service"
	fluffycore_runtime "github.com/fluffy-bunny/fluffycore/runtime"
	fluffycore_services_common "github.com/fluffy-bunny/fluffycore/services/common"
	uuid "github.com/google/uuid"
	table "github.com/jedib0t/go-pretty/v6/table"
	echo "github.com/labstack/echo/v5"
	async "github.com/reugn/async"
	zerolog "github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	pkgerrors "github.com/rs/zerolog/pkgerrors"
)

type (
	// Runtime ...
	Runtime struct {
		Startup       echo_contracts_startup.IStartup
		Container     di.Container
		echo          *echo.Echo
		instanceID    string
		configOptions *fluffycore_contracts_runtime.ConfigOptions
		waitChannel   chan os.Signal
		cancel        context.CancelFunc
	}
)

// New creates a new runtime
func New(startup echo_contracts_startup.IStartup) *Runtime {
	return &Runtime{
		Startup:     startup,
		instanceID:  uuid.New().String(),
		waitChannel: make(chan os.Signal, 1),
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

// GetContainer returns the container
func (s *Runtime) GetContainer() di.Container {
	return s.Container
}
func (s *Runtime) phase1() error {
	s.configOptions = s.Startup.GetConfigOptions()
	err := fluffycore_runtime.LoadConfig(s.configOptions)
	if err != nil {
		return err
	}
	prettyLog := false
	prettyLogValue := os.Getenv("PRETTY_LOG")

	if len(prettyLogValue) != 0 {
		b, err := strconv.ParseBool(prettyLogValue)
		if err == nil {
			prettyLog = b
		}
	}
	if prettyLog {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logLevel := os.Getenv("LOG_LEVEL")
	if len(logLevel) == 0 {
		logLevel = "info"
	}
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
	log.Info().Msgf("Starting %s", log.Logger.GetLevel().String())
	return nil
}

func (s *Runtime) phase2() error {
	builder := di.Builder()
	builder.ConfigureOptions(func(o *di.Options) {
		o.ValidateScopes = true
		o.ValidateOnBuild = true
	})
	err := s.addDefaultServices(builder)
	if err != nil {
		log.Error().Err(err).Msg("Failed to add default services")
		return err
	}
	containerAccessor := func() di.Container {
		return s.Container
	}
	di.AddSingleton[contracts_container.ContainerAccessor](builder,
		func() contracts_container.ContainerAccessor {
			return containerAccessor
		})

	err = s.Startup.ConfigureServices(builder)
	if err != nil {
		log.Error().Err(err).Msg("Failed to configure services")
		return err
	}
	for _, hooks := range s.Startup.GetHooks() {
		if hooks.PrebuildHook != nil {
			err = hooks.PrebuildHook(builder)
			if err != nil {
				log.Error().Err(err).Msg("Failed to prebuild hook")
				return err
			}
		}
	}
	s.Container = builder.Build()

	for _, hooks := range s.Startup.GetHooks() {
		if hooks.PostBuildHook != nil {
			err = hooks.PostBuildHook(s.Container)
			if err != nil {
				log.Error().Err(err).Msg("Failed to postbuild hook")
				return err
			}
		}
	}

	s.Startup.SetContainer(s.Container)
	return nil
}
func (s *Runtime) phase3() error {
	s.echo = echo.New()
	//Set Renderer
	s.echo.Renderer = core_echo_templates.GetTemplateRender("./static/templates")

	// MIDDELWARE
	//-------------------------------------------------------
	s.echo.Use(middleware_logger.EnsureContextLogger(s.Container))
	//s.echo.Use(middleware_logger.EnsureContextLoggerCorrelation(s.Container))

	s.echo.Use(middleware_logger.EnsureContextLoggerOTEL(s.Container))

	s.echo.Use(middleware_container.EnsureScopedContainer(s.Container))

	app := s.echo.Group("")

	// we have all our required upfront middleware running
	// now we can add the optional startup ones.
	s.Startup.Configure(s.echo, s.Container)

	// our middleware that runs at the end
	//-------------------------------------------------------
	s.echo.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = echo.ErrInternalServerError
				}
			}()
			return next(c)
		}
	})
	s.Startup.RegisterStaticRoutes(s.echo)

	// register our handlers
	handlerFactory := di.Get[contracts_handler.IHandlerFactory](s.Container)
	handlerFactory.RegisterHandlers(app)
	scopeFactory := di.Get[di.ScopeFactory](s.Container)
	scope := scopeFactory.CreateScope()
	defer scope.Dispose()
	scopedContainer := scope.Container()
	descriptors := scopedContainer.GetDescriptors()

	t := table.NewWriter()
	t.SetTitle("Handler Definitions")
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Verbs", "Path"})

	for _, descriptor := range descriptors {
		found := false
		for _, serviceType := range descriptor.ImplementedInterfaceTypes {
			if serviceType == reflect.TypeOf((*contracts_handler.IHandler)(nil)).Elem() {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		metaData := descriptor.Metadata
		httpVerbs, _ := metaData["httpVerbs"].([]contracts_handler.HTTPVERB)
		verbBldr := strings.Builder{}

		for idx, verb := range httpVerbs {
			verbBldr.WriteString(verb.String())
			if idx < len(httpVerbs)-1 {
				verbBldr.WriteString(",")
			}
		}
		path, _ := metaData["path"].(string)

		t.AppendRow([]interface{}{verbBldr.String(), string(path)})
	}
	t.Render()
	return nil
}
func (s *Runtime) finalPhase() error {
	// Finally start the server
	//----------------------------------------------------------------------------------
	startupOptions := s.Startup.GetOptions()
	if startupOptions == nil {
		err := errors.New("startup options are nil")
		log.Error().Err(err).Msg("Failed to start server")
		return err
	}
	address := fmt.Sprintf(":%v", startupOptions.Port)

	for _, hooks := range s.Startup.GetHooks() {
		if hooks.PreStartHook != nil {
			err := hooks.PreStartHook(s.echo)
			if err != nil {
				log.Error().Err(err).Msg("Failed to prestart hook")
				return err
			}
		}
	}
	future := fluffycore_async.ExecuteWithPromiseAsync(func(promise async.Promise[*fluffycore_async.AsyncResponse]) {
		var err error
		defer func() {
			promise.Success(&fluffycore_async.AsyncResponse{
				Message: "End Serve - echo Server",
				Error:   err,
			})
		}()
		log.Info().Msg("server starting up")
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		sc := echo.StartConfig{Address: address}
		err = sc.Start(ctx, s.echo)
		if err != nil {
			log.Error().Err(err).Msg("failed to start server")
		}
	})
	// wait for an interupt to come in
	s.Wait()

	for _, hooks := range s.Startup.GetHooks() {
		if hooks.PreShutdownHook != nil {
			err := hooks.PreShutdownHook(s.echo)
			if err != nil {
				log.Error().Err(err).Msg("Failed to pre shutdown hook")
			}
		}
	}

	if s.cancel != nil {
		s.cancel()
	}

	response, err := future.Join()
	if err != nil {
		log.Error().Err(err).Msg("Failed to join future")
		return err
	}
	asyncResponse := response
	err = asyncResponse.Error
	fmt.Println(asyncResponse.Message)
	if asyncResponse.Error != nil {
		fmt.Printf("Error: %v\n", err)
	}
	return err
}

// Run ...
func (s *Runtime) Run() error {
	// Phase 1
	// Load config
	// Setup Logger
	err := s.phase1()
	if err != nil {
		log.Fatal().Err(err).Msg("phase1")
	}
	// Phase 2
	// Setup our DI Container
	// Configure services
	err = s.phase2()
	if err != nil {
		log.Fatal().Err(err).Msg("phase2")
	}

	// Phase 3
	// Setup Echo
	// Configure middlewares
	err = s.phase3()
	if err != nil {
		log.Fatal().Err(err).Msg("phase3")
	}

	// Final Phase
	// Setup Echo
	// Configure middlewares
	err = s.finalPhase()
	if err != nil {
		log.Error().Err(err).Msg("finalPhase")
	}

	return err
}

func (s *Runtime) addDefaultServices(builder di.ContainerBuilder) error {
	fluffycore_services_common.AddCommonServices(builder)
	services_contextaccessor.AddScopedIEchoContextAccessor(builder)
	services_handler.AddSingletonIHandlerFactory(builder)
	services_cookies_insecure.AddCookies(builder)
	nats_micro_service.AddCommonNATSServices(builder)
	return nil
}
