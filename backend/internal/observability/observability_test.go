package observability

import (
	"context"
	"reflect"
	"testing"

	"quizapp/internal/config"

	"github.com/stretchr/testify/require"
	autosdk "go.opentelemetry.io/auto/sdk"
	"go.opentelemetry.io/otel/trace/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
	// // Check that the tracer provider is a noop (no-op) implementation
	// _, isNoop := tp.(noop.TracerProvider)
	// require.True(t, isNoop, "expected tp to be a noop.TracerProvider when tracing is disabled")
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

func TestSetupObservability_UseAutoSDK_True(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		EnableTracing: true,
		UseAutoSDK:    true,
		ServiceName:   "test-service",
		ServiceVersion: "1.0.0",
		Protocol:      "grpc",
		Endpoint:      "localhost:4317",
		Insecure:      true,
	}
	tp, _, logger, err := SetupObservability(cfg, "test-service")
	require.NoError(t, err)
	require.NotNil(t, tp)
	require.NotNil(t, logger)

	// Verify it's using Auto SDK by checking the type
	// Auto SDK TracerProvider should not be a standard SDK TracerProvider
	_, isStandardSDK := tp.(*sdktrace.TracerProvider)
	require.False(t, isStandardSDK, "Expected Auto SDK TracerProvider, got standard SDK")
}

func TestSetupObservability_UseAutoSDK_False(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		EnableTracing: true,
		UseAutoSDK:    false,
		ServiceName:   "test-service",
		ServiceVersion: "1.0.0",
		Protocol:      "grpc",
		Endpoint:      "localhost:4317",
		Insecure:      true,
		SamplingRate:  1.0,
	}
	tp, _, logger, err := SetupObservability(cfg, "test-service")
	require.NoError(t, err)
	require.NotNil(t, tp)
	require.NotNil(t, logger)

	// Verify it's using standard SDK by checking the type
	_, isStandardSDK := tp.(*sdktrace.TracerProvider)
	require.True(t, isStandardSDK, "Expected standard SDK TracerProvider")
}

func TestSetupObservability_UseAutoSDK_Default(t *testing.T) {
	// Test that UseAutoSDK defaults to false (Go zero value) but config.yaml sets it to true
	// Since we can't test the YAML default here, we'll test the zero value behavior
	cfg := &config.OpenTelemetryConfig{
		EnableTracing: true,
		// UseAutoSDK not set, defaults to false in Go
		ServiceName:   "test-service",
		ServiceVersion: "1.0.0",
		Protocol:      "grpc",
		Endpoint:      "localhost:4317",
		Insecure:      true,
		SamplingRate:  1.0,
	}
	tp, _, logger, err := SetupObservability(cfg, "test-service")
	require.NoError(t, err)
	require.NotNil(t, tp)
	require.NotNil(t, logger)

	// With UseAutoSDK=false (zero value), should use standard SDK
	_, isStandardSDK := tp.(*sdktrace.TracerProvider)
	require.True(t, isStandardSDK, "Expected standard SDK TracerProvider when UseAutoSDK is false")
}

func TestInitStandardTracing_GRPC(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		ServiceName:   "test-service",
		ServiceVersion: "1.0.0",
		Protocol:      "grpc",
		Endpoint:      "localhost:4317",
		Insecure:      true,
		SamplingRate:  1.0,
	}
	tp, err := InitStandardTracing(cfg)
	require.NoError(t, err)
	require.NotNil(t, tp)

	// Verify it's a standard SDK TracerProvider
	_, ok := tp.(*sdktrace.TracerProvider)
	require.True(t, ok, "Expected *sdktrace.TracerProvider")
}

func TestInitStandardTracing_HTTP(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		ServiceName:   "test-service",
		ServiceVersion: "1.0.0",
		Protocol:      "http",
		Endpoint:      "localhost:4318",
		Insecure:      true,
		SamplingRate:  0.5,
	}
	tp, err := InitStandardTracing(cfg)
	require.NoError(t, err)
	require.NotNil(t, tp)

	// Verify it's a standard SDK TracerProvider
	_, ok := tp.(*sdktrace.TracerProvider)
	require.True(t, ok, "Expected *sdktrace.TracerProvider")
}

func TestInitStandardTracing_InvalidProtocol(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		ServiceName:   "test-service",
		ServiceVersion: "1.0.0",
		Protocol:      "invalid",
		Endpoint:      "localhost:4317",
		Insecure:      true,
		SamplingRate:  1.0,
	}
	tp, err := InitStandardTracing(cfg)
	require.Error(t, err)
	require.Nil(t, tp)
	require.Contains(t, err.Error(), "unsupported otel protocol")
}

func TestConfig_UseAutoSDK_EnvironmentVariable(t *testing.T) {
	// Test that when UseAutoSDK is set to false, standard SDK is used
	// This simulates what happens when OPEN_TELEMETRY_USE_AUTO_SDK=false is set
	cfg := &config.OpenTelemetryConfig{
		EnableTracing: true,
		UseAutoSDK:    false, // Simulates env var override
		ServiceName:   "test-service",
		ServiceVersion: "1.0.0",
		Protocol:      "grpc",
		Endpoint:      "localhost:4317",
		Insecure:      true,
		SamplingRate:  1.0,
	}

	tp, _, _, err := SetupObservability(cfg, "test-service")
	require.NoError(t, err)
	require.NotNil(t, tp)

	// Should use standard SDK when UseAutoSDK is false
	_, isStandardSDK := tp.(*sdktrace.TracerProvider)
	require.True(t, isStandardSDK, "Expected standard SDK when UseAutoSDK is false")
}

func TestSetupObservability_AutoSDK_TypeCheck(t *testing.T) {
	cfg := &config.OpenTelemetryConfig{
		EnableTracing: true,
		UseAutoSDK:    true,
		ServiceName:   "test-service",
		ServiceVersion: "1.0.0",
	}

	// Create Auto SDK tracer provider directly for comparison
	autoTp := autosdk.TracerProvider()

	tp, _, _, err := SetupObservability(cfg, "test-service")
	require.NoError(t, err)
	require.NotNil(t, tp)

	// Check that both are the same type
	autoType := reflect.TypeOf(autoTp)
	tpType := reflect.TypeOf(tp)

	// They should have the same type name/package
	require.Equal(t, autoType, tpType, "SetupObservability with UseAutoSDK=true should return Auto SDK TracerProvider")
}
