package observability

import (
	"context"
	"testing"

	"quizapp/internal/config"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestSetupObservability_AllEnabled(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		EnableTracing: true,
		EnableMetrics: true,
		EnableLogging: true,
		ServiceName:   "test-service",
		Protocol:      "grpc",
		Endpoint:      "localhost:4317",
		Insecure:      true,
	}
	tp, mp, logger, err := SetupObservability(cfg, "test-service")
	require.NoError(t, err)
	require.NotNil(t, tp)
	require.NotNil(t, mp)
	require.NotNil(t, logger)
}

func TestSetupObservability_NoneEnabled(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		EnableTracing: false,
		EnableMetrics: false,
		EnableLogging: false,
		ServiceName:   "test-service",
		Protocol:      "grpc",
		Endpoint:      "localhost:4317",
		Insecure:      true,
	}
	tp, mp, logger, err := SetupObservability(cfg, "test-service")
	require.NoError(t, err)
	require.Nil(t, tp)
	require.Nil(t, mp)
	require.NotNil(t, logger) // Logger is always returned (no-op when disabled)
}

func TestLogger_TraceCorrelation(_ *testing.T) {
	cfg := &config.OpenTelemetryConfig{EnableLogging: true}
	logger := NewLogger(cfg)
	ctx := context.Background()
	// Test basic logging functionality
	logger.Info(ctx, "test message")
	logger.Error(ctx, "test error", nil)
	// With span in context
	ctx, span := noop.NewTracerProvider().Tracer("test").Start(ctx, "test-span")
	logger.Info(ctx, "test message with span")
	span.End()
}
