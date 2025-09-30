package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"
	"quizapp/internal/worker"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// WorkerAdminHandler handles worker administration endpoints
type WorkerAdminHandler struct {
	userService          services.UserServiceInterface
	questionService      services.QuestionServiceInterface
	aiService            services.AIServiceInterface
	config               *config.Config
	worker               *worker.Worker
	workerService        services.WorkerServiceInterface
	templates            *template.Template
	learningService      services.LearningServiceInterface
	dailyQuestionService services.DailyQuestionServiceInterface
	logger               *observability.Logger
}

// NewWorkerAdminHandlerWithLogger creates a new WorkerAdminHandler
func NewWorkerAdminHandlerWithLogger(
	userService services.UserServiceInterface,
	questionService services.QuestionServiceInterface,
	aiService services.AIServiceInterface,
	cfg *config.Config,
	worker *worker.Worker,
	workerService services.WorkerServiceInterface,
	learningService services.LearningServiceInterface,
	dailyQuestionService services.DailyQuestionServiceInterface,
	logger *observability.Logger,
) *WorkerAdminHandler {
	return &WorkerAdminHandler{
		userService:          userService,
		questionService:      questionService,
		aiService:            aiService,
		config:               cfg,
		worker:               worker,
		workerService:        workerService,
		templates:            nil,
		learningService:      learningService,
		dailyQuestionService: dailyQuestionService,
		logger:               logger,
	}
}

// GetWorkerDetails returns detailed worker information
func (h *WorkerAdminHandler) GetWorkerDetails(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_worker_details")
	defer span.End()
	// Get worker status from local instance if available
	var localStatus worker.Status
	var localHistory []worker.RunRecord
	if h.worker != nil {
		localStatus = h.worker.GetStatus()
		localHistory = h.worker.GetHistory()
	}

	// Get global pause status
	globalPaused, err := h.workerService.IsGlobalPaused(ctx)
	if err != nil {
		// Log the error but continue with default value
		h.logger.Warn(ctx, "Failed to get global pause status", map[string]interface{}{"error": err.Error()})
		globalPaused = false
	}

	response := gin.H{
		"status":        localStatus,
		"history":       localHistory,
		"global_paused": globalPaused,
	}

	c.JSON(http.StatusOK, response)
}

// GetActivityLogs returns recent activity logs from the worker
func (h *WorkerAdminHandler) GetActivityLogs(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_activity_logs")
	defer span.End()
	if h.worker == nil {
		HandleAppError(c, contextutils.ErrServiceUnavailable)
		return
	}

	logs := h.worker.GetActivityLogs()
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// PauseWorker pauses the worker globally
func (h *WorkerAdminHandler) PauseWorker(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "pause_worker")
	defer span.End()
	if err := h.workerService.SetGlobalPause(ctx, true); err != nil {
		HandleAppError(c, contextutils.WrapError(err, "failed to pause worker globally"))
		return
	}

	// Also pause the local worker instance if available
	if h.worker != nil {
		h.worker.Pause(ctx)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Worker paused globally"})
}

// ResumeWorker resumes the worker globally
func (h *WorkerAdminHandler) ResumeWorker(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "resume_worker")
	defer span.End()
	if err := h.workerService.SetGlobalPause(ctx, false); err != nil {
		HandleAppError(c, contextutils.WrapError(err, "failed to resume worker globally"))
		return
	}

	// Also resume the local worker instance if available
	if h.worker != nil {
		h.worker.Resume(ctx)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Worker resumed globally"})
}

// GetWorkerStatus returns current worker status
func (h *WorkerAdminHandler) GetWorkerStatus(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_worker_status")
	defer span.End()
	instance := c.DefaultQuery("instance", "default")

	status, err := h.workerService.GetWorkerStatus(ctx, instance)
	if err != nil {
		HandleAppError(c, contextutils.WrapError(err, "failed to get worker status"))
		return
	}

	c.JSON(http.StatusOK, status)
}

// TriggerWorkerRun triggers a manual worker run
func (h *WorkerAdminHandler) TriggerWorkerRun(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "trigger_worker_run")
	defer span.End()
	if h.worker != nil {
		h.worker.TriggerManualRun()
		c.JSON(http.StatusOK, gin.H{"message": "Worker run triggered"})
	} else {
		HandleAppError(c, contextutils.ErrServiceUnavailable)
	}
}

