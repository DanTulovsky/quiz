package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// QuestionServiceInterface defines the interface for question-related operations.
// This allows for easier mocking in tests.
type QuestionServiceInterface interface {
	SaveQuestion(ctx context.Context, question *models.Question) error
	AssignQuestionToUser(ctx context.Context, questionID, userID int) error
	GetQuestionByID(ctx context.Context, id int) (*models.Question, error)
	GetQuestionWithStats(ctx context.Context, id int) (*QuestionWithStats, error)
	GetQuestionsByFilter(ctx context.Context, userID int, language, level string, questionType models.QuestionType, limit int) ([]models.Question, error)
	GetNextQuestion(ctx context.Context, userID int, language, level string, qType models.QuestionType) (*QuestionWithStats, error)
	GetAdaptiveQuestionsForDaily(ctx context.Context, userID int, language, level string, limit int) ([]*QuestionWithStats, error)
	ReportQuestion(ctx context.Context, questionID, userID int, reportReason string) error
	GetQuestionStats(ctx context.Context) (map[string]interface{}, error)
	GetDetailedQuestionStats(ctx context.Context) (map[string]interface{}, error)
	GetRecentQuestionContentsForUser(ctx context.Context, userID, limit int) ([]string, error)
	GetReportedQuestions(ctx context.Context) ([]*ReportedQuestionWithUser, error)
	MarkQuestionAsFixed(ctx context.Context, questionID int) error
	UpdateQuestion(ctx context.Context, questionID int, content map[string]interface{}, correctAnswerIndex int, explanation string) error
	DeleteQuestion(ctx context.Context, questionID int) error
	GetUserQuestions(ctx context.Context, userID, limit int) ([]*models.Question, error)
	GetUserQuestionsWithStats(ctx context.Context, userID, limit int) ([]*QuestionWithStats, error)
	GetQuestionsPaginated(ctx context.Context, userID, page, pageSize int, search, typeFilter, statusFilter string) ([]*QuestionWithStats, int, error)
	GetAllQuestionsPaginated(ctx context.Context, page, pageSize int, search, typeFilter, statusFilter, languageFilter, levelFilter string, userID *int) ([]*QuestionWithStats, int, error)
	GetReportedQuestionsPaginated(ctx context.Context, page, pageSize int, search, typeFilter, languageFilter, levelFilter string) ([]*QuestionWithStats, int, error)
	GetReportedQuestionsStats(ctx context.Context) (map[string]interface{}, error)
	GetUserQuestionCount(ctx context.Context, userID int) (int, error)
	GetUserResponseCount(ctx context.Context, userID int) (int, error)
	GetRandomGlobalQuestionForUser(ctx context.Context, userID int, language, level string, qType models.QuestionType) (*QuestionWithStats, error)
	GetUsersForQuestion(ctx context.Context, questionID int) ([]*models.User, int, error)
	AssignUsersToQuestion(ctx context.Context, questionID int, userIDs []int) error
	UnassignUsersFromQuestion(ctx context.Context, questionID int, userIDs []int) error
	DB() *sql.DB
}

// QuestionService provides methods for question management.
type QuestionService struct {
	db              *sql.DB
	learningService *LearningService
	logger          *observability.Logger
	cfg             *config.Config
}

// Shared query constants to eliminate duplication
const (
	// questionSelectFields contains all question fields for SELECT queries
	questionSelectFields = `id, type, language, level, difficulty_score, content, correct_answer, explanation, created_at, status, topic_category, grammar_focus, vocabulary_domain, scenario, style_modifier, difficulty_modifier, time_context`
)

// scanQuestionFromRow scans a database row into a models.Question struct
func (s *QuestionService) scanQuestionFromRow(row *sql.Row) (result0 *models.Question, err error) {
	question := &models.Question{}
	var contentJSON string
	var topicCategory sql.NullString
	var grammarFocus sql.NullString
	var vocabularyDomain sql.NullString
	var scenario sql.NullString
	var styleModifier sql.NullString
	var difficultyModifier sql.NullString
	var timeContext sql.NullString

	err = row.Scan(
		&question.ID,
		&question.Type,
		&question.Language,
		&question.Level,
		&question.DifficultyScore,
		&contentJSON,
		&question.CorrectAnswer,
		&question.Explanation,
		&question.CreatedAt,
		&question.Status,
		&topicCategory,
		&grammarFocus,
		&vocabularyDomain,
		&scenario,
		&styleModifier,
		&difficultyModifier,
		&timeContext,
	)
	if err != nil {
		return nil, err
	}

	// Set optional string fields if they have values
	if topicCategory.Valid {
		question.TopicCategory = topicCategory.String
	}
	if grammarFocus.Valid {
		question.GrammarFocus = grammarFocus.String
	}
	if vocabularyDomain.Valid {
		question.VocabularyDomain = vocabularyDomain.String
	}
	if scenario.Valid {
		question.Scenario = scenario.String
	}
	if styleModifier.Valid {
		question.StyleModifier = styleModifier.String
	}
	if difficultyModifier.Valid {
		question.DifficultyModifier = difficultyModifier.String
	}
	if timeContext.Valid {
		question.TimeContext = timeContext.String
	}

	if err := question.UnmarshalContentFromJSON(contentJSON); err != nil {
		return nil, err
	}

	return question, nil
}

// scanQuestionFromRows scans a database rows into a models.Question struct
func (s *QuestionService) scanQuestionFromRows(rows *sql.Rows) (result0 *models.Question, err error) {
	question := &models.Question{}
	var contentJSON string
	var topicCategory sql.NullString
	var grammarFocus sql.NullString
	var vocabularyDomain sql.NullString
	var scenario sql.NullString
	var styleModifier sql.NullString
	var difficultyModifier sql.NullString
	var timeContext sql.NullString

	err = rows.Scan(
		&question.ID,
		&question.Type,
		&question.Language,
		&question.Level,
		&question.DifficultyScore,
		&contentJSON,
		&question.CorrectAnswer,
		&question.Explanation,
		&question.CreatedAt,
		&question.Status,
		&topicCategory,
		&grammarFocus,
		&vocabularyDomain,
		&scenario,
		&styleModifier,
		&difficultyModifier,
		&timeContext,
	)
	if err != nil {
		return nil, err
	}

	// Set optional string fields if they have values
	if topicCategory.Valid {
		question.TopicCategory = topicCategory.String
	}
	if grammarFocus.Valid {
		question.GrammarFocus = grammarFocus.String
	}
	if vocabularyDomain.Valid {
		question.VocabularyDomain = vocabularyDomain.String
	}
	if scenario.Valid {
		question.Scenario = scenario.String
	}
	if styleModifier.Valid {
		question.StyleModifier = styleModifier.String
	}
	if difficultyModifier.Valid {
		question.DifficultyModifier = difficultyModifier.String
	}
	if timeContext.Valid {
		question.TimeContext = timeContext.String
	}

	if err := question.UnmarshalContentFromJSON(contentJSON); err != nil {
		return nil, err
	}

	return question, nil
}

// scanQuestionBasicFromRows scans a database rows into a models.Question struct (basic fields only)
func (s *QuestionService) scanQuestionBasicFromRows(rows *sql.Rows) (result0 *models.Question, err error) {
	question := &models.Question{}
	var contentJSON string

	err = rows.Scan(
		&question.ID,
		&question.Type,
		&question.Language,
		&question.Level,
		&question.DifficultyScore,
		&contentJSON,
		&question.CorrectAnswer,
		&question.Explanation,
		&question.CreatedAt,
		&question.Status,
	)
	if err != nil {
		return nil, err
	}

	if err := question.UnmarshalContentFromJSON(contentJSON); err != nil {
		return nil, err
	}

	return question, nil
}

// scanQuestionWithStatsFromRows scans a database rows into a QuestionWithStats struct
func (s *QuestionService) scanQuestionWithStatsFromRows(rows *sql.Rows) (result0 *QuestionWithStats, err error) {
	questionWithStats := &QuestionWithStats{
		Question: &models.Question{},
	}
	var contentJSON string

	err = rows.Scan(
		&questionWithStats.ID,
		&questionWithStats.Type,
		&questionWithStats.Language,
		&questionWithStats.Level,
		&questionWithStats.DifficultyScore,
		&contentJSON,
		&questionWithStats.CorrectAnswer,
		&questionWithStats.Explanation,
		&questionWithStats.CreatedAt,
		&questionWithStats.Status,
		&questionWithStats.CorrectCount,
		&questionWithStats.IncorrectCount,
		&questionWithStats.TotalResponses,
		&questionWithStats.UserCount,
	)
	if err != nil {
		return nil, err
	}

	if err := questionWithStats.UnmarshalContentFromJSON(contentJSON); err != nil {
		return nil, err
	}

	return questionWithStats, nil
}

// scanQuestionWithStatsAndAllFieldsFromRows scans a database rows into a QuestionWithStats struct (with all fields)
func (s *QuestionService) scanQuestionWithStatsAndAllFieldsFromRows(rows *sql.Rows) (result0 *QuestionWithStats, err error) {
	questionWithStats := &QuestionWithStats{
		Question: &models.Question{},
	}
	var contentJSON string
	var topicCategory sql.NullString
	var grammarFocus sql.NullString
	var vocabularyDomain sql.NullString
	var scenario sql.NullString
	var styleModifier sql.NullString
	var difficultyModifier sql.NullString
	var timeContext sql.NullString

	err = rows.Scan(
		&questionWithStats.ID,
		&questionWithStats.Type,
		&questionWithStats.Language,
		&questionWithStats.Level,
		&questionWithStats.DifficultyScore,
		&contentJSON,
		&questionWithStats.CorrectAnswer,
		&questionWithStats.Explanation,
		&questionWithStats.CreatedAt,
		&questionWithStats.Status,
		&topicCategory,
		&grammarFocus,
		&vocabularyDomain,
		&scenario,
		&styleModifier,
		&difficultyModifier,
		&timeContext,
		&questionWithStats.CorrectCount,
		&questionWithStats.IncorrectCount,
		&questionWithStats.TotalResponses,
		&questionWithStats.UserCount,
	)
	if err != nil {
		return nil, err
	}

	// Set optional string fields if they have values
	if topicCategory.Valid {
		questionWithStats.TopicCategory = topicCategory.String
	}
	if grammarFocus.Valid {
		questionWithStats.GrammarFocus = grammarFocus.String
	}
	if vocabularyDomain.Valid {
		questionWithStats.VocabularyDomain = vocabularyDomain.String
	}
	if scenario.Valid {
		questionWithStats.Scenario = scenario.String
	}
	if styleModifier.Valid {
		questionWithStats.StyleModifier = styleModifier.String
	}
	if difficultyModifier.Valid {
		questionWithStats.DifficultyModifier = difficultyModifier.String
	}
	if timeContext.Valid {
		questionWithStats.TimeContext = timeContext.String
	}

	if err := questionWithStats.UnmarshalContentFromJSON(contentJSON); err != nil {
		return nil, err
	}

	return questionWithStats, nil
}

