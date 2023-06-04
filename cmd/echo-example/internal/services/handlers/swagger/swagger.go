package swagger

import (
	di "github.com/dozm/di"
	wellknown "github.com/fluffy-bunny/fluffycore/cmd/echo-example/internal/wellknown"
	contracts_handler "github.com/fluffy-bunny/fluffycore/echo/contracts/handler"
	echo "github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type (
	service struct {
		swaggerFunc echo.HandlerFunc
	}
)

func init() {
	var _ contracts_handler.IHandler = (*service)(nil)
}

// AddScopedIHandler registers the *service as a singleton.
func AddScopedIHandler(builder di.ContainerBuilder) {
	contracts_handler.AddScopedIHandleWithMetadata[*service](builder,
		ctor,
		[]contracts_handler.HTTPVERB{
			contracts_handler.GET,
		},
		wellknown.SwaggerPath,
	)

}
func ctor() (*service, error) {
	return &service{
		swaggerFunc: echoSwagger.WrapHandler,
	}, nil
}
func (s *service) GetMiddleware() []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{}
}

func (s *service) Do(c echo.Context) error {
	return s.swaggerFunc(c)
}
