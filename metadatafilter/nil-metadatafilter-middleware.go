package metadatafilter

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_metadatafilter "github.com/fluffy-bunny/fluffycore/contracts/metadatafilter"
	grpc "google.golang.org/grpc"
)

type (
	nilMetadataFilterMiddleware struct{}
)

func init() {
	var _ contracts_metadatafilter.IMetadataFilterMiddleware = (*nilMetadataFilterMiddleware)(nil)
}

// AddSingletonIMetadataFilterMiddlewareNil adds service to the DI container
func AddSingletonIMetadataFilterMiddlewareNil(builder di.ContainerBuilder) {
	di.AddSingleton[contracts_metadatafilter.IMetadataFilterMiddleware](builder, func() (contracts_metadatafilter.IMetadataFilterMiddleware, error) {
		return &nilMetadataFilterMiddleware{}, nil
	})
}

// GetUnaryServerInterceptor ...
func (s *nilMetadataFilterMiddleware) GetUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}
}
