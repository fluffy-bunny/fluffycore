package tests

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	grpc "google.golang.org/grpc"
)

type TestStartupWrapperConfig struct {
	InnerStartup          fluffycore_contracts_runtime.IStartup
	ConfigureServicesHook func(builder di.ContainerBuilder)
	SetRootContainerHook  func(container di.Container)
	OnPreServerStartup    func(ctx context.Context) error
}

// TestStartupWrapper struct
type TestStartupWrapper struct {
	fluffycore_contracts_runtime.UnimplementedStartup

	Config *TestStartupWrapperConfig
}

// NewTestStartupWrapper creates a new TestStartupWrapper
func NewTestStartupWrapper(config *TestStartupWrapperConfig) *TestStartupWrapper {
	return &TestStartupWrapper{
		Config: config,
	}
}
func (s *TestStartupWrapper) GetConfigOptions() *fluffycore_contracts_runtime.ConfigOptions {
	return s.Config.InnerStartup.GetConfigOptions()
}

// ConfigureService is where you add your objects to the DI container
func (s *TestStartupWrapper) ConfigureServices(ctx context.Context, builder di.ContainerBuilder) {
	s.Config.InnerStartup.ConfigureServices(ctx, builder)
	if s.Config.ConfigureServicesHook != nil {
		s.Config.ConfigureServicesHook(builder)
	}
}
func (s *TestStartupWrapper) SetRootContainer(container di.Container) {
	s.Config.InnerStartup.SetRootContainer(container)
	if s.Config.SetRootContainerHook != nil {
		s.Config.SetRootContainerHook(container)
	}
}
func (s *TestStartupWrapper) GetRootContainer() di.Container {
	return s.Config.InnerStartup.GetRootContainer()
}

// ConfigureServerOpts is where you set up your interceptors and tracing.
func (s *TestStartupWrapper) ConfigureServerOpts(ctx context.Context) []grpc.ServerOption {
	return s.Config.InnerStartup.ConfigureServerOpts(ctx)
}

func (s *TestStartupWrapper) OnPreServerStartup(ctx context.Context) error {
	err := s.Config.InnerStartup.OnPreServerStartup(ctx)
	if err != nil {
		return err
	}
	if s.Config.OnPreServerStartup != nil {
		err = s.Config.OnPreServerStartup(ctx)
	}
	return err
}
func (s *TestStartupWrapper) OnPreServerShutdown(ctx context.Context) {
	s.Config.InnerStartup.OnPreServerShutdown(ctx)
}
func (s *TestStartupWrapper) OnPostServerShutdown(ctx context.Context) {
	s.Config.InnerStartup.OnPostServerShutdown(ctx)

}
