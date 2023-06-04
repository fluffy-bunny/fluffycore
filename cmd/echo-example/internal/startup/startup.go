package startup

import (
	di "github.com/dozm/di"
	contracts_config "github.com/fluffy-bunny/fluffycore/cmd/echo-example/internal/contracts/config"
	services_handlers_healthz "github.com/fluffy-bunny/fluffycore/cmd/echo-example/internal/services/handlers/healthz"
	services_handlers_swagger "github.com/fluffy-bunny/fluffycore/cmd/echo-example/internal/services/handlers/swagger"
	services_probe_database "github.com/fluffy-bunny/fluffycore/cmd/echo-example/internal/services/probes/database"
	fluffycore_contracts_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	contracts_startup "github.com/fluffy-bunny/fluffycore/echo/contracts/startup"
	services_startup "github.com/fluffy-bunny/fluffycore/echo/services/startup"
	echo "github.com/labstack/echo/v4"
	log "github.com/rs/zerolog/log"
)

type (
	startup struct {
		services_startup.StartupBase
		config *contracts_config.Config
	}
)

func init() {
	var _ contracts_startup.IStartup = (*startup)(nil)
}

// GetConfigOptions ...
func (s *startup) GetConfigOptions() *fluffycore_contracts_runtime.ConfigOptions {
	return &fluffycore_contracts_runtime.ConfigOptions{
		RootConfig:  []byte(contracts_config.ConfigDefaultJSON),
		Destination: s.config,
	}
}
func NewStartup() contracts_startup.IStartup {
	myStartup := &startup{
		config: &contracts_config.Config{},
	}
	hooks := &contracts_startup.Hooks{
		PostBuildHook:   myStartup.PostBuildHook,
		PreStartHook:    myStartup.PreStartHook,
		PreShutdownHook: myStartup.PreShutdownHook,
	}
	myStartup.AddHooks(hooks)
	return myStartup
}

// ConfigureServices ...
func (s *startup) ConfigureServices(builder di.ContainerBuilder) error {
	s.SetOptions(&contracts_startup.Options{
		Port: s.config.Port,
	})
	services_probe_database.AddSingletonIProbe(builder)
	s.addAppHandlers(builder)
	return nil
}

func (s *startup) PreStartHook(echo *echo.Echo) error {
	log.Info().Msg("PreStartHook")
	return nil
}
func (s *startup) PostBuildHook(container di.Container) error {
	log.Info().Msg("PostBuildHook")
	return nil
}
func (s *startup) PreShutdownHook(echo *echo.Echo) error {
	log.Info().Msg("PreShutdownHook")
	return nil
}
func (s *startup) addAppHandlers(builder di.ContainerBuilder) {
	// add your handlers here
	services_handlers_healthz.AddScopedIHandler(builder)
	services_handlers_swagger.AddScopedIHandler(builder)
}
