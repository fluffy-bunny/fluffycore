package session_factory

import (
	"sync"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_contextaccessor "github.com/fluffy-bunny/fluffycore/echo/contracts/contextaccessor"
	contracts_sessions "github.com/fluffy-bunny/fluffycore/echo/contracts/sessions"
)

type (
	service struct {
		container           di.Container
		cookieSessionStore  contracts_sessions.ICookieSessionStore
		backendSessionStore contracts_sessions.IBackendSessionStore
		contextAccessor     contracts_contextaccessor.IEchoContextAccessor

		mtx             sync.Mutex
		cookieSessions  map[string]contracts_sessions.ISession
		backendSessions map[string]contracts_sessions.ISession
	}
)

var stemService = (*service)(nil)

func init() {
	var _ contracts_sessions.ISessionFactory = (*service)(nil)
}

func (s *service) Ctor(
	container di.Container,
	cookieSessionStore contracts_sessions.ICookieSessionStore,
	contextAccessor contracts_contextaccessor.IEchoContextAccessor,
	backendSessionStore contracts_sessions.IBackendSessionStore,
) (*service, error) {

	return &service{
		container:           container,
		cookieSessionStore:  cookieSessionStore,
		contextAccessor:     contextAccessor,
		backendSessionStore: backendSessionStore,
		cookieSessions:      make(map[string]contracts_sessions.ISession),
		backendSessions:     make(map[string]contracts_sessions.ISession),
	}, nil
}

func AddScopedSessionFactory(b di.ContainerBuilder) {
	di.AddScoped[contracts_sessions.ISessionFactory](b,
		stemService.Ctor,
	)
}

func (s *service) GetCookieSession(name string) (contracts_sessions.ISession, error) {
	//--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--
	s.mtx.Lock()
	defer s.mtx.Unlock()
	//--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--
	dd, ok := s.cookieSessions[name]
	if ok {
		return dd, nil
	}

	obj := di.Get[contracts_sessions.IInternalCookieSession](s.container)
	obj.Initialize(&contracts_sessions.InitializeRequest{
		Name:                name,
		EchoContextAccessor: s.contextAccessor,
	})
	s.cookieSessions[name] = obj
	return obj, nil
}
func (s *service) GetBackendSession(name string) (contracts_sessions.ISession, error) {
	//--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--
	s.mtx.Lock()
	defer s.mtx.Unlock()
	//--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--~--
	dd, ok := s.backendSessions[name]
	if ok {
		return dd, nil
	}

	obj := di.Get[contracts_sessions.IInternalBackendSession](s.container)
	obj.Initialize(&contracts_sessions.InitializeRequest{
		Name:                name,
		EchoContextAccessor: s.contextAccessor,
	})
	s.backendSessions[name] = obj
	return obj, nil
}
