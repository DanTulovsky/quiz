package services

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// LearningServiceInterface defines the interface for the learning service
type LearningServiceInterface interface {
	RecordUserResponse(ctx context.Context, response *models.UserResponse) error
	GetUserProgress(ctx context.Context, userID int) (*models.UserProgress, error)
	GetWeakestTopics(ctx context.Context, userID, limit int) ([]*models.PerformanceMetrics, error)
	ShouldAvoidQuestion(ctx context.Context, userID, questionID int) (bool, error)
	GetUserQuestionStats(ctx context.Context, userID int) (*UserQuestionStats, error)
	// Priority system methods
	RecordAnswerWithPriority(ctx context.Context, userID, questionID, answerIndex int, isCorrect bool, responseTime int) error
	// RecordAnswerWithPriorityReturningID records the response and returns the created user_responses.id
	RecordAnswerWithPriorityReturningID(ctx context.Context, userID, questionID, answerIndex int, isCorrect bool, responseTime int) (int, error)
	MarkQuestionAsKnown(ctx context.Context, userID, questionID int, confidenceLevel *int) error
	GetUserLearningPreferences(ctx context.Context, userID int) (*models.UserLearningPreferences, error)
	UpdateLastDailyReminderSent(ctx context.Context, userID int) error
	CalculatePriorityScore(ctx context.Context, userID, questionID int) (float64, error)
	UpdateUserLearningPreferences(ctx context.Context, userID int, prefs *models.UserLearningPreferences) (*models.UserLearningPreferences, error)
	GetUserQuestionConfidenceLevel(ctx context.Context, userID, questionID int) (*int, error)
	// Analytics methods
	GetPriorityScoreDistribution(ctx context.Context) (map[string]interface{}, error)
	GetHighPriorityQuestions(ctx context.Context, limit int) ([]map[string]interface{}, error)
	GetWeakAreasByTopic(ctx context.Context, limit int) ([]map[string]interface{}, error)
	GetLearningPreferencesUsage(ctx context.Context) (map[string]interface{}, error)
	GetQuestionTypeGaps(ctx context.Context) ([]map[string]interface{}, error)
	GetGenerationSuggestions(ctx context.Context) ([]map[string]interface{}, error)
	GetPrioritySystemPerformance(ctx context.Context) (map[string]interface{}, error)
	GetBackgroundJobsStatus(ctx context.Context) (map[string]interface{}, error)
	// User-specific analytics methods
	GetUserPriorityScoreDistribution(ctx context.Context, userID int) (map[string]interface{}, error)
	GetUserHighPriorityQuestions(ctx context.Context, userID, limit int) ([]map[string]interface{}, error)
	GetUserWeakAreas(ctx context.Context, userID, limit int) ([]map[string]interface{}, error)
	// Additional analytics methods for progress API
	GetHighPriorityTopics(ctx context.Context, userID int) ([]string, error)
	GetGapAnalysis(ctx context.Context, userID int) (map[string]interface{}, error)
	GetPriorityDistribution(ctx context.Context, userID int) (map[string]int, error)
}

// UserQuestionStats represents per-user question statistics
type UserQuestionStats struct {
	UserID           int                `json:"user_id"`
	TotalAnswered    int                `json:"total_answered"`
	CorrectAnswers   int                `json:"correct_answers"`
	IncorrectAnswers int                `json:"incorrect_answers"`
	AccuracyRate     float64            `json:"accuracy_rate"`
	AnsweredByType   map[string]int     `json:"answered_by_type"`
	AnsweredByLevel  map[string]int     `json:"answered_by_level"`
	AccuracyByType   map[string]float64 `json:"accuracy_by_type"`
	AccuracyByLevel  map[string]float64 `json:"accuracy_by_level"`
	AvailableByType  map[string]int     `json:"available_by_type"`
	AvailableByLevel map[string]int     `json:"available_by_level"`
	RecentlyAnswered int                `json:"recently_answered"` // Within last hour
}

// contextutils.ErrQuestionNotFound is returned when a question does not exist in the database
// contextutils.ErrQuestionNotFound is now imported from contextutils

// LearningService provides methods for managing user learning progress
type LearningService struct {
	db     *sql.DB
	cfg    *config.Config
	logger *observability.Logger
}

// NewLearningServiceWithLogger creates a new LearningService with a logger
func NewLearningServiceWithLogger(db *sql.DB, cfg *config.Config, logger *observability.Logger) *LearningService {
	return &LearningService{
		db:     db,
		cfg:    cfg,
		logger: logger,
	}
}

// RecordUserResponse records a user's response to a question and updates metrics
func (s *LearningService) RecordUserResponse(ctx context.Context, response *models.UserResponse) (err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "record_user_response",
		observability.AttributeUserID(response.UserID),
		observability.AttributeQuestionID(response.QuestionID),
		attribute.Bool("response.is_correct", response.IsCorrect),
		attribute.Int("response.time_ms", response.ResponseTimeMs),
	)
	defer observability.FinishSpan(span, &err)

	query := `
		INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, response_time_ms)
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`

	var id int
	err = s.db.QueryRowContext(ctx, query,
		response.UserID,
		response.QuestionID,
		response.UserAnswerIndex,
		response.IsCorrect,
		response.ResponseTimeMs,
	).Scan(&id)
	if err != nil {
		return err
	}

	response.ID = id

	// Update performance metrics
	return s.updatePerformanceMetrics(ctx, response)
}

func (s *LearningService) updatePerformanceMetrics(ctx context.Context, response *models.UserResponse) (err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "update_performance_metrics",
		observability.AttributeUserID(response.UserID),
		observability.AttributeQuestionID(response.QuestionID),
		attribute.Bool("response.is_correct", response.IsCorrect),
	)
	defer observability.FinishSpan(span, &err)

	// Get question details
	var question *models.Question
	question, err = s.getQuestionDetails(ctx, response.QuestionID)
	if err != nil {
		return err
	}

	// Update or create performance metrics
	query := `
		INSERT INTO performance_metrics (
			user_id, topic, language, level, total_attempts, correct_attempts,
			average_response_time_ms, difficulty_adjustment, last_updated
		)
		VALUES ($1, $2, $3, $4, 1, $5, $6, 0.0, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, topic, language, level) DO UPDATE SET
			total_attempts = performance_metrics.total_attempts + 1,
			correct_attempts = performance_metrics.correct_attempts + $7,
			average_response_time_ms = (performance_metrics.average_response_time_ms * (performance_metrics.total_attempts - 1) + $8) / performance_metrics.total_attempts,
			last_updated = CURRENT_TIMESTAMP
	`

	correctIncrement := 0
	if response.IsCorrect {
		correctIncrement = 1
	}

	_, err = s.db.ExecContext(ctx, query,
		response.UserID,
		question.TopicCategory,
		question.Language,
		question.Level,
		correctIncrement,                 // For initial correct_attempts in VALUES
		float64(response.ResponseTimeMs), // For initial average_response_time_ms in VALUES
		correctIncrement,                 // For correct_attempts increment in UPDATE
		response.ResponseTimeMs,          // For average_response_time_ms calculation in UPDATE
	)

	return err
}

// getUserByID is a lightweight helper for LearningService to fetch a user row.
func (s *LearningService) getUserByID(ctx context.Context, userID int) (*models.User, error) {
	query := `
        SELECT id, username, email, timezone, password_hash, last_active,
               preferred_language, current_level, ai_provider, ai_model,
               ai_enabled, ai_api_key, created_at, updated_at
        FROM users
        WHERE id = $1
    `

	var u models.User
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&u.ID, &u.Username, &u.Email, &u.Timezone, &u.PasswordHash, &u.LastActive,
		&u.PreferredLanguage, &u.CurrentLevel, &u.AIProvider, &u.AIModel,
		&u.AIEnabled, &u.AIAPIKey, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (s *LearningService) getQuestionDetails(ctx context.Context, questionID int) (result0 *models.Question, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_question_details",
		observability.AttributeQuestionID(questionID),
	)
	defer observability.FinishSpan(span, &err)

	query := `SELECT type, language, level, topic_category FROM questions WHERE id = $1`

	question := &models.Question{}
	var topicCategory sql.NullString
	err = s.db.QueryRowContext(ctx, query, questionID).Scan(
		&question.Type,
		&question.Language,
		&question.Level,
		&topicCategory,
	)

	if topicCategory.Valid {
		question.TopicCategory = topicCategory.String
	}

	return question, err
}

