// Package services provides business logic services for the quiz application.
package services

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"strings"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	serviceinterfaces "quizapp/internal/services/interfaces"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/mail.v2"
)

// EmailService implements the interfaces.EmailService interface using gomail
type EmailService struct {
	cfg    *config.Config
	logger *observability.Logger
	dialer *mail.Dialer
	db     *sql.DB
}

// EmailServiceInterface defines the interface for email functionality
type EmailServiceInterface = serviceinterfaces.EmailService

// Ensure EmailService implements the EmailServiceInterface
var _ serviceinterfaces.EmailService = (*EmailService)(nil)

// NewEmailService creates a new EmailService instance
func NewEmailService(cfg *config.Config, logger *observability.Logger) *EmailService {
	var dialer *mail.Dialer
	if cfg.Email.Enabled && cfg.Email.SMTP.Host != "" {
		dialer = mail.NewDialer(
			cfg.Email.SMTP.Host,
			cfg.Email.SMTP.Port,
			cfg.Email.SMTP.Username,
			cfg.Email.SMTP.Password,
		)
	}

	return &EmailService{
		cfg:    cfg,
		logger: logger,
		dialer: dialer,
	}
}

// NewEmailServiceWithDB creates a new EmailService instance with database connection
func NewEmailServiceWithDB(cfg *config.Config, logger *observability.Logger, db *sql.DB) *EmailService {
	if db == nil {
		panic("EmailService requires a non-nil database connection")
	}

	var dialer *mail.Dialer
	if cfg.Email.Enabled && cfg.Email.SMTP.Host != "" {
		dialer = mail.NewDialer(
			cfg.Email.SMTP.Host,
			cfg.Email.SMTP.Port,
			cfg.Email.SMTP.Username,
			cfg.Email.SMTP.Password,
		)
	}

	return &EmailService{
		cfg:    cfg,
		logger: logger,
		dialer: dialer,
		db:     db,
	}
}

// SendDailyReminder sends a daily reminder email to a user
func (e *EmailService) SendDailyReminder(ctx context.Context, user *models.User) (err error) {
	ctx, span := otel.Tracer("email-service").Start(ctx, "SendDailyReminder",
		trace.WithAttributes(
			attribute.Int("user.id", user.ID),
			attribute.String("user.email", user.Email.String),
		),
	)
	defer observability.FinishSpan(span, &err)

	if !e.IsEnabled() {
		e.logger.Info(ctx, "Email disabled, skipping daily reminder", map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email.String,
		})
		return nil
	}

	if !user.Email.Valid || user.Email.String == "" {
		e.logger.Warn(ctx, "User has no email address, skipping daily reminder", map[string]interface{}{
			"user_id": user.ID,
		})
		return nil
	}

	// Determine daily goal from DB
	dailyGoal := 10
	var dg sql.NullInt64
	if err := e.db.QueryRowContext(ctx, "SELECT daily_goal FROM user_learning_preferences WHERE user_id = $1", user.ID).Scan(&dg); err == nil && dg.Valid {
		dailyGoal = int(dg.Int64)
	}

	// Generate email data
	data := map[string]interface{}{
		"Username":       user.Username,
		"QuizAppURL":     e.cfg.Server.AppBaseURL, // Frontend app URL for email links
		"CurrentDate":    time.Now().Format("January 2, 2006"),
		"DailyGoal":      dailyGoal,
		"UnsubscribeURL": fmt.Sprintf("%s/settings", e.cfg.Server.AppBaseURL),
	}

	subject := "Time for your daily quiz! 🧠"

	err = e.SendEmail(ctx, user.Email.String, subject, "daily_reminder", data)
	if err != nil {
		return contextutils.WrapError(err, "failed to send daily reminder")
	}

	e.logger.Info(ctx, "Daily reminder sent successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email.String,
	})

	return nil
}

// SendEmail sends a generic email with the given parameters
func (e *EmailService) SendEmail(ctx context.Context, to, subject, templateName string, data map[string]interface{}) (err error) {
	ctx, span := otel.Tracer("email-service").Start(ctx, "SendEmail",
		trace.WithAttributes(
			attribute.String("email.to", to),
			attribute.String("email.subject", subject),
			attribute.String("email.template", templateName),
		),
	)
	defer observability.FinishSpan(span, &err)

	if !e.IsEnabled() {
		e.logger.Info(ctx, "Email disabled, skipping email send", map[string]interface{}{
			"to":       to,
			"template": templateName,
		})
		return nil
	}

	if e.dialer == nil {
		return contextutils.ErrorWithContextf("email service not properly configured")
	}

	// Create email message
	m := mail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", e.cfg.Email.SMTP.FromName, e.cfg.Email.SMTP.FromAddress))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)

	// Generate email content from template
	content, err := e.generateEmailContent(templateName, data)
	if err != nil {
		return contextutils.WrapError(err, "failed to generate email content")
	}

	m.SetBody("text/html", content)

	// Send email
	if err = e.dialer.DialAndSend(m); err != nil {
		e.logger.Error(ctx, "Failed to send email", err, map[string]interface{}{
			"to":       to,
			"template": templateName,
			"subject":  subject,
		})
		return contextutils.WrapError(err, "failed to send email")
	}

	e.logger.Info(ctx, "Email sent successfully", map[string]interface{}{
		"to":       to,
		"template": templateName,
		"subject":  subject,
	})

	return nil
}

