package main

import (
	"github.com/dozm/di"
	fluffycore_contracts_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	fluffycore_contracts_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	internal_version "github.com/fluffy-bunny/fluffycore/internal/version"
)

type (
	startup struct {
		fluffycore_contracts_runtime.UnimplementedStartup
	}
)

func NewStartup() fluffycore_contracts_runtime.IStartup {
	return &startup{}
}

func (s *startup) GetStartupManifest() fluffycore_contracts_runtime.StartupManifest {
	return fluffycore_contracts_runtime.StartupManifest{
		Name:    "HelloWorld",
		Version: internal_version.Version(),
		Port:    8080,
	}
}
func (s *startup) GetConfigOptions() *fluffycore_contracts_runtime.ConfigOptions {
	return nil
}
func (s *startup) ConfigureServices(builder di.ContainerBuilder) {

}
func (s *startup) Configure(unaryServerInterceptorBuilder fluffycore_contracts_middleware.IUnaryServerInterceptorBuilder, streamServerInterceptorBuilder fluffycore_contracts_middleware.IStreamServerInterceptorBuilder) {

}
