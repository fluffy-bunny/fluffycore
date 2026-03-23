package config

import (
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
	fluffycore_contracts_ddprofiler "github.com/fluffy-bunny/fluffycore/contracts/ddprofiler"
	fluffycore_contracts_otel "github.com/fluffy-bunny/fluffycore/contracts/otel"
	fluffycore_nats_micro_service "github.com/fluffy-bunny/fluffycore/nats/nats_micro_service"
)

type (
	JWTValidators struct {
		Issuers  []string `json:"issuers"`
		JWKSURLS []string `json:"jwksUrls"`
	}
	ConfigFiles struct {
		ClientPath string `json:"clientPath"`
	}
)

// Config is the v1 config struct. Kept for backward compatibility.
type Config struct {
	fluffycore_contracts_config.CoreConfig `mapstructure:",squash"`

	ConfigFiles     ConfigFiles                                    `json:"configFiles"`
	CustomString    string                                         `json:"customString"`
	SomeSecret      string                                         `json:"someSecret" redact:"true"`
	OAuth2Port      int                                            `json:"oauth2Port"`
	JWTValidators   JWTValidators                                  `json:"jwtValidators"`
	DDConfig        *fluffycore_contracts_ddprofiler.Config        `json:"ddConfig"`
	OTELConfig      *fluffycore_contracts_otel.OTELConfig          `json:"otelConfig"`
	NATSMicroConfig *fluffycore_nats_micro_service.NATSMicroConfig `json:"natsMicroConfig"`
}

func AddConfig(builder di.ContainerBuilder, config *Config) {
	di.AddInstance[*Config](builder, config)
}

// ConfigV2 is the v2 config struct. All value types, no pointers.
// Defaults are set in NewDefaultConfigV2() instead of ConfigDefaultJSON.
type ConfigV2 struct {
	fluffycore_contracts_config.CoreConfig

	ConfigFiles     ConfigFiles                                   `json:"configFiles"`
	CustomString    string                                        `json:"customString"`
	SomeSecret      string                                        `json:"someSecret" redact:"true"`
	OAuth2Port      int                                           `json:"oauth2Port"`
	JWTValidators   JWTValidators                                 `json:"jwtValidators"`
	DDConfig        fluffycore_contracts_ddprofiler.Config        `json:"ddConfig"`
	OTELConfig      fluffycore_contracts_otel.OTELConfig          `json:"otelConfig"`
	NATSMicroConfig fluffycore_nats_micro_service.NATSMicroConfig `json:"natsMicroConfig"`
}

func AddConfigV2(builder di.ContainerBuilder, config *ConfigV2) {
	di.AddInstance[*ConfigV2](builder, config)
}

// NewDefaultConfigV2 returns the config with all Go-code defaults.
// No JSON blob needed — sparse JSON overlays and env vars can override these.
func NewDefaultConfigV2() ConfigV2 {
	return ConfigV2{
		CoreConfig: fluffycore_contracts_config.CoreConfig{
			ApplicationName:        "in-environment",
			ApplicationEnvironment: "in-environment",
			PrettyLog:              false,
			LogLevel:               "info",
			PORT:                   50051,
			RESTPort:               50052,
			GRPCGateWayEnabled:     true,
		},
		OAuth2Port:   50053,
		CustomString: "some default value",
		SomeSecret:   "password",
		ConfigFiles: ConfigFiles{
			ClientPath: "./config/clients.json",
		},
		DDConfig: fluffycore_contracts_ddprofiler.Config{
			DDProfilerConfig: &fluffycore_contracts_ddprofiler.DDProfilerConfig{
				Enabled: false,
			},
			TracingEnabled:         false,
			ServiceName:            "in-environment",
			ApplicationEnvironment: "in-environment",
			Version:                "1.0.0",
		},
		NATSMicroConfig: fluffycore_nats_micro_service.NATSMicroConfig{
			NATSUrl:         "nats://127.0.0.1:4222",
			ClientID:        "nats-micro-god",
			ClientSecret:    "secret",
			TimeoutDuration: "5s",
		},
		OTELConfig: fluffycore_contracts_otel.OTELConfig{
			ServiceName: "in-environment",
			TracingConfig: fluffycore_contracts_otel.TracingConfig{
				Enabled:      false,
				EndpointType: fluffycore_contracts_otel.STDOUT,
				Endpoint:     "localhost:4318",
			},
			MetricConfig: fluffycore_contracts_otel.MetricConfig{
				Enabled:         false,
				EndpointType:    fluffycore_contracts_otel.STDOUT,
				IntervalSeconds: 10,
				Endpoint:        "localhost:4318",
				RuntimeEnabled:  false,
				HostEnabled:     false,
			},
		},
	}
}

// ConfigDefaultJSON default json (v1 — kept for backward compatibility)
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
    "ddConfig": {
        "ddProfilerConfig": {
            "enabled": false
        },
        "tracingEnabled": false,
        "serviceName": "in-environment",
        "applicationEnvironment": "in-environment",
        "version": "1.0.0"
    },
    "natsMicroConfig":{
        "natsUrl": "nats://127.0.0.1:4222",
        "clientId": "nats-micro-god",
        "clientSecret": "secret",
        "timeoutDuration": "5s"
    },
    "otelConfig": {
        "serviceName": "in-environment",
        "tracingConfig": {
            "enabled": false,
            "endpointType": "stdout",
            "endpoint": "localhost:4318"
        },
        "metricConfig": {
            "enabled": false,
            "endpointType": "stdout",
            "intervalSeconds": 10,
            "endpoint": "localhost:4318",
            "runtimeEnabled": false,
            "hostEnabled": false
        } 
    }
}
	`)
