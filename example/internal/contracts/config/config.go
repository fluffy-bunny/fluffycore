package config

import (
	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
)

type Config struct {
	fluffycore_contracts_config.CoreConfig `mapstructure:",squash"`
	CustomString                           string `mapstructure:"CUSTOM_STRING"`
	SomeSecret                             string `mapstructure:"SOME_SECRET" redact:"true"`
	OAuth2Port                             int    `mapstructure:"OAuth2Port"`
}

// ConfigDefaultJSON default json
var ConfigDefaultJSON = []byte(`
{
	"APPLICATION_NAME": "in-environment",
	"APPLICATION_ENVIRONMENT": "in-environment",
	"PRETTY_LOG": false,
	"LOG_LEVEL": "info",
	"PORT": 1111,
	"OAuth2Port": 1113,
	"CUSTOM_STRING": "some default value",
	"SOME_SECRET": "password",
	"REST_PORT": 50052,
	"GRPC_GATEWAY_ENABLED": true

  }
`)
