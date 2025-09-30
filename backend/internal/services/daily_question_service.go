package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// DailyQuestionServiceInterface defines the interface for daily question operations
type DailyQuestionServiceInterface interface {
	AssignDailyQuestions(ctx context.Context, userID int, date time.Time) error
	RegenerateDailyQuestions(ctx context.Context, userID int, date time.Time) error
	GetDailyQuestions(ctx context.Context, userID int, date time.Time) ([]*models.DailyQuestionAssignmentWithQuestion, error)
	MarkQuestionCompleted(ctx context.Context, userID, questionID int, date time.Time) error
	ResetQuestionCompleted(ctx context.Context, userID, questionID int, date time.Time) error
	SubmitDailyQuestionAnswer(ctx context.Context, userID, questionID int, date time.Time, userAnswerIndex int) (*api.AnswerResponse, error)
	GetAvailableDates(ctx context.Context, userID int) ([]time.Time, error)
	GetDailyProgress(ctx context.Context, userID int, date time.Time) (*models.DailyProgress, error)
	GetDailyQuestionsCount(ctx context.Context, userID int, date time.Time) (int, error)
	GetCompletedDailyQuestionsCount(ctx context.Context, userID int, date time.Time) (int, error)
	GetQuestionHistory(ctx context.Context, userID, questionID, days int) ([]*models.DailyQuestionHistory, error)
}

// DailyQuestionService implements daily question assignment and management
type DailyQuestionService struct {
	db              *sql.DB
	logger          *observability.Logger
	questionService QuestionServiceInterface
	learningService LearningServiceInterface
}

// NewDailyQuestionService creates a new DailyQuestionService instance
func NewDailyQuestionService(db *sql.DB, logger *observability.Logger, questionService QuestionServiceInterface, learningService LearningServiceInterface) *DailyQuestionService {
	return &DailyQuestionService{
		db:              db,
		logger:          logger,
		questionService: questionService,
		learningService: learningService,
	}
}

