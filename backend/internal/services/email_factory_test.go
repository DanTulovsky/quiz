package services

import (
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
)

func TestCreateEmailService_TestMode(t *testing.T) {
	cfg := &config.Config{
		IsTest: true,
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	service := CreateEmailService(cfg, logger)

	// Should create a test email service
	assert.IsType(t, &TestEmailService{}, service)
	assert.True(t, service.IsEnabled())
}

func TestCreateEmailService_ProductionMode(t *testing.T) {
	cfg := &config.Config{
		IsTest: false,
		Email: config.EmailConfig{
			Enabled: true,
			SMTP: config.SMTPConfig{
				Host: "smtp.example.com",
			},
		},
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	service := CreateEmailService(cfg, logger)

	// Should create a real email service
	assert.IsType(t, &EmailService{}, service)
}

func TestCreateEmailServiceWithDB_TestMode(t *testing.T) {
	cfg := &config.Config{
		IsTest: true,
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	service := CreateEmailServiceWithDB(cfg, logger, nil)

	// Should create a test email service
	assert.IsType(t, &TestEmailService{}, service)
	assert.True(t, service.IsEnabled())
}

func TestCreateEmailServiceWithDB_ProductionMode(t *testing.T) {
	cfg := &config.Config{
		IsTest: false,
		Email: config.EmailConfig{
			Enabled: true,
			SMTP: config.SMTPConfig{
				Host: "smtp.example.com",
			},
		},
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Should panic when database is nil
	assert.Panics(t, func() {
		CreateEmailServiceWithDB(cfg, logger, nil)
	}, "CreateEmailServiceWithDB should panic when database is nil")
}
