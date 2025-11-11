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

func TestUsageStatsService_UserAITokenUsageAggregations(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg := &config.Config{
		Translation: config.TranslationConfig{
			Quota: config.TranslationQuotaConfig{
				Enabled:             true,
				GoogleMonthlyQuota:  1000,
				DefaultMonthlyQuota: 500,
			},
		},
	}

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUsageStatsService(cfg, db, logger)
	userService := NewUserServiceWithLogger(db, cfg, logger)

	ctx := context.Background()

	user, err := userService.CreateUserWithPassword(ctx, "usage_stats_user", "password", "italian", "A1")
	require.NoError(t, err)

	baseDate := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
	nextDate := baseDate.AddDate(0, 0, 1)

	require.NoError(t, service.RecordUserAITokenUsage(ctx, user.ID, nil, "openai", "gpt-4", "chat", 10, 5, 15, 1))

	_, err = db.Exec(`
		INSERT INTO user_usage_stats (user_id, api_key_id, usage_date, usage_hour, service_name, provider, model, usage_type, prompt_tokens, completion_tokens, total_tokens, requests_made)
		VALUES
			($1, NULL, $2, 9, 'ai', 'openai', 'gpt-4', 'chat', 100, 40, 140, 2),
			($1, NULL, $2, 10, 'ai', 'openai', 'gpt-4', 'chat', 60, 20, 80, 1),
			($1, NULL, $3, 3, 'ai', 'anthropic', 'claude-3', 'story', 50, 30, 80, 1)
	`, user.ID, baseDate, nextDate)
	require.NoError(t, err)

	startRange := baseDate.AddDate(0, 0, -1)
	endRange := nextDate

	t.Run("raw usage stats", func(t *testing.T) {
		stats, err := service.GetUserAITokenUsageStats(ctx, user.ID, startRange, endRange)
		require.NoError(t, err)
		require.Len(t, stats, 3)

		byHour := make(map[int]int)
		for _, stat := range stats {
			if stat.UsageDate.Equal(nextDate) {
				assert.Equal(t, "anthropic", stat.Provider)
				assert.Equal(t, "claude-3", stat.Model)
			}
			byHour[stat.UsageHour] += stat.TotalTokens
		}
		assert.Equal(t, 140, byHour[9])
		assert.Equal(t, 80, byHour[10])
		assert.Equal(t, 80, byHour[3])
	})

	t.Run("daily aggregation", func(t *testing.T) {
		stats, err := service.GetUserAITokenUsageStatsByDay(ctx, user.ID, startRange, endRange)
		require.NoError(t, err)
		require.Len(t, stats, 2)

		var baseFound, nextFound bool
		for _, stat := range stats {
			if stat.UsageDate.Time.Equal(baseDate) {
				baseFound = true
				assert.Equal(t, 220, stat.TotalTokens)
				assert.Equal(t, 160, stat.TotalPromptTokens)
				assert.Equal(t, 60, stat.TotalCompletionTokens)
				assert.Equal(t, 3, stat.TotalRequests)
			} else if stat.UsageDate.Time.Equal(nextDate) {
				nextFound = true
				assert.Equal(t, 80, stat.TotalTokens)
				assert.Equal(t, 50, stat.TotalPromptTokens)
				assert.Equal(t, 30, stat.TotalCompletionTokens)
				assert.Equal(t, 1, stat.TotalRequests)
			}
		}
		assert.True(t, baseFound, "expected base date aggregate")
		assert.True(t, nextFound, "expected next date aggregate")
	})

	t.Run("record user ai token usage", func(t *testing.T) {
		today := time.Now().UTC().Truncate(24 * time.Hour)
		stats, err := service.GetUserAITokenUsageStats(ctx, user.ID, today, today)
		require.NoError(t, err)
		require.Len(t, stats, 1)
		assert.Equal(t, 15, stats[0].TotalTokens)
		assert.Equal(t, 1, stats[0].RequestsMade)
	})

	t.Run("hourly aggregation", func(t *testing.T) {
		stats, err := service.GetUserAITokenUsageStatsByHour(ctx, user.ID, baseDate)
		require.NoError(t, err)
		require.Len(t, stats, 2)

		for _, stat := range stats {
			switch stat.UsageHour {
			case 9:
				assert.Equal(t, 140, stat.TotalTokens)
				assert.Equal(t, 2, stat.TotalRequests)
			case 10:
				assert.Equal(t, 80, stat.TotalTokens)
				assert.Equal(t, 1, stat.TotalRequests)
			default:
				t.Fatalf("unexpected usage hour %d", stat.UsageHour)
			}
		}
	})
}

func TestUsageStatsService_AdminUsageQueries(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg := &config.Config{
		Translation: config.TranslationConfig{
			Quota: config.TranslationQuotaConfig{
				Enabled:             true,
				GoogleMonthlyQuota:  1000,
				DefaultMonthlyQuota: 500,
			},
		},
	}

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUsageStatsService(cfg, db, logger)

	ctx := context.Background()

	require.NoError(t, service.RecordUsage(ctx, "google", "translation", 250, 1))
	require.NoError(t, service.RecordUsage(ctx, "anthropic", "chat", 100, 2))

	allStats, err := service.GetAllUsageStats(ctx)
	require.NoError(t, err)
	require.Len(t, allStats, 2)

	googleStats, err := service.GetUsageStatsByService(ctx, "google")
	require.NoError(t, err)
	require.Len(t, googleStats, 1)
	assert.Equal(t, "translation", googleStats[0].UsageType)

	currentMonth := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -time.Now().UTC().Day()+1)
	monthStats, err := service.GetUsageStatsByMonth(ctx, currentMonth.Year(), int(currentMonth.Month()))
	require.NoError(t, err)
	require.Len(t, monthStats, 2)
}
