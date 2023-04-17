package config

import (
	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
)

type Config struct {
	fluffycore_contracts_config.CoreConfig `mapstructure:",squash"`
}

// ConfigDefaultJSON default json
var ConfigDefaultJSON = []byte(`
{
	"APPLICATION_NAME": "in-environment",
	"APPLICATION_ENVIRONMENT": "in-environment",
	"PRETTY_LOG": false,
	"LOG_LEVEL": "info",
	"PORT": 1111
  }
`)
