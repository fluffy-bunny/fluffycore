package logger

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	core_utils "github.com/fluffy-bunny/fluffycore/utils"
	utils "github.com/fluffy-bunny/fluffycore/utils"
	wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	echo "github.com/labstack/echo/v4"
	zerolog "github.com/rs/zerolog"
)

// EnsureContextLoggerCorrelation ...
func EnsureContextLoggerCorrelation(_ di.Container) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var loggerMap = make(map[string]string)
			headers := c.Request().Header

			// CORRELATION ID
			correlationID := headers.Get(wellknown.XCorrelationIDName)
			if core_utils.IsEmptyOrNil(correlationID) {
				correlationID = utils.GenerateUniqueID()
			}
			loggerMap["correlation_id"] = correlationID

			// SPANS
			span := headers.Get(wellknown.XSpanName)

			if !core_utils.IsEmptyOrNil(span) {
				loggerMap[wellknown.LogParentName] = span
				span = utils.GenerateUniqueID()
			}
			// generate a new span for this context
			newSpanID := utils.GenerateUniqueID()
			loggerMap[wellknown.LogSpanName] = newSpanID

			ctx := c.Request().Context()
			// add the correlation id to the context
			ctx = context.
				WithValue(ctx, wellknown.XCorrelationIDName, correlationID)
			ctx = context.
				WithValue(ctx, wellknown.XParentName, span)
			ctx = context.
				WithValue(ctx, wellknown.XSpanName, newSpanID)

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
