package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.opentelemetry.io/otel/attribute"
)

// UsageStatsServiceInterface defines the interface for usage statistics tracking
type UsageStatsServiceInterface interface {
	// CheckQuota checks if a translation request would exceed the monthly quota
	CheckQuota(ctx context.Context, serviceName, usageType string, characters int) error
	// RecordUsage records the usage of a translation service
	RecordUsage(ctx context.Context, serviceName, usageType string, characters, requests int) error
	// GetCurrentMonthUsage returns the current month's usage for a service and type
	GetCurrentMonthUsage(ctx context.Context, serviceName, usageType string) (*UsageStats, error)
	// GetMonthlyQuota returns the monthly quota for a service
	GetMonthlyQuota(serviceName string) int64
	// GetAllUsageStats returns all usage statistics (for admin interface)
	GetAllUsageStats(ctx context.Context) ([]*UsageStats, error)
	// GetUsageStatsByService returns usage statistics for a specific service
	GetUsageStatsByService(ctx context.Context, serviceName string) ([]*UsageStats, error)
	// GetUsageStatsByMonth returns usage statistics for a specific month
	GetUsageStatsByMonth(ctx context.Context, year, month int) ([]*UsageStats, error)

	// AI Token usage tracking for users
	// RecordUserAITokenUsage records AI token usage for a specific user
	RecordUserAITokenUsage(ctx context.Context, userID int, apiKeyID *int, provider, model, usageType string, promptTokens, completionTokens, totalTokens, requests int) error
	// GetUserAITokenUsageStats returns AI token usage statistics for a specific user
	GetUserAITokenUsageStats(ctx context.Context, userID int, startDate, endDate time.Time) ([]*UserUsageStats, error)
	// GetUserAITokenUsageStatsByDay returns daily aggregated AI token usage for a user
	GetUserAITokenUsageStatsByDay(ctx context.Context, userID int, startDate, endDate time.Time) ([]*UserUsageStatsDaily, error)
	// GetUserAITokenUsageStatsByHour returns hourly aggregated AI token usage for a user on a specific day
	GetUserAITokenUsageStatsByHour(ctx context.Context, userID int, date time.Time) ([]*UserUsageStatsHourly, error)
}

