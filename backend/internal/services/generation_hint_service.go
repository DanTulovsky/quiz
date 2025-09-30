package services

import (
	"context"
	"database/sql"
	"time"

	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"
)

// GenerationHint represents an active generation hint
type GenerationHint struct {
	ID             int       `db:"id"`
	UserID         int       `db:"user_id"`
	Language       string    `db:"language"`
	Level          string    `db:"level"`
	QuestionType   string    `db:"question_type"`
	PriorityWeight int       `db:"priority_weight"`
	ExpiresAt      time.Time `db:"expires_at"`
	CreatedAt      time.Time `db:"created_at"`
}

// GenerationHintServiceInterface defines the API for managing generation hints
type GenerationHintServiceInterface interface {
	UpsertHint(ctx context.Context, userID int, language, level string, qType models.QuestionType, ttl time.Duration) error
	GetActiveHintsForUser(ctx context.Context, userID int) ([]GenerationHint, error)
	ClearHint(ctx context.Context, userID int, language, level string, qType models.QuestionType) error
}

// GenerationHintService implements hint management
type GenerationHintService struct {
	db     *sql.DB
	logger *observability.Logger
}

// NewGenerationHintService constructs a service for managing short-lived per-user
// generation hints that nudge the worker to prioritize specific question types
// (e.g., reading comprehension) when the user is waiting for generation.
func NewGenerationHintService(db *sql.DB, logger *observability.Logger) *GenerationHintService {
	return &GenerationHintService{db: db, logger: logger}
}

// UpsertHint creates or refreshes a hint with the given TTL
func (s *GenerationHintService) UpsertHint(ctx context.Context, userID int, language, level string, qType models.QuestionType, ttl time.Duration) (err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "upsert_generation_hint")
	defer observability.FinishSpan(span, &err)

	expiresAt := time.Now().Add(ttl)
	_, err = s.db.ExecContext(ctx, `
        INSERT INTO generation_hints (user_id, language, level, question_type, priority_weight, expires_at)
        VALUES ($1, $2, $3, $4, 1, $5)
        ON CONFLICT (user_id, language, level, question_type) DO UPDATE SET
            priority_weight = generation_hints.priority_weight + 1,
            expires_at = EXCLUDED.expires_at,
            created_at = generation_hints.created_at
    `, userID, language, level, string(qType), expiresAt)
	if err != nil {
		return contextutils.WrapError(err, "failed to upsert generation hint")
	}
	return nil
}

// GetActiveHintsForUser returns non-expired hints for the user
func (s *GenerationHintService) GetActiveHintsForUser(ctx context.Context, userID int) (result0 []GenerationHint, err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "get_active_generation_hints")
	defer observability.FinishSpan(span, &err)

	rows, err := s.db.QueryContext(ctx, `
        SELECT id, user_id, language, level, question_type, priority_weight, expires_at, created_at
        FROM generation_hints
        WHERE user_id = $1 AND expires_at > NOW()
        ORDER BY created_at ASC
    `, userID)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to query generation hints")
	}
	defer func() { _ = rows.Close() }()

	var hints []GenerationHint
	for rows.Next() {
		var h GenerationHint
		if err := rows.Scan(&h.ID, &h.UserID, &h.Language, &h.Level, &h.QuestionType, &h.PriorityWeight, &h.ExpiresAt, &h.CreatedAt); err != nil {
			return nil, contextutils.WrapError(err, "failed to scan generation hint")
		}
		hints = append(hints, h)
	}
	if err := rows.Err(); err != nil {
		return nil, contextutils.WrapError(err, "error iterating generation hints")
	}
	return hints, nil
}

// ClearHint deletes a specific hint
func (s *GenerationHintService) ClearHint(ctx context.Context, userID int, language, level string, qType models.QuestionType) (err error) {
	ctx, span := observability.TraceWorkerFunction(ctx, "clear_generation_hint")
	defer observability.FinishSpan(span, &err)

	_, err = s.db.ExecContext(ctx, `
        DELETE FROM generation_hints
        WHERE user_id = $1 AND language = $2 AND level = $3 AND question_type = $4
    `, userID, language, level, string(qType))
	if err != nil {
		return contextutils.WrapError(err, "failed to clear generation hint")
	}
	return nil
}
