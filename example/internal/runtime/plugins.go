package runtime

import (
	fluffycore_services_myplugin "github.com/fluffy-bunny/fluffycore/example/internal/services/myplugin"
	fluffycore_plugin "github.com/fluffy-bunny/fluffycore/plugin"
)

func init() {

	plugin := fluffycore_services_myplugin.NewServicePlugin()
	fluffycore_plugin.AddServicePlugin(plugin)

}
