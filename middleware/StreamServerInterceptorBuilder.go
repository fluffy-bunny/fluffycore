package middleware

import (
	contract_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	"google.golang.org/grpc"
)

// StreamServerInterceptorBuilder struct
type StreamServerInterceptorBuilder struct {
	StreamServerInterceptors []grpc.StreamServerInterceptor
}

// NewStreamServerInterceptorBuilder helper
func NewStreamServerInterceptorBuilder() contract_middleware.IStreamServerInterceptorBuilder {
	return &StreamServerInterceptorBuilder{}
}

func (s *StreamServerInterceptorBuilder) GetStreamServerInterceptors() []grpc.StreamServerInterceptor {
	return s.StreamServerInterceptors
}
func (s *StreamServerInterceptorBuilder) Use(intercepter grpc.StreamServerInterceptor) {
	s.StreamServerInterceptors = append(s.StreamServerInterceptors, intercepter)
}
