package dicontext

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_middleware "github.com/fluffy-bunny/fluffycore/middleware"
	"google.golang.org/grpc"
)

func ScopedContextUnaryServerInterceptor(rootContainer di.Container) grpc.UnaryServerInterceptor {
	// Resolved once: it used to be looked up in the root container on every request.
	scopeFactory := di.Get[di.ScopeFactory](rootContainer)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		scope := scopeFactory.CreateScope()
		defer scope.Dispose()
		requestContainer := scope.Container()
		ctx = SetRequestContainer(ctx, requestContainer)
		return handler(ctx, req)
	}
}

func ScopedContextStreamServerInterceptor(rootContainer di.Container) grpc.StreamServerInterceptor {
	scopeFactory := di.Get[di.ScopeFactory](rootContainer)
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		scope := scopeFactory.CreateScope()
		defer scope.Dispose()
		requestContainer := scope.Container()
		ctx := ss.Context()
		newCtx := SetRequestContainer(ctx, requestContainer)
		sw := fluffycore_middleware.NewStreamContextWrapper(ss)
		sw.SetContext(newCtx)
		return handler(srv, sw)
	}
}
