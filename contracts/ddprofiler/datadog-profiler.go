package ddprofiler

import "context"

//go:generate mockgen -package=$GOPACKAGE -destination=../mocks/$GOPACKAGE/mock_$GOFILE  github.com/fluffy-bunny/fluffycore/contracts/$GOPACKAGE IDataDogProfiler

type (
	// IProfiler abstraction, i.e. datadog profiler
	IDataDogProfiler interface {
		Start(ctx context.Context)
		Stop(ctx context.Context)
	}
	Config struct {
		Enabled                bool   `json:"enabled" mapstructure:"ENABLED"`
		ServiceName            string `json:"serviceName" mapstructure:"SERVICE_NAME"`
		ApplicationEnvironment string `json:"applicationEnvironment" mapstructure:"APPLICATION_ENVIRONMENT"`
		Version                string `json:"version" mapstructure:"VERSION"`
	}
)
