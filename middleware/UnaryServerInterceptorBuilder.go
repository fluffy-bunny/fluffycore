package middleware

import (
	contract_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	"google.golang.org/grpc"
)

// UnaryServerInterceptorBuilder struct
type UnaryServerInterceptorBuilder struct {
	UnaryServerInterceptors []grpc.UnaryServerInterceptor
}

// NewUnaryServerInterceptorBuilder helper
func NewUnaryServerInterceptorBuilder() contract_middleware.IUnaryServerInterceptorBuilder {
	return &UnaryServerInterceptorBuilder{}
}

// Use helper
func (s *UnaryServerInterceptorBuilder) Use(interceptor grpc.UnaryServerInterceptor) {
	s.UnaryServerInterceptors = append(s.UnaryServerInterceptors, interceptor)
}

// GetUnaryServerInterceptors helper
func (s *UnaryServerInterceptorBuilder) GetUnaryServerInterceptors() []grpc.UnaryServerInterceptor {
	return s.UnaryServerInterceptors
}