// scanQuestionWithPriorityAndStatsFromRows scans a database rows into a QuestionWithStats struct (with priority and stats)
func (s *QuestionService) scanQuestionWithPriorityAndStatsFromRows(rows *sql.Rows) (result0 *QuestionWithStats, err error) {
	questionWithStats := &QuestionWithStats{
		Question: &models.Question{},
	}
	var contentJSON string
	var priorityScore float64
	var timesAnswered int
	var lastAnsweredAt sql.NullTime
	var confidenceLevel sql.NullInt32
	var topicCategory sql.NullString
	var grammarFocus sql.NullString
	var vocabularyDomain sql.NullString
	var scenario sql.NullString
	var styleModifier sql.NullString
	var difficultyModifier sql.NullString
	var timeContext sql.NullString

	err = rows.Scan(
		&questionWithStats.ID,
		&questionWithStats.Type,
		&questionWithStats.Language,
		&questionWithStats.Level,
		&questionWithStats.DifficultyScore,
		&contentJSON,
		&questionWithStats.CorrectAnswer,
		&questionWithStats.Explanation,
		&questionWithStats.CreatedAt,
		&questionWithStats.Status,
		&topicCategory,
		&grammarFocus,
		&vocabularyDomain,
		&scenario,
		&styleModifier,
		&difficultyModifier,
		&timeContext,
		&priorityScore,
		&timesAnswered,
		&lastAnsweredAt,
		&questionWithStats.CorrectCount,
		&questionWithStats.IncorrectCount,
		&questionWithStats.TotalResponses,
		&confidenceLevel,
	)
	if err != nil {
		return nil, err
	}

	// Set optional string fields if they have values
	if topicCategory.Valid {
		questionWithStats.TopicCategory = topicCategory.String
	}
	if grammarFocus.Valid {
		questionWithStats.GrammarFocus = grammarFocus.String
	}
	if vocabularyDomain.Valid {
		questionWithStats.VocabularyDomain = vocabularyDomain.String
	}
	if scenario.Valid {
		questionWithStats.Scenario = scenario.String
	}
	if styleModifier.Valid {
		questionWithStats.StyleModifier = styleModifier.String
	}
	if difficultyModifier.Valid {
		questionWithStats.DifficultyModifier = difficultyModifier.String
	}
	if timeContext.Valid {
		questionWithStats.TimeContext = timeContext.String
	}

	if err := questionWithStats.UnmarshalContentFromJSON(contentJSON); err != nil {
		return nil, err
	}

	// Set confidence level if it exists
	if confidenceLevel.Valid {
		level := int(confidenceLevel.Int32)
		questionWithStats.ConfidenceLevel = &level
	}

	// Populate per-user times answered from the scanned value
	questionWithStats.TimesAnswered = timesAnswered

	return questionWithStats, nil
}

// scanQuestionWithStatsAndReportersFromRows scans a database rows into a QuestionWithStats struct (with reporter information)
func (s *QuestionService) scanQuestionWithStatsAndReportersFromRows(rows *sql.Rows) (result0 *QuestionWithStats, err error) {
	questionWithStats := &QuestionWithStats{
		Question: &models.Question{},
	}
	var contentJSON string
	var reporters sql.NullString
	var reportReasons sql.NullString
	var topicCategory sql.NullString
	var grammarFocus sql.NullString
	var vocabularyDomain sql.NullString
	var scenario sql.NullString
	var styleModifier sql.NullString
	var difficultyModifier sql.NullString
	var timeContext sql.NullString

	err = rows.Scan(
		&questionWithStats.ID,
		&questionWithStats.Type,
		&questionWithStats.Language,
		&questionWithStats.Level,
		&questionWithStats.DifficultyScore,
		&contentJSON,
		&questionWithStats.CorrectAnswer,
		&questionWithStats.Explanation,
		&questionWithStats.CreatedAt,
		&questionWithStats.Status,
		&topicCategory,
		&grammarFocus,
		&vocabularyDomain,
		&scenario,
		&styleModifier,
		&difficultyModifier,
		&timeContext,
		&questionWithStats.CorrectCount,
		&questionWithStats.IncorrectCount,
		&questionWithStats.TotalResponses,
		&reporters,
		&reportReasons,
	)
	if err != nil {
		return nil, err
	}

	// Set optional string fields if they have values
	if topicCategory.Valid {
		questionWithStats.TopicCategory = topicCategory.String
	}
	if grammarFocus.Valid {
		questionWithStats.GrammarFocus = grammarFocus.String
	}
	if vocabularyDomain.Valid {
		questionWithStats.VocabularyDomain = vocabularyDomain.String
	}
	if scenario.Valid {
		questionWithStats.Scenario = scenario.String
	}
	if styleModifier.Valid {
		questionWithStats.StyleModifier = styleModifier.String
	}
	if difficultyModifier.Valid {
		questionWithStats.DifficultyModifier = difficultyModifier.String
	}
	if timeContext.Valid {
		questionWithStats.TimeContext = timeContext.String
	}

	if err := questionWithStats.UnmarshalContentFromJSON(contentJSON); err != nil {
		return nil, err
	}

	// Store reporter information
	if reporters.Valid && reporters.String != "" {
		questionWithStats.Reporters = reporters.String
	}

	// Store report reasons information
	if reportReasons.Valid && reportReasons.String != "" {
		questionWithStats.ReportReasons = reportReasons.String
	}

	return questionWithStats, nil
}

// getQuestionByQuery is a shared method for getting a question by any query
func (s *QuestionService) getQuestionByQuery(ctx context.Context, query string, args ...interface{}) (result0 *models.Question, err error) {
	row := s.db.QueryRowContext(ctx, query, args...)
	var question *models.Question
	question, err = s.scanQuestionFromRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows // Propagate sql.ErrNoRows for not found
		}
		return nil, err
	}
	return question, nil
}

// NewQuestionServiceWithLogger creates a new QuestionService instance with logger
func NewQuestionServiceWithLogger(db *sql.DB, learningService *LearningService, cfg *config.Config, logger *observability.Logger) *QuestionService {
	if db == nil {
		panic("database connection cannot be nil")
	}
	if logger == nil {
		panic("logger cannot be nil")
	}

	return &QuestionService{
		db:              db,
		learningService: learningService,
		logger:          logger,
		cfg:             cfg,
	}
}

// getDailyRepeatAvoidDays returns the configured number of days to avoid repeating
// questions in daily assignments. Defaults to 7 when not configured or invalid.
func (s *QuestionService) getDailyRepeatAvoidDays() int {
	if s.cfg != nil {
		if days := s.cfg.Server.DailyRepeatAvoidDays; days > 0 {
			return days
		}
	}
	return 7
}

// SaveQuestion saves a question to the database
func (s *QuestionService) SaveQuestion(ctx context.Context, question *models.Question) (err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "save_question", observability.AttributeQuestion(question))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Validate question content before saving using shared validation helper
	if err := contextutils.ValidateQuestionContent(question.Content, question.ID); err != nil {
		return err
	}

	// Make a deep copy of content before marshaling to avoid modifying the original
	// MarshalContentToJSON modifies the content map in place (removes correct_answer, explanation)
	var contentCopy map[string]interface{}
	if question.Content != nil {
		contentCopy = make(map[string]interface{})
		for k, v := range question.Content {
			// Deep copy slices to avoid sharing references
			if slice, ok := v.([]interface{}); ok {
				sliceCopy := make([]interface{}, len(slice))
				copy(sliceCopy, slice)
				contentCopy[k] = sliceCopy
			} else if slice, ok := v.([]string); ok {
				sliceCopy := make([]string, len(slice))
				copy(sliceCopy, slice)
				contentCopy[k] = sliceCopy
			} else {
				contentCopy[k] = v
			}
		}
	}

	var contentJSON []byte
	if contentCopy != nil {
		// Temporarily set Content to the copy for marshaling
		originalContent := question.Content
		question.Content = contentCopy
		contentJSONStr, err := question.MarshalContentToJSON()
		question.Content = originalContent // Restore original
		if err != nil {
			return contextutils.WrapError(err, "failed to marshal question content")
		}
		contentJSON = []byte(contentJSONStr)
	} else {
		contentJSON = []byte("{}")
	}

	if question.Status == "" {
		question.Status = models.QuestionStatusActive
	}

	query := `
		INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation, status, topic_category, grammar_focus, vocabulary_domain, scenario, style_modifier, difficulty_modifier, time_context)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING id
	`

	var id int
	err = s.db.QueryRowContext(ctx, query,
		question.Type,
		question.Language,
		question.Level,
		question.DifficultyScore,
		string(contentJSON),
		question.CorrectAnswer,
		question.Explanation,
		question.Status,
		question.TopicCategory,
		question.GrammarFocus,
		question.VocabularyDomain,
		question.Scenario,
		question.StyleModifier,
		question.DifficultyModifier,
		question.TimeContext,
	).Scan(&id)
	if err != nil {
		return contextutils.WrapError(err, "failed to save question to database")
	}

	question.ID = id
	return nil
}

// AssignQuestionToUser assigns a question to a user
func (s *QuestionService) AssignQuestionToUser(ctx context.Context, questionID, userID int) (err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "assign_question_to_user", observability.AttributeQuestionID(questionID), observability.AttributeUserID(userID))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	query := `
		INSERT INTO user_questions (user_id, question_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, question_id) DO NOTHING
	`
	_, err = s.db.ExecContext(ctx, query, userID, questionID)
	return contextutils.WrapError(err, "failed to assign question to user")
}

// GetQuestionByID retrieves a question by its ID
func (s *QuestionService) GetQuestionByID(ctx context.Context, id int) (result0 *models.Question, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_question_by_id", observability.AttributeQuestionID(id))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	query := fmt.Sprintf("SELECT %s FROM questions WHERE id = $1", questionSelectFields)
	return s.getQuestionByQuery(ctx, query, id)
}

