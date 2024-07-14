package otel

import "context"

//go:generate mockgen -package=$GOPACKAGE -destination=../mocks/$GOPACKAGE/mock_$GOFILE  github.com/fluffy-bunny/fluffycore/contracts/$GOPACKAGE IOpenTelemetry

type EndpointType string

const (
	STDOUT EndpointType = "stdout"
	HTTP   EndpointType = "http"
	GRPC   EndpointType = "grpc"
)

type (
	IOpenTelemetry interface {
		Start(ctx context.Context)
		Stop(ctx context.Context)
	}
	TracingConfig struct {
		Enabled      bool         `json:"enabled"`
		EndpointType EndpointType `json:"endpointType"`
		Endpoint     string       `json:"endpoint"`
	}
	MetricConfig struct {
		Enabled         bool         `json:"enabled"`
		EndpointType    EndpointType `json:"endpointType"`
		Endpoint        string       `json:"endpoint"`
		IntervalSeconds int          `json:"intervalSeconds"`
	}
	OTELConfig struct {
		ServiceName   string        `json:"serviceName"`
		TracingConfig TracingConfig `json:"tracingConfig"`
		MetricConfig  MetricConfig  `json:"metricConfig"`
	}
)
