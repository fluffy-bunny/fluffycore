package logger

import (
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	echo "github.com/labstack/echo/v4"
	log "github.com/rs/zerolog/log"
)

// EnsureContextLogger ...
func EnsureContextLogger(_ di.Container) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			logger := log.With().Caller().Timestamp().Logger()
			newCtx := logger.WithContext(ctx)
			c.SetRequest(c.Request().WithContext(newCtx))
			return next(c)
		}
	}
}
