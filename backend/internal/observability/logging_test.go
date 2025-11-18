package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogWithContextAddsTraceInfo(t *testing.T) {
	// Setup OpenTelemetry
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test-tracer")

	// Setup Zap observer
	core, observedLogs := observer.New(zap.InfoLevel)
	zapLogger := zap.New(core)
	logger := &Logger{Logger: zapLogger}

	// Start a span
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Log something with the context
	logger.Info(ctx, "test message", nil)

	// Verify log entry
	requireLogs := observedLogs.All()
	assert.Equal(t, 1, len(requireLogs), "Expected 1 log entry")

	entry := requireLogs[0]
	assert.Equal(t, "test message", entry.Message)

	// Check for trace_id and span_id fields
	fields := entry.ContextMap()
	assert.Contains(t, fields, "trace_id", "Log should contain trace_id")
	assert.Contains(t, fields, "span_id", "Log should contain span_id")

	// Verify values match the span
	spanContext := span.SpanContext()
	assert.Equal(t, spanContext.TraceID().String(), fields["trace_id"])
	assert.Equal(t, spanContext.SpanID().String(), fields["span_id"])
}

func TestLogWithContextNoSpan(t *testing.T) {
	// Setup Zap observer
	core, observedLogs := observer.New(zap.InfoLevel)
	zapLogger := zap.New(core)
	logger := &Logger{Logger: zapLogger}

	// Log without a span
	logger.Info(context.Background(), "test message", nil)

	// Verify log entry
	requireLogs := observedLogs.All()
	assert.Equal(t, 1, len(requireLogs), "Expected 1 log entry")

	entry := requireLogs[0]
	fields := entry.ContextMap()

	// Should not contain trace info
	assert.NotContains(t, fields, "trace_id")
	assert.NotContains(t, fields, "span_id")
}
