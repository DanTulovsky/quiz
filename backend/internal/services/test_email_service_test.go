package services

import (
	"context"
	"database/sql"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
)

func TestTestEmailService_IsEnabled(t *testing.T) {
	cfg := &config.Config{}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	service := NewTestEmailService(cfg, logger)

	// Test email service should always be enabled
	assert.True(t, service.IsEnabled())
}

func TestTestEmailService_SendDailyReminder(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			AppBaseURL: "http://localhost:3000",
		},
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	service := NewTestEmailService(cfg, logger)

	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    sql.NullString{String: "test@example.com", Valid: true},
	}

	ctx := context.Background()
	err := service.SendDailyReminder(ctx, user)

	// Should not return an error
	assert.NoError(t, err)
}

func TestTestEmailService_SendEmail(t *testing.T) {
	cfg := &config.Config{}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	service := NewTestEmailService(cfg, logger)

	ctx := context.Background()
	data := map[string]interface{}{
		"test": "value",
	}

	err := service.SendEmail(ctx, "test@example.com", "Test Subject", "test_email", data)

	// Should not return an error
	assert.NoError(t, err)
}

func TestTestEmailService_RecordSentNotification(t *testing.T) {
	cfg := &config.Config{}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	service := NewTestEmailService(cfg, logger)

	ctx := context.Background()
	err := service.RecordSentNotification(ctx, 1, "test_type", "Test Subject", "test_template", "sent", "")

	// Should not return an error (even without DB, it should handle gracefully)
	assert.NoError(t, err)
}
