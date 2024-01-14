package logging

import (
	"context"

	fluffycore_middleware "github.com/fluffy-bunny/fluffycore/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

func EnsureContextLoggingUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger := log.With().Caller().Logger()
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