// AssignDailyQuestions assigns 10 random questions to a user for a specific date
func (s *DailyQuestionService) AssignDailyQuestions(ctx context.Context, userID int, date time.Time) (err error) {
	ctx, span := otel.Tracer("daily-question-service").Start(ctx, "AssignDailyQuestions",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.String("date", date.Format("2006-01-02")),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Get user to determine language and level preferences
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to get user")
	}

	if user == nil {
		return contextutils.ErrorWithContextf("user not found: %d", userID)
	}

	language := user.PreferredLanguage.String
	level := user.CurrentLevel.String

	if language == "" || level == "" {
		return contextutils.ErrorWithContextf("user missing language or level preferences")
	}

	// Get user's daily goal from learning preferences
	prefs, perr := s.learningService.GetUserLearningPreferences(ctx, userID)
	if perr != nil {
		span.RecordError(perr)
		return contextutils.WrapError(perr, "failed to get user learning preferences")
	}
	goal := 10
	if prefs != nil && prefs.DailyGoal > 0 {
		goal = prefs.DailyGoal
	}

	// Check existing assignments and only fill missing slots up to the user's goal
	existingCount, err := s.GetDailyQuestionsCount(ctx, userID, date)
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to check existing assignments")
	}
	if existingCount >= goal {
		s.logger.Info(ctx, "Daily questions already assigned for date", map[string]interface{}{
			"user_id": userID,
			"date":    date.Format("2006-01-02"),
			"count":   existingCount,
			"goal":    goal,
		})
		return nil // Already assigned
	}

	// Request more candidates than strictly needed to allow filtering out already-assigned questions
	buffer := 10 // request this many extra candidates beyond the user's goal
	reqLimit := goal + buffer

	// Get adaptive questions using an expanded limit so we can filter and still meet goal
	questionsWithStats, err := s.questionService.GetAdaptiveQuestionsForDaily(ctx, userID, language, level, reqLimit)
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to get adaptive questions for assignment")
	}

	if len(questionsWithStats) == 0 {
		// Gather diagnostics to explain why no questions were available
		var candidateIDs []int
		candidateCount := 0
		totalMatching := 0
		if s.questionService != nil {
			if candidates, qerr := s.questionService.GetAdaptiveQuestionsForDaily(ctx, userID, language, level, 50); qerr == nil && candidates != nil {
				candidateCount = len(candidates)
				for i, q := range candidates {
					if i >= 10 {
						break
					}
					if q != nil {
						candidateIDs = append(candidateIDs, q.ID)
					}
				}
			}
			if _, total, terr := s.questionService.GetAllQuestionsPaginated(ctx, 1, 1, "", "", "", language, level, nil); terr == nil {
				totalMatching = total
			}
		}

		return &NoQuestionsAvailableError{
			Language:       language,
			Level:          level,
			CandidateIDs:   candidateIDs,
			CandidateCount: candidateCount,
			TotalMatching:  totalMatching,
		}
	}

	// Filter out questions that are already assigned for this user/date to
	// avoid selecting already-inserted questions and thus underfilling the goal.
	assignedIDs := make(map[int]bool)
	rows, qerr := s.db.QueryContext(ctx, `SELECT question_id FROM daily_question_assignments WHERE user_id = $1 AND assignment_date = $2`, userID, date)
	if qerr == nil {
		defer func() {
			if closeErr := rows.Close(); closeErr != nil {
				s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": closeErr.Error()})
			}
		}()
		for rows.Next() {
			var qid int
			if err := rows.Scan(&qid); err == nil {
				assignedIDs[qid] = true
			}
		}
	}

	// Convert QuestionWithStats to Question for assignment, skipping already-assigned
	var questions []models.Question
	for _, qws := range questionsWithStats {
		if qws == nil || qws.Question == nil {
			continue
		}
		if assignedIDs[qws.ID] {
			// already assigned for this date, skip
			continue
		}
		questions = append(questions, *qws.Question)
	}

	// Only insert up to the number of slots we need to fill
	toAssign := goal - existingCount
	if toAssign < 0 {
		toAssign = 0
	}
	if len(questions) > toAssign {
		questions = questions[:toAssign]
	}

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to begin transaction")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Error(ctx, "Failed to rollback transaction", rollbackErr, map[string]interface{}{
					"user_id": userID,
					"date":    date.Format("2006-01-02"),
				})
			}
		}
	}()

	// Insert assignments (idempotent via conditional INSERT to avoid duplicate rows)
	insertQuery := `
		INSERT INTO daily_question_assignments (user_id, question_id, assignment_date, created_at)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (
			SELECT 1 FROM daily_question_assignments WHERE user_id = $1 AND question_id = $2 AND assignment_date = $3
		)
	`

	for _, question := range questions {
		_, err = tx.ExecContext(ctx, insertQuery, userID, question.ID, date, time.Now())
		if err != nil {
			span.RecordError(err)
			return contextutils.WrapError(err, "failed to insert assignment")
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to commit transaction")
	}

	s.logger.Info(ctx, "Daily questions assigned successfully", map[string]interface{}{
		"user_id": userID,
		"date":    date.Format("2006-01-02"),
		"count":   len(questions),
	})

	return nil
}

