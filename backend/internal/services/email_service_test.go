package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services/mailer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMailer is a mock implementation of the Mailer interface for testing
type MockMailer struct {
	mock.Mock
}

func (m *MockMailer) SendDailyReminder(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockMailer) SendEmail(ctx context.Context, to, subject, templateName string, data map[string]interface{}) error {
	args := m.Called(ctx, to, subject, templateName, data)
	return args.Error(0)
}

func (m *MockMailer) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

// createTestLogger creates a logger for testing
func createTestLogger() *observability.Logger {
	cfg := &config.OpenTelemetryConfig{
		EnableLogging: false, // Disable logging for tests
	}
	return observability.NewLogger(cfg)
}

func TestNewEmailService(t *testing.T) {
	// Test with email enabled
	cfg := &config.Config{
		Email: config.EmailConfig{
			Enabled: true,
			SMTP: config.SMTPConfig{
				Host:        "smtp.gmail.com",
				Port:        587,
				Username:    "test@example.com",
				Password:    "password",
				FromAddress: "noreply@example.com",
				FromName:    "Test App",
			},
		},
	}

	logger := createTestLogger()
	service := NewEmailService(cfg, logger)

	assert.NotNil(t, service)
	assert.True(t, service.IsEnabled())
}

func TestNewEmailService_Disabled(t *testing.T) {
	// Test with email disabled
	cfg := &config.Config{
		Email: config.EmailConfig{
			Enabled: false,
		},
	}

	logger := createTestLogger()
	service := NewEmailService(cfg, logger)

	assert.NotNil(t, service)
	assert.False(t, service.IsEnabled())
}

func TestEmailService_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		expected bool
	}{
		{
			name: "enabled with valid config",
			cfg: &config.Config{
				Email: config.EmailConfig{
					Enabled: true,
					SMTP: config.SMTPConfig{
						Host: "smtp.gmail.com",
					},
				},
			},
			expected: true,
		},
		{
			name: "disabled",
			cfg: &config.Config{
				Email: config.EmailConfig{
					Enabled: false,
				},
			},
			expected: false,
		},
		{
			name: "enabled but no host",
			cfg: &config.Config{
				Email: config.EmailConfig{
					Enabled: true,
					SMTP: config.SMTPConfig{
						Host: "",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := createTestLogger()
			service := NewEmailService(tt.cfg, logger)
			assert.Equal(t, tt.expected, service.IsEnabled())
		})
	}
}

func TestEmailService_SendDailyReminder_Disabled(t *testing.T) {
	cfg := &config.Config{
		Email: config.EmailConfig{
			Enabled: false,
		},
	}

	logger := createTestLogger()
	service := NewEmailService(cfg, logger)

	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    sql.NullString{String: "test@example.com", Valid: true},
	}

	err := service.SendDailyReminder(context.Background(), user)
	assert.NoError(t, err) // Should not error when disabled
}

func TestEmailService_SendDailyReminder_NoEmail(t *testing.T) {
	cfg := &config.Config{
		Email: config.EmailConfig{
			Enabled: true,
			SMTP: config.SMTPConfig{
				Host:        "smtp.gmail.com",
				Port:        587,
				Username:    "test@example.com",
				Password:    "password",
				FromAddress: "noreply@example.com",
				FromName:    "Test App",
			},
		},
	}

	logger := createTestLogger()
	service := NewEmailService(cfg, logger)

	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    sql.NullString{String: "", Valid: false},
	}

	err := service.SendDailyReminder(context.Background(), user)
	assert.NoError(t, err) // Should not error when user has no email
}

func TestEmailService_GenerateDailyReminderTemplate(t *testing.T) {
	cfg := &config.Config{
		Email: config.EmailConfig{
			Enabled: true,
			SMTP: config.SMTPConfig{
				Host:        "smtp.gmail.com",
				Port:        587,
				Username:    "test@example.com",
				Password:    "password",
				FromAddress: "noreply@example.com",
				FromName:    "Test App",
			},
		},
		Server: config.ServerConfig{
			AppBaseURL: "http://localhost:3000",
		},
	}

	logger := createTestLogger()
	service := NewEmailService(cfg, logger)

	data := map[string]interface{}{
		"Username":       "testuser",
		"QuizAppURL":     "http://localhost:8080",
		"CurrentDate":    time.Now().Format("January 2, 2006"),
		"DailyGoal":      10,
		"UnsubscribeURL": "http://localhost:8080/settings",
	}

	content, err := service.generateEmailContent("daily_reminder", data)
	assert.NoError(t, err)
	assert.Contains(t, content, "Hello testuser!")
	assert.Contains(t, content, "Daily Quiz Reminder")
	assert.Contains(t, content, "10 questions")
	assert.Contains(t, content, "Start Your Daily Questions")
}

func TestEmailService_GenerateEmailContent_UnknownTemplate(t *testing.T) {
	cfg := &config.Config{
		Email: config.EmailConfig{
			Enabled: true,
		},
	}

	logger := createTestLogger()
	service := NewEmailService(cfg, logger)

	_, err := service.generateEmailContent("unknown_template", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown template")
}

func TestMockMailer(t *testing.T) {
	mockMailer := &MockMailer{}
	ctx := context.Background()
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    sql.NullString{String: "test@example.com", Valid: true},
	}

	// Test SendDailyReminder
	mockMailer.On("SendDailyReminder", ctx, user).Return(nil)
	err := mockMailer.SendDailyReminder(ctx, user)
	assert.NoError(t, err)
	mockMailer.AssertExpectations(t)

	// Test SendEmail
	data := map[string]interface{}{"test": "data"}
	mockMailer.On("SendEmail", ctx, "test@example.com", "Test Subject", "test_template", data).Return(nil)
	err = mockMailer.SendEmail(ctx, "test@example.com", "Test Subject", "test_template", data)
	assert.NoError(t, err)
	mockMailer.AssertExpectations(t)

	// Test IsEnabled
	mockMailer.On("IsEnabled").Return(true)
	enabled := mockMailer.IsEnabled()
	assert.True(t, enabled)
	mockMailer.AssertExpectations(t)
}

