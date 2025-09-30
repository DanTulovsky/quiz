package services

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"quizapp/internal/observability"
)

// CleanupService handles database maintenance and cleanup tasks
type CleanupService struct {
	db     *sql.DB
	logger *observability.Logger
}

// NewCleanupServiceWithLogger creates a new cleanup service with logger
func NewCleanupServiceWithLogger(db *sql.DB, logger *observability.Logger) *CleanupService {
	return &CleanupService{
		db:     db,
		logger: logger,
	}
}

// CleanupLegacyQuestionTypes removes questions with unsupported question types
func (c *CleanupService) CleanupLegacyQuestionTypes(ctx context.Context) (err error) {
	ctx, span := observability.TraceCleanupFunction(ctx, "cleanup_legacy_question_types")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Check if database is available
	if c.db == nil {
		return errors.New("database connection not available")
	}

	// Get count of legacy questions first
	var count int
	err = c.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM questions
		WHERE type NOT IN ('vocabulary', 'fill_blank', 'qa', 'reading_comprehension')
	`).Scan(&count)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return err
	}

	span.SetAttributes(attribute.Int("cleanup.legacy_questions_count", count))

	if count == 0 {
		c.logger.Info(ctx, "No legacy question types found to cleanup", map[string]interface{}{})
		span.SetAttributes(attribute.String("cleanup.result", "no_legacy_questions"))
		return nil
	}

	c.logger.Info(ctx, "Found questions with legacy types to cleanup", map[string]interface{}{"count": count})

	// Delete questions with unsupported types
	result, err := c.db.ExecContext(ctx, `
		DELETE FROM questions
		WHERE type NOT IN ('vocabulary', 'fill_blank', 'qa', 'reading_comprehension')
	`)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return err
	}

	span.SetAttributes(
		attribute.Int64("cleanup.rows_affected", rowsAffected),
		attribute.String("cleanup.result", "success"),
	)

	c.logger.Info(ctx, "Successfully cleaned up questions with legacy types", map[string]interface{}{"rows_affected": rowsAffected})
	return nil
}

// CleanupOrphanedResponses removes user responses for questions that no longer exist
func (c *CleanupService) CleanupOrphanedResponses(ctx context.Context) (err error) {
	ctx, span := observability.TraceCleanupFunction(ctx, "cleanup_orphaned_responses")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Check if database is available
	if c.db == nil {
		return errors.New("database connection not available")
	}

	var count int
	err = c.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM user_responses ur
		LEFT JOIN questions q ON ur.question_id = q.id
		WHERE q.id IS NULL
	`).Scan(&count)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return err
	}

	span.SetAttributes(attribute.Int("cleanup.orphaned_responses_count", count))

	if count == 0 {
		c.logger.Info(ctx, "No orphaned responses found to cleanup", map[string]interface{}{})
		span.SetAttributes(attribute.String("cleanup.result", "no_orphaned_responses"))
		return nil
	}

	c.logger.Info(ctx, "Found orphaned responses to cleanup", map[string]interface{}{"count": count})

	result, err := c.db.ExecContext(ctx, `
		DELETE FROM user_responses
		WHERE question_id NOT IN (SELECT id FROM questions)
	`)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return err
	}

	span.SetAttributes(
		attribute.Int64("cleanup.rows_affected", rowsAffected),
		attribute.String("cleanup.result", "success"),
	)

	c.logger.Info(ctx, "Successfully cleaned up orphaned responses", map[string]interface{}{"rows_affected": rowsAffected})
	return nil
}

// RunFullCleanup performs all cleanup operations
func (c *CleanupService) RunFullCleanup(ctx context.Context) (err error) {
	ctx, span := observability.TraceCleanupFunction(ctx, "run_full_cleanup")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	span.SetAttributes(attribute.String("cleanup.start_time", time.Now().Format(time.RFC3339)))

	c.logger.Info(ctx, "Starting database cleanup", map[string]interface{}{"start_time": time.Now().Format(time.RFC3339)})

	if err = c.CleanupLegacyQuestionTypes(ctx); err != nil {
		c.logger.Error(ctx, "Failed to cleanup legacy question types", err, map[string]interface{}{})
		span.SetAttributes(attribute.String("error", err.Error()))
		return err
	}

	if err := c.CleanupOrphanedResponses(ctx); err != nil {
		c.logger.Error(ctx, "Failed to cleanup orphaned responses", err, map[string]interface{}{})
		span.SetAttributes(attribute.String("error", err.Error()))
		return err
	}

	span.SetAttributes(
		attribute.String("cleanup.end_time", time.Now().Format(time.RFC3339)),
		attribute.String("cleanup.result", "success"),
	)

	c.logger.Info(ctx, "Database cleanup completed successfully", map[string]interface{}{"end_time": time.Now().Format(time.RFC3339)})
	return nil
}

// GetCleanupStats returns statistics about cleanup operations
func (c *CleanupService) GetCleanupStats(ctx context.Context) (result0 map[string]int, err error) {
	ctx, span := observability.TraceCleanupFunction(ctx, "get_cleanup_stats")
	defer func() {
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	// Check if database is available
	if c.db == nil {
		return nil, errors.New("database connection not available")
	}

	stats := make(map[string]int)

	// Count legacy question types
	var legacyCount int
	err = c.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM questions
		WHERE type NOT IN ('vocabulary', 'fill_blank', 'qa', 'reading_comprehension')
	`).Scan(&legacyCount)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, err
	}
	stats["legacy_questions"] = legacyCount

	// Count orphaned responses
	var orphanedCount int
	err = c.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM user_responses ur
		LEFT JOIN questions q ON ur.question_id = q.id
		WHERE q.id IS NULL
	`).Scan(&orphanedCount)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, err
	}
	stats["orphaned_responses"] = orphanedCount

	span.SetAttributes(
		attribute.Int("cleanup.stats.legacy_questions", legacyCount),
		attribute.Int("cleanup.stats.orphaned_responses", orphanedCount),
	)

	return stats, nil
}
