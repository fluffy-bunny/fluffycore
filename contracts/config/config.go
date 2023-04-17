package config

type (
	CoreConfig struct {
		ApplicationEnvironment string `json:"applicationEnvironment" mapstructure:"APPLICATION_ENVIRONMENT"`
		APPLICATION_NAME       string `json:"applicationName" mapstructure:"APPLICATION_NAME"`
		PORT                   int    `json:"port" mapstructure:"PORT"`
		PrettyLog              bool   `json:"prettyLog" mapstructure:"PRETTY_LOG"`
		LogLevel               string `json:"logLevel" mapstructure:"LOG_LEVEL"`
	}
)
