// Package sessionutil holds helpers shared by the echo session wrappers.
package sessionutil

import (
	"errors"

	securecookie "github.com/gorilla/securecookie"
)

// IsUnreadableCookie reports whether err from a gorilla session store's Get/New only means
// the request's session cookie could not be decoded: signed with a rotated key, expired, or
// tampered with. The store still returns a fresh session in that case, and callers should
// use it -- the next Save overwrites the bad cookie. Treating it as a hard error locked the
// user out until they cleared their cookies. Other errors (e.g. a backend store being
// unreachable) are not unreadable-cookie errors.
func IsUnreadableCookie(err error) bool {
	var scErr securecookie.Error
	return errors.As(err, &scErr) && scErr.IsDecode()
}
