package correlation

import (
	"context"
	"strings"

	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	metautils "github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	zerolog "github.com/rs/zerolog"
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
