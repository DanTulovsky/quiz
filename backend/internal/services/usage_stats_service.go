package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

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

	currentMonth := time.Now().Truncate(24*time.Hour).AddDate(0, 0, -time.Now().Day()+1) // First day of current month

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

// GetCurrentMonthUsage returns the current month's usage for a service and type
func (s *UsageStatsService) GetCurrentMonthUsage(ctx context.Context, serviceName, usageType string) (stats *UsageStats, err error) {
	ctx, span := observability.TraceUsageStatsFunction(ctx, "get_current_month_usage",
		attribute.String("service_name", serviceName),
		attribute.String("usage_type", usageType),
	)
	defer observability.FinishSpan(span, &err)

	currentMonth := time.Now().Truncate(24*time.Hour).AddDate(0, 0, -time.Now().Day()+1) // First day of current month

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
