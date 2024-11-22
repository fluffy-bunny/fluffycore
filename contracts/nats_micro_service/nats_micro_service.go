package nats_micro_service

import (
	"context"

	nats "github.com/nats-io/nats.go"
	micro "github.com/nats-io/nats.go/micro"
	grpc "google.golang.org/grpc"
)

type (
	ContextHandlerFunc[NReq any, NResp any] func(ctx context.Context, request *NReq) (*NResp, error)

	NATSInterceptorConfig struct {
		TimeoutDuration string `json:"timeoutDuration"`
	}
	INATSMicroService interface {
		Interceptors() INATSMicroInterceptors
	}
	INATSMicroInterceptors interface {
		WithHandler(finalHandler grpc.UnaryHandler) grpc.UnaryHandler
	}

	NATSMicroServiceRegisrationOption struct {
		NATSMicroConfigOptions []NATSMicroConfigOption `json:"NATSMicroConfigOption"`
		GroupName              string                  `json:"groupName"`
	}
	INATSMicroServiceRegisration interface {
		AddService(nc *nats.Conn, option *NATSMicroServiceRegisrationOption) (micro.Service, error)
	}
	NATSMicroConfigOption func(config *micro.Config) *micro.Config
)

func WithMicroConfigNamne(name string) NATSMicroConfigOption {
	return func(config *micro.Config) *micro.Config {
		config.Name = name
		return config
	}
}
func WithMicroConfigVersion(version string) NATSMicroConfigOption {
	return func(config *micro.Config) *micro.Config {
		config.Version = version
		return config
	}
}
func WithMicroConfigEndpoint(Endpoint *micro.EndpointConfig) NATSMicroConfigOption {
	return func(config *micro.Config) *micro.Config {
		config.Endpoint = Endpoint
		return config
	}
}
func WithMicroConfigDescription(description string) NATSMicroConfigOption {
	return func(config *micro.Config) *micro.Config {
		config.Description = description
		return config
	}
}

func WithMicroConfigMetadata(metadata map[string]string) NATSMicroConfigOption {
	return func(config *micro.Config) *micro.Config {
		config.Metadata = metadata
		return config
	}
}

func WithMicroConfigQueueGroup(queueGroup string) NATSMicroConfigOption {
	return func(config *micro.Config) *micro.Config {
		config.QueueGroup = queueGroup
		return config
	}
}

func WithMicroConfigStatsHandler(statsHandler micro.StatsHandler) NATSMicroConfigOption {
	return func(config *micro.Config) *micro.Config {
		config.StatsHandler = statsHandler
		return config
	}
}
func WithMicroConfigDoneHandler(doneHandler micro.DoneHandler) NATSMicroConfigOption {
	return func(config *micro.Config) *micro.Config {
		config.DoneHandler = doneHandler
		return config
	}
}

func WithMicroConfigErrorHandler(errorHandler micro.ErrHandler) NATSMicroConfigOption {
	return func(config *micro.Config) *micro.Config {
		config.ErrorHandler = errorHandler
		return config
	}
}