// GetQuestionWithStats retrieves a question by its ID with response statistics
func (s *QuestionService) GetQuestionWithStats(ctx context.Context, id int) (result0 *QuestionWithStats, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_question_with_stats", observability.AttributeQuestionID(id))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	query := `
		SELECT
			q.id, q.type, q.language, q.level, q.difficulty_score,
			q.content, q.correct_answer, q.explanation, q.created_at, q.status,
			q.topic_category, q.grammar_focus, q.vocabulary_domain, q.scenario, q.style_modifier, q.difficulty_modifier, q.time_context,
			COALESCE(SUM(CASE WHEN ur.is_correct = true THEN 1 ELSE 0 END), 0) as correct_count,
			COALESCE(SUM(CASE WHEN ur.is_correct = false THEN 1 ELSE 0 END), 0) as incorrect_count,
			COALESCE(COUNT(ur.id), 0) as total_responses
		FROM questions q
		LEFT JOIN user_responses ur ON q.id = ur.question_id
		WHERE q.id = $1
		GROUP BY q.id, q.type, q.language, q.level, q.difficulty_score,
				 q.content, q.correct_answer, q.explanation, q.created_at, q.status,
				 q.topic_category, q.grammar_focus, q.vocabulary_domain, q.scenario, q.style_modifier, q.difficulty_modifier, q.time_context
	`

	q := &models.Question{}
	stats := &QuestionWithStats{Question: q}

	var contentJSON string
	err = s.db.QueryRowContext(ctx, query, id).Scan(
		&q.ID, &q.Type, &q.Language, &q.Level, &q.DifficultyScore,
		&contentJSON, &q.CorrectAnswer, &q.Explanation, &q.CreatedAt, &q.Status,
		&q.TopicCategory, &q.GrammarFocus, &q.VocabularyDomain, &q.Scenario, &q.StyleModifier, &q.DifficultyModifier, &q.TimeContext,
		&stats.CorrectCount, &stats.IncorrectCount, &stats.TotalResponses,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contextutils.ErrQuestionNotFound
		}
		return nil, contextutils.WrapError(err, "failed to get question with stats")
	}

	// Parse JSON content
	if err := q.UnmarshalContentFromJSON(contentJSON); err != nil {
		return nil, contextutils.WrapError(err, "failed to unmarshal question content")
	}

	return stats, nil
}

// GetQuestionsByFilter retrieves questions matching the specified criteria
func (s *QuestionService) GetQuestionsByFilter(ctx context.Context, userID int, language, level string, questionType models.QuestionType, limit int) (result0 []models.Question, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_questions_by_filter", observability.AttributeUserID(userID), observability.AttributeLanguage(language), observability.AttributeLevel(level), observability.AttributeQuestionType(questionType))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	var query string
	var args []interface{}

	if questionType == "" {
		// Don't filter by type if questionType is empty
		query = `
			SELECT q.id, q.type, q.language, q.level, q.difficulty_score, q.content, q.correct_answer, q.explanation, q.created_at, q.status
			FROM questions q
			JOIN user_questions uq ON q.id = uq.question_id
			WHERE uq.user_id = $1 AND q.language = $2 AND q.level = $3 AND q.status = $4
			ORDER BY RANDOM()
			LIMIT $5
		`
		args = []interface{}{userID, language, level, models.QuestionStatusActive, limit}
	} else {
		// Filter by specific type
		query = `
			SELECT q.id, q.type, q.language, q.level, q.difficulty_score, q.content, q.correct_answer, q.explanation, q.created_at, q.status
			FROM questions q
			JOIN user_questions uq ON q.id = uq.question_id
			WHERE uq.user_id = $1 AND q.language = $2 AND q.level = $3 AND q.type = $4 AND q.status = $5
			ORDER BY RANDOM()
			LIMIT $6
		`
		args = []interface{}{userID, language, level, questionType, models.QuestionStatusActive, limit}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query questions by filter")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var questions []models.Question
	for rows.Next() {
		question, err := s.scanQuestionBasicFromRows(rows)
		if err != nil {
			return nil, contextutils.WrapError(err, "failed to scan question from rows")
		}
		questions = append(questions, *question)
	}

	return questions, nil
}

// ReportedQuestionWithUser represents a reported question with user information
type ReportedQuestionWithUser struct {
	*models.Question
	ReportedByUsername string `json:"reported_by_username"`
	TotalResponses     int    `json:"total_responses"`
}

// GetReportedQuestions retrieves all questions that have been reported as problematic
func (s *QuestionService) GetReportedQuestions(ctx context.Context) (result0 []*ReportedQuestionWithUser, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_reported_questions")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	query := `
		SELECT q.id, q.type, q.language, q.level, q.difficulty_score, q.content, q.correct_answer, q.explanation, q.created_at, q.status, u.username,
		       COALESCE(COUNT(ur.id), 0) as total_responses
		FROM questions q
		LEFT JOIN user_questions uq ON q.id = uq.question_id
		LEFT JOIN users u ON uq.user_id = u.id
		LEFT JOIN user_responses ur ON q.id = ur.question_id
		WHERE q.status = $1
		GROUP BY q.id, q.type, q.language, q.level, q.difficulty_score, q.content, q.correct_answer, q.explanation, q.created_at, q.status, u.username
		ORDER BY q.created_at DESC
	`

	var rows *sql.Rows
	rows, err = s.db.QueryContext(ctx, query, models.QuestionStatusReported)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query reported questions")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var questions []*ReportedQuestionWithUser
	for rows.Next() {
		var question models.Question
		var reportedByUsername sql.NullString
		var contentJSON string
		var totalResponses int

		err = rows.Scan(
			&question.ID,
			&question.Type,
			&question.Language,
			&question.Level,
			&question.DifficultyScore,
			&contentJSON,
			&question.CorrectAnswer,
			&question.Explanation,
			&question.CreatedAt,
			&question.Status,
			&reportedByUsername,
			&totalResponses,
		)
		if err != nil {
			return nil, err
		}

		if err := question.UnmarshalContentFromJSON(contentJSON); err != nil {
			return nil, err
		}

		username := ""
		if reportedByUsername.Valid {
			username = reportedByUsername.String
		}

		reportedQuestion := &ReportedQuestionWithUser{
			Question:           &question,
			ReportedByUsername: username,
			TotalResponses:     totalResponses,
		}

		questions = append(questions, reportedQuestion)
	}

	return questions, nil
}

// MarkQuestionAsFixed marks a reported question as fixed and puts it back in rotation
func (s *QuestionService) MarkQuestionAsFixed(ctx context.Context, questionID int) (err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "mark_question_as_fixed", observability.AttributeQuestionID(questionID))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `UPDATE questions SET status = $1 WHERE id = $2`
	var result sql.Result
	result, err = s.db.ExecContext(ctx, query, models.QuestionStatusActive, questionID)
	if err != nil {
		return contextutils.WrapError(err, "failed to mark question as fixed")
	}

	// Check if the question was actually updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.WrapErrorf(contextutils.ErrRecordNotFound, "question with ID %d not found", questionID)
	}

	return nil
}

// UpdateQuestion updates a question's content, correct answer, and explanation
func (s *QuestionService) UpdateQuestion(ctx context.Context, questionID int, content map[string]interface{}, correctAnswerIndex int, explanation string) (err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "update_question", observability.AttributeQuestionID(questionID))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Validate question content before updating using shared validation helper
	if err := contextutils.ValidateQuestionContent(content, questionID); err != nil {
		return err
	}

	var contentJSON []byte
	// Marshal provided content map via a temporary Question instance to reuse method
	// Note: MarshalContentToJSON modifies the content map in place, but since this is a request
	// payload that won't be reused, that's acceptable here
	tempQ := &models.Question{Content: content}
	contentJSONStr, err := tempQ.MarshalContentToJSON()
	if err != nil {
		return contextutils.WrapError(err, "failed to marshal content JSON")
	}
	contentJSON = []byte(contentJSONStr)

	query := `UPDATE questions SET content = $1, correct_answer = $2, explanation = $3 WHERE id = $4`
	var result sql.Result
	result, err = s.db.ExecContext(ctx, query, string(contentJSON), correctAnswerIndex, explanation, questionID)
	if err != nil {
		return contextutils.WrapError(err, "failed to update question")
	}

	// Check if the question was actually updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.WrapErrorf(contextutils.ErrRecordNotFound, "question with ID %d not found", questionID)
	}

	return nil
}

// DeleteQuestion permanently deletes a question from the database
func (s *QuestionService) DeleteQuestion(ctx context.Context, questionID int) (err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "delete_question", observability.AttributeQuestionID(questionID))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	// First, delete associated user responses
	deleteResponsesQuery := `DELETE FROM user_responses WHERE question_id = $1`
	_, err = s.db.ExecContext(ctx, deleteResponsesQuery, questionID)
	if err != nil {
		return contextutils.WrapError(err, "failed to delete associated user responses")
	}

	// Then delete the question itself
	deleteQuestionQuery := `DELETE FROM questions WHERE id = $1`
	var result sql.Result
	result, err = s.db.ExecContext(ctx, deleteQuestionQuery, questionID)
	if err != nil {
		return contextutils.WrapError(err, "failed to delete question")
	}

	// Check if the question was actually deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.WrapErrorf(contextutils.ErrRecordNotFound, "question with ID %d not found", questionID)
	}

	return nil
}

// ReportQuestion marks a question as reported/problematic by a specific user
func (s *QuestionService) ReportQuestion(ctx context.Context, questionID, userID int, reportReason string) (err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "report_question", observability.AttributeQuestionID(questionID), observability.AttributeUserID(userID))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Start a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contextutils.WrapError(err, "failed to begin transaction")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Warn(ctx, "Failed to rollback transaction", map[string]interface{}{"error": rollbackErr.Error()})
			}
		}
	}()

	// Check if question exists first
	var questionExists bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM questions WHERE id = $1)`, questionID).Scan(&questionExists)
	if err != nil {
		return contextutils.WrapError(err, "failed to check if question exists")
	}
	if !questionExists {
		return contextutils.WrapErrorf(contextutils.ErrRecordNotFound, "question with id %d not found", questionID)
	}

	// Update question status to reported
	updateQuery := `UPDATE questions SET status = $1 WHERE id = $2`
	var result sql.Result
	result, err = tx.ExecContext(ctx, updateQuery, models.QuestionStatusReported, questionID)
	if err != nil {
		return contextutils.WrapError(err, "failed to update question status")
	}

	// Check if the question was actually updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.WrapErrorf(contextutils.ErrRecordNotFound, "question with ID %d not found", questionID)
	}

	// Use provided report reason or default message
	reason := reportReason
	if reason == "" {
		reason = "Question reported by user"
	}

	// Create or update a report record: if the same user reports the same question again,
	// update the report_reason to the new value instead of doing nothing. Also update created_at
	// so admin views show the time of the latest report by that user.
	reportQuery := `INSERT INTO question_reports (question_id, reported_by_user_id, report_reason) VALUES ($1, $2, $3) ON CONFLICT (question_id, reported_by_user_id) DO UPDATE SET report_reason = EXCLUDED.report_reason, created_at = now()`
	_, err = tx.ExecContext(ctx, reportQuery, questionID, userID, reason)
	if err != nil {
		return contextutils.WrapError(err, "failed to create question report")
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return contextutils.WrapError(err, "failed to commit transaction")
	}

	return nil
}

// GetNextQuestion gets the next question for a user based on usage count and availability
func (s *QuestionService) GetNextQuestion(ctx context.Context, userID int, language, level string, qType models.QuestionType) (result0 *QuestionWithStats, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_next_question", observability.AttributeUserID(userID), observability.AttributeLanguage(language), observability.AttributeLevel(level), observability.AttributeQuestionType(qType))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	// Use priority-based selection with stats included
	return s.getNextQuestionWithPriority(ctx, userID, language, level, qType)
}

