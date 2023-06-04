package claimsprincipal

import (
	"net/http"

	di "github.com/dozm/di"
	contracts_core_claimsprincipal "github.com/fluffy-bunny/fluffycore/contracts/common"
	wellknown "github.com/fluffy-bunny/fluffycore/echo/wellknown"
	middleware_claimsprincipal "github.com/fluffy-bunny/fluffycore/middleware/claimsprincipal"
	middleware_oidc "github.com/fluffy-bunny/fluffycore/middleware/oidc"
	core_utils "github.com/fluffy-bunny/fluffycore/utils"
	echo "github.com/labstack/echo/v4"
	log "github.com/rs/zerolog/log"
)

// OnUnauthorizedAction ...
type OnUnauthorizedAction int64

const (
	OnUnauthorizedAction_Unspecified OnUnauthorizedAction = 0
	OnUnauthorizedAction_Redirect                         = 1
)

// EntryPointConfigEx ...
type EntryPointConfigEx struct {
	middleware_oidc.EntryPointConfig
	OnUnauthorizedAction OnUnauthorizedAction
}

// FinalAuthVerificationMiddlewareUsingClaimsMap ...
func FinalAuthVerificationMiddlewareUsingClaimsMap(entrypointClaimsMap map[string]*middleware_oidc.EntryPointConfig, enableZeroTrust bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Path()
			subLogger := log.With().
				Bool("enableZeroTrust", enableZeroTrust).
				Str("FullMethod", path).
				Logger()

			scopedContainer := c.Get(wellknown.SCOPED_CONTAINER_KEY).(di.Container)
			claimsPrincipal := di.Get[contracts_core_claimsprincipal.IClaimsPrincipal](scopedContainer)

			authenticated := claimsPrincipal.HasClaimType(wellknown.ClaimTypeAuthenticated)
			elem, ok := entrypointClaimsMap[path]
			permissionDeniedFunc := func() error {
				if !authenticated {
					if !core_utils.IsNil(elem) {
						directive, ok := elem.MetaData["onUnauthenticated"]
						if ok && directive == "login" {
							return c.Redirect(http.StatusFound, "/login?redirect_url="+c.Request().URL.String())
						}
						return c.String(http.StatusUnauthorized, "Unauthorized")
					}
				}
				return c.Redirect(http.StatusFound, "/unauthorized")
			}
			if !ok && enableZeroTrust {
				subLogger.Debug().Msg("FullMethod not found in entrypoint claims map")
				return permissionDeniedFunc()
			}
			if !middleware_claimsprincipal.Validate(&subLogger, elem.ClaimsConfig, claimsPrincipal) {
				subLogger.Debug().Msg("ClaimsConfig validation failed")
				return permissionDeniedFunc()
			}
			return next(c)
		}
	}
}
