package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// WordOfTheDayServiceInterface defines the interface for word of the day operations
type WordOfTheDayServiceInterface interface {
	GetWordOfTheDay(ctx context.Context, userID int, date time.Time) (*models.WordOfTheDayDisplay, error)
	SelectWordOfTheDay(ctx context.Context, userID int, date time.Time) (*models.WordOfTheDayDisplay, error)
	GetWordHistory(ctx context.Context, userID int, startDate, endDate time.Time) ([]*models.WordOfTheDayDisplay, error)
}

// WordOfTheDayService implements word of the day operations
type WordOfTheDayService struct {
	db     *sql.DB
	logger *observability.Logger
}

// NewWordOfTheDayService creates a new WordOfTheDayService instance
func NewWordOfTheDayService(db *sql.DB, logger *observability.Logger) *WordOfTheDayService {
	return &WordOfTheDayService{
		db:     db,
		logger: logger,
	}
}

// GetWordOfTheDay retrieves the word of the day for a user and date
// If not exists, it will generate one by calling SelectWordOfTheDay
func (s *WordOfTheDayService) GetWordOfTheDay(ctx context.Context, userID int, date time.Time) (*models.WordOfTheDayDisplay, error) {
	ctx, span := otel.Tracer("word-of-the-day-service").Start(ctx, "GetWordOfTheDay",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.String("date", date.Format("2006-01-02")),
		),
	)
	defer observability.FinishSpan(span, nil)

	// Normalize date to just the date part (no time)
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	// Try to get existing word of the day
	word, err := s.getWordOfTheDayFromDB(ctx, userID, date)
	if err != nil && err != sql.ErrNoRows {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, contextutils.WrapError(err, "failed to get word of the day from database")
	}

	// If exists, return it
	if word != nil {
		span.SetAttributes(
			attribute.String("source_type", string(word.SourceType)),
			attribute.Int("source_id", word.SourceID),
		)
		return s.convertToDisplay(ctx, word)
	}

	// If not exists, generate one
	s.logger.Info(ctx, "Word of the day not found, generating new one", map[string]interface{}{
		"user_id": userID,
		"date":    date.Format("2006-01-02"),
	})

	return s.SelectWordOfTheDay(ctx, userID, date)
}

// SelectWordOfTheDay selects and assigns a word of the day for a user and date
func (s *WordOfTheDayService) SelectWordOfTheDay(ctx context.Context, userID int, date time.Time) (*models.WordOfTheDayDisplay, error) {
	ctx, span := otel.Tracer("word-of-the-day-service").Start(ctx, "SelectWordOfTheDay",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.String("date", date.Format("2006-01-02")),
		),
	)
	defer observability.FinishSpan(span, nil)

	// Normalize date to just the date part (no time)
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	// Get user preferences
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, contextutils.WrapError(err, "failed to get user")
	}

	if user == nil {
		err := contextutils.ErrorWithContextf("user not found: %d", userID)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	language := user.PreferredLanguage.String
	level := user.CurrentLevel.String

	if language == "" {
		err := contextutils.ErrorWithContextf("user missing language preference")
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.String("language", language),
		attribute.String("level", level),
	)

	// Randomly decide between vocabulary question (70%) or snippet (30%)
	useVocabulary := rand.Float32() < 0.7

	var word *models.WordOfTheDay
	if useVocabulary {
		word, err = s.selectVocabularyQuestion(ctx, userID, language, level, date)
		if err != nil || word == nil {
			s.logger.Warn(ctx, "Failed to select vocabulary question, trying snippet instead", map[string]interface{}{
				"error": err,
			})
			// Fallback to snippet
			word, err = s.selectSnippet(ctx, userID, language, date)
		}
	} else {
		word, err = s.selectSnippet(ctx, userID, language, date)
		if err != nil || word == nil {
			s.logger.Warn(ctx, "Failed to select snippet, trying vocabulary question instead", map[string]interface{}{
				"error": err,
			})
			// Fallback to vocabulary question
			word, err = s.selectVocabularyQuestion(ctx, userID, language, level, date)
		}
	}

	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, contextutils.WrapError(err, "failed to select word of the day")
	}

	if word == nil {
		err := contextutils.ErrorWithContextf("no suitable word found for user")
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Save to database
	err = s.saveWordOfTheDay(ctx, word)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, contextutils.WrapError(err, "failed to save word of the day")
	}

	span.SetAttributes(
		attribute.String("source_type", string(word.SourceType)),
		attribute.Int("source_id", word.SourceID),
	)

	s.logger.Info(ctx, "Word of the day selected", map[string]interface{}{
		"user_id":     userID,
		"date":        date.Format("2006-01-02"),
		"source_type": word.SourceType,
		"source_id":   word.SourceID,
	})

	return s.convertToDisplay(ctx, word)
}