// getNextQuestionWithPriority implements priority-based question selection with stats
func (s *QuestionService) getNextQuestionWithPriority(ctx context.Context, userID int, language, level string, qType models.QuestionType) (result0 *QuestionWithStats, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_next_question_with_priority", observability.AttributeUserID(userID), observability.AttributeLanguage(language), observability.AttributeLevel(level), observability.AttributeQuestionType(qType))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	// Get user preferences
	var prefs *models.UserLearningPreferences
	prefs, err = s.learningService.GetUserLearningPreferences(ctx, userID)
	if err != nil {
		s.logger.Warn(ctx, "Failed to get user preferences", map[string]interface{}{"user_id": userID, "error": err.Error()})
		// Fall back to default preferences
		prefs = s.learningService.GetDefaultLearningPreferences()
	}

	// Get available questions with priority scores and stats
	var questions []*QuestionWithStats
	questions, err = s.getAvailableQuestionsWithPriority(ctx, userID, language, level, qType, prefs)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get available questions")
	}

	if len(questions) == 0 {
		// Fallback: try to get a random global question and assign it to the user
		globalQ, err := s.GetRandomGlobalQuestionForUser(ctx, userID, language, level, qType)
		if err != nil {
			return nil, contextutils.WrapError(err, "no personalized questions, and failed to get global fallback question")
		}
		if globalQ != nil {
			return globalQ, nil
		}
		return nil, nil // No questions available at all
	}

	// Apply FreshQuestionRatio logic (NEW)
	selectedQuestion, err := s.selectQuestionWithFreshnessRatio(questions, prefs.FreshQuestionRatio)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to select question with freshness ratio")
	}

	// Return the selected question with stats (already included)
	return selectedQuestion, nil
}

// GetAdaptiveQuestionsForDaily selects multiple adaptive questions for daily assignments
func (s *QuestionService) GetAdaptiveQuestionsForDaily(ctx context.Context, userID int, language, level string, limit int) (result0 []*QuestionWithStats, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_adaptive_questions_for_daily")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Get user learning preferences
	prefs, err := s.learningService.GetUserLearningPreferences(ctx, userID)
	if err != nil {
		s.logger.Warn(ctx, "Failed to get user learning preferences, using defaults", map[string]interface{}{
			"user_id": userID, "error": err.Error(),
		})
		prefs = &models.UserLearningPreferences{
			FreshQuestionRatio: 0.7,
		}
	}

	var selectedQuestions []*QuestionWithStats
	selectedQuestionIDs := make(map[int]bool) // Track selected question IDs to prevent duplicates

	// Select questions across different types to provide variety
	questionTypes := []models.QuestionType{models.Vocabulary, models.FillInBlank, models.QuestionAnswer, models.ReadingComprehension}

	// Calculate how many questions to select from each type
	questionsPerType := limit / len(questionTypes)
	remainingQuestions := limit % len(questionTypes)

	for i, qType := range questionTypes {
		// Calculate how many questions to get for this type
		currentLimit := questionsPerType
		if i < remainingQuestions {
			currentLimit++ // Distribute remaining questions evenly
		}

		if currentLimit == 0 {
			continue
		}

		// Get available questions for DAILY with 2-day recent-correct exclusion
		questions, err := s.getAvailableQuestionsForDailyWithPriority(ctx, userID, language, level, qType, prefs)
		if err != nil {
			s.logger.Warn(ctx, "Failed to get questions for type", map[string]interface{}{
				"user_id": userID, "type": qType, "error": err.Error(),
			})
			continue
		}

		// Filter out questions that have already been selected
		var availableQuestions []*QuestionWithStats
		for _, q := range questions {
			if !selectedQuestionIDs[q.ID] {
				availableQuestions = append(availableQuestions, q)
			}
		}

		if len(availableQuestions) == 0 {
			// Try to get a global fallback question for this type
			globalQ, err := s.GetRandomGlobalQuestionForUser(ctx, userID, language, level, qType)
			if err != nil {
				s.logger.Warn(ctx, "Failed to get global fallback question", map[string]interface{}{
					"user_id": userID, "type": qType, "error": err.Error(),
				})
				continue
			}
			if globalQ != nil && !selectedQuestionIDs[globalQ.ID] {
				selectedQuestions = append(selectedQuestions, globalQ)
				selectedQuestionIDs[globalQ.ID] = true
				s.logger.Info(ctx, "Added global fallback question", map[string]interface{}{
					"user_id": userID, "type": qType, "question_id": globalQ.ID,
				})
			}
			continue
		}

		// Select questions for this type using adaptive selection
		s.logger.Info(ctx, "Starting selection for question type", map[string]interface{}{
			"user_id": userID, "type": qType, "current_limit": currentLimit, "available_questions": len(availableQuestions),
		})

		questionsSelected := 0
		remainingQuestionsForType := availableQuestions

		for j := 0; j < currentLimit && len(remainingQuestionsForType) > 0; j++ {
			// Apply freshness ratio logic for each selection
			selectedQuestion, err := s.selectQuestionWithFreshnessRatio(remainingQuestionsForType, prefs.FreshQuestionRatio)
			if err != nil {
				s.logger.Warn(ctx, "Failed to select question with freshness ratio", map[string]interface{}{
					"user_id": userID, "type": qType, "error": err.Error(),
				})
				// Fallback to simple random selection
				if len(remainingQuestionsForType) > 0 {
					selectedQuestion = remainingQuestionsForType[rand.Intn(len(remainingQuestionsForType))]
				} else {
					break
				}
			}

			if selectedQuestion != nil && !selectedQuestionIDs[selectedQuestion.ID] {
				selectedQuestions = append(selectedQuestions, selectedQuestion)
				selectedQuestionIDs[selectedQuestion.ID] = true
				questionsSelected++

				// Remove the selected question from the remaining pool
				var newRemainingQuestions []*QuestionWithStats
				for _, q := range remainingQuestionsForType {
					if q.ID != selectedQuestion.ID {
						newRemainingQuestions = append(newRemainingQuestions, q)
					}
				}
				remainingQuestionsForType = newRemainingQuestions

				s.logger.Info(ctx, "Successfully selected question", map[string]interface{}{
					"user_id": userID, "type": qType, "iteration": j, "question_id": selectedQuestion.ID,
					"total_selected": len(selectedQuestions),
				})
			} else {
				s.logger.Warn(ctx, "Failed to select question for type", map[string]interface{}{
					"user_id": userID, "type": qType, "iteration": j, "current_limit": currentLimit,
					"selected_question_nil": selectedQuestion == nil,
					"already_selected":      selectedQuestion != nil && selectedQuestionIDs[selectedQuestion.ID],
				})
				// Remove the question from the pool even if it was already selected
				if selectedQuestion != nil {
					var newRemainingQuestions []*QuestionWithStats
					for _, q := range remainingQuestionsForType {
						if q.ID != selectedQuestion.ID {
							newRemainingQuestions = append(newRemainingQuestions, q)
						}
					}
					remainingQuestionsForType = newRemainingQuestions
				}
			}
		}

		// If we didn't select enough questions for this type, try simple selection from all available questions
		if questionsSelected < currentLimit {
			s.logger.Info(ctx, "Using simple selection to fill remaining slots", map[string]interface{}{
				"user_id": userID, "type": qType, "questions_selected": questionsSelected, "current_limit": currentLimit,
			})

			// Get all questions for this type again and filter out already selected ones
			allQuestionsForType, err := s.getAvailableQuestionsForDailyWithPriority(ctx, userID, language, level, qType, prefs)
			if err == nil {
				for _, q := range allQuestionsForType {
					if !selectedQuestionIDs[q.ID] && questionsSelected < currentLimit {
						selectedQuestions = append(selectedQuestions, q)
						selectedQuestionIDs[q.ID] = true
						questionsSelected++
					}
				}
			}
		}

		s.logger.Info(ctx, "Completed selection for question type", map[string]interface{}{
			"user_id": userID, "type": qType, "questions_selected": questionsSelected, "target": currentLimit,
		})
	}

	// If we don't have enough questions, fill with random questions from any type
	if len(selectedQuestions) < limit {
		remainingNeeded := limit - len(selectedQuestions)
		s.logger.Info(ctx, "Not enough questions from type-based selection, using fallback", map[string]interface{}{
			"user_id": userID, "selected_count": len(selectedQuestions), "limit": limit, "remaining_needed": remainingNeeded,
		})

		// Get all available questions by trying each question type
		var allQuestions []*QuestionWithStats
		questionIDMap := make(map[int]bool) // Track seen question IDs to avoid duplicates

		for _, qType := range questionTypes {
			questions, err := s.getAvailableQuestionsForDailyWithPriority(ctx, userID, language, level, qType, prefs)
			if err == nil {
				for _, q := range questions {
					if !questionIDMap[q.ID] && !selectedQuestionIDs[q.ID] {
						allQuestions = append(allQuestions, q)
						questionIDMap[q.ID] = true
					}
				}
			}
		}

		s.logger.Info(ctx, "Fallback questions available", map[string]interface{}{
			"user_id": userID, "all_questions_count": len(allQuestions),
		})

		if len(allQuestions) > 0 {
			// Select random questions to fill the remaining slots
			for i := 0; i < remainingNeeded && i < len(allQuestions); i++ {
				selectedQuestion, err := s.selectQuestionWithFreshnessRatio(allQuestions, prefs.FreshQuestionRatio)
				if err != nil {
					s.logger.Warn(ctx, "Failed to select question with freshness ratio in fallback", map[string]interface{}{
						"user_id": userID, "error": err.Error(),
					})
					// Fallback to simple random selection
					if len(allQuestions) > 0 {
						selectedQuestion = allQuestions[rand.Intn(len(allQuestions))]
					} else {
						break
					}
				}

				if selectedQuestion != nil && !selectedQuestionIDs[selectedQuestion.ID] {
					selectedQuestions = append(selectedQuestions, selectedQuestion)
					selectedQuestionIDs[selectedQuestion.ID] = true

					// Remove the selected question from the pool
					var newAllQuestions []*QuestionWithStats
					for _, q := range allQuestions {
						if q.ID != selectedQuestion.ID {
							newAllQuestions = append(newAllQuestions, q)
						}
					}
					allQuestions = newAllQuestions
				} else if selectedQuestion != nil {
					// Remove the question from the pool even if it was already selected
					var newAllQuestions []*QuestionWithStats
					for _, q := range allQuestions {
						if q.ID != selectedQuestion.ID {
							newAllQuestions = append(newAllQuestions, q)
						}
					}
					allQuestions = newAllQuestions
				}
			}
		}
	}

	// Ensure we don't exceed the limit
	if len(selectedQuestions) > limit {
		selectedQuestions = selectedQuestions[:limit]
	}

	// Final duplicate check - this should never happen but provides extra safety
	finalSelectedQuestions := make([]*QuestionWithStats, 0, len(selectedQuestions))
	finalSelectedIDs := make(map[int]bool)

	for _, q := range selectedQuestions {
		if !finalSelectedIDs[q.ID] {
			finalSelectedQuestions = append(finalSelectedQuestions, q)
			finalSelectedIDs[q.ID] = true
		} else {
			s.logger.Warn(ctx, "Duplicate question detected in final selection", map[string]interface{}{
				"user_id": userID, "question_id": q.ID,
			})
		}
	}

	// Interleave selected questions by type to avoid bias toward types that were
	// selected earlier in the algorithm. This ensures that when callers slice the
	// returned list (e.g., to meet a smaller goal), later types like
	// ReadingComprehension are not systematically excluded.
	typeBuckets := make(map[models.QuestionType][]*QuestionWithStats)
	var typeOrder []models.QuestionType
	for _, q := range finalSelectedQuestions {
		if _, ok := typeBuckets[q.Type]; !ok {
			typeOrder = append(typeOrder, q.Type)
		}
		typeBuckets[q.Type] = append(typeBuckets[q.Type], q)
	}

	interleaved := make([]*QuestionWithStats, 0, len(finalSelectedQuestions))
	for len(interleaved) < len(finalSelectedQuestions) {
		added := false
		for _, t := range typeOrder {
			if len(typeBuckets[t]) > 0 {
				interleaved = append(interleaved, typeBuckets[t][0])
				typeBuckets[t] = typeBuckets[t][1:]
				added = true
				if len(interleaved) >= len(finalSelectedQuestions) {
					break
				}
			}
		}
		if !added {
			break
		}
	}
	finalSelectedQuestions = interleaved

	s.logger.Info(ctx, "Selected adaptive questions for daily assignment", map[string]interface{}{
		"user_id":            userID,
		"language":           language,
		"level":              level,
		"requested_limit":    limit,
		"selected_count":     len(finalSelectedQuestions),
		"duplicates_removed": len(selectedQuestions) - len(finalSelectedQuestions),
	})

	return finalSelectedQuestions, nil
}

