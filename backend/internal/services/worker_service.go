package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel/attribute"
)

// ErrSettingNotFound is returned when a setting is not found in the database
var ErrSettingNotFound = errors.New("setting not found")

// WorkerServiceInterface defines the interface for worker management operations
type WorkerServiceInterface interface {
	// Settings management
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	IsGlobalPaused(ctx context.Context) (bool, error)
	SetGlobalPause(ctx context.Context, paused bool) error
	IsUserPaused(ctx context.Context, userID int) (bool, error)
	SetUserPause(ctx context.Context, userID int, paused bool) error

	// Status management
	UpdateWorkerStatus(ctx context.Context, instance string, status *models.WorkerStatus) error
	GetWorkerStatus(ctx context.Context, instance string) (*models.WorkerStatus, error)
	GetAllWorkerStatuses(ctx context.Context) ([]models.WorkerStatus, error)
	UpdateHeartbeat(ctx context.Context, instance string) error
	IsWorkerHealthy(ctx context.Context, instance string) (bool, error)

	// Control operations
	PauseWorker(ctx context.Context, instance string) error
	ResumeWorker(ctx context.Context, instance string) error
	GetWorkerHealth(ctx context.Context) (map[string]interface{}, error)
	GetHighPriorityTopics(ctx context.Context, userID int, language, level, questionType string) ([]string, error)
	GetGapAnalysis(ctx context.Context, userID int, language, level, questionType string) (map[string]int, error)
	GetPriorityDistribution(ctx context.Context, userID int, language, level, questionType string) (map[string]int, error)

	// Notification management
	GetNotificationStats(ctx context.Context) (map[string]interface{}, error)
	GetNotificationErrors(ctx context.Context, page, pageSize int, errorType, notificationType, resolved string) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error)
	GetUpcomingNotifications(ctx context.Context, page, pageSize int, notificationType, status, scheduledAfter, scheduledBefore string) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error)
	GetSentNotifications(ctx context.Context, page, pageSize int, notificationType, status, sentAfter, sentBefore string) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error)

	// Test methods for creating test data
	CreateTestSentNotification(ctx context.Context, userID int, notificationType, subject, templateName, status, errorMessage string) error
}

// WorkerService implements worker management operations
type WorkerService struct {
	db     *sql.DB
	logger *observability.Logger
}

// NewWorkerServiceWithLogger creates a new WorkerService instance with logger
func NewWorkerServiceWithLogger(db *sql.DB, logger *observability.Logger) *WorkerService {
	return &WorkerService{
		db:     db,
		logger: logger,
	}
}

// GetSetting retrieves a setting value by key
func (s *WorkerService) GetSetting(ctx context.Context, key string) (result0 string, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_setting", attribute.String("setting.key", key))
	defer observability.FinishSpan(span, &err)

	// Validate key
	if len(key) == 0 || len(strings.TrimSpace(key)) == 0 {
		return "", contextutils.WrapErrorf(errors.New("invalid setting key"), "setting key cannot be empty")
	}

	var value string
	err = s.db.QueryRowContext(ctx, `
		SELECT setting_value FROM worker_settings WHERE setting_key = $1
	`, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.Debug(ctx, "Setting not found", map[string]interface{}{"setting_key": key})
			return "", contextutils.WrapErrorf(ErrSettingNotFound, "%s", key)
		}
		s.logger.Error(ctx, "Failed to get setting", err, map[string]interface{}{"setting_key": key})
		return "", contextutils.WrapErrorf(err, "failed to get setting %s", key)
	}

	return value, nil
}

