package services

import (
	"context"
	"database/sql"
	"fmt"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/serviceinterfaces"
	contextutils "quizapp/internal/utils"
)

// SnippetsServiceInterface defines the interface for snippets services
type SnippetsServiceInterface = serviceinterfaces.SnippetsService

// SnippetsService handles snippets related business logic
type SnippetsService struct {
	db     *sql.DB
	cfg    *config.Config
	logger *observability.Logger
}

// NewSnippetsService creates a new SnippetsService instance
func NewSnippetsService(db *sql.DB, cfg *config.Config, logger *observability.Logger) *SnippetsService {
	return &SnippetsService{
		db:     db,
		cfg:    cfg,
		logger: logger,
	}
}

// getDefaultDifficultyLevel returns a sensible default difficulty level when no question context is available
func (s *SnippetsService) getDefaultDifficultyLevel() string {
	// Default to B1 (Intermediate) as a reasonable middle ground for vocabulary snippets
	// Users can always update this through the UI if needed
	return "Unknown"
}

// getQuestionLevel retrieves the difficulty level of a specific question
func (s *SnippetsService) getQuestionLevel(ctx context.Context, questionID int64) (result string, err error) {
	ctx, span := observability.TraceQuestionFunction(ctx, "get_question_level",
		observability.AttributeQuestionID(int(questionID)),
	)
	defer observability.FinishSpan(span, &err)

	query := `SELECT level FROM questions WHERE id = $1`

	err = s.db.QueryRowContext(ctx, query, questionID).Scan(&result)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", contextutils.WrapErrorf(contextutils.ErrRecordNotFound, "question with id %d not found", questionID)
		}
		return "", contextutils.WrapErrorf(err, "failed to get question level for question %d", questionID)
	}
	return result, nil
}