// GetUserProgress retrieves comprehensive learning progress for a user
func (s *LearningService) GetUserProgress(ctx context.Context, userID int) (result0 *models.UserProgress, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_user_progress",
		attribute.String("user.username", ""),
		attribute.String("language", ""),
		attribute.String("level", ""),
	)
	defer observability.FinishSpan(span, &err)

	progress := &models.UserProgress{
		PerformanceByTopic: make(map[string]*models.PerformanceMetrics),
	}

	// Get overall stats
	overallQuery := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN is_correct THEN 1 ELSE 0 END), 0) as correct
		FROM user_responses
		WHERE user_id = $1
	`

	err = s.db.QueryRowContext(ctx, overallQuery, userID).Scan(
		&progress.TotalQuestions,
		&progress.CorrectAnswers,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if progress.TotalQuestions > 0 {
		progress.AccuracyRate = float64(progress.CorrectAnswers) / float64(progress.TotalQuestions) * 100
	}

	// Get performance by topic
	metricsQuery := `
		SELECT id, topic, language, level, total_attempts, correct_attempts,
			   average_response_time_ms, difficulty_adjustment, last_updated
		FROM performance_metrics
		WHERE user_id = $1
	`

	rows, err := s.db.QueryContext(ctx, metricsQuery, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	for rows.Next() {
		metric := &models.PerformanceMetrics{UserID: userID}
		err = rows.Scan(
			&metric.ID,
			&metric.Topic,
			&metric.Language,
			&metric.Level,
			&metric.TotalAttempts,
			&metric.CorrectAttempts,
			&metric.AverageResponseTimeMs,
			&metric.DifficultyAdjustment,
			&metric.LastUpdated,
		)
		if err != nil {
			return nil, err
		}

		key := metric.Topic + "_" + metric.Language + "_" + metric.Level
		progress.PerformanceByTopic[key] = metric
	}

	// Identify weak areas (accuracy < 60%)
	progress.WeakAreas = s.identifyWeakAreas(progress.PerformanceByTopic)

	// Get recent activity
	progress.RecentActivity, err = s.getRecentActivity(ctx, userID, 10)
	if err != nil {
		return nil, err
	}

	// Get current level from user
	currentLevel, err := s.getCurrentUserLevel(ctx, userID)
	if err != nil {
		return nil, err
	}
	progress.CurrentLevel = currentLevel

	// Suggest level adjustment if needed
	progress.SuggestedLevel = s.suggestLevelAdjustment(progress)

	return progress, nil
}

func (s *LearningService) identifyWeakAreas(metrics map[string]*models.PerformanceMetrics) []string {
	// Note: This is a pure function that doesn't need tracing since it doesn't make external calls
	// But we could add tracing if we want to track the analysis performance
	var weakAreas []string

	for key, metric := range metrics {
		if metric.TotalAttempts > 0 && metric.AccuracyRate() < 60.0 && metric.TotalAttempts >= 3 {
			weakAreas = append(weakAreas, key)
		}
	}

	return weakAreas
}

func (s *LearningService) getRecentActivity(ctx context.Context, userID, limit int) (result0 []models.UserResponse, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_recent_activity",
		observability.AttributeUserID(userID),
		attribute.Int("limit", limit),
	)
	defer observability.FinishSpan(span, &err)

	query := `
		SELECT id, user_id, question_id, user_answer_index, is_correct, response_time_ms, created_at
		FROM user_responses
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var responses []models.UserResponse
	for rows.Next() {
		var response models.UserResponse
		err = rows.Scan(
			&response.ID,
			&response.UserID,
			&response.QuestionID,
			&response.UserAnswerIndex,
			&response.IsCorrect,
			&response.ResponseTimeMs,
			&response.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		responses = append(responses, response)
	}

	return responses, nil
}

func (s *LearningService) getCurrentUserLevel(ctx context.Context, userID int) (result0 string, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_current_user_level",
		observability.AttributeUserID(userID),
	)
	defer observability.FinishSpan(span, &err)

	query := `SELECT current_level FROM users WHERE id = $1`

	var level sql.NullString
	err = s.db.QueryRowContext(ctx, query, userID).Scan(&level)
	if err != nil {
		return "", err
	}

	// Return default level if NULL
	if !level.Valid || level.String == "" {
		return "A1", nil // Default level
	}

	return level.String, nil
}

func (s *LearningService) suggestLevelAdjustment(progress *models.UserProgress) string {
	// Note: This is a pure function that doesn't need tracing since it doesn't make external calls
	// But we could add tracing if we want to track the analysis performance
	if progress.TotalQuestions < 20 {
		return "" // Not enough data
	}

	// If accuracy is consistently high (>85%), suggest level up
	if progress.AccuracyRate > 85.0 {
		return s.getNextLevel(progress.CurrentLevel)
	}

	// If accuracy is consistently low (<50%), suggest level down
	if progress.AccuracyRate < 50.0 {
		return s.getPreviousLevel(progress.CurrentLevel)
	}

	return ""
}

func (s *LearningService) getNextLevel(currentLevel string) string {
	// Note: This is a pure function that doesn't need tracing since it doesn't make external calls
	levels := s.cfg.GetAllLevels()

	for i, level := range levels {
		if level == currentLevel && i < len(levels)-1 {
			return levels[i+1]
		}
	}

	return currentLevel
}

func (s *LearningService) getPreviousLevel(currentLevel string) string {
	// Note: This is a pure function that doesn't need tracing since it doesn't make external calls
	levels := s.cfg.GetAllLevels()

	for i, level := range levels {
		if level == currentLevel && i > 0 {
			return levels[i-1]
		}
	}

	return currentLevel
}

