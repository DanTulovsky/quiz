package serviceinterfaces

import (
	"context"
	"testing"

	"quizapp/internal/models"

	"github.com/stretchr/testify/assert"
)

// MockEmailService implements EmailService for testing
type MockEmailService struct {
	SendDailyReminderCalled      bool
	SendEmailCalled              bool
	RecordSentNotificationCalled bool
	IsEnabledResult              bool
}

func (m *MockEmailService) SendDailyReminder(ctx context.Context, user *models.User) error {
	m.SendDailyReminderCalled = true
	return nil
}

func (m *MockEmailService) SendEmail(ctx context.Context, to, subject, templateName string, data map[string]interface{}) error {
	m.SendEmailCalled = true
	return nil
}

func (m *MockEmailService) RecordSentNotification(ctx context.Context, userID int, notificationType, subject, templateName, status, errorMessage string) error {
	m.RecordSentNotificationCalled = true
	return nil
}

func (m *MockEmailService) IsEnabled() bool {
	return m.IsEnabledResult
}

// MockLifecycleService implements Lifecycle for testing
type MockLifecycleService struct {
	StartupCalled  bool
	ShutdownCalled bool
	IsReadyResult  bool
}

func (m *MockLifecycleService) Startup(ctx context.Context) error {
	m.StartupCalled = true
	return nil
}

func (m *MockLifecycleService) Shutdown(ctx context.Context) error {
	m.ShutdownCalled = true
	return nil
}

func (m *MockLifecycleService) IsReady() bool {
	return m.IsReadyResult
}

func TestEmailServiceInterface_Implementation(t *testing.T) {
	// Test that our mock implements the interface
	var _ EmailService = (*MockEmailService)(nil)

	mock := &MockEmailService{}

	// Test interface methods
	ctx := context.Background()
	user := &models.User{ID: 1, Username: "test"}

	err := mock.SendDailyReminder(ctx, user)
	assert.NoError(t, err)
	assert.True(t, mock.SendDailyReminderCalled)

	err = mock.SendEmail(ctx, "test@example.com", "Test Subject", "test_template", map[string]interface{}{})
	assert.NoError(t, err)
	assert.True(t, mock.SendEmailCalled)

	err = mock.RecordSentNotification(ctx, 1, "test_type", "Test Subject", "test_template", "sent", "")
	assert.NoError(t, err)
	assert.True(t, mock.RecordSentNotificationCalled)

	enabled := mock.IsEnabled()
	assert.False(t, enabled) // Default value

	mock.IsEnabledResult = true
	enabled = mock.IsEnabled()
	assert.True(t, enabled)
}

func TestLifecycleInterface_Implementation(t *testing.T) {
	// Test that our mock implements the interface
	var _ Lifecycle = (*MockLifecycleService)(nil)

	mock := &MockLifecycleService{}

	// Test interface methods
	ctx := context.Background()

	err := mock.Startup(ctx)
	assert.NoError(t, err)
	assert.True(t, mock.StartupCalled)

	err = mock.Shutdown(ctx)
	assert.NoError(t, err)
	assert.True(t, mock.ShutdownCalled)

	ready := mock.IsReady()
	assert.False(t, ready) // Default value

	mock.IsReadyResult = true
	ready = mock.IsReady()
	assert.True(t, ready)
}

func TestInterface_MethodSignatures(t *testing.T) {
	// Test that interfaces have the expected method signatures
	// This is mainly compile-time verification that interfaces are properly defined

	// Test that we can create instances of the mocks (proves interfaces are implemented)
	emailService := &MockEmailService{}
	assert.NotNil(t, emailService)

	lifecycle := &MockLifecycleService{}
	assert.NotNil(t, lifecycle)

	// Verify interface compliance at compile time
	var _ EmailService = emailService
	var _ Lifecycle = lifecycle
}

func TestInterface_Compatibility(t *testing.T) {
	// Test that interfaces can be used polymorphically
	var emailServices []EmailService
	var lifecycleServices []Lifecycle

	mockEmail := &MockEmailService{}
	mockLifecycle := &MockLifecycleService{}

	emailServices = append(emailServices, mockEmail)
	lifecycleServices = append(lifecycleServices, mockLifecycle)

	// Should be able to call interface methods
	ctx := context.Background()

	for _, service := range emailServices {
		err := service.SendEmail(ctx, "test@example.com", "Test", "template", nil)
		assert.NoError(t, err)
	}

	for _, service := range lifecycleServices {
		err := service.Startup(ctx)
		assert.NoError(t, err)
	}
}