// GetQuestionStats returns basic statistics about questions in the system
func (s *QuestionService) GetQuestionStats(ctx context.Context) (result0 map[string]interface{}, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_question_stats")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	stats := make(map[string]interface{})

	// Total questions
	var totalQuestions int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM questions").Scan(&totalQuestions)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get total questions count")
	}
	stats["total_questions"] = totalQuestions

	// Questions by type
	typeQuery := `
		SELECT type, COUNT(*) as count
		FROM questions
		GROUP BY type
	`
	rows, err := s.db.QueryContext(ctx, typeQuery)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query questions by type")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	questionsByType := make(map[string]int)
	for rows.Next() {
		var qType string
		var count int
		if err := rows.Scan(&qType, &count); err != nil {
			return nil, contextutils.WrapError(err, "failed to scan question type count")
		}
		questionsByType[qType] = count
	}
	stats["questions_by_type"] = questionsByType

	// Questions by level
	levelQuery := `
		SELECT level, COUNT(*) as count
		FROM questions
		GROUP BY level
	`
	rows, err = s.db.QueryContext(ctx, levelQuery)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query questions by level")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	questionsByLevel := make(map[string]int)
	for rows.Next() {
		var level string
		var count int
		if err := rows.Scan(&level, &count); err != nil {
			return nil, err
		}
		questionsByLevel[level] = count
	}
	stats["questions_by_level"] = questionsByLevel

	return stats, nil
}

// GetDetailedQuestionStats returns detailed statistics about questions
func (s *QuestionService) GetDetailedQuestionStats(ctx context.Context) (result0 map[string]interface{}, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_detailed_question_stats")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	stats := make(map[string]interface{})

	// Total questions
	var totalQuestions int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM questions").Scan(&totalQuestions)
	if err != nil {
		return nil, err
	}
	stats["total_questions"] = totalQuestions

	// Questions by language, level, and type combination
	detailQuery := `
		SELECT language, level, type, COUNT(*) as count
		FROM questions
		GROUP BY language, level, type
		ORDER BY language, level, type
	`
	rows, err := s.db.QueryContext(ctx, detailQuery)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	// Create nested structure: language -> level -> type -> count
	questionsByDetail := make(map[string]map[string]map[string]int)
	for rows.Next() {
		var language, level, qType string
		var count int
		if err := rows.Scan(&language, &level, &qType, &count); err != nil {
			return nil, err
		}

		if questionsByDetail[language] == nil {
			questionsByDetail[language] = make(map[string]map[string]int)
		}
		if questionsByDetail[language][level] == nil {
			questionsByDetail[language][level] = make(map[string]int)
		}
		questionsByDetail[language][level][qType] = count
	}
	stats["questions_by_detail"] = questionsByDetail

	// Questions by language
	languageQuery := `
		SELECT language, COUNT(*) as count
		FROM questions
		GROUP BY language
	`
	rows, err = s.db.QueryContext(ctx, languageQuery)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	questionsByLanguage := make(map[string]int)
	for rows.Next() {
		var language string
		var count int
		if err := rows.Scan(&language, &count); err != nil {
			return nil, err
		}
		questionsByLanguage[language] = count
	}
	stats["questions_by_language"] = questionsByLanguage

	// Questions by type
	typeQuery := `
		SELECT type, COUNT(*) as count
		FROM questions
		GROUP BY type
	`
	rows, err = s.db.QueryContext(ctx, typeQuery)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	questionsByType := make(map[string]int)
	for rows.Next() {
		var qType string
		var count int
		if err := rows.Scan(&qType, &count); err != nil {
			return nil, err
		}
		questionsByType[qType] = count
	}
	stats["questions_by_type"] = questionsByType

	// Questions by level
	levelQuery := `
		SELECT level, COUNT(*) as count
		FROM questions
		GROUP BY level
	`
	rows, err = s.db.QueryContext(ctx, levelQuery)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	questionsByLevel := make(map[string]int)
	for rows.Next() {
		var level string
		var count int
		if err := rows.Scan(&level, &count); err != nil {
			return nil, err
		}
		questionsByLevel[level] = count
	}
	stats["questions_by_level"] = questionsByLevel

	return stats, nil
}

