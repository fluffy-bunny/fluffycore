package claimsast

import (
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
)

// newClaimFactType wraps claimsprincipal.NewClaimFactType, keeping the
// claimsprincipal import out of the main claimsast.go file.
func newClaimFactType(claimTypeName string) fluffycore_contracts_common.IClaimFact {
	return claimsprincipal.NewClaimFactType(claimTypeName)
}

// newClaimFact wraps claimsprincipal.NewClaimFact.
func newClaimFact(claimTypeName, value string) fluffycore_contracts_common.IClaimFact {
	return claimsprincipal.NewClaimFact(fluffycore_contracts_common.Claim{
		Type:  claimTypeName,
		Value: value,
	})
}
