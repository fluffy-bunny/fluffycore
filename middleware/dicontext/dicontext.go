package dicontext

import (
	"context"

	di "github.com/dozm/di"
)

type ctxRequestContainerContext struct{}

var ctxRequestContainerContextKey = &ctxRequestContainerContext{}

// GetRequestContainer pulls the request container from the context
func GetRequestContainer(ctx context.Context) di.Container {
	val := ctx.Value(ctxRequestContainerContextKey)
	if val == nil {
		return nil
	}
	requestContainer, ok := val.(di.Container)
	if !ok {
		return nil
	}
	return requestContainer
}

// SetRequestContainer adds the request container to the context
func SetRequestContainer(ctx context.Context, requestContainer di.Container) context.Context {
	ctx = context.WithValue(ctx, ctxRequestContainerContextKey, requestContainer)
	return ctx
}