// GetWordHistory retrieves word of the day history for a date range
func (s *WordOfTheDayService) GetWordHistory(ctx context.Context, userID int, startDate, endDate time.Time) ([]*models.WordOfTheDayDisplay, error) {
	ctx, span := otel.Tracer("word-of-the-day-service").Start(ctx, "GetWordHistory",
		trace.WithAttributes(
			attribute.Int("user.id", userID),
			attribute.String("start_date", startDate.Format("2006-01-02")),
			attribute.String("end_date", endDate.Format("2006-01-02")),
		),
	)
	defer observability.FinishSpan(span, nil)

	// Normalize dates
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.UTC)

	query := `
		SELECT id, user_id, assignment_date, source_type, source_id, created_at
		FROM word_of_the_day
		WHERE user_id = $1 AND assignment_date >= $2 AND assignment_date <= $3
		ORDER BY assignment_date DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, startDate, endDate)
	if err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, contextutils.WrapError(err, "failed to query word history")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			span.RecordError(closeErr, trace.WithStackTrace(true))
			s.logger.Warn(ctx, "Failed to close rows", map[string]interface{}{"error": closeErr.Error()})
		}
	}()

	var words []*models.WordOfTheDay
	for rows.Next() {
		var w models.WordOfTheDay
		err := rows.Scan(&w.ID, &w.UserID, &w.AssignmentDate, &w.SourceType, &w.SourceID, &w.CreatedAt)
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
			return nil, contextutils.WrapError(err, "failed to scan word row")
		}
		words = append(words, &w)
	}

	if err = rows.Err(); err != nil {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, contextutils.WrapError(err, "error iterating word rows")
	}

	// Convert to display format
	var displays []*models.WordOfTheDayDisplay
	for _, w := range words {
		display, err := s.convertToDisplay(ctx, w)
		if err != nil {
			s.logger.Error(ctx, "Failed to convert word to display", err, map[string]interface{}{
				"word_id":     w.ID,
				"source_type": w.SourceType,
				"source_id":   w.SourceID,
			})
			continue
		}
		displays = append(displays, display)
	}

	span.SetAttributes(attribute.Int("count", len(displays)))

	return displays, nil
}

// selectVocabularyQuestion selects a vocabulary question for word of the day
func (s *WordOfTheDayService) selectVocabularyQuestion(ctx context.Context, userID int, language, level string, date time.Time) (*models.WordOfTheDay, error) {
	ctx, span := otel.Tracer("word-of-the-day-service").Start(ctx, "selectVocabularyQuestion")
	defer observability.FinishSpan(span, nil)

	// Query for vocabulary questions that haven't been used as word of the day recently
	query := `
		SELECT q.id
		FROM questions q
		WHERE q.type = 'vocabulary'
		  AND q.language = $1
		  AND q.status = 'active'
		  AND ($2 = '' OR q.level = $2)
		  AND NOT EXISTS (
			SELECT 1 FROM word_of_the_day wotd
			WHERE wotd.user_id = $3
			  AND wotd.source_type = 'vocabulary_question'
			  AND wotd.source_id = q.id
			  AND wotd.assignment_date > $4
		  )
		ORDER BY RANDOM()
		LIMIT 1
	`

	// Don't reuse words from the last 60 days
	cutoffDate := date.AddDate(0, 0, -60)

	var questionID int
	err := s.db.QueryRowContext(ctx, query, language, level, userID, cutoffDate).Scan(&questionID)
	if err == sql.ErrNoRows {
		// Try without the recency check
		queryNoRecency := `
			SELECT q.id
			FROM questions q
			WHERE q.type = 'vocabulary'
			  AND q.language = $1
			  AND q.status = 'active'
			  AND ($2 = '' OR q.level = $2)
			ORDER BY RANDOM()
			LIMIT 1
		`
		err = s.db.QueryRowContext(ctx, queryNoRecency, language, level).Scan(&questionID)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No vocabulary questions available
		}
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, contextutils.WrapError(err, "failed to query vocabulary question")
	}

	return &models.WordOfTheDay{
		UserID:         userID,
		AssignmentDate: date,
		SourceType:     models.WordSourceVocabularyQuestion,
		SourceID:       questionID,
	}, nil
}

// selectSnippet selects a user snippet for word of the day
func (s *WordOfTheDayService) selectSnippet(ctx context.Context, userID int, language string, date time.Time) (*models.WordOfTheDay, error) {
	ctx, span := otel.Tracer("word-of-the-day-service").Start(ctx, "selectSnippet")
	defer observability.FinishSpan(span, nil)

	// Query for user's snippets that haven't been used as word of the day recently
	// Prefer more recent snippets (created in last 30 days)
	query := `
		SELECT s.id
		FROM snippets s
		WHERE s.user_id = $1
		  AND s.source_language = $2
		  AND NOT EXISTS (
			SELECT 1 FROM word_of_the_day wotd
			WHERE wotd.user_id = $1
			  AND wotd.source_type = 'snippet'
			  AND wotd.source_id = s.id
			  AND wotd.assignment_date > $3
		  )
		ORDER BY
		  CASE WHEN s.created_at > $4 THEN 0 ELSE 1 END,
		  RANDOM()
		LIMIT 1
	`

	// Don't reuse snippets from the last 60 days
	cutoffDate := date.AddDate(0, 0, -60)
	// Prefer snippets from the last 30 days
	recentCutoff := date.AddDate(0, 0, -30)

	var snippetID int
	err := s.db.QueryRowContext(ctx, query, userID, language, cutoffDate, recentCutoff).Scan(&snippetID)
	if err == sql.ErrNoRows {
		// Try without the recency check
		queryNoRecency := `
			SELECT s.id
			FROM snippets s
			WHERE s.user_id = $1
			  AND s.source_language = $2
			ORDER BY RANDOM()
			LIMIT 1
		`
		err = s.db.QueryRowContext(ctx, queryNoRecency, userID, language).Scan(&snippetID)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No snippets available
		}
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return nil, contextutils.WrapError(err, "failed to query snippet")
	}

	return &models.WordOfTheDay{
		UserID:         userID,
		AssignmentDate: date,
		SourceType:     models.WordSourceSnippet,
		SourceID:       snippetID,
	}, nil
}

// getWordOfTheDayFromDB retrieves a word of the day from the database
func (s *WordOfTheDayService) getWordOfTheDayFromDB(ctx context.Context, userID int, date time.Time) (*models.WordOfTheDay, error) {
	query := `
		SELECT id, user_id, assignment_date, source_type, source_id, created_at
		FROM word_of_the_day
		WHERE user_id = $1 AND assignment_date = $2
	`

	var w models.WordOfTheDay
	err := s.db.QueryRowContext(ctx, query, userID, date).Scan(
		&w.ID, &w.UserID, &w.AssignmentDate, &w.SourceType, &w.SourceID, &w.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}

	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query word of the day")
	}

	return &w, nil
}

// saveWordOfTheDay saves a word of the day to the database
func (s *WordOfTheDayService) saveWordOfTheDay(ctx context.Context, word *models.WordOfTheDay) error {
	query := `
		INSERT INTO word_of_the_day (user_id, assignment_date, source_type, source_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, assignment_date) DO NOTHING
		RETURNING id
	`

	err := s.db.QueryRowContext(ctx, query,
		word.UserID,
		word.AssignmentDate,
		word.SourceType,
		word.SourceID,
		time.Now(),
	).Scan(&word.ID)
	if err != nil {
		return contextutils.WrapError(err, "failed to insert word of the day")
	}

	return nil
}

// convertToDisplay converts a WordOfTheDay to WordOfTheDayDisplay format
func (s *WordOfTheDayService) convertToDisplay(ctx context.Context, word *models.WordOfTheDay) (*models.WordOfTheDayDisplay, error) {
	ctx, span := otel.Tracer("word-of-the-day-service").Start(ctx, "convertToDisplay")
	defer observability.FinishSpan(span, nil)

	display := &models.WordOfTheDayDisplay{
		Date:       word.AssignmentDate,
		SourceType: word.SourceType,
		SourceID:   word.SourceID,
	}

	switch word.SourceType {
	case models.WordSourceVocabularyQuestion:
		question, err := s.getQuestionByID(ctx, word.SourceID)
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
			return nil, contextutils.WrapError(err, "failed to get question")
		}

		// Extract word, translation, and sentence from question content
		content := question.Content
		if sentenceRaw, ok := content["sentence"]; ok {
			display.Sentence = fmt.Sprintf("%v", sentenceRaw)
		}
		if questionRaw, ok := content["question"]; ok {
			display.Word = fmt.Sprintf("%v", questionRaw)
		}
		if optionsRaw, ok := content["options"]; ok {
			if options, ok := optionsRaw.([]interface{}); ok && len(options) > question.CorrectAnswer {
				display.Translation = fmt.Sprintf("%v", options[question.CorrectAnswer])
			}
		}

		display.Language = question.Language
		display.Level = question.Level
		display.Explanation = question.Explanation
		display.TopicCategory = question.TopicCategory

	case models.WordSourceSnippet:
		snippet, err := s.getSnippetByID(ctx, word.SourceID)
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
			return nil, contextutils.WrapError(err, "failed to get snippet")
		}

		display.Word = snippet.OriginalText
		display.Translation = snippet.TranslatedText
		display.Language = snippet.SourceLanguage
		if snippet.Context != nil {
			display.Context = *snippet.Context
			display.Sentence = *snippet.Context
		}
		if snippet.DifficultyLevel != nil {
			display.Level = *snippet.DifficultyLevel
		}
	}

	return display, nil
}

// getUserByID retrieves a user by ID
func (s *WordOfTheDayService) getUserByID(ctx context.Context, userID int) (*models.User, error) {
	query := `
		SELECT id, username, email, preferred_language, current_level, timezone
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PreferredLanguage,
		&user.CurrentLevel,
		&user.Timezone,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query user")
	}

	return &user, nil
}

