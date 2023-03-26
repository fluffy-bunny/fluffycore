package logging

import (
	"context"

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
