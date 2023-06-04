package common

type (
	Operand int

	IClaimsValidator interface {
		Validate(claimsPrincipal IClaimsPrincipal) bool
		ValidateWithOperand(claimsPrincipal IClaimsPrincipal, op Operand) bool
		String() string
		StringWithOperand(op Operand) string
	}
)