// getQuestionByID retrieves a question by ID
func (s *WordOfTheDayService) getQuestionByID(ctx context.Context, questionID int) (*models.Question, error) {
	query := `
		SELECT id, type, language, level, difficulty_score, content, correct_answer,
		       explanation, created_at, status, topic_category, grammar_focus,
		       vocabulary_domain, scenario, style_modifier, difficulty_modifier, time_context
		FROM questions
		WHERE id = $1
	`

	var question models.Question
	var contentJSON []byte

	err := s.db.QueryRowContext(ctx, query, questionID).Scan(
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
		&question.TopicCategory,
		&question.GrammarFocus,
		&question.VocabularyDomain,
		&question.Scenario,
		&question.StyleModifier,
		&question.DifficultyModifier,
		&question.TimeContext,
	)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query question")
	}

	// Parse JSON content
	content := make(map[string]interface{})
	if err := json.Unmarshal(contentJSON, &content); err != nil {
		return nil, contextutils.WrapError(err, "failed to parse question content")
	}
	question.Content = content

	return &question, nil
}

// getSnippetByID retrieves a snippet by ID
func (s *WordOfTheDayService) getSnippetByID(ctx context.Context, snippetID int) (*models.Snippet, error) {
	query := `
		SELECT id, user_id, original_text, translated_text, source_language,
		       target_language, question_id, section_id, story_id, context,
		       difficulty_level, created_at, updated_at
		FROM snippets
		WHERE id = $1
	`

	var snippet models.Snippet
	err := s.db.QueryRowContext(ctx, query, snippetID).Scan(
		&snippet.ID,
		&snippet.UserID,
		&snippet.OriginalText,
		&snippet.TranslatedText,
		&snippet.SourceLanguage,
		&snippet.TargetLanguage,
		&snippet.QuestionID,
		&snippet.SectionID,
		&snippet.StoryID,
		&snippet.Context,
		&snippet.DifficultyLevel,
		&snippet.CreatedAt,
		&snippet.UpdatedAt,
	)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query snippet")
	}

	return &snippet, nil
}
