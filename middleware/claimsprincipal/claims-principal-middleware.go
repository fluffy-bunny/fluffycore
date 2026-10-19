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
		if err := authorize(ctx, info.FullMethod, grpcEntrypointClaimsMap, enableZeroTrust); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// FinalAuthVerificationStreamMiddlewareUsingClaimsMapWithZeroTrustV2 is the streaming
// counterpart of FinalAuthVerificationMiddlewareUsingClaimsMapWithZeroTrustV2: the same
// entrypoint map and zero-trust rule, keyed on the stream's FullMethod.
func FinalAuthVerificationStreamMiddlewareUsingClaimsMapWithZeroTrustV2(grpcEntrypointClaimsMap map[string]claimsprincipalContracts.IEntryPointConfig) grpc.StreamServerInterceptor {
	return FinalAuthVerificationStreamMiddlewareUsingClaimsMapWithTrustOptionV2(grpcEntrypointClaimsMap, true)
}

// FinalAuthVerificationStreamMiddlewareUsingClaimsMapWithTrustOptionV2 is the streaming
// counterpart of FinalAuthVerificationMiddlewareUsingClaimsMapWithTrustOptionV2.
func FinalAuthVerificationStreamMiddlewareUsingClaimsMapWithTrustOptionV2(grpcEntrypointClaimsMap map[string]claimsprincipalContracts.IEntryPointConfig, enableZeroTrust bool) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := authorize(ss.Context(), info.FullMethod, grpcEntrypointClaimsMap, enableZeroTrust); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

// authorize is the single decision point shared by the unary and stream gates, so the two
// can never drift apart. It fails closed: when there's no request-scoped container to read
// the claims principal from (dicontext not registered, or registered after this gate), the
// method's requirement is evaluated against an EMPTY principal rather than skipped -- so open
// methods still work, and anything that requires a claim is denied. (Previously a missing
// container skipped every check, even under zero trust.)
func authorize(ctx context.Context, fullMethod string,
	grpcEntrypointClaimsMap map[string]claimsprincipalContracts.IEntryPointConfig, enableZeroTrust bool) error {
	elem, ok := grpcEntrypointClaimsMap[fullMethod]
	if !ok {
		if enableZeroTrust {
			zerolog.Ctx(ctx).Debug().Str("FullMethod", fullMethod).Msg("FullMethod not found in entrypoint claims map; permission denied")
			return status.Error(codes.PermissionDenied, "permission denied")
		}
		return nil
	}
	if elem == nil {
		return nil
	}
	var claimsPrincipal claimsprincipalContracts.IClaimsPrincipal
	if requestContainer := middleware_dicontext.GetRequestContainer(ctx); requestContainer != nil {
		claimsPrincipal, _ = di.TryGet[claimsprincipalContracts.IClaimsPrincipal](requestContainer)
	}
	if claimsPrincipal == nil {
		zerolog.Ctx(ctx).Error().Str("FullMethod", fullMethod).
			Msg("no request-scoped IClaimsPrincipal (is dicontext registered before the auth gate?); evaluating with no claims")
		claimsPrincipal = services_claimfact.NewIClaimsPrincipal()
	}
	if !elem.GetClaimsAST().Validate(claimsPrincipal) {
		zerolog.Ctx(ctx).Debug().Str("FullMethod", fullMethod).Msg("ClaimsConfig validation failed; permission denied")
		return status.Error(codes.PermissionDenied, "permission denied")
	}
	return nil
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