// CreateSnippet creates a new vocabulary snippet
func (s *SnippetsService) CreateSnippet(ctx context.Context, userID int64, req api.CreateSnippetRequest) (result *models.Snippet, err error) {
	ctx, span := observability.TraceFunction(ctx, "snippets", "create_snippet")
	defer observability.FinishSpan(span, &err)

	span.SetAttributes(observability.AttributeUserID(int(userID)))

	// Check if snippet already exists for this user and text combination
	exists, err := s.snippetExists(ctx, userID, req.OriginalText, req.SourceLanguage, req.TargetLanguage)
	if err != nil {
		return nil, fmt.Errorf("failed to check snippet existence: %w", err)
	}
	if exists {
		return nil, contextutils.WrapError(contextutils.ErrRecordExists, "snippet already exists for this user and text combination")
	}

	// Determine difficulty level - use question's level if question_id is provided
	var difficultyLevel string
	var levelSource string

	if req.QuestionId != nil {
		// Get the question's difficulty level
		questionLevel, err := s.getQuestionLevel(ctx, *req.QuestionId)
		if err != nil {
			// If we can't get the question level, use default
			s.logger.Warn(ctx, "Failed to get question level, using default",
				map[string]any{"question_id": *req.QuestionId, "error": err.Error()})
			difficultyLevel = s.getDefaultDifficultyLevel()
			levelSource = "default_fallback"
		} else {
			difficultyLevel = questionLevel
			levelSource = "question"
		}
	} else {
		// No question context, use default
		difficultyLevel = s.getDefaultDifficultyLevel()
		levelSource = "default"
	}
	span.SetAttributes(observability.AttributeLevel(difficultyLevel))

	// Insert new snippet
	query := `
		INSERT INTO snippets (user_id, original_text, translated_text, source_language, target_language, question_id, context, difficulty_level)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	err = s.db.QueryRowContext(ctx, query,
		userID,
		req.OriginalText,
		req.TranslatedText,
		req.SourceLanguage,
		req.TargetLanguage,
		req.QuestionId,
		req.Context,
		difficultyLevel,
	).Scan(&result.ID, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to create snippet")
	}

	// Set the remaining fields
	result.UserID = userID
	result.OriginalText = req.OriginalText
	result.TranslatedText = req.TranslatedText
	result.SourceLanguage = req.SourceLanguage
	result.TargetLanguage = req.TargetLanguage
	result.QuestionID = req.QuestionId
	result.Context = req.Context

	s.logger.Info(ctx, "Created new snippet",
		map[string]any{
			"snippet_id":       result.ID,
			"user_id":          userID,
			"original_text":    req.OriginalText,
			"source_language":  req.SourceLanguage,
			"difficulty_level": difficultyLevel,
			"level_source":     levelSource,
			"question_id":      req.QuestionId,
		})

	return result, nil
}

// GetSnippets retrieves snippets for a user with optional filtering
func (s *SnippetsService) GetSnippets(ctx context.Context, userID int64, params api.GetV1SnippetsParams) (result *api.SnippetList, err error) {
	ctx, span := observability.TraceFunction(ctx, "snippets", "get_snippets")
	defer observability.FinishSpan(span, &err)

	span.SetAttributes(observability.AttributeUserID(int(userID)))

	query := `
		SELECT id, user_id, original_text, translated_text, source_language, target_language,
		       question_id, context, difficulty_level, created_at, updated_at
		FROM snippets
		WHERE user_id = $1`

	args := []any{userID}
	argCount := 1

	// Add search filter if provided
	if params.Q != nil && *params.Q != "" {
		argCount++
		query += fmt.Sprintf(" AND (original_text ILIKE $%d OR translated_text ILIKE $%d)", argCount, argCount)
		searchTerm := "%" + *params.Q + "%"
		args = append(args, searchTerm)
	}

	// Add source language filter if provided
	if params.SourceLang != nil && *params.SourceLang != "" {
		argCount++
		query += fmt.Sprintf(" AND source_language = $%d", argCount)
		args = append(args, *params.SourceLang)
	}

	// Add target language filter if provided
	if params.TargetLang != nil && *params.TargetLang != "" {
		argCount++
		query += fmt.Sprintf(" AND target_language = $%d", argCount)
		args = append(args, *params.TargetLang)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"

	if params.Limit != nil && *params.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		limit := *params.Limit
		if limit > 100 { // Max limit
			limit = 100
		}
		args = append(args, limit)
	}

	if params.Offset != nil && *params.Offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, *params.Offset)
	}

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to query snippets")
	}
	defer rows.Close()

	var snippets []api.Snippet
	for rows.Next() {
		var snippet models.Snippet
		err := rows.Scan(
			&snippet.ID,
			&snippet.UserID,
			&snippet.OriginalText,
			&snippet.TranslatedText,
			&snippet.SourceLanguage,
			&snippet.TargetLanguage,
			&snippet.QuestionID,
			&snippet.Context,
			&snippet.DifficultyLevel,
			&snippet.CreatedAt,
			&snippet.UpdatedAt,
		)
		if err != nil {
			return nil, contextutils.WrapErrorf(err, "failed to scan snippet")
		}

		snippets = append(snippets, api.Snippet{
			Id:              &snippet.ID,
			UserId:          &snippet.UserID,
			OriginalText:    &snippet.OriginalText,
			TranslatedText:  &snippet.TranslatedText,
			SourceLanguage:  &snippet.SourceLanguage,
			TargetLanguage:  &snippet.TargetLanguage,
			QuestionId:      snippet.QuestionID,
			Context:         snippet.Context,
			DifficultyLevel: snippet.DifficultyLevel,
			CreatedAt:       &snippet.CreatedAt,
			UpdatedAt:       &snippet.UpdatedAt,
		})
	}

	// Get total count for pagination info
	totalQuery := "SELECT COUNT(*) FROM snippets WHERE user_id = $1"
	totalArgs := []interface{}{userID}

	// Apply the same filters for total count
	if params.Q != nil && *params.Q != "" {
		totalQuery += " AND (original_text ILIKE $2 OR translated_text ILIKE $2)"
		totalArgs = append(totalArgs, "%"+*params.Q+"%")
	}
	if params.SourceLang != nil && *params.SourceLang != "" {
		totalQuery += fmt.Sprintf(" AND source_language = $%d", len(totalArgs)+1)
		totalArgs = append(totalArgs, *params.SourceLang)
	}
	if params.TargetLang != nil && *params.TargetLang != "" {
		totalQuery += fmt.Sprintf(" AND target_language = $%d", len(totalArgs)+1)
		totalArgs = append(totalArgs, *params.TargetLang)
	}

	var total int
	err = s.db.QueryRowContext(ctx, totalQuery, totalArgs...).Scan(&total)
	if err != nil {
		return nil, contextutils.WrapErrorf(err, "failed to get total count")
	}

	// Build response
	limit := 50 // default
	offset := 0 // default
	if params.Limit != nil {
		limit = *params.Limit
	}
	if params.Offset != nil {
		offset = *params.Offset
	}

	result = &api.SnippetList{
		Snippets: &snippets,
		Total:    &total,
		Limit:    &limit,
		Offset:   &offset,
		Query:    params.Q,
	}
	return result, nil
}

// GetSnippet retrieves a specific snippet by ID
func (s *SnippetsService) GetSnippet(ctx context.Context, userID, snippetID int64) (result *models.Snippet, err error) {
	ctx, span := observability.TraceFunction(ctx, "snippets", "get_snippet")
	defer observability.FinishSpan(span, &err)

	span.SetAttributes(observability.AttributeUserID(int(userID)))
	span.SetAttributes(observability.AttributeSnippetID(int(snippetID)))

	query := `
		SELECT id, user_id, original_text, translated_text, source_language, target_language,
		       question_id, context, difficulty_level, created_at, updated_at
		FROM snippets
		WHERE id = $1 AND user_id = $2`

	err = s.db.QueryRowContext(ctx, query, snippetID, userID).Scan(
		&result.ID,
		&result.UserID,
		&result.OriginalText,
		&result.TranslatedText,
		&result.SourceLanguage,
		&result.TargetLanguage,
		&result.QuestionID,
		&result.Context,
		&result.DifficultyLevel,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, contextutils.WrapError(contextutils.ErrRecordNotFound, "snippet not found")
		}
		return nil, contextutils.WrapErrorf(err, "failed to get snippet")
	}

	return result, nil
}

// UpdateSnippet updates a snippet's context
func (s *SnippetsService) UpdateSnippet(ctx context.Context, userID, snippetID int64, req api.UpdateSnippetRequest) (result *models.Snippet, err error) {
	ctx, span := observability.TraceFunction(ctx, "snippets", "update_snippet")
	defer observability.FinishSpan(span, &err)

	span.SetAttributes(observability.AttributeUserID(int(userID)))
	span.SetAttributes(observability.AttributeSnippetID(int(snippetID)))

	query := `
		UPDATE snippets
		SET context = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND user_id = $3
		RETURNING id, user_id, original_text, translated_text, source_language, target_language,
		          question_id, context, difficulty_level, created_at, updated_at`

	err = s.db.QueryRowContext(ctx, query, req.Context, snippetID, userID).Scan(
		&result.ID,
		&result.UserID,
		&result.OriginalText,
		&result.TranslatedText,
		&result.SourceLanguage,
		&result.TargetLanguage,
		&result.QuestionID,
		&result.Context,
		&result.DifficultyLevel,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, contextutils.WrapError(contextutils.ErrRecordNotFound, "snippet not found")
		}
		return nil, contextutils.WrapErrorf(err, "failed to update snippet")
	}

	s.logger.Info(ctx, "Updated snippet",
		map[string]any{
			"snippet_id": result.ID,
			"user_id":    userID,
		})

	return result, nil
}

// DeleteSnippet deletes a snippet
func (s *SnippetsService) DeleteSnippet(ctx context.Context, userID, snippetID int64) (err error) {
	ctx, span := observability.TraceFunction(ctx, "snippets", "delete_snippet")
	defer observability.FinishSpan(span, &err)

	span.SetAttributes(observability.AttributeUserID(int(userID)))
	span.SetAttributes(observability.AttributeSnippetID(int(snippetID)))

	result, err := s.db.ExecContext(ctx, "DELETE FROM snippets WHERE id = $1 AND user_id = $2", snippetID, userID)
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to delete snippet")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapErrorf(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.WrapError(contextutils.ErrRecordNotFound, "snippet not found")
	}

	s.logger.Info(ctx, "Deleted snippet",
		map[string]any{
			"snippet_id": snippetID,
			"user_id":    userID,
		})

	return nil
}

// snippetExists checks if a snippet already exists for the given user and text combination
func (s *SnippetsService) snippetExists(ctx context.Context, userID int64, originalText, sourceLang, targetLang string) (result bool, err error) {
	ctx, span := observability.TraceFunction(ctx, "snippets", "snippet_exists")
	defer observability.FinishSpan(span, &err)

	span.SetAttributes(observability.AttributeUserID(int(userID)))
	span.SetAttributes(observability.AttributeLanguage(sourceLang))
	span.SetAttributes(observability.AttributeLanguage(targetLang))

	var count int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM snippets
		WHERE user_id = $1 AND original_text = $2 AND source_language = $3 AND target_language = $4`,
		userID, originalText, sourceLang, targetLang).Scan(&count)
	if err != nil {
		return false, contextutils.WrapErrorf(err, "failed to check snippet existence")
	}

	result = count > 0
	return result, nil
}
