package claimsprincipal

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	claimsprincipalContracts "github.com/fluffy-bunny/fluffycore/contracts/common"
	middleware_dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	middleware_oidc "github.com/fluffy-bunny/fluffycore/middleware/oidc"
	services_claimfact "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	utils "github.com/fluffy-bunny/fluffycore/utils"
	status "github.com/gogo/status"
	zerolog "github.com/rs/zerolog"
	log "github.com/rs/zerolog/log"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
)

// Validate ...
func Validate(logger *zerolog.Logger, claimsConfig *middleware_oidc.ClaimsConfig,
	claimsPrincipal claimsprincipalContracts.IClaimsPrincipal) bool {
	if !validateAND(claimsConfig, claimsPrincipal) {
		logger.Debug().Msg("AND validation failed")
		return false
	}

	if !validateOR(claimsConfig, claimsPrincipal) {
		logger.Debug().Msg("OR validation failed")
		return false
	}
	if claimsConfig.Child != nil {
		return Validate(logger, claimsConfig.Child, claimsPrincipal)
	}
	return true
}
func validateAND(claimsConfig *middleware_oidc.ClaimsConfig,
	claimsPrincipal claimsprincipalContracts.IClaimsPrincipal) bool {
	if utils.IsEmptyOrNil(claimsConfig.AND) {
		return true
	}

	if !utils.IsEmptyOrNil(claimsConfig.AND) {
		for _, v := range claimsConfig.AND {
			if v.Directive == services_claimfact.ClaimTypeAndValue {
				if !claimsPrincipal.HasClaim(v.Claim) {
					return false
				}
			}
			if v.Directive == services_claimfact.ClaimType {
				if !claimsPrincipal.HasClaimType(v.Claim.Type) {
					return false
				}
			}
		}
	}

	return true
}

func validateOR(claimsConfig *middleware_oidc.ClaimsConfig,
	claimsPrincipal claimsprincipalContracts.IClaimsPrincipal) bool {
	if utils.IsEmptyOrNil(claimsConfig.OR) {
		return true
	}
	if !utils.IsEmptyOrNil(claimsConfig.OR) {
		for _, v := range claimsConfig.OR {
			if v.Directive == services_claimfact.ClaimTypeAndValue {
				if claimsPrincipal.HasClaim(v.Claim) {
					return true
				}
			}
			if v.Directive == services_claimfact.ClaimType {
				if claimsPrincipal.HasClaimType(v.Claim.Type) {
					return true
				}
			}
		}
	}

	return false
}

// FinalAuthVerificationMiddlewareUsingClaimsMapWithZeroTrustV2 evaluates the claims principal
func FinalAuthVerificationMiddlewareUsingClaimsMapWithZeroTrustV2(grpcEntrypointClaimsMap map[string]claimsprincipalContracts.IEntryPointConfig) grpc.UnaryServerInterceptor {
	return FinalAuthVerificationMiddlewareUsingClaimsMapWithTrustOptionV2(grpcEntrypointClaimsMap, true)
}

// FinalAuthVerificationMiddlewareUsingClaimsMapWithTrustOptionV2 evaluates the claims principal
func FinalAuthVerificationMiddlewareUsingClaimsMapWithTrustOptionV2(grpcEntrypointClaimsMap map[string]claimsprincipalContracts.IEntryPointConfig, enableZeroTrust bool) grpc.UnaryServerInterceptor {
	log.Info().Interface("entryPointConfig", grpcEntrypointClaimsMap).Send()

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger := zerolog.Ctx(ctx)
		subLogger := logger.With().
			Bool("enableZeroTrust", enableZeroTrust).
			Str("FullMethod", info.FullMethod).
			Logger()
		requestContainer := middleware_dicontext.GetRequestContainer(ctx)

		subLogger = subLogger.With().Logger()

		if requestContainer != nil {

			claimsPrincipal := di.Get[claimsprincipalContracts.IClaimsPrincipal](requestContainer)
			permissionDeniedFunc := func() (interface{}, error) {
				subLogger.Debug().Msg("Permission denied")
				return nil, status.Errorf(codes.PermissionDenied, "permission denied")
			}
			elem, ok := grpcEntrypointClaimsMap[info.FullMethod]
			if !ok {
				if enableZeroTrust {
					subLogger.Debug().Msg("FullMethod not found in entrypoint claims map")
					return permissionDeniedFunc()
				}
			}
			if !ok && enableZeroTrust {
				subLogger.Debug().Msg("FullMethod not found in entrypoint claims map")
				return permissionDeniedFunc()
			}
			if !ok || elem == nil {
				return handler(ctx, req)
			}
			claimsAST := elem.GetClaimsAST()
			valid := claimsAST.Validate(claimsPrincipal)
			if !valid {
				subLogger.Debug().Msg("ClaimsConfig validation failed")
				return permissionDeniedFunc()
			}
		}
		return handler(ctx, req)
	}
}
func FinalAuthVerificationMiddlewareNilVerification() grpc.UnaryServerInterceptor {
	log.Info().Msg("FinalAuthVerificationMiddlewareNilVerification")
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}
}

// Deprecated: Use FinalAuthVerificationMiddlewareNilVerification instead.
func FinalAuthVerificationMiddlewareNilVefication() grpc.UnaryServerInterceptor {
	return FinalAuthVerificationMiddlewareNilVerification()
}
