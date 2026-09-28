package cookies

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/fluffy-bunny/fluffycore/go-app/internal/valuecodec"
	app "github.com/maxence-charriere/go-app/v10/pkg/app"
)

type CookieOptions struct {
	Path     string `json:"path"`
	MaxAge   int    `json:"max_age"` // in seconds, 0 means session cookie, -1 deletes cookie
	Domain   string `json:"domain"`
	Secure   bool   `json:"secure"`
	HttpOnly bool   `json:"http_only"` // Note: cannot be set via JavaScript
	SameSite string `json:"same_site"` // "Strict", "Lax", or "None"
}

// DefaultCookieOptions returns sensible defaults
func DefaultCookieOptions() CookieOptions {
	return CookieOptions{
		Path:     "/",
		MaxAge:   3600, // 1 hour
		SameSite: "Lax",
	}
}

// GetCookie reads a cookie and decodes it into type T
// For structs: expects base64-encoded JSON
// For primitives: reads the value directly
func GetCookie[T any](name string) (T, error) {
	var result T

	// Get all cookies
	cookies := app.Window().Get("document").Get("cookie").String()

	// Find the cookie
	for _, cookie := range strings.Split(cookies, ";") {
		cookie = strings.TrimSpace(cookie)
		if strings.HasPrefix(cookie, name+"=") {
			value := strings.TrimPrefix(cookie, name+"=")

			// Decode based on type
			return decodeCookieValue[T](value)
		}
	}

	return result, fmt.Errorf("cookie %s not found", name)
}

// SetCookie writes a cookie with the given value
// For structs: encodes as base64 JSON
// For primitives: stores the value directly
func SetCookie[T any](name string, value T, options CookieOptions) error {
	// Encode the value
	encodedValue, err := encodeCookieValue(value)
	if err != nil {
		return fmt.Errorf("failed to encode cookie value: %w", err)
	}

	// Build cookie string
	cookieStr := fmt.Sprintf("%s=%s", name, encodedValue)

	if options.Path != "" {
		cookieStr += fmt.Sprintf("; path=%s", options.Path)
	}

	if options.MaxAge != 0 {
		cookieStr += fmt.Sprintf("; max-age=%d", options.MaxAge)
	}

	if options.Domain != "" {
		cookieStr += fmt.Sprintf("; domain=%s", options.Domain)
	}

	if options.Secure {
		cookieStr += "; secure"
	}

	if options.SameSite != "" {
		cookieStr += fmt.Sprintf("; SameSite=%s", options.SameSite)
	}

	// Set the cookie
	app.Window().Get("document").Set("cookie", cookieStr)

	return nil
}

// GetOrSetCookie gets a cookie value, or sets and returns the default value if not found
func GetOrSetCookie[T any](name string, defaultValue T, options CookieOptions) (T, error) {
	// Try to get existing cookie
	value, err := GetCookie[T](name)
	if err == nil {
		return value, nil
	}

	// Cookie doesn't exist, set the default value
	err = SetCookie(name, defaultValue, options)
	if err != nil {
		return defaultValue, fmt.Errorf("failed to set default cookie: %w", err)
	}

	return defaultValue, nil
} // DeleteCookie removes a cookie by setting max-age to -1
func DeleteCookie(name string, options CookieOptions) {
	options.MaxAge = -1
	// Use empty string as value when deleting
	SetCookie(name, "", options)
}

// encodeCookieValue encodes a value for storage in a cookie: primitives as text (strings
// path-escaped so ';', ',' and spaces can't cut the cookie short), anything else as
// base64 JSON. A nil pointer encodes as "".
func encodeCookieValue[T any](value T) (string, error) {
	v, ok := valuecodec.Deref(value)
	if !ok {
		return "", nil
	}
	if s, handled := valuecodec.EncodePrimitive(v); handled {
		if reflect.TypeOf(v).Kind() == reflect.String {
			s = url.PathEscape(s)
		}
		return s, nil
	}
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(jsonBytes), nil
}

// decodeCookieValue decodes a cookie value into type T
func decodeCookieValue[T any](value string) (T, error) {
	var result T
	if value == "" {
		return result, fmt.Errorf("cookie value is empty")
	}
	if rt := reflect.TypeOf(result); rt != nil && elemKind(rt) == reflect.String {
		if unescaped, err := url.PathUnescape(value); err == nil {
			value = unescaped // cookies written before escaping was added have no escapes
		}
	}
	if result, handled, err := valuecodec.DecodePrimitive[T](value); handled {
		return result, err
	}

	// For structs and other complex types: decode base64 + JSON
	decodedBytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return result, fmt.Errorf("failed to decode base64: %w", err)
	}
	target, get := valuecodec.NewTarget[T]()
	if err := json.Unmarshal(decodedBytes, target); err != nil {
		return result, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return get(), nil
}

func elemKind(rt reflect.Type) reflect.Kind {
	if rt.Kind() == reflect.Ptr {
		return rt.Elem().Kind()
	}
	return rt.Kind()
}
