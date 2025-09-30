package observability

import (
	"quizapp/internal/config"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

// SetupObservability initializes tracing, metrics, and logging for a service
func SetupObservability(cfg *config.OpenTelemetryConfig, serviceName string) (result0 *trace.TracerProvider, result1 *metric.MeterProvider, result2 *Logger, err error) {
	if serviceName != "" {
		cfg.ServiceName = serviceName
	}

	var tp *trace.TracerProvider
	var mp *metric.MeterProvider
	var logger *Logger

	if cfg.EnableTracing {
		tp, err = InitTracing(cfg)
		if err != nil {
			return nil, nil, nil, err
		}
		// Initialize the global tracer
		InitGlobalTracer()
	}

	if cfg.EnableMetrics {
		mp, err = InitMetrics(cfg)
		if err != nil {
			return tp, nil, nil, err
		}
	}

	if cfg.EnableLogging {
		logger = NewLogger(cfg)
	} else {
		// Return a no-op logger when logging is disabled
		logger = NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	}

	return tp, mp, logger, nil
}
