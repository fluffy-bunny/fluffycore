package otel

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_services_common "github.com/fluffy-bunny/fluffycore/services/common"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
	grpc_health "google.golang.org/grpc/health/grpc_health_v1"
	metadata "google.golang.org/grpc/metadata"
)

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}
func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// loggingHealth logs from inside the handler and sets its own response header.
type loggingHealth struct {
	grpc_health.UnimplementedHealthServer
}

func (loggingHealth) Check(ctx context.Context, _ *grpc_health.HealthCheckRequest) (*grpc_health.HealthCheckResponse, error) {
	zerolog.Ctx(ctx).Info().Msg("inside-handler")
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-handler", "yes")); err != nil {
		return nil, err
	}
	return &grpc_health.HealthCheckResponse{Status: grpc_health.HealthCheckResponse_SERVING}, nil
}

// Correlation and trace IDs never reached handler log lines, and the correlation
// interceptor's grpc.SendHeader made every later grpc.SetHeader in the handler fail.
func TestStandardChain_CorrelationInLogsAndHandlerHeaders(t *testing.T) {
	buf := &syncBuffer{}
	prev := zlog.Logger
	zlog.Logger = zerolog.New(buf)
	t.Cleanup(func() { zlog.Logger = prev })

	b := di.Builder()
	fluffycore_services_common.AddCommonServices(b)
	client := startServerWith(t, b.Build(), loggingHealth{})

	ctx := metadata.AppendToOutgoingContext(callCtx(t), fluffycore_wellknown.XCorrelationIDName, "corr-123")
	var header metadata.MD
	_, err := client.Check(ctx, &grpc_health.HealthCheckRequest{}, grpc.Header(&header))
	require.NoError(t, err, "the handler's SetHeader must not fail")
	require.Equal(t, []string{"yes"}, header.Get("x-handler"))
	require.Equal(t, []string{"corr-123"}, header.Get(fluffycore_wellknown.XCorrelationIDName))

	var line string
	for _, l := range strings.Split(buf.String(), "\n") {
		if strings.Contains(l, "inside-handler") {
			line = l
		}
	}
	require.NotEmpty(t, line, "handler log line not found in: %s", buf.String())
	require.Contains(t, line, `"`+fluffycore_wellknown.LogCorrelationIDName+`":"corr-123"`)
	require.Contains(t, line, `"trace_id"`)
}
