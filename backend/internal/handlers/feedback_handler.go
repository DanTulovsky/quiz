package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"quizapp/internal/models"
	"quizapp/internal/observability"
	serviceinterfaces "quizapp/internal/serviceinterfaces"
	contextutils "quizapp/internal/utils"
)

// FeedbackResponse represents the JSON response for feedback listing
type FeedbackResponse struct {
	ID               int                    `json:"id"`
	UserID           int                    `json:"user_id"`
	FeedbackText     string                 `json:"feedback_text"`
	FeedbackType     string                 `json:"feedback_type"`
	ContextData      map[string]interface{} `json:"context_data"`
	ScreenshotData   *string                `json:"screenshot_data"`
	ScreenshotURL    *string                `json:"screenshot_url"`
	Status           string                 `json:"status"`
	AdminNotes       *string                `json:"admin_notes"`
	AssignedToUserID *int32                 `json:"assigned_to_user_id"`
	ResolvedAt       *string                `json:"resolved_at"`
	ResolvedByUserID *int32                 `json:"resolved_by_user_id"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
}

// convertFeedbackToResponse converts FeedbackReport to FeedbackResponse
func convertFeedbackToResponse(fr models.FeedbackReport) FeedbackResponse {
	response := FeedbackResponse{
		ID:           fr.ID,
		UserID:       fr.UserID,
		FeedbackText: fr.FeedbackText,
		FeedbackType: fr.FeedbackType,
		ContextData:  fr.ContextData,
		Status:       fr.Status,
		CreatedAt:    fr.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    fr.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if fr.ScreenshotData.Valid {
		response.ScreenshotData = &fr.ScreenshotData.String
	}
	if fr.ScreenshotURL.Valid {
		response.ScreenshotURL = &fr.ScreenshotURL.String
	}
	if fr.AdminNotes.Valid {
		response.AdminNotes = &fr.AdminNotes.String
	}
	if fr.AssignedToUserID.Valid {
		response.AssignedToUserID = &fr.AssignedToUserID.Int32
	}
	if fr.ResolvedAt.Valid {
		at := fr.ResolvedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		response.ResolvedAt = &at
	}
	if fr.ResolvedByUserID.Valid {
		response.ResolvedByUserID = &fr.ResolvedByUserID.Int32
	}

	return response
}

// FeedbackHandler handles feedback report endpoints.
type FeedbackHandler struct {
	feedbackService serviceinterfaces.FeedbackServiceInterface
	logger          *observability.Logger
}

// NewFeedbackHandler creates a FeedbackHandler.
func NewFeedbackHandler(fs serviceinterfaces.FeedbackServiceInterface, logger *observability.Logger) *FeedbackHandler {
	return &FeedbackHandler{feedbackService: fs, logger: logger}
}

// FeedbackSubmissionRequest represents a POST request.
type FeedbackSubmissionRequest struct {
	FeedbackText   string                 `json:"feedback_text" binding:"required"`
	FeedbackType   string                 `json:"feedback_type"`
	ContextData    map[string]interface{} `json:"context_data"`
	ScreenshotData string                 `json:"screenshot_data"`
}

// SubmitFeedback handles POST /v1/feedback.
func (h *FeedbackHandler) SubmitFeedback(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "submit_feedback")
	defer observability.FinishSpan(span, nil)

	// Get user ID from Gin context (set by auth middleware)
	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Add user ID to Go context for service layers
	ctx = contextutils.WithUserID(ctx, userID)

	var req FeedbackSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleAppError(c, contextutils.WrapError(err, "invalid request body"))
		return
	}

	feedbackType := req.FeedbackType
	if feedbackType == "" {
		feedbackType = "general"
	}

	var screenshotData sql.NullString
	if req.ScreenshotData != "" {
		screenshotData = sql.NullString{String: req.ScreenshotData, Valid: true}
	}

	fr := &models.FeedbackReport{
		UserID:         userID,
		FeedbackText:   req.FeedbackText,
		FeedbackType:   feedbackType,
		ContextData:    req.ContextData,
		ScreenshotData: screenshotData,
		Status:         "new",
	}

	created, err := h.feedbackService.CreateFeedback(ctx, fr)
	if err != nil {
		h.logger.Error(ctx, "create feedback failed", err, nil)
		HandleAppError(c, err)
		return
	}

	c.JSON(http.StatusCreated, convertFeedbackToResponse(*created))
}

// ListFeedback handles GET /v1/admin/feedback.
func (h *FeedbackHandler) ListFeedback(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "list_feedback")
	defer observability.FinishSpan(span, nil)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	feedbackType := c.Query("feedback_type")
	userIDStr := c.Query("user_id")

	var userID *int
	if userIDStr != "" {
		id, _ := strconv.Atoi(userIDStr)
		userID = &id
	}

	list, total, err := h.feedbackService.GetFeedbackPaginated(ctx, page, pageSize, status, feedbackType, userID)
	if err != nil {
		h.logger.Error(ctx, "list feedback failed", err, nil)
		HandleAppError(c, err)
		return
	}

	// Convert each feedback item to response format
	items := make([]FeedbackResponse, len(list))
	for i, item := range list {
		items[i] = convertFeedbackToResponse(item)
	}

	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

// UpdateFeedback handles PATCH /v1/admin/feedback/:id.
func (h *FeedbackHandler) UpdateFeedback(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "update_feedback")
	defer observability.FinishSpan(span, nil)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		HandleAppError(c, contextutils.ErrorWithContextf("invalid feedback ID"))
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		HandleAppError(c, contextutils.WrapError(err, "invalid request body"))
		return
	}

	updated, err := h.feedbackService.UpdateFeedback(ctx, id, updates)
	if err != nil {
		h.logger.Error(ctx, "update feedback failed", err, nil)
		HandleAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, convertFeedbackToResponse(*updated))
}
