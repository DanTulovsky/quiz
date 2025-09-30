// Package mailer defines service interfaces for the quiz application.
package mailer

import (
	"context"

	"quizapp/internal/models"
)

// Mailer defines the interface for email sending functionality
type Mailer interface {
	// SendDailyReminder sends a daily reminder email to a user
	SendDailyReminder(ctx context.Context, user *models.User) error

	// SendEmail sends a generic email with the given parameters
	SendEmail(ctx context.Context, to, subject, templateName string, data map[string]interface{}) error

	// IsEnabled returns whether email functionality is enabled
	IsEnabled() bool

	// RecordSentNotification records a sent notification in the database
	RecordSentNotification(ctx context.Context, userID int, notificationType, subject, templateName, status, errorMessage string) error
}
