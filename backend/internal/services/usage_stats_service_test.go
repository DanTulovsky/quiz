//go:build integration

package services

import (
	"context"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsageStatsService_CheckQuota(t *testing.T) {
	// Use shared test database setup
	db := SharedTestDBSetup(t)
	defer db.Close()

	// Create a test config with quotas enabled
	cfg := &config.Config{
		Translation: config.TranslationConfig{
			Quota: config.TranslationQuotaConfig{
				Enabled:             true,
				GoogleMonthlyQuota:  1000, // Small quota for testing
				DefaultMonthlyQuota: 500,
			},
		},
	}

	// Create logger and service
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUsageStatsService(cfg, db, logger)

	ctx := context.Background()

	t.Run("Should allow usage within quota", func(t *testing.T) {
		// Check quota for 100 characters (within 1000 limit)
		err := service.CheckQuota(ctx, "google", "translation", 100)
		assert.NoError(t, err)
	})

	t.Run("Should deny usage exceeding quota", func(t *testing.T) {
		// First record some usage to get close to quota
		err := service.RecordUsage(ctx, "google", "translation", 900, 1)
		require.NoError(t, err)

		// Now check quota for 200 characters (would exceed 1000 limit: 900 + 200 = 1100)
		err = service.CheckQuota(ctx, "google", "translation", 200)
		assert.Error(t, err)

		// Check that it's a structured error with the correct code
		var appErr *contextutils.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, contextutils.ErrorCodeQuotaExceeded, appErr.Code)
	})

	t.Run("Should work with different services", func(t *testing.T) {
		// Check quota for a different service
		err := service.CheckQuota(ctx, "azure", "translation", 100)
		assert.NoError(t, err) // Should use default quota (500)
	})
}

func TestUsageStatsService_RecordUsage(t *testing.T) {
	// Use shared test database setup
	db := SharedTestDBSetup(t)
	defer db.Close()

	// Create a test config
	cfg := &config.Config{
		Translation: config.TranslationConfig{
			Quota: config.TranslationQuotaConfig{
				Enabled:             true,
				GoogleMonthlyQuota:  1000,
				DefaultMonthlyQuota: 500,
			},
		},
	}

	// Create logger and service
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUsageStatsService(cfg, db, logger)

	ctx := context.Background()

	t.Run("Should record usage correctly", func(t *testing.T) {
		currentMonth := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -time.Now().UTC().Day()+1)

		// Record usage
		err := service.RecordUsage(ctx, "google", "translation", 500, 2)
		require.NoError(t, err)

		// Verify it was recorded
		usage, err := service.GetCurrentMonthUsage(ctx, "google", "translation")
		require.NoError(t, err)
		assert.Equal(t, 500, usage.CharactersUsed)
		assert.Equal(t, 2, usage.RequestsMade)
		// Compare year, month, day instead of exact time to avoid timezone issues
		assert.Equal(t, currentMonth.Year(), usage.UsageMonth.Year())
		assert.Equal(t, currentMonth.Month(), usage.UsageMonth.Month())
		assert.Equal(t, currentMonth.Day(), usage.UsageMonth.Day())
	})

	t.Run("Should accumulate usage", func(t *testing.T) {
		// Record more usage
		err := service.RecordUsage(ctx, "google", "translation", 300, 1)
		require.NoError(t, err)

		// Verify it accumulated
		usage, err := service.GetCurrentMonthUsage(ctx, "google", "translation")
		require.NoError(t, err)
		assert.Equal(t, 800, usage.CharactersUsed) // 500 + 300
		assert.Equal(t, 3, usage.RequestsMade)     // 2 + 1
	})
}