// SetSetting updates or creates a setting
func (s *WorkerService) SetSetting(ctx context.Context, key, value string) (err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "set_setting", attribute.String("setting.key", key))
	defer observability.FinishSpan(span, &err)

	// Validate key
	if len(key) == 0 || len(strings.TrimSpace(key)) == 0 {
		return contextutils.WrapErrorf(errors.New("invalid setting key"), "setting key cannot be empty")
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO worker_settings (setting_key, setting_value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (setting_key) DO UPDATE SET
			setting_value = EXCLUDED.setting_value,
			updated_at = EXCLUDED.updated_at
	`, key, value)
	if err != nil {
		s.logger.Error(ctx, "Failed to set setting", err, map[string]interface{}{"setting_key": key, "setting_value": value})
		return contextutils.WrapErrorf(err, "failed to set setting %s", key)
	}

	s.logger.Debug(ctx, "Setting updated", map[string]interface{}{"setting_key": key, "setting_value": value})
	return nil
}

// IsGlobalPaused checks if the worker is globally paused
func (s *WorkerService) IsGlobalPaused(ctx context.Context) (result0 bool, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "is_global_paused")
	defer observability.FinishSpan(span, &err)

	var value string
	value, err = s.GetSetting(ctx, "global_pause")
	if err != nil {
		// If setting doesn't exist, default to false (not paused)
		if errors.Is(err, ErrSettingNotFound) {
			// Initialize the setting with default value
			if setErr := s.SetSetting(ctx, "global_pause", "false"); setErr != nil {
				s.logger.Error(ctx, "Failed to initialize global_pause setting", setErr, map[string]interface{}{})
				return false, contextutils.WrapError(setErr, "failed to initialize global_pause setting")
			}
			return false, nil
		}
		s.logger.Error(ctx, "Failed to check global pause status", err, map[string]interface{}{})
		return false, err
	}

	paused := value == "true"
	s.logger.Debug(ctx, "Global pause status checked", map[string]interface{}{"global_paused": paused})
	return paused, nil
}

// SetGlobalPause sets the global pause state
func (s *WorkerService) SetGlobalPause(ctx context.Context, paused bool) (err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "set_global_pause", attribute.Bool("paused", paused))
	defer observability.FinishSpan(span, &err)

	value := "false"
	if paused {
		value = "true"
	}

	err = s.SetSetting(ctx, "global_pause", value)
	if err != nil {
		return err
	}

	s.logger.Info(ctx, "Global pause state updated", map[string]interface{}{"global_paused": paused})
	return nil
}

// IsUserPaused checks if a specific user is paused
func (s *WorkerService) IsUserPaused(ctx context.Context, userID int) (result0 bool, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "is_user_paused", observability.AttributeUserID(userID))
	defer observability.FinishSpan(span, &err)

	key := fmt.Sprintf("user_pause_%d", userID)
	var value string
	err = s.db.QueryRowContext(ctx, `
		SELECT setting_value FROM worker_settings WHERE setting_key = $1
	`, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			// If setting doesn't exist, user is not paused (this is the default state)
			s.logger.Debug(ctx, "User pause setting not found, defaulting to not paused", map[string]interface{}{"user_id": userID})
			return false, nil
		}
		s.logger.Error(ctx, "Failed to check user pause status", err, map[string]interface{}{"user_id": userID})
		return false, contextutils.WrapErrorf(err, "failed to check user pause status for user %d", userID)
	}

	paused := value == "true"
	s.logger.Debug(ctx, "User pause status checked", map[string]interface{}{"user_id": userID, "user_paused": paused})
	return paused, nil
}

// SetUserPause sets the pause state for a specific user
func (s *WorkerService) SetUserPause(ctx context.Context, userID int, paused bool) (err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "set_user_pause", observability.AttributeUserID(userID), attribute.Bool("paused", paused))
	defer observability.FinishSpan(span, &err)

	key := fmt.Sprintf("user_pause_%d", userID)
	value := "false"
	if paused {
		value = "true"
	}

	err = s.SetSetting(ctx, key, value)
	if err != nil {
		return err
	}

	s.logger.Info(ctx, "User pause state updated", map[string]interface{}{"user_id": userID, "user_paused": paused})
	return nil
}

// UpdateWorkerStatus updates the worker status in the database
func (s *WorkerService) UpdateWorkerStatus(ctx context.Context, instance string, status *models.WorkerStatus) (err error) {
	activity := ""
	if status.CurrentActivity.Valid {
		activity = status.CurrentActivity.String
	}

	ctx, span := observability.TraceWorkerFunction(ctx, "update_worker_status",
		attribute.String("worker.instance", instance),
		attribute.Bool("worker.is_running", status.IsRunning),
		attribute.Bool("worker.is_paused", status.IsPaused),
		attribute.String("worker.activity", activity),
	)
	defer observability.FinishSpan(span, &err)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO worker_status (
			worker_instance, is_running, is_paused, current_activity,
			last_heartbeat, last_run_start, last_run_finish, last_run_error,
			total_questions_generated, total_runs, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (worker_instance) DO UPDATE SET
			is_running = EXCLUDED.is_running,
			is_paused = EXCLUDED.is_paused,
			current_activity = EXCLUDED.current_activity,
			last_heartbeat = EXCLUDED.last_heartbeat,
			last_run_start = EXCLUDED.last_run_start,
			last_run_finish = EXCLUDED.last_run_finish,
			last_run_error = EXCLUDED.last_run_error,
			total_questions_generated = EXCLUDED.total_questions_generated,
			total_runs = EXCLUDED.total_runs,
			updated_at = EXCLUDED.updated_at
	`, instance, status.IsRunning, status.IsPaused, status.CurrentActivity,
		status.LastHeartbeat, status.LastRunStart, status.LastRunFinish,
		status.LastRunError, status.TotalQuestionsGenerated, status.TotalRuns)
	if err != nil {
		s.logger.Error(ctx, "Failed to update worker status", err, map[string]interface{}{
			"worker_instance": instance,
			"is_running":      status.IsRunning,
			"is_paused":       status.IsPaused,
			"activity":        activity,
		})
		err = contextutils.WrapErrorf(err, "failed to update worker status for instance %s", instance)
		return err
	}

	s.logger.Debug(ctx, "Worker status updated", map[string]interface{}{
		"worker_instance": instance,
		"is_running":      status.IsRunning,
		"is_paused":       status.IsPaused,
		"activity":        activity,
	})
	return nil
}

// GetWorkerStatus retrieves worker status by instance
func (s *WorkerService) GetWorkerStatus(ctx context.Context, instance string) (result0 *models.WorkerStatus, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_worker_status", attribute.String("worker.instance", instance))
	defer observability.FinishSpan(span, &err)

	var status models.WorkerStatus
	err = s.db.QueryRowContext(ctx, `
		SELECT id, worker_instance, is_running, is_paused, current_activity,
			   last_heartbeat, last_run_start, last_run_finish, last_run_error,
			   total_questions_generated, total_runs, created_at, updated_at
		FROM worker_status WHERE worker_instance = $1
	`, instance).Scan(
		&status.ID, &status.WorkerInstance, &status.IsRunning, &status.IsPaused,
		&status.CurrentActivity, &status.LastHeartbeat, &status.LastRunStart,
		&status.LastRunFinish, &status.LastRunError, &status.TotalQuestionsGenerated,
		&status.TotalRuns, &status.CreatedAt, &status.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.Debug(ctx, "Worker status not found", map[string]interface{}{"worker_instance": instance})
			return nil, contextutils.WrapErrorf(err, "worker status not found for instance %s", instance)
		}
		s.logger.Error(ctx, "Failed to get worker status", err, map[string]interface{}{"worker_instance": instance})
		return nil, contextutils.WrapErrorf(err, "failed to get worker status for instance %s", instance)
	}

	return &status, nil
}

// GetAllWorkerStatuses retrieves all worker statuses
func (s *WorkerService) GetAllWorkerStatuses(ctx context.Context) (result0 []models.WorkerStatus, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_all_worker_statuses")
	defer observability.FinishSpan(span, &err)

	var rows *sql.Rows
	rows, err = s.db.QueryContext(ctx, `
		SELECT id, worker_instance, is_running, is_paused, current_activity,
			   last_heartbeat, last_run_start, last_run_finish, last_run_error,
			   total_questions_generated, total_runs, created_at, updated_at
		FROM worker_status ORDER BY worker_instance
	`)
	if err != nil {
		s.logger.Error(ctx, "Failed to get all worker statuses", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "failed to get all worker statuses")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error(ctx, "Failed to close rows", err, map[string]interface{}{})
		}
	}()

	var statuses []models.WorkerStatus
	for rows.Next() {
		var status models.WorkerStatus
		err = rows.Scan(
			&status.ID, &status.WorkerInstance, &status.IsRunning, &status.IsPaused,
			&status.CurrentActivity, &status.LastHeartbeat, &status.LastRunStart,
			&status.LastRunFinish, &status.LastRunError, &status.TotalQuestionsGenerated,
			&status.TotalRuns, &status.CreatedAt, &status.UpdatedAt,
		)
		if err != nil {
			s.logger.Error(ctx, "Failed to scan worker status row", err, map[string]interface{}{})
			return nil, contextutils.WrapError(err, "failed to scan worker status row")
		}
		statuses = append(statuses, status)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error(ctx, "Error iterating worker status rows", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "error iterating worker status rows")
	}

	s.logger.Debug(ctx, "Retrieved all worker statuses", map[string]interface{}{"count": len(statuses)})
	return statuses, nil
}

