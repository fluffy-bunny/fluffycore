package myplugin

import (
	"encoding/json"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_myplugin "github.com/fluffy-bunny/fluffycore/example/internal/contracts/myplugin"
	fluffycore_plugin "github.com/fluffy-bunny/fluffycore/plugin"
)

type (
	service struct{}
)

var _ fluffycore_contracts_myplugin.IMyPlugin = &service{}

var stemService = (*service)(nil)

func (s *service) Ctor() (fluffycore_contracts_myplugin.IMyPlugin, error) {
	return &service{}, nil
}
func AddSingletonMyPlugin(cb di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_myplugin.IMyPlugin](cb, stemService.Ctor)
}

func (s *service) DoSomething() (string, error) {
	return "Hello World", nil
}

func NewServicePlugin() fluffycore_plugin.IServicePlugin {
	config := &fluffycore_contracts_myplugin.Config{}
	jsonB, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}
	return NewServicePluginWithDefaultConfig(string(jsonB))
}

func NewServicePluginWithDefaultConfig(defaultConfigJson string) fluffycore_plugin.IServicePlugin {
	// this is the config that will be filled out at runtime
	config := &fluffycore_contracts_myplugin.Config{}
	return fluffycore_plugin.NewServicePlugin(func(builder di.ContainerBuilder) {
		AddSingletonMyPlugin(builder)
		// add the filled out config to the container
		di.AddInstance[*fluffycore_contracts_myplugin.Config](builder, config)
	}, &fluffycore_plugin.ConfigOptions{
		DefaultConfigJson: defaultConfigJson,
		NewConfig: func() interface{} {
			return config
		},
	})
}
