package client_credentials

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_tokensource "github.com/fluffy-bunny/fluffycore/contracts/tokensource"
	oauth2 "golang.org/x/oauth2"
	cc "golang.org/x/oauth2/clientcredentials"
)

type (
	service struct {
		tokenSource oauth2.TokenSource
		config      *fluffycore_contracts_tokensource.AppTokenSourceConfig
	}
)

var stemService = (*service)(nil)

var _ fluffycore_contracts_tokensource.IAppTokenSource = stemService

func (s *service) Ctor(config *fluffycore_contracts_tokensource.AppTokenSourceConfig) (fluffycore_contracts_tokensource.IAppTokenSource, error) {

	clientId := config.ClientID
	clientSecret := config.ClientSecret
	tokenUrl := config.TokenURL
	scopes := config.Scopes // space separated

	ccConfig := &cc.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		TokenURL:     tokenUrl,
		Scopes:       scopes,
	}

	tokenSource := ccConfig.TokenSource(context.Background())
	return &service{
		config:      config,
		tokenSource: tokenSource,
	}, nil
}

// AddSingletonIAppTokenSource ...
func AddSingletonIAppTokenSource(builder di.ContainerBuilder, config *fluffycore_contracts_tokensource.AppTokenSourceConfig) {
	di.AddInstance[*fluffycore_contracts_tokensource.AppTokenSourceConfig](builder, config)
	di.AddSingleton[fluffycore_contracts_tokensource.IAppTokenSource](builder, stemService.Ctor)
}

func (s *service) GetTokenSource() (oauth2.TokenSource, error) {
	return s.tokenSource, nil
}
