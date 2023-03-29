package main

import (
	"github.com/dozm/di"
	fluffycore_contracts_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	fluffycore_contracts_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	"github.com/fluffy-bunny/fluffycore/example/server/config"
	services_greeter "github.com/fluffy-bunny/fluffycore/example/server/services/greeter"
	"github.com/fluffy-bunny/fluffycore/example/server/services/health"
	version "github.com/fluffy-bunny/fluffycore/example/server/version"
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

func (s *startup) GetStartupManifest() fluffycore_contracts_runtime.StartupManifest {
	return fluffycore_contracts_runtime.StartupManifest{
		Name:    s.config.ApplicationName,
		Version: version.Version(),
		Port:    s.config.GRPCConfig.Port,
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
	health.AddHealthService(builder)
	services_greeter.AddGreeterService(builder)
}
func (s *startup) Configure(unaryServerInterceptorBuilder fluffycore_contracts_middleware.IUnaryServerInterceptorBuilder, streamServerInterceptorBuilder fluffycore_contracts_middleware.IStreamServerInterceptorBuilder) {

}
