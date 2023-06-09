package config

import (
	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
)

type Config struct {
	fluffycore_contracts_config.CoreConfig `mapstructure:",squash"`
	CustomString                           string `mapstructure:"CUSTOM_STRING"`
	SomeSecret                             string `mapstructure:"SOME_SECRET" redact:"true"`
}

// ConfigDefaultJSON default json
var ConfigDefaultJSON = []byte(`
{
	"APPLICATION_NAME": "in-environment",
	"APPLICATION_ENVIRONMENT": "in-environment",
	"PRETTY_LOG": false,
	"LOG_LEVEL": "info",
	"PORT": 1111,
	"CUSTOM_STRING": "some default value",
	"SOME_SECRET": "password"
  }
`)
