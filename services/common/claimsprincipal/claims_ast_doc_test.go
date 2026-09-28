package claimsprincipal

import (
	"testing"

	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	"github.com/stretchr/testify/require"
)

// Pins the ClaimsAST doc example. The doc used to claim the two Or siblings were OR'd
// together; they are combined by the parent's operator (AND).
func TestClaimsAST_DocExample(t *testing.T) {
	facts := func(vals ...string) []fluffycore_contracts_common.IClaimFact {
		var out []fluffycore_contracts_common.IClaimFact
		for _, v := range vals {
			out = append(out, NewClaimFact(fluffycore_contracts_common.Claim{Type: "permissions", Value: v}))
		}
		return out
	}
	ast := ClaimsAST{
		ClaimFacts: facts("A", "B"),
		Or: []fluffycore_contracts_common.IClaimsValidator{
			&ClaimsAST{ClaimFacts: facts("C", "D")},
			&ClaimsAST{
				ClaimFacts: facts("E", "F"),
				And:        []fluffycore_contracts_common.IClaimsValidator{&ClaimsAST{ClaimFacts: facts("G", "H")}},
			},
		},
		Not: []fluffycore_contracts_common.IClaimsValidator{
			&ClaimsAST{Or: []fluffycore_contracts_common.IClaimsValidator{&ClaimsAST{ClaimFacts: facts("I", "J")}}},
		},
	}
	doc := func(has map[string]bool) bool { // A && B && (C || D) && (E || F || (G && H)) && !(I || J)
		return has["A"] && has["B"] && (has["C"] || has["D"]) && (has["E"] || has["F"] || (has["G"] && has["H"])) && !(has["I"] || has["J"])
	}
	all := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	for mask := 0; mask < 1<<len(all); mask++ {
		var held []string
		has := map[string]bool{}
		for i, v := range all {
			if mask&(1<<i) != 0 {
				held = append(held, v)
				has[v] = true
			}
		}
		require.Equal(t, doc(has), ast.Validate(NewmockClaimsPrincipalToken(held...)), "claims %v", held)
	}
}
