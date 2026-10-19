package claimsprincipal

import (
	"testing"

	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	"github.com/stretchr/testify/require"
)

func TestWithoutReservedMTLSClaims(t *testing.T) {
	in := []fluffycore_contracts_common.Claim{
		{Type: "sub", Value: "u"},
		{Type: fluffycore_wellknown.ClaimTypeMTLSVerified, Value: "true"},
		{Type: fluffycore_wellknown.ClaimTypeMTLSCommonName, Value: "cn"},
		{Type: fluffycore_wellknown.ClaimTypeMTLSFingerprint, Value: "ab"},
		{Type: fluffycore_wellknown.ClaimTypeMTLSSANURI, Value: "spiffe://x"},
		{Type: "scope", Value: "read"},
	}
	out := WithoutReservedMTLSClaims(in)
	require.Equal(t, []fluffycore_contracts_common.Claim{{Type: "sub", Value: "u"}, {Type: "scope", Value: "read"}}, out)
	require.Equal(t, fluffycore_wellknown.ClaimTypeMTLSVerified, in[1].Type, "input must not be modified")
	require.Empty(t, WithoutReservedMTLSClaims(nil))
}

func TestClaimsAddedSince(t *testing.T) {
	a := fluffycore_contracts_common.Claim{Type: "a", Value: "1"}
	b := fluffycore_contracts_common.Claim{Type: "b", Value: "2"}
	a2 := fluffycore_contracts_common.Claim{Type: "a", Value: "other"}
	require.Equal(t, []fluffycore_contracts_common.Claim{b, a2}, ClaimsAddedSince([]fluffycore_contracts_common.Claim{a}, []fluffycore_contracts_common.Claim{a, b, a2}))
	require.Empty(t, ClaimsAddedSince([]fluffycore_contracts_common.Claim{a}, []fluffycore_contracts_common.Claim{a}))
}

// A JWT carrying "mtls_verified": true must not satisfy an mTLS requirement.
func TestClaimsPrincipalFromToken_ReservedClaimsStripped(t *testing.T) {
	scratch := ClaimsPrincipalFromClaimsMap(map[string]interface{}{"sub": "u", "mtls_verified": true})
	p := NewIClaimsPrincipal()
	p.AddClaim(WithoutReservedMTLSClaims(scratch.GetClaims())...)
	require.True(t, p.HasClaimType("sub"))
	require.False(t, p.HasClaimType(fluffycore_wellknown.ClaimTypeMTLSVerified))
}
