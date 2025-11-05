package client_credentials

import (
	"context"
	"os"
	"strings"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_tokensource "github.com/fluffy-bunny/fluffycore/contracts/tokensource"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	oauth2 "golang.org/x/oauth2"
	cc "golang.org/x/oauth2/clientcredentials"
)

type (
	service struct {
		tokenSource oauth2.TokenSource
	}
)

var stemService = (*service)(nil)

var _ fluffycore_contracts_tokensource.IAppTokenSource = stemService

func (s *service) Ctor() (fluffycore_contracts_tokensource.IAppTokenSource, error) {
	clientId := os.Getenv("FLUFFYCORE_APP_OAUTH2_CLIENT_ID")
	clientSecret := os.Getenv("FLUFFYCORE_APP_OAUTH2_CLIENT_SECRET")
	tokenUrl := os.Getenv("FLUFFYCORE_APP_OAUTH2_TOKEN_URL")
	scopes := os.Getenv("FLUFFYCORE_APP_OAUTH2_SCOPES") // space separated
	// parse scopes a,b,c
	sliceScopes := []string{}
	if fluffycore_utils.IsNotEmptyOrNil(scopes) {
		for _, scope := range strings.Split(scopes, " ") {
			trimmed := strings.TrimSpace(scope)
			if fluffycore_utils.IsNotEmptyOrNil(trimmed) {
				sliceScopes = append(sliceScopes, trimmed)
			}
		}
	}
	config := &cc.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		TokenURL:     tokenUrl,
		Scopes:       sliceScopes,
	}

	tokenSource := config.TokenSource(context.Background())
	return &service{tokenSource: tokenSource}, nil
}

// AddSingletonIAppTokenSource ...
func AddSingletonIAppTokenSource(builder di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_tokensource.IAppTokenSource](builder, stemService.Ctor)
}

func (s *service) GetTokenSource() (oauth2.TokenSource, error) {
	return s.tokenSource, nil
}
