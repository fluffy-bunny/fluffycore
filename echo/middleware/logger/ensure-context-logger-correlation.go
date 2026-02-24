package logger

import (
	"context"

	"github.com/rs/zerolog"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	echo "github.com/labstack/echo/v5"
)

// EnsureContextLoggerCorrelation ...
func EnsureContextLoggerCorrelation(_ di.Container) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			var loggerMap = make(map[string]string)
			headers := c.Request().Header

			// CORRELATION ID
			correlationID := headers.Get(fluffycore_wellknown.XCorrelationIDName)
			if fluffycore_utils.IsEmptyOrNil(correlationID) {
				correlationID = fluffycore_utils.GenerateUniqueID()
			}
			loggerMap["correlation_id"] = correlationID

			// SPANS
			span := headers.Get(fluffycore_wellknown.XSpanName)

			if !fluffycore_utils.IsEmptyOrNil(span) {
				loggerMap[fluffycore_wellknown.LogParentName] = span
				span = fluffycore_utils.GenerateUniqueID()
			}
			// generate a new span for this context
			newSpanID := fluffycore_utils.GenerateUniqueID()
			loggerMap[fluffycore_wellknown.LogSpanName] = newSpanID

			ctx := c.Request().Context()
			// add the correlation id to the context
			ctx = context.
				WithValue(ctx, fluffycore_wellknown.XCorrelationIDName, correlationID)
			ctx = context.
				WithValue(ctx, fluffycore_wellknown.XParentName, span)
			ctx = context.
				WithValue(ctx, fluffycore_wellknown.XSpanName, newSpanID)

			log := zerolog.Ctx(ctx)
			log.UpdateContext(func(c zerolog.Context) zerolog.Context {
				for k, v := range loggerMap {
					c = c.Str(k, v)
				}
				return c
			})

			return next(c)
		}
	}
}
