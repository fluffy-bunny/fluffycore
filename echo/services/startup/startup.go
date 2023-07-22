package startup

import (
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_startup "github.com/fluffy-bunny/fluffycore/echo/contracts/startup"
	echo "github.com/labstack/echo/v4"
)

type (
	StartupBase struct {
		container di.Container
		hooks     []*contracts_startup.Hooks
		options   *contracts_startup.Options
	}
)

func (s *StartupBase) GetContainer() di.Container {
	return s.container
}
func (s *StartupBase) SetContainer(container di.Container) {
	s.container = container
}

func (s *StartupBase) AddHooks(hooks ...*contracts_startup.Hooks) {
	s.hooks = append(s.hooks, hooks...)
}
func (s *StartupBase) GetHooks() []*contracts_startup.Hooks {
	return s.hooks
}
func (s *StartupBase) GetOptions() *contracts_startup.Options {
	return s.options
}
func (s *StartupBase) SetOptions(options *contracts_startup.Options) {
	s.options = options
}
func (s *StartupBase) RegisterStaticRoutes(e *echo.Echo) error {
	return nil
}

// Configure
func (s *StartupBase) Configure(e *echo.Echo, root di.Container) error {

	return nil
}
