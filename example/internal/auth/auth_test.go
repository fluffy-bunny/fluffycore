package auth

import (
	"testing"

	contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	services_common_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	"github.com/stretchr/testify/require"
)

func secretEndpointClaimsAST(t *testing.T) contracts_common.IClaimsAST {
	t.Helper()
	claimsMap := BuildGrpcEntrypointPermissionsClaimsMap()
	entry, ok := claimsMap[proto_helloworld.Greeter_SetSecret_FullMethodName]
	require.True(t, ok)
	return entry.GetClaimsAST()
}

func TestSecretEndpoints_DeniesUnauthenticatedCaller(t *testing.T) {
	ast := secretEndpointClaimsAST(t)
	cp := services_common_claimsprincipal.NewIClaimsPrincipal()
	require.False(t, ast.Validate(cp), "a request with no claims at all must be denied")
}

func TestSecretEndpoints_DeniesAnonymousFallback(t *testing.T) {
	// This is what the JWT middleware itself adds when there's no Authorization
	// header at all -- it must NOT be mistaken for "authenticated via JWT".
	ast := secretEndpointClaimsAST(t)
	cp := services_common_claimsprincipal.NewIClaimsPrincipal()
	cp.AddClaim(contracts_common.Claim{Type: fluffycore_wellknown.ClaimTypeSub, Value: fluffycore_wellknown.AnonymousSubject})
	require.False(t, ast.Validate(cp), "the anonymous-fallback sub claim must not satisfy the JWT branch")
}

func TestSecretEndpoints_AllowsRealJWT(t *testing.T) {
	ast := secretEndpointClaimsAST(t)
	cp := services_common_claimsprincipal.NewIClaimsPrincipal()
	cp.AddClaim(contracts_common.Claim{Type: fluffycore_wellknown.ClaimTypeSub, Value: "real-user-123"})
	require.True(t, ast.Validate(cp), "a real, non-anonymous sub claim must satisfy the JWT branch")
}

func TestSecretEndpoints_AllowsMTLSOnly(t *testing.T) {
	ast := secretEndpointClaimsAST(t)
	cp := services_common_claimsprincipal.NewIClaimsPrincipal()
	cp.AddClaim(contracts_common.Claim{Type: fluffycore_wellknown.ClaimTypeMTLSVerified, Value: "true"})
	require.True(t, ast.Validate(cp), "a verified mTLS claim alone must satisfy the mTLS branch")
}

func TestSecretEndpoints_AllowsMTLSEvenWithAnonymousJWTFallback(t *testing.T) {
	// The realistic shape of an mTLS-only caller: no bearer token was ever
	// presented, so the JWT middleware still adds sub=anonymous, but a client
	// certificate verified -- the OR must still allow it via the mTLS branch.
	ast := secretEndpointClaimsAST(t)
	cp := services_common_claimsprincipal.NewIClaimsPrincipal()
	cp.AddClaim(contracts_common.Claim{Type: fluffycore_wellknown.ClaimTypeSub, Value: fluffycore_wellknown.AnonymousSubject})
	cp.AddClaim(contracts_common.Claim{Type: fluffycore_wellknown.ClaimTypeMTLSVerified, Value: "true"})
	require.True(t, ast.Validate(cp))
}

func TestSecretEndpoints_UnrelatedPermissionClaimDoesNotLeakIn(t *testing.T) {
	ast := secretEndpointClaimsAST(t)
	cp := services_common_claimsprincipal.NewIClaimsPrincipal()
	cp.AddClaim(contracts_common.Claim{Type: "permissions", Value: "write"})
	require.False(t, ast.Validate(cp), "the write-permission claim used by other endpoints must not satisfy this OR")
}
