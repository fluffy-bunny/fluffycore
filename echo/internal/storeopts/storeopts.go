// Package storeopts applies fluffycore's cookie-hardening defaults to gorilla session stores.
package storeopts

import (
	"net/http"

	contracts_sessions "github.com/fluffy-bunny/fluffycore/echo/contracts/sessions"
	securecookie "github.com/gorilla/securecookie"
	gorilla_sessions "github.com/gorilla/sessions"
)

// DefaultSignedValueMaxAgeSeconds bounds how long a signed session cookie value is accepted
// when SessionConfig.MaxAge is 0 (a browser-session cookie). Without it securecookie treats
// MaxAge 0 as "never expires": a captured cookie stays valid forever.
const DefaultSignedValueMaxAgeSeconds = 86400

// sessionCookieValidity is DefaultSignedValueMaxAgeSeconds; a var so tests can shorten it.
var sessionCookieValidity = DefaultSignedValueMaxAgeSeconds

// Apply sets the store's cookie options from config: HttpOnly always, SameSite (default Lax),
// Secure unless config.Insecure, Domain, and MaxAge (see SetMaxAge).
func Apply(opts *gorilla_sessions.Options, codecs []securecookie.Codec, config *contracts_sessions.SessionConfig) {
	opts.Domain = config.Domain
	opts.HttpOnly = true
	opts.SameSite = config.SameSite
	if opts.SameSite == 0 {
		opts.SameSite = http.SameSiteLaxMode
	}
	opts.Secure = !config.Insecure
	SetMaxAge(opts, codecs, config.MaxAge)
}

// SetMaxAge sets the cookie's Max-Age to age and the signed value's validity to age, or to
// DefaultSignedValueMaxAgeSeconds when age <= 0 -- so a 0 (session-cookie) MaxAge never
// disables server-side expiry.
func SetMaxAge(opts *gorilla_sessions.Options, codecs []securecookie.Codec, age int) {
	opts.MaxAge = age
	validity := age
	if validity <= 0 {
		validity = sessionCookieValidity
	}
	for _, codec := range codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(validity)
		}
	}
}
