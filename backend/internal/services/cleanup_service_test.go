package services

import (
	"context"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
)

func TestNewCleanupService(t *testing.T) {
	// Use nil database for testing tracer functionality
	service := NewCleanupServiceWithLogger(nil, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	assert.NotNil(t, service)
	assert.Nil(t, service.db)
	assert.NotNil(t, service.logger, "CleanupService should have a logger")
}

func TestCleanupService_GlobalTracer(t *testing.T) {
	// Use nil database for testing tracer functionality
	service := NewCleanupServiceWithLogger(nil, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Verify that the service uses the global tracer
	assert.NotNil(t, service.logger, "CleanupService should have a logger")

	// Test that the global tracer is properly initialized
	ctx := context.Background()
	ctx, span := observability.TraceCleanupFunction(ctx, "test_function")
	assert.NotNil(t, span, "Global tracer should create valid spans")
	span.End()

	// Test error handling with the global tracer
	err := observability.TraceFunctionWithErrorHandling(ctx, "cleanup", "test_error_function", func() error {
		return assert.AnError
	})
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}
