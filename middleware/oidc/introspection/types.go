package introspection

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

type IntrospectionValidationOptions struct {
	// Logger is optional; nil disables logging.
	Logger *logrus.Logger `json:"-"`

	// HTTPClient is optional. nil uses a client with DefaultTimeout -- never a client with
	// no timeout, which would let a hung introspection endpoint block callers forever.
	HTTPClient *http.Client `json:"-"`

	IntrospectionURL string `json:"introspection_url"`

	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`

	Token string `json:"token"`
}

// IntrospectionResponse is the RFC 7662 §2.2 introspection response. Only Active is
// guaranteed to be present; everything else is optional and server-dependent.
type IntrospectionResponse struct {
	Active    bool   `json:"active"`
	Scope     string `json:"scope,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	Username  string `json:"username,omitempty"`
	TokenType string `json:"token_type,omitempty"`
	Exp       int64  `json:"exp,omitempty"`
	Iat       int64  `json:"iat,omitempty"`
	Nbf       int64  `json:"nbf,omitempty"`
	Sub       string `json:"sub,omitempty"`
	// Aud may be a single string or an array of strings.
	Aud any    `json:"aud,omitempty"`
	Iss string `json:"iss,omitempty"`
	Jti string `json:"jti,omitempty"`
}
