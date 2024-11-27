package config

type (
	CoreConfig struct {
		ApplicationEnvironment string `json:"applicationEnvironment" mapstructure:"APPLICATION_ENVIRONMENT"`
		ApplicationName        string `json:"applicationName" mapstructure:"APPLICATION_NAME"`
		PORT                   int    `json:"port" mapstructure:"PORT"`
		GRPCGateWayEnabled     bool   `json:"grpcGateWayEnabled" mapstructure:"GRPC_GATEWAY_ENABLED"`
		RESTPort               int    `json:"restPort" mapstructure:"REST_PORT"`
		PrettyLog              bool   `json:"prettyLog" mapstructure:"PRETTY_LOG"`
		LogLevel               string `json:"logLevel" mapstructure:"LOG_LEVEL"`
		NATSEnabled            bool   `json:"enableNats" mapstructure:"NATS_ENABLED"`
	}
)
