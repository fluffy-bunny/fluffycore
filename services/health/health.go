// Package health is fluffycore's gRPC health service, built on grpc-go's health.Server.
package health

import (
	"context"
	"sync"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_health "github.com/fluffy-bunny/fluffycore/contracts/health"
	grpc_health_server "google.golang.org/grpc/health"
	grpc_health "google.golang.org/grpc/health/grpc_health_v1"
)

// watchDrainGrace is how long Watch streams stay open after Shutdown, so the NOT_SERVING
// update reaches watchers before their streams end.
const watchDrainGrace = 250 * time.Millisecond

// Service answers Check, List and Watch from real serving status. The previous service
// answered SERVING to every Check -- even while the server was shutting down, so load
// balancers kept routing to it -- and its Watch returned immediately, which ends the stream
// instead of streaming status changes.
//
// Shutdown marks everything NOT_SERVING (the runtime calls it before draining) and then ends
// open Watch streams: a Watch otherwise lasts until the client leaves, and GracefulStop waits
// for every open stream.
type Service struct {
	*grpc_health_server.Server
	stopOnce sync.Once
	stopped  chan struct{}
}

var _ fluffycore_contracts_health.IHealthServer = (*Service)(nil)

// New returns a health service reporting the overall server ("") as SERVING.
func New() *Service {
	return &Service{Server: grpc_health_server.NewServer(), stopped: make(chan struct{})}
}

// AddHealthService registers a singleton health service.
func AddHealthService(cb di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_health.IHealthServer](cb, func() fluffycore_contracts_health.IHealthServer {
		return New()
	})
}

// Shutdown marks every service NOT_SERVING, ignores later status updates, and ends open
// Watch streams after watchDrainGrace. Safe to call more than once.
func (s *Service) Shutdown() {
	s.stopOnce.Do(func() {
		s.Server.Shutdown()
		close(s.stopped)
	})
}

// Watch streams status changes until the client leaves or the service shuts down.
func (s *Service) Watch(in *grpc_health.HealthCheckRequest, stream grpc_health.Health_WatchServer) error {
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	go func() {
		select {
		case <-s.stopped:
			select {
			case <-time.After(watchDrainGrace):
			case <-ctx.Done():
			}
			cancel()
		case <-ctx.Done():
		}
	}()
	err := s.Server.Watch(in, &watchStream{Health_WatchServer: stream, ctx: ctx})
	select {
	case <-s.stopped:
		if stream.Context().Err() == nil {
			return nil // we ended it: a clean end of stream, not an error
		}
	default:
	}
	return err
}

// watchStream overrides the stream's context so Watch can be ended from Shutdown.
type watchStream struct {
	grpc_health.Health_WatchServer
	ctx context.Context
}

func (w *watchStream) Context() context.Context { return w.ctx }
