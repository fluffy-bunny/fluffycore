package sessions

type (
	ISessionStore interface {
		// creates a new session
		New() error
		Set(key string, value interface{}) error
		Get(key string) (interface{}, error)
		Save() error
	}
	ICookieSessionStore interface {
		ISessionStore
	}
	IMemorySessionStore interface {
		ISessionStore
	}
	IRedisSessionStore interface {
		ISessionStore
	}
	SessionConfig struct {
		SessionName       string
		EncryptionKey     string
		AuthenticationKey string
		Domain            string
	}
)
