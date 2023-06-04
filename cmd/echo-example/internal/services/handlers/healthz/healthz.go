package healthz

import (
	"net/http"

	di "github.com/dozm/di"
	wellknown "github.com/fluffy-bunny/fluffycore/cmd/echo-example/internal/wellknown"
	contracts_handler "github.com/fluffy-bunny/fluffycore/echo/contracts/handler"
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
		wellknown.HealthzPath,
	)

}
func ctor() (*service, error) {
	return &service{}, nil
}
func (s *service) GetMiddleware() []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{}
}

// HealthCheck godoc
// @Summary Show the status of server.
// @Description get the status of server.
// @Tags root
// @Accept */*
// @Produce json
// @Success 200 {object} string
// @Router /healthz [get]
func (s *service) Do(c echo.Context) error {
	return c.JSON(http.StatusOK, "ok")
}