// UpdateHeartbeat updates the heartbeat for a worker instance
func (s *WorkerService) UpdateHeartbeat(ctx context.Context, instance string) (err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "update_heartbeat", attribute.String("worker.instance", instance))
	defer observability.FinishSpan(span, &err)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO worker_status (worker_instance, last_heartbeat, updated_at)
		VALUES ($1, NOW(), NOW())
		ON CONFLICT (worker_instance) DO UPDATE SET
			last_heartbeat = EXCLUDED.last_heartbeat,
			updated_at = EXCLUDED.updated_at
	`, instance)
	if err != nil {
		s.logger.Error(ctx, "Failed to update heartbeat", err, map[string]interface{}{"worker_instance": instance})
		return contextutils.WrapErrorf(err, "failed to update heartbeat for instance %s", instance)
	}

	s.logger.Debug(ctx, "Heartbeat updated", map[string]interface{}{"worker_instance": instance})
	return nil
}

// IsWorkerHealthy checks if a worker instance is healthy based on recent heartbeat
func (s *WorkerService) IsWorkerHealthy(ctx context.Context, instance string) (result0 bool, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "is_worker_healthy", attribute.String("worker.instance", instance))
	defer observability.FinishSpan(span, &err)

	var lastHeartbeat sql.NullTime
	err = s.db.QueryRowContext(ctx, `
		SELECT last_heartbeat FROM worker_status WHERE worker_instance = $1
	`, instance).Scan(&lastHeartbeat)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.Debug(ctx, "Worker not found, considered unhealthy", map[string]interface{}{"worker_instance": instance})
			return false, nil
		}
		s.logger.Error(ctx, "Failed to check worker health", err, map[string]interface{}{"worker_instance": instance})
		return false, contextutils.WrapErrorf(err, "failed to check worker health for instance %s", instance)
	}

	if !lastHeartbeat.Valid {
		s.logger.Debug(ctx, "Worker has no heartbeat, considered unhealthy", map[string]interface{}{"worker_instance": instance})
		return false, nil
	}

	// Consider worker healthy if heartbeat is within the last 5 minutes
	healthy := time.Since(lastHeartbeat.Time) < 5*time.Minute
	s.logger.Debug(ctx, "Worker health checked", map[string]interface{}{
		"worker_instance": instance,
		"healthy":         healthy,
		"last_heartbeat":  lastHeartbeat.Time,
		"time_since":      time.Since(lastHeartbeat.Time).String(),
	})
	return healthy, nil
}

// PauseWorker pauses a specific worker instance
func (s *WorkerService) PauseWorker(ctx context.Context, instance string) (err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "pause_worker", attribute.String("worker.instance", instance))
	defer observability.FinishSpan(span, &err)

	_, err = s.db.ExecContext(ctx, `
		UPDATE worker_status SET is_paused = true, updated_at = NOW()
		WHERE worker_instance = $1
	`, instance)
	if err != nil {
		s.logger.Error(ctx, "Failed to pause worker", err, map[string]interface{}{"worker_instance": instance})
		return contextutils.WrapErrorf(err, "failed to pause worker instance %s", instance)
	}

	s.logger.Info(ctx, "Worker paused", map[string]interface{}{"worker_instance": instance})
	return nil
}

// ResumeWorker resumes a specific worker instance
func (s *WorkerService) ResumeWorker(ctx context.Context, instance string) (err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "resume_worker", attribute.String("worker.instance", instance))
	defer observability.FinishSpan(span, &err)

	_, err = s.db.ExecContext(ctx, `
		UPDATE worker_status SET is_paused = false, updated_at = NOW()
		WHERE worker_instance = $1
	`, instance)
	if err != nil {
		s.logger.Error(ctx, "Failed to resume worker", err, map[string]interface{}{"worker_instance": instance})
		return contextutils.WrapErrorf(err, "failed to resume worker instance %s", instance)
	}

	s.logger.Info(ctx, "Worker resumed", map[string]interface{}{"worker_instance": instance})
	return nil
}

// GetWorkerHealth returns a map of worker health information
func (s *WorkerService) GetWorkerHealth(ctx context.Context) (result0 map[string]interface{}, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_worker_health")
	defer observability.FinishSpan(span, &err)

	var statuses []models.WorkerStatus
	statuses, err = s.GetAllWorkerStatuses(ctx)
	if err != nil {
		return nil, err
	}

	var globalPaused bool
	globalPaused, err = s.IsGlobalPaused(ctx)
	if err != nil {
		s.logger.Error(ctx, "Failed to get global pause state", err, map[string]interface{}{})
		globalPaused = false // Default to false if we can't get the state
	}

	health := make(map[string]interface{})
	workerInstances := make([]map[string]interface{}, 0)
	healthyCount := 0
	totalCount := len(statuses)

	for _, status := range statuses {
		healthy, err := s.IsWorkerHealthy(ctx, status.WorkerInstance)
		if err != nil {
			s.logger.Error(ctx, "Failed to check health for worker", err, map[string]interface{}{"worker_instance": status.WorkerInstance})
			continue
		}

		if healthy {
			healthyCount++
		}

		// Convert sql.NullString to string for last_run_error
		var lastRunError string
		if status.LastRunError.Valid {
			lastRunError = status.LastRunError.String
		}

		workerInstance := map[string]interface{}{
			"worker_instance":           status.WorkerInstance,
			"healthy":                   healthy,
			"is_running":                status.IsRunning,
			"is_paused":                 status.IsPaused,
			"last_heartbeat":            status.LastHeartbeat,
			"last_run_error":            lastRunError,
			"total_questions_generated": status.TotalQuestionsGenerated,
			"total_runs":                status.TotalRuns,
		}
		workerInstances = append(workerInstances, workerInstance)
	}

	// Build comprehensive health summary
	health["global_paused"] = globalPaused
	health["worker_instances"] = workerInstances
	health["total_count"] = totalCount
	health["healthy_count"] = healthyCount

	s.logger.Debug(ctx, "Worker health retrieved", map[string]interface{}{"worker_count": len(health)})
	return health, nil
}

// GetHighPriorityTopics returns topics with high average priority scores for a user
func (s *WorkerService) GetHighPriorityTopics(ctx context.Context, userID int, language, level, questionType string) (result0 []string, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_high_priority_topics",
		observability.AttributeUserID(userID),
		observability.AttributeLanguage(language),
		observability.AttributeLevel(level),
		attribute.String("question.type", questionType),
	)
	defer observability.FinishSpan(span, &err)

	query := `
		SELECT q.topic_category, AVG(qps.priority_score) as avg_score
		FROM questions q
		JOIN user_questions uq ON q.id = uq.question_id
		JOIN question_priority_scores qps ON q.id = qps.question_id AND qps.user_id = $1
		WHERE uq.user_id = $1
		AND q.language = $2
		AND q.level = $3
		AND q.type = $4
		AND q.topic_category IS NOT NULL
		AND q.topic_category != ''
		GROUP BY q.topic_category
		HAVING AVG(qps.priority_score) >= 7.0
		ORDER BY avg_score DESC
		LIMIT 5
	`
	rows, err := s.db.QueryContext(ctx, query, userID, language, level, questionType)
	if err != nil {
		s.logger.Error(ctx, "Failed to get high priority topics", err, map[string]interface{}{
			"user_id": userID, "language": language, "level": level, "question_type": questionType,
		})
		return nil, contextutils.WrapError(err, "failed to get high priority topics")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error(ctx, "Failed to close rows", err, map[string]interface{}{})
		}
	}()
	var topics []string
	for rows.Next() {
		var topic string
		var avgScore float64
		if err := rows.Scan(&topic, &avgScore); err != nil {
			s.logger.Error(ctx, "Failed to scan high priority topics row", err, map[string]interface{}{})
			return nil, contextutils.WrapError(err, "failed to scan high priority topics row")
		}
		topics = append(topics, topic)
	}
	if err := rows.Err(); err != nil {
		s.logger.Error(ctx, "Error iterating high priority topics rows", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "error iterating high priority topics rows")
	}
	s.logger.Debug(ctx, "Retrieved high priority topics", map[string]interface{}{"user_id": userID, "count": len(topics)})
	return topics, nil
}

// GetGapAnalysis identifies areas with poor user performance (knowledge gaps)
func (s *WorkerService) GetGapAnalysis(ctx context.Context, userID int, language, level, questionType string) (result0 map[string]int, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_gap_analysis",
		observability.AttributeUserID(userID),
		observability.AttributeLanguage(language),
		observability.AttributeLevel(level),
		attribute.String("question.type", questionType),
	)
	defer observability.FinishSpan(span, &err)

	// Query to find areas where user has poor performance (low accuracy)
	// This analyzes gaps in user's knowledge across topics and varieties
	query := `
		WITH user_performance AS (
			SELECT
				q.topic_category,
				q.grammar_focus,
				q.vocabulary_domain,
				q.scenario,
				COUNT(*) as total_questions,
				COUNT(CASE WHEN ur.is_correct = true THEN 1 END) as correct_answers,
				ROUND(
					COUNT(CASE WHEN ur.is_correct = true THEN 1 END)::decimal / COUNT(*)::decimal * 100, 2
				) as accuracy_percentage
			FROM questions q
			JOIN user_questions uq ON q.id = uq.question_id
			LEFT JOIN user_responses ur ON q.id = ur.question_id AND ur.user_id = $1
			WHERE uq.user_id = $1
			AND q.language = $2
			AND q.level = $3
			AND q.type = $4
			GROUP BY q.topic_category, q.grammar_focus, q.vocabulary_domain, q.scenario
		)
		SELECT
			COALESCE(topic_category, 'unknown') as area,
			'topic' as gap_type,
			total_questions,
			accuracy_percentage
		FROM user_performance
		WHERE accuracy_percentage < 60 OR accuracy_percentage IS NULL
		UNION ALL
		SELECT
			COALESCE(grammar_focus, 'unknown') as area,
			'grammar' as gap_type,
			total_questions,
			accuracy_percentage
		FROM user_performance
		WHERE (accuracy_percentage < 60 OR accuracy_percentage IS NULL) AND grammar_focus IS NOT NULL
		UNION ALL
		SELECT
			COALESCE(vocabulary_domain, 'unknown') as area,
			'vocabulary' as gap_type,
			total_questions,
			accuracy_percentage
		FROM user_performance
		WHERE (accuracy_percentage < 60 OR accuracy_percentage IS NULL) AND vocabulary_domain IS NOT NULL
		UNION ALL
		SELECT
			COALESCE(scenario, 'unknown') as area,
			'scenario' as gap_type,
			total_questions,
			accuracy_percentage
		FROM user_performance
		WHERE (accuracy_percentage < 60 OR accuracy_percentage IS NULL) AND scenario IS NOT NULL
		ORDER BY accuracy_percentage ASC, total_questions DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, language, level, questionType)
	if err != nil {
		s.logger.Error(ctx, "Failed to get gap analysis", err, map[string]interface{}{
			"user_id": userID, "language": language, "level": level, "question_type": questionType,
		})
		return nil, contextutils.WrapError(err, "failed to get gap analysis")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error(ctx, "Failed to close rows", err, map[string]interface{}{})
		}
	}()

	gaps := make(map[string]int)
	for rows.Next() {
		var area, gapType string
		var totalQuestions int
		var accuracyPercentage sql.NullFloat64

		if err := rows.Scan(&area, &gapType, &totalQuestions, &accuracyPercentage); err != nil {
			s.logger.Error(ctx, "Failed to scan gap analysis row", err, map[string]interface{}{})
			return nil, contextutils.WrapError(err, "failed to scan gap analysis row")
		}

		// Create a key that includes the gap type for better identification
		key := fmt.Sprintf("%s_%s", gapType, area)

		// Use the number of questions as the gap severity indicator
		// Areas with more questions but poor performance are bigger gaps
		gaps[key] = totalQuestions
	}

	if err := rows.Err(); err != nil {
		s.logger.Error(ctx, "Error iterating gap analysis rows", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "error iterating gap analysis rows")
	}
	s.logger.Debug(ctx, "Retrieved gap analysis", map[string]interface{}{"user_id": userID, "count": len(gaps)})
	return gaps, nil
}