// GetRecentQuestionContentsForUser retrieves recent question contents for a user
func (s *QuestionService) GetRecentQuestionContentsForUser(ctx context.Context, userID, limit int) (result0 []string, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_recent_question_contents_for_user", observability.AttributeUserID(userID), observability.AttributeLimit(limit))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	query := `
		SELECT DISTINCT q.content
		FROM user_responses ur
		JOIN questions q ON ur.question_id = q.id
		JOIN user_questions uq ON q.id = uq.question_id
		WHERE ur.user_id = $1 AND uq.user_id = $2
		ORDER BY q.content DESC
		LIMIT $3
	`

	var rows *sql.Rows
	rows, err = s.db.QueryContext(ctx, query, userID, userID, limit)
	if err != nil {
		return []string{}, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var contents []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return []string{}, err
		}
		contents = append(contents, content)
	}

	// Ensure we always return an empty slice instead of nil
	if contents == nil {
		contents = []string{}
	}

	return contents, nil
}

// GetUserQuestions retrieves actual questions for a user (not just content)
func (s *QuestionService) GetUserQuestions(ctx context.Context, userID, limit int) (result0 []*models.Question, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_user_questions", observability.AttributeUserID(userID), observability.AttributeLimit(limit))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	query := `
		SELECT q.id, q.type, q.language, q.level, q.difficulty_score, q.content, q.correct_answer, q.explanation, q.created_at, q.status, q.topic_category, q.grammar_focus, q.vocabulary_domain, q.scenario, q.style_modifier, q.difficulty_modifier, q.time_context
		FROM questions q
		JOIN user_questions uq ON q.id = uq.question_id
		WHERE uq.user_id = $1
		ORDER BY q.created_at DESC
		LIMIT $2
	`

	var rows *sql.Rows
	rows, err = s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var questions []*models.Question
	for rows.Next() {
		question, err := s.scanQuestionFromRows(rows)
		if err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}

	return questions, nil
}

// GetUserQuestionsWithStats retrieves questions for a user with response statistics
func (s *QuestionService) GetUserQuestionsWithStats(ctx context.Context, userID, limit int) (result0 []*QuestionWithStats, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_user_questions_with_stats", observability.AttributeUserID(userID), observability.AttributeLimit(limit))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	query := `
		SELECT
			q.id, q.type, q.language, q.level, q.difficulty_score,
			q.content, q.correct_answer, q.explanation, q.created_at, q.status,
			COALESCE(SUM(CASE WHEN ur.is_correct = true THEN 1 ELSE 0 END), 0) as correct_count,
			COALESCE(SUM(CASE WHEN ur.is_correct = false THEN 1 ELSE 0 END), 0) as incorrect_count,
			COALESCE(COUNT(ur.id), 0) as total_responses,
			COALESCE(uq_stats.user_count, 0) as user_count
		FROM questions q
		JOIN user_questions uq ON q.id = uq.question_id
		LEFT JOIN user_responses ur ON q.id = ur.question_id
		LEFT JOIN (
			SELECT
				question_id,
				COUNT(*) as user_count
			FROM user_questions
			GROUP BY question_id
		) uq_stats ON q.id = uq_stats.question_id
		WHERE uq.user_id = $1
		GROUP BY q.id, q.type, q.language, q.level, q.difficulty_score,
			q.content, q.correct_answer, q.explanation, q.created_at, q.status,
			uq_stats.user_count
		ORDER BY q.created_at DESC
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

	var questions []*QuestionWithStats
	for rows.Next() {
		questionWithStats, err := s.scanQuestionWithStatsFromRows(rows)
		if err != nil {
			return nil, err
		}
		questions = append(questions, questionWithStats)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return questions, nil
}

// QuestionWithStats represents a question with response statistics
type QuestionWithStats struct {
	*models.Question
	CorrectCount   int `json:"correct_count"`
	IncorrectCount int `json:"incorrect_count"`
	TotalResponses int `json:"total_responses"`
	// TimesAnswered tracks how many times THIS user answered the question (per-user)
	TimesAnswered   int    `json:"times_answered"`
	UserCount       int    `json:"user_count"`
	Reporters       string `json:"reporters,omitempty"`
	ReportReasons   string `json:"report_reasons,omitempty"`
	ConfidenceLevel *int   `json:"confidence_level,omitempty"`
}

// GetQuestionsPaginated retrieves questions with pagination and response statistics
func (s *QuestionService) GetQuestionsPaginated(ctx context.Context, userID, page, pageSize int, search, typeFilter, statusFilter string) (result0 []*QuestionWithStats, result1 int, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_questions_paginated", observability.AttributeUserID(userID), observability.AttributePage(page), observability.AttributePageSize(pageSize), observability.AttributeSearch(search), observability.AttributeTypeFilter(typeFilter), observability.AttributeStatusFilter(statusFilter))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Build WHERE clause with filters using parameterized queries
	whereConditions := []string{"uq.user_id = $1"}
	args := []interface{}{userID}
	argCount := 1

	// Add search filter
	if search != "" {
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("(q.content::text ILIKE $%d OR q.explanation ILIKE $%d)", argCount, argCount))
		args = append(args, "%"+search+"%")
	}

	// Add type filter
	if typeFilter != "" {
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("q.type = $%d", argCount))
		args = append(args, typeFilter)
	}

	// Add status filter
	if statusFilter != "" {
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("q.status = $%d", argCount))
		args = append(args, statusFilter)
	}

	// Join all conditions
	whereClause := "WHERE " + strings.Join(whereConditions, " AND ")

	// First get the total count with filters
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM questions q JOIN user_questions uq ON q.id = uq.question_id %s", whereClause)
	var totalCount int
	err = s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Build main query with pagination
	query := fmt.Sprintf(`
		SELECT
			q.id, q.type, q.language, q.level, q.difficulty_score,
			q.content, q.correct_answer, q.explanation, q.created_at, q.status,
			q.topic_category, q.grammar_focus, q.vocabulary_domain, q.scenario, q.style_modifier, q.difficulty_modifier, q.time_context,
			COALESCE(SUM(CASE WHEN ur.is_correct = true THEN 1 ELSE 0 END), 0) as correct_count,
			COALESCE(SUM(CASE WHEN ur.is_correct = false THEN 1 ELSE 0 END), 0) as incorrect_count,
			COALESCE(COUNT(ur.id), 0) as total_responses,
			COALESCE(uq_stats.user_count, 0) as user_count
		FROM questions q
		JOIN user_questions uq ON q.id = uq.question_id
		LEFT JOIN user_responses ur ON q.id = ur.question_id
		LEFT JOIN (
			SELECT
				question_id,
				COUNT(*) as user_count
			FROM user_questions
			GROUP BY question_id
		) uq_stats ON q.id = uq_stats.question_id
		%s
		GROUP BY q.id, q.type, q.language, q.level, q.difficulty_score,
			q.content, q.correct_answer, q.explanation, q.created_at, q.status,
			q.topic_category, q.grammar_focus, q.vocabulary_domain, q.scenario, q.style_modifier, q.difficulty_modifier, q.time_context,
			uq_stats.user_count
		ORDER BY q.id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount+1, argCount+2)

	// Add pagination parameters
	args = append(args, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var questions []*QuestionWithStats
	for rows.Next() {
		questionWithStats, err := s.scanQuestionWithStatsAndAllFieldsFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		questions = append(questions, questionWithStats)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return questions, totalCount, nil
}

// PRIORITY-BASED QUESTION SELECTION METHODS

// getAvailableQuestionsWithPriority retrieves available questions with priority scores and stats
func (s *QuestionService) getAvailableQuestionsWithPriority(ctx context.Context, userID int, language, level string, qType models.QuestionType, _ *models.UserLearningPreferences) (result0 []*QuestionWithStats, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_available_questions_with_priority", observability.AttributeUserID(userID), observability.AttributeLanguage(language), observability.AttributeLevel(level), observability.AttributeQuestionType(qType))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	// Build SQL query with priority scoring and stats
	query := `
		SELECT q.id, q.type, q.language, q.level, q.difficulty_score, q.content, q.correct_answer, q.explanation, q.created_at, q.status,
		       q.topic_category, q.grammar_focus, q.vocabulary_domain, q.scenario, q.style_modifier, q.difficulty_modifier, q.time_context,
		       COALESCE(qps.priority_score, 100.0) as priority_score,
		       COALESCE(uq_stats.times_answered, 0) as times_answered,
		       uq_stats.last_answered_at,
		       COALESCE(stats.correct_count, 0) as correct_count,
		       COALESCE(stats.incorrect_count, 0) as incorrect_count,
		       COALESCE(stats.total_responses, 0) as total_responses,
		       uqm.confidence_level
		FROM questions q
		JOIN user_questions uq ON q.id = uq.question_id
		LEFT JOIN question_priority_scores qps ON q.id = qps.question_id AND qps.user_id = $1
		LEFT JOIN (
			SELECT question_id,
			       COUNT(*) as times_answered,
			       MAX(created_at) as last_answered_at
			FROM user_responses
			WHERE user_id = $1
			GROUP BY question_id
		) uq_stats ON q.id = uq_stats.question_id
		LEFT JOIN (
			SELECT
				question_id,
				COUNT(CASE WHEN is_correct = true THEN 1 END) as correct_count,
				COUNT(CASE WHEN is_correct = false THEN 1 END) as incorrect_count,
				COUNT(*) as total_responses
			FROM user_responses
			GROUP BY question_id
		) stats ON q.id = stats.question_id
		LEFT JOIN user_question_metadata uqm ON q.id = uqm.question_id AND uqm.user_id = $1
		WHERE uq.user_id = $1
		AND q.language = $2
		AND q.level = $3
		AND q.type = $4
        AND q.status = 'active'
        AND q.id NOT IN (
            SELECT ur.question_id
            FROM user_responses ur
            WHERE ur.user_id = $1
              AND ur.created_at > NOW() - INTERVAL '1 hour'
        )
        -- Exclude questions where the user's last 3 responses were all correct within the last 90 days
        AND NOT EXISTS (
            SELECT 1 FROM (
                SELECT ur2.is_correct
                FROM user_responses ur2
                WHERE ur2.user_id = $1
                  AND ur2.question_id = q.id
                  AND ur2.created_at >= NOW() - INTERVAL '90 days'
                ORDER BY ur2.created_at DESC
                LIMIT 3
            ) recent_three
            WHERE (SELECT COUNT(*) FROM (
                SELECT 1 FROM (
                    SELECT ur3.is_correct
                    FROM user_responses ur3
                    WHERE ur3.user_id = $1
                      AND ur3.question_id = q.id
                      AND ur3.created_at >= NOW() - INTERVAL '90 days'
                    ORDER BY ur3.created_at DESC
                    LIMIT 3
                ) t WHERE t.is_correct = TRUE
            ) c) = 3
        )
        -- Exclude questions the user explicitly marked as known with max confidence (5)
        -- within the last 60 days (approx. 2 months)
        AND NOT EXISTS (
            SELECT 1 FROM user_question_metadata uqm2
            WHERE uqm2.user_id = $1
              AND uqm2.question_id = q.id
              AND uqm2.marked_as_known = TRUE
              AND uqm2.confidence_level = 5
              AND uqm2.marked_as_known_at >= NOW() - INTERVAL '60 days'
        )
		ORDER BY priority_score DESC, RANDOM()
		LIMIT 50
	`

	rows, err := s.db.QueryContext(ctx, query, userID, language, level, qType)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to query questions: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var questions []*QuestionWithStats
	for rows.Next() {
		questionWithStats, err := s.scanQuestionWithPriorityAndStatsFromRows(rows)
		if err != nil {
			s.logger.Error(ctx, "Error scanning question", err, map[string]interface{}{})
			continue // Skip malformed rows
		}
		questions = append(questions, questionWithStats)
	}

	return questions, nil
}

// getAvailableQuestionsForDailyWithPriority applies daily-specific eligibility:
// exclude questions answered correctly within the last 2 days for the user.
func (s *QuestionService) getAvailableQuestionsForDailyWithPriority(ctx context.Context, userID int, language, level string, qType models.QuestionType, _ *models.UserLearningPreferences) (result0 []*QuestionWithStats, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_available_questions_for_daily_with_priority", observability.AttributeUserID(userID), observability.AttributeLanguage(language), observability.AttributeLevel(level), observability.AttributeQuestionType(qType))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	avoidDays := s.getDailyRepeatAvoidDays()
	query := `
        SELECT q.id, q.type, q.language, q.level, q.difficulty_score, q.content, q.correct_answer, q.explanation, q.created_at, q.status,
               q.topic_category, q.grammar_focus, q.vocabulary_domain, q.scenario, q.style_modifier, q.difficulty_modifier, q.time_context,
               COALESCE(qps.priority_score, 100.0) as priority_score,
               COALESCE(uq_stats.times_answered, 0) as times_answered,
               uq_stats.last_answered_at,
               COALESCE(stats.correct_count, 0) as correct_count,
               COALESCE(stats.incorrect_count, 0) as incorrect_count,
               COALESCE(stats.total_responses, 0) as total_responses,
               uqm.confidence_level
        FROM questions q
        JOIN user_questions uq ON q.id = uq.question_id
        LEFT JOIN question_priority_scores qps ON q.id = qps.question_id AND qps.user_id = $1
        LEFT JOIN (
            SELECT question_id,
                   COUNT(*) as times_answered,
                   MAX(created_at) as last_answered_at
            FROM user_responses
            WHERE user_id = $1
            GROUP BY question_id
        ) uq_stats ON q.id = uq_stats.question_id
        LEFT JOIN (
            SELECT
                question_id,
                COUNT(CASE WHEN is_correct = true THEN 1 END) as correct_count,
                COUNT(CASE WHEN is_correct = false THEN 1 END) as incorrect_count,
                COUNT(*) as total_responses
            FROM user_responses
            GROUP BY question_id
        ) stats ON q.id = stats.question_id
        LEFT JOIN user_question_metadata uqm ON q.id = uqm.question_id AND uqm.user_id = $1
        WHERE uq.user_id = $1
        AND q.language = $2
        AND q.level = $3
        AND q.type = $4
        AND q.status = 'active'
        AND NOT EXISTS (
            SELECT 1
            FROM user_responses ur
            WHERE ur.user_id = $1
              AND ur.question_id = q.id
              AND ur.is_correct = TRUE
              AND ur.created_at >= NOW() - ($5 || ' days')::interval
        )
        -- Exclude questions the user marked as known with confidence 5 within last 60 days
        AND NOT EXISTS (
            SELECT 1 FROM user_question_metadata uqm2
            WHERE uqm2.user_id = $1
              AND uqm2.question_id = q.id
              AND uqm2.marked_as_known = TRUE
              AND uqm2.confidence_level = 5
              AND uqm2.marked_as_known_at >= NOW() - INTERVAL '60 days'
        )
        ORDER BY priority_score DESC, RANDOM()
        LIMIT 50
    `

	rows, err := s.db.QueryContext(ctx, query, userID, language, level, qType, avoidDays)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to query questions (daily): %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var questions []*QuestionWithStats
	for rows.Next() {
		questionWithStats, err := s.scanQuestionWithPriorityAndStatsFromRows(rows)
		if err != nil {
			s.logger.Error(ctx, "Error scanning question (daily)", err, map[string]interface{}{})
			continue
		}
		questions = append(questions, questionWithStats)
	}

	return questions, nil
}

// selectQuestionWithWeightedRandomness selects a question using weighted random selection
func (s *QuestionService) selectQuestionWithWeightedRandomness(questions []*QuestionWithStats) (result0 *QuestionWithStats, err error) {
	if len(questions) == 0 {
		return nil, contextutils.WrapError(contextutils.ErrRecordNotFound, "no questions available")
	}

	// Use weighted random selection based on usage count (lower = higher priority)
	totalWeight := 0.0
	for _, q := range questions {
		// Prefer per-user times answered when available
		usageCount := q.TotalResponses
		if q.TimesAnswered >= 0 {
			usageCount = q.TimesAnswered
		}
		// Lower usage count = higher weight
		weight := 1.0 / (float64(usageCount) + 1.0)
		totalWeight += weight
	}

	// Handle edge case where all questions have zero weight or floating-point precision issues
	if totalWeight <= 0 {
		// If all questions have equal weight (e.g., all TotalResponses = 0), use simple random selection
		return questions[rand.Intn(len(questions))], nil
	}

	target := rand.Float64() * totalWeight
	currentWeight := 0.0

	for _, q := range questions {
		usageCount := q.TotalResponses
		if q.TimesAnswered >= 0 {
			usageCount = q.TimesAnswered
		}
		weight := 1.0 / (float64(usageCount) + 1.0)
		currentWeight += weight
		if currentWeight >= target {
			return q, nil
		}
	}

	// Fallback: if we reach the end without selecting (due to floating-point precision),
	// return the last question or a random one
	if len(questions) > 0 {
		return questions[len(questions)-1], nil
	}

	return nil, contextutils.WrapError(contextutils.ErrInternalError, "failed to select question with weighted randomness")
}

// selectQuestionWithFreshnessRatio selects a question based on freshness ratio
func (s *QuestionService) selectQuestionWithFreshnessRatio(questions []*QuestionWithStats, freshnessRatio float64) (result0 *QuestionWithStats, err error) {
	if len(questions) == 0 {
		return nil, contextutils.WrapError(contextutils.ErrRecordNotFound, "no questions available")
	}

	// Separate fresh and review questions based on total responses
	var freshQuestions []*QuestionWithStats
	var reviewQuestions []*QuestionWithStats

	for _, q := range questions {
		// Consider fresh relative to this user (TimesAnswered==0). Fall back to TotalResponses if TimesAnswered not set.
		isFresh := false
		if q.TimesAnswered >= 0 {
			isFresh = q.TimesAnswered == 0
		} else {
			isFresh = q.TotalResponses == 0
		}
		if isFresh {
			freshQuestions = append(freshQuestions, q)
		} else {
			reviewQuestions = append(reviewQuestions, q)
		}
	}

	// Use probabilistic selection based on the freshness ratio
	var selectedQuestions []*QuestionWithStats
	if len(freshQuestions) > 0 && len(reviewQuestions) > 0 {
		// Both categories available - use probabilistic selection
		if rand.Float64() < freshnessRatio {
			selectedQuestions = freshQuestions
		} else {
			selectedQuestions = reviewQuestions
		}
	} else if len(freshQuestions) > 0 {
		// Only fresh questions available
		selectedQuestions = freshQuestions
	} else if len(reviewQuestions) > 0 {
		// Only review questions available
		selectedQuestions = reviewQuestions
	} else {
		// Fallback to all questions if no separation possible
		selectedQuestions = questions
	}

	if len(selectedQuestions) == 0 {
		return nil, contextutils.WrapError(contextutils.ErrRecordNotFound, "no questions available after freshness filtering")
	}

	// Use weighted random selection within the chosen category
	result, err := s.selectQuestionWithWeightedRandomness(selectedQuestions)
	if err != nil {
		// Log debug info about the selection failure
		s.logger.Warn(context.Background(), "selectQuestionWithWeightedRandomness failed", map[string]interface{}{
			"total_questions":        len(questions),
			"fresh_questions":        len(freshQuestions),
			"review_questions":       len(reviewQuestions),
			"selected_category_size": len(selectedQuestions),
			"freshness_ratio":        freshnessRatio,
			"error":                  err.Error(),
		})
	}
	return result, err
}

// GetUserQuestionCount returns the total number of questions available for a user
func (s *QuestionService) GetUserQuestionCount(ctx context.Context, userID int) (result0 int, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_user_question_count", observability.AttributeUserID(userID))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	query := `
		SELECT COUNT(DISTINCT q.id)
		FROM questions q
		JOIN user_questions uq ON q.id = uq.question_id
		WHERE uq.user_id = $1 AND q.status = 'active'
	`

	var count int
	err = s.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get user question count: %v", err)
	}
	return count, nil
}

// GetUserResponseCount returns the total number of responses for a user
func (s *QuestionService) GetUserResponseCount(ctx context.Context, userID int) (result0 int, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_user_response_count", observability.AttributeUserID(userID))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	query := `SELECT COUNT(*) FROM user_responses WHERE user_id = $1`

	var count int
	err = s.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get user response count: %v", err)
	}
	return count, nil
}

// GetUsersForQuestion returns the users assigned to a question, up to 5 users, and the total count
func (s *QuestionService) GetUsersForQuestion(ctx context.Context, questionID int) (result0 []*models.User, result1 int, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_users_for_question", observability.AttributeQuestionID(questionID))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// First get the total count
	countQuery := `SELECT COUNT(*) FROM user_questions WHERE question_id = $1`
	var totalCount int
	err = s.db.QueryRowContext(ctx, countQuery, questionID).Scan(&totalCount)
	if err != nil {
		return nil, 0, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get user count for question: %v", err)
	}

	// Then get up to 5 users
	usersQuery := `
		SELECT u.id, u.username, u.email, u.timezone, u.password_hash, u.last_active,
		       u.preferred_language, u.current_level, u.ai_provider, u.ai_model,
		       u.ai_enabled, u.ai_api_key, u.created_at, u.updated_at
		FROM users u
		JOIN user_questions uq ON u.id = uq.user_id
		WHERE uq.question_id = $1
		ORDER BY u.username
		LIMIT 5
	`

	rows, err := s.db.QueryContext(ctx, usersQuery, questionID)
	if err != nil {
		return nil, 0, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get users for question: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": err.Error()})
		}
	}()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err = rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Timezone,
			&user.PasswordHash,
			&user.LastActive,
			&user.PreferredLanguage,
			&user.CurrentLevel,
			&user.AIProvider,
			&user.AIModel,
			&user.AIEnabled,
			&user.AIAPIKey,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to scan user: %v", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "error iterating users: %v", err)
	}

	// Ensure we always return an empty slice instead of nil
	if users == nil {
		users = make([]*models.User, 0)
	}

	return users, totalCount, nil
}

// Helper: scan a *sql.Row into a QuestionWithStats (for single-row queries)
func (s *QuestionService) scanQuestionWithPriorityAndStatsFromRow(row *sql.Row) (result0 *QuestionWithStats, err error) {
	questionWithStats := &QuestionWithStats{
		Question: &models.Question{},
	}
	var contentJSON string
	var priorityScore float64
	var timesAnswered int
	var lastAnsweredAt sql.NullTime

	err = row.Scan(
		&questionWithStats.ID,
		&questionWithStats.Type,
		&questionWithStats.Language,
		&questionWithStats.Level,
		&questionWithStats.DifficultyScore,
		&contentJSON,
		&questionWithStats.CorrectAnswer,
		&questionWithStats.Explanation,
		&questionWithStats.CreatedAt,
		&questionWithStats.Status,
		&questionWithStats.TopicCategory,
		&questionWithStats.GrammarFocus,
		&questionWithStats.VocabularyDomain,
		&questionWithStats.Scenario,
		&questionWithStats.StyleModifier,
		&questionWithStats.DifficultyModifier,
		&questionWithStats.TimeContext,
		&priorityScore,
		&timesAnswered,
		&lastAnsweredAt,
		&questionWithStats.CorrectCount,
		&questionWithStats.IncorrectCount,
		&questionWithStats.TotalResponses,
	)
	if err != nil {
		return nil, err
	}

	if err := questionWithStats.UnmarshalContentFromJSON(contentJSON); err != nil {
		return nil, err
	}

	return questionWithStats, nil
}

// GetRandomGlobalQuestionForUser finds a random question from the global pool for the given language, level, and type that is not already assigned to the user, assigns it, and returns it.
func (s *QuestionService) GetRandomGlobalQuestionForUser(ctx context.Context, userID int, language, level string, qType models.QuestionType) (result0 *QuestionWithStats, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_random_global_question_for_user", observability.AttributeUserID(userID), observability.AttributeLanguage(language), observability.AttributeLevel(level), observability.AttributeQuestionType(qType))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT q.id, q.type, q.language, q.level, q.difficulty_score, q.content, q.correct_answer, q.explanation, q.created_at, q.status,
		       q.topic_category, q.grammar_focus, q.vocabulary_domain, q.scenario, q.style_modifier, q.difficulty_modifier, q.time_context,
		       100.0 as priority_score, 0 as times_answered, NULL as last_answered_at, 0 as correct_count, 0 as incorrect_count, 0 as total_responses
		FROM questions q
		WHERE q.language = $1
		  AND q.level = $2
        AND q.type = $3
          AND q.status = 'active'
          AND q.id NOT IN (
            SELECT uq.question_id
            FROM user_questions uq
            WHERE uq.user_id = $4
          )
          -- Exclude questions the user marked as known with confidence 5 within last 60 days
          AND NOT EXISTS (
            SELECT 1 FROM user_question_metadata uqm2
            WHERE uqm2.user_id = $4
              AND uqm2.question_id = q.id
              AND uqm2.marked_as_known = TRUE
              AND uqm2.confidence_level = 5
              AND uqm2.marked_as_known_at >= NOW() - INTERVAL '60 days'
          )
		ORDER BY RANDOM()
		LIMIT 1
	`

	row := s.db.QueryRowContext(ctx, query, language, level, qType, userID)
	questionWithStats, err := s.scanQuestionWithPriorityAndStatsFromRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No global questions available
		}
		return nil, err
	}

	// Assign the question to the user
	err = s.AssignQuestionToUser(ctx, questionWithStats.ID, userID)
	if err != nil {
		s.logger.Warn(ctx, "Failed to assign global question to user", map[string]interface{}{"question_id": questionWithStats.ID, "user_id": userID, "error": err.Error()})
		// Still return the question, but log the error
	}

	return questionWithStats, nil
}