// GetWeakestTopics returns the topics where the user performs poorest
func (s *LearningService) GetWeakestTopics(ctx context.Context, userID, limit int) (result0 []*models.PerformanceMetrics, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_weakest_topics",
		observability.AttributeUserID(userID),
		attribute.Int("limit", limit),
	)
	defer observability.FinishSpan(span, &err)

	query := `
		SELECT id, topic, language, level, total_attempts, correct_attempts, average_response_time_ms, difficulty_adjustment, last_updated
		FROM performance_metrics
		WHERE user_id = $1 AND total_attempts >= 3
		ORDER BY (correct_attempts * 1.0 / total_attempts) ASC, last_updated ASC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var topics []*models.PerformanceMetrics
	for rows.Next() {
		metric := &models.PerformanceMetrics{UserID: userID}
		err = rows.Scan(
			&metric.ID,
			&metric.Topic,
			&metric.Language,
			&metric.Level,
			&metric.TotalAttempts,
			&metric.CorrectAttempts,
			&metric.AverageResponseTimeMs,
			&metric.DifficultyAdjustment,
			&metric.LastUpdated,
		)
		if err != nil {
			return nil, err
		}
		topics = append(topics, metric)
	}

	return topics, nil
}

// ShouldAvoidQuestion determines if a question should be avoided for a user
func (s *LearningService) ShouldAvoidQuestion(ctx context.Context, userID, questionID int) (result0 bool, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "should_avoid_question",
		observability.AttributeUserID(userID),
		observability.AttributeQuestionID(questionID),
	)
	defer observability.FinishSpan(span, &err)

	// Determine user's local 1-day window and convert to UTC timestamps
	startUTC, endUTC, _, err := contextutils.UserLocalDayRange(ctx, userID, 1, s.getUserByID)
	if err != nil {
		return false, contextutils.WrapError(err, "failed to compute user local day range")
	}

	query := `
		SELECT COUNT(*)
		FROM user_responses
		WHERE user_id = $1 AND question_id = $2 AND is_correct = true
		AND created_at >= $3 AND created_at < $4
	`

	var count int
	err = s.db.QueryRowContext(ctx, query, userID, questionID, startUTC, endUTC).Scan(&count)

	span.SetAttributes(attribute.Bool("should_avoid", count > 0))
	return count > 0, err
}

// GetUserQuestionStats returns comprehensive per-user question statistics
func (s *LearningService) GetUserQuestionStats(ctx context.Context, userID int) (result0 *UserQuestionStats, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_user_question_stats",
		observability.AttributeUserID(userID),
	)
	defer observability.FinishSpan(span, &err)

	stats := &UserQuestionStats{
		UserID:           userID,
		AnsweredByType:   make(map[string]int),
		AnsweredByLevel:  make(map[string]int),
		AccuracyByType:   make(map[string]float64),
		AccuracyByLevel:  make(map[string]float64),
		AvailableByType:  make(map[string]int),
		AvailableByLevel: make(map[string]int),
	}

	// Get user's language and level preferences
	var userLanguage, userLevel string
	userQuery := `SELECT COALESCE(preferred_language, 'italian'), COALESCE(current_level, 'B1') FROM users WHERE id = $1`
	err = s.db.QueryRowContext(ctx, userQuery, userID).Scan(&userLanguage, &userLevel)
	if err != nil {
		return nil, err
	}

	span.SetAttributes(
		attribute.String("user.language", userLanguage),
		attribute.String("user.level", userLevel),
	)

	// Get questions answered by user with stats
	answeredQuery := `
		SELECT
			q.type,
			q.level,
			COUNT(*) as total,
			SUM(CASE WHEN ur.is_correct THEN 1 ELSE 0 END) as correct
		FROM user_responses ur
		JOIN questions q ON ur.question_id = q.id
		WHERE ur.user_id = $1
		GROUP BY q.type, q.level
	`

	rows, err := s.db.QueryContext(ctx, answeredQuery, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	for rows.Next() {
		var qType, level string
		var total, correct int

		if err := rows.Scan(&qType, &level, &total, &correct); err != nil {
			return nil, err
		}

		stats.AnsweredByType[qType] += total
		stats.AnsweredByLevel[level] += total
		stats.TotalAnswered += total

		// Calculate accuracy rates
		accuracy := float64(correct) / float64(total) * 100

		// For type accuracy, we need to aggregate across levels
		if _, exists := stats.AnsweredByType[qType]; exists {
			// Recalculate accuracy for this type
			typeQuery := `
				SELECT
					COUNT(*) as total,
					SUM(CASE WHEN ur.is_correct THEN 1 ELSE 0 END) as correct
				FROM user_responses ur
				JOIN questions q ON ur.question_id = q.id
				WHERE ur.user_id = $1 AND q.type = $2
			`
			var typeTotal, typeCorrect int
			if err := s.db.QueryRowContext(ctx, typeQuery, userID, qType).Scan(&typeTotal, &typeCorrect); err != nil {
				s.logger.Warn(ctx, "Failed to scan type query result", map[string]interface{}{"error": err.Error()})
			}
			if typeTotal > 0 {
				stats.AccuracyByType[qType] = float64(typeCorrect) / float64(typeTotal) * 100
			}
		} else {
			stats.AccuracyByType[qType] = accuracy
		}

		// For level accuracy
		if _, exists := stats.AnsweredByLevel[level]; exists {
			// Recalculate accuracy for this level
			levelQuery := `
				SELECT
					COUNT(*) as total,
					SUM(CASE WHEN ur.is_correct THEN 1 ELSE 0 END) as correct
				FROM user_responses ur
				JOIN questions q ON ur.question_id = q.id
				WHERE ur.user_id = $1 AND q.level = $2
			`
			var levelTotal, levelCorrect int
			if err := s.db.QueryRowContext(ctx, levelQuery, userID, level).Scan(&levelTotal, &levelCorrect); err != nil {
				s.logger.Warn(ctx, "Failed to scan level query result", map[string]interface{}{"error": err.Error()})
			}
			if levelTotal > 0 {
				stats.AccuracyByLevel[level] = float64(levelCorrect) / float64(levelTotal) * 100
			}
		} else {
			stats.AccuracyByLevel[level] = accuracy
		}
	}

	// Get available questions (not answered by user) that belong to this user
	availableQuery := `
		SELECT
			q.type,
			q.level,
			COUNT(*) as available
		FROM questions q
		JOIN user_questions uq ON uq.question_id = q.id
		WHERE uq.user_id = $1
		AND q.language = $2
		AND q.status = 'active'
		AND q.id NOT IN (
			SELECT DISTINCT question_id
			FROM user_responses
			WHERE user_id = $3
		)
		GROUP BY q.type, q.level
	`

	rows, err = s.db.QueryContext(ctx, availableQuery, userID, userLanguage, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	for rows.Next() {
		var qType, level string
		var available int

		if err := rows.Scan(&qType, &level, &available); err != nil {
			return nil, err
		}

		stats.AvailableByType[qType] += available
		stats.AvailableByLevel[level] += available
	}

	// Get recently answered questions (within last hour)
	recentQuery := `
		SELECT COUNT(*)
		FROM user_responses ur
		WHERE ur.user_id = $1
		AND ur.created_at > NOW() - INTERVAL '1 hour'
	`

	err = s.db.QueryRowContext(ctx, recentQuery, userID).Scan(&stats.RecentlyAnswered)
	if err != nil {
		stats.RecentlyAnswered = 0 // Default to 0 if query fails
	}

	// Calculate overall correct/incorrect answers and accuracy rate
	overallQuery := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN is_correct THEN 1 ELSE 0 END) as correct
		FROM user_responses
		WHERE user_id = $1
	`

	var total, correct int
	err = s.db.QueryRowContext(ctx, overallQuery, userID).Scan(&total, &correct)
	if err != nil {
		// Default values if query fails
		stats.CorrectAnswers = 0
		stats.IncorrectAnswers = 0
		stats.AccuracyRate = 0.0
	} else {
		stats.CorrectAnswers = correct
		stats.IncorrectAnswers = total - correct
		if total > 0 {
			stats.AccuracyRate = float64(correct) / float64(total) * 100
		} else {
			stats.AccuracyRate = 0.0
		}
	}

	return stats, nil
}

// PRIORITY SYSTEM METHODS

// RecordAnswerWithPriority records a user's response and updates priority scores
func (s *LearningService) RecordAnswerWithPriority(ctx context.Context, userID, questionID, answerIndex int, isCorrect bool, responseTime int) error {
	// Create user response object
	response := &models.UserResponse{
		UserID:          userID,
		QuestionID:      questionID,
		UserAnswerIndex: answerIndex,
		IsCorrect:       isCorrect,
		ResponseTimeMs:  responseTime,
		CreatedAt:       time.Now(),
	}

	// Use existing RecordUserResponse method
	err := s.RecordUserResponse(ctx, response)
	if err != nil {
		return contextutils.WrapError(err, "failed to record user response")
	}

	// Update priority score in background
	go s.updatePriorityScoreAsync(ctx, userID, questionID)

	return nil
}

// RecordAnswerWithPriorityReturningID records a user's response, updates priority async, and returns the new user_responses ID
func (s *LearningService) RecordAnswerWithPriorityReturningID(ctx context.Context, userID, questionID, answerIndex int, isCorrect bool, responseTime int) (int, error) {
	response := &models.UserResponse{
		UserID:          userID,
		QuestionID:      questionID,
		UserAnswerIndex: answerIndex,
		IsCorrect:       isCorrect,
		ResponseTimeMs:  responseTime,
		CreatedAt:       time.Now(),
	}

	// Insert and get ID
	if err := s.RecordUserResponse(ctx, response); err != nil {
		return 0, contextutils.WrapError(err, "failed to record user response")
	}

	// Update priority score in background
	go s.updatePriorityScoreAsync(ctx, userID, questionID)

	return response.ID, nil
}

