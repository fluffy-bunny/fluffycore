package cookies

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type (
	SetCookieRequest struct {
		Name     string                 `json:"name"`
		Value    map[string]interface{} `json:"value"`
		Path     string                 `json:"path"`
		Secure   bool                   `json:"secure"`
		HttpOnly bool                   `json:"httpOnly"`
		Expires  time.Time              `json:"expires"`
		MaxAge   int                    `json:"maxAge"`
		Domain   string                 `json:"domain"`
		SameSite http.SameSite		  `json:"sameSite"`
	}
	SetCookieResponse struct{}

	GetCookieRequest struct {
		Name string `json:"name"`
	}
	GetCookieResponse struct {
		Value map[string]interface{} `json:"value"`
	}
	ICookies interface {
		SetCookie(c echo.Context, request *SetCookieRequest) (*SetCookieResponse, error)
		GetCookie(c echo.Context, name string) (*GetCookieResponse, error)
		DeleteCookie(c echo.Context, name string) error
	}
	SecureCookiesConfig struct {
		HashKey  string `json:"hashKey"`
		BlockKey string `json:"blockKey"`
	}
	ISecureCookies interface {
		ICookies
	}
)
