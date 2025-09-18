package GRPCClientFactory

import (
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_GRPCClientFactory "github.com/fluffy-bunny/fluffycore/contracts/GRPCClientFactory"
	fluffycore_grpcclient "github.com/fluffy-bunny/fluffycore/grpcclient"
)

type (
	service struct {
		config *fluffycore_contracts_GRPCClientFactory.GRPCClientConfig
	}
)

var stemService = (*service)(nil)

func (s *service) Ctor(config *fluffycore_contracts_GRPCClientFactory.GRPCClientConfig) (fluffycore_contracts_GRPCClientFactory.IGRPCClientFactory, error) {
	return &service{
		config: config,
	}, nil
}

var _ fluffycore_contracts_GRPCClientFactory.IGRPCClientFactory = (*service)(nil)

// AddSingletonIGRPCClientFactory ...
func AddSingletonIGRPCClientFactory(builder di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_GRPCClientFactory.IGRPCClientFactory](builder, stemService.Ctor)
}

func (s *service) NewGrpcClient(opts ...fluffycore_grpcclient.GrpcClientOption) (*fluffycore_grpcclient.GrpcClient, error) {
	if s.config.OTELTracingEnabled {
		opts = append(opts, fluffycore_grpcclient.WithOTELTracer(s.config.OTELTracingEnabled))
	}
	grpcClient, err := fluffycore_grpcclient.NewGrpcClient(opts...)
	return grpcClient, err
}
