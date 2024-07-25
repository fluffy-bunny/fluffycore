package nilverification

import (
	fluffycore_contracts_auth "github.com/fluffy-bunny/fluffycore/contracts/auth"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
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
func AddFinalAuthVerificationServerOptionAccessor(b di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_auth.IFinalAuthVerificationServerOptionAccessor](b, stemService.Ctor)
}
func (s *service) GetServerOption() *grpc.ServerOption {
	opt := grpc.ChainUnaryInterceptor(fluffycore_middleware_claimsprincipal.FinalAuthVerificationMiddlewareNilVefication())
	return &opt
}
