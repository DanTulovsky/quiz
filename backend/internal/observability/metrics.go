package observability

import (
	"context"

	"quizapp/internal/config"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// InitMetrics initializes OpenTelemetry metrics
func InitMetrics(cfg *config.OpenTelemetryConfig) (result0 *metric.MeterProvider, err error) {
	ctx := context.Background()

	// Set up resource attributes
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to create otel resource: %w", err)
	}

	// Set up exporter
	var exporter metric.Exporter
	switch cfg.Protocol {
	case "grpc":
		// For gRPC, strip http:// prefix if present, otherwise use endpoint as-is
		endpoint := cfg.Endpoint
		exp, err := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(endpoint),
			func() otlpmetricgrpc.Option {
				if cfg.Insecure {
					return otlpmetricgrpc.WithInsecure()
				}
				return nil
			}(),
			otlpmetricgrpc.WithHeaders(cfg.Headers),
		)
		if err != nil {
			return nil, contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to create otlp grpc metric exporter: %w", err)
		}
		exporter = exp
	case "http":
		exp, err := otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(cfg.Endpoint),
			func() otlpmetrichttp.Option {
				if cfg.Insecure {
					return otlpmetrichttp.WithInsecure()
				}
				return nil
			}(),
			otlpmetrichttp.WithHeaders(cfg.Headers),
		)
		if err != nil {
			return nil, contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to create otlp http metric exporter: %w", err)
		}
		exporter = exp
	default:
		return nil, contextutils.WrapErrorf(contextutils.ErrInternalError, "unsupported otel protocol: %s", cfg.Protocol)
	}

	// Set up meter provider
	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(res),
	)
	return mp, nil
}