// MarkQuestionAsKnown marks a question as known for a user with optional confidence level
func (s *LearningService) MarkQuestionAsKnown(ctx context.Context, userID, questionID int, confidenceLevel *int) (err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "mark_question_as_known",
		observability.AttributeUserID(userID),
		observability.AttributeQuestionID(questionID),
	)
	defer observability.FinishSpan(span, &err)

	// DEBUG: Log the attempt
	s.logger.Debug(ctx, "MarkQuestionAsKnown called", map[string]interface{}{
		"user_id":     userID,
		"question_id": questionID,
	})

	// Update user_question_metadata table with confidence level
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO user_question_metadata (user_id, question_id, marked_as_known, marked_as_known_at, confidence_level, created_at, updated_at)
		VALUES ($1, $2, TRUE, NOW(), $3, NOW(), NOW())
		ON CONFLICT (user_id, question_id) DO UPDATE
		SET marked_as_known = TRUE, marked_as_known_at = NOW(), confidence_level = $3, updated_at = NOW()
	`, userID, questionID, confidenceLevel)
	if err != nil {
		// DEBUG: Log the actual error
		s.logger.Debug(ctx, "MarkQuestionAsKnown error", map[string]interface{}{
			"user_id":     userID,
			"question_id": questionID,
			"error":       err.Error(),
			"error_type":  fmt.Sprintf("%T", err),
		})

		if isForeignKeyConstraintViolation(err) {
			s.logger.Debug(ctx, "Foreign key constraint violation detected", map[string]interface{}{
				"user_id":     userID,
				"question_id": questionID,
			})
			return contextutils.ErrQuestionNotFound
		}
		s.logger.Debug(ctx, "Not a foreign key constraint violation, returning original error", map[string]interface{}{
			"user_id":     userID,
			"question_id": questionID,
		})
		return err
	}

	s.logger.Debug(ctx, "MarkQuestionAsKnown succeeded", map[string]interface{}{
		"user_id":     userID,
		"question_id": questionID,
	})

	// Update priority score in background so the new confidence affects selection immediately
	go s.updatePriorityScoreAsync(ctx, userID, questionID)
	return nil
}

// GetUserLearningPreferences retrieves user learning preferences
func (s *LearningService) GetUserLearningPreferences(ctx context.Context, userID int) (result0 *models.UserLearningPreferences, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_user_learning_preferences",
		observability.AttributeUserID(userID),
	)
	defer observability.FinishSpan(span, &err)

	var prefs models.UserLearningPreferences
	err = s.db.QueryRowContext(ctx, `
        SELECT id, user_id, focus_on_weak_areas, include_review_questions, fresh_question_ratio,
               known_question_penalty, review_interval_days, weak_area_boost, daily_reminder_enabled,
               tts_voice, last_daily_reminder_sent, daily_goal, created_at, updated_at
        FROM user_learning_preferences
        WHERE user_id = $1
    `, userID).Scan(
		&prefs.ID, &prefs.UserID, &prefs.FocusOnWeakAreas, &prefs.IncludeReviewQuestions,
		&prefs.FreshQuestionRatio, &prefs.KnownQuestionPenalty, &prefs.ReviewIntervalDays,
		&prefs.WeakAreaBoost, &prefs.DailyReminderEnabled,
		&prefs.TTSVoice,
		&prefs.LastDailyReminderSent,
		&prefs.DailyGoal,
		&prefs.CreatedAt, &prefs.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Check if user exists before creating default preferences
		var userExists bool
		err = s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&userExists)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to check if user exists")
		}
		if !userExists {
			return nil, contextutils.WrapErrorf(contextutils.ErrRecordNotFound, "user %d not found", userID)
		}
		// Create default preferences if none exist
		return s.createDefaultPreferences(ctx, userID)
	}

	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get user preferences")
	}

	return &prefs, nil
}

// UpdateLastDailyReminderSent updates the last daily reminder sent timestamp for a user
func (s *LearningService) UpdateLastDailyReminderSent(ctx context.Context, userID int) (err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "update_last_daily_reminder_sent",
		observability.AttributeUserID(userID),
	)
	defer observability.FinishSpan(span, &err)

	// Use INSERT ... ON CONFLICT to create the record if it doesn't exist
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO user_learning_preferences (user_id, last_daily_reminder_sent, updated_at)
		VALUES ($1, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			last_daily_reminder_sent = NOW(),
			updated_at = NOW()
	`, userID)
	if err != nil {
		return contextutils.WrapError(err, "failed to update last daily reminder sent")
	}

	return nil
}

// UpdateUserLearningPreferences updates user learning preferences
func (s *LearningService) UpdateUserLearningPreferences(ctx context.Context, userID int, prefs *models.UserLearningPreferences) (result0 *models.UserLearningPreferences, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "update_user_learning_preferences",
		observability.AttributeUserID(userID),
		attribute.Bool("prefs.focus_on_weak_areas", prefs.FocusOnWeakAreas),
		attribute.Bool("prefs.include_review_questions", prefs.IncludeReviewQuestions),
		attribute.Float64("prefs.fresh_question_ratio", prefs.FreshQuestionRatio),
		attribute.Float64("prefs.known_question_penalty", prefs.KnownQuestionPenalty),
		attribute.Int("prefs.review_interval_days", prefs.ReviewIntervalDays),
		attribute.Float64("prefs.weak_area_boost", prefs.WeakAreaBoost),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	var updatedPrefs models.UserLearningPreferences
	err = s.db.QueryRowContext(ctx, `
        UPDATE user_learning_preferences
        SET focus_on_weak_areas = $2, include_review_questions = $3, fresh_question_ratio = $4,
            known_question_penalty = $5, review_interval_days = $6, weak_area_boost = $7,
            daily_reminder_enabled = $8, tts_voice = $9, daily_goal = COALESCE(NULLIF($10, 0), daily_goal), updated_at = NOW()
        WHERE user_id = $1
        RETURNING id, user_id, focus_on_weak_areas, include_review_questions, fresh_question_ratio,
                  known_question_penalty, review_interval_days, weak_area_boost, daily_reminder_enabled,
                  tts_voice, last_daily_reminder_sent, daily_goal, created_at, updated_at
    `, userID, prefs.FocusOnWeakAreas, prefs.IncludeReviewQuestions, prefs.FreshQuestionRatio,
		prefs.KnownQuestionPenalty, prefs.ReviewIntervalDays, prefs.WeakAreaBoost, prefs.DailyReminderEnabled, prefs.TTSVoice, prefs.DailyGoal).Scan(
		&updatedPrefs.ID, &updatedPrefs.UserID, &updatedPrefs.FocusOnWeakAreas, &updatedPrefs.IncludeReviewQuestions,
		&updatedPrefs.FreshQuestionRatio, &updatedPrefs.KnownQuestionPenalty, &updatedPrefs.ReviewIntervalDays,
		&updatedPrefs.WeakAreaBoost, &updatedPrefs.DailyReminderEnabled, &updatedPrefs.TTSVoice, &updatedPrefs.LastDailyReminderSent,
		&updatedPrefs.DailyGoal, &updatedPrefs.CreatedAt, &updatedPrefs.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// If no preferences exist, create them with the provided values
		return s.createPreferencesWithValues(ctx, userID, prefs)
	}

	if err != nil {
		return nil, contextutils.WrapError(err, "failed to update user preferences")
	}

	return &updatedPrefs, nil
}

// createPreferencesWithValues creates learning preferences for a user with the provided values
func (s *LearningService) createPreferencesWithValues(ctx context.Context, userID int, prefs *models.UserLearningPreferences) (result0 *models.UserLearningPreferences, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "create_preferences_with_values",
		observability.AttributeUserID(userID),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Use the provided values, falling back to defaults for any missing fields
	defaultPrefs := s.GetDefaultLearningPreferences()
	prefs.UserID = userID

	// Merge provided values with defaults
	if prefs.FocusOnWeakAreas == defaultPrefs.FocusOnWeakAreas && !prefs.FocusOnWeakAreas {
		prefs.FocusOnWeakAreas = defaultPrefs.FocusOnWeakAreas
	}
	if prefs.IncludeReviewQuestions == defaultPrefs.IncludeReviewQuestions && !prefs.IncludeReviewQuestions {
		prefs.IncludeReviewQuestions = defaultPrefs.IncludeReviewQuestions
	}
	if prefs.FreshQuestionRatio == 0 {
		prefs.FreshQuestionRatio = defaultPrefs.FreshQuestionRatio
	}
	if prefs.KnownQuestionPenalty == 0 {
		prefs.KnownQuestionPenalty = defaultPrefs.KnownQuestionPenalty
	}
	if prefs.ReviewIntervalDays == 0 {
		prefs.ReviewIntervalDays = defaultPrefs.ReviewIntervalDays
	}
	if prefs.WeakAreaBoost == 0 {
		prefs.WeakAreaBoost = defaultPrefs.WeakAreaBoost
	}
	if prefs.DailyGoal == 0 {
		prefs.DailyGoal = defaultPrefs.DailyGoal
	}

	// Try to insert with ON CONFLICT DO NOTHING to handle race conditions
	_, err = s.db.ExecContext(ctx, `
        INSERT INTO user_learning_preferences (user_id, focus_on_weak_areas, include_review_questions,
                                               fresh_question_ratio, known_question_penalty,
                                               review_interval_days, weak_area_boost, daily_reminder_enabled,
                                               tts_voice, daily_goal, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
        ON CONFLICT (user_id) DO NOTHING
    `, userID, prefs.FocusOnWeakAreas, prefs.IncludeReviewQuestions,
		prefs.FreshQuestionRatio, prefs.KnownQuestionPenalty,
		prefs.ReviewIntervalDays, prefs.WeakAreaBoost, prefs.DailyReminderEnabled, prefs.TTSVoice, prefs.DailyGoal)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to create preferences with values")
	}

	// Now fetch the preferences (either the ones we just created or the ones created by another concurrent request)
	err = s.db.QueryRowContext(ctx, `
        SELECT id, user_id, focus_on_weak_areas, include_review_questions, fresh_question_ratio,
               known_question_penalty, review_interval_days, weak_area_boost, daily_reminder_enabled,
               tts_voice, last_daily_reminder_sent, daily_goal, created_at, updated_at
        FROM user_learning_preferences
        WHERE user_id = $1
    `, userID).Scan(
		&prefs.ID, &prefs.UserID, &prefs.FocusOnWeakAreas, &prefs.IncludeReviewQuestions,
		&prefs.FreshQuestionRatio, &prefs.KnownQuestionPenalty, &prefs.ReviewIntervalDays,
		&prefs.WeakAreaBoost, &prefs.DailyReminderEnabled, &prefs.TTSVoice, &prefs.LastDailyReminderSent,
		&prefs.DailyGoal, &prefs.CreatedAt, &prefs.UpdatedAt,
	)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to fetch created preferences")
	}

	return prefs, nil
}