// GetAllQuestionsPaginated returns all questions with pagination and filtering
func (s *QuestionService) GetAllQuestionsPaginated(ctx context.Context, page, pageSize int, search, typeFilter, statusFilter, languageFilter, levelFilter string, userID *int) (result0 []*QuestionWithStats, result1 int, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_all_questions_paginated")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Build the base query
	baseQuery := `
		SELECT q.id, q.type, q.language, q.level, q.difficulty_score, q.content, q.correct_answer, q.explanation, q.created_at, q.status,
		       q.topic_category, q.grammar_focus, q.vocabulary_domain, q.scenario, q.style_modifier, q.difficulty_modifier, q.time_context,
		       COALESCE(ur_stats.correct_count, 0) as correct_count,
		       COALESCE(ur_stats.incorrect_count, 0) as incorrect_count,
		       COALESCE(ur_stats.total_responses, 0) as total_responses,
		       COALESCE(uq_stats.user_count, 0) as user_count
		FROM questions q
		LEFT JOIN (
			SELECT
				question_id,
				COUNT(CASE WHEN is_correct = true THEN 1 END) as correct_count,
				COUNT(CASE WHEN is_correct = false THEN 1 END) as incorrect_count,
				COUNT(*) as total_responses
			FROM user_responses
			GROUP BY question_id
		) ur_stats ON q.id = ur_stats.question_id
		LEFT JOIN (
			SELECT
				question_id,
				COUNT(*) as user_count
			FROM user_questions
			GROUP BY question_id
		) uq_stats ON q.id = uq_stats.question_id
		WHERE 1=1
	`

	// Build the count query
	countQuery := `
		SELECT COUNT(*)
		FROM questions q
		WHERE 1=1
	`

	var args []interface{}
	argIndex := 1

	// Add filters
	if search != "" {
		searchCondition := ` AND (q.content::text ILIKE $` + strconv.Itoa(argIndex) + ` OR q.explanation ILIKE $` + strconv.Itoa(argIndex) + `)`
		baseQuery += searchCondition
		countQuery += searchCondition
		args = append(args, "%"+search+"%")
		argIndex++
	}

	if typeFilter != "" {
		typeCondition := ` AND q.type = $` + strconv.Itoa(argIndex)
		baseQuery += typeCondition
		countQuery += typeCondition
		args = append(args, typeFilter)
		argIndex++
	}

	if statusFilter != "" {
		statusCondition := ` AND q.status = $` + strconv.Itoa(argIndex)
		baseQuery += statusCondition
		countQuery += statusCondition
		args = append(args, statusFilter)
		argIndex++
	}

	if languageFilter != "" {
		languageCondition := ` AND q.language = $` + strconv.Itoa(argIndex)
		baseQuery += languageCondition
		countQuery += languageCondition
		args = append(args, languageFilter)
		argIndex++
	}

	if levelFilter != "" {
		levelCondition := ` AND q.level = $` + strconv.Itoa(argIndex)
		baseQuery += levelCondition
		countQuery += levelCondition
		args = append(args, levelFilter)
		argIndex++
	}

	if userID != nil {
		userCondition := ` AND q.id IN (SELECT question_id FROM user_questions WHERE user_id = $` + strconv.Itoa(argIndex) + `)`
		baseQuery += userCondition
		countQuery += userCondition
		args = append(args, *userID)
		argIndex++
	}

	// Get total count
	var total int
	err = s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get total count: %v", err)
	}

	// Add pagination
	offset := (page - 1) * pageSize
	baseQuery += ` ORDER BY q.created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	args = append(args, pageSize, offset)

	// Execute the main query
	rows, err := s.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get questions: %v", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Warning: failed to close rows", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	var questions []*QuestionWithStats
	for rows.Next() {
		question, err := s.scanQuestionWithStatsAndAllFieldsFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		questions = append(questions, question)
	}

	return questions, total, nil
}

// GetReportedQuestionsPaginated returns reported questions with pagination and filtering
func (s *QuestionService) GetReportedQuestionsPaginated(ctx context.Context, page, pageSize int, search, typeFilter, languageFilter, levelFilter string) (result0 []*QuestionWithStats, result1 int, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_reported_questions_paginated")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	// Build WHERE clause with filters using parameterized queries
	whereConditions := []string{"q.status = 'reported'"}
	args := []interface{}{}
	argCount := 0

	// Add search filter
	if search != "" {
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("(q.content::text ILIKE $%d OR q.explanation ILIKE $%d)", argCount, argCount))
		args = append(args, "%"+search+"%")
	}

	// Add type filter
	if typeFilter != "" {
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("q.type = $%d", argCount))
		args = append(args, typeFilter)
	}

	// Add language filter
	if languageFilter != "" {
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("q.language = $%d", argCount))
		args = append(args, languageFilter)
	}

	// Add level filter
	if levelFilter != "" {
		argCount++
		whereConditions = append(whereConditions, fmt.Sprintf("q.level = $%d", argCount))
		args = append(args, levelFilter)
	}

	// Join all conditions
	whereClause := "WHERE " + strings.Join(whereConditions, " AND ")

	// Build the count query
	countQuery := fmt.Sprintf("SELECT COUNT(DISTINCT q.id) FROM questions q %s", whereClause)
	var total int
	err = s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get total count: %v", err)
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Build main query with pagination
	query := fmt.Sprintf(`
		SELECT q.id, q.type, q.language, q.level, q.difficulty_score, q.content, q.correct_answer, q.explanation, q.created_at, q.status,
		       q.topic_category, q.grammar_focus, q.vocabulary_domain, q.scenario, q.style_modifier, q.difficulty_modifier, q.time_context,
		       COALESCE(ur_stats.correct_count, 0) as correct_count,
		       COALESCE(ur_stats.incorrect_count, 0) as incorrect_count,
		       COALESCE(ur_stats.total_responses, 0) as total_responses,
		       STRING_AGG(DISTINCT u.username, ', ') as reporters,
		       STRING_AGG(DISTINCT qr.report_reason, ' | ') as report_reasons
		FROM questions q
		LEFT JOIN (
			SELECT
				question_id,
				COUNT(CASE WHEN is_correct = true THEN 1 END) as correct_count,
				COUNT(CASE WHEN is_correct = false THEN 1 END) as incorrect_count,
				COUNT(*) as total_responses
			FROM user_responses
			GROUP BY question_id
		) ur_stats ON q.id = ur_stats.question_id
		LEFT JOIN question_reports qr ON q.id = qr.question_id
		LEFT JOIN users u ON qr.reported_by_user_id = u.id
		%s
		GROUP BY q.id, q.type, q.language, q.level, q.difficulty_score, q.content, q.correct_answer, q.explanation, q.created_at, q.status,
		         q.topic_category, q.grammar_focus, q.vocabulary_domain, q.scenario, q.style_modifier, q.difficulty_modifier, q.time_context,
		         ur_stats.correct_count, ur_stats.incorrect_count, ur_stats.total_responses
		ORDER BY q.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount+1, argCount+2)

	// Add pagination parameters
	args = append(args, pageSize, offset)

	// Execute the main query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get reported questions: %v", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Warning: failed to close rows", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	var questions []*QuestionWithStats
	for rows.Next() {
		question, err := s.scanQuestionWithStatsAndReportersFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		questions = append(questions, question)
	}

	return questions, total, nil
}

