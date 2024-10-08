package claimsprincipal

import (
	"strings"

	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"

	"github.com/rs/zerolog/log"
)

const (
	and fluffycore_contracts_common.Operand = 1
	or  fluffycore_contracts_common.Operand = 2
)

// ClaimsAST is a light-weight AST that allows for logical collections of claims to
// be defined and tested by go grpc based services. Grouping is implicit in the tree's structure
// such that the root arrays form grouped AND operations, and branches are processed by
// their placement in the parent. For example:
// ```
//
//	ClaimsAST{
//		Values: []string{"A", "B"},
//		Or: []ClaimsAST{
//			{Values: []string{"C", "D"}},
//			{
//				Values: []string{"E", "F"},
//				And: []ClaimsAST{
//					{Values: []string{"G", "H"}},
//				},
//			},
//		},
//		Not: []ClaimsAST{
//			{
//				Or: []ClaimsAST{
//					{Values: []string{"I", "J"}},
//				},
//			},
//		},
//	}
//
// ```
//
// Is the equivalent to:
// if A && B && ((C || D) || (E || F || (G && H))) && !(I || J)
type ClaimsAST struct {
	ClaimFacts []fluffycore_contracts_common.IClaimFact

	And []fluffycore_contracts_common.IClaimsValidator
	Or  []fluffycore_contracts_common.IClaimsValidator
	Not []fluffycore_contracts_common.IClaimsValidator
}

func init() {
	var _ fluffycore_contracts_common.IClaimsAST = (*ClaimsAST)(nil)
}

func (p *ClaimsAST) AppendClaimsFact(claimFact ...fluffycore_contracts_common.IClaimFact) {
	p.ClaimFacts = append(p.ClaimFacts, claimFact...)
}

func (p *ClaimsAST) GetClaimsFact() []fluffycore_contracts_common.IClaimFact {
	return p.ClaimFacts
}
func (p *ClaimsAST) GetAndClaimsValidator() []fluffycore_contracts_common.IClaimsValidator {
	return p.And
}
func (p *ClaimsAST) GetNotClaimsValidator() []fluffycore_contracts_common.IClaimsValidator {
	return p.Not
}
func (p *ClaimsAST) GetOrClaimsValidator() []fluffycore_contracts_common.IClaimsValidator {
	return p.Or
}

// Validate the assumptions made in a Claims object
func (p *ClaimsAST) Validate(claimsPrincipal fluffycore_contracts_common.IClaimsPrincipal) bool {
	// Root is processed as an AND operation
	return p._validate(claimsPrincipal, and)
}

// ValidateWithOperand ...
func (p *ClaimsAST) ValidateWithOperand(claimsPrincipal fluffycore_contracts_common.IClaimsPrincipal, op fluffycore_contracts_common.Operand) bool {
	return p._validate(claimsPrincipal, op)
}

func (p *ClaimsAST) _validate(claimsPrincipal fluffycore_contracts_common.IClaimsPrincipal, op fluffycore_contracts_common.Operand) bool {
	switch op {
	case and:
		// Return false on the first false, true if everything is true

		// Values
		for _, val := range p.ClaimFacts {
			if !val.HasClaim(claimsPrincipal) {
				return false
			}
		}

		// Ands
		for _, andVal := range p.And {
			if !andVal.ValidateWithOperand(claimsPrincipal, and) {
				return false
			}
		}

		// Ors
		for _, orVal := range p.Or {
			if !orVal.ValidateWithOperand(claimsPrincipal, or) {
				return false
			}
		}

		// Nots - processed with our op, but negated (we are and an, so fail on true)
		for _, notVal := range p.Not {
			if notVal.ValidateWithOperand(claimsPrincipal, op) {
				return false
			}
		}

		// All good
		return true
	case or:
		// Return true on the first true, false if everything is false

		// Values
		for _, val := range p.ClaimFacts {
			if val.HasClaim(claimsPrincipal) {
				return true
			}
		}

		// Ands
		for _, andVal := range p.And {
			if andVal.ValidateWithOperand(claimsPrincipal, and) {
				return true
			}
		}

		// Ors
		for _, orVal := range p.Or {
			if orVal.ValidateWithOperand(claimsPrincipal, or) {
				return true
			}
		}

		// Nots - processed with our op, but negated (we are an or, so true on false)
		for _, notVal := range p.Not {
			if !notVal.ValidateWithOperand(claimsPrincipal, op) {
				return true
			}
		}

		// Nothing was true
		return false
	}

	log.Fatal().Int("op", int(op)).Msg("invalid operand")
	return false
}

// String ...
func (p *ClaimsAST) String() string {
	return p._string(and)
}

// StringWithOperand ...
func (p *ClaimsAST) StringWithOperand(op fluffycore_contracts_common.Operand) string {
	return p._string(op)
}
func (p *ClaimsAST) _string(op fluffycore_contracts_common.Operand) string {
	var groups []string

	// Values
	for _, claimFacts := range p.ClaimFacts {
		val := claimFacts.Expression()
		groups = append(groups, val)
	}

	// Ands
	for _, andVal := range p.And {
		groups = append(groups, andVal.StringWithOperand(and))
	}

	// Ors
	for _, orVal := range p.Or {
		groups = append(groups, orVal.StringWithOperand(or))
	}

	// Nots - processed with our op, but negated (we are and an, so fail on true)
	for _, notVal := range p.Not {
		groups = append(groups, "!"+notVal.StringWithOperand(op))
	}

	switch op {
	case and:
		return "(" + strings.Join(groups, " AND ") + ")"
	case or:
		return "(" + strings.Join(groups, " OR ") + ")"
	}

	log.Fatal().Int("op", int(op)).Msg("invalid operand")
	return ""
}
