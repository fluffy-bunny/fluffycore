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

	// ConfigOptionsV2 is the v2 configuration options.
	// Destination must be a pointer to a pre-populated struct (defaults set in Go code).
	// JSON sources are applied as sparse overlays in order. Environment variables
	// (PREFIX__section__field) are applied last with the highest priority.
	ConfigOptionsV2 struct {
		// Destination is a pointer to the config struct, pre-populated with Go defaults.
		Destination interface{}
		// JSONSources are optional sparse JSON overlays applied in order.
		JSONSources [][]byte
		// ConfigPath for appsettings.{env}.json file merging.
		ConfigPath string
		// EnvPrefix for environment variable filtering (e.g., "MYAPP").
		// Env vars: PREFIX__section__field using __ as the path delimiter.
		EnvPrefix string
	}

	UnimplementedStartup struct {
		RootContainer di.Container
	}
)

func (UnimplementedStartup) mustEmbedUnimplementedStartup() {}
func (s *UnimplementedStartup) SetRootContainer(container di.Container) {
	s.RootContainer = container
}

func (u UnimplementedStartup) GetPreConfigureServerOpts(ctx context.Context) []grpc.ServerOption {
	return []grpc.ServerOption{}
}
func (u UnimplementedStartup) GetPostConfigureServerOpts(ctx context.Context) []grpc.ServerOption {
	return []grpc.ServerOption{}
}

func (s *UnimplementedStartup) GetRootContainer() di.Container {
	return s.RootContainer
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

	GetRootContainer() di.Container

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

// IStartupConfigV2 is an optional interface that IStartup implementations can
// also implement to use the v2 config system. When the runtime detects this
// interface, it calls GetConfigOptionsV2 + LoadConfigV2 instead of GetConfigOptions + LoadConfig.
type IStartupConfigV2 interface {
	GetConfigOptionsV2() *ConfigOptionsV2
}
