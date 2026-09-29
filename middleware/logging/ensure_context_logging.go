package logging

import (
	"context"
	"encoding/json"
	"sync"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	fluffycore_contracts_propertybag "github.com/fluffy-bunny/fluffycore/contracts/propertybag"
	fluffycore_middleware "github.com/fluffy-bunny/fluffycore/middleware"
	middleware_dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	status "github.com/gogo/status"
	zerolog "github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	metadata "google.golang.org/grpc/metadata"
)

type PropertyBagHook struct {
	PropertyBag fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag
}

func (h PropertyBagHook) Run(e *zerolog.Event, l zerolog.Level, msg string) {

	propertyMap := h.PropertyBag.AsMap()
	if len(propertyMap) == 0 {
		return
	}
	// Dict+Fields writes strings, numbers and []string directly; e.Interface used to
	// json.Marshal the whole map through reflection on every log line. Same output shape.
	e.Dict("ctx", zerolog.Dict().Fields(propertyMap))
}

func getIncomingOriginContextJson(ctx context.Context) (map[string]interface{}, error) {
	// pull bearer token from context using metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "no metadata found")
	}
	// its an Authorization : Bearer {{token}}
	ctxOrigin := md.Get(fluffycore_wellknown.ContextOriginKey)
	if len(ctxOrigin) == 0 {
		// not having anything is ok.
		return nil, nil
	}
	generic := make(map[string]interface{})
	// ensure it is valid json
	err := json.Unmarshal([]byte(ctxOrigin[0]), &generic)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ctxOrigin")
	}
	return generic, nil
}

// baseLogger is the logger already on ctx (e.g. carrying the correlation interceptor's
// correlation_id / trace_id), or the global logger when there is none. These interceptors
// used to always start from the global logger, dropping those fields.
func baseLogger(ctx context.Context) zerolog.Logger {
	if l := zerolog.Ctx(ctx); l.GetLevel() != zerolog.Disabled {
		return *l
	}
	return log.Logger
}

func EnsureContextLoggingUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	var ctxOriginOnce sync.Once
	var ctxOriginName string
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger := baseLogger(ctx).With().Caller().Logger()

		scopedContainer := middleware_dicontext.GetRequestContainer(ctx)
		propertyBag := di.Get[fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag](scopedContainer)

		// Resolved on the first request and reused (it is configuration): it used to be looked
		// up -- usually failing -- on every request.
		ctxOriginOnce.Do(func() {
			ctxOriginName = fluffycore_wellknown.ContextOriginKey
			if cfg, err := di.TryGet[*fluffycore_contracts_middleware.RequestClaimsContextPropagateConfig](scopedContainer); err == nil && cfg != nil && cfg.ContextOrigin != "" {
				ctxOriginName = cfg.ContextOrigin
			}
		})
		ctxOrigin, err := getIncomingOriginContextJson(ctx)
		if err != nil {
			// log the error, but continue on.
			logger.Error().Err(err).Msg("getIncomingOriginContextJson failed")
		} else {
			if ctxOrigin != nil {
				propertyBag.Set(ctxOriginName, ctxOrigin)
			}
		}

		//propertyBag.Set("method", info.FullMethod)
		propertyBagHook := &PropertyBagHook{
			PropertyBag: propertyBag,
		}
		mappHook := zerolog.Hook(propertyBagHook)
		logger = logger.Hook(mappHook)

		newCtx := logger.WithContext(ctx)
		return handler(newCtx, req)
	}
}
func EnsureContextLoggingStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		logger := baseLogger(ctx).With().Caller().Logger()
		newCtx := logger.WithContext(ctx)
		sw := fluffycore_middleware.NewStreamContextWrapper(ss)
		sw.SetContext(newCtx)
		return handler(srv, sw)
	}
}

// LoggingUnaryServerInterceptor returns a new unary server interceptors that performs request logging in JSON format
func LoggingUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger := zerolog.Ctx(ctx)
		if logger != nil {
			logger.Trace().
				Interface("request", req).
				Msg("Handling request")
		}

		return handler(ctx, req)
	}
}
