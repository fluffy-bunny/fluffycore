package main

import (
	di "github.com/dozm/di"
	fluffycore_contracts_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	fluffycore_contracts_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	config "github.com/fluffy-bunny/fluffycore/example/server/config"
	services_greeter "github.com/fluffy-bunny/fluffycore/example/server/services/greeter"
	services_health "github.com/fluffy-bunny/fluffycore/example/server/services/health"
	version "github.com/fluffy-bunny/fluffycore/example/server/version"
	fluffycore_middleware_dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_middleware_logging "github.com/fluffy-bunny/fluffycore/middleware/logging"
)

type (
	startup struct {
		fluffycore_contracts_runtime.UnimplementedStartup
		configOptions *fluffycore_contracts_runtime.ConfigOptions
		config        *config.Config
	}
)

func NewStartup() fluffycore_contracts_runtime.IStartup {
	return &startup{}
}

func (s *startup) GetApplicationManifest() fluffycore_contracts_runtime.ApplicationManifest {
	return fluffycore_contracts_runtime.ApplicationManifest{
		Version: version.Version(),
	}
}
func (s *startup) GetConfigOptions() *fluffycore_contracts_runtime.ConfigOptions {
	s.config = &config.Config{}
	s.configOptions = &fluffycore_contracts_runtime.ConfigOptions{
		Destination: s.config,
		RootConfig:  config.ConfigDefaultJSON,
	}
	return s.configOptions
}
func (s *startup) ConfigureServices(builder di.ContainerBuilder) {
	di.AddSingleton[*config.Config](builder, func() *config.Config {
		return s.configOptions.Destination.(*config.Config)
	})
	services_health.AddHealthService(builder)
	services_greeter.AddGreeterService(builder)
}
func (s *startup) Configure(rootContainer di.Container, unaryServerInterceptorBuilder fluffycore_contracts_middleware.IUnaryServerInterceptorBuilder, streamServerInterceptorBuilder fluffycore_contracts_middleware.IStreamServerInterceptorBuilder) {

	// puts a zerlog logger into the request context
	unaryServerInterceptorBuilder.Use(fluffycore_middleware_logging.EnsureContextLoggingUnaryServerInterceptor())

	// dicontext is responsible of create a scoped context for each request.
	unaryServerInterceptorBuilder.Use(fluffycore_middleware_dicontext.UnaryServerInterceptor(rootContainer))

}
