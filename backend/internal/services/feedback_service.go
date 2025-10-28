package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"go.opentelemetry.io/otel/attribute"
)

// FeedbackService implements FeedbackServiceInterface for managing feedback reports.
type FeedbackService struct {
	db     *sql.DB
	logger *observability.Logger
}

// NewFeedbackService creates a new FeedbackService instance.
func NewFeedbackService(db *sql.DB, logger *observability.Logger) *FeedbackService {
	if db == nil {
		panic("NewFeedbackService: db is nil")
	}
	if logger == nil {
		panic("NewFeedbackService: logger is nil")
	}
	return &FeedbackService{db: db, logger: logger}
}

// CreateFeedback inserts a new feedback report.
func (s *FeedbackService) CreateFeedback(ctx context.Context, fr *models.FeedbackReport) (result0 *models.FeedbackReport, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "create_feedback")
	defer observability.FinishSpan(span, &err)

	contextJSON, err := json.Marshal(fr.ContextData)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to marshal context_data")
	}

	query := `INSERT INTO feedback_reports (user_id, feedback_text, feedback_type, context_data, screenshot_data, screenshot_url, status, created_at, updated_at)
              VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at, updated_at`
	now := time.Now()
	var id int
	var createdAt, updatedAt time.Time
	err = s.db.QueryRowContext(ctx, query, fr.UserID, fr.FeedbackText, fr.FeedbackType, contextJSON, fr.ScreenshotData, fr.ScreenshotURL, "new", now, now).
		Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, contextutils.WrapError(err, "failed to insert feedback report")
	}
	fr.ID = id
	fr.Status = "new"
	fr.CreatedAt = createdAt
	fr.UpdatedAt = updatedAt
	return fr, nil
}

// GetFeedbackByID fetches single feedback.
func (s *FeedbackService) GetFeedbackByID(ctx context.Context, id int) (result0 *models.FeedbackReport, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "get_feedback_by_id")
	defer observability.FinishSpan(span, &err)

	query := `SELECT id, user_id, feedback_text, feedback_type, context_data, screenshot_data, screenshot_url, status, admin_notes, assigned_to_user_id, resolved_at, resolved_by_user_id, created_at, updated_at FROM feedback_reports WHERE id=$1`
	row := s.db.QueryRowContext(ctx, query, id)
	var fr models.FeedbackReport
	var contextJSON []byte
	err = row.Scan(&fr.ID, &fr.UserID, &fr.FeedbackText, &fr.FeedbackType, &contextJSON, &fr.ScreenshotData, &fr.ScreenshotURL, &fr.Status, &fr.AdminNotes, &fr.AssignedToUserID, &fr.ResolvedAt, &fr.ResolvedByUserID, &fr.CreatedAt, &fr.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, contextutils.ErrRecordNotFound
		}
		return nil, contextutils.WrapError(err, "failed to scan feedback")
	}
	_ = json.Unmarshal(contextJSON, &fr.ContextData)
	return &fr, nil
}

// GetFeedbackPaginated returns list of feedback reports with filters.
func (s *FeedbackService) GetFeedbackPaginated(ctx context.Context, page, pageSize int, status, feedbackType string, userID *int) (result0 []models.FeedbackReport, result1 int, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "get_feedback_paginated")
	defer observability.FinishSpan(span, &err)

	var conditions []string
	var args []interface{}
	idx := 1
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status=$%d", idx))
		args = append(args, status)
		idx++
	}
	if feedbackType != "" {
		conditions = append(conditions, fmt.Sprintf("feedback_type=$%d", idx))
		args = append(args, feedbackType)
		idx++
	}
	if userID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id=$%d", idx))
		args = append(args, *userID)
		idx++
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM feedback_reports %s", where)
	var total int
	if err = s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, contextutils.WrapError(err, "failed to count feedback")
	}

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	query := fmt.Sprintf("SELECT id, user_id, feedback_text, feedback_type, context_data, screenshot_data, screenshot_url, status, admin_notes, assigned_to_user_id, resolved_at, resolved_by_user_id, created_at, updated_at FROM feedback_reports %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", where, idx, idx+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, contextutils.WrapError(err, "failed to query feedback list")
	}
	defer func() {
		_ = rows.Close()
	}()

	list := []models.FeedbackReport{}
	for rows.Next() {
		var fr models.FeedbackReport
		var contextJSON []byte
		if err := rows.Scan(&fr.ID, &fr.UserID, &fr.FeedbackText, &fr.FeedbackType, &contextJSON, &fr.ScreenshotData, &fr.ScreenshotURL, &fr.Status, &fr.AdminNotes, &fr.AssignedToUserID, &fr.ResolvedAt, &fr.ResolvedByUserID, &fr.CreatedAt, &fr.UpdatedAt); err != nil {
			return nil, 0, contextutils.WrapError(err, "scan feedback list")
		}
		_ = json.Unmarshal(contextJSON, &fr.ContextData)
		list = append(list, fr)
	}
	return list, total, nil
}

// UpdateFeedback allows status/notes assignment updates.
func (s *FeedbackService) UpdateFeedback(ctx context.Context, id int, updates map[string]interface{}) (result0 *models.FeedbackReport, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "update_feedback", attribute.Int("feedback.id", id))
	defer observability.FinishSpan(span, &err)

	if len(updates) == 0 {
		return s.GetFeedbackByID(ctx, id)
	}

	var sets []string
	var args []interface{}
	idx := 1
	for k, v := range updates {
		sets = append(sets, fmt.Sprintf("%s=$%d", k, idx))
		args = append(args, v)
		idx++
	}
	sets = append(sets, fmt.Sprintf("updated_at=$%d", idx))
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE feedback_reports SET %s WHERE id=$%d", strings.Join(sets, ","), idx+1)
	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return nil, contextutils.WrapError(err, "failed to update feedback")
	}
	return s.GetFeedbackByID(ctx, id)
}

// DeleteFeedback deletes a single feedback report by ID.
func (s *FeedbackService) DeleteFeedback(ctx context.Context, id int) (err error) {
	ctx, span := observability.TraceUserFunction(ctx, "delete_feedback", attribute.Int("feedback.id", id))
	defer observability.FinishSpan(span, &err)

	query := `DELETE FROM feedback_reports WHERE id=$1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return contextutils.WrapError(err, "failed to delete feedback")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return contextutils.WrapError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return contextutils.WrapErrorf(contextutils.ErrRecordNotFound, "feedback with ID %d not found", id)
	}

	return nil
}

// DeleteFeedbackByStatus deletes all feedback reports with a specific status.
func (s *FeedbackService) DeleteFeedbackByStatus(ctx context.Context, status string) (result0 int, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "delete_feedback_by_status", attribute.String("status", status))
	defer observability.FinishSpan(span, &err)

	query := `DELETE FROM feedback_reports WHERE status=$1`
	result, err := s.db.ExecContext(ctx, query, status)
	if err != nil {
		return 0, contextutils.WrapError(err, "failed to delete feedback by status")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, contextutils.WrapError(err, "failed to get rows affected")
	}

	return int(rowsAffected), nil
}

// DeleteAllFeedback deletes all feedback reports regardless of status.
func (s *FeedbackService) DeleteAllFeedback(ctx context.Context) (result0 int, err error) {
	ctx, span := observability.TraceUserFunction(ctx, "delete_all_feedback")
	defer observability.FinishSpan(span, &err)

	query := `DELETE FROM feedback_reports`
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, contextutils.WrapError(err, "failed to delete all feedback")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, contextutils.WrapError(err, "failed to get rows affected")
	}

	return int(rowsAffected), nil
}