// PauseWorkerUser pauses question generation for a specific user
func (h *WorkerAdminHandler) PauseWorkerUser(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "pause_user")
	defer span.End()
	var req struct {
		UserID int `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request",
			"",
			err,
		))
		return
	}

	if err := h.workerService.SetUserPause(ctx, req.UserID, true); err != nil {
		HandleAppError(c, contextutils.WrapError(err, "failed to pause user"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User paused successfully"})
}

// ResumeWorkerUser resumes question generation for a specific user
func (h *WorkerAdminHandler) ResumeWorkerUser(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "resume_user")
	defer span.End()
	var req struct {
		UserID int `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request",
			"",
			err,
		))
		return
	}

	if err := h.workerService.SetUserPause(ctx, req.UserID, false); err != nil {
		HandleAppError(c, contextutils.WrapError(err, "failed to resume user"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User resumed successfully"})
}

// GetWorkerUsers returns basic user list for worker controls
func (h *WorkerAdminHandler) GetWorkerUsers(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_worker_users")
	defer span.End()
	users, err := h.userService.GetAllUsers(ctx)
	if err != nil {
		HandleAppError(c, contextutils.WrapError(err, "failed to get users"))
		return
	}

	// Add pause status for each user
	var userList []gin.H
	for _, user := range users {
		isPaused, _ := h.workerService.IsUserPaused(ctx, user.ID)
		userList = append(userList, gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"is_paused": isPaused,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": userList})
}

// GetSystemHealth returns comprehensive system health
func (h *WorkerAdminHandler) GetSystemHealth(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_system_health")
	defer span.End()
	health, err := h.workerService.GetWorkerHealth(ctx)
	if err != nil {
		HandleAppError(c, contextutils.WrapError(err, "failed to get system health"))
		return
	}

	c.JSON(http.StatusOK, health)
}

// GetAIConcurrencyStats returns AI service concurrency metrics from the worker
func (h *WorkerAdminHandler) GetAIConcurrencyStats(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_ai_concurrency_stats")
	defer span.End()
	if h.aiService == nil {
		HandleAppError(c, contextutils.ErrAIProviderUnavailable)
		return
	}

	stats := h.aiService.GetConcurrencyStats()
	c.JSON(http.StatusOK, gin.H{
		"ai_concurrency": stats,
	})
}

// GetPriorityAnalytics returns priority system analytics
func (h *WorkerAdminHandler) GetPriorityAnalytics(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_priority_analytics")
	defer span.End()
	// Get priority score distribution
	distribution, err := h.learningService.GetPriorityScoreDistribution(ctx)
	if err != nil {
		h.logger.Error(ctx, "Error getting priority score distribution", err, map[string]interface{}{})
		distribution = map[string]interface{}{
			"high":    0,
			"medium":  0,
			"low":     0,
			"average": 0.0,
		}
	}

	// Get high priority questions
	highPriorityQuestions, err := h.learningService.GetHighPriorityQuestions(ctx, 5)
	if err != nil {
		h.logger.Error(ctx, "Error getting high priority questions", err, map[string]interface{}{})
		highPriorityQuestions = []map[string]interface{}{}
	}

	response := gin.H{
		"distribution":          distribution,
		"highPriorityQuestions": highPriorityQuestions,
	}

	c.JSON(http.StatusOK, response)
}

// GetUserPriorityAnalytics returns priority analytics for a specific user
func (h *WorkerAdminHandler) GetUserPriorityAnalytics(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_user_priority_analytics")
	defer span.End()
	userIDStr := c.Param("userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Verify user exists
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	// Get user-specific priority score distribution
	distribution, err := h.learningService.GetUserPriorityScoreDistribution(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Error getting user priority score distribution", err, map[string]interface{}{})
		distribution = map[string]interface{}{
			"high":    0,
			"medium":  0,
			"low":     0,
			"average": 0.0,
		}
	}

	// Get user's high priority questions
	highPriorityQuestions, err := h.learningService.GetUserHighPriorityQuestions(ctx, userID, 10)
	if err != nil {
		h.logger.Error(ctx, "Error getting user high priority questions", err, map[string]interface{}{})
		highPriorityQuestions = []map[string]interface{}{}
	}

	// Get user's weak areas
	weakAreas, err := h.learningService.GetUserWeakAreas(ctx, userID, 5)
	if err != nil {
		h.logger.Error(ctx, "Error getting user weak areas", err, map[string]interface{}{})
		weakAreas = []map[string]interface{}{}
	}

	// Get user's learning preferences
	preferences, err := h.learningService.GetUserLearningPreferences(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Error getting user learning preferences", err, map[string]interface{}{})
		preferences = nil
	}

	response := gin.H{
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
		"distribution":          distribution,
		"highPriorityQuestions": highPriorityQuestions,
		"weakAreas":             weakAreas,
		"learningPreferences":   preferences,
	}

	c.JSON(http.StatusOK, response)
}

// GetUserPerformanceAnalytics returns user performance analytics
func (h *WorkerAdminHandler) GetUserPerformanceAnalytics(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_user_performance_analytics")
	defer span.End()
	// Get weak areas by topic
	weakAreas, err := h.learningService.GetWeakAreasByTopic(ctx, 5)
	if err != nil {
		h.logger.Error(ctx, "Error getting weak areas", err, map[string]interface{}{})
		weakAreas = []map[string]interface{}{}
	}

	// Get learning preferences usage
	learningPreferences, err := h.learningService.GetLearningPreferencesUsage(ctx)
	if err != nil {
		h.logger.Error(ctx, "Error getting learning preferences usage", err, map[string]interface{}{})
		learningPreferences = map[string]interface{}{}
	}

	response := gin.H{
		"weakAreas":           weakAreas,
		"learningPreferences": learningPreferences,
	}

	c.JSON(http.StatusOK, response)
}

// GetGenerationIntelligence returns question generation intelligence
func (h *WorkerAdminHandler) GetGenerationIntelligence(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_generation_intelligence")
	defer span.End()
	// Get gap analysis
	gapAnalysis, err := h.learningService.GetQuestionTypeGaps(ctx)
	if err != nil {
		h.logger.Error(ctx, "Error getting gap analysis", err, map[string]interface{}{})
		gapAnalysis = []map[string]interface{}{}
	}

	// Get generation suggestions
	generationSuggestions, err := h.learningService.GetGenerationSuggestions(ctx)
	if err != nil {
		h.logger.Error(ctx, "Error getting generation suggestions", err, map[string]interface{}{})
		generationSuggestions = []map[string]interface{}{}
	}

	// Ensure we always return arrays, not nil
	if gapAnalysis == nil {
		gapAnalysis = []map[string]interface{}{}
	}
	if generationSuggestions == nil {
		generationSuggestions = []map[string]interface{}{}
	}

	response := gin.H{
		"gapAnalysis":           gapAnalysis,
		"generationSuggestions": generationSuggestions,
	}

	c.JSON(http.StatusOK, response)
}

// GetSystemHealthAnalytics returns system health analytics
func (h *WorkerAdminHandler) GetSystemHealthAnalytics(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_system_health_analytics")
	defer span.End()
	// Get performance metrics
	performance, err := h.learningService.GetPrioritySystemPerformance(ctx)
	if err != nil {
		h.logger.Error(ctx, "Error getting performance metrics", err, map[string]interface{}{})
		performance = map[string]interface{}{}
	}

	// Get background jobs status
	backgroundJobs, err := h.learningService.GetBackgroundJobsStatus(ctx)
	if err != nil {
		h.logger.Error(ctx, "Error getting background jobs status", err, map[string]interface{}{})
		backgroundJobs = map[string]interface{}{}
	}

	response := gin.H{
		"performance":    performance,
		"backgroundJobs": backgroundJobs,
	}

	c.JSON(http.StatusOK, response)
}

// GetUserComparisonAnalytics returns comparison analytics between users
func (h *WorkerAdminHandler) GetUserComparisonAnalytics(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_user_comparison_analytics")
	defer span.End()
	userIDsParam := c.Query("user_ids")
	if userIDsParam == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Split comma-separated user IDs
	userIDsStr := strings.Split(userIDsParam, ",")
	if len(userIDsStr) == 0 {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	var userIDs []int
	for _, idStr := range userIDsStr {
		idStr = strings.TrimSpace(idStr) // Remove whitespace
		if idStr == "" {
			continue
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			HandleAppError(c, contextutils.NewAppErrorWithCause(
				contextutils.ErrorCodeInvalidFormat,
				contextutils.SeverityWarn,
				"Invalid user ID",
				idStr,
				err,
			))
			return
		}
		userIDs = append(userIDs, id)
	}

	if len(userIDs) == 0 {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Get comparison data for each user
	var comparisonData []gin.H
	for _, userID := range userIDs {
		user, err := h.userService.GetUserByID(ctx, userID)
		if err != nil {
			continue // Skip invalid users
		}

		distribution, _ := h.learningService.GetUserPriorityScoreDistribution(ctx, userID)
		weakAreas, _ := h.learningService.GetUserWeakAreas(ctx, userID, 3)

		userData := gin.H{
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
			},
			"distribution": distribution,
			"weakAreas":    weakAreas,
		}
		comparisonData = append(comparisonData, userData)
	}

	c.JSON(http.StatusOK, gin.H{"comparison": comparisonData})
}

// GetConfigz returns the merged config as pretty-printed JSON
func (h *WorkerAdminHandler) GetConfigz(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_configz")
	defer span.End()
	c.IndentedJSON(http.StatusOK, h.config)
}

// GetNotificationStats returns comprehensive notification statistics
func (h *WorkerAdminHandler) GetNotificationStats(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_notification_stats")
	defer span.End()

	// Get notification statistics from database
	stats, err := h.workerService.GetNotificationStats(ctx)
	if err != nil {
		h.logger.Error(ctx, "Failed to get notification stats", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get notification statistics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetNotificationErrors returns paginated notification errors
func (h *WorkerAdminHandler) GetNotificationErrors(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_notification_errors")
	defer span.End()

	// Parse pagination and filters
	page, pageSize := ParsePagination(c, 1, 20, 100)
	f := ParseFilters(c, "error_type", "notification_type", "resolved")
	errorType := f["error_type"]
	notificationType := f["notification_type"]
	resolved := f["resolved"]

	// Get notification errors from database
	errors, pagination, stats, err := h.workerService.GetNotificationErrors(ctx, page, pageSize, errorType, notificationType, resolved)
	if err != nil {
		h.logger.Error(ctx, "Failed to get notification errors", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get notification errors",
			"details": err.Error(),
		})
		return
	}

	WritePaginated(c, "errors", errors, pagination, gin.H{"stats": stats})
}

// GetSentNotifications returns paginated sent notifications
func (h *WorkerAdminHandler) GetSentNotifications(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_sent_notifications")
	defer span.End()

	// Parse pagination and filters
	page, pageSize := ParsePagination(c, 1, 20, 100)
	f := ParseFilters(c, "notification_type", "status", "sent_after", "sent_before")
	notificationType := f["notification_type"]
	status := f["status"]
	sentAfter := f["sent_after"]
	sentBefore := f["sent_before"]

	// Get sent notifications from database
	notifications, pagination, stats, err := h.workerService.GetSentNotifications(ctx, page, pageSize, notificationType, status, sentAfter, sentBefore)
	if err != nil {
		h.logger.Error(ctx, "Failed to get sent notifications", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get sent notifications",
			"details": err.Error(),
		})
		return
	}

	WritePaginated(c, "notifications", notifications, pagination, gin.H{"stats": stats})
}

// CreateTestSentNotification creates a test sent notification for testing
func (h *WorkerAdminHandler) CreateTestSentNotification(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "create_test_sent_notification")
	defer span.End()

	// Parse request body
	var request struct {
		UserID           int    `json:"user_id" binding:"required"`
		NotificationType string `json:"notification_type" binding:"required"`
		Subject          string `json:"subject" binding:"required"`
		TemplateName     string `json:"template_name" binding:"required"`
		Status           string `json:"status" binding:"required"`
		ErrorMessage     string `json:"error_message"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request body",
			"",
			err,
		))
		return
	}

	// Create test notification
	err := h.workerService.CreateTestSentNotification(ctx, request.UserID, request.NotificationType, request.Subject, request.TemplateName, request.Status, request.ErrorMessage)
	if err != nil {
		h.logger.Error(ctx, "Failed to create test sent notification", err, map[string]interface{}{
			"user_id":           request.UserID,
			"notification_type": request.NotificationType,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create test sent notification",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test sent notification created successfully"})
}

// ForceSendNotification forces sending a notification to a user, bypassing normal checks
func (h *WorkerAdminHandler) ForceSendNotification(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "force_send_notification")
	defer span.End()

	// Parse request body
	var request struct {
		Username string `json:"username" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request body",
			"",
			err,
		))
		return
	}

	// Get user by username
	user, err := h.userService.GetUserByUsername(ctx, request.Username)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user by username", err, map[string]interface{}{
			"username": request.Username,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get user",
			"details": err.Error(),
		})
		return
	}

	if user == nil {
		HandleAppError(c, contextutils.NewAppError(
			contextutils.ErrorCodeRecordNotFound,
			contextutils.SeverityInfo,
			fmt.Sprintf("User '%s' not found", request.Username),
			"",
		))
		return
	}

	// Check if user has email address
	if !user.Email.Valid || user.Email.String == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Get user's learning preferences to check daily reminder setting
	prefs, err := h.learningService.GetUserLearningPreferences(ctx, user.ID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user learning preferences", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get user preferences",
			"details": err.Error(),
		})
		return
	}

	// Check if daily reminders are enabled for this user
	if prefs == nil || !prefs.DailyReminderEnabled {
		HandleAppError(c, contextutils.NewAppError(contextutils.ErrorCodeInvalidInput, contextutils.SeverityWarn, "User has daily reminders disabled", ""))
		return
	}

	// Force send the daily reminder (bypassing time and date checks)
	subject := "Time for your daily quiz! 🧠"
	status := "sent"
	errorMsg := ""

	// Get email service from worker
	emailService := h.worker.GetEmailService()
	if emailService == nil {
		HandleAppError(c, contextutils.ErrServiceUnavailable)
		return
	}

	// Send the email
	if err := emailService.SendDailyReminder(ctx, user); err != nil {
		h.logger.Error(ctx, "Failed to send forced daily reminder", err, map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email.String,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to send notification"))
		return
	}

	// Record the sent notification in the database
	if err := emailService.RecordSentNotification(ctx, user.ID, "daily_reminder", subject, "daily_reminder", status, errorMsg); err != nil {
		h.logger.Error(ctx, "Failed to record sent notification", err, map[string]interface{}{
			"user_id": user.ID,
		})
		// Don't fail the request if recording fails
	}

	// Update the last reminder sent timestamp for this user
	if err := h.learningService.UpdateLastDailyReminderSent(ctx, user.ID); err != nil {
		h.logger.Error(ctx, "Failed to update last daily reminder sent timestamp", err, map[string]interface{}{
			"user_id": user.ID,
		})
		// Don't fail the request if timestamp update fails
	}

	h.logger.Info(ctx, "Forced notification sent successfully", map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email.String,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Notification sent successfully",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email.String,
		},
		"notification": gin.H{
			"type":    "daily_reminder",
			"subject": subject,
			"status":  status,
		},
	})
}

