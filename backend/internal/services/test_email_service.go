// Package services provides business logic services for the quiz application.
package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TestEmailService implements the Mailer interface for testing purposes
// It doesn't actually send emails but logs the operations and records them in the database
type TestEmailService struct {
	cfg    *config.Config
	logger *observability.Logger
	db     *sql.DB
}

// NewTestEmailService creates a new TestEmailService instance
func NewTestEmailService(cfg *config.Config, logger *observability.Logger) *TestEmailService {
	return &TestEmailService{
		cfg:    cfg,
		logger: logger,
	}
}

// NewTestEmailServiceWithDB creates a new TestEmailService instance with database connection
func NewTestEmailServiceWithDB(cfg *config.Config, logger *observability.Logger, db *sql.DB) *TestEmailService {
	return &TestEmailService{
		cfg:    cfg,
		logger: logger,
		db:     db,
	}
}

// SendDailyReminder sends a daily reminder email to a user (test mode - just logs)
func (e *TestEmailService) SendDailyReminder(ctx context.Context, user *models.User) error {
	ctx, span := otel.Tracer("test-email-service").Start(ctx, "SendDailyReminder",
		trace.WithAttributes(
			attribute.Int("user.id", user.ID),
			attribute.String("user.email", user.Email.String),
		),
	)
	defer span.End()

	if !user.Email.Valid || user.Email.String == "" {
		e.logger.Warn(ctx, "User has no email address, skipping daily reminder", map[string]interface{}{
			"user_id": user.ID,
		})
		return nil
	}

	// Generate email data (same as real service) - not used in test mode but kept for consistency
	_ = map[string]interface{}{
		"Username":       user.Username,
		"QuizAppURL":     e.cfg.Server.AppBaseURL,
		"CurrentDate":    time.Now().Format("January 2, 2006"),
		"DailyGoal":      10,
		"StreakDays":     5,
		"TotalQuestions": 150,
		"Level":          "B1",
		"Language":       "Italian",
	}

	// Log the email operation instead of sending. Use the same subject as the
	// real service to avoid confusion, but do NOT record a second entry in the
	// database here — recording is handled by caller to ensure a single source
	// of truth for sent notifications.
	e.logger.Info(ctx, "TEST MODE: Would send daily reminder email", map[string]interface{}{
		"user_id":   user.ID,
		"email":     user.Email.String,
		"template":  "daily_reminder",
		"subject":   "Time for your daily quiz! 🧠",
		"test_mode": true,
	})

	return nil
}

// SendWordOfTheDayEmail logs sending a word of the day email (test mode) and records it if a DB is available
func (e *TestEmailService) SendWordOfTheDayEmail(ctx context.Context, userID int, date time.Time, wordOfTheDay *models.WordOfTheDayDisplay) error {
	ctx, span := otel.Tracer("test-email-service").Start(ctx, "SendWordOfTheDayEmail",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.String("date", date.Format("2006-01-02")),
		),
	)
	defer span.End()

	if wordOfTheDay == nil {
		err := contextutils.ErrorWithContextf("word of the day data is nil")
		span.RecordError(err)
		return contextutils.ErrorWithContextf("word of the day data is nil")
	}

	span.SetAttributes(attribute.String("word", wordOfTheDay.Word))

	e.logger.Info(ctx, "TEST MODE: Would send word of the day email", map[string]interface{}{
		"user_id":   userID,
		"word":      wordOfTheDay.Word,
		"date":      date.Format("2006-01-02"),
		"template":  "word_of_the_day",
		"test_mode": true,
	})

	if e.db != nil {
		subject := fmt.Sprintf("Word of the Day: %s - %s", wordOfTheDay.Word, date.Format("January 2, 2006"))
		if err := e.RecordSentNotification(ctx, userID, "word_of_the_day", subject, "word_of_the_day", "sent", ""); err != nil {
			return contextutils.WrapError(err, "failed to record word of the day notification in test mode")
		}
	}

	return nil
}

