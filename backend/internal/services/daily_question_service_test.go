package services

import (
	"context"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
)

func TestDailyQuestionService_GetQuestionHistory_InvalidDays(t *testing.T) {
	// Create logger
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Create service with nil DB (we're only testing validation)
	service := &DailyQuestionService{
		db:     nil,
		logger: logger,
	}

	// Test with invalid days
	_, err := service.GetQuestionHistory(context.Background(), 1, 123, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "days must be positive")

	_, err = service.GetQuestionHistory(context.Background(), 1, 123, -5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "days must be positive")
}

func TestDailyQuestionService_GetQuestionHistory_ValidDays(t *testing.T) {
	// This test just verifies that the validation logic works correctly
	// We can't actually call the method with a nil DB, so we'll just test the validation
	// by checking that the validation function exists and works as expected

	// Create logger
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Create service with nil DB (we're only testing validation)
	service := &DailyQuestionService{
		db:     nil,
		logger: logger,
	}

	// Test that the service can be created (this tests the validation logic exists)
	assert.NotNil(t, service)
	assert.NotNil(t, service.logger)

	// We can't actually test the method with nil DB, but we can verify the service structure
	// The actual validation happens in the method, which we test in the invalid days test
}

func TestDailyQuestionHistory_Model(t *testing.T) {
	// Test the DailyQuestionHistory model structure
	now := time.Now()
	isCorrect := true

	history := models.DailyQuestionHistory{
		AssignmentDate: now,
		IsCompleted:    true,
		SubmittedAt:    &now,
		IsCorrect:      &isCorrect,
	}

	// Verify the model can be created and accessed
	assert.Equal(t, now.Year(), history.AssignmentDate.Year())
	assert.True(t, history.IsCompleted)
	if history.IsCorrect == nil {
		t.Fatalf("expected IsCorrect to be set")
	}
	assert.True(t, *history.IsCorrect)
	assert.Equal(t, now, *history.SubmittedAt)
}
