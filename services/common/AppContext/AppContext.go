package AppContext

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
)

var _appContext context.Context

func SetAppContext(ctx context.Context) {
	_appContext = ctx
}
func getAppContext() context.Context {
	return _appContext
}

// AddAppContext adds service to the DI container

func AddAppContext(b di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_common.AppContext](b,
		func() fluffycore_contracts_common.AppContext {
			return getAppContext
		})
}
