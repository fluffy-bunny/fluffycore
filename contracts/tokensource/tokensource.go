package tokensource

import (
	"golang.org/x/oauth2"
)

type (
	ITokenSource interface {
		GetTokenSource() (oauth2.TokenSource, error)
	}
	IAppTokenSource interface {
		ITokenSource
	}
)
