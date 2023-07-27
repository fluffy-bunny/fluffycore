package handler

import (
	"reflect"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_container "github.com/fluffy-bunny/fluffycore/echo/contracts/container"
	contracts_handler "github.com/fluffy-bunny/fluffycore/echo/contracts/handler"
	wellknown "github.com/fluffy-bunny/fluffycore/echo/wellknown"
	echo "github.com/labstack/echo/v4"
	log "github.com/rs/zerolog/log"
)

type (
	service struct {
		ContainerAccessor contracts_container.ContainerAccessor
	}
)

func init() {
	var _ contracts_handler.IHandlerFactory = (*service)(nil)
}

// AddSingletonIHandlerFactory registers the *service as a singleton.
func AddSingletonIHandlerFactory(builder di.ContainerBuilder) {
	log.Info().Str("DI", "IHandlerFactory").Send()
	di.AddSingleton[contracts_handler.IHandlerFactory](builder, func(
		containerAccessor contracts_container.ContainerAccessor,
	) contracts_handler.IHandlerFactory {
		return &service{
			ContainerAccessor: containerAccessor,
		}
	})
}

func (s *service) RegisterHandlers(app *echo.Group) {
	rootContainer := s.ContainerAccessor()

	scopeFactory := di.Get[di.ScopeFactory](rootContainer)
	scope := scopeFactory.CreateScope()
	scopedContainer := scope.Container()
	descriptors := scopedContainer.GetDescriptors()
	for _, descriptor := range descriptors {
		found := false
		for _, serviceType := range descriptor.ImplementedInterfaceTypes {
			if serviceType == reflect.TypeOf((*contracts_handler.IHandler)(nil)).Elem() {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		metadata := descriptor.Metadata
		path := metadata["path"].(string)
		httpVerbs := metadata["httpVerbs"].([]contracts_handler.HTTPVERB)
		doFunc := func(c echo.Context) error {
			scopedContainer = c.Get(wellknown.SCOPED_CONTAINER_KEY).(di.Container)
			handlerInstance := di.GetByLookupKey[contracts_handler.IHandler](scopedContainer, path)
			return handlerInstance.Do(c)
		}
		for _, httpVerb := range httpVerbs {
			switch httpVerb {
			case contracts_handler.GET:
				app.GET(path, doFunc)
			case contracts_handler.POST:
				app.POST(path, doFunc)
			case contracts_handler.PUT:
				app.PUT(path, doFunc)
			case contracts_handler.DELETE:
				app.DELETE(path, doFunc)
			case contracts_handler.PATCH:
				app.PATCH(path, doFunc)
			case contracts_handler.HEAD:
				app.HEAD(path, doFunc)
			case contracts_handler.OPTIONS:
				app.OPTIONS(path, doFunc)
			case contracts_handler.CONNECT:
				app.CONNECT(path, doFunc)
			case contracts_handler.TRACE:
				app.TRACE(path, doFunc)
			}
			log.Info().Str("echo", "RegisterHandlers").Str("path", path).Send()

		}
	}

}