// RegenerateDailyQuestions clears existing daily question assignments and creates new ones for a user and date
func (s *DailyQuestionService) RegenerateDailyQuestions(ctx context.Context, userID int, date time.Time) (err error) {
	ctx, span := otel.Tracer("daily-question-service").Start(ctx, "RegenerateDailyQuestions",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.String("date", date.Format("2006-01-02")),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Get user to determine language and level preferences
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to get user")
	}

	if user == nil {
		return contextutils.ErrorWithContextf("user not found: %d", userID)
	}

	language := user.PreferredLanguage.String
	level := user.CurrentLevel.String

	if language == "" || level == "" {
		return contextutils.ErrorWithContextf("user missing language or level preferences")
	}

	// Get user's daily goal from learning preferences
	prefs, perr := s.learningService.GetUserLearningPreferences(ctx, userID)
	if perr != nil {
		span.RecordError(perr)
		return contextutils.WrapError(perr, "failed to get user learning preferences")
	}
	goal := 10
	if prefs != nil && prefs.DailyGoal > 0 {
		goal = prefs.DailyGoal
	}

	// Request more candidates than strictly needed to allow filtering out already-assigned questions
	buffer := 10 // request this many extra candidates beyond the user's goal
	reqLimit := goal + buffer

	// Get adaptive questions using an expanded limit so we can filter and still meet goal
	questionsWithStats, err := s.questionService.GetAdaptiveQuestionsForDaily(ctx, userID, language, level, reqLimit)
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to get adaptive questions for assignment")
	}

	if len(questionsWithStats) == 0 {
		// Gather diagnostics to explain why no questions were available
		var candidateIDs []int
		candidateCount := 0
		totalMatching := 0
		if s.questionService != nil {
			if candidates, qerr := s.questionService.GetAdaptiveQuestionsForDaily(ctx, userID, language, level, 50); qerr == nil && candidates != nil {
				candidateCount = len(candidates)
				for i, q := range candidates {
					if i >= 10 {
						break
					}
					if q != nil {
						candidateIDs = append(candidateIDs, q.ID)
					}
				}
			}
			if _, total, terr := s.questionService.GetAllQuestionsPaginated(ctx, 1, 1, "", "", "", language, level, nil); terr == nil {
				totalMatching = total
			}
		}

		return &NoQuestionsAvailableError{
			Language:       language,
			Level:          level,
			CandidateIDs:   candidateIDs,
			CandidateCount: candidateCount,
			TotalMatching:  totalMatching,
		}
	}

	// Convert QuestionWithStats to Question for assignment
	var questions []models.Question
	for _, qws := range questionsWithStats {
		questions = append(questions, *qws.Question)
	}

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to begin transaction")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Error(ctx, "Failed to rollback transaction", rollbackErr, map[string]interface{}{
					"user_id": userID,
					"date":    date.Format("2006-01-02"),
				})
			}
		}
	}()

	// First, delete existing assignments for this user and date
	deleteQuery := `DELETE FROM daily_question_assignments WHERE user_id = $1 AND assignment_date = $2`
	_, err = tx.ExecContext(ctx, deleteQuery, userID, date)
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to delete existing assignments")
	}

	// Insert new assignments
	insertQuery := `
		INSERT INTO daily_question_assignments (user_id, question_id, assignment_date, created_at)
		VALUES ($1, $2, $3, $4)
	`

	stmt, err := tx.PrepareContext(ctx, insertQuery)
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to prepare statement")
	}
	defer func() {
		if closeErr := stmt.Close(); closeErr != nil {
			s.logger.Error(ctx, "Failed to close statement", closeErr, map[string]interface{}{
				"user_id": userID,
				"date":    date.Format("2006-01-02"),
			})
		}
	}()

	// Only assign up to the goal amount
	assignedCount := 0
	for _, question := range questions {
		if assignedCount >= goal {
			break
		}
		_, err = stmt.ExecContext(ctx, userID, question.ID, date, time.Now())
		if err != nil {
			span.RecordError(err)
			return contextutils.WrapError(err, "failed to insert assignment")
		}
		assignedCount++
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to commit transaction")
	}

	s.logger.Info(ctx, "Daily questions regenerated successfully", map[string]interface{}{
		"user_id": userID,
		"date":    date.Format("2006-01-02"),
		"count":   len(questions),
	})

	return nil
}

