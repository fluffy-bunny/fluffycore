package echo

import (
	"github.com/fluffy-bunny/fluffycore/utils"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	"github.com/labstack/echo/v5"
)

// HasWellknownAuthHeaders checks for wellknown auth headers
// 1 - Authorization
// 2 - X-Api-Key
func HasWellknownAuthHeaders(c echo.Context) bool {
	authorizationHeader := c.Request().Header.Get(fluffycore_wellknown.HeaderAuthorization)
	if !utils.IsEmptyOrNil(authorizationHeader) {
		return true
	}
	authorizationHeader = c.Request().Header.Get(fluffycore_wellknown.HeaderXApiKey)
	if !utils.IsEmptyOrNil(authorizationHeader) {
		return true
	}
	return false
}
