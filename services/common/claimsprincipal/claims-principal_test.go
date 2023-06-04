package claimsprincipal

import (
	"testing"

	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	"github.com/stretchr/testify/assert"
)

func Test_add_bad_claim(t *testing.T) {
	cp := NewIClaimsPrincipal()
	cp.AddClaim(fluffycore_contracts_common.Claim{
		Type:  "",
		Value: "",
	})
	assert.Empty(t, cp.GetClaims())
}
func Test_add_claim(t *testing.T) {
	cp := NewIClaimsPrincipal()
	claim := fluffycore_contracts_common.Claim{
		Type:  "a",
		Value: "b",
	}
	cp.AddClaim(claim)
	claims := cp.GetClaims()
	assert.NotEmpty(t, claims)
	assert.Equal(t, 1, len(claims))
	assert.True(t, cp.HasClaim(claim))
	assert.False(t, cp.HasClaim(fluffycore_contracts_common.Claim{
		Type:  "junk",
		Value: "junk",
	}))

	cp.RemoveClaim(claim)
	claims = cp.GetClaims()
	assert.Empty(t, claims)
}