// SendEmail sends a generic email with the given parameters (test mode - just logs)
func (e *TestEmailService) SendEmail(ctx context.Context, to, subject, templateName string, data map[string]interface{}) error {
	ctx, span := otel.Tracer("test-email-service").Start(ctx, "SendEmail",
		trace.WithAttributes(
			attribute.String("email.to", to),
			attribute.String("email.subject", subject),
			attribute.String("email.template", templateName),
		),
	)
	defer span.End()

	// Log the email operation instead of sending
	e.logger.Info(ctx, "TEST MODE: Would send email", map[string]interface{}{
		"to":        to,
		"subject":   subject,
		"template":  templateName,
		"test_mode": true,
		"data_keys": getMapKeys(data),
	})

	// Record the notification in the database if we have a DB connection
	if e.db != nil {
		// For test emails, we don't have a user ID, so we'll use 0
		err := e.RecordSentNotification(ctx, 0, "test_email", subject, templateName, "sent", "")
		if err != nil {
			e.logger.Error(ctx, "Failed to record test notification", err, map[string]interface{}{
				"to":       to,
				"template": templateName,
			})
		}
	}

	return nil
}

// HasSentWordOfTheDayEmail determines if a word-of-the-day email has already been sent for the provided date (test mode)
func (e *TestEmailService) HasSentWordOfTheDayEmail(ctx context.Context, userID int, date time.Time) (bool, error) {
	ctx, span := otel.Tracer("test-email-service").Start(ctx, "HasSentWordOfTheDayEmail",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.String("date", date.Format("2006-01-02")),
		),
	)
	defer span.End()

	if e.db == nil {
		// Without a database we cannot track sent notifications; act as if none was sent
		e.logger.Warn(ctx, "No database connection available for querying word-of-day history", map[string]interface{}{
			"user_id": userID,
			"date":    date.Format("2006-01-02"),
		})
		return false, nil
	}

	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	end := start.Add(24 * time.Hour)

	const query = `
		SELECT EXISTS(
			SELECT 1
			FROM sent_notifications
			WHERE user_id = $1
			  AND notification_type = 'word_of_the_day'
			  AND status = 'sent'
			  AND sent_at >= $2
			  AND sent_at < $3
		)
	`

	var exists bool
	if err := e.db.QueryRowContext(ctx, query, userID, start.UTC(), end.UTC()).Scan(&exists); err != nil {
		span.RecordError(err)
		return false, contextutils.WrapError(err, "failed to check word of the day notification history")
	}

	span.SetAttributes(attribute.Bool("word_of_day.already_sent", exists))

	return exists, nil
}

// RecordSentNotification records a sent notification in the database
func (e *TestEmailService) RecordSentNotification(ctx context.Context, userID int, notificationType, subject, templateName, status, errorMessage string) error {
	ctx, span := otel.Tracer("test-email-service").Start(ctx, "RecordSentNotification",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.String("notification.type", notificationType),
			attribute.String("notification.status", status),
		),
	)
	defer span.End()

	if e.db == nil {
		e.logger.Warn(ctx, "No database connection available for recording notification", map[string]interface{}{
			"user_id":           userID,
			"notification_type": notificationType,
		})
		return nil
	}

	query := `
		INSERT INTO sent_notifications (user_id, notification_type, subject, template_name, sent_at, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := e.db.ExecContext(ctx, query, userID, notificationType, subject, templateName, time.Now(), status, errorMessage)
	if err != nil {
		span.RecordError(err)
		e.logger.Error(ctx, "Failed to record sent notification", err, map[string]interface{}{
			"user_id":           userID,
			"notification_type": notificationType,
			"status":            status,
		})
		return contextutils.WrapError(err, "failed to record sent notification")
	}

	e.logger.Info(ctx, "Recorded sent notification", map[string]interface{}{
		"user_id":           userID,
		"notification_type": notificationType,
		"status":            status,
	})

	return nil
}

// IsEnabled returns whether email functionality is enabled (always true for test service)
func (e *TestEmailService) IsEnabled() bool {
	return true
}

// getMapKeys returns the keys of a map as a slice of strings
func getMapKeys(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}
