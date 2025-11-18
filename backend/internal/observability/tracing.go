package observability

import (
	"context"

	"quizapp/internal/config"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// InitTracing initializes OpenTelemetry tracing
func InitTracing(_ *config.OpenTelemetryConfig) (err error) {
	// ctx := context.Background()

	// // Set up resource attributes
	// res, err := resource.New(ctx,
	// 	resource.WithAttributes(
	// 		semconv.ServiceName(cfg.ServiceName),
	// 		semconv.ServiceVersion(cfg.ServiceVersion),
	// 	),
	// )
	// if err != nil {
	// 	return nil, contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to create otel resource: %w", err)
	// }

	// Set up exporter
	// var exporter trace.SpanExporter
	// switch cfg.Protocol {
	// case "grpc":
	// 	// For gRPC, strip http:// prefix if present, otherwise use endpoint as-is
	// 	endpoint := cfg.Endpoint
	// 	exp, err := otlptracegrpc.New(ctx,
	// 		otlptracegrpc.WithEndpoint(endpoint),
	// 		func() otlptracegrpc.Option {
	// 			if cfg.Insecure {
	// 				return otlptracegrpc.WithInsecure()
	// 			}
	// 			return nil
	// 		}(),
	// 		otlptracegrpc.WithHeaders(cfg.Headers),
	// 	)
	// 	if err != nil {
	// 		return nil, contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to create otlp grpc exporter: %w", err)
	// 	}
	// 	exporter = exp
	// case "http":
	// 	exp, err := otlptracehttp.New(ctx,
	// 		otlptracehttp.WithEndpoint(cfg.Endpoint),
	// 		otlptracehttp.WithInsecure(),
	// 		otlptracehttp.WithHeaders(cfg.Headers),
	// 	)
	// 	if err != nil {
	// 		return nil, contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to create otlp http exporter: %w", err)
	// 	}
	// 	exporter = exp
	// default:
	// 	return nil, contextutils.WrapErrorf(contextutils.ErrInternalError, "unsupported otel protocol: %s", cfg.Protocol)
	// }

	// Set up sampler
	// sampler := trace.ParentBased(trace.TraceIDRatioBased(cfg.SamplingRate))

	// // Set up tracer provider
	// tp := trace.NewTracerProvider(
	// 	trace.WithBatcher(exporter),
	// 	trace.WithResource(res),
	// 	trace.WithSampler(sampler),
	// )
	// otel.SetTracerProvider(tp)

	// Set up text map propagator for trace context propagation
	// This enables the backend to receive and process trace headers from NGINX
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return nil
}

// InitStandardTracing initializes a standard OpenTelemetry SDK TracerProvider with OTLP exporter
func InitStandardTracing(cfg *config.OpenTelemetryConfig) (result0 trace.TracerProvider, err error) {
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
	var exporter sdktrace.SpanExporter
	switch cfg.Protocol {
	case "grpc":
		// For gRPC, strip http:// prefix if present, otherwise use endpoint as-is
		endpoint := cfg.Endpoint
		exp, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(endpoint),
			func() otlptracegrpc.Option {
				if cfg.Insecure {
					return otlptracegrpc.WithInsecure()
				}
				return nil
			}(),
			otlptracegrpc.WithHeaders(cfg.Headers),
		)
		if err != nil {
			return nil, contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to create otlp grpc exporter: %w", err)
		}
		exporter = exp
	case "http":
		exp, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(cfg.Endpoint),
			func() otlptracehttp.Option {
				if cfg.Insecure {
					return otlptracehttp.WithInsecure()
				}
				return nil
			}(),
			otlptracehttp.WithHeaders(cfg.Headers),
		)
		if err != nil {
			return nil, contextutils.WrapErrorf(contextutils.ErrInternalError, "failed to create otlp http exporter: %w", err)
		}
		exporter = exp
	default:
		return nil, contextutils.WrapErrorf(contextutils.ErrInternalError, "unsupported otel protocol: %s", cfg.Protocol)
	}

	// Set up sampler
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplingRate))

	// Set up tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	return tp, nil
}
