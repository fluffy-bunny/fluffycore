package correlation

import (
	"context"
	"strings"

	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	metautils "github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	zerolog "github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	otel_trace "go.opentelemetry.io/otel/trace"
	grpc "google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
)

// EnsureCorrelationIDUnaryServerInterceptor ...
func EnsureCorrelationIDUnaryServerInterceptor() grpc.UnaryServerInterceptor {
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

		loggerMap[fluffycore_wellknown.LogCorrelationIDName] = correlationID
		// this came into us, so its a parent
		items := md[fluffycore_wellknown.XSpanName]
		if items != nil && len(items) > 0 {
			loggerMap[fluffycore_wellknown.LogParentName] = items[0]
			md[fluffycore_wellknown.XParentName] = []string{items[0]}
		}
		// generate a new span for this context
		newSpanID := fluffycore_utils.GenerateUniqueID()
		md[fluffycore_wellknown.XSpanName] = []string{newSpanID}
		loggerMap[fluffycore_wellknown.LogSpanName] = newSpanID
		// Return the cleansed metadata context
		ctx = withLoggerFields(md.ToIncoming(ctx), loggerMap)

		md2 := metadata.Pairs(
			fluffycore_wellknown.XRequestID, requestID,
			fluffycore_wellknown.XCorrelationIDName, correlationID)
		// SetHeader, not SendHeader: SendHeader flushes the headers now, after which the
		// handler's own grpc.SetHeader calls fail. SetHeader merges and sends with the response.
		_ = grpc.SetHeader(ctx, md2)
		return handler(ctx, req)
	}
}
func EnsureOTELCorrelationIDUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var correlationID string // if not found in header, we generate a new one
		var requestID = "0"
		md := metautils.ExtractIncoming(ctx)
		var loggerMap = make(map[string]string)

		// Extract the TraceID from the traceparent header
		// THIS MUST EXIST.  We require otel to be putting this in before this comes in.
		sc := otel_trace.SpanContextFromContext(ctx)
		traceID := sc.TraceID().String()

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

		loggerMap[fluffycore_wellknown.LogCorrelationIDName] = correlationID
		loggerMap["trace_id"] = traceID

		// Return the cleansed metadata context
		ctx = withLoggerFields(md.ToIncoming(ctx), loggerMap)

		md2 := metadata.Pairs(
			fluffycore_wellknown.XRequestID, requestID,
			fluffycore_wellknown.XCorrelationIDName, correlationID)
		// SetHeader, not SendHeader: SendHeader flushes the headers now, after which the
		// handler's own grpc.SetHeader calls fail. SetHeader merges and sends with the response.
		_ = grpc.SetHeader(ctx, md2)
		return handler(ctx, req)
	}
}

// withLoggerFields returns ctx carrying a logger with fields added: a child of the logger
// already on ctx, or of the global logger when there is none. The interceptors used to call
// zerolog.Ctx(ctx).UpdateContext: with no logger on ctx yet (this runs before the logging
// interceptor) that updated zerolog's disabled logger, so correlation and trace IDs never
// reached any log line -- and with one, it would have mutated a logger shared across requests.
func withLoggerFields(ctx context.Context, fields map[string]string) context.Context {
	base := log.Logger
	if l := zerolog.Ctx(ctx); l.GetLevel() != zerolog.Disabled {
		base = *l
	}
	c := base.With()
	for k, v := range fields {
		c = c.Str(k, v)
	}
	return c.Logger().WithContext(ctx)
}
