package serviceinterfaces

import (
	"context"

	"quizapp/internal/models"
)

// FeedbackServiceInterface defines operations for feedback reports.
type FeedbackServiceInterface interface {
	CreateFeedback(ctx context.Context, fr *models.FeedbackReport) (*models.FeedbackReport, error)
	GetFeedbackByID(ctx context.Context, id int) (*models.FeedbackReport, error)
	GetFeedbackPaginated(ctx context.Context, page, pageSize int, status, feedbackType string, userID *int) ([]models.FeedbackReport, int, error)
	UpdateFeedback(ctx context.Context, id int, updates map[string]interface{}) (*models.FeedbackReport, error)
}