// GetDailyQuestions retrieves all daily questions for a user on a specific date
func (s *DailyQuestionService) GetDailyQuestions(ctx context.Context, userID int, date time.Time) (result0 []*models.DailyQuestionAssignmentWithQuestion, err error) {
	ctx, span := otel.Tracer("daily-question-service").Start(ctx, "GetDailyQuestions",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.String("date", date.Format("2006-01-02")),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
        SELECT dqa.id, dqa.user_id, dqa.question_id, dqa.assignment_date,
               dqa.is_completed, dqa.completed_at, dqa.created_at,
               dqa.user_answer_index, dqa.submitted_at,
               q.id, q.type, q.language, q.level, q.difficulty_score, q.content,
               q.correct_answer, q.explanation, q.created_at, q.status,
               q.topic_category, q.grammar_focus, q.vocabulary_domain, q.scenario,
               q.style_modifier, q.difficulty_modifier, q.time_context,
               -- Daily shown count per user: how many times this user has seen this question in Daily across all dates
               (SELECT COUNT(*) FROM daily_question_assignments dqa_all WHERE dqa_all.question_id = dqa.question_id AND dqa_all.user_id = dqa.user_id) AS daily_shown_count,
               -- Per-user correctness stats across all time
               COALESCE((SELECT COUNT(*) FROM user_responses ur WHERE ur.user_id = dqa.user_id AND ur.question_id = dqa.question_id), 0) AS user_total_responses,
               COALESCE((SELECT COUNT(*) FROM user_responses ur WHERE ur.user_id = dqa.user_id AND ur.question_id = dqa.question_id AND ur.is_correct = TRUE), 0) AS user_correct_count,
               COALESCE((SELECT COUNT(*) FROM user_responses ur WHERE ur.user_id = dqa.user_id AND ur.question_id = dqa.question_id AND ur.is_correct = FALSE), 0) AS user_incorrect_count
        FROM daily_question_assignments dqa
        JOIN questions q ON dqa.question_id = q.id
        WHERE dqa.user_id = $1 AND dqa.assignment_date = $2
        ORDER BY dqa.created_at ASC
    `

	rows, err := s.db.QueryContext(ctx, query, userID, date)
	if err != nil {
		span.RecordError(err)
		return nil, contextutils.WrapError(err, "failed to query daily questions")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{
				"user_id": userID,
				"date":    date.Format("2006-01-02"),
			})
		}
	}()

	var assignments []*models.DailyQuestionAssignmentWithQuestion
	for rows.Next() {
		var assignment models.DailyQuestionAssignmentWithQuestion
		var question models.Question
		var contentJSON string

		err := rows.Scan(
			&assignment.ID, &assignment.UserID, &assignment.QuestionID, &assignment.AssignmentDate,
			&assignment.IsCompleted, &assignment.CompletedAt, &assignment.CreatedAt,
			&assignment.UserAnswerIndex, &assignment.SubmittedAt,
			&question.ID, &question.Type, &question.Language, &question.Level, &question.DifficultyScore,
			&contentJSON, &question.CorrectAnswer, &question.Explanation, &question.CreatedAt, &question.Status,
			&question.TopicCategory, &question.GrammarFocus, &question.VocabularyDomain, &question.Scenario,
			&question.StyleModifier, &question.DifficultyModifier, &question.TimeContext,
			&assignment.DailyShownCount,
			&assignment.UserTotalResponses,
			&assignment.UserCorrectCount,
			&assignment.UserIncorrectCount,
		)
		if err != nil {
			s.logger.Error(ctx, "Failed to scan daily question assignment", err, map[string]interface{}{
				"user_id": userID,
				"date":    date.Format("2006-01-02"),
			})
			continue
		}

		// Unmarshal the JSON content
		if err := question.UnmarshalContentFromJSON(contentJSON); err != nil {
			s.logger.Error(ctx, "Failed to unmarshal question content", err, map[string]interface{}{
				"user_id": userID,
				"date":    date.Format("2006-01-02"),
				"content": contentJSON,
			})
			continue
		}

		assignment.Question = &question
		assignments = append(assignments, &assignment)
	}

	if err = rows.Err(); err != nil {
		span.RecordError(err)
		return nil, contextutils.WrapError(err, "error iterating over rows")
	}

	return assignments, nil
}

// MarkQuestionCompleted marks a daily question as completed
func (s *DailyQuestionService) MarkQuestionCompleted(ctx context.Context, userID, questionID int, date time.Time) (err error) {
	ctx, span := otel.Tracer("daily-question-service").Start(ctx, "MarkQuestionCompleted",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.Int("question.id", questionID),
			attribute.String("date", date.Format("2006-01-02")),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		UPDATE daily_question_assignments
		SET is_completed = true, completed_at = $1
		WHERE user_id = $2 AND question_id = $3 AND assignment_date = $4
	`

	result, err := s.db.ExecContext(ctx, query, time.Now(), userID, questionID, date)
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to mark question as completed")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.ErrAssignmentNotFound
	}

	s.logger.Info(ctx, "Question marked as completed", map[string]interface{}{
		"user_id":     userID,
		"question_id": questionID,
		"date":        date.Format("2006-01-02"),
	})

	return nil
}

