package claimsast

import (
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
)

// Root wraps a DSL validator in a *ClaimsAST shell, which is what
// EntryPointConfig requires. The single Or[] child means the shell
// passes through to the validator using the OR operand, which every
// DSL node ignores — so the validator evaluates with its own semantics.
func Root(v fluffycore_contracts_common.IClaimsValidator) *claimsprincipal.ClaimsAST {
	return &claimsprincipal.ClaimsAST{
		Or: []fluffycore_contracts_common.IClaimsValidator{v},
	}
}
