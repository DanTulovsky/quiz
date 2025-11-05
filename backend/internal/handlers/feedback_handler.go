package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	serviceinterfaces "quizapp/internal/serviceinterfaces"
	"quizapp/internal/services"
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

// ensureContextDataNotNull returns an empty map if the input is nil
func ensureContextDataNotNull(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return map[string]interface{}{}
	}
	return data
}

// convertFeedbackToResponse converts FeedbackReport to FeedbackResponse
func convertFeedbackToResponse(fr models.FeedbackReport) FeedbackResponse {
	response := FeedbackResponse{
		ID:           fr.ID,
		UserID:       fr.UserID,
		FeedbackText: fr.FeedbackText,
		FeedbackType: fr.FeedbackType,
		ContextData:  ensureContextDataNotNull(fr.ContextData),
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
	linearService   *services.LinearService
	userService     services.UserServiceInterface
	config          *config.Config
	logger          *observability.Logger
}

// NewFeedbackHandler creates a FeedbackHandler.
func NewFeedbackHandler(fs serviceinterfaces.FeedbackServiceInterface, linearService *services.LinearService, userService services.UserServiceInterface, cfg *config.Config, logger *observability.Logger) *FeedbackHandler {
	return &FeedbackHandler{
		feedbackService: fs,
		linearService:   linearService,
		userService:     userService,
		config:          cfg,
		logger:          logger,
	}
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
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request body",
			"",
			err,
		))
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