// createDefaultPreferences creates default learning preferences for a user
func (s *LearningService) createDefaultPreferences(ctx context.Context, userID int) (result0 *models.UserLearningPreferences, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "create_default_preferences",
		observability.AttributeUserID(userID),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	defaultPrefs := s.GetDefaultLearningPreferences()
	defaultPrefs.UserID = userID

	// Try to insert with ON CONFLICT DO NOTHING to handle race conditions
	_, err = s.db.ExecContext(ctx, `
        INSERT INTO user_learning_preferences (user_id, focus_on_weak_areas, include_review_questions,
                                               fresh_question_ratio, known_question_penalty,
                                               review_interval_days, weak_area_boost, daily_reminder_enabled,
                                               tts_voice, daily_goal, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
        ON CONFLICT (user_id) DO NOTHING
    `, userID, defaultPrefs.FocusOnWeakAreas, defaultPrefs.IncludeReviewQuestions,
		defaultPrefs.FreshQuestionRatio, defaultPrefs.KnownQuestionPenalty,
		defaultPrefs.ReviewIntervalDays, defaultPrefs.WeakAreaBoost, defaultPrefs.DailyReminderEnabled, defaultPrefs.TTSVoice, defaultPrefs.DailyGoal)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to create default preferences")
	}

	// Now fetch the preferences (either the ones we just created or the ones created by another concurrent request)
	err = s.db.QueryRowContext(ctx, `
        SELECT id, user_id, focus_on_weak_areas, include_review_questions, fresh_question_ratio,
               known_question_penalty, review_interval_days, weak_area_boost, daily_reminder_enabled,
               tts_voice, last_daily_reminder_sent, daily_goal, created_at, updated_at
        FROM user_learning_preferences
        WHERE user_id = $1
    `, userID).Scan(
		&defaultPrefs.ID, &defaultPrefs.UserID, &defaultPrefs.FocusOnWeakAreas, &defaultPrefs.IncludeReviewQuestions,
		&defaultPrefs.FreshQuestionRatio, &defaultPrefs.KnownQuestionPenalty, &defaultPrefs.ReviewIntervalDays,
		&defaultPrefs.WeakAreaBoost, &defaultPrefs.DailyReminderEnabled, &defaultPrefs.TTSVoice, &defaultPrefs.LastDailyReminderSent,
		&defaultPrefs.DailyGoal, &defaultPrefs.CreatedAt, &defaultPrefs.UpdatedAt,
	)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to fetch created preferences")
	}

	return defaultPrefs, nil
}

// GetDefaultLearningPreferences returns default learning preferences
func (s *LearningService) GetDefaultLearningPreferences() *models.UserLearningPreferences {
	return &models.UserLearningPreferences{
		FocusOnWeakAreas:       true,
		IncludeReviewQuestions: true,
		FreshQuestionRatio:     0.3,
		KnownQuestionPenalty:   0.1,
		ReviewIntervalDays:     7,
		WeakAreaBoost:          2.0,
		DailyReminderEnabled:   false, // Default to false for daily reminders
		DailyGoal:              10,
		TTSVoice:               "",
	}
}

// CalculatePriorityScore calculates priority score for a specific question for a user
func (s *LearningService) CalculatePriorityScore(ctx context.Context, userID, questionID int) (result0 float64, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "calculate_priority_score",
		observability.AttributeUserID(userID),
		observability.AttributeQuestionID(questionID),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Get user preferences
	prefs, err := s.GetUserLearningPreferences(ctx, userID)
	if err != nil {
		return 0, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get user preferences: %v", err)
	}

	// Get user's performance history for this question
	performance, err := s.getQuestionPerformance(ctx, userID, questionID)
	if err != nil {
		return 0, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get question performance: %v", err)
	}

	// Calculate components
	baseScore := 100.0
	performanceMultiplier := s.calculatePerformanceMultiplier(performance, prefs.WeakAreaBoost)
	spacedRepetitionBoost := s.calculateSpacedRepetitionBoost(performance.LastSeenAt)
	userPreferenceMultiplier := s.calculateUserPreferenceMultiplier(performance, prefs)
	freshnessBoost := s.calculateFreshnessBoost(performance.TimesAnswered)

	// Final score with bounds checking
	finalScore := baseScore * performanceMultiplier * spacedRepetitionBoost * userPreferenceMultiplier * freshnessBoost

	// Apply bounds to prevent extreme values
	if finalScore < 1.0 {
		finalScore = 1.0
	} else if finalScore > 1000.0 {
		finalScore = 1000.0
	}

	return finalScore, nil
}

// updatePriorityScoreAsync updates priority score for a question asynchronously
func (s *LearningService) updatePriorityScoreAsync(ctx context.Context, userID, questionID int) {
	ctx, span := observability.TraceLearningFunction(ctx, "update_priority_score_async",
		observability.AttributeUserID(userID),
		observability.AttributeQuestionID(questionID),
	)
	defer span.End()

	score, err := s.CalculatePriorityScore(ctx, userID, questionID)
	if err != nil {
		s.logger.Error(ctx, "Failed to calculate priority score", err, map[string]interface{}{
			"user_id":     userID,
			"question_id": questionID,
		})
		return
	}

	// Update or insert priority score
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO question_priority_scores (user_id, question_id, priority_score, last_calculated_at, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW(), NOW())
		ON CONFLICT (user_id, question_id) DO UPDATE
		SET priority_score = $3, last_calculated_at = NOW(), updated_at = NOW()
	`, userID, questionID, score)
	if err != nil {
		s.logger.Error(ctx, "Failed to update priority score", err, map[string]interface{}{
			"user_id":     userID,
			"question_id": questionID,
			"score":       score,
		})
	}
}

// QuestionPerformance represents performance data for a specific question
type QuestionPerformance struct {
	TimesAnswered   int
	CorrectAnswers  int
	LastSeenAt      *time.Time
	MarkedAsKnown   bool
	MarkedAsKnownAt *time.Time
	ConfidenceLevel *int
}

// getQuestionPerformance retrieves performance data for a specific question
func (s *LearningService) getQuestionPerformance(ctx context.Context, userID, questionID int) (result0 *QuestionPerformance, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_question_performance",
		observability.AttributeUserID(userID),
		observability.AttributeQuestionID(questionID),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	performance := &QuestionPerformance{}

	// Get response statistics
	err = s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as times_answered,
			COALESCE(SUM(CASE WHEN is_correct THEN 1 ELSE 0 END), 0) as correct_answers,
			MAX(created_at) as last_seen_at
		FROM user_responses
		WHERE user_id = $1 AND question_id = $2
	`, userID, questionID).Scan(
		&performance.TimesAnswered,
		&performance.CorrectAnswers,
		&performance.LastSeenAt,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get response statistics: %v", err)
	}

	// Get metadata
	var markedAsKnownAt sql.NullTime
	var confidenceLevel sql.NullInt32
	err = s.db.QueryRowContext(ctx, `
		SELECT marked_as_known, marked_as_known_at, confidence_level
		FROM user_question_metadata
		WHERE user_id = $1 AND question_id = $2
	`, userID, questionID).Scan(&performance.MarkedAsKnown, &markedAsKnownAt, &confidenceLevel)

	if err != nil && err != sql.ErrNoRows {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get question metadata: %v", err)
	}

	if markedAsKnownAt.Valid {
		performance.MarkedAsKnownAt = &markedAsKnownAt.Time
	}

	if confidenceLevel.Valid {
		level := int(confidenceLevel.Int32)
		performance.ConfidenceLevel = &level
	}

	return performance, nil
}

