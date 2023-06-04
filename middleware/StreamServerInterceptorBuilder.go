package middleware

import (
	"context"

	contract_middleware "github.com/fluffy-bunny/fluffycore/contracts/middleware"
	"google.golang.org/grpc"
)

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

func (w *wrappedStream) SetContext(ctx context.Context) {
	w.ctx = ctx
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	return w.ServerStream.RecvMsg(m)
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	return w.ServerStream.SendMsg(m)
}

type IStreamContextWrapper interface {
	grpc.ServerStream
	SetContext(context.Context)
}

func NewStreamContextWrapper(ss grpc.ServerStream) IStreamContextWrapper {
	ctx := ss.Context()
	return &wrappedStream{
		ss,
		ctx,
	}
}

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
