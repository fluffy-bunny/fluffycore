package logger

import (
	"context"
	"strings"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	echo "github.com/labstack/echo/v4"
	zerolog "github.com/rs/zerolog"
	otel "go.opentelemetry.io/otel"
	attribute "go.opentelemetry.io/otel/attribute"
	propagation "go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	trace "go.opentelemetry.io/otel/trace"
)

const version = "0.0.8"

// TraceOption takes a traceConfig struct and applies changes.
// It can be passed to the TraceWithOptions function to configure a traceConfig struct.
type TraceOption func(*traceConfig)

// traceConfig contains all the configuration for the library.
type traceConfig struct {
	serviceName string
	tracer      trace.Tracer
	propagator  propagation.TextMapPropagator
	attributes  []attribute.KeyValue
}

func Proto(proto string) string {
	switch proto {
	case "HTTP/1.0":
		return "1.0"
	case "HTTP/1.1":
		return "1.1"
	case "HTTP/2":
		return "2"
	case "HTTP/3":
		return "3"
	default:
		return proto
	}
}

// EnsureContextLoggerCorrelation ...
func EnsureContextLoggerOTEL(_ di.Container, opt ...TraceOption) echo.MiddlewareFunc {
	// initialize an empty traceConfig.
	config := &traceConfig{}

	// apply the configuration passed to the function.
	for _, o := range opt {
		o(config)
	}
	// check for the traceConfig.tracer if absent use a default value.
	if config.tracer == nil {
		config.tracer = otel.Tracer("otel-tracer", trace.WithInstrumentationVersion(version))
	}
	// check for the traceConfig.propagator if absent use a default value.
	if config.propagator == nil {
		config.propagator = otel.GetTextMapPropagator()
	}
	// check for the traceConfig.serviceName if absent use a default value.
	if config.serviceName == "" {
		config.serviceName = "TracedApplication"
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := c.Request()
			requestCtx := r.Context()
			// extract the OpenTelemetry span context from the context.Context object.
			ctx := config.propagator.Extract(requestCtx, propagation.HeaderCarrier(r.Header))
			traceS := ""
			traceID, err := trace.TraceIDFromHex(r.Header.Get("x-b3-traceid"))
			if err == nil {
				traceS = traceID.String()
			}

			// the standard trace.SpanStartOption options whom are applied to every server handler.
			opts := []trace.SpanStartOption{
				trace.WithAttributes(semconv.ServiceNameKey.String(config.serviceName)),
				trace.WithAttributes(semconv.HTTPRequestMethodKey.String(r.Method)),
				trace.WithAttributes(semconv.NetworkProtocolName(strings.ToLower(r.Proto))),
				trace.WithAttributes(semconv.NetworkProtocolVersion(Proto(r.Proto))),
				trace.WithAttributes(semconv.ServerAddress(r.Host)),
				trace.WithAttributes(semconv.TelemetrySDKLanguageGo),
				trace.WithSpanKind(trace.SpanKindClient),
			}
			// check for the traceConfig.attributes if present apply them to the trace.Span.
			if len(config.attributes) > 0 {
				opts = append(opts, trace.WithAttributes(config.attributes...))
			}
			// extract the route name which is used for setting a usable name of the span.
			spanName := extractRoute(r.RequestURI)
			if spanName == "" {
				// no path available
				spanName = "HTTP " + r.Method + " /"
			}

			// create a good name to recognize where the span originated.
			spanName = r.Method + " /" + spanName

			// start the actual trace.Span.
			ctx, span := config.tracer.Start(ctx, spanName, opts...)

			defer span.End()

			// pass the span through the request context.
			r = r.WithContext(ctx)
			carrier := propagation.HeaderCarrier(r.Header)
			otel.GetTextMapPropagator().Inject(ctx, carrier)

			// old
			var loggerMap = make(map[string]string)
			headers := c.Request().Header

			// CORRELATION ID
			correlationID := headers.Get(wellknown.XCorrelationIDName)
			if fluffycore_utils.IsEmptyOrNil(correlationID) {
				correlationID = fluffycore_utils.GenerateUniqueID()
			}
			loggerMap["correlation_id"] = correlationID
			if fluffycore_utils.IsNotEmptyOrNil(traceS) {
				loggerMap["trace_id"] = traceS
			}

			// add the correlation id to the context
			ctx = context.
				WithValue(ctx, wellknown.XCorrelationIDName, correlationID)

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

// extract the route name.
func extractRoute(uri string) string {
	return uri[1:]
}