// GetFeedback handles GET /v1/admin/backend/feedback/:id.
func (h *FeedbackHandler) GetFeedback(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_feedback")
	defer observability.FinishSpan(span, nil)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	feedback, err := h.feedbackService.GetFeedbackByID(ctx, id)
	if err != nil {
		if contextutils.IsError(err, contextutils.ErrRecordNotFound) {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}
		h.logger.Error(ctx, "get feedback failed", err, nil)
		HandleAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, convertFeedbackToResponse(*feedback))
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

// DeleteFeedback handles DELETE /v1/admin/backend/feedback/:id.
func (h *FeedbackHandler) DeleteFeedback(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "delete_feedback")
	defer observability.FinishSpan(span, nil)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		HandleAppError(c, contextutils.ErrorWithContextf("invalid feedback ID"))
		return
	}

	err = h.feedbackService.DeleteFeedback(ctx, id)
	if err != nil {
		h.logger.Error(ctx, "delete feedback failed", err, nil)
		HandleAppError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// DeleteFeedbackByStatus handles DELETE /v1/admin/backend/feedback?status=resolved.
func (h *FeedbackHandler) DeleteFeedbackByStatus(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "delete_feedback_by_status")
	defer observability.FinishSpan(span, nil)

	status := c.Query("status")
	if status == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	count, err := h.feedbackService.DeleteFeedbackByStatus(ctx, status)
	if err != nil {
		h.logger.Error(ctx, "delete feedback by status failed", err, nil)
		HandleAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted_count": count})
}

// DeleteAllFeedback handles DELETE /v1/admin/backend/feedback?all=true.
func (h *FeedbackHandler) DeleteAllFeedback(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "delete_all_feedback")
	defer observability.FinishSpan(span, nil)

	count, err := h.feedbackService.DeleteAllFeedback(ctx)
	if err != nil {
		h.logger.Error(ctx, "delete all feedback failed", err, nil)
		HandleAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted_count": count})
}

// CreateLinearIssueResponse represents the response for creating a Linear issue
type CreateLinearIssueResponse struct {
	IssueID  string `json:"issue_id"`
	IssueURL string `json:"issue_url"`
	Title    string `json:"title"`
}

// CreateLinearIssue handles POST /v1/admin/backend/feedback/:id/linear-issue.
func (h *FeedbackHandler) CreateLinearIssue(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "create_linear_issue")
	defer observability.FinishSpan(span, nil)

	if h.linearService == nil {
		HandleAppError(c, contextutils.NewAppError(
			contextutils.ErrorCodeServiceUnavailable,
			contextutils.SeverityError,
			"Linear integration is not available",
			"",
		))
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Get feedback by ID
	feedback, err := h.feedbackService.GetFeedbackByID(ctx, id)
	if err != nil {
		if contextutils.IsError(err, contextutils.ErrRecordNotFound) {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}
		h.logger.Error(ctx, "get feedback failed", err, nil)
		HandleAppError(c, err)
		return
	}

	// Format title - only include feedback type and number
	title := fmt.Sprintf("[Feedback #%d] %s", feedback.ID, getTypeLabel(feedback.FeedbackType))

	// Get username and user for metadata
	username := fmt.Sprintf("User %d", feedback.UserID)
	var user *models.User
	if h.userService != nil {
		user, err = h.userService.GetUserByID(ctx, feedback.UserID)
		if err == nil && user != nil {
			username = user.Username
		}
	}

	// Build description with feedback details
	var descriptionBuilder strings.Builder
	descriptionBuilder.WriteString(feedback.FeedbackText)
	descriptionBuilder.WriteString("\n\n")

	descriptionBuilder.WriteString("### Metadata\n\n")
	descriptionBuilder.WriteString(fmt.Sprintf("- **Type**: %s\n", getTypeLabel(feedback.FeedbackType)))
	descriptionBuilder.WriteString(fmt.Sprintf("- **Status**: %s\n", feedback.Status))
	descriptionBuilder.WriteString(fmt.Sprintf("- **User ID**: %d\n", feedback.UserID))
	descriptionBuilder.WriteString(fmt.Sprintf("- **Username**: %s\n", username))
	descriptionBuilder.WriteString(fmt.Sprintf("- **Feedback ID**: %d\n", feedback.ID))
	// Format created timestamp in user's timezone
	createdFormatted := feedback.CreatedAt.Format("January 2, 2006 at 3:04 PM")
	timezoneLabel := "UTC"
	if h.userService != nil {
		if formatted, tz, err := contextutils.FormatTimeInUserTimezone(ctx, feedback.UserID, feedback.CreatedAt, "January 2, 2006 at 3:04 PM", h.userService.GetUserByID); err == nil {
			createdFormatted = formatted
			timezoneLabel = tz
		}
	}
	descriptionBuilder.WriteString(fmt.Sprintf("- **Created**: %s (%s)\n", createdFormatted, timezoneLabel))

	if feedback.AdminNotes.Valid && feedback.AdminNotes.String != "" {
		descriptionBuilder.WriteString(fmt.Sprintf("- **Admin Notes**: %s\n", feedback.AdminNotes.String))
	}

	// Add context data if available
	if len(feedback.ContextData) > 0 {
		descriptionBuilder.WriteString("\n### Context Data\n\n")
		for key, value := range feedback.ContextData {
			switch key {
			case "page_url":
				// Handle page_url specially - make it a full URL if it's a relative path
				pageURL := fmt.Sprintf("%v", value)
				if strings.HasPrefix(pageURL, "/") {
					// It's a relative path, construct full URL
					// Try to get base URL from config first
					baseURL := ""
					if h.config != nil && h.config.Server.AppBaseURL != "" {
						baseURL = h.config.Server.AppBaseURL
					}
					// Fallback to request headers if config not available
					if baseURL == "" {
						baseURL = c.Request.Header.Get("Origin")
					}
					if baseURL == "" {
						baseURL = c.Request.Header.Get("Referer")
						if baseURL != "" {
							// Extract base URL from referer (protocol + host)
							// Find the first "/" after the protocol
							if schemeIdx := strings.Index(baseURL, "://"); schemeIdx > 0 {
								if pathIdx := strings.Index(baseURL[schemeIdx+3:], "/"); pathIdx > 0 {
									baseURL = baseURL[:schemeIdx+3+pathIdx]
								}
							}
						}
					}
					// Remove trailing slash if present
					if baseURL != "" {
						baseURL = strings.TrimSuffix(baseURL, "/")
						descriptionBuilder.WriteString(fmt.Sprintf("- **%s**: %s%s\n", key, baseURL, pageURL))
					} else {
						// If we can't determine base URL, just use the relative path
						descriptionBuilder.WriteString(fmt.Sprintf("- **%s**: %s\n", key, pageURL))
					}
				} else {
					descriptionBuilder.WriteString(fmt.Sprintf("- **%s**: %s\n", key, pageURL))
				}
			case "timestamp":
				// Format timestamp as human readable in user's timezone
				if tsStr, ok := value.(string); ok {
					if ts, err := time.Parse(time.RFC3339, tsStr); err == nil {
						// Convert to user's timezone
						formatted := ts.Format("January 2, 2006 at 3:04 PM")
						timezoneLabel := "UTC"
						if h.userService != nil {
							if fmtTime, tz, err := contextutils.FormatTimeInUserTimezone(ctx, feedback.UserID, ts, "January 2, 2006 at 3:04 PM", h.userService.GetUserByID); err == nil {
								formatted = fmtTime
								timezoneLabel = tz
							}
						}
						descriptionBuilder.WriteString(fmt.Sprintf("- **%s**: %s (%s)\n", key, formatted, timezoneLabel))
					} else {
						descriptionBuilder.WriteString(fmt.Sprintf("- **%s**: %v\n", key, value))
					}
				} else {
					descriptionBuilder.WriteString(fmt.Sprintf("- **%s**: %v\n", key, value))
				}
			default:
				descriptionBuilder.WriteString(fmt.Sprintf("- **%s**: %v\n", key, value))
			}
		}
	}

	// Add screenshot - embed as base64 data URI in markdown if available
	if feedback.ScreenshotURL.Valid && feedback.ScreenshotURL.String != "" {
		descriptionBuilder.WriteString("\n### Screenshot\n\n")
		descriptionBuilder.WriteString(fmt.Sprintf("![Screenshot](%s)\n", feedback.ScreenshotURL.String))
	} else if feedback.ScreenshotData.Valid && feedback.ScreenshotData.String != "" {
		descriptionBuilder.WriteString("\n### Screenshot\n\n")
		// Embed screenshot as base64 data URI
		screenshotData := feedback.ScreenshotData.String
		// Ensure it has the data URI prefix
		if !strings.HasPrefix(screenshotData, "data:") {
			screenshotData = "data:image/png;base64," + screenshotData
		}
		descriptionBuilder.WriteString(fmt.Sprintf("![Screenshot](%s)\n", screenshotData))
	}

	descriptionBuilder.WriteString("\n---\n*Created from Quiz Admin Feedback Reports*")

	description := descriptionBuilder.String()

	// Determine labels based on feedback type
	var labels []string
	switch feedback.FeedbackType {
	case "bug":
		labels = []string{"Bug"}
	case "feature_request":
		labels = []string{"Feature"}
	case "improvement":
		labels = []string{"Improvement"}
	}

	// Create Linear issue (use config defaults for team and project)
	result, err := h.linearService.CreateIssue(ctx, title, description, "", "", labels, "")
	if err != nil {
		h.logger.Error(ctx, "create linear issue failed", err, nil)
		HandleAppError(c, err)
		return
	}

	response := CreateLinearIssueResponse{
		IssueID:  result.IssueID,
		IssueURL: result.IssueURL,
		Title:    result.Title,
	}

	c.JSON(http.StatusOK, response)
}

// getTypeLabel converts feedback type to human-readable label
func getTypeLabel(feedbackType string) string {
	switch feedbackType {
	case "bug":
		return "Bug Report"
	case "feature_request":
		return "Feature Request"
	case "general":
		return "General Feedback"
	case "improvement":
		return "Improvement"
	default:
		return feedbackType
	}
}
