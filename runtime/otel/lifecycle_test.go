package otel

import (
	"context"
	"io"
	"os"
	"testing"

	fluffycore_contracts_otel "github.com/fluffy-bunny/fluffycore/contracts/otel"
	"github.com/stretchr/testify/require"
	otel "go.opentelemetry.io/otel"
	tracetest "go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// captureStdout runs fn with os.Stdout redirected and returns what was written. The stdout
// exporters bind os.Stdout when they are created, so fn must create them.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	prev := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() { b, _ := io.ReadAll(r); done <- string(b) }()
	fn()
	os.Stdout = prev
	_ = w.Close()
	return <-done
}

func metricsConfig(runtime, host bool, endpoint fluffycore_contracts_otel.EndpointType) *fluffycore_contracts_otel.OTELConfig {
	c := &fluffycore_contracts_otel.OTELConfig{ServiceName: "lifecycle-test"}
	c.MetricConfig.Enabled = true
	c.MetricConfig.RuntimeEnabled = runtime
	c.MetricConfig.HostEnabled = host
	c.MetricConfig.EndpointType = endpoint
	c.MetricConfig.IntervalSeconds = 3600 // only the shutdown flush will export
	return c
}

// A host-metrics-only config used to get no MeterProvider at all.
func TestInitMeterProvider_HostOnly(t *testing.T) {
	c := NewOTELContainer()
	c.Config = metricsConfig(false, true, fluffycore_contracts_otel.STDOUT)
	require.NoError(t, c.InitMeterProvider(context.Background()))
	require.NotNil(t, c.MeterProvider)
	require.NoError(t, c.MeterProvider.Shutdown(context.Background()))
}

func TestInitMeterProvider_UnknownEndpointIsAnError(t *testing.T) {
	c := NewOTELContainer()
	c.Config = metricsConfig(true, false, "carrier-pigeon")
	require.Error(t, c.InitMeterProvider(context.Background()))
}

// Stop used to shut down only the exporter, never the MeterProvider, so the final
// collection was never exported.
func TestStop_FlushesFinalMetrics(t *testing.T) {
	out := captureStdout(t, func() {
		c := NewOTELContainer()
		c.Config = metricsConfig(true, false, fluffycore_contracts_otel.STDOUT)
		require.NoError(t, c.Init(context.Background()))
		counter, err := c.MeterProvider.Meter("t").Int64Counter("flush_me_counter")
		require.NoError(t, err)
		counter.Add(context.Background(), 1)
		require.NoError(t, c.Stop(context.Background()))
	})
	require.Contains(t, out, "flush_me_counter")
}

// OTEL used to stop in OnPreServerShutdown -- before the server drained -- so spans from
// requests still in flight were lost. A span ending during the drain must still be exported.
func TestShutdownHooks_SpansDuringDrainAreExported(t *testing.T) {
	prev := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	startup := NewFluffyCoreOTELStartup(&FluffyCoreOTELStartupConfig{})
	cfg := &fluffycore_contracts_otel.OTELConfig{ServiceName: "lifecycle-test"}
	cfg.TracingConfig.Enabled = true
	cfg.TracingConfig.EndpointType = fluffycore_contracts_otel.STDOUT
	startup.SetConfig(cfg)
	require.NoError(t, startup.OTELContainer.Init(context.Background()))
	// Once the provider is shut down, ended spans reach no processor.
	recorder := tracetest.NewSpanRecorder()
	startup.TracerProvider.RegisterSpanProcessor(recorder)

	ctx := context.Background()
	_, span := startup.TracerProvider.Tracer("t").Start(ctx, "in-flight-during-drain")
	startup.OnPreServerShutdown(ctx)
	span.End() // the request finishes while the server drains
	startup.OnPostServerShutdown(ctx)

	require.Len(t, recorder.Ended(), 1, "a span ending during the drain must still be processed")
	require.Equal(t, "in-flight-during-drain", recorder.Ended()[0].Name())
}
