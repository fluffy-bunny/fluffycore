package runtime

import (
	"github.com/dozm/di"
	fluffycore_contract_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
)

type (
	// ApplicationManifest information
	ApplicationManifest struct {
		Version string
	}
	ConfigOptions struct {
		Destination interface{}
		RootConfig  []byte
		ConfigPath  string
	}

	UnimplementedStartup struct {
	}
)

func (UnimplementedStartup) mustEmbedUnimplementedStartup() {}

// OnPreServerStartup ...
func (u UnimplementedStartup) OnPreServerStartup() error { return nil }

// OnPostServerShutdown ...
func (u UnimplementedStartup) OnPostServerShutdown() {}

// GetPort ...
func (u UnimplementedStartup) GetPort() int {
	return 0
}

// IStartup contract
type IStartup interface {
	mustEmbedUnimplementedStartup()
	GetConfigOptions() *ConfigOptions
	ConfigureServices(builder di.ContainerBuilder)
	Configure(rootContainer di.Container,
		unaryServerInterceptorBuilder fluffycore_contract_middleware.IUnaryServerInterceptorBuilder,
		streamServerInterceptorBuilder fluffycore_contract_middleware.IStreamServerInterceptorBuilder)
	OnPreServerStartup() error
	OnPostServerShutdown()
}
