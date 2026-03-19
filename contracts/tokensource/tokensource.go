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
	AppTokenSourceConfig struct {
		ClientID     string   `json:"client_id"`
		ClientSecret string   `json:"client_secret"`
		TokenURL     string   `json:"token_url"`
		Scopes       []string `json:"scopes"`
	}
)
