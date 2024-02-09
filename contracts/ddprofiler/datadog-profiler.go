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
		Enabled                bool   `json:"enabled"`
		ServiceName            string `json:"serviceName"`
		ApplicationEnvironment string `json:"applicationEnvironment"`
		Version                string `json:"version"`
	}
)
