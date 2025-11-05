package GRPCClientFactory

//go:generate mockgen -package=$GOPACKAGE -destination=../mocks/$GOPACKAGE/mock_$GOFILE  github.com/fluffy-bunny/fluffycore/contracts/$GOPACKAGE IGRPCClientFactory

import (
	fluffycore_grpcclient "github.com/fluffy-bunny/fluffycore/grpcclient"
)

type (
	GRPCClientConfig struct {
		OTELTracingEnabled    bool `json:"otelTracingEnabled"`
		DataDogTracingEnabled bool `json:"dataDogTracingEnabled"`
	}
	// IGRPCClientFactory ...
	IGRPCClientFactory interface {
		NewGrpcClient(opts ...fluffycore_grpcclient.GrpcClientOption) (*fluffycore_grpcclient.GrpcClient, error)
	}
)
