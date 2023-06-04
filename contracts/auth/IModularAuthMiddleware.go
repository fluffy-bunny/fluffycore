package auth

import (
	"google.golang.org/grpc"
)

type (
	// IModularAuthMiddleware ...
	IModularAuthMiddleware interface {
		// GetUnaryServerInterceptor ...
		GetUnaryServerInterceptor() grpc.UnaryServerInterceptor
	}
)
