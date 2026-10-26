package jwt

import (
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	jwxt "github.com/lestrrat-go/jwx/v4/jwt"
)

type jwtToken struct {
	InnerToken jwxt.Token             `json:"innerToken"`
	ID         string                 `json:"id"`
	Claims     map[string]interface{} `json:"claims"`
	Bearer     string                 `json:"bearer"`
}

func newJWTToken(innerToken jwxt.Token, bearer string) *jwtToken {
	// Turn permissions in to a map
	pMap := make(map[string]bool)
	val, ok := innerToken.Field(fluffycore_wellknown.ClaimTypePermissions)
	if ok {
		perms, ok := val.([]interface{})
		if ok {
			for _, perm := range perms {
				key, ok := perm.(string)
				if ok {
					pMap[key] = true
				}
			}
		}
	}

	subject, _ := innerToken.Subject()

	return &jwtToken{
		InnerToken: innerToken,
		ID:         subject,
		Claims:     privateClaims(innerToken),
		Bearer:     bearer,
	}
}

func (t *jwtToken) AddClaim(key string, value interface{}) {
	t.Claims[key] = value
}
func (t *jwtToken) GetAccessToken() string {
	return t.Bearer
}

func (t *jwtToken) GetID() string {
	return t.ID
}

func (t *jwtToken) GetClaims() map[string]interface{} {
	return t.Claims
}

func (t *jwtToken) GetClaim(key string) interface{} {
	return t.Claims[key]
}

func (t *jwtToken) CheckClaim(key string, expected interface{}) bool {
	val, ok := t.Claims[key]
	if !ok {
		val = nil
	}

	return val == expected
}

func (t *jwtToken) GetInnerToken() interface{} {
	return t.InnerToken
}

// privateClaims returns a copy of the token's non-registered claims. v4's Token interface
// no longer has PrivateClaims(), and the concrete type's map is the live one (AddClaim
// writes into this map), so build a fresh one from the claim iterator.
func privateClaims(token jwxt.Token) map[string]interface{} {
	out := make(map[string]interface{})
	for name, value := range token.Claims() {
		switch name {
		case jwxt.IssuerKey, jwxt.SubjectKey, jwxt.AudienceKey, jwxt.ExpirationKey,
			jwxt.NotBeforeKey, jwxt.IssuedAtKey, jwxt.JwtIDKey:
			continue
		}
		out[name] = value
	}
	return out
}