// ResetQuestionCompleted resets a daily question to not completed
func (s *DailyQuestionService) ResetQuestionCompleted(ctx context.Context, userID, questionID int, date time.Time) (err error) {
	ctx, span := otel.Tracer("daily-question-service").Start(ctx, "ResetQuestionCompleted",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.Int("question.id", questionID),
			attribute.String("date", date.Format("2006-01-02")),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		UPDATE daily_question_assignments
		SET is_completed = false, completed_at = NULL, user_answer_index = NULL, submitted_at = NULL
		WHERE user_id = $1 AND question_id = $2 AND assignment_date = $3
	`

	result, err := s.db.ExecContext(ctx, query, userID, questionID, date)
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to reset question completion")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.ErrAssignmentNotFound
	}

	s.logger.Info(ctx, "Question reset to not completed", map[string]interface{}{
		"user_id":     userID,
		"question_id": questionID,
		"date":        date.Format("2006-01-02"),
	})

	return nil
}

// GetAvailableDates retrieves all dates for which a user has daily question assignments
func (s *DailyQuestionService) GetAvailableDates(ctx context.Context, userID int) (result0 []time.Time, err error) {
	ctx, span := otel.Tracer("daily-question-service").Start(ctx, "GetAvailableDates",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT DISTINCT assignment_date
		FROM daily_question_assignments
		WHERE user_id = $1
		ORDER BY assignment_date DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		span.RecordError(err)
		return nil, contextutils.WrapError(err, "failed to query available dates")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{
				"user_id": userID,
			})
		}
	}()

	var dates []time.Time
	for rows.Next() {
		var date time.Time
		err := rows.Scan(&date)
		if err != nil {
			s.logger.Error(ctx, "Failed to scan date", err, map[string]interface{}{
				"user_id": userID,
			})
			continue
		}
		dates = append(dates, date)
	}

	if err = rows.Err(); err != nil {
		span.RecordError(err)
		return nil, contextutils.WrapError(err, "error iterating over rows")
	}

	return dates, nil
}

// GetDailyProgress retrieves the progress for a specific date
func (s *DailyQuestionService) GetDailyProgress(ctx context.Context, userID int, date time.Time) (result0 *models.DailyProgress, err error) {
	ctx, span := otel.Tracer("daily-question-service").Start(ctx, "GetDailyProgress",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.String("date", date.Format("2006-01-02")),
		),
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
			COUNT(*) as total,
			COUNT(CASE WHEN is_completed = true THEN 1 END) as completed
		FROM daily_question_assignments
		WHERE user_id = $1 AND assignment_date = $2
	`

	var total, completed int
	err = s.db.QueryRowContext(ctx, query, userID, date).Scan(&total, &completed)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get daily progress")
	}

	progress := &models.DailyProgress{
		Date:      date,
		Completed: completed,
		Total:     total,
	}

	return progress, nil
}

