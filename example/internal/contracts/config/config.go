package config

import (
	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
	fluffycore_contracts_ddprofiler "github.com/fluffy-bunny/fluffycore/contracts/ddprofiler"
)

type (
	JWTValidators struct {
		Issuers  []string `json:"issuers" mapstructure:"ISSUERS"`
		JWKSURLS []string `json:"jwksUrls" mapstructure:"JWKS_URLS"`
	}
	ConfigFiles struct {
		ClientPath string `json:"clientPath"`
	}
)
type Config struct {
	ConfigFiles                            ConfigFiles `json:"configFiles"`
	fluffycore_contracts_config.CoreConfig `mapstructure:",squash"`
	CustomString                           string                                  `json:"customString"`
	SomeSecret                             string                                  `json:"someSecret" redact:"true"`
	OAuth2Port                             int                                     `json:"oauth2Port"`
	JWTValidators                          JWTValidators                           `json:"jwtValidators"`
	DDProfilerConfig                       *fluffycore_contracts_ddprofiler.Config `json:"ddProfilerConfig"`
}

// ConfigDefaultJSON default json
var ConfigDefaultJSON = []byte(`
{
	"APPLICATION_NAME": "in-environment",
	"APPLICATION_ENVIRONMENT": "in-environment",
	"PRETTY_LOG": false,
	"LOG_LEVEL": "info",
	"PORT": 50051,
	"REST_PORT": 50052,
	"oauth2Port": 50053,
	"customString": "some default value",
	"someSecret": "password",
	"GRPC_GATEWAY_ENABLED": true,
	"jwtValidators": {},
	"configFiles": {
		"clientPath": "./config/clients.json"
	},
	"ddProfilerConfig": {
		"ENABLED": false,
		"SERVICE_NAME": "in-environment",
		"APPLICATION_ENVIRONMENT": "in-environment",
		"VERSION": "1.0.0"
	}

  }
`)
