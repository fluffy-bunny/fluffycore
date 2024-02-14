package claims

import (
	jwt "github.com/golang-jwt/jwt/v4"
)

type (
	IClaim interface {
		Type() string
		Value() interface{}
	}
	Claims  []IClaim
	IClaims interface {
		Valid() error
		Set(key string, value interface{}) error
		Delete(key string) error
		Get(key string) interface{}
		JwtClaims() jwt.Claims
		Claims() Claims
	}
)