// calculatePerformanceMultiplier calculates the performance-based multiplier
func (s *LearningService) calculatePerformanceMultiplier(performance *QuestionPerformance, weakAreaBoost float64) float64 {
	// Note: This is a pure function that doesn't need tracing since it doesn't make external calls
	if performance.TimesAnswered == 0 {
		return 1.0 // Neutral for new questions
	}

	errorRate := float64(performance.TimesAnswered-performance.CorrectAnswers) / float64(performance.TimesAnswered)
	successRate := float64(performance.CorrectAnswers) / float64(performance.TimesAnswered)

	// Apply weak area boost for questions with high error rates
	multiplier := 1.0 + (errorRate * weakAreaBoost) - (successRate * 0.5)

	// Apply bounds to prevent extreme values
	if multiplier < 0.1 {
		multiplier = 0.1
	} else if multiplier > 10.0 {
		multiplier = 10.0
	}

	return multiplier
}

// calculateSpacedRepetitionBoost calculates the spaced repetition boost
func (s *LearningService) calculateSpacedRepetitionBoost(lastSeenAt *time.Time) float64 {
	// Note: This is a pure function that doesn't need tracing since it doesn't make external calls
	if lastSeenAt == nil {
		return 1.0 // No boost for never-seen questions
	}

	daysSinceLastSeen := time.Since(*lastSeenAt).Hours() / 24.0
	boost := 1.0 + (daysSinceLastSeen * 0.1)

	// Cap the boost at 5.0x multiplier
	return math.Min(boost, 5.0)
}

// calculateUserPreferenceMultiplier calculates how user preference ("mark known" with confidence)
// influences question priority.
//
// New policy:
// - Confidence 1–2: show MORE (boost priority) → multipliers > 1
// - Confidence 3: neutral → multiplier = 1
// - Confidence 4–5: show LESS (reduce priority) → multiplier < 1 using KnownQuestionPenalty
func (s *LearningService) calculateUserPreferenceMultiplier(performance *QuestionPerformance, prefs *models.UserLearningPreferences) float64 {
	// Note: This is a pure function that doesn't need tracing since it doesn't make external calls
	if performance.MarkedAsKnown {
		if performance.ConfidenceLevel != nil {
			switch *performance.ConfidenceLevel {
			case 1:
				// Low confidence → increase frequency noticeably
				return 1.25
			case 2:
				// Some confidence → slight increase in frequency
				return 1.10
			case 3:
				// Neutral → no change
				return 1.0
			case 4:
				// Very confident → decrease frequency using half of penalty
				return prefs.KnownQuestionPenalty * 0.5
			case 5:
				// Extremely confident → strong decrease using 10% of penalty
				return prefs.KnownQuestionPenalty * 0.1
			default:
				return 1.0
			}
		}
		// Fallback when confidence not provided → use configured penalty
		return prefs.KnownQuestionPenalty
	}
	return 1.0
}

// calculateFreshnessBoost calculates the freshness boost for new questions
func (s *LearningService) calculateFreshnessBoost(timesAnswered int) float64 {
	// Note: This is a pure function that doesn't need tracing since it doesn't make external calls
	if timesAnswered == 0 {
		return 1.5 // Boost for fresh questions
	}
	return 1.0
}

// isForeignKeyConstraintViolation checks if the error is a foreign key constraint violation
func isForeignKeyConstraintViolation(err error) bool {
	if err == nil {
		return false
	}

	// Check for PostgreSQL foreign key constraint violation error code
	if pqErr, ok := err.(*pq.Error); ok {
		// PostgreSQL error code 23503 is for foreign key constraint violations
		if pqErr.Code == "23503" {
			return true
		}
	}

	// Also check for the error message pattern as a fallback
	errorStr := err.Error()
	return strings.Contains(errorStr, "violates foreign key constraint")
}

// Analytics Methods

// GetPriorityScoreDistribution returns the distribution of priority scores
func (s *LearningService) GetPriorityScoreDistribution(ctx context.Context) (result0 map[string]interface{}, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_priority_score_distribution")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT
			COUNT(CASE WHEN qps.priority_score > 200 THEN 1 END) as high,
			COUNT(CASE WHEN qps.priority_score BETWEEN 100 AND 200 THEN 1 END) as medium,
			COUNT(CASE WHEN qps.priority_score < 100 THEN 1 END) as low,
			AVG(qps.priority_score) as average
		FROM question_priority_scores qps
		JOIN questions q ON qps.question_id = q.id
		WHERE qps.priority_score > 0
	`

	var high, medium, low int
	var average sql.NullFloat64

	err = s.db.QueryRowContext(ctx, query).Scan(&high, &medium, &low, &average)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get priority score distribution: %v", err)
	}

	result := map[string]interface{}{
		"high":    high,
		"medium":  medium,
		"low":     low,
		"average": 0.0,
	}

	if average.Valid {
		result["average"] = average.Float64
	}

	span.SetAttributes(
		attribute.Int("high_count", high),
		attribute.Int("medium_count", medium),
		attribute.Int("low_count", low),
		attribute.Float64("average_score", result["average"].(float64)),
	)

	return result, nil
}

// GetHighPriorityQuestions returns the highest priority questions
func (s *LearningService) GetHighPriorityQuestions(ctx context.Context, limit int) (result0 []map[string]interface{}, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_high_priority_questions",
		attribute.Int("limit", limit),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT
			q.type as question_type,
			q.level,
			q.topic_category as topic,
			qps.priority_score
		FROM question_priority_scores qps
		JOIN questions q ON qps.question_id = q.id
		WHERE qps.priority_score > 200
		ORDER BY qps.priority_score DESC
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get high priority questions: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var questions []map[string]interface{}
	for rows.Next() {
		var questionType, level, topic sql.NullString
		var priorityScore float64

		err = rows.Scan(&questionType, &level, &topic, &priorityScore)
		if err != nil {
			continue
		}

		question := map[string]interface{}{
			"question_type":  questionType.String,
			"level":          level.String,
			"topic":          topic.String,
			"priority_score": priorityScore,
		}
		questions = append(questions, question)
	}

	span.SetAttributes(attribute.Int("questions_count", len(questions)))
	return questions, nil
}

// GetWeakAreasByTopic returns weak areas by topic
func (s *LearningService) GetWeakAreasByTopic(ctx context.Context, limit int) (result0 []map[string]interface{}, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_weak_areas_by_topic",
		attribute.Int("limit", limit),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT
			topic,
			SUM(total_attempts) as total_attempts,
			SUM(correct_attempts) as correct_attempts
		FROM performance_metrics
		WHERE total_attempts > 0
		GROUP BY topic
		ORDER BY (SUM(correct_attempts)::float / SUM(total_attempts)) ASC
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get weak areas: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var weakAreas []map[string]interface{}
	for rows.Next() {
		var topic sql.NullString
		var totalAttempts, correctAttempts int

		err = rows.Scan(&topic, &totalAttempts, &correctAttempts)
		if err != nil {
			continue
		}

		area := map[string]interface{}{
			"topic":            topic.String,
			"total_attempts":   totalAttempts,
			"correct_attempts": correctAttempts,
		}
		weakAreas = append(weakAreas, area)
	}

	span.SetAttributes(attribute.Int("weak_areas_count", len(weakAreas)))
	return weakAreas, nil
}

// GetLearningPreferencesUsage returns learning preferences usage statistics
func (s *LearningService) GetLearningPreferencesUsage(ctx context.Context) (result0 map[string]interface{}, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_learning_preferences_usage")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT
			COUNT(*) as total_users,
			AVG(focus_on_weak_areas::int) as avg_focus_on_weak_areas,
			AVG(fresh_question_ratio) as avg_fresh_question_ratio,
			AVG(weak_area_boost) as avg_weak_area_boost,
			AVG(known_question_penalty) as avg_known_question_penalty
		FROM user_learning_preferences
	`

	var totalUsers int
	var avgFocusOnWeakAreas, avgFreshQuestionRatio, avgWeakAreaBoost, avgKnownQuestionPenalty sql.NullFloat64

	err = s.db.QueryRowContext(ctx, query).Scan(
		&totalUsers,
		&avgFocusOnWeakAreas,
		&avgFreshQuestionRatio,
		&avgWeakAreaBoost,
		&avgKnownQuestionPenalty,
	)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get learning preferences usage: %v", err)
	}

	result := map[string]interface{}{
		"total_users":          0,
		"focusOnWeakAreas":     false,
		"freshQuestionRatio":   0.3,
		"weakAreaBoost":        2.0,
		"knownQuestionPenalty": 0.1,
	}

	if totalUsers > 0 {
		result["total_users"] = totalUsers
		if avgFocusOnWeakAreas.Valid {
			result["focusOnWeakAreas"] = avgFocusOnWeakAreas.Float64 > 0.5
		}
		if avgFreshQuestionRatio.Valid {
			result["freshQuestionRatio"] = avgFreshQuestionRatio.Float64
		}
		if avgWeakAreaBoost.Valid {
			result["weakAreaBoost"] = avgWeakAreaBoost.Float64
		}
		if avgKnownQuestionPenalty.Valid {
			result["knownQuestionPenalty"] = avgKnownQuestionPenalty.Float64
		}
	}

	span.SetAttributes(
		attribute.Int("total_users", result["total_users"].(int)),
		attribute.Bool("focus_on_weak_areas", result["focusOnWeakAreas"].(bool)),
		attribute.Float64("fresh_question_ratio", result["freshQuestionRatio"].(float64)),
		attribute.Float64("weak_area_boost", result["weakAreaBoost"].(float64)),
		attribute.Float64("known_question_penalty", result["knownQuestionPenalty"].(float64)),
	)

	return result, nil
}

