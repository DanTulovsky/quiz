package contextutils

import (
	"context"
	"time"

	"quizapp/internal/models"
)

// ParseDateInUserTimezone parses a YYYY-MM-DD date string in the user's timezone.
// The userLookup function is injected to fetch the user (to avoid tight coupling and enable testing).
// Returns the parsed time (in the location), the effective timezone name (or "UTC" on fallback), and an error.
// If the date format is invalid, the returned error will be wrapped with the message "invalid date format".
func ParseDateInUserTimezone(
	ctx context.Context,
	userID int,
	dateStr string,
	userLookup func(context.Context, int) (*models.User, error),
) (time.Time, string, error) {
	user, err := userLookup(ctx, userID)
	if err != nil {
		return time.Time{}, "", err
	}

	timezone := "UTC"
	if user != nil && user.Timezone.Valid && user.Timezone.String != "" {
		timezone = user.Timezone.String
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Fallback to UTC if invalid timezone
		loc = time.UTC
		timezone = "UTC"
	}

	date, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return time.Time{}, timezone, WrapError(err, "invalid date format")
	}

	return date, timezone, nil
}

// ConvertTimeToUserLocation converts the provided time to the user's timezone.
// Returns the converted time and the effective timezone name (or "UTC" on fallback).
func ConvertTimeToUserLocation(
	ctx context.Context,
	userID int,
	t time.Time,
	userLookup func(context.Context, int) (*models.User, error),
) (time.Time, string, error) {
	user, err := userLookup(ctx, userID)
	if err != nil {
		return time.Time{}, "", err
	}

	timezone := "UTC"
	if user != nil && user.Timezone.Valid && user.Timezone.String != "" {
		timezone = user.Timezone.String
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
		timezone = "UTC"
	}

	return t.In(loc), timezone, nil
}

// FormatTimeInUserTimezone formats the provided time in the user's timezone using the given layout.
// Returns the formatted string and the effective timezone name.
func FormatTimeInUserTimezone(
	ctx context.Context,
	userID int,
	t time.Time,
	layout string,
	userLookup func(context.Context, int) (*models.User, error),
) (string, string, error) {
	// If the stored timestamp is exactly midnight UTC with zero nanoseconds,
	// it may be a date-only value (missing timezone). We only treat it as
	// missing if the user has a configured timezone that is not UTC.
	if t.Location() == time.UTC && t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
		if userLookup != nil {
			if u, err := userLookup(ctx, userID); err == nil && u != nil && u.Timezone.Valid && u.Timezone.String != "" && u.Timezone.String != "UTC" {
				return "", "", ErrTimestampMissingTimezone
			}
		}
	}

	tt, tz, err := ConvertTimeToUserLocation(ctx, userID, t, userLookup)
	if err != nil {
		return "", tz, err
	}
	res := tt.Format(layout)
	return res, tz, nil
}

// UserLocalDayRange returns the UTC start and end timestamps that cover the
// last `days` calendar days for the given user in their configured timezone.
// The range is [startUTC, endUTC) where startUTC is the start of the earliest
// local day at 00:00 and endUTC is the start of the day after "today" at 00:00
// in UTC. The userLookup function is used to fetch the user's timezone.
func UserLocalDayRange(ctx context.Context, userID, days int, userLookup func(context.Context, int) (*models.User, error)) (time.Time, time.Time, string, error) {
	if days <= 0 {
		days = 1
	}
	user, err := userLookup(ctx, userID)
	if err != nil {
		return time.Time{}, time.Time{}, "", err
	}

	timezone := "UTC"
	if user != nil && user.Timezone.Valid && user.Timezone.String != "" {
		timezone = user.Timezone.String
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
		timezone = "UTC"
	}

	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	startLocal := today.AddDate(0, 0, -(days - 1))
	// start of the day after today
	endLocal := today.Add(24 * time.Hour)

	startUTC := startLocal.UTC()
	endUTC := endLocal.UTC()
	return startUTC, endUTC, timezone, nil
}
