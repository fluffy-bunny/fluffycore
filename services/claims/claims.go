package claims

import (
	fluffycore_contracts_claims "github.com/fluffy-bunny/fluffycore/contracts/claims"
	jwt "github.com/golang-jwt/jwt/v4"
)

type (
	Claims map[string]interface{}
	Claim  struct {
		claimType  string
		claimValue interface{}
	}
)

func NewClaims() fluffycore_contracts_claims.IClaims {
	return &Claims{}
}

func (c *Claim) Type() string {
	return c.claimType
}
func (c *Claim) Value() interface{} {
	return c.claimValue
}

func NewClaim(claimType string, claimValue interface{}) fluffycore_contracts_claims.IClaim {
	return &Claim{
		claimType:  claimType,
		claimValue: claimValue,
	}
}

// Valid claims verification
func (a *Claims) Valid() error {
	return nil
}
func (a *Claims) Claims() fluffycore_contracts_claims.Claims {
	claims := make(fluffycore_contracts_claims.Claims, 0)
	for k, v := range *a {
		claims = append(claims, NewClaim(k, v))
	}
	return claims
}

func (a *Claims) Set(key string, value interface{}) error {
	(*a)[key] = value
	return nil
}
func (a *Claims) Delete(key string) error {
	delete(*a, key)
	return nil
}
func (a *Claims) Get(key string) interface{} {
	return (*a)[key]
}
func (a *Claims) JwtClaims() jwt.Claims {
	return a
}
