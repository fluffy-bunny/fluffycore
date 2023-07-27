package contextaccessor

import (
	"reflect"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_contextaccessor "github.com/fluffy-bunny/fluffycore/echo/contracts/contextaccessor"
	echo "github.com/labstack/echo/v4"
)

type (
	service struct {
		context echo.Context
	}
)

func init() {
	var _ contracts_contextaccessor.IInternalEchoContextAccessor = (*service)(nil)
	var _ contracts_contextaccessor.IEchoContextAccessor = (*service)(nil)

}

// AddScopedIEchoContextAccessor registers the *service as a singleton.
func AddScopedIEchoContextAccessor(builder di.ContainerBuilder) {
	di.AddScoped[*service](builder,
		func() *service {
			return &service{}
		},
		reflect.TypeOf((*contracts_contextaccessor.IEchoContextAccessor)(nil)),
		reflect.TypeOf((*contracts_contextaccessor.IInternalEchoContextAccessor)(nil)))

}

func (s *service) SetContext(context echo.Context) {
	s.context = context
}
func (s *service) GetContext() echo.Context {
	return s.context
}
