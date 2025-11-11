package observability

import (
	"os"
	"quizapp/internal/config"

	autosdk "go.opentelemetry.io/auto/sdk"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/trace"
)

// SetupObservability initializes tracing, metrics, and logging for a service
func SetupObservability(cfg *config.OpenTelemetryConfig, serviceName string) (result0 trace.TracerProvider, result1 *metric.MeterProvider, result2 *Logger, err error) {
	if serviceName != "" {
		cfg.ServiceName = serviceName
	}

	var tp trace.TracerProvider
	var mp *metric.MeterProvider
	var logger *Logger

	os.Setenv("OTEL_SERVICE_NAME", cfg.ServiceName)
	os.Setenv("OTEL_SERVICE_VERSION", cfg.ServiceVersion)

	if cfg.EnableTracing {
		tp = autosdk.TracerProvider()
		otel.SetTracerProvider(tp)

		err = InitTracing(cfg)
		if err != nil {
			panic(err)
		}
		// Initialize the global tracer
		InitGlobalTracer()

	}

	if cfg.EnableMetrics {
		mp, err = InitMetrics(cfg)
		if err != nil {
			panic(err)
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
