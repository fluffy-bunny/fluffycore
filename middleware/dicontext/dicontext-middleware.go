package dicontext

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_middleware "github.com/fluffy-bunny/fluffycore/middleware"
	grpc "google.golang.org/grpc"
)

func UnaryServerInterceptor(rootContainer di.Container) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		scopeFactory := di.Get[di.ScopeFactory](rootContainer)
		scope := scopeFactory.CreateScope()
		defer scope.Dispose()
		requestContainer := scope.Container()
		ctx = SetRequestContainer(ctx, requestContainer)
		return handler(ctx, req)
	}
}

func StreamServerInterceptor(rootContainer di.Container) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		scopeFactory := di.Get[di.ScopeFactory](rootContainer)
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
