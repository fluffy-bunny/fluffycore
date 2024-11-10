package claims

import (
	"fmt"
	"time"

	fluffycore_contracts_claims "github.com/fluffy-bunny/fluffycore/contracts/claims"
	jwt "github.com/golang-jwt/jwt/v5"
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

func (a *Claims) GetAudience() (jwt.ClaimStrings, error) {
	aud, ok := (*a)["aud"].([]string)
	if !ok {
		return nil, fmt.Errorf("audience is not a string array")
	}
	return aud, nil
}

func (a *Claims) GetExpirationTime() (*jwt.NumericDate, error) {
	exp, ok := (*a)["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("expiration time is not a float64")
	}

	expTime := time.Unix(int64(exp), 0)
	return &jwt.NumericDate{
		Time: expTime,
	}, nil
}
func (a *Claims) GetIssuedAt() (*jwt.NumericDate, error) {
	iat, ok := (*a)["iat"].(float64)
	if !ok {
		return nil, fmt.Errorf("issued at time is not a float64")
	}

	iatTime := time.Unix(int64(iat), 0)
	return &jwt.NumericDate{
		Time: iatTime,
	}, nil
}
func (a *Claims) GetNotBefore() (*jwt.NumericDate, error) {
	nbf, ok := (*a)["nbf"].(float64)
	if !ok {
		return nil, fmt.Errorf("not before time is not a float64")
	}

	nbfTime := time.Unix(int64(nbf), 0)
	return &jwt.NumericDate{
		Time: nbfTime,
	}, nil
}
func (a *Claims) GetIssuer() (string, error) {
	iss, ok := (*a)["iss"].(string)
	if !ok {
		return "", fmt.Errorf("issuer is not a string")
	}
	return iss, nil
}
func (a *Claims) GetSubject() (string, error) {
	sub, ok := (*a)["sub"].(string)
	if !ok {
		return "", fmt.Errorf("subject is not a string")
	}
	return sub, nil
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
