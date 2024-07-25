package claimsprincipal

import (
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_auth "github.com/fluffy-bunny/fluffycore/contracts/auth"
	contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	fluffycore_middleware_claimsprincipal "github.com/fluffy-bunny/fluffycore/middleware/claimsprincipal"
	grpc "google.golang.org/grpc"
)

type (
	service struct {
		GetEntryPointConfig fluffycore_contracts_auth.GetEntryPointConfig
	}
)

var stemService = (*service)(nil)
var _ fluffycore_contracts_auth.IFinalAuthVerificationServerOptionAccessor = (*service)(nil)

func (s *service) Ctor(getEntryPointConfig fluffycore_contracts_auth.GetEntryPointConfig) fluffycore_contracts_auth.IFinalAuthVerificationServerOptionAccessor {
	return &service{
		GetEntryPointConfig: getEntryPointConfig,
	}
}

// AddFinalAuthVerificationServerOptionAccessor ...
func AddFinalAuthVerificationServerOptionAccessor(b di.ContainerBuilder, config map[string]contracts_common.IEntryPointConfig) {
	fluffycore_contracts_auth.AddGetEntryPointConfigFunc(b, config)
	di.AddSingleton[fluffycore_contracts_auth.IFinalAuthVerificationServerOptionAccessor](b, stemService.Ctor)
}
func (s *service) GetServerOption() *grpc.ServerOption {
	config := s.GetEntryPointConfig()
	opt := grpc.ChainUnaryInterceptor(fluffycore_middleware_claimsprincipal.FinalAuthVerificationMiddlewareUsingClaimsMapWithZeroTrustV2(config))
	return &opt
}