// GetDailyQuestionsCount retrieves the total number of questions assigned for a date
func (s *DailyQuestionService) GetDailyQuestionsCount(ctx context.Context, userID int, date time.Time) (result0 int, err error) {
	ctx, span := otel.Tracer("daily-question-service").Start(ctx, "GetDailyQuestionsCount",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.String("date", date.Format("2006-01-02")),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT COUNT(*)
		FROM daily_question_assignments
		WHERE user_id = $1 AND assignment_date = $2
	`

	var count int
	err = s.db.QueryRowContext(ctx, query, userID, date).Scan(&count)
	if err != nil {
		return 0, contextutils.WrapError(err, "failed to get daily questions count")
	}

	return count, nil
}

// GetCompletedDailyQuestionsCount retrieves the number of completed questions for a date
func (s *DailyQuestionService) GetCompletedDailyQuestionsCount(ctx context.Context, userID int, date time.Time) (result0 int, err error) {
	ctx, span := otel.Tracer("daily-question-service").Start(ctx, "GetCompletedDailyQuestionsCount",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.String("date", date.Format("2006-01-02")),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	query := `
		SELECT COUNT(*)
		FROM daily_question_assignments
		WHERE user_id = $1 AND assignment_date = $2 AND is_completed = true
	`

	var count int
	err = s.db.QueryRowContext(ctx, query, userID, date).Scan(&count)
	if err != nil {
		return 0, contextutils.WrapError(err, "failed to get completed daily questions count")
	}

	return count, nil
}

// GetQuestionHistory retrieves the history of a specific question for a user over a given number of days
func (s *DailyQuestionService) GetQuestionHistory(ctx context.Context, userID, questionID, days int) (result0 []*models.DailyQuestionHistory, err error) {
	ctx, span := otel.Tracer("daily-question-service").Start(ctx, "GetQuestionHistory",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.Int("question.id", questionID),
			attribute.Int("days", days),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	if days <= 0 {
		return nil, contextutils.ErrorWithContextf("days must be positive")
	}

	query := `
		SELECT dqa.assignment_date, dqa.is_completed, dqa.submitted_at,
		       ur.is_correct
		FROM daily_question_assignments dqa
		LEFT JOIN daily_assignment_responses dar ON dar.assignment_id = dqa.id
		LEFT JOIN user_responses ur ON ur.id = dar.user_response_id
		WHERE dqa.user_id = $1 AND dqa.question_id = $2 AND dqa.assignment_date >= NOW() - INTERVAL '` + fmt.Sprintf("%d days", days) + `'
		ORDER BY dqa.assignment_date ASC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, questionID)
	if err != nil {
		span.RecordError(err)
		return nil, contextutils.WrapError(err, "failed to query question history")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.logger.Error(ctx, "Failed to close rows", closeErr, map[string]interface{}{
				"user_id":     userID,
				"question_id": questionID,
				"days":        days,
			})
		}
	}()

	var history []*models.DailyQuestionHistory
	for rows.Next() {
		var historyEntry models.DailyQuestionHistory
		var isCorrect sql.NullBool
		err := rows.Scan(
			&historyEntry.AssignmentDate,
			&historyEntry.IsCompleted,
			&historyEntry.SubmittedAt,
			&isCorrect,
		)
		if err != nil {
			s.logger.Error(ctx, "Failed to scan question history entry", err, map[string]interface{}{
				"user_id":         userID,
				"question_id":     questionID,
				"assignment_date": historyEntry.AssignmentDate,
			})
			continue
		}
		if isCorrect.Valid {
			historyEntry.IsCorrect = &isCorrect.Bool
		} else {
			historyEntry.IsCorrect = nil
		}
		history = append(history, &historyEntry)
	}

	if err = rows.Err(); err != nil {
		span.RecordError(err)
		return nil, contextutils.WrapError(err, "error iterating over rows")
	}

	return history, nil
}

