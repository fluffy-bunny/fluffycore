package common

const (
	// ClaimTypeAndValue ...
	ClaimTypeAndValue ClaimFactDirective = 0
	// ClaimType ...
	ClaimType = 1
)

type (
	ClaimFactDirective int64

	// Claim ...
	Claim struct {
		Type  string `json:"type" mapstructure:"TYPE"`
		Value string `json:"value" mapstructure:"VALUE"`
	}
	// IClaimsPrincipal interface
	IClaimsPrincipal interface {
		GetClaims() []Claim
		HasClaim(claim Claim) bool
		AddClaim(claim ...Claim)
		RemoveClaim(claim ...Claim)
		GetClaimsByType(claimType string) []Claim
		HasClaimType(claimType string) bool
	}
)
