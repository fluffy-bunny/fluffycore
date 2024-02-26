package sessions

import (
	"net/http"

	contracts_contextaccessor "github.com/fluffy-bunny/fluffycore/echo/contracts/contextaccessor"
	gorilla_sessions "github.com/gorilla/sessions"
)

type (
	ISession interface {
		// creates a new session
		New() error
		Set(key string, value interface{}) error
		Get(key string) (interface{}, error)
		Save() error
	}
	IInternalSession interface {
		Initialize(request *InitializeRequest) error
	}
	ICookieSession interface {
		ISession
	}
	IBackendSession interface {
		ISession
	}
	IInternalCookieSession interface {
		IInternalSession
		ICookieSession
	}
	IInternalBackendSession interface {
		IInternalSession
		IBackendSession
	}
	ISessionStore interface {
		New(r *http.Request, name string) (*gorilla_sessions.Session, error)
		Get(r *http.Request, name string) (*gorilla_sessions.Session, error)
		Save(r *http.Request, w http.ResponseWriter, s *gorilla_sessions.Session) error
		MaxAge(age int)
	}
	ISessionFactory interface {
		GetCookieSession(name string) (ISession, error)
		GetBackendSession(name string) (ISession, error)
	}
	InitializeRequest struct {
		Name                string
		EchoContextAccessor contracts_contextaccessor.IEchoContextAccessor
	}

	ICookieSessionStore interface {
		ISessionStore
	}
	IBackendSessionStore interface {
		ISessionStore
	}

	SessionConfig struct {
		EncryptionKey     string `json:"encryptionKey"`
		AuthenticationKey string `json:"authenticationKey"`
		Domain            string `json:"domain"`
		MaxAge            int    `json:"maxAge"`
	}
)
