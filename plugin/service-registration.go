package plugin

import (
	"sync"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
)

type (
	RegisterService func(builder di.ContainerBuilder)
	NewConfigFunc   func() interface{}

	ConfigOptions struct {
		DefaultConfigJson string
		NewConfig         NewConfigFunc
	}
	ServicePlugin struct {
		registerService RegisterService
		configOptions   *ConfigOptions
	}
	IServicePlugin interface {
		RegisterService(builder di.ContainerBuilder)
		GetConfigOptions() *ConfigOptions
	}
	ServicePluginsContainer struct {
		lock    sync.Mutex
		plugins []IServicePlugin
	}
	IServicePluginsContainer interface {
		GetPlugins() []IServicePlugin
	}
)

var _ IServicePluginsContainer = (*ServicePluginsContainer)(nil)
var _ IServicePlugin = (*ServicePlugin)(nil)

func NewServicePlugin(registerService RegisterService, configOptions *ConfigOptions) IServicePlugin {
	return &ServicePlugin{
		registerService: registerService,
		configOptions:   configOptions,
	}
}

func (s *ServicePlugin) RegisterService(builder di.ContainerBuilder) {
	s.registerService(builder)
}

func (s *ServicePlugin) GetConfigOptions() *ConfigOptions {
	return s.configOptions
}

var ServicePluginsContainerInstance = &ServicePluginsContainer{}

func AddServicePlugin(plugin IServicePlugin) {
	//--~--~--~--~-- BARBED WIRE ~--~--~--~--~--//
	ServicePluginsContainerInstance.lock.Lock()
	defer ServicePluginsContainerInstance.lock.Unlock()
	//--~--~--~--~-- BARBED WIRE ~--~--~--~--~--//
	ServicePluginsContainerInstance.plugins = append(ServicePluginsContainerInstance.plugins, plugin)
}

func (s *ServicePluginsContainer) GetPlugins() []IServicePlugin {
	return s.plugins
}
