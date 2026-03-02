package metadatafilter

import (
	"context"
	"strings"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_metadatafilter "github.com/fluffy-bunny/fluffycore/contracts/metadatafilter"
	hashset "github.com/fluffy-bunny/fluffycore/gods/sets/hashset"
	metautils "github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	grpc "google.golang.org/grpc"
)

type (
	metadataFilterMiddleware struct {
		allowedGlobal       *hashset.StringSet
		allowedByEntryPoint map[string]*hashset.StringSet
	}
)

func init() {
	var _ contracts_metadatafilter.IMetadataFilterMiddleware = (*metadataFilterMiddleware)(nil)
}

// AddSingletonIMetadataFilterMiddleware adds service to the DI container
func AddSingletonIMetadataFilterMiddleware(builder di.ContainerBuilder,
	allowedGlobal *hashset.StringSet,
	allowedByEntryPoint map[string]*hashset.StringSet) {
	di.AddSingleton[contracts_metadatafilter.IMetadataFilterMiddleware](builder, func() (contracts_metadatafilter.IMetadataFilterMiddleware, error) {
		return &metadataFilterMiddleware{
			allowedGlobal:       allowedGlobal,
			allowedByEntryPoint: allowedByEntryPoint,
		}, nil
	})
}

// GetUnaryServerInterceptor ...
func (s *metadataFilterMiddleware) GetUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md := metautils.ExtractIncoming(ctx)

		entryPointAllowed, entryPointExists := s.allowedByEntryPoint[info.FullMethod]
		notAllowedHeaders := []string{}
		for header := range md {
			exists := s.allowedGlobal.Contains(header)
			if exists {
				continue
			}
			// is it explicitly allowed for this entry point?
			if entryPointExists {
				exists := entryPointAllowed.Contains(strings.ToLower(header))
				if exists {
					continue
				}
			}
			notAllowedHeaders = append(notAllowedHeaders, header)
		}
		for _, header := range notAllowedHeaders {
			md.Del(header)
		}
		// commit our changes
		newCtx := md.ToIncoming(ctx)
		return handler(newCtx, req)
	}
}
