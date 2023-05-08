package jwt

import (
	jwxt "github.com/lestrrat-go/jwx/v2/jwt"
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
	val, ok := innerToken.Get("permissions")
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

	return &jwtToken{
		InnerToken: innerToken,
		ID:         innerToken.Subject(),
		Claims:     innerToken.PrivateClaims(),
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
