package auth

import (
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	grpc "google.golang.org/grpc"
)

type (
	// IModularAuthMiddleware ...
	IModularAuthMiddleware interface {
		// GetUnaryServerInterceptor ...
		GetUnaryServerInterceptor() grpc.UnaryServerInterceptor
	}
	IServerOptionAccessor interface {
		GetServerOption() *grpc.ServerOption
	}
	IFinalAuthVerificationServerOptionAccessor interface {
		IServerOptionAccessor
	}
	GetEntryPointConfig func() map[string]contracts_common.IEntryPointConfig
)

func AddGetEntryPointConfigFunc(builder di.ContainerBuilder,
	config map[string]contracts_common.IEntryPointConfig) {
	di.AddFunc[GetEntryPointConfig](builder, func() GetEntryPointConfig {
		return func() map[string]contracts_common.IEntryPointConfig {
			return config
		}
	})
}
