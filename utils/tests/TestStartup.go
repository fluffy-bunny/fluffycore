package tests

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_async "github.com/fluffy-bunny/fluffycore/async"
	fluffycore_contracts_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	fluffycore_runtime "github.com/fluffy-bunny/fluffycore/runtime"
	async "github.com/reugn/async"
	bufconn "google.golang.org/grpc/test/bufconn"
)

type (
	ITestStartup interface {
		GetConfigureServicesHook() func(builder di.ContainerBuilder)
		GetSetRootContainerHook() func(container di.Container)
		GetOnPreServerStartup() func(ctx context.Context) error
	}
	TestStartup struct {
		ConfigureServicesHook func(builder di.ContainerBuilder)
		SetRootContainerHook  func(container di.Container)
		OnPreServerStartup    func(ctx context.Context) error
	}
)

func (s *TestStartup) GetOnPreServerStartup() func(ctx context.Context) error {
	return s.OnPreServerStartup
}
func (s *TestStartup) GetSetRootContainerHook() func(container di.Container) {
	return s.SetRootContainerHook
}

func (s *TestStartup) GetConfigureServicesHook() func(builder di.ContainerBuilder) {
	return s.ConfigureServicesHook
}
func NewTestStartup(testStartupHook ITestStartup, innerStartup fluffycore_contracts_runtime.IStartup) fluffycore_contracts_runtime.IStartup {

	config := &TestStartupWrapperConfig{
		InnerStartup:          innerStartup,
		ConfigureServicesHook: testStartupHook.GetConfigureServicesHook(),
		SetRootContainerHook:  testStartupHook.GetSetRootContainerHook(),
		OnPreServerStartup:    testStartupHook.GetOnPreServerStartup(),
	}
	wrapper := NewTestStartupWrapper(config)

	return wrapper
}
func ExecuteWithPromiseAsync(runtime *fluffycore_runtime.Runtime, statup fluffycore_contracts_runtime.IStartup, lis *bufconn.Listener) async.Future[*fluffycore_async.AsyncResponse] {
	future := fluffycore_async.ExecuteWithPromiseAsync(func(promise async.Promise[*fluffycore_async.AsyncResponse]) {
		var err error
		defer func() {
			promise.Success(&fluffycore_async.AsyncResponse{
				Message: "End Serve - grpc Server",
				Error:   err,
			})
		}()
		runtime.StartWithListenter(lis, statup)
	})
	return future
}
