package ddprofiler

import (
	"context"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contractsProfiler "github.com/fluffy-bunny/fluffycore/contracts/ddprofiler"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

type (
	service struct {
		errProfiler     error
		config          *contractsProfiler.Config
		profilerEnabled bool
	}

	DatadogTracerLoggerShim struct {
		logger *zerolog.Logger
	}
)

var stemService = new(service)

func init() {
	var _ contractsProfiler.IDataDogProfiler = (*service)(nil)
}

// Shim method to make Datadog tracer log in JSON format instead of plain text
func (d *DatadogTracerLoggerShim) Log(msg string) {
	if strings.Contains(msg, "ERROR") {
		d.logger.Error().Msg(msg)
	} else if strings.Contains(msg, "WARN") {
		d.logger.Warn().Msg(msg)
	} else {
		d.logger.Info().Msg(msg)
	}
}

func NewDatadogTracerLoggerShim(logger *zerolog.Logger) *DatadogTracerLoggerShim {
	return &DatadogTracerLoggerShim{logger: logger}
}

func AddSingletonIProfiler(builder di.ContainerBuilder) {
	di.AddSingleton[contractsProfiler.IDataDogProfiler](builder, stemService.Ctor)
}
func (s *service) Ctor(config *contractsProfiler.Config) (contractsProfiler.IDataDogProfiler, error) {
	obj := &service{
		config:          config,
		profilerEnabled: s.config.DDProfilerConfig != nil && s.config.DDProfilerConfig.Enabled,
	}
	return obj, nil
}
func (s *service) Start(ctx context.Context) {
	log := zerolog.Ctx(ctx).With().Logger()
	if s.profilerEnabled {
		log.Info().Msg("Starting Datadog Tracer and Profiler")
		// Start datadog tracing
		tracer.Start(
			tracer.WithService(s.config.ServiceName),
			tracer.WithServiceVersion(s.config.Version),
			tracer.WithLogger(NewDatadogTracerLoggerShim(&log)),
		)
		// Start datadog profiling
		s.errProfiler = profiler.Start(
			profiler.WithService(s.config.ServiceName),
			profiler.WithEnv(s.config.ApplicationEnvironment),
			profiler.WithVersion(s.config.Version),
			profiler.WithProfileTypes(
				profiler.CPUProfile,
				profiler.HeapProfile,
				// The profiles below are disabled by default to keep overhead
				// low, but can be enabled as needed.

				// profiler.BlockProfile,
				// profiler.MutexProfile,
				// profiler.GoroutineProfile,
			),
		)
		if s.errProfiler != nil {
			log.Error().Err(s.errProfiler).Msg("Failed to start DataDog profiling - continuing without it")
		}
	}
}
func (s *service) Stop(ctx context.Context) {
	if s.profilerEnabled {
		log.Info().Msg("Stoping Datadog Tracer and Profiler")
		tracer.Stop()
		if s.errProfiler == nil {
			profiler.Stop()
		}
	}
}
