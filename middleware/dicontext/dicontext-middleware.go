package dicontext

import (
	"context"

	di "github.com/dozm/di"
	"google.golang.org/grpc"
)

func UnaryServerInterceptor(rootContainer di.Container) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		scopeFactory := di.Get[di.ScopeFactory](rootContainer)
		scope := scopeFactory.CreateScope()
		requestContainer := scope.Container()
		ctx = SetRequestContainer(ctx, requestContainer)
		return handler(ctx, req)
	}
}
