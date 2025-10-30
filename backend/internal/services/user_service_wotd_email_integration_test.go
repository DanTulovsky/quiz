//go:build integration

package services

import (
	"context"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/require"
)

func TestUserService_UpdateWordOfDayEmailEnabled_Integration(t *testing.T) {
	// Arrange: real DB per integration pattern
	db := SharedTestDBSetup(t)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)

	userSvc := NewUserServiceWithLogger(db, cfg, logger)

	// Create a user
	user, err := userSvc.CreateUser(context.Background(), "wotd-email-user", "italian", "A2")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Act: enable emails
	err = userSvc.UpdateWordOfDayEmailEnabled(context.Background(), user.ID, true)
	require.NoError(t, err)

	// Assert: fetched user reflects enabled
	refetched, err := userSvc.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, refetched)
	require.True(t, refetched.WordOfDayEmailEnabled.Valid)
	require.True(t, refetched.WordOfDayEmailEnabled.Bool)

	// Act: disable emails
	err = userSvc.UpdateWordOfDayEmailEnabled(context.Background(), user.ID, false)
	require.NoError(t, err)

	// Assert: fetched user reflects disabled
	refetched2, err := userSvc.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, refetched2)
	require.True(t, refetched2.WordOfDayEmailEnabled.Valid)
	require.False(t, refetched2.WordOfDayEmailEnabled.Bool)
}
