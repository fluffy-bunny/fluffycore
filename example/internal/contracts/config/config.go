package config

import (
	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
)

type Config struct {
	fluffycore_contracts_config.CoreConfig `mapstructure:",squash"`
	CustomString                           string `json:"CUSTOM_STRING" mapstructure:"CUSTOM_STRING"`
	SomeSecret                             string `json:"SOME_SECRET" mapstructure:"SOME_SECRET" redact:"true"`
	OAuth2Port                             int    `json:"oauth2Port"  mapstructure:"OAUTH2_PORT"`
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
	"OAUTH2_PORT": 50053,
	"CUSTOM_STRING": "some default value",
	"SOME_SECRET": "password",
	"GRPC_GATEWAY_ENABLED": true

  }
`)
