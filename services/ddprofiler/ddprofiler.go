package ddprofiler

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_profiler "github.com/fluffy-bunny/fluffycore/contracts/ddprofiler"
	"github.com/fluffy-bunny/fluffycore/utils"
	tracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	profiler "gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

type (
	service struct {
		errProfiler error
		config      *contracts_profiler.Config
		once        sync.Once
	}

	DatadogTracerLoggerShim struct {
		logger *zerolog.Logger
	}
)

var stemService = (*service)(nil)

var _ contracts_profiler.IDataDogProfiler = (*service)(nil)

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

func AddSingletonIProfiler(builder di.ContainerBuilder, config *contracts_profiler.Config) {
	contracts_profiler.AddDDConfig(builder, config)
	di.AddSingleton[contracts_profiler.IDataDogProfiler](builder, stemService.Ctor)
}
func (s *service) Ctor(config *contracts_profiler.Config) (contracts_profiler.IDataDogProfiler, error) {

	if config.DDProfilerConfig == nil {
		config.DDProfilerConfig = &contracts_profiler.DDProfilerConfig{}
	}
	obj := &service{
		config: config,
	}
	return obj, nil
}
func (s *service) Start(ctx context.Context) {
	log := zerolog.Ctx(ctx).With().Logger()

	s.once.Do(func() {
		// Environment variable takes precedence over injected config
		if _, exists := os.LookupEnv("DATADOG_ENABLED"); exists {
			log.Info().Msg("DATADOG_ENABLED environment variable detected, overriding config value")
			s.config.DDProfilerConfig.Enabled = utils.BoolEnv("DATADOG_ENABLED", false)
		}
		log = log.With().Interface("config", s.config).Logger()

		log.Info().Msg("Datadog Profiler configuration")

		if s.config.DDProfilerConfig.Enabled {
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
	})
}
func (s *service) Stop(ctx context.Context) {
	if s.config.DDProfilerConfig.Enabled {
		log.Info().Msg("Stoping Datadog Tracer and Profiler")
		tracer.Stop()
		if s.errProfiler == nil {
			profiler.Stop()
		}
	}
}
