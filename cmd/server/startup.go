package main

import (
	"github.com/dozm/di"
	"github.com/fluffy-bunny/fluffycore/cmd/server/services/health"
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

func (s *startup) GetApplicationManifest() fluffycore_contracts_runtime.ApplicationManifest {
	return fluffycore_contracts_runtime.ApplicationManifest{

		Version: internal_version.Version(),
	}
}
func (s *startup) GetConfigOptions() *fluffycore_contracts_runtime.ConfigOptions {
	return nil
}
func (s *startup) ConfigureServices(builder di.ContainerBuilder) {
	health.AddHealthService(builder)
}
func (s *startup) Configure(rootContainer di.Container, unaryServerInterceptorBuilder fluffycore_contracts_middleware.IUnaryServerInterceptorBuilder, streamServerInterceptorBuilder fluffycore_contracts_middleware.IStreamServerInterceptorBuilder) {

}