// getUserByID is a helper method to get user information
func (s *DailyQuestionService) getUserByID(ctx context.Context, userID int) (*models.User, error) {
	query := `
		SELECT id, username, email, timezone, password_hash, last_active,
		       preferred_language, current_level, ai_provider, ai_model,
		       ai_enabled, ai_api_key, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.Timezone, &user.PasswordHash,
		&user.LastActive, &user.PreferredLanguage, &user.CurrentLevel, &user.AIProvider,
		&user.AIModel, &user.AIEnabled, &user.AIAPIKey, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// SubmitDailyQuestionAnswer submits an answer for a daily question and marks it as completed
func (s *DailyQuestionService) SubmitDailyQuestionAnswer(ctx context.Context, userID, questionID int, date time.Time, userAnswerIndex int) (result *api.AnswerResponse, err error) {
	ctx, span := otel.Tracer("daily-question-service").Start(ctx, "SubmitDailyQuestionAnswer",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.Int("question.id", questionID),
			attribute.String("date", date.Format("2006-01-02")),
			attribute.Int("user_answer_index", userAnswerIndex),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	s.logger.Info(ctx, "SubmitDailyQuestionAnswer started", map[string]interface{}{
		"user_id":           userID,
		"question_id":       questionID,
		"date":              date.Format("2006-01-02"),
		"user_answer_index": userAnswerIndex,
	})

	// Check if the question is already answered
	s.logger.Info(ctx, "Checking if question is already answered", map[string]interface{}{
		"user_id":     userID,
		"question_id": questionID,
		"date":        date.Format("2006-01-02"),
	})

	query := `
		SELECT id, is_completed, user_answer_index, submitted_at
		FROM daily_question_assignments
		WHERE user_id = $1 AND question_id = $2 AND assignment_date = $3
	`

	var assignmentID int
	var isCompleted bool
	var existingUserAnswerIndex *int
	var existingSubmittedAt *time.Time

	err = s.db.QueryRowContext(ctx, query, userID, questionID, date).Scan(
		&assignmentID, &isCompleted, &existingUserAnswerIndex, &existingSubmittedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, contextutils.ErrAssignmentNotFound
		}
		return nil, contextutils.WrapError(err, "failed to check question assignment")
	}

	// Check if already answered
	if isCompleted && existingUserAnswerIndex != nil && existingSubmittedAt != nil {
		return nil, contextutils.ErrQuestionAlreadyAnswered
	}

	// Get the question details to validate answer and get correct answer
	question, err := s.questionService.GetQuestionByID(ctx, questionID)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to get question details")
	}

	if question == nil {
		return nil, contextutils.ErrQuestionNotFound
	}

	// Extract options from content map
	contentMap := question.Content
	s.logger.Info(ctx, "Question content debug", map[string]interface{}{
		"question_id": questionID,
		"content_map": contentMap,
	})

	optionsInterface, ok := contentMap["options"]
	if !ok {
		s.logger.Error(ctx, "Question content missing options", nil, map[string]interface{}{
			"question_id": questionID,
			"content_map": contentMap,
		})
		return nil, contextutils.ErrorWithContextf("question content missing options")
	}

	options, ok := optionsInterface.([]interface{})
	if !ok {
		s.logger.Error(ctx, "Invalid options format", nil, map[string]interface{}{
			"question_id":       questionID,
			"options_interface": optionsInterface,
			"options_type":      fmt.Sprintf("%T", optionsInterface),
		})
		return nil, contextutils.ErrorWithContextf("invalid options format")
	}

	// Validate user answer index
	if userAnswerIndex < 0 || userAnswerIndex >= len(options) {
		return nil, contextutils.ErrInvalidAnswerIndex
	}

	// Check if answer is correct
	isCorrect := question.CorrectAnswer == userAnswerIndex

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to begin transaction")
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Error(ctx, "Failed to rollback transaction", rollbackErr, map[string]interface{}{
					"error": rollbackErr.Error(),
				})
			}
		}
	}()

	// Update the assignment with the user's answer and mark as completed
	updateQuery := `
		UPDATE daily_question_assignments
		SET is_completed = true, completed_at = NOW(), user_answer_index = $1, submitted_at = NOW()
		WHERE id = $2
	`

	_, err = tx.ExecContext(ctx, updateQuery, userAnswerIndex, assignmentID)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to update assignment")
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to commit transaction")
	}

	// Record canonical user response via learningService so history queries see is_correct
	// Use RecordAnswerWithPriorityReturningID to obtain user_responses.id so we can link it to the assignment.
	if s.learningService != nil {
		// record synchronously so we have the response id for mapping
		respID, recErr := s.learningService.RecordAnswerWithPriorityReturningID(ctx, userID, questionID, userAnswerIndex, isCorrect, 0)
		if recErr != nil {
			s.logger.Error(ctx, "Failed to record user response for daily answer", recErr, map[string]interface{}{
				"user_id":           userID,
				"question_id":       questionID,
				"user_answer_index": userAnswerIndex,
			})
		} else {
			// Insert mapping to daily_assignment_responses synchronously so tests that run immediately can observe it
			_, mapErr := s.db.ExecContext(ctx, `
				INSERT INTO daily_assignment_responses (assignment_id, user_response_id, created_at)
				VALUES ($1, $2, NOW())
				ON CONFLICT (assignment_id) DO UPDATE SET user_response_id = EXCLUDED.user_response_id, created_at = EXCLUDED.created_at
			`, assignmentID, respID)
			if mapErr != nil {
				// Log but don't fail user's request
				s.logger.Error(ctx, "Failed to insert daily_assignment_responses mapping", mapErr, map[string]interface{}{
					"assignment_id":    assignmentID,
					"user_response_id": respID,
				})
			}

			// If the answer was correct, remove future assignments for this question within the avoid window
			if isCorrect {
				// Determine avoidDays via questionService if possible; default to 7
				avoidDays := 7
				switch qs := s.questionService.(type) {
				case interface{ getDailyRepeatAvoidDays() int }:
					avoidDays = qs.getDailyRepeatAvoidDays()
				default:
					// leave default
				}

				startDate := date.AddDate(0, 0, 1)
				endDate := date.AddDate(0, 0, avoidDays)

				deleteQuery := `DELETE FROM daily_question_assignments WHERE user_id = $1 AND question_id = $2 AND assignment_date >= $3 AND assignment_date <= $4`
				if _, delErr := s.db.ExecContext(ctx, deleteQuery, userID, questionID, startDate, endDate); delErr != nil {
					s.logger.Error(ctx, "Failed to delete future daily assignments", delErr, map[string]interface{}{
						"user_id":     userID,
						"question_id": questionID,
						"start":       startDate,
						"end":         endDate,
					})
				} else {
					// Future assignments removed successfully; worker will top up missing slots on its next run
					s.logger.Info(ctx, "Deleted future daily assignments for question; worker will refill dates as needed", map[string]interface{}{
						"user_id":     userID,
						"question_id": questionID,
						"start":       startDate,
						"end":         endDate,
					})
				}
			}
		}
	}

	// Build response
	userAnswer := options[userAnswerIndex].(string)
	response := &api.AnswerResponse{
		UserAnswerIndex: &userAnswerIndex,
		UserAnswer:      &userAnswer,
		IsCorrect:       &isCorrect,
	}

	// Add correct answer and explanation if available
	response.CorrectAnswerIndex = &question.CorrectAnswer
	if question.Explanation != "" {
		response.Explanation = &question.Explanation
	}

	s.logger.Info(ctx, "Daily question answer submitted", map[string]interface{}{
		"user_id":           userID,
		"question_id":       questionID,
		"date":              date.Format("2006-01-02"),
		"user_answer_index": userAnswerIndex,
		"is_correct":        isCorrect,
	})

	return response, nil
}
