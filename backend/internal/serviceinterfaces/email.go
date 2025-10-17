// Package serviceinterfaces defines service interfaces for dependency injection and testing.
package serviceinterfaces

import (
	"context"

	"quizapp/internal/models"
)

// EmailService defines the interface for email functionality
type EmailService interface {
	// SendDailyReminder sends a daily reminder email to a user
	SendDailyReminder(ctx context.Context, user *models.User) error

	// SendEmail sends a generic email with the given parameters
	SendEmail(ctx context.Context, to, subject, templateName string, data map[string]interface{}) error

	// RecordSentNotification records a notification in the database
	RecordSentNotification(ctx context.Context, userID int, notificationType, subject, templateName, status, errorMessage string) error

	// IsEnabled returns whether email functionality is enabled
	IsEnabled() bool
}