// RecordSentNotification records a sent notification in the database
func (e *EmailService) RecordSentNotification(ctx context.Context, userID int, notificationType, subject, templateName, status, errorMessage string) (err error) {
	ctx, span := otel.Tracer("email-service").Start(ctx, "RecordSentNotification",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.String("notification.type", notificationType),
			attribute.String("notification.status", status),
		),
	)
	defer observability.FinishSpan(span, &err)

	if e.db == nil {
		e.logger.Error(ctx, "Database connection is nil, cannot record notification", nil, map[string]interface{}{
			"user_id":           userID,
			"notification_type": notificationType,
		})
		return contextutils.ErrorWithContextf("EmailService database connection is nil")
	}

	query := `
		INSERT INTO sent_notifications (user_id, notification_type, subject, template_name, sent_at, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = e.db.ExecContext(ctx, query, userID, notificationType, subject, templateName, time.Now(), status, errorMessage)
	if err != nil {
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

// IsEnabled returns whether email functionality is enabled
func (e *EmailService) IsEnabled() bool {
	return e.cfg.Email.Enabled && e.cfg.Email.SMTP.Host != ""
}

// generateEmailContent generates email content from templates
func (e *EmailService) generateEmailContent(templateName string, data map[string]interface{}) (string, error) {
	// For now, we'll use a simple template system
	// In a real implementation, you might load templates from files or database
	switch templateName {
	case "daily_reminder":
		return e.generateDailyReminderTemplate(data)
	case "test_email":
		return e.generateTestEmailTemplate(data)
	default:
		return "", contextutils.ErrorWithContextf("unknown template: %s", templateName)
	}
}

// generateDailyReminderTemplate generates the daily reminder email template
func (e *EmailService) generateDailyReminderTemplate(data map[string]interface{}) (string, error) {
	const templateStr = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Daily Quiz Reminder</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { background-color: #f9f9f9; padding: 20px; }
        .button { display: inline-block; background-color: #4CAF50; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; margin: 20px 0; }
        .footer { background-color: #eee; padding: 15px; text-align: center; font-size: 12px; color: #666; border-radius: 0 0 5px 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🧠 Daily Quiz Reminder</h1>
        </div>
        <div class="content">
            <h2>Hello {{.Username}}!</h2>
            <p>It's {{.CurrentDate}} and time for your daily questions!</p>
            <p>Your goal today: <strong>{{.DailyGoal}} questions</strong></p>
            <p>Keep up the great work and continue improving your language skills!</p>
            <div style="text-align: center;">
                <a href="{{.QuizAppURL}}/daily" class="button">Start Your Daily Questions</a>
            </div>
        </div>
        <div class="footer">
            <p>This email was sent by Quiz App. If you no longer wish to receive these reminders, you can <a href="{{.UnsubscribeURL}}">unsubscribe here</a>.</p>
        </div>
    </div>
</body>
</html>`

	tmpl, err := template.New("daily_reminder").Parse(templateStr)
	if err != nil {
		return "", contextutils.WrapError(err, "failed to parse template")
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", contextutils.WrapError(err, "failed to execute template")
	}

	return buf.String(), nil
}

// generateTestEmailTemplate generates the test email template
func (e *EmailService) generateTestEmailTemplate(data map[string]interface{}) (string, error) {
	const templateStr = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Test Email</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #2196F3; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { background-color: #f9f9f9; padding: 20px; }
        .footer { background-color: #eee; padding: 15px; text-align: center; font-size: 12px; color: #666; border-radius: 0 0 5px 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📧 Test Email</h1>
        </div>
        <div class="content">
            <h2>Hello {{.Username}}!</h2>
            <p>This is a test email to verify that your email settings are working correctly.</p>
            <p><strong>Test Time:</strong> {{.TestTime}}</p>
            <p><strong>Message:</strong> {{.Message}}</p>
            <p>If you received this email, your email configuration is working properly!</p>
        </div>
        <div class="footer">
            <p>This is a test email from Quiz App. No action is required.</p>
        </div>
    </div>
</body>
</html>
`

	tmpl, err := template.New("test_email").Parse(templateStr)
	if err != nil {
		return "", contextutils.WrapError(err, "failed to parse template")
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", contextutils.WrapError(err, "failed to execute template")
	}

	return buf.String(), nil
}
