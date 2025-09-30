package contextutils

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"quizapp/internal/models"

	"github.com/stretchr/testify/require"
)

func TestParseDateInUserTimezone_ValidTimezone(t *testing.T) {
	user := &models.User{ID: 1, Timezone: sql.NullString{String: "America/Los_Angeles", Valid: true}}
	userLookup := func(context.Context, int) (*models.User, error) { return user, nil }
	date, tz, err := ParseDateInUserTimezone(context.Background(), 1, "2025-08-19", userLookup)
	require.NoError(t, err)
	require.Equal(t, "America/Los_Angeles", tz)
	// Ensure parsed time has local location
	require.Equal(t, 0, date.Hour())
}

func TestUserLocalDayRange_DefaultUTC(t *testing.T) {
	userLookup := func(context.Context, int) (*models.User, error) { return nil, nil }
	start, end, tz, err := UserLocalDayRange(context.Background(), 1, 2, userLookup)
	require.NoError(t, err)
	require.Equal(t, "UTC", tz)
	// start should be before end by exactly 3 days-1? check ordering
	require.True(t, start.Before(end))
}

func TestFormatTimeInUserTimezone_DateOnlyMissingTimezone(t *testing.T) {
	// Time exactly midnight UTC
	midnightUTC := time.Date(2025, 8, 19, 0, 0, 0, 0, time.UTC)
	user := &models.User{ID: 1, Timezone: sql.NullString{String: "America/Los_Angeles", Valid: true}}
	userLookup := func(context.Context, int) (*models.User, error) { return user, nil }
	_, _, err := FormatTimeInUserTimezone(context.Background(), 1, midnightUTC, time.RFC3339, userLookup)
	require.ErrorIs(t, err, ErrTimestampMissingTimezone)
}

func TestUserLocalDayRange_WithTimezone(t *testing.T) {
	user := &models.User{ID: 1, Timezone: sql.NullString{String: "America/Los_Angeles", Valid: true}}
	userLookup := func(context.Context, int) (*models.User, error) { return user, nil }
	start, end, tz, err := UserLocalDayRange(context.Background(), 1, 1, userLookup)
	require.NoError(t, err)
	require.Equal(t, "America/Los_Angeles", tz)
	// start and end should be UTC times where end-start == 24h
	require.Equal(t, 24*time.Hour, end.Sub(start))
}
