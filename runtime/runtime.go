package runtime

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/dozm/di"
	"github.com/fatih/structs"
	fluffycore_async "github.com/fluffy-bunny/fluffycore/async"
	fluffycore_contract_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	fluffycore_middleware "github.com/fluffy-bunny/fluffycore/middleware"
	"github.com/fluffy-bunny/viperEx"
	"github.com/reugn/async"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

const bufSize = 1024 * 1024

type ServerInstance struct {
	StartupManifest fluffycore_contract_runtime.StartupManifest
	Server          *grpc.Server
	Future          async.Future[interface{}]
	Endpoints       []interface{}
	RootContainer   di.Container
}
type Runtime struct {
	ServerInstances *ServerInstance
	waitChannel     chan os.Signal
}

// NewRuntime returns an instance of a new Runtime
func NewRuntime() *Runtime {
	return &Runtime{
		waitChannel: make(chan os.Signal),
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
func (s *Runtime) StartWithListenterAndPlugins(lis net.Listener, startup fluffycore_contract_runtime.IStartup) {
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
	logFormat := os.Getenv("LOG_FORMAT")
	if len(logFormat) == 0 {
		logFormat = "json"
	}
	logFileName := os.Getenv("LOG_FILE")
	if len(logFileName) == 0 {
		logFileName = "stderr"
	}
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

	logLevel := os.Getenv("LOG_LEVEL")
	if len(logLevel) == 0 {
		logLevel = "info"
	}
	prettyLog := false
	prettyLogValue := os.Getenv("PRETTY_LOG")
	if len(prettyLogValue) != 0 {
		b, err := strconv.ParseBool(prettyLogValue)
		if err == nil {
			prettyLog = b
		}
	}
	if prettyLog || logFormat == "pretty" {
		target = zerolog.ConsoleWriter{Out: target}
	}
	log.Logger = log.Output(target)

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
	builder := di.Builder()
	configOptions := startup.GetConfigOptions()
	err = LoadConfig(configOptions)
	if err != nil {
		panic(err)
	}
	si := &ServerInstance{}
	si.StartupManifest = startup.GetStartupManifest()
	startup.ConfigureServices(builder)
	si.RootContainer = builder.Build()

	unaryServerInterceptorBuilder := fluffycore_middleware.NewUnaryServerInterceptorBuilder()
	streamServerInterceptorBuilder := fluffycore_middleware.NewStreamServerInterceptorBuilder()
	startup.Configure(unaryServerInterceptorBuilder, streamServerInterceptorBuilder)
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			unaryServerInterceptorBuilder.GetUnaryServerInterceptors()...,
		),
		grpc.ChainStreamInterceptor(
			streamServerInterceptorBuilder.GetStreamServerInterceptors()...,
		),
	)
	if server == nil {
		panic("server is nil")
	}

}
func LoadConfig(configOptions *fluffycore_contract_runtime.ConfigOptions) error {
	v := viper.NewWithOptions(viper.KeyDelimiter("__"))
	var err error
	v.SetConfigType("json")
	// Environment Variables override everything.
	v.AutomaticEnv()

	// 1. Read in as buffer to set a default baseline.
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
	myViperEx, err := viperEx.New(allSettings, func(ve *viperEx.ViperEx) error {
		ve.KeyDelimiter = "__"
		return nil
	})
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
func asyncServeGRPC(grpcServer *grpc.Server, lis net.Listener) async.Future[interface{}] {
	return fluffycore_async.ExecuteWithPromiseAsync(func(promise async.Promise[interface{}]) {
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