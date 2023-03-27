package runtime

import (
	"github.com/dozm/di"
	fluffycore_contract_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
)

type (
	// StartupManifest information
	StartupManifest struct {
		Name    string
		Version string
		Port    int
	}
	ConfigOptions struct {
		Destination            interface{}
		RootConfig             []byte
		ConfigPath             string
		ApplicationEnvironment string `json:"applicationEnvironment" mapstructure:"APPLICATION_ENVIRONMENT"`
		PrettyLog              bool   `json:"prettyLog" mapstructure:"PRETTY_LOG"`
		LogLevel               string `json:"logLevel" mapstructure:"LOG_LEVEL"`
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
	GetStartupManifest() StartupManifest
	GetConfigOptions() *ConfigOptions
	ConfigureServices(builder di.ContainerBuilder)
	Configure(unaryServerInterceptorBuilder fluffycore_contract_middleware.IUnaryServerInterceptorBuilder, streamServerInterceptorBuilder fluffycore_contract_middleware.IStreamServerInterceptorBuilder)
	OnPreServerStartup() error
	OnPostServerShutdown()
}