// GetQuestionTypeGaps returns gaps in question types
func (s *LearningService) GetQuestionTypeGaps(ctx context.Context) (result0 []map[string]interface{}, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_question_type_gaps")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT
			q.type as question_type,
			q.level,
			COUNT(q.id) as available,
			COUNT(qps.question_id) as with_priority_scores
		FROM questions q
		LEFT JOIN question_priority_scores qps ON q.id = qps.question_id
		GROUP BY q.type, q.level
		HAVING COUNT(qps.question_id) < COUNT(q.id) * 0.8
		ORDER BY (COUNT(qps.question_id)::float / COUNT(q.id)) ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		span.SetAttributes(attribute.String("error.type", "database_query_failed"), attribute.String("error", err.Error()))
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get question type gaps: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows in GetQuestionTypeGaps", map[string]interface{}{"error": err.Error()})
		}
	}()

	var gaps []map[string]interface{}
	var scanErrors int

	for rows.Next() {
		var questionType, level sql.NullString
		var available, withPriorityScores int

		err = rows.Scan(&questionType, &level, &available, &withPriorityScores)
		if err != nil {
			scanErrors++
			span.SetAttributes(attribute.String("error.type", "row_scan_failed"), attribute.String("error", err.Error()))
			continue
		}

		gap := map[string]interface{}{
			"question_type": questionType.String,
			"level":         level.String,
			"available":     available,
			"demand":        available - withPriorityScores,
		}
		gaps = append(gaps, gap)
	}

	if err := rows.Err(); err != nil {
		span.SetAttributes(attribute.String("error.type", "rows_iteration_failed"), attribute.String("error", err.Error()))
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "error during rows iteration: %v", err)
	}

	span.SetAttributes(
		attribute.Int("gaps_count", len(gaps)),
		attribute.Int("scan_errors", scanErrors),
	)
	return gaps, nil
}

// GetGenerationSuggestions returns suggestions for question generation
func (s *LearningService) GetGenerationSuggestions(ctx context.Context) (result0 []map[string]interface{}, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_generation_suggestions")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT
			q.type as question_type,
			q.level,
			q.language,
			COUNT(q.id) as available,
			COUNT(CASE WHEN qps.priority_score > 100 THEN 1 END) as high_priority,
			AVG(qps.priority_score) as avg_priority
		FROM questions q
		LEFT JOIN question_priority_scores qps ON q.id = qps.question_id
		GROUP BY q.type, q.level, q.language
		HAVING COUNT(q.id) < 50 OR COUNT(CASE WHEN qps.priority_score > 100 THEN 1 END) < 10
		ORDER BY COUNT(q.id) ASC, AVG(qps.priority_score) DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		span.SetAttributes(attribute.String("error.type", "database_query_failed"), attribute.String("error", err.Error()))
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get generation suggestions: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows in GetGenerationSuggestions", map[string]interface{}{"error": err.Error()})
		}
	}()

	var suggestions []map[string]interface{}
	var scanErrors int

	for rows.Next() {
		var questionType, level, language sql.NullString
		var available, highPriority int
		var avgPriority sql.NullFloat64

		err = rows.Scan(&questionType, &level, &language, &available, &highPriority, &avgPriority)
		if err != nil {
			scanErrors++
			span.SetAttributes(attribute.String("error.type", "row_scan_failed"), attribute.String("error", err.Error()))
			continue
		}

		suggestion := map[string]interface{}{
			"question_type":  questionType.String,
			"level":          level.String,
			"language":       language.String,
			"available":      available,
			"high_priority":  highPriority,
			"avg_priority":   0.0,
			"priority_score": 0.0,
		}

		if avgPriority.Valid {
			suggestion["avg_priority"] = avgPriority.Float64
			suggestion["priority_score"] = avgPriority.Float64
		}

		suggestions = append(suggestions, suggestion)
	}

	if err := rows.Err(); err != nil {
		span.SetAttributes(attribute.String("error.type", "rows_iteration_failed"), attribute.String("error", err.Error()))
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "error during rows iteration: %v", err)
	}

	span.SetAttributes(
		attribute.Int("suggestions_count", len(suggestions)),
		attribute.Int("scan_errors", scanErrors),
	)
	return suggestions, nil
}

// GetPrioritySystemPerformance returns performance metrics for the priority system
func (s *LearningService) GetPrioritySystemPerformance(ctx context.Context) (result0 map[string]interface{}, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_priority_system_performance")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// This is a simplified implementation - in a real system, this would track actual performance metrics
	query := `
		SELECT
			COUNT(*) as total_calculations,
			AVG(priority_score) as avg_score,
			MAX(last_calculated_at) as last_calculation
		FROM question_priority_scores
		WHERE last_calculated_at > NOW() - INTERVAL '1 hour'
	`

	var totalCalculations int
	var avgScore sql.NullFloat64
	var lastCalculation sql.NullTime

	err = s.db.QueryRowContext(ctx, query).Scan(&totalCalculations, &avgScore, &lastCalculation)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get priority system performance: %v", err)
	}

	result := map[string]interface{}{
		"calculationsPerSecond": float64(totalCalculations) / 3600.0, // Per hour converted to per second
		"avgCalculationTime":    0.0,                                 // Would need to track actual calculation times
		"avgQueryTime":          0.0,                                 // Would need to track actual query times
		"memoryUsage":           0.0,                                 // Would need to track actual memory usage
		"avgScore":              0.0,                                 // Default value
	}

	if avgScore.Valid {
		result["avgScore"] = avgScore.Float64
	}

	if lastCalculation.Valid {
		result["lastCalculation"] = lastCalculation.Time.Format(time.RFC3339)
	}

	span.SetAttributes(
		attribute.Float64("calculations_per_second", result["calculationsPerSecond"].(float64)),
		attribute.Float64("avg_score", result["avgScore"].(float64)),
		attribute.Int("total_calculations", totalCalculations),
	)

	return result, nil
}

// GetBackgroundJobsStatus returns the status of background jobs
func (s *LearningService) GetBackgroundJobsStatus(ctx context.Context) (result0 map[string]interface{}, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_background_jobs_status")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// This is a simplified implementation - in a real system, this would track actual background job status
	query := `
		SELECT
			COUNT(*) as total_updates,
			MAX(updated_at) as last_update
		FROM question_priority_scores
		WHERE updated_at > NOW() - INTERVAL '1 minute'
	`

	var totalUpdates int
	var lastUpdate sql.NullTime

	err = s.db.QueryRowContext(ctx, query).Scan(&totalUpdates, &lastUpdate)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get background jobs status")
	}

	result := map[string]interface{}{
		"priorityUpdates": totalUpdates,
		"lastUpdate":      "N/A",
		"queueSize":       0, // Would need to track actual queue size
		"status":          "healthy",
	}

	if lastUpdate.Valid {
		result["lastUpdate"] = lastUpdate.Time.Format(time.RFC3339)
	}

	if totalUpdates == 0 {
		result["status"] = "idle"
	}

	span.SetAttributes(
		attribute.Int("priority_updates", totalUpdates),
		attribute.String("status", result["status"].(string)),
		attribute.Int("queue_size", result["queueSize"].(int)),
	)

	return result, nil
}

