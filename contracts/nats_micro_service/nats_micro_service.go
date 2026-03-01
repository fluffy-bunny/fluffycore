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
	INATSMicroClientInterceptors interface {
		Interceptors() INATSMicroInterceptors
	}
	INATSMicroInterceptors interface {
		WithHandler(finalHandler grpc.UnaryHandler) grpc.UnaryHandler
	}

	NATSMicroServiceRegisrationOption struct {
		ConfigServiceMicroOptions []ConfigServiceMicroOption `json:"ServiceMicroOption"`
		ConfigNATSMicroConfigs    []ConfigNATSMicroConfig    `json:"NATSMicroConfigOption"`
		GroupName                 string                     `json:"groupName"`
	}
	INATSMicroServiceRegisration interface {
		AddService(nc *nats.Conn, option *NATSMicroServiceRegisrationOption) (micro.Service, error)
	}
	ConfigNATSMicroConfig func(config *micro.Config) *micro.Config
	ServiceMicroOption    struct {
		GroupName string
	}
	ConfigServiceMicroOption func(config *ServiceMicroOption) *ServiceMicroOption
)

func WithServiceMicroOptionGroupName(groupName string) ConfigServiceMicroOption {
	return func(config *ServiceMicroOption) *ServiceMicroOption {
		config.GroupName = groupName
		return config
	}
}

// WithMicroConfigName sets the name on the NATS micro config.
func WithMicroConfigName(name string) ConfigNATSMicroConfig {
	return func(config *micro.Config) *micro.Config {
		config.Name = name
		return config
	}
}

// Deprecated: Use WithMicroConfigName instead.
func WithMicroConfigNamne(name string) ConfigNATSMicroConfig {
	return WithMicroConfigName(name)
}
func WithMicroConfigVersion(version string) ConfigNATSMicroConfig {
	return func(config *micro.Config) *micro.Config {
		config.Version = version
		return config
	}
}
func WithMicroConfigEndpoint(Endpoint *micro.EndpointConfig) ConfigNATSMicroConfig {
	return func(config *micro.Config) *micro.Config {
		config.Endpoint = Endpoint
		return config
	}
}
func WithMicroConfigDescription(description string) ConfigNATSMicroConfig {
	return func(config *micro.Config) *micro.Config {
		config.Description = description
		return config
	}
}

func WithMicroConfigMetadata(metadata map[string]string) ConfigNATSMicroConfig {
	return func(config *micro.Config) *micro.Config {
		config.Metadata = metadata
		return config
	}
}

func WithMicroConfigQueueGroup(queueGroup string) ConfigNATSMicroConfig {
	return func(config *micro.Config) *micro.Config {
		config.QueueGroup = queueGroup
		return config
	}
}

func WithMicroConfigStatsHandler(statsHandler micro.StatsHandler) ConfigNATSMicroConfig {
	return func(config *micro.Config) *micro.Config {
		config.StatsHandler = statsHandler
		return config
	}
}
func WithMicroConfigDoneHandler(doneHandler micro.DoneHandler) ConfigNATSMicroConfig {
	return func(config *micro.Config) *micro.Config {
		config.DoneHandler = doneHandler
		return config
	}
}

func WithMicroConfigErrorHandler(errorHandler micro.ErrHandler) ConfigNATSMicroConfig {
	return func(config *micro.Config) *micro.Config {
		config.ErrorHandler = errorHandler
		return config
	}
}
