package ddprofiler

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
)

type (
	// IProfiler abstraction, i.e. datadog profiler
	IDataDogProfiler interface {
		Start(ctx context.Context)
		Stop(ctx context.Context)
	}
	DDProfilerConfig struct {
		Enabled bool `json:"enabled"`
	}
	Config struct {
		TracingEnabled         bool              `json:"tracingEnabled"`
		DDProfilerConfig       *DDProfilerConfig `json:"ddProfilerConfig"`
		ServiceName            string            `json:"serviceName"`
		ApplicationEnvironment string            `json:"applicationEnvironment"`
		Version                string            `json:"version"`
	}
)

func AddDDConfig(builder di.ContainerBuilder, config *Config) {
	di.AddInstance[*Config](builder, config)
}
