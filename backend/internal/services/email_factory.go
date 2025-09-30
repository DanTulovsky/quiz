// Package services provides business logic services for the quiz application.
package services

import (
	"context"
	"database/sql"

	"quizapp/internal/config"
	"quizapp/internal/observability"
	"quizapp/internal/services/mailer"
)

// CreateEmailService creates an appropriate email service based on configuration
// If the application is running in test mode, it returns a TestEmailService
// Otherwise, it returns the regular EmailService
func CreateEmailService(cfg *config.Config, logger *observability.Logger) mailer.Mailer {
	if cfg.IsTest {
		logger.Info(context.Background(), "Using test email service", map[string]interface{}{
			"test_mode": true,
		})
		return NewTestEmailService(cfg, logger)
	}

	return NewEmailService(cfg, logger)
}

// CreateEmailServiceWithDB creates an appropriate email service with database connection based on configuration
// If the application is running in test mode, it returns a TestEmailService
// Otherwise, it returns the regular EmailService
func CreateEmailServiceWithDB(cfg *config.Config, logger *observability.Logger, db *sql.DB) mailer.Mailer {
	if cfg.IsTest {
		logger.Info(context.Background(), "Using test email service with DB", map[string]interface{}{
			"test_mode": true,
		})
		return NewTestEmailServiceWithDB(cfg, logger, db)
	}

	if db == nil {
		logger.Error(context.Background(), "Database connection is nil, cannot create EmailService", nil, map[string]interface{}{
			"error": "nil_database_connection",
		})
		panic("EmailService requires a non-nil database connection")
	}

	return NewEmailServiceWithDB(cfg, logger, db)
}
