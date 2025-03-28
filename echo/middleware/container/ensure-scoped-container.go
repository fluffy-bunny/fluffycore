package container

import (
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_contextaccessor "github.com/fluffy-bunny/fluffycore/echo/contracts/contextaccessor"
	"github.com/fluffy-bunny/fluffycore/echo/wellknown"
	"github.com/labstack/echo/v4"
)

// EnsureScopedContainer ...
func EnsureScopedContainer(root di.Container) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			scopeFactory := di.Get[di.ScopeFactory](root)
			scope := scopeFactory.CreateScope()
			defer scope.Dispose()

			subContainer := scope.Container()
			c.Set(wellknown.SCOPED_CONTAINER_KEY, subContainer)
			defer c.Set(wellknown.SCOPED_CONTAINER_KEY, nil)
			internalContextAccessor := di.Get[contracts_contextaccessor.IInternalEchoContextAccessor](subContainer)
			internalContextAccessor.SetContext(c)

			return next(c)
		}
	}
}
