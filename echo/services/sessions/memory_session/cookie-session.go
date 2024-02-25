package cookie_session

import (
	"reflect"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_contextaccessor "github.com/fluffy-bunny/fluffycore/echo/contracts/contextaccessor"
	contracts_sessions "github.com/fluffy-bunny/fluffycore/echo/contracts/sessions"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	status "github.com/gogo/status"
	gorilla_sessions "github.com/gorilla/sessions"
	zerolog "github.com/rs/zerolog"
	codes "google.golang.org/grpc/codes"
)

type (
	service struct {
		contextAccessor contracts_contextaccessor.IEchoContextAccessor
		session         *gorilla_sessions.Session
		Name            string
		sessionStore    contracts_sessions.IBackendSessionStore
	}
)

var stemService = (*service)(nil)

func init() {
	var _ contracts_sessions.ICookieSession = (*service)(nil)
	var _ contracts_sessions.IInternalCookieSession = (*service)(nil)

}

func (s *service) Ctor(
	sessionStore contracts_sessions.IBackendSessionStore,
) (*service, error) {

	return &service{
		sessionStore: sessionStore,
	}, nil
}

func AddTransientCookieSession(b di.ContainerBuilder) {
	di.AddTransient[*service](b,
		stemService.Ctor,
		reflect.TypeOf((*contracts_sessions.IInternalBackendSession)(nil)),
		reflect.TypeOf((*contracts_sessions.IBackendSession)(nil)),
	)
}
func (s *service) validateInitializeRequest(request *contracts_sessions.InitializeRequest) error {
	if request == nil {
		return status.Error(codes.InvalidArgument, "request is required")
	}
	if request.EchoContextAccessor == nil {
		return status.Error(codes.InvalidArgument, "request.EchoContextAccessor is required")
	}
	if fluffycore_utils.IsEmptyOrNil(request.Name) {
		return status.Error(codes.InvalidArgument, "request.Name is required")
	}
	return nil

}

func (s *service) Initialize(request *contracts_sessions.InitializeRequest) error {
	err := s.validateInitializeRequest(request)
	if err != nil {
		return err
	}
	s.contextAccessor = request.EchoContextAccessor
	s.Name = request.Name

	r := s.contextAccessor.GetContext().Request()
	s.session, err = s.sessionStore.Get(r, s.Name)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) Set(key string, value interface{}) error {
	s.session.Values[key] = value
	return nil
}

func (s *service) Get(key string) (interface{}, error) {
	value, ok := s.session.Values[key]
	if !ok {
		return nil, status.Error(codes.NotFound, "key not found")
	}
	return value, nil
}

func (s *service) Save() error {
	echoContext := s.contextAccessor.GetContext()
	ctx := echoContext.Request().Context()
	log := zerolog.Ctx(ctx).With().Logger()
	err := s.sessionStore.Save(echoContext.Request(), echoContext.Response(), s.session)
	if err != nil {
		log.Err(err).Msg("s.cookieStore.Save")
		return err
	}
	return nil
}

func (s *service) New() error {
	newSession, err := s.sessionStore.New(s.contextAccessor.GetContext().Request(), s.Name)
	if err != nil {
		return err
	}
	s.session = newSession
	return err
}
