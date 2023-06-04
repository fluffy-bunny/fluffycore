package common

type (
	// IClaimFact interface
	IClaimFact interface {
		HasClaim(claimsprincipal IClaimsPrincipal) bool
		Expression() string
	}
	IClaimsAST interface {
		GetClaimsFact() []IClaimFact
		GetAndClaimsValidator() []IClaimsValidator
		GetNotClaimsValidator() []IClaimsValidator
		GetOrClaimsValidator() []IClaimsValidator
		Validate(claimsPrincipal IClaimsPrincipal) bool
	}

	IEntryPointConfig interface {
		GetFullMethodName() string
		GetClaimsAST() IClaimsAST
	}
	GetEntryPointConfigs func() map[string]IEntryPointConfig
)
