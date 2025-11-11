//go:build integration

package services

import (
	"context"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailService_HasSentWordOfTheDayEmail_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	cfg, err := config.NewConfig()
	require.NoError(t, err)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	emailService := NewEmailServiceWithDB(cfg, logger, db)

	ctx := context.Background()

	var userID int
	err = db.QueryRowContext(ctx, `
		INSERT INTO users (username, email, word_of_day_email_enabled)
		VALUES ($1, $2, $3)
		RETURNING id
	`, "wotd_integration_user", "wotd_integration@example.com", true).Scan(&userID)
	require.NoError(t, err)

	loc := time.FixedZone("UTC-5", -5*60*60)
	testDate := time.Date(2025, time.November, 10, 0, 0, 0, 0, loc)

	sent, err := emailService.HasSentWordOfTheDayEmail(ctx, userID, testDate)
	require.NoError(t, err)
	assert.False(t, sent, "expected no email to be recorded initially")

	subject := "Word of the Day: integration - November 10, 2025"
	_, err = db.ExecContext(ctx, `
		INSERT INTO sent_notifications (user_id, notification_type, subject, template_name, sent_at, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, userID, "word_of_the_day", subject, "word_of_the_day", testDate.In(time.UTC), "sent", nil)
	require.NoError(t, err)

	sent, err = emailService.HasSentWordOfTheDayEmail(ctx, userID, testDate)
	require.NoError(t, err)
	assert.True(t, sent, "expected email to be detected after recording notification")

	nextDay := testDate.Add(24 * time.Hour)
	sent, err = emailService.HasSentWordOfTheDayEmail(ctx, userID, nextDay)
	require.NoError(t, err)
	assert.False(t, sent, "expected no email to be recorded for the following day")
}
