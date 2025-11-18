package observability

import (
	"context"
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

	if err := os.Setenv("OTEL_SERVICE_NAME", cfg.ServiceName); err != nil {
		return nil, nil, nil, err
	}
	if err := os.Setenv("OTEL_SERVICE_VERSION", cfg.ServiceVersion); err != nil {
		return nil, nil, nil, err
	}

	if cfg.EnableLogging {
		logger = NewLogger(cfg)
	} else {
		// Return a no-op logger when logging is disabled
		logger = NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	}

	if cfg.EnableTracing {
		if cfg.UseAutoSDK {
			// Use Auto SDK (default behavior, compatible with OBI)
			tp = autosdk.TracerProvider()
			otel.SetTracerProvider(tp)

			logger.Info(context.Background(), "Tracing enabled with Auto SDK", map[string]interface{}{"service_name": cfg.ServiceName})
		} else {
			// Use standard OpenTelemetry SDK with OTLP exporter
			tp, err = InitStandardTracing(cfg)
			if err != nil {
				panic(err)
			}
			otel.SetTracerProvider(tp)

			logger.Info(context.Background(), "Tracing enabled with standard SDK", map[string]interface{}{"service_name": cfg.ServiceName})
		}

		err := InitTracing(cfg)
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

	return tp, mp, logger, nil
}
