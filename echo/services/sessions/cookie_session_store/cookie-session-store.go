package cookie_session_store

import (
	"net/http"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_sessions "github.com/fluffy-bunny/fluffycore/echo/contracts/sessions"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	status "github.com/gogo/status"
	gorilla_sessions "github.com/gorilla/sessions"
	codes "google.golang.org/grpc/codes"
)

type (
	service struct {
		store    *gorilla_sessions.CookieStore
		config   *contracts_sessions.SessionConfig
		sessions map[string]contracts_sessions.ISession
	}
)

var stemService = (*service)(nil)

func init() {
	var _ contracts_sessions.ICookieSessionStore = (*service)(nil)
}
func validateSessionConfig(config *contracts_sessions.SessionConfig) error {
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

	return nil
}
func (s *service) Ctor(
	config *contracts_sessions.SessionConfig,
) (*service, error) {

	err := validateSessionConfig(config)
	if err != nil {
		return nil, err
	}
	var store = gorilla_sessions.NewCookieStore(
		[]byte(config.AuthenticationKey),
		[]byte(config.EncryptionKey),
	)
	store.Options.Domain = config.Domain
	store.MaxAge(config.MaxAge)
	return &service{
		config:   config,
		store:    store,
		sessions: make(map[string]contracts_sessions.ISession),
	}, nil
}

func AddSingletonCookieSessionStore(b di.ContainerBuilder) {
	di.AddSingleton[contracts_sessions.ICookieSessionStore](b,
		stemService.Ctor,
	)
}

func (s *service) New(r *http.Request, name string) (*gorilla_sessions.Session, error) {
	session, err := s.store.New(r, name)
	if err != nil {
		return nil, err
	}
	return session, nil

}
func (s *service) Get(r *http.Request, name string) (*gorilla_sessions.Session, error) {
	session, err := s.store.Get(r, name)
	if err != nil {
		return nil, err
	}
	return session, nil
}
func (s *service) Save(r *http.Request, w http.ResponseWriter, gs *gorilla_sessions.Session) error {
	return s.store.Save(r, w, gs)
}
func (s *service) MaxAge(age int) {
	s.store.MaxAge(age)
}