// GetPriorityDistribution returns the distribution of priority scores by topic
func (s *WorkerService) GetPriorityDistribution(ctx context.Context, userID int, language, level, questionType string) (result0 map[string]int, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_priority_distribution",
		observability.AttributeUserID(userID),
		observability.AttributeLanguage(language),
		observability.AttributeLevel(level),
		attribute.String("question.type", questionType),
	)
	defer observability.FinishSpan(span, &err)

	// Query to get priority score distribution by topic
	query := `
		SELECT q.topic_category, COUNT(*) as question_count
		FROM questions q
		JOIN user_questions uq ON q.id = uq.question_id
		JOIN question_priority_scores qps ON q.id = qps.question_id AND qps.user_id = $1
		WHERE uq.user_id = $1
		AND q.language = $2
		AND q.level = $3
		AND q.type = $4
		GROUP BY q.topic_category
	`

	rows, err := s.db.QueryContext(ctx, query, userID, language, level, questionType)
	if err != nil {
		s.logger.Error(ctx, "Failed to get priority distribution", err, map[string]interface{}{
			"user_id": userID, "language": language, "level": level, "question_type": questionType,
		})
		return nil, contextutils.WrapError(err, "failed to get priority distribution")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error(ctx, "Failed to close rows", err, map[string]interface{}{})
		}
	}()

	distribution := make(map[string]int)
	for rows.Next() {
		var topic string
		var count int
		if err := rows.Scan(&topic, &count); err != nil {
			s.logger.Error(ctx, "Failed to scan priority distribution row", err, map[string]interface{}{})
			return nil, contextutils.WrapError(err, "failed to scan priority distribution row")
		}
		distribution[topic] = count
	}

	if err := rows.Err(); err != nil {
		s.logger.Error(ctx, "Error iterating priority distribution rows", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "error iterating priority distribution rows")
	}
	s.logger.Debug(ctx, "Retrieved priority distribution", map[string]interface{}{"user_id": userID, "count": len(distribution)})
	return distribution, nil
}

