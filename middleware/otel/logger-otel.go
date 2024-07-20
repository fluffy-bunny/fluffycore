package otel

import (
	"context"
	"strings"

	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	metautils "github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	zerolog "github.com/rs/zerolog"
	otel "go.opentelemetry.io/otel"
	attribute "go.opentelemetry.io/otel/attribute"
	propagation "go.opentelemetry.io/otel/propagation"
	trace "go.opentelemetry.io/otel/trace"
	grpc "google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
)

// version is used as the instrumentation version.
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

// EnsureOpenTelemetryUnaryServerInterceptor ...
func EnsureOpenTelemetryUnaryServerInterceptor(opt ...TraceOption) grpc.UnaryServerInterceptor {
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

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var correlationID string // if not found in header, we generate a new one
		var requestID = "0"
		md := metautils.ExtractIncoming(ctx)
		var loggerMap = make(map[string]string)

		for key, v := range md {
			lowerKey := strings.ToLower(key)
			if lowerKey == fluffycore_wellknown.XCorrelationIDName {
				correlationID = v[0]
			}
			if lowerKey == fluffycore_wellknown.XRequestID {
				requestID = v[0]
			}
		}

		if len(correlationID) == 0 {
			correlationID = fluffycore_utils.GenerateUniqueID()
			md[fluffycore_wellknown.XCorrelationIDName] = []string{correlationID}
		}

		loggerMap["correlation_id"] = correlationID
		// this came into us, so its a parent
		items := md[fluffycore_wellknown.XSpanName]
		if len(items) > 0 {
			loggerMap[fluffycore_wellknown.LogParentName] = items[0]
			md[fluffycore_wellknown.XParentName] = []string{items[0]}
		}
		// generate a new span for this context
		newSpanID := fluffycore_utils.GenerateUniqueID()
		md[fluffycore_wellknown.XSpanName] = []string{newSpanID}
		loggerMap[fluffycore_wellknown.LogSpanName] = newSpanID
		log := zerolog.Ctx(ctx)
		log.UpdateContext(func(c zerolog.Context) zerolog.Context {
			for k, v := range loggerMap {
				c = c.Str(k, v)
			}
			return c
		})
		// Return the cleansed metadata context
		ctx = md.ToIncoming(ctx)

		md2 := metadata.Pairs(
			fluffycore_wellknown.XRequestID, requestID,
			fluffycore_wellknown.XCorrelationIDName, correlationID)
		grpc.SendHeader(ctx, md2)
		return handler(ctx, req)
	}
}

// extract the route name.
func extractRoute(uri string) string {
	return uri[1:]
}

// WithTracer is a TraceOption to inject your own trace.Tracer.
func WithTracer(tracer trace.Tracer) TraceOption {
	return func(c *traceConfig) {
		c.tracer = tracer
	}
}

// WithPropagator is a TraceOption to inject your own propagation.
func WithPropagator(p propagation.TextMapPropagator) TraceOption {
	return func(c *traceConfig) {
		c.propagator = p
	}
}

// WithServiceName is a TraceOption to inject your own serviceName.
func WithServiceName(serviceName string) TraceOption {
	return func(c *traceConfig) {
		c.serviceName = serviceName
	}
}

// WithAttributes is a TraceOption to inject your own attributes.
// Attributes are applied to the trace.Span.
func WithAttributes(attributes ...attribute.KeyValue) TraceOption {
	return func(c *traceConfig) {
		c.attributes = attributes
	}
}
