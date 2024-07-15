package runtime

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contract_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	grpc "google.golang.org/grpc"
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
		EnvPrefix   string
	}

	UnimplementedStartup struct {
	}
)

func (UnimplementedStartup) mustEmbedUnimplementedStartup() {}

func (u UnimplementedStartup) GetPreConfigureServerOpts(ctx context.Context) []grpc.ServerOption {
	return []grpc.ServerOption{}
}
func (u UnimplementedStartup) GetPostConfigureServerOpts(ctx context.Context) []grpc.ServerOption {
	return []grpc.ServerOption{}
}

// OnPreServerStartup ...
func (u UnimplementedStartup) OnPreServerStartup(ctx context.Context) error { return nil }

// OnPostServerShutdown ...
func (u UnimplementedStartup) OnPostServerShutdown(ctx context.Context) {}

func (u UnimplementedStartup) OnPreServerShutdown(ctx context.Context) {}

// GetPort ...
func (u UnimplementedStartup) GetPort() int {
	return 0
}
func (u UnimplementedStartup) ConfigureServerOpts(ctx context.Context) []grpc.ServerOption {
	return []grpc.ServerOption{}
}
func (u UnimplementedStartup) Configure(ctx context.Context, rootContainer di.Container,
	unaryServerInterceptorBuilder fluffycore_contract_middleware.IUnaryServerInterceptorBuilder,
	streamServerInterceptorBuilder fluffycore_contract_middleware.IStreamServerInterceptorBuilder) {
}

// IStartup contract
type IStartup interface {
	mustEmbedUnimplementedStartup()
	GetConfigOptions() *ConfigOptions
	// ConfigureService is where you add your objects to the DI container
	ConfigureServices(ctx context.Context, builder di.ContainerBuilder)
	SetRootContainer(container di.Container)
	// ConfigureServerOpts is where you set up your interceptors and tracing.
	ConfigureServerOpts(ctx context.Context) []grpc.ServerOption
	// Deprecated: use ConfigureServerOpts
	Configure(ctx context.Context, rootContainer di.Container,
		unaryServerInterceptorBuilder fluffycore_contract_middleware.IUnaryServerInterceptorBuilder,
		streamServerInterceptorBuilder fluffycore_contract_middleware.IStreamServerInterceptorBuilder)

	OnPreServerStartup(ctx context.Context) error
	OnPreServerShutdown(ctx context.Context)
	OnPostServerShutdown(ctx context.Context)
}
