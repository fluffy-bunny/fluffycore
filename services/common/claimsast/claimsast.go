// Package claimsast provides a readable Go DSL for composing ClaimsAST
// validation expressions. It mirrors the expression string output of
// ClaimsAST.String() directly in code.
//
// Example — the raw ClaimsAST struct form:
//
//	&ClaimsAST{Or: []IClaimsValidator{
//	    &ClaimsAST{And: []IClaimsValidator{
//	        &ClaimsAST{ClaimFacts: []IClaimFact{NewClaimFactType("org")}},
//	        &ClaimsAST{ClaimFacts: []IClaimFact{
//	            NewClaimFact(Claim{Type:"permissions", Value:"UsageMetrics.Read"}),
//	            NewClaimFact(Claim{Type:"permissions", Value:"UsageMetrics.ReadWrite"}),
//	        }},
//	    }},
//	}}
//
// Becomes:
//
//	Root(Or(
//	    And(
//	        Leaf(claimTypeOrgFact),
//	        Or(Leaf(readFact), Leaf(readWriteFact)),
//	    ),
//	))
package claimsast

import (
	"strings"

	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
)

// or is the OR operand value matching the one inside claims_ast.go
const orOperand fluffycore_contracts_common.Operand = 2

// ─── or node ─────────────────────────────────────────────────────────────────

type orNode struct {
	children []fluffycore_contracts_common.IClaimsValidator
}

// Or returns a validator that passes when at least one child passes.
func Or(children ...fluffycore_contracts_common.IClaimsValidator) fluffycore_contracts_common.IClaimsValidator {
	return &orNode{children: children}
}

func (n *orNode) Validate(p fluffycore_contracts_common.IClaimsPrincipal) bool {
	for _, c := range n.children {
		if c.Validate(p) {
			return true
		}
	}
	return false
}

// ValidateWithOperand ignores the parent operand and always evaluates as OR.
func (n *orNode) ValidateWithOperand(p fluffycore_contracts_common.IClaimsPrincipal, _ fluffycore_contracts_common.Operand) bool {
	return n.Validate(p)
}

func (n *orNode) String() string {
	return n.StringWithOperand(orOperand)
}

func (n *orNode) StringWithOperand(_ fluffycore_contracts_common.Operand) string {
	parts := make([]string, len(n.children))
	for i, c := range n.children {
		parts[i] = c.String()
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

// ─── and node ────────────────────────────────────────────────────────────────

type andNode struct {
	children []fluffycore_contracts_common.IClaimsValidator
}

// And returns a validator that passes only when all children pass.
func And(children ...fluffycore_contracts_common.IClaimsValidator) fluffycore_contracts_common.IClaimsValidator {
	return &andNode{children: children}
}

func (n *andNode) Validate(p fluffycore_contracts_common.IClaimsPrincipal) bool {
	for _, c := range n.children {
		if !c.Validate(p) {
			return false
		}
	}
	return true
}

// ValidateWithOperand ignores the parent operand and always evaluates as AND.
func (n *andNode) ValidateWithOperand(p fluffycore_contracts_common.IClaimsPrincipal, _ fluffycore_contracts_common.Operand) bool {
	return n.Validate(p)
}

func (n *andNode) String() string {
	parts := make([]string, len(n.children))
	for i, c := range n.children {
		parts[i] = c.String()
	}
	return "(" + strings.Join(parts, " AND ") + ")"
}

func (n *andNode) StringWithOperand(_ fluffycore_contracts_common.Operand) string {
	return n.String()
}

// ─── not node ────────────────────────────────────────────────────────────────

type notNode struct {
	child fluffycore_contracts_common.IClaimsValidator
}

// Not returns a validator that inverts the result of child.
func Not(child fluffycore_contracts_common.IClaimsValidator) fluffycore_contracts_common.IClaimsValidator {
	return &notNode{child: child}
}

func (n *notNode) Validate(p fluffycore_contracts_common.IClaimsPrincipal) bool {
	return !n.child.Validate(p)
}

func (n *notNode) ValidateWithOperand(p fluffycore_contracts_common.IClaimsPrincipal, _ fluffycore_contracts_common.Operand) bool {
	return n.Validate(p)
}

func (n *notNode) String() string {
	return "!(" + n.child.String() + ")"
}

func (n *notNode) StringWithOperand(_ fluffycore_contracts_common.Operand) string {
	return n.String()
}

// ─── leaf node ───────────────────────────────────────────────────────────────

type leafNode struct {
	fact fluffycore_contracts_common.IClaimFact
}

// Leaf wraps a single IClaimFact as a validator node.
func Leaf(fact fluffycore_contracts_common.IClaimFact) fluffycore_contracts_common.IClaimsValidator {
	return &leafNode{fact: fact}
}

// ClaimType returns a validator that passes when the principal has a claim with
// the given type (value is not checked).
func ClaimType(claimTypeName string) fluffycore_contracts_common.IClaimsValidator {
	return Leaf(newClaimFactType(claimTypeName))
}

// ClaimTypeAndValue returns a validator that passes when the principal has a
// claim matching both the given type and value.
func ClaimTypeAndValue(claimTypeName, value string) fluffycore_contracts_common.IClaimsValidator {
	return Leaf(newClaimFact(claimTypeName, value))
}

func (n *leafNode) Validate(p fluffycore_contracts_common.IClaimsPrincipal) bool {
	return n.fact.HasClaim(p)
}

func (n *leafNode) ValidateWithOperand(p fluffycore_contracts_common.IClaimsPrincipal, _ fluffycore_contracts_common.Operand) bool {
	return n.Validate(p)
}

func (n *leafNode) String() string {
	return n.fact.Expression()
}

func (n *leafNode) StringWithOperand(_ fluffycore_contracts_common.Operand) string {
	return n.String()
}