// GetUserPriorityScoreDistribution returns priority score distribution for a specific user
func (s *LearningService) GetUserPriorityScoreDistribution(ctx context.Context, userID int) (result0 map[string]interface{}, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_user_priority_score_distribution",
		observability.AttributeUserID(userID),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT
			COUNT(CASE WHEN priority_score > 200 THEN 1 END) as high,
			COUNT(CASE WHEN priority_score BETWEEN 100 AND 200 THEN 1 END) as medium,
			COUNT(CASE WHEN priority_score < 100 THEN 1 END) as low,
			AVG(priority_score) as average
		FROM question_priority_scores
		WHERE user_id = $1 AND priority_score > 0
	`

	var high, medium, low int
	var average sql.NullFloat64

	err = s.db.QueryRowContext(ctx, query, userID).Scan(&high, &medium, &low, &average)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get user priority score distribution")
	}

	result := map[string]interface{}{
		"high":    high,
		"medium":  medium,
		"low":     low,
		"average": 0.0,
	}

	if average.Valid {
		result["average"] = average.Float64
	}

	span.SetAttributes(
		attribute.Int("high_count", high),
		attribute.Int("medium_count", medium),
		attribute.Int("low_count", low),
		attribute.Float64("average_score", result["average"].(float64)),
	)

	return result, nil
}

// GetUserHighPriorityQuestions returns the highest priority questions for a specific user
func (s *LearningService) GetUserHighPriorityQuestions(ctx context.Context, userID, limit int) (result0 []map[string]interface{}, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_user_high_priority_questions",
		observability.AttributeUserID(userID),
		attribute.Int("limit", limit),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT
			q.type as question_type,
			q.level,
			q.topic_category as topic,
			qps.priority_score
		FROM question_priority_scores qps
		JOIN questions q ON qps.question_id = q.id
		WHERE qps.user_id = $1 AND qps.priority_score > 200
		ORDER BY qps.priority_score DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get user high priority questions")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var questions []map[string]interface{}
	for rows.Next() {
		var questionType, level, topic sql.NullString
		var priorityScore float64

		err = rows.Scan(&questionType, &level, &topic, &priorityScore)
		if err != nil {
			continue
		}

		question := map[string]interface{}{
			"question_type":  questionType.String,
			"level":          level.String,
			"topic":          topic.String,
			"priority_score": priorityScore,
		}
		questions = append(questions, question)
	}

	span.SetAttributes(attribute.Int("questions_count", len(questions)))
	return questions, nil
}

// GetUserWeakAreas returns weak areas for a specific user
func (s *LearningService) GetUserWeakAreas(ctx context.Context, userID, limit int) (result0 []map[string]interface{}, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_user_weak_areas",
		observability.AttributeUserID(userID),
		attribute.Int("limit", limit),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT
			topic,
			total_attempts,
			correct_attempts
		FROM performance_metrics
		WHERE user_id = $1 AND total_attempts > 0
		ORDER BY (correct_attempts::float / total_attempts) ASC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get user weak areas")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var weakAreas []map[string]interface{}
	for rows.Next() {
		var topic sql.NullString
		var totalAttempts, correctAttempts int

		err = rows.Scan(&topic, &totalAttempts, &correctAttempts)
		if err != nil {
			continue
		}

		area := map[string]interface{}{
			"topic":            topic.String,
			"total_attempts":   totalAttempts,
			"correct_attempts": correctAttempts,
		}
		weakAreas = append(weakAreas, area)
	}

	span.SetAttributes(attribute.Int("weak_areas_count", len(weakAreas)))
	return weakAreas, nil
}

// Priority generation methods moved to worker

// GetHighPriorityTopics returns topics with high average priority scores for a user
func (s *LearningService) GetHighPriorityTopics(ctx context.Context, userID int) (result0 []string, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_high_priority_topics",
		observability.AttributeUserID(userID),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT q.topic_category, AVG(qps.priority_score) as avg_score
		FROM questions q
		JOIN user_questions uq ON q.id = uq.question_id
		JOIN question_priority_scores qps ON q.id = qps.question_id AND qps.user_id = $1
		WHERE uq.user_id = $1
		AND q.topic_category IS NOT NULL
		AND q.topic_category != ''
		GROUP BY q.topic_category
		HAVING AVG(qps.priority_score) >= 150.0
		ORDER BY avg_score DESC
		LIMIT 5
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get high priority topics")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var topics []string
	for rows.Next() {
		var topic string
		var avgScore float64
		if err := rows.Scan(&topic, &avgScore); err != nil {
			continue
		}
		topics = append(topics, topic)
	}

	span.SetAttributes(attribute.Int("topics_count", len(topics)))
	// Ensure we always return a slice, not nil
	if topics == nil {
		topics = []string{}
	}
	return topics, nil
}

// GetGapAnalysis identifies areas with poor user performance (knowledge gaps)
func (s *LearningService) GetGapAnalysis(ctx context.Context, userID int) (result0 map[string]interface{}, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_gap_analysis",
		observability.AttributeUserID(userID),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Query to find areas where user has poor performance (low accuracy)
	query := `
		SELECT
			pm.topic,
			COUNT(*) as total_questions,
			ROUND((pm.correct_attempts * 100.0 / pm.total_attempts), 2) as accuracy_percentage
		FROM performance_metrics pm
		WHERE pm.user_id = $1
		AND pm.total_attempts >= 3
		AND (pm.correct_attempts * 100.0 / pm.total_attempts) < 70.0
		GROUP BY pm.topic, pm.correct_attempts, pm.total_attempts
		ORDER BY accuracy_percentage ASC
		LIMIT 10
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get gap analysis")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	gaps := make(map[string]interface{})
	for rows.Next() {
		var topic string
		var totalQuestions int
		var accuracyPercentage sql.NullFloat64

		if err := rows.Scan(&topic, &totalQuestions, &accuracyPercentage); err != nil {
			continue
		}

		gapInfo := map[string]interface{}{
			"topic":               topic,
			"total_questions":     totalQuestions,
			"accuracy_percentage": 0.0,
		}

		if accuracyPercentage.Valid {
			gapInfo["accuracy_percentage"] = accuracyPercentage.Float64
		}

		gaps[topic] = gapInfo
	}

	span.SetAttributes(attribute.Int("gaps_count", len(gaps)))
	return gaps, nil
}

// GetPriorityDistribution returns the distribution of priority scores by topic for a user
func (s *LearningService) GetPriorityDistribution(ctx context.Context, userID int) (result0 map[string]int, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_priority_distribution",
		observability.AttributeUserID(userID),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Query to get priority score distribution by topic
	query := `
		SELECT q.topic_category, COUNT(*) as question_count
		FROM questions q
		JOIN user_questions uq ON q.id = uq.question_id
		JOIN question_priority_scores qps ON q.id = qps.question_id AND qps.user_id = $1
		WHERE uq.user_id = $1
		AND q.topic_category IS NOT NULL
		AND q.topic_category != ''
		GROUP BY q.topic_category
		ORDER BY question_count DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get priority distribution")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	distribution := make(map[string]int)
	for rows.Next() {
		var topic string
		var count int
		if err := rows.Scan(&topic, &count); err != nil {
			continue
		}
		distribution[topic] = count
	}

	span.SetAttributes(attribute.Int("topics_count", len(distribution)))
	return distribution, nil
}

// GetUserQuestionConfidenceLevel retrieves the confidence level for a specific question and user
func (s *LearningService) GetUserQuestionConfidenceLevel(ctx context.Context, userID, questionID int) (result0 *int, err error) {
	ctx, span := observability.TraceLearningFunction(ctx, "get_user_question_confidence_level",
		observability.AttributeUserID(userID),
		observability.AttributeQuestionID(questionID),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT confidence_level
		FROM user_question_metadata
		WHERE user_id = $1 AND question_id = $2
	`

	var confidenceLevel sql.NullInt32
	err = s.db.QueryRowContext(ctx, query, userID, questionID).Scan(&confidenceLevel)
	if err != nil {
		if err == sql.ErrNoRows {
			// No confidence level recorded for this user-question pair
			return nil, nil
		}
		return nil, contextutils.WrapError(err, "failed to get user question confidence level")
	}

	if confidenceLevel.Valid {
		level := int(confidenceLevel.Int32)
		return &level, nil
	}

	return nil, nil
}