// GetNotificationStats returns comprehensive notification statistics
func (s *WorkerService) GetNotificationStats(ctx context.Context) (result0 map[string]interface{}, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_notification_stats")
	defer observability.FinishSpan(span, &err)

	// Get total notifications sent
	var totalSent int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sent_notifications WHERE status = 'sent'
	`).Scan(&totalSent)
	if err != nil {
		s.logger.Error(ctx, "Failed to get total notifications sent", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "failed to get total notifications sent")
	}

	// Get total notifications failed
	var totalFailed int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sent_notifications WHERE status = 'failed'
	`).Scan(&totalFailed)
	if err != nil {
		s.logger.Error(ctx, "Failed to get total notifications failed", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "failed to get total notifications failed")
	}

	// Calculate success rate
	var successRate float64
	if totalSent+totalFailed > 0 {
		successRate = float64(totalSent) / float64(totalSent+totalFailed)
	}

	// Get users with notifications enabled
	var usersWithNotifications int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT user_id) FROM user_learning_preferences WHERE daily_reminder_enabled = true
	`).Scan(&usersWithNotifications)
	if err != nil {
		s.logger.Error(ctx, "Failed to get users with notifications enabled", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "failed to get users with notifications enabled")
	}

	// Get total users
	var totalUsers int
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	if err != nil {
		s.logger.Error(ctx, "Failed to get total users", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "failed to get total users")
	}

	// Get notifications sent today
	var sentToday int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sent_notifications
		WHERE status = 'sent' AND DATE(sent_at) = CURRENT_DATE
	`).Scan(&sentToday)
	if err != nil {
		s.logger.Error(ctx, "Failed to get notifications sent today", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "failed to get notifications sent today")
	}

	// Get notifications sent this week
	var sentThisWeek int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sent_notifications
		WHERE status = 'sent' AND sent_at >= DATE_TRUNC('week', CURRENT_DATE)
	`).Scan(&sentThisWeek)
	if err != nil {
		s.logger.Error(ctx, "Failed to get notifications sent this week", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "failed to get notifications sent this week")
	}

	// Get upcoming notifications
	var upcomingNotifications int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM upcoming_notifications WHERE status = 'pending'
	`).Scan(&upcomingNotifications)
	if err != nil {
		s.logger.Error(ctx, "Failed to get upcoming notifications", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "failed to get upcoming notifications")
	}

	// Get unresolved errors
	var unresolvedErrors int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notification_errors WHERE resolved_at IS NULL
	`).Scan(&unresolvedErrors)
	if err != nil {
		s.logger.Error(ctx, "Failed to get unresolved errors", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "failed to get unresolved errors")
	}

	// Get notifications by type
	notificationsByType := make(map[string]int)
	rows, err := s.db.QueryContext(ctx, `
		SELECT notification_type, COUNT(*)
		FROM sent_notifications
		WHERE status = 'sent'
		GROUP BY notification_type
	`)
	if err != nil {
		s.logger.Error(ctx, "Failed to get notifications by type", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "failed to get notifications by type")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{})
		}
	}()

	for rows.Next() {
		var notificationType string
		var count int
		if err := rows.Scan(&notificationType, &count); err != nil {
			s.logger.Error(ctx, "Failed to scan notifications by type", err, map[string]interface{}{})
			return nil, contextutils.WrapError(err, "failed to scan notifications by type")
		}
		notificationsByType[notificationType] = count
	}

	// Get errors by type
	errorsByType := make(map[string]int)
	rows, err = s.db.QueryContext(ctx, `
		SELECT error_type, COUNT(*)
		FROM notification_errors
		GROUP BY error_type
	`)
	if err != nil {
		s.logger.Error(ctx, "Failed to get errors by type", err, map[string]interface{}{})
		return nil, contextutils.WrapError(err, "failed to get errors by type")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{})
		}
	}()

	for rows.Next() {
		var errorType string
		var count int
		if err := rows.Scan(&errorType, &count); err != nil {
			s.logger.Error(ctx, "Failed to scan errors by type", err, map[string]interface{}{})
			return nil, contextutils.WrapError(err, "failed to scan errors by type")
		}
		errorsByType[errorType] = count
	}

	stats := map[string]interface{}{
		"total_notifications_sent":         totalSent,
		"total_notifications_failed":       totalFailed,
		"success_rate":                     successRate,
		"users_with_notifications_enabled": usersWithNotifications,
		"total_users":                      totalUsers,
		"notifications_sent_today":         sentToday,
		"notifications_sent_this_week":     sentThisWeek,
		"notifications_by_type":            notificationsByType,
		"errors_by_type":                   errorsByType,
		"upcoming_notifications":           upcomingNotifications,
		"unresolved_errors":                unresolvedErrors,
	}

	s.logger.Debug(ctx, "Retrieved notification stats", map[string]interface{}{"stats": stats})
	return stats, nil
}

// GetNotificationErrors returns paginated notification errors with filtering
func (s *WorkerService) GetNotificationErrors(ctx context.Context, page, pageSize int, errorType, notificationType, resolved string) (result0 []map[string]interface{}, result1, result2 map[string]interface{}, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_notification_errors",
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
		attribute.String("error_type", errorType),
		attribute.String("notification_type", notificationType),
		attribute.String("resolved", resolved),
	)
	defer observability.FinishSpan(span, &err)

	// Build WHERE clause
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if errorType != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("error_type = $%d", argIndex))
		args = append(args, errorType)
		argIndex++
	}

	if notificationType != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("notification_type = $%d", argIndex))
		args = append(args, notificationType)
		argIndex++
	}

	switch resolved {
	case "true":
		whereConditions = append(whereConditions, "resolved_at IS NOT NULL")
	case "false":
		whereConditions = append(whereConditions, "resolved_at IS NULL")
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Get total count
	var totalErrors int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notification_errors %s", whereClause)
	err = s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalErrors)
	if err != nil {
		s.logger.Error(ctx, "Failed to get total notification errors", err, map[string]interface{}{})
		return nil, nil, nil, contextutils.WrapError(err, "failed to get total notification errors")
	}

	// Calculate pagination
	offset := (page - 1) * pageSize
	totalPages := (totalErrors + pageSize - 1) / pageSize

	// Get errors with pagination
	args = append(args, pageSize, offset)
	query := fmt.Sprintf(`
		SELECT ne.id, ne.user_id, u.username, ne.notification_type, ne.error_type,
		       ne.error_message, ne.email_address, ne.occurred_at, ne.resolved_at, ne.resolution_notes
		FROM notification_errors ne
		LEFT JOIN users u ON ne.user_id = u.id
		%s
		ORDER BY ne.occurred_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		s.logger.Error(ctx, "Failed to get notification errors", err, map[string]interface{}{})
		return nil, nil, nil, contextutils.WrapError(err, "failed to get notification errors")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{})
		}
	}()

	var errors []map[string]interface{}
	for rows.Next() {
		var errorData map[string]interface{}
		var id int
		var userID sql.NullInt64
		var username sql.NullString
		var notificationType, errorType, errorMessage string
		var emailAddress sql.NullString
		var occurredAt time.Time
		var resolvedAt sql.NullTime
		var resolutionNotes sql.NullString

		err := rows.Scan(&id, &userID, &username, &notificationType, &errorType, &errorMessage, &emailAddress, &occurredAt, &resolvedAt, &resolutionNotes)
		if err != nil {
			s.logger.Error(ctx, "Failed to scan notification error", err, map[string]interface{}{})
			return nil, nil, nil, contextutils.WrapError(err, "failed to scan notification error")
		}

		errorData = map[string]interface{}{
			"id":                id,
			"notification_type": notificationType,
			"error_type":        errorType,
			"error_message":     errorMessage,
			"occurred_at":       occurredAt.Format(time.RFC3339),
		}

		if userID.Valid {
			errorData["user_id"] = userID.Int64
		}
		if username.Valid {
			errorData["username"] = username.String
		}
		if emailAddress.Valid {
			errorData["email_address"] = emailAddress.String
		}
		if resolvedAt.Valid {
			errorData["resolved_at"] = resolvedAt.Time.Format(time.RFC3339)
		}
		if resolutionNotes.Valid {
			errorData["resolution_notes"] = resolutionNotes.String
		}

		errors = append(errors, errorData)
	}

	// Get stats
	stats := map[string]interface{}{
		"total_errors":      totalErrors,
		"unresolved_errors": 0, // Will be calculated separately
	}

	// Get unresolved errors count
	var unresolvedCount int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM notification_errors WHERE resolved_at IS NULL").Scan(&unresolvedCount)
	if err != nil {
		s.logger.Error(ctx, "Failed to get unresolved errors count", err, map[string]interface{}{})
	} else {
		stats["unresolved_errors"] = unresolvedCount
	}

	// Get errors by type
	errorsByType := make(map[string]int)
	rows, err = s.db.QueryContext(ctx, "SELECT error_type, COUNT(*) FROM notification_errors GROUP BY error_type")
	if err != nil {
		s.logger.Error(ctx, "Failed to get errors by type", err, map[string]interface{}{})
	} else {
		defer func() {
			if closeErr := rows.Close(); closeErr != nil {
				s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{})
			}
		}()
		for rows.Next() {
			var errorType string
			var count int
			if err := rows.Scan(&errorType, &count); err != nil {
				s.logger.Error(ctx, "Failed to scan errors by type", err, map[string]interface{}{})
				continue
			}
			errorsByType[errorType] = count
		}
		stats["errors_by_type"] = errorsByType
	}

	// Get errors by notification type
	errorsByNotificationType := make(map[string]int)
	rows, err = s.db.QueryContext(ctx, "SELECT notification_type, COUNT(*) FROM notification_errors GROUP BY notification_type")
	if err != nil {
		s.logger.Error(ctx, "Failed to get errors by notification type", err, map[string]interface{}{})
	} else {
		defer func() {
			if closeErr := rows.Close(); closeErr != nil {
				s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{})
			}
		}()
		for rows.Next() {
			var notificationType string
			var count int
			if err := rows.Scan(&notificationType, &count); err != nil {
				s.logger.Error(ctx, "Failed to scan errors by notification type", err, map[string]interface{}{})
				continue
			}
			errorsByNotificationType[notificationType] = count
		}
		stats["errors_by_notification_type"] = errorsByNotificationType
	}

	pagination := map[string]interface{}{
		"page":        page,
		"page_size":   pageSize,
		"total":       totalErrors,
		"total_pages": totalPages,
	}

	s.logger.Debug(ctx, "Retrieved notification errors", map[string]interface{}{
		"count": len(errors), "page": page, "total": totalErrors,
	})

	return errors, pagination, stats, nil
}

