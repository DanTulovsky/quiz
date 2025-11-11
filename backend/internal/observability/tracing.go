package observability

import (
	"quizapp/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// InitTracing initializes OpenTelemetry tracing
func InitTracing(cfg *config.OpenTelemetryConfig) (err error) {
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
