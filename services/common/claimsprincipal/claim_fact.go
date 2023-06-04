package claimsprincipal

import (
	"fmt"

	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
)

type (
	// Directive tells if we want only the type validated vs type and value
	Directive int64
)

const (
	// ClaimTypeAndValue ...
	ClaimTypeAndValue Directive = 0
	// ClaimType ...
	ClaimType = 1
)

// ClaimFact used for authorization
type ClaimFact struct {
	Claim     fluffycore_contracts_common.Claim
	Directive Directive
}

func init() {
	var _ fluffycore_contracts_common.IClaimFact = &ClaimFact{}
}

// NewClaimFact ...
func NewClaimFact(claim fluffycore_contracts_common.Claim) fluffycore_contracts_common.IClaimFact {
	return &ClaimFact{
		Claim:     claim,
		Directive: ClaimTypeAndValue,
	}
}
func NewClaimFactType(claimType string) fluffycore_contracts_common.IClaimFact {
	return &ClaimFact{
		Claim: fluffycore_contracts_common.Claim{
			Type: claimType,
		},
		Directive: fluffycore_contracts_common.ClaimType,
	}
}

// HasClaim ...
func (s *ClaimFact) HasClaim(claimsPrincipal fluffycore_contracts_common.IClaimsPrincipal) bool {
	if s.Directive == fluffycore_contracts_common.ClaimType {
		return claimsPrincipal.HasClaimType(s.Claim.Type)
	}
	return claimsPrincipal.HasClaim(s.Claim)
}

const (
	// ClaimTypeAndValueExpression ...
	ClaimTypeAndValueExpression = "has_claim(%s|%s)"
	// ClaimTypeExpression ...
	ClaimTypeExpression = "has_claim_type(%s)"
)

// Expression ...
func (s *ClaimFact) Expression() string {
	if s.Directive == fluffycore_contracts_common.ClaimType {
		return fmt.Sprintf(ClaimTypeExpression, s.Claim.Type)
	}
	return fmt.Sprintf(ClaimTypeAndValueExpression, s.Claim.Type, s.Claim.Value)
}