// UsageStats represents usage statistics for a service in a given month
type UsageStats struct {
	ID             int       `json:"id"`
	ServiceName    string    `json:"service_name"`
	UsageType      string    `json:"usage_type"`
	UsageMonth     time.Time `json:"usage_month"`
	CharactersUsed int       `json:"characters_used"`
	RequestsMade   int       `json:"requests_made"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UserUsageStats represents detailed usage statistics for a user
type UserUsageStats struct {
	ID               int       `json:"id"`
	UserID           int       `json:"user_id"`
	APIKeyID         *int      `json:"api_key_id,omitempty"`
	UsageDate        time.Time `json:"usage_date"`
	UsageHour        int       `json:"usage_hour"`
	ServiceName      string    `json:"service_name"`
	Provider         string    `json:"provider"`
	Model            string    `json:"model"`
	UsageType        string    `json:"usage_type"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	RequestsMade     int       `json:"requests_made"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// UserUsageStatsDaily represents daily aggregated usage for a user
type UserUsageStatsDaily struct {
	UsageDate             openapi_types.Date `json:"usage_date"`
	ServiceName           string             `json:"service_name"`
	Provider              string             `json:"provider"`
	Model                 string             `json:"model"`
	UsageType             string             `json:"usage_type"`
	TotalPromptTokens     int                `json:"total_prompt_tokens"`
	TotalCompletionTokens int                `json:"total_completion_tokens"`
	TotalTokens           int                `json:"total_tokens"`
	TotalRequests         int                `json:"total_requests"`
}

// UserUsageStatsHourly represents hourly usage for a user on a specific day
type UserUsageStatsHourly struct {
	UsageHour             int    `json:"usage_hour"`
	ServiceName           string `json:"service_name"`
	Provider              string `json:"provider"`
	Model                 string `json:"model"`
	UsageType             string `json:"usage_type"`
	TotalPromptTokens     int    `json:"total_prompt_tokens"`
	TotalCompletionTokens int    `json:"total_completion_tokens"`
	TotalTokens           int    `json:"total_tokens"`
	TotalRequests         int    `json:"total_requests"`
}

// UsageStatsService handles usage statistics tracking and quota management
type UsageStatsService struct {
	config *config.Config
	db     *sql.DB
	logger *observability.Logger
}

// NewUsageStatsService creates a new usage stats service
func NewUsageStatsService(config *config.Config, db *sql.DB, logger *observability.Logger) *UsageStatsService {
	return &UsageStatsService{
		config: config,
		db:     db,
		logger: logger,
	}
}

// CheckQuota checks if a translation request would exceed the monthly quota
func (s *UsageStatsService) CheckQuota(ctx context.Context, serviceName, usageType string, characters int) (err error) {
	ctx, span := observability.TraceUsageStatsFunction(ctx, "check_quota",
		attribute.String("service_name", serviceName),
		attribute.String("usage_type", usageType),
		attribute.Int("characters", characters),
	)
	defer observability.FinishSpan(span, &err)

	if !s.config.Translation.Quota.Enabled {
		return nil // Quota checking disabled
	}

	currentUsage, err := s.GetCurrentMonthUsage(ctx, serviceName, usageType)
	if err != nil {
		return contextutils.WrapError(err, "failed to get current usage")
	}

	quota := s.GetMonthlyQuota(serviceName)
	newTotal := currentUsage.CharactersUsed + characters

	if newTotal > int(quota) {
		return contextutils.NewAppError(
			contextutils.ErrorCodeQuotaExceeded,
			contextutils.SeverityWarn,
			fmt.Sprintf("Monthly quota exceeded for %s %s service. Used: %d/%d characters",
				serviceName, usageType, newTotal, quota),
			"",
		)
	}

	return nil
}

// RecordUsage records the usage of a translation service
func (s *UsageStatsService) RecordUsage(ctx context.Context, serviceName, usageType string, characters, requests int) (err error) {
	ctx, span := observability.TraceUsageStatsFunction(ctx, "record_usage",
		attribute.String("service_name", serviceName),
		attribute.String("usage_type", usageType),
		attribute.Int("characters", characters),
		attribute.Int("requests", requests),
	)
	defer observability.FinishSpan(span, &err)

	currentMonth := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -time.Now().UTC().Day()+1) // First day of current month

	query := `
		INSERT INTO usage_stats (service_name, usage_type, usage_month, characters_used, requests_made, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (service_name, usage_type, usage_month)
		DO UPDATE SET
			characters_used = usage_stats.characters_used + $4,
			requests_made = usage_stats.requests_made + $5,
			updated_at = NOW()`

	_, err = s.db.ExecContext(ctx, query, serviceName, usageType, currentMonth, characters, requests)
	if err != nil {
		return contextutils.WrapError(err, "failed to record usage")
	}

	return nil
}

// RecordUserAITokenUsage records AI token usage for a specific user
func (s *UsageStatsService) RecordUserAITokenUsage(ctx context.Context, userID int, apiKeyID *int, provider, model, usageType string, promptTokens, completionTokens, totalTokens, requests int) (err error) {
	ctx, span := observability.TraceUsageStatsFunction(ctx, "record_user_ai_token_usage",
		attribute.Int("user_id", userID),
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("usage_type", usageType),
		attribute.Int("prompt_tokens", promptTokens),
		attribute.Int("completion_tokens", completionTokens),
		attribute.Int("total_tokens", totalTokens),
		attribute.Int("requests", requests),
	)
	defer observability.FinishSpan(span, &err)

	now := time.Now()
	usageDate := now.Truncate(24 * time.Hour) // Start of day
	usageHour := now.Hour()

	query := `
		INSERT INTO user_usage_stats (user_id, api_key_id, usage_date, usage_hour, service_name, provider, model, usage_type, prompt_tokens, completion_tokens, total_tokens, requests_made, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (user_id, api_key_id, usage_date, usage_hour, service_name, provider, model, usage_type)
		DO UPDATE SET
			prompt_tokens = user_usage_stats.prompt_tokens + $9,
			completion_tokens = user_usage_stats.completion_tokens + $10,
			total_tokens = user_usage_stats.total_tokens + $11,
			requests_made = user_usage_stats.requests_made + $12,
			updated_at = NOW()`

	_, err = s.db.ExecContext(ctx, query, userID, apiKeyID, usageDate, usageHour, "ai", provider, model, usageType, promptTokens, completionTokens, totalTokens, requests)
	if err != nil {
		return contextutils.WrapError(err, "failed to record user ai token usage")
	}

	return nil
}

// GetCurrentMonthUsage returns the current month's usage for a service and type
func (s *UsageStatsService) GetCurrentMonthUsage(ctx context.Context, serviceName, usageType string) (stats *UsageStats, err error) {
	ctx, span := observability.TraceUsageStatsFunction(ctx, "get_current_month_usage",
		attribute.String("service_name", serviceName),
		attribute.String("usage_type", usageType),
	)
	defer observability.FinishSpan(span, &err)

	currentMonth := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -time.Now().UTC().Day()+1) // First day of current month

	query := `
		SELECT id, service_name, usage_type, usage_month, characters_used, requests_made, created_at, updated_at
		FROM usage_stats
		WHERE service_name = $1 AND usage_type = $2 AND usage_month = $3`

	stats = &UsageStats{}
	err = s.db.QueryRowContext(ctx, query, serviceName, usageType, currentMonth).Scan(
		&stats.ID, &stats.ServiceName, &stats.UsageType, &stats.UsageMonth,
		&stats.CharactersUsed, &stats.RequestsMade, &stats.CreatedAt, &stats.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty stats for new service/month
			return &UsageStats{
				ServiceName:    serviceName,
				UsageType:      usageType,
				UsageMonth:     currentMonth,
				CharactersUsed: 0,
				RequestsMade:   0,
			}, nil
		}
		return nil, contextutils.WrapError(err, "failed to get usage stats")
	}

	return stats, nil
}

// GetMonthlyQuota returns the monthly quota for a service
func (s *UsageStatsService) GetMonthlyQuota(serviceName string) int64 {
	if !s.config.Translation.Quota.Enabled {
		return 0 // No quota limit when disabled
	}

	switch serviceName {
	case "google":
		return s.config.Translation.Quota.GoogleMonthlyQuota
	default:
		return s.config.Translation.Quota.DefaultMonthlyQuota
	}
}

// GetAllUsageStats returns all usage statistics (for admin interface)
func (s *UsageStatsService) GetAllUsageStats(ctx context.Context) (stats []*UsageStats, err error) {
	ctx, span := observability.TraceUsageStatsFunction(ctx, "get_all_usage_stats")
	defer observability.FinishSpan(span, &err)

	query := `
		SELECT id, service_name, usage_type, usage_month, characters_used, requests_made, created_at, updated_at
		FROM usage_stats
		ORDER BY usage_month DESC, service_name, usage_type`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query usage stats")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{})
		}
	}()

	stats = []*UsageStats{}
	for rows.Next() {
		var stat UsageStats
		err := rows.Scan(
			&stat.ID, &stat.ServiceName, &stat.UsageType, &stat.UsageMonth,
			&stat.CharactersUsed, &stat.RequestsMade, &stat.CreatedAt, &stat.UpdatedAt,
		)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to scan usage stats")
		}
		stats = append(stats, &stat)
	}

	if err := rows.Err(); err != nil {
		return nil, contextutils.WrapError(err, "error iterating usage stats")
	}

	return stats, nil
}

// GetUsageStatsByService returns usage statistics for a specific service
func (s *UsageStatsService) GetUsageStatsByService(ctx context.Context, serviceName string) (stats []*UsageStats, err error) {
	ctx, span := observability.TraceUsageStatsFunction(ctx, "get_usage_stats_by_service",
		attribute.String("service_name", serviceName),
	)
	defer observability.FinishSpan(span, &err)

	query := `
		SELECT id, service_name, usage_type, usage_month, characters_used, requests_made, created_at, updated_at
		FROM usage_stats
		WHERE service_name = $1
		ORDER BY usage_month DESC, usage_type`

	rows, err := s.db.QueryContext(ctx, query, serviceName)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query usage stats by service")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{})
		}
	}()

	stats = []*UsageStats{}
	for rows.Next() {
		var stat UsageStats
		err := rows.Scan(
			&stat.ID, &stat.ServiceName, &stat.UsageType, &stat.UsageMonth,
			&stat.CharactersUsed, &stat.RequestsMade, &stat.CreatedAt, &stat.UpdatedAt,
		)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to scan usage stats")
		}
		stats = append(stats, &stat)
	}

	if err := rows.Err(); err != nil {
		return nil, contextutils.WrapError(err, "error iterating usage stats")
	}

	return stats, nil
}

// GetUsageStatsByMonth returns usage statistics for a specific month
func (s *UsageStatsService) GetUsageStatsByMonth(ctx context.Context, year, month int) (stats []*UsageStats, err error) {
	ctx, span := observability.TraceUsageStatsFunction(ctx, "get_usage_stats_by_month",
		attribute.Int("year", year),
		attribute.Int("month", month),
	)
	defer observability.FinishSpan(span, &err)

	// Create date for the first day of the specified month
	targetMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)

	query := `
		SELECT id, service_name, usage_type, usage_month, characters_used, requests_made, created_at, updated_at
		FROM usage_stats
		WHERE usage_month = $1
		ORDER BY service_name, usage_type`

	rows, err := s.db.QueryContext(ctx, query, targetMonth)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query usage stats by month")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{})
		}
	}()

	stats = []*UsageStats{}
	for rows.Next() {
		var stat UsageStats
		err := rows.Scan(
			&stat.ID, &stat.ServiceName, &stat.UsageType, &stat.UsageMonth,
			&stat.CharactersUsed, &stat.RequestsMade, &stat.CreatedAt, &stat.UpdatedAt,
		)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to scan usage stats")
		}
		stats = append(stats, &stat)
	}

	if err := rows.Err(); err != nil {
		return nil, contextutils.WrapError(err, "error iterating usage stats")
	}

	return stats, nil
}

// GetUserAITokenUsageStats returns AI token usage statistics for a specific user
func (s *UsageStatsService) GetUserAITokenUsageStats(ctx context.Context, userID int, startDate, endDate time.Time) (stats []*UserUsageStats, err error) {
	ctx, span := observability.TraceUsageStatsFunction(ctx, "get_user_ai_token_usage_stats",
		attribute.Int("user_id", userID),
		attribute.String("start_date", startDate.Format("2006-01-02")),
		attribute.String("end_date", endDate.Format("2006-01-02")),
	)
	defer observability.FinishSpan(span, &err)

	query := `
		SELECT id, user_id, api_key_id, usage_date, usage_hour, service_name, provider, model, usage_type, prompt_tokens, completion_tokens, total_tokens, requests_made, created_at, updated_at
		FROM user_usage_stats
		WHERE user_id = $1 AND usage_date >= $2 AND usage_date <= $3
		ORDER BY usage_date DESC, usage_hour DESC`

	rows, err := s.db.QueryContext(ctx, query, userID, startDate, endDate)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query user usage stats")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Failed to close user usage stats query", map[string]interface{}{
				"error": closeErr.Error(),
			})
		}
	}()

	stats = []*UserUsageStats{}
	for rows.Next() {
		var stat UserUsageStats
		err = rows.Scan(
			&stat.ID, &stat.UserID, &stat.APIKeyID, &stat.UsageDate, &stat.UsageHour,
			&stat.ServiceName, &stat.Provider, &stat.Model, &stat.UsageType,
			&stat.PromptTokens, &stat.CompletionTokens, &stat.TotalTokens, &stat.RequestsMade,
			&stat.CreatedAt, &stat.UpdatedAt,
		)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to scan user usage stats")
		}
		stats = append(stats, &stat)
	}

	if err = rows.Err(); err != nil {
		return nil, contextutils.WrapError(err, "error iterating user usage stats")
	}

	return stats, nil
}

// GetUserAITokenUsageStatsByDay returns daily aggregated AI token usage for a user
func (s *UsageStatsService) GetUserAITokenUsageStatsByDay(ctx context.Context, userID int, startDate, endDate time.Time) (stats []*UserUsageStatsDaily, err error) {
	ctx, span := observability.TraceUsageStatsFunction(ctx, "get_user_ai_token_usage_stats_by_day",
		attribute.Int("user_id", userID),
		attribute.String("start_date", startDate.Format("2006-01-02")),
		attribute.String("end_date", endDate.Format("2006-01-02")),
	)
	defer observability.FinishSpan(span, &err)

	query := `
		SELECT usage_date, service_name, provider, model, usage_type,
		       SUM(prompt_tokens) as total_prompt_tokens,
		       SUM(completion_tokens) as total_completion_tokens,
		       SUM(total_tokens) as total_tokens,
		       SUM(requests_made) as total_requests
		FROM user_usage_stats
		WHERE user_id = $1 AND usage_date >= $2 AND usage_date <= $3
		GROUP BY usage_date, service_name, provider, model, usage_type
		ORDER BY usage_date DESC, service_name, provider, model, usage_type`

	rows, err := s.db.QueryContext(ctx, query, userID, startDate, endDate)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query user daily usage stats")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Failed to close user daily usage stats query", map[string]interface{}{
				"error": closeErr.Error(),
			})
		}
	}()

	stats = []*UserUsageStatsDaily{}
	for rows.Next() {
		var stat UserUsageStatsDaily
		var usageDate time.Time
		err = rows.Scan(
			&usageDate, &stat.ServiceName, &stat.Provider, &stat.Model, &stat.UsageType,
			&stat.TotalPromptTokens, &stat.TotalCompletionTokens, &stat.TotalTokens, &stat.TotalRequests,
		)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to scan user daily usage stats")
		}
		stat.UsageDate = openapi_types.Date{Time: usageDate}
		stats = append(stats, &stat)
	}

	if err = rows.Err(); err != nil {
		return nil, contextutils.WrapError(err, "error iterating user daily usage stats")
	}

	return stats, nil
}

// GetUserAITokenUsageStatsByHour returns hourly aggregated AI token usage for a user on a specific day
func (s *UsageStatsService) GetUserAITokenUsageStatsByHour(ctx context.Context, userID int, date time.Time) (stats []*UserUsageStatsHourly, err error) {
	ctx, span := observability.TraceUsageStatsFunction(ctx, "get_user_ai_token_usage_stats_by_hour",
		attribute.Int("user_id", userID),
		attribute.String("date", date.Format("2006-01-02")),
	)
	defer observability.FinishSpan(span, &err)

	startOfDay := date.Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)

	query := `
		SELECT usage_hour, service_name, provider, model, usage_type,
		       SUM(prompt_tokens) as total_prompt_tokens,
		       SUM(completion_tokens) as total_completion_tokens,
		       SUM(total_tokens) as total_tokens,
		       SUM(requests_made) as total_requests
		FROM user_usage_stats
		WHERE user_id = $1 AND usage_date >= $2 AND usage_date <= $3
		GROUP BY usage_hour, service_name, provider, model, usage_type
		ORDER BY usage_hour, service_name, provider, model, usage_type`

	rows, err := s.db.QueryContext(ctx, query, userID, startOfDay, endOfDay)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query user hourly usage stats")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Failed to close user hourly usage stats query", map[string]interface{}{
				"error": closeErr.Error(),
			})
		}
	}()

	stats = []*UserUsageStatsHourly{}
	for rows.Next() {
		var stat UserUsageStatsHourly
		err = rows.Scan(
			&stat.UsageHour, &stat.ServiceName, &stat.Provider, &stat.Model, &stat.UsageType,
			&stat.TotalPromptTokens, &stat.TotalCompletionTokens, &stat.TotalTokens, &stat.TotalRequests,
		)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to scan user hourly usage stats")
		}
		stats = append(stats, &stat)
	}

	if err = rows.Err(); err != nil {
		return nil, contextutils.WrapError(err, "error iterating user hourly usage stats")
	}

	return stats, nil
}

// NoopUsageStatsService is a no-operation implementation for testing and when quotas are disabled
type NoopUsageStatsService struct{}

// NewNoopUsageStatsService creates a new noop usage stats service
func NewNoopUsageStatsService() *NoopUsageStatsService {
	return &NoopUsageStatsService{}
}

// CheckQuota always returns nil (no quota checking)
func (s *NoopUsageStatsService) CheckQuota(_ context.Context, _, _ string, _ int) (err error) {
	return nil
}

// RecordUsage always returns nil (no usage recording)
func (s *NoopUsageStatsService) RecordUsage(_ context.Context, _, _ string, _, _ int) (err error) {
	return nil
}

// GetCurrentMonthUsage returns empty stats
func (s *NoopUsageStatsService) GetCurrentMonthUsage(_ context.Context, _, _ string) (stats *UsageStats, err error) {
	return &UsageStats{
		ServiceName:    "",
		UsageType:      "",
		CharactersUsed: 0,
		RequestsMade:   0,
	}, nil
}

// GetMonthlyQuota always returns 0 (no quota limit)
func (s *NoopUsageStatsService) GetMonthlyQuota(_ string) int64 {
	return 0
}

// GetAllUsageStats returns all usage statistics (for admin interface)
func (s *NoopUsageStatsService) GetAllUsageStats(_ context.Context) ([]*UsageStats, error) {
	return []*UsageStats{}, nil
}

// GetUsageStatsByService returns usage statistics for a specific service
func (s *NoopUsageStatsService) GetUsageStatsByService(_ context.Context, _ string) ([]*UsageStats, error) {
	return []*UsageStats{}, nil
}

// GetUsageStatsByMonth returns usage statistics for a specific month
func (s *NoopUsageStatsService) GetUsageStatsByMonth(_ context.Context, _, _ int) ([]*UsageStats, error) {
	return []*UsageStats{}, nil
}

// RecordUserAITokenUsage always returns nil (no usage recording)
func (s *NoopUsageStatsService) RecordUserAITokenUsage(_ context.Context, _ int, _ *int, _, _, _ string, _, _, _, _ int) error {
	return nil
}

// GetUserAITokenUsageStats returns empty stats
func (s *NoopUsageStatsService) GetUserAITokenUsageStats(_ context.Context, _ int, _, _ time.Time) ([]*UserUsageStats, error) {
	return []*UserUsageStats{}, nil
}

// GetUserAITokenUsageStatsByDay returns empty stats
func (s *NoopUsageStatsService) GetUserAITokenUsageStatsByDay(_ context.Context, _ int, _, _ time.Time) ([]*UserUsageStatsDaily, error) {
	return []*UserUsageStatsDaily{}, nil
}

// GetUserAITokenUsageStatsByHour returns empty stats
func (s *NoopUsageStatsService) GetUserAITokenUsageStatsByHour(_ context.Context, _ int, _ time.Time) ([]*UserUsageStatsHourly, error) {
	return []*UserUsageStatsHourly{}, nil
}
