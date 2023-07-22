package container

import (
	di "github.com/dozm/di"
	contracts_contextaccessor "github.com/fluffy-bunny/fluffycore/echo/contracts/contextaccessor"
	wellknown "github.com/fluffy-bunny/fluffycore/echo/wellknown"
	echo "github.com/labstack/echo/v4"
)

// EnsureScopedContainer ...
func EnsureScopedContainer(root di.Container) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			scopeFactory := di.Get[di.ScopeFactory](root)
			scope := scopeFactory.CreateScope()
			subContainer := scope.Container()

			c.Set(wellknown.SCOPED_CONTAINER_KEY, subContainer)
			internalContextAccessor := di.Get[contracts_contextaccessor.IInternalEchoContextAccessor](subContainer)
			internalContextAccessor.SetContext(c)

			return next(c)
		}
	}
}
