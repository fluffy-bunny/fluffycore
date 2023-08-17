package metadatafilter

import (
	"google.golang.org/grpc"
)

//go:generate mockgen -package=$GOPACKAGE -destination=../../mocks/$GOPACKAGE/mock_$GOFILE   github.com/fluffy-bunny/grpcdotnetgo/pkg/contracts/$GOPACKAGE IMetadataFilterMiddleware

type (
	// IMetadataFilterMiddleware ...
	IMetadataFilterMiddleware interface {
		// GetUnaryServerInterceptor ...
		GetUnaryServerInterceptor() grpc.UnaryServerInterceptor
	}
)
