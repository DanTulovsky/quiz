package mailer

import (
	"context"
	"testing"

	"quizapp/internal/models"

	"github.com/stretchr/testify/assert"
)

// MockMailer implements Mailer for testing
type MockMailer struct {
	SendDailyReminderCalled      bool
	SendEmailCalled              bool
	RecordSentNotificationCalled bool
	IsEnabledResult              bool
}

func (m *MockMailer) SendDailyReminder(_ context.Context, _ *models.User) error {
	m.SendDailyReminderCalled = true
	return nil
}

func (m *MockMailer) SendEmail(_ context.Context, _, _, _ string, _ map[string]interface{}) error {
	m.SendEmailCalled = true
	return nil
}

func (m *MockMailer) RecordSentNotification(_ context.Context, _ int, _, _, _, _, _ string) error {
	m.RecordSentNotificationCalled = true
	return nil
}

func (m *MockMailer) IsEnabled() bool {
	return m.IsEnabledResult
}

func TestMailerInterface_Implementation(t *testing.T) {
	// Test that our mock implements the interface
	var _ Mailer = (*MockMailer)(nil)

	mock := &MockMailer{}

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

func TestMailerInterface_MethodSignatures(t *testing.T) {
	// Test that interface has the expected method signatures
	// This is mainly compile-time verification that interface is properly defined

	// Test that we can create instances of the mock (proves interface is implemented)
	mailer := &MockMailer{}
	assert.NotNil(t, mailer)

	// Verify interface compliance at compile time
	var _ Mailer = mailer
}

func TestMailerInterface_Compatibility(t *testing.T) {
	// Test that interface can be used polymorphically
	var mailers []Mailer

	mockMailer := &MockMailer{}
	mailers = append(mailers, mockMailer)

	// Should be able to call interface methods
	ctx := context.Background()

	for _, mailer := range mailers {
		err := mailer.SendEmail(ctx, "test@example.com", "Test", "template", nil)
		assert.NoError(t, err)
	}
}