// GetUserDailyQuestions returns daily questions for a specific user and date (admin only)
func (h *WorkerAdminHandler) GetUserDailyQuestions(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "admin_get_user_daily_questions")
	defer span.End()

	// Parse user ID
	userIDStr := c.Param("userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Check if user exists
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user for daily questions", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to get user"))
		return
	}
	if user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	// Parse date
	dateStr := c.Param("date")
	if dateStr == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("date", dateStr),
	)

	// Get daily questions for the user and date
	questions, err := h.dailyQuestionService.GetDailyQuestions(ctx, userID, date)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user daily questions", err, map[string]interface{}{
			"user_id": userID,
			"date":    dateStr,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get daily questions",
			"details": err.Error(),
		})
		return
	}

	// Convert to API format (similar to the daily question handler)
	apiQuestions := make([]gin.H, len(questions))
	for i, q := range questions {
		var completedAt *time.Time
		if q.CompletedAt.Valid {
			completedAt = &q.CompletedAt.Time
		}

		apiQuestions[i] = gin.H{
			"id":              q.ID,
			"user_id":         q.UserID,
			"question_id":     q.QuestionID,
			"assignment_date": q.AssignmentDate,
			"is_completed":    q.IsCompleted,
			"completed_at":    completedAt,
			"created_at":      q.CreatedAt,
			// Per-user stats for admin UI
			"user_shown_count":     q.DailyShownCount,
			"user_total_responses": q.UserTotalResponses,
			"user_correct_count":   q.UserCorrectCount,
			"user_incorrect_count": q.UserIncorrectCount,
			"question": gin.H{
				"id":                  q.Question.ID,
				"type":                q.Question.Type,
				"language":            q.Question.Language,
				"level":               q.Question.Level,
				"difficulty_score":    q.Question.DifficultyScore,
				"content":             q.Question.Content,
				"correct_answer":      q.Question.CorrectAnswer,
				"explanation":         q.Question.Explanation,
				"created_at":          q.Question.CreatedAt,
				"status":              q.Question.Status,
				"topic_category":      q.Question.TopicCategory,
				"grammar_focus":       q.Question.GrammarFocus,
				"vocabulary_domain":   q.Question.VocabularyDomain,
				"scenario":            q.Question.Scenario,
				"style_modifier":      q.Question.StyleModifier,
				"difficulty_modifier": q.Question.DifficultyModifier,
				"time_context":        q.Question.TimeContext,
			},
		}
	}

	c.JSON(http.StatusOK, gin.H{"questions": apiQuestions})
}

