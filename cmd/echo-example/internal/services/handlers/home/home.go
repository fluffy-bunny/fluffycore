package healthz

import (
	"net/http"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	wellknown "github.com/fluffy-bunny/fluffycore/cmd/echo-example/internal/wellknown"
	contracts_handler "github.com/fluffy-bunny/fluffycore/echo/contracts/handler"
	"github.com/fluffy-bunny/fluffycore/echo/templates"
	echo "github.com/labstack/echo/v4"
)

type (
	service struct{}
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
		wellknown.HomePath,
	)

}
func ctor() (*service, error) {
	return &service{}, nil
}
func (s *service) GetMiddleware() []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{}
}

func (s *service) Do(c echo.Context) error {
	return templates.Render(c, http.StatusOK, "views/home/index", map[string]interface{}{})

}