// TestEmailServiceInterface ensures EmailService implements the Mailer interface
func TestEmailServiceInterface(_ *testing.T) {
	var _ mailer.Mailer = (*EmailService)(nil)
}

func TestEmailService_GenerateTestEmailTemplate(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			AppBaseURL: "http://localhost:3000",
		},
		Email: config.EmailConfig{
			Enabled: true,
			SMTP: config.SMTPConfig{
				Host:        "smtp.example.com",
				Port:        587,
				Username:    "user",
				Password:    "pass",
				FromName:    "Quiz App",
				FromAddress: "noreply@quizapp.com",
			},
		},
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewEmailService(cfg, logger)

	data := map[string]interface{}{
		"Username": "testuser",
		"TestTime": "January 15, 2024 10:30:00",
		"Message":  "This is a test email",
	}

	content, err := service.generateTestEmailTemplate(data)

	assert.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, content, "testuser")
	assert.Contains(t, content, "Test Email")
	assert.Contains(t, content, "📧")
	assert.Contains(t, content, "This is a test email")
	assert.Contains(t, content, "January 15, 2024 10:30:00")
}

func TestEmailService_SendEmail(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.Config
		to            string
		subject       string
		template      string
		data          map[string]interface{}
		expectedError bool
	}{
		{
			name: "email disabled",
			cfg: &config.Config{
				Email: config.EmailConfig{
					Enabled: false,
					SMTP: config.SMTPConfig{
						Host:        "smtp.example.com",
						Port:        587,
						Username:    "user",
						Password:    "pass",
						FromName:    "Quiz App",
						FromAddress: "noreply@quizapp.com",
					},
				},
			},
			to:            "test@example.com",
			subject:       "Test Subject",
			template:      "test_email",
			data:          map[string]interface{}{},
			expectedError: false, // Should not error, just skip
		},
		{
			name: "email enabled but no dialer",
			cfg: &config.Config{
				Email: config.EmailConfig{
					Enabled: true,
					SMTP: config.SMTPConfig{
						Host:        "", // Empty host should cause no dialer
						Port:        587,
						Username:    "user",
						Password:    "pass",
						FromName:    "Quiz App",
						FromAddress: "noreply@quizapp.com",
					},
				},
			},
			to:            "test@example.com",
			subject:       "Test Subject",
			template:      "test_email",
			data:          map[string]interface{}{},
			expectedError: false, // Should not error because IsEnabled() returns false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
			service := NewEmailService(tt.cfg, logger)

			err := service.SendEmail(context.Background(), tt.to, tt.subject, tt.template, tt.data)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailService_TemplateParsing(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			AppBaseURL: "http://localhost:3000",
		},
		Email: config.EmailConfig{
			Enabled: true,
			SMTP: config.SMTPConfig{
				Host:        "smtp.example.com",
				Port:        587,
				Username:    "user",
				Password:    "pass",
				FromName:    "Quiz App",
				FromAddress: "noreply@quizapp.com",
			},
		},
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewEmailService(cfg, logger)

	// Test that templates can handle various data types
	testData := map[string]interface{}{
		"String":      "test string",
		"Int":         42,
		"Float":       3.14,
		"Bool":        true,
		"Slice":       []string{"item1", "item2"},
		"Map":         map[string]string{"key": "value"},
		"Nil":         nil,
		"EmptyString": "",
	}

	// Test daily reminder template
	content, err := service.generateDailyReminderTemplate(testData)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)

	// Test test email template
	content, err = service.generateTestEmailTemplate(testData)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestEmailService_DatabaseNilPanic(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			AppBaseURL: "http://localhost:3000",
		},
		Email: config.EmailConfig{
			Enabled: true,
			SMTP: config.SMTPConfig{
				Host:        "smtp.example.com",
				Port:        587,
				Username:    "user",
				Password:    "pass",
				FromName:    "Quiz App",
				FromAddress: "noreply@quizapp.com",
			},
		},
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Test that NewEmailServiceWithDB panics when db is nil
	assert.Panics(t, func() {
		NewEmailServiceWithDB(cfg, logger, nil)
	}, "EmailServiceWithDB should panic when database is nil")
}

func TestEmailService_ContextHandling(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			AppBaseURL: "http://localhost:3000",
		},
		Email: config.EmailConfig{
			Enabled: true,
			SMTP: config.SMTPConfig{
				Host:        "smtp.example.com",
				Port:        587,
				Username:    "user",
				Password:    "pass",
				FromName:    "Quiz App",
				FromAddress: "noreply@quizapp.com",
			},
		},
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewTestEmailService(cfg, logger)

	// Test with context that has timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    sql.NullString{String: "test@example.com", Valid: true},
	}

	// This should not panic and should handle the context properly
	err := service.SendDailyReminder(ctx, user)
	// TestEmailService should return nil (success) since it just logs
	assert.NoError(t, err)
}
