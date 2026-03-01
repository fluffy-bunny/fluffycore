package claimsprincipal

import (
	"testing"

	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
)

// mockClaimFact implements IClaimFact for testing
type mockClaimFactMedium struct {
	hasClaim bool
	expr     string
}

func (m *mockClaimFactMedium) HasClaim(cp fluffycore_contracts_common.IClaimsPrincipal) bool {
	return m.hasClaim
}
func (m *mockClaimFactMedium) Expression() string {
	return m.expr
}

// mockClaimsPrincipal implements IClaimsPrincipal for testing
type mockClaimsPrincipalMedium struct{}

func (m *mockClaimsPrincipalMedium) GetClaims() []fluffycore_contracts_common.Claim {
	return nil
}
func (m *mockClaimsPrincipalMedium) HasClaim(claim fluffycore_contracts_common.Claim) bool {
	return false
}
func (m *mockClaimsPrincipalMedium) AddClaim(claims ...fluffycore_contracts_common.Claim) {
}
func (m *mockClaimsPrincipalMedium) RemoveClaim(claims ...fluffycore_contracts_common.Claim) {
}
func (m *mockClaimsPrincipalMedium) RemoveClaimType(claimTypes ...string) {
}
func (m *mockClaimsPrincipalMedium) GetClaimsByType(claimType string) []fluffycore_contracts_common.Claim {
	return nil
}
func (m *mockClaimsPrincipalMedium) HasClaimType(claimType string) bool {
	return false
}

func TestClaimsAST_InvalidOperand_Validate_Panics(t *testing.T) {
	ast := &ClaimsAST{
		ClaimFacts: []fluffycore_contracts_common.IClaimFact{
			&mockClaimFactMedium{hasClaim: true, expr: "test"},
		},
	}
	cp := &mockClaimsPrincipalMedium{}

	// Invalid operand should panic instead of calling log.Fatal
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic for invalid operand, but none occurred")
		}
	}()
	ast.ValidateWithOperand(cp, fluffycore_contracts_common.Operand(99))
}

func TestClaimsAST_InvalidOperand_String_Panics(t *testing.T) {
	ast := &ClaimsAST{
		ClaimFacts: []fluffycore_contracts_common.IClaimFact{
			&mockClaimFactMedium{hasClaim: true, expr: "test"},
		},
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic for invalid operand in String, but none occurred")
		}
	}()
	ast.StringWithOperand(fluffycore_contracts_common.Operand(99))
}

func TestClaimsAST_ValidOperands_DontPanic(t *testing.T) {
	ast := &ClaimsAST{
		ClaimFacts: []fluffycore_contracts_common.IClaimFact{
			&mockClaimFactMedium{hasClaim: true, expr: "test"},
		},
	}
	cp := &mockClaimsPrincipalMedium{}

	// AND operand should not panic
	result := ast.ValidateWithOperand(cp, and)
	if !result {
		t.Error("expected true for AND with hasClaim=true")
	}

	// OR operand should not panic
	result = ast.ValidateWithOperand(cp, or)
	if !result {
		t.Error("expected true for OR with hasClaim=true")
	}

	// String with valid operands
	s := ast.StringWithOperand(and)
	if s == "" {
		t.Error("expected non-empty string for AND")
	}
	s = ast.StringWithOperand(or)
	if s == "" {
		t.Error("expected non-empty string for OR")
	}
}
