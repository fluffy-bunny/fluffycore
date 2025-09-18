package logging

import (
	"context"
	"encoding/json"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_propertybag "github.com/fluffy-bunny/fluffycore/contracts/propertybag"
	fluffycore_middleware "github.com/fluffy-bunny/fluffycore/middleware"
	middleware_dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	"github.com/gogo/status"
	zerolog "github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type PropertyBagHook struct {
	PropertyBag fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag
}

func (h PropertyBagHook) Run(e *zerolog.Event, l zerolog.Level, msg string) {

	propertyMap := h.PropertyBag.AsMap()

	e.Interface("ctx", propertyMap)
}

func getIncomingContextJson(ctx context.Context) (map[string]interface{}, error) {
	// pull bearer token from context using metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "no metadata found")
	}
	// its an Authorization : Bearer {{token}}
	jsonContextPropagation := md.Get("jsonContextPropagation")
	if len(jsonContextPropagation) == 0 {
		// not having anything is ok.
		return nil, nil
	}
	generic := make(map[string]interface{})
	// ensure it is valid json
	err := json.Unmarshal([]byte(jsonContextPropagation[0]), &generic)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid jsonContextPropagation")
	}
	return generic, nil
}
func EnsureContextLoggingUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger := log.With().Caller().Logger()

		scopedContainer := middleware_dicontext.GetRequestContainer(ctx)
		propertyBag := di.Get[fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag](scopedContainer)

		jsonContextPropagation, err := getIncomingContextJson(ctx)
		if err != nil {
			// log the error, but continue on.
			logger.Error().Err(err).Msg("getIncomingContextJson failed")
		} else {
			if jsonContextPropagation != nil {
				propertyBag.Set("jsonContextPropagation", jsonContextPropagation)
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
		logger := log.With().Caller().Logger()
		ctx := ss.Context()
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