// RegenerateUserDailyQuestions clears and regenerates daily questions for a specific user and date (admin only)
func (h *WorkerAdminHandler) RegenerateUserDailyQuestions(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "admin_regenerate_user_daily_questions")
	defer span.End()

	// Parse user ID
	userIDStr := c.Param("userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Check if user exists
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user for daily questions regeneration", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to get user"))
		return
	}
	if user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	// Parse date
	dateStr := c.Param("date")
	if dateStr == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("date", dateStr),
	)

	// For regeneration, we need to manually clear existing assignments and create new ones
	// Since the daily question service doesn't expose a direct way to clear assignments,
	// we'll use the worker service which should have database access for this admin operation

	// Check if worker service is available
	if h.workerService == nil {
		HandleAppError(c, contextutils.ErrServiceUnavailable)
		return
	}

	// Use the new RegenerateDailyQuestions method which clears existing assignments and creates new ones
	err = h.dailyQuestionService.RegenerateDailyQuestions(ctx, userID, date)
	if err != nil {
		h.logger.Error(ctx, "Failed to regenerate daily questions", err, map[string]interface{}{
			"user_id": userID,
			"date":    dateStr,
		})

		// If there are no questions available for assignment, prefer the structured error from the service
		var nqErr *services.NoQuestionsAvailableError
		if errors.As(err, &nqErr) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":                    "Failed to regenerate daily questions",
				"details":                  err.Error(),
				"user":                     gin.H{"id": user.ID, "username": user.Username, "language": nqErr.Language, "level": nqErr.Level},
				"candidate_count":          nqErr.CandidateCount,
				"candidate_ids":            nqErr.CandidateIDs,
				"total_matching_questions": nqErr.TotalMatching,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to regenerate daily questions",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info(ctx, "Daily questions regenerated successfully", map[string]interface{}{
		"user_id": userID,
		"date":    dateStr,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Daily questions regenerated successfully. All existing assignments have been cleared and new questions assigned."})
}
