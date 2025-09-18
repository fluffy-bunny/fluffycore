package middleware

import "google.golang.org/grpc"

// IUnaryServerInterceptorBuilder ...
type IUnaryServerInterceptorBuilder interface {
	GetUnaryServerInterceptors() []grpc.UnaryServerInterceptor
	Use(intercepter grpc.UnaryServerInterceptor)
}

type IStreamServerInterceptorBuilder interface {
	GetStreamServerInterceptors() []grpc.StreamServerInterceptor
	Use(intercepter grpc.StreamServerInterceptor)
}

type RequestContextClaimsToPropagate struct {
	ClaimTypes []string `json:"claimTypes"`
}