// GetUpcomingNotifications returns paginated upcoming notifications with filtering
func (s *WorkerService) GetUpcomingNotifications(ctx context.Context, page, pageSize int, notificationType, status, scheduledAfter, scheduledBefore string) (result0 []map[string]interface{}, result1, result2 map[string]interface{}, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_upcoming_notifications",
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
		attribute.String("notification_type", notificationType),
		attribute.String("status", status),
		attribute.String("scheduled_after", scheduledAfter),
		attribute.String("scheduled_before", scheduledBefore),
	)
	defer observability.FinishSpan(span, &err)

	// Build WHERE clause
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if notificationType != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("notification_type = $%d", argIndex))
		args = append(args, notificationType)
		argIndex++
	}

	if status != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	if scheduledAfter != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("scheduled_for >= $%d", argIndex))
		args = append(args, scheduledAfter)
		argIndex++
	}

	if scheduledBefore != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("scheduled_for <= $%d", argIndex))
		args = append(args, scheduledBefore)
		argIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Get total count
	var totalNotifications int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM upcoming_notifications %s", whereClause)
	err = s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalNotifications)
	if err != nil {
		s.logger.Error(ctx, "Failed to get total upcoming notifications", err, map[string]interface{}{})
		return nil, nil, nil, contextutils.WrapError(err, "failed to get total upcoming notifications")
	}

	// Calculate pagination
	offset := (page - 1) * pageSize
	totalPages := (totalNotifications + pageSize - 1) / pageSize

	// Get notifications with pagination
	args = append(args, pageSize, offset)
	query := fmt.Sprintf(`
		SELECT un.id, un.user_id, u.username, u.email, un.notification_type,
		       un.scheduled_for, un.status, un.created_at
		FROM upcoming_notifications un
		LEFT JOIN users u ON un.user_id = u.id
		%s
		ORDER BY un.scheduled_for ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		s.logger.Error(ctx, "Failed to get upcoming notifications", err, map[string]interface{}{})
		return nil, nil, nil, contextutils.WrapError(err, "failed to get upcoming notifications")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{})
		}
	}()

	var notifications []map[string]interface{}
	for rows.Next() {
		var notification map[string]interface{}
		var id, userID int
		var username, notificationType, status string
		var scheduledFor, createdAt time.Time
		var email sql.NullString

		err := rows.Scan(&id, &userID, &username, &email, &notificationType, &scheduledFor, &status, &createdAt)
		if err != nil {
			s.logger.Error(ctx, "Failed to scan upcoming notification", err, map[string]interface{}{})
			return nil, nil, nil, contextutils.WrapError(err, "failed to scan upcoming notification")
		}

		notification = map[string]interface{}{
			"id":                id,
			"user_id":           userID,
			"username":          username,
			"notification_type": notificationType,
			"scheduled_for":     scheduledFor.Format(time.RFC3339),
			"status":            status,
			"created_at":        createdAt.Format(time.RFC3339),
		}

		if email.Valid {
			notification["email_address"] = email.String
		} else {
			notification["email_address"] = ""
		}

		notifications = append(notifications, notification)
	}

	// Get stats
	stats := map[string]interface{}{
		"total_pending":             0,
		"total_scheduled_today":     0,
		"total_scheduled_this_week": 0,
	}

	// Get total pending
	var totalPending int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM upcoming_notifications WHERE status = 'pending'").Scan(&totalPending)
	if err != nil {
		s.logger.Error(ctx, "Failed to get total pending", err, map[string]interface{}{})
	} else {
		stats["total_pending"] = totalPending
	}

	// Get scheduled today
	var scheduledToday int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM upcoming_notifications
		WHERE status = 'pending' AND DATE(scheduled_for) = CURRENT_DATE
	`).Scan(&scheduledToday)
	if err != nil {
		s.logger.Error(ctx, "Failed to get scheduled today", err, map[string]interface{}{})
	} else {
		stats["total_scheduled_today"] = scheduledToday
	}

	// Get scheduled this week
	var scheduledThisWeek int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM upcoming_notifications
		WHERE status = 'pending' AND scheduled_for >= DATE_TRUNC('week', CURRENT_DATE)
	`).Scan(&scheduledThisWeek)
	if err != nil {
		s.logger.Error(ctx, "Failed to get scheduled this week", err, map[string]interface{}{})
	} else {
		stats["total_scheduled_this_week"] = scheduledThisWeek
	}

	// Get notifications by type
	notificationsByType := make(map[string]int)
	rows, err = s.db.QueryContext(ctx, "SELECT notification_type, COUNT(*) FROM upcoming_notifications GROUP BY notification_type")
	if err != nil {
		s.logger.Error(ctx, "Failed to get notifications by type", err, map[string]interface{}{})
	} else {
		defer func() {
			if closeErr := rows.Close(); closeErr != nil {
				s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{})
			}
		}()
		for rows.Next() {
			var notificationType string
			var count int
			if err := rows.Scan(&notificationType, &count); err != nil {
				s.logger.Error(ctx, "Failed to scan notifications by type", err, map[string]interface{}{})
				continue
			}
			notificationsByType[notificationType] = count
		}
		stats["notifications_by_type"] = notificationsByType
	}

	pagination := map[string]interface{}{
		"page":        page,
		"page_size":   pageSize,
		"total":       totalNotifications,
		"total_pages": totalPages,
	}

	s.logger.Debug(ctx, "Retrieved upcoming notifications", map[string]interface{}{
		"count": len(notifications), "page": page, "total": totalNotifications,
	})

	return notifications, pagination, stats, nil
}

// GetSentNotifications returns paginated sent notifications with filtering
func (s *WorkerService) GetSentNotifications(ctx context.Context, page, pageSize int, notificationType, status, sentAfter, sentBefore string) (result0 []map[string]interface{}, result1, result2 map[string]interface{}, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_sent_notifications",
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
		attribute.String("notification_type", notificationType),
		attribute.String("status", status),
		attribute.String("sent_after", sentAfter),
		attribute.String("sent_before", sentBefore),
	)
	defer observability.FinishSpan(span, &err)

	// Build WHERE clause
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if notificationType != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("notification_type = $%d", argIndex))
		args = append(args, notificationType)
		argIndex++
	}

	if status != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	if sentAfter != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("sent_at >= $%d", argIndex))
		args = append(args, sentAfter)
		argIndex++
	}

	if sentBefore != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("sent_at <= $%d", argIndex))
		args = append(args, sentBefore)
		argIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Get total count
	var totalNotifications int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM sent_notifications %s", whereClause)
	err = s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalNotifications)
	if err != nil {
		s.logger.Error(ctx, "Failed to get total sent notifications", err, map[string]interface{}{})
		return nil, nil, nil, contextutils.WrapError(err, "failed to get total sent notifications")
	}

	// Calculate pagination
	offset := (page - 1) * pageSize
	totalPages := (totalNotifications + pageSize - 1) / pageSize

	// Get notifications with pagination
	args = append(args, pageSize, offset)
	query := fmt.Sprintf(`
		SELECT sn.id, sn.user_id, u.username, u.email, sn.notification_type,
		       sn.subject, sn.template_name, sn.sent_at, sn.status, sn.error_message, sn.retry_count
		FROM sent_notifications sn
		LEFT JOIN users u ON sn.user_id = u.id
		%s
		ORDER BY sn.sent_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		s.logger.Error(ctx, "Failed to get sent notifications", err, map[string]interface{}{})
		return nil, nil, nil, contextutils.WrapError(err, "failed to get sent notifications")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{})
		}
	}()

	var notifications []map[string]interface{}
	for rows.Next() {
		var notification map[string]interface{}
		var id, userID int
		var username, notificationType, subject, templateName, status string
		var sentAt time.Time
		var errorMessage sql.NullString
		var retryCount int
		var email sql.NullString

		err := rows.Scan(&id, &userID, &username, &email, &notificationType, &subject, &templateName, &sentAt, &status, &errorMessage, &retryCount)
		if err != nil {
			s.logger.Error(ctx, "Failed to scan sent notification", err, map[string]interface{}{})
			return nil, nil, nil, contextutils.WrapError(err, "failed to scan sent notification")
		}

		notification = map[string]interface{}{
			"id":                id,
			"user_id":           userID,
			"username":          username,
			"notification_type": notificationType,
			"subject":           subject,
			"template_name":     templateName,
			"sent_at":           sentAt.Format(time.RFC3339),
			"status":            status,
			"retry_count":       retryCount,
		}

		if email.Valid {
			notification["email_address"] = email.String
		} else {
			notification["email_address"] = ""
		}

		if errorMessage.Valid {
			notification["error_message"] = errorMessage.String
		}

		notifications = append(notifications, notification)
	}

	// Get stats
	stats := map[string]interface{}{
		"total_sent":     0,
		"total_failed":   0,
		"success_rate":   0.0,
		"sent_today":     0,
		"sent_this_week": 0,
	}

	// Get total sent
	var totalSent int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sent_notifications WHERE status = 'sent'").Scan(&totalSent)
	if err != nil {
		s.logger.Error(ctx, "Failed to get total sent", err, map[string]interface{}{})
	} else {
		stats["total_sent"] = totalSent
	}

	// Get total failed
	var totalFailed int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sent_notifications WHERE status = 'failed'").Scan(&totalFailed)
	if err != nil {
		s.logger.Error(ctx, "Failed to get total failed", err, map[string]interface{}{})
	} else {
		stats["total_failed"] = totalFailed
	}

	// Calculate success rate
	if totalSent+totalFailed > 0 {
		stats["success_rate"] = float64(totalSent) / float64(totalSent+totalFailed)
	}

	// Get sent today
	var sentToday int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sent_notifications
		WHERE status = 'sent' AND DATE(sent_at) = CURRENT_DATE
	`).Scan(&sentToday)
	if err != nil {
		s.logger.Error(ctx, "Failed to get sent today", err, map[string]interface{}{})
	} else {
		stats["sent_today"] = sentToday
	}

	// Get sent this week
	var sentThisWeek int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sent_notifications
		WHERE status = 'sent' AND sent_at >= DATE_TRUNC('week', CURRENT_DATE)
	`).Scan(&sentThisWeek)
	if err != nil {
		s.logger.Error(ctx, "Failed to get sent this week", err, map[string]interface{}{})
	} else {
		stats["sent_this_week"] = sentThisWeek
	}

	// Get notifications by type
	notificationsByType := make(map[string]int)
	rows, err = s.db.QueryContext(ctx, "SELECT notification_type, COUNT(*) FROM sent_notifications GROUP BY notification_type")
	if err != nil {
		s.logger.Error(ctx, "Failed to get notifications by type", err, map[string]interface{}{})
	} else {
		defer func() {
			if closeErr := rows.Close(); closeErr != nil {
				s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{})
			}
		}()
		for rows.Next() {
			var notificationType string
			var count int
			if err := rows.Scan(&notificationType, &count); err != nil {
				s.logger.Error(ctx, "Failed to scan notifications by type", err, map[string]interface{}{})
				continue
			}
			notificationsByType[notificationType] = count
		}
		stats["notifications_by_type"] = notificationsByType
	}

	pagination := map[string]interface{}{
		"page":        page,
		"page_size":   pageSize,
		"total":       totalNotifications,
		"total_pages": totalPages,
	}

	s.logger.Debug(ctx, "Retrieved sent notifications", map[string]interface{}{
		"count": len(notifications), "page": page, "total": totalNotifications,
	})

	return notifications, pagination, stats, nil
}

// CreateTestSentNotification creates a test sent notification for testing purposes
func (s *WorkerService) CreateTestSentNotification(ctx context.Context, userID int, notificationType, subject, templateName, status, errorMessage string) error {
	ctx, span := observability.TraceWorkerFunction(ctx, "create_test_sent_notification",
		attribute.Int("user.id", userID),
		attribute.String("notification.type", notificationType),
		attribute.String("notification.status", status),
	)
	defer span.End()

	query := `
		INSERT INTO sent_notifications (user_id, notification_type, subject, template_name, sent_at, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.ExecContext(ctx, query, userID, notificationType, subject, templateName, time.Now(), status, errorMessage)
	if err != nil {
		span.RecordError(err)
		s.logger.Error(ctx, "Failed to create test sent notification", err, map[string]interface{}{
			"user_id":           userID,
			"notification_type": notificationType,
			"status":            status,
		})
		return contextutils.WrapError(err, "failed to create test sent notification")
	}

	s.logger.Info(ctx, "Created test sent notification", map[string]interface{}{
		"user_id":           userID,
		"notification_type": notificationType,
		"status":            status,
	})

	return nil
}
