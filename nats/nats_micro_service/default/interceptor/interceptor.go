package interceptor

import (
	"context"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_nats_micro_service "github.com/fluffy-bunny/fluffycore/contracts/nats_micro_service"
	fluffycore_middleware_dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	log "github.com/rs/zerolog/log"
	grpc "google.golang.org/grpc"
)

type UnaryNATSInterceptor func(ctx context.Context, req any, handler grpc.UnaryHandler) (resp any, err error)

func EnsureRequestContextUnaryNATSInterceptor(rootContainer di.Container) UnaryNATSInterceptor {
	return func(ctx context.Context, req interface{}, handler grpc.UnaryHandler) (interface{}, error) {
		scopeFactory := di.Get[di.ScopeFactory](rootContainer)
		scope := scopeFactory.CreateScope()
		defer scope.Dispose()
		requestContainer := scope.Container()
		ctx = fluffycore_middleware_dicontext.SetRequestContainer(ctx, requestContainer)
		return handler(ctx, req)
	}
}
func EnsureContextLoggingUnaryNATSInterceptor() UnaryNATSInterceptor {
	return func(ctx context.Context, req interface{}, handler grpc.UnaryHandler) (interface{}, error) {
		logger := log.With().Caller().Logger()
		newCtx := logger.WithContext(ctx)
		return handler(newCtx, req)
	}
}
func EnsureContextTimeoutUnaryNATSInterceptor(timeout time.Duration) UnaryNATSInterceptor {
	return func(ctx context.Context, req interface{}, handler grpc.UnaryHandler) (interface{}, error) {
		newCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return handler(newCtx, req)
	}
}

type service struct {
	interceptors []UnaryNATSInterceptor
}

var stemService = (*service)(nil)
var _ contracts_nats_micro_service.INATSMicroInterceptors = stemService

func (c *service) Ctor(
	rootContainer di.Container,
	config *contracts_nats_micro_service.NATSInterceptorConfig,
) (contracts_nats_micro_service.INATSMicroInterceptors, error) {

	timeoutDuration, err := time.ParseDuration(config.TimeoutDuration)
	if err != nil {
		return nil, err
	}

	return &service{
		interceptors: []UnaryNATSInterceptor{
			EnsureRequestContextUnaryNATSInterceptor(rootContainer),
			EnsureContextLoggingUnaryNATSInterceptor(),
			EnsureContextTimeoutUnaryNATSInterceptor(timeoutDuration),
		},
	}, nil

}

func AddSingletonNATSMicroInterceptors(cb di.ContainerBuilder) {
	di.AddSingleton[contracts_nats_micro_service.INATSMicroInterceptors](cb, stemService.Ctor)
}

func (c *service) WithHandler(finalHandler grpc.UnaryHandler) grpc.UnaryHandler {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		// Start with the final handler
		handler := finalHandler

		// Chain interceptors in reverse order
		for i := len(c.interceptors) - 1; i >= 0; i-- {
			currentInterceptor := c.interceptors[i]
			previousHandler := handler
			handler = func(ctx context.Context, req interface{}) (interface{}, error) {
				return currentInterceptor(ctx, req, previousHandler)
			}
		}

		// Execute the chained interceptors
		return handler(ctx, req)
	}
}
