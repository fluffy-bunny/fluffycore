package otel

import (
	"context"
	"time"

	fluffycore_contracts_otel "github.com/fluffy-bunny/fluffycore/contracts/otel"
	status "github.com/gogo/status"
	zerolog "github.com/rs/zerolog"
	otel_instrumentation_host "go.opentelemetry.io/contrib/instrumentation/host"
	otel_instrumentation_runtime "go.opentelemetry.io/contrib/instrumentation/runtime"
	otel "go.opentelemetry.io/otel"
	otlpmetricgrpc "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	otlpmetrichttp "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otlptrace "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	otlptracegrpc "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otlptracehttp "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	stdoutmetric "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	stdouttrace "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	propagation "go.opentelemetry.io/otel/propagation"
	otel_sdk_metric "go.opentelemetry.io/otel/sdk/metric"
	otel_sdk_resource "go.opentelemetry.io/otel/sdk/resource"
	otel_sdk_trace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	codes "google.golang.org/grpc/codes"
)

type (
	OTELContainer struct {
		TracerProvider *otel_sdk_trace.TracerProvider
		MetricExporter otel_sdk_metric.Exporter
		MeterProvider  *otel_sdk_metric.MeterProvider

		Config   *fluffycore_contracts_otel.OTELConfig
		Resource *otel_sdk_resource.Resource
	}
)

func NewOTELContainer() *OTELContainer {
	obj := &OTELContainer{}
	return obj
}
func (s *OTELContainer) GetOTELResource(ctx context.Context) *otel_sdk_resource.Resource {
	rr := otel_sdk_resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(s.Config.ServiceName),
	)
	return rr
}

func (s *OTELContainer) Init(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().Str("method", "OTELContainer.Start").Logger()
	s.Resource = s.GetOTELResource(ctx)
	err := s.InitOTELTraceProvider(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to InitOTELTraceProvider")
		return err
	}
	err = s.InitMeterProvider(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to InitOTELMetricProvider")
		return err
	}
	return nil
}

func (s *OTELContainer) Start(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().Str("method", "OTELContainer.Start").Logger()

	err := s.StartRuntimeMetric(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to Start OTLPMetricContainer")
		return err
	}
	err = s.StartHostMetric(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to Start OTLPMetricContainer")
		return err
	}
	return nil
}

func (s *OTELContainer) Stop(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().Str("method", "OTELContainer.Stop").Logger()
	if s.TracerProvider != nil {
		log.Info().Msg("Shutting down OTEL TracerProvider")
		s.TracerProvider.Shutdown(ctx)
	}
	if s.MetricExporter != nil {
		log.Info().Msg("Shutting down OTEL MetricExporter")
		s.MetricExporter.Shutdown(ctx)
	}
	return nil
}

func (s *OTELContainer) StartRuntimeMetric(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().Str("method", "OTELContainer.StartRuntimeMetric").Logger()
	if s.Config == nil {
		return nil
	}
	if !s.Config.MetricConfig.Enabled || !s.Config.MetricConfig.RuntimeEnabled {
		return nil
	}
	err := otel_instrumentation_runtime.Start(
		otel_instrumentation_runtime.WithMeterProvider(s.MeterProvider))
	if err != nil {
		log.Error().Err(err).Msg("failed to start runtime")
	}
	return nil
}

func (s *OTELContainer) StartHostMetric(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().Str("method", "OTELContainer.StartHostMetric").Logger()
	if s.Config == nil {
		return nil
	}
	if !s.Config.MetricConfig.Enabled || !s.Config.MetricConfig.HostEnabled {
		return nil
	}
	err := otel_instrumentation_host.Start(
		otel_instrumentation_host.WithMeterProvider(s.MeterProvider))
	if err != nil {
		log.Error().Err(err).Msg("failed to start runtime")
	}
	return nil
}

func (s *OTELContainer) InitMeterProvider(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().Str("method", "OTELContainer.InitMetricProvier").Logger()
	if s.Config == nil {
		return nil
	}
	if !s.Config.MetricConfig.Enabled || !s.Config.MetricConfig.RuntimeEnabled {
		return nil
	}
	var err error
	switch s.Config.MetricConfig.EndpointType {
	case fluffycore_contracts_otel.STDOUT:
		s.MetricExporter, err = stdoutmetric.New(stdoutmetric.WithPrettyPrint())
		if err != nil {
			log.Error().Err(err).Msg("failed to create exporter")
			return err
		}
	case fluffycore_contracts_otel.HTTP:
		s.MetricExporter, err = otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(s.Config.MetricConfig.Endpoint),
			otlpmetrichttp.WithInsecure())
		if err != nil {
			log.Error().Err(err).Msg("failed to create exporter")
			return err
		}
	case fluffycore_contracts_otel.GRPC:
		s.MetricExporter, err = otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(s.Config.MetricConfig.Endpoint),
			otlpmetricgrpc.WithInsecure())
		if err != nil {
			log.Error().Err(err).Msg("failed to create exporter")
			return err
		}
	}
	interval := time.Duration(s.Config.MetricConfig.IntervalSeconds) * time.Second
	read := otel_sdk_metric.NewPeriodicReader(s.MetricExporter, otel_sdk_metric.WithInterval(interval))
	provider := otel_sdk_metric.NewMeterProvider(
		otel_sdk_metric.WithResource(s.Resource),
		otel_sdk_metric.WithReader(read))
	s.MeterProvider = provider
	return nil
}

func (s *OTELContainer) InitOTELTraceProvider(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().Str("method", "InitOTELTraceProvider").Logger()
	if s.Config == nil || !s.Config.TracingConfig.Enabled {
		return nil
	}

	var err error
	var exporter *otlptrace.Exporter
	var traceProvider *otel_sdk_trace.TracerProvider
	switch s.Config.TracingConfig.EndpointType {
	case fluffycore_contracts_otel.HTTP:

		exporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithInsecure(),
			otlptracehttp.WithEndpoint(s.Config.TracingConfig.Endpoint),
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to create exporter")
			return err
		}
		traceProvider = otel_sdk_trace.NewTracerProvider(
			otel_sdk_trace.WithSampler(otel_sdk_trace.AlwaysSample()),
			otel_sdk_trace.WithBatcher(exporter),
			otel_sdk_trace.WithResource(s.Resource),
		)

	case fluffycore_contracts_otel.GRPC:
		exporter, err = otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(s.Config.TracingConfig.Endpoint),
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to create exporter")
			return err
		}
		traceProvider = otel_sdk_trace.NewTracerProvider(
			otel_sdk_trace.WithSampler(otel_sdk_trace.AlwaysSample()),
			otel_sdk_trace.WithBatcher(exporter),
			otel_sdk_trace.WithResource(s.Resource),
		)
	case fluffycore_contracts_otel.STDOUT:
		exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			log.Error().Err(err).Msg("failed to create exporter")
			return err
		}
		traceProvider = otel_sdk_trace.NewTracerProvider(
			otel_sdk_trace.WithSampler(otel_sdk_trace.AlwaysSample()),
			otel_sdk_trace.WithBatcher(exporter),
			otel_sdk_trace.WithResource(s.Resource),
		)
	default:
		return status.Error(codes.InvalidArgument, "Invalid OTEL endpoint type")
	}
	// Register the trace provider as the global provider.
	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{}),
	)
	s.TracerProvider = traceProvider
	return nil
}
