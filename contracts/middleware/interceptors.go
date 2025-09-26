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
	// JSONRequestPropagationName
	JSONRequestPropagationName string `json:"jsonRequestPropagationName"`
	// ClaimToContextMap
	ClaimToContextMap map[string]string `json:"claimToContextMap"`
}
