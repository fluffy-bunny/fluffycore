package cookie_session_store

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
		store           *gorilla_sessions.CookieStore
		config          *contracts_sessions.SessionConfig
		contextAccessor contracts_contextaccessor.IEchoContextAccessor
		session         *gorilla_sessions.Session
	}
)

var stemService = (*service)(nil)

func init() {
	var _ contracts_sessions.ICookieSessionStore = (*service)(nil)
}
func validateCookieSessionStoreConfig(config *contracts_sessions.SessionConfig) error {
	if config == nil {
		return status.Error(codes.InvalidArgument, "config is required")
	}

	if fluffycore_utils.IsEmptyOrNil(config.EncryptionKey) {
		return status.Error(codes.InvalidArgument, "config.EncryptionKey is required")
	}
	if fluffycore_utils.IsEmptyOrNil(config.AuthenticationKey) {
		return status.Error(codes.InvalidArgument, "config.AuthenticationKey is required")
	}
	if fluffycore_utils.IsEmptyOrNil(config.Domain) {
		return status.Error(codes.InvalidArgument, "config.Domain is required")
	}
	if fluffycore_utils.IsEmptyOrNil(config.SessionName) {
		return status.Error(codes.InvalidArgument, "config.SessionName is required")
	}
	return nil
}
func (s *service) Ctor(
	config *contracts_sessions.SessionConfig,
	contextAccessor contracts_contextaccessor.IEchoContextAccessor,
) (*service, error) {

	err := validateCookieSessionStoreConfig(config)
	if err != nil {
		return nil, err
	}
	var store = gorilla_sessions.NewCookieStore(
		[]byte(config.AuthenticationKey),
		[]byte(config.EncryptionKey),
	)
	store.Options.Domain = config.Domain

	echoContext := contextAccessor.GetContext()
	r := echoContext.Request()
	session, err := store.Get(r, config.SessionName)
	if err != nil {
		return nil, err
	}
	if session == nil {
		session, err = store.New(r, config.SessionName)
		if err != nil {
			return nil, err
		}
	}
	return &service{
		config:  config,
		store:   store,
		session: session,
	}, nil
}

func AddScopedCookieSessionStore(b di.ContainerBuilder) {
	di.AddScoped[*service](b,
		stemService.Ctor,
		reflect.TypeOf((*contracts_sessions.ISessionStore)(nil)),
		reflect.TypeOf((*contracts_sessions.ICookieSessionStore)(nil)),
	)
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
	err := s.store.Save(echoContext.Request(), echoContext.Response(), s.session)
	if err != nil {
		log.Err(err).Msg("s.cookieStore.Save")
		return err
	}
	return nil
}

func (s *service) New() error {
	newSession, err := s.store.New(s.contextAccessor.GetContext().Request(), s.config.SessionName)
	if err != nil {
		return err
	}
	s.session = newSession
	return err
}