// GetReportedQuestionsStats returns statistics about reported questions
func (s *QuestionService) GetReportedQuestionsStats(ctx context.Context) (result0 map[string]interface{}, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_reported_questions_stats")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	stats := make(map[string]interface{})

	// Get total reported questions
	var totalReported int
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM questions WHERE status = 'reported'`).Scan(&totalReported)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get total reported questions: %v", err)
	}
	stats["total_reported"] = totalReported

	// Get reported questions by type
	rows, err := s.db.QueryContext(ctx, `
		SELECT type, COUNT(*) as count
		FROM questions
		WHERE status = 'reported'
		GROUP BY type
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get reported questions by type: %v", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Warning: failed to close rows", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	reportedByType := make(map[string]int)
	for rows.Next() {
		var questionType string
		var count int
		if err := rows.Scan(&questionType, &count); err != nil {
			return nil, err
		}
		reportedByType[questionType] = count
	}
	stats["reported_by_type"] = reportedByType

	// Get reported questions by level
	rows, err = s.db.QueryContext(ctx, `
		SELECT level, COUNT(*) as count
		FROM questions
		WHERE status = 'reported'
		GROUP BY level
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get reported questions by level: %v", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Warning: failed to close rows", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	reportedByLevel := make(map[string]int)
	for rows.Next() {
		var level string
		var count int
		if err := rows.Scan(&level, &count); err != nil {
			return nil, err
		}
		reportedByLevel[level] = count
	}
	stats["reported_by_level"] = reportedByLevel

	// Get reported questions by language
	rows, err = s.db.QueryContext(ctx, `
		SELECT language, COUNT(*) as count
		FROM questions
		WHERE status = 'reported'
		GROUP BY language
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, contextutils.WrapErrorf(contextutils.ErrDatabaseQuery, "failed to get reported questions by language: %v", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Warning: failed to close rows", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	reportedByLanguage := make(map[string]int)
	for rows.Next() {
		var language string
		var count int
		if err := rows.Scan(&language, &count); err != nil {
			return nil, err
		}
		reportedByLanguage[language] = count
	}
	stats["reported_by_language"] = reportedByLanguage

	return stats, nil
}

// AssignUsersToQuestion assigns multiple users to a question
func (s *QuestionService) AssignUsersToQuestion(ctx context.Context, questionID int, userIDs []int) (err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "assign_users_to_question", observability.AttributeQuestionID(questionID))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Start a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contextutils.WrapError(err, "failed to begin transaction")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Warn(ctx, "Failed to rollback transaction", map[string]interface{}{"error": rollbackErr.Error()})
			}
		}
	}()

	// Prepare the insert statement
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO user_questions (user_id, question_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, question_id) DO NOTHING
	`)
	if err != nil {
		return contextutils.WrapError(err, "failed to prepare insert statement")
	}
	defer func() {
		if closeErr := stmt.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Warning: failed to close statement", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	// Insert each user-question mapping
	for _, userID := range userIDs {
		_, err = stmt.ExecContext(ctx, userID, questionID)
		if err != nil {
			return contextutils.WrapErrorf(err, "failed to assign user %d to question %d", userID, questionID)
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return contextutils.WrapError(err, "failed to commit transaction")
	}

	return nil
}

// UnassignUsersFromQuestion removes multiple users from a question
func (s *QuestionService) UnassignUsersFromQuestion(ctx context.Context, questionID int, userIDs []int) (err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "unassign_users_from_question", observability.AttributeQuestionID(questionID))
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Start a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contextutils.WrapError(err, "failed to begin transaction")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Warn(ctx, "Failed to rollback transaction", map[string]interface{}{"error": rollbackErr.Error()})
			}
		}
	}()

	// Prepare the delete statement
	stmt, err := tx.PrepareContext(ctx, `
		DELETE FROM user_questions
		WHERE user_id = $1 AND question_id = $2
	`)
	if err != nil {
		return contextutils.WrapError(err, "failed to prepare delete statement")
	}
	defer func() {
		if closeErr := stmt.Close(); closeErr != nil {
			s.logger.Warn(ctx, "Warning: failed to close statement", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	// Delete each user-question mapping
	for _, userID := range userIDs {
		_, err = stmt.ExecContext(ctx, userID, questionID)
		if err != nil {
			return contextutils.WrapErrorf(err, "failed to unassign user %d from question %d", userID, questionID)
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return contextutils.WrapError(err, "failed to commit transaction")
	}

	return nil
}

// DB returns the underlying *sql.DB instance
func (s *QuestionService) DB() *sql.DB {
	return s.db
}
