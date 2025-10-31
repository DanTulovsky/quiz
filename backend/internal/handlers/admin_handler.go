// Package handlers provides HTTP request handlers for the quiz application API.
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"html/template"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// AdminHandler handles administrative HTTP requests and dashboard functionality
type AdminHandler struct {
	userService     services.UserServiceInterface
	questionService services.QuestionServiceInterface
	aiService       services.AIServiceInterface
	config          *config.Config
	templates       *template.Template
	learningService services.LearningServiceInterface
	workerService   services.WorkerServiceInterface
	logger          *observability.Logger
	storyService    services.StoryServiceInterface
	usageStatsSvc   services.UsageStatsServiceInterface
}

// NewAdminHandlerWithLogger creates a new AdminHandler with the provided services and logger.
func NewAdminHandlerWithLogger(userService services.UserServiceInterface, questionService services.QuestionServiceInterface, aiService services.AIServiceInterface, cfg *config.Config, learningService services.LearningServiceInterface, workerService services.WorkerServiceInterface, logger *observability.Logger, usageStatsSvc services.UsageStatsServiceInterface) *AdminHandler {
	return &AdminHandler{
		userService:     userService,
		questionService: questionService,
		aiService:       aiService,
		config:          cfg,
		templates:       nil,
		learningService: learningService,
		workerService:   workerService,
		logger:          logger,
		usageStatsSvc:   usageStatsSvc,
	}
}

// GetBackendAdminData returns the backend administration data as JSON
func (h *AdminHandler) GetBackendAdminData(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_backend_admin_data")
	defer observability.FinishSpan(span, nil)

	// Get all users for aggregate statistics
	users, err := h.userService.GetAllUsers(ctx)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		HandleAppError(c, contextutils.WrapError(err, "failed to get users"))
		return
	}

	// Calculate aggregate user statistics
	userStats := calculateUserAggregateStats(ctx, users, h.learningService, h.logger)

	// Get question statistics
	questionStats, err := h.questionService.GetDetailedQuestionStats(ctx)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get question stats", map[string]interface{}{"error": err.Error()})
		questionStats = make(map[string]interface{})
	}

	// Get worker health if available
	var workerHealth map[string]interface{}
	if h.workerService != nil {
		workerHealth, err = h.workerService.GetWorkerHealth(ctx)
		if err != nil {
			h.logger.Warn(ctx, "Failed to get worker health", map[string]interface{}{"error": err.Error()})
			workerHealth = map[string]interface{}{
				"error": "Failed to get worker health",
			}
		}
	}

	// Get AI concurrency stats
	aiStatsStruct := h.aiService.GetConcurrencyStats()
	aiConcurrencyStats := map[string]interface{}{
		"active_requests":   aiStatsStruct.ActiveRequests,
		"max_concurrent":    aiStatsStruct.MaxConcurrent,
		"queued_requests":   aiStatsStruct.QueuedRequests,
		"total_requests":    aiStatsStruct.TotalRequests,
		"user_active_count": aiStatsStruct.UserActiveCount,
		"max_per_user":      aiStatsStruct.MaxPerUser,
	}

	data := gin.H{
		"user_stats":           userStats,
		"question_stats":       questionStats,
		"worker_health":        workerHealth,
		"ai_concurrency_stats": aiConcurrencyStats,
		"worker_port":          h.config.Server.WorkerPort,
		"worker_base_url":      h.config.Server.WorkerBaseURL,
	}

	c.JSON(http.StatusOK, data)
}

// GetBackendAdminPage renders the backend administration dashboard
func (h *AdminHandler) GetBackendAdminPage(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_backend_admin_page")
	defer observability.FinishSpan(span, nil)

	// Get all users with progress and question stats
	users, err := h.userService.GetAllUsers(ctx)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		HandleAppError(c, contextutils.WrapError(err, "failed to get users"))
		return
	}

	type UserWithProgress struct {
		User               models.User
		Progress           *models.UserProgress
		QuestionStats      *services.UserQuestionStats
		UserQuestionCounts map[string]interface{}
	}

	var usersWithProgress []UserWithProgress
	for _, user := range users {
		progress, err := h.learningService.GetUserProgress(ctx, user.ID)
		if err != nil {
			h.logger.Warn(ctx, "Failed to get progress for user", map[string]interface{}{"user_id": user.ID, "error": err.Error()})
			progress = &models.UserProgress{
				CurrentLevel:   "A1",
				TotalQuestions: 0,
				CorrectAnswers: 0,
				AccuracyRate:   0,
			}
		}

		questionStats, err := h.learningService.GetUserQuestionStats(ctx, user.ID)
		if err != nil {
			h.logger.Warn(ctx, "Failed to get question stats for user", map[string]interface{}{"user_id": user.ID, "error": err.Error()})
			questionStats = &services.UserQuestionStats{
				UserID:        user.ID,
				TotalAnswered: 0,
			}
		}

		// Get per-user question counts by type and level
		userQuestionCounts := make(map[string]interface{})

		// Use the available stats from UserQuestionStats
		if questionStats != nil {
			userQuestionCounts["total_answered"] = questionStats.TotalAnswered
			userQuestionCounts["answered_by_type"] = questionStats.AnsweredByType
			userQuestionCounts["answered_by_level"] = questionStats.AnsweredByLevel
			userQuestionCounts["accuracy_by_type"] = questionStats.AccuracyByType
			userQuestionCounts["accuracy_by_level"] = questionStats.AccuracyByLevel
			userQuestionCounts["available_by_type"] = questionStats.AvailableByType
			userQuestionCounts["available_by_level"] = questionStats.AvailableByLevel
		}

		usersWithProgress = append(usersWithProgress, UserWithProgress{
			User:               user,
			Progress:           progress,
			QuestionStats:      questionStats,
			UserQuestionCounts: userQuestionCounts,
		})
	}

	// Get question statistics
	questionStats, err := h.questionService.GetDetailedQuestionStats(ctx)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get question stats", map[string]interface{}{"error": err.Error()})
		questionStats = make(map[string]interface{})
	}

	// Get worker health if available
	var workerHealth map[string]interface{}
	if h.workerService != nil {
		workerHealth, err = h.workerService.GetWorkerHealth(ctx)
		if err != nil {
			h.logger.Warn(ctx, "Failed to get worker health", map[string]interface{}{"error": err.Error()})
			workerHealth = map[string]interface{}{
				"error": "Failed to get worker health",
			}
		}
	}

	// Get AI concurrency stats
	aiStatsStruct := h.aiService.GetConcurrencyStats()
	aiConcurrencyStats := map[string]interface{}{
		"active_requests":   aiStatsStruct.ActiveRequests,
		"max_concurrent":    aiStatsStruct.MaxConcurrent,
		"queued_requests":   aiStatsStruct.QueuedRequests,
		"total_requests":    aiStatsStruct.TotalRequests,
		"user_active_count": aiStatsStruct.UserActiveCount,
		"max_per_user":      aiStatsStruct.MaxPerUser,
	}

	data := gin.H{
		"Title":              "Backend Administration",
		"Users":              usersWithProgress,
		"QuestionStats":      questionStats,
		"WorkerHealth":       workerHealth,
		"AIConcurrencyStats": aiConcurrencyStats,
		"IsBackend":          true,
		"WorkerPort":         h.config.Server.WorkerPort,
		"CurrentPage":        "backend_admin",
		"WorkerBaseURL":      h.config.Server.WorkerBaseURL,
	}

	// Try to render template, fallback to JSON if template fails
	if h.templates != nil {
		// Add no-cache headers
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		if err := h.templates.ExecuteTemplate(c.Writer, "backend_admin.html", data); err != nil {
			h.logger.Error(ctx, "Template execution failed", err, map[string]interface{}{})
			HandleAppError(c, contextutils.WrapError(err, "failed to render template"))
			return
		}
	} else {
		c.JSON(http.StatusOK, data)
	}
}

// UserData represents user information combined with their progress data
type UserData struct {
	User     models.User
	Progress *models.UserProgress
}

// UserDataWithQuestions represents user information with questions and responses
type UserDataWithQuestions struct {
	User            models.User
	Progress        *models.UserProgress
	QuestionStats   *services.UserQuestionStats
	TotalQuestions  int
	TotalResponses  int
	RecentQuestions []string
	Questions       []*services.QuestionWithStats // Actual question objects with stats
}

// ReportedQuestionsData represents the structure for reported questions page data
type ReportedQuestionsData struct {
	Users             []UserDataWithQuestions
	ReportedQuestions []*services.ReportedQuestionWithUser
}

// ShowDatazPage - Removed: Use frontend admin interface instead

// MarkQuestionAsFixed marks a reported question as fixed and puts it back in rotation
func (h *AdminHandler) MarkQuestionAsFixed(c *gin.Context) {
	questionIDStr := c.Param("id")
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	if err := h.questionService.MarkQuestionAsFixed(c.Request.Context(), questionID); err != nil {
		h.logger.Error(c.Request.Context(), "Failed to mark question as fixed", err, map[string]interface{}{"question_id": questionID})

		// Check if the error is due to question not found
		if contextutils.IsError(err, contextutils.ErrRecordNotFound) {
			HandleAppError(c, contextutils.ErrQuestionNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to mark question as fixed"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Question marked as fixed successfully"})
}

// UpdateQuestion updates a question's content, correct answer, and explanation
func (h *AdminHandler) UpdateQuestion(c *gin.Context) {
	questionIDStr := c.Param("id")
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	var req struct {
		Content       map[string]interface{} `json:"content" binding:"required"`
		CorrectAnswer int                    `json:"correct_answer" binding:"gte=0,lte=3"`
		Explanation   string                 `json:"explanation" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request format",
			"",
			err,
		))
		return
	}

	// Sanitize incoming content to avoid nested `content.content` and duplicated fields.
	content := req.Content
	for {
		if inner, ok := content["content"]; ok {
			if innerMap, ok2 := inner.(map[string]interface{}); ok2 {
				content = innerMap
				continue
			}
		}
		break
	}

	// Remove duplicate top-level keys from the content payload if present.
	// Defensive cleanup while migrating to strict OpenAPI validation.
	delete(content, "correct_answer")
	delete(content, "explanation")
	delete(content, "change_reason")

	// Ensure options is not nil (convert null -> empty slice)
	if opts, exists := content["options"]; !exists || opts == nil {
		content["options"] = []string{}
	}

	if err := h.questionService.UpdateQuestion(c.Request.Context(), questionID, content, req.CorrectAnswer, req.Explanation); err != nil {
		h.logger.Error(c.Request.Context(), "Failed to update question", err, map[string]interface{}{"question_id": questionID})

		// Check if the error is due to question not found
		if contextutils.IsError(err, contextutils.ErrRecordNotFound) {
			HandleAppError(c, contextutils.ErrQuestionNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to update question"))
		return
	}

	// If requested, mark the question as fixed and clear reports
	if strings.ToLower(c.Query("mark_fixed")) == "true" {
		ctx := c.Request.Context()
		// Mark as fixed (sets status to active)
		if err := h.questionService.MarkQuestionAsFixed(ctx, questionID); err != nil {
			h.logger.Error(ctx, "Failed to mark question as fixed after update", err, map[string]interface{}{"question_id": questionID})
			HandleAppError(c, contextutils.WrapError(err, "failed to mark question as fixed"))
			return
		}

		// Clear question reports
		db := h.questionService.DB()
		if _, err := db.ExecContext(ctx, `DELETE FROM question_reports WHERE question_id = $1`, questionID); err != nil {
			h.logger.Warn(ctx, "Failed to clear question reports", map[string]interface{}{"question_id": questionID, "error": err.Error()})
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Question updated successfully"})
}

// FixQuestionWithAI uses AI to suggest fixes for a problematic question
func (h *AdminHandler) FixQuestionWithAI(c *gin.Context) {
	questionIDStr := c.Param("id")
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Get the original question
	question, err := h.questionService.GetQuestionByID(c.Request.Context(), questionID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get question", err, map[string]interface{}{"question_id": questionID})

		// Check if the error is due to question not found
		if errors.Is(err, sql.ErrNoRows) {
			HandleAppError(c, contextutils.ErrQuestionNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to get question"))
		return
	}

	// Find reporter(s) and choose a configured AI provider/model from the reporting user(s)
	ctx := c.Request.Context()
	db := h.questionService.DB()
	rows, err := db.QueryContext(ctx, `SELECT u.id, u.username, u.ai_provider, u.ai_model, qr.report_reason FROM question_reports qr JOIN users u ON qr.reported_by_user_id = u.id WHERE qr.question_id = $1 ORDER BY qr.created_at ASC`, questionID)
	if err != nil {
		h.logger.Error(ctx, "Failed to query question reports", err, map[string]interface{}{"question_id": questionID})
		HandleAppError(c, contextutils.WrapError(err, "failed to get report details"))
		return
	}
	if err := rows.Err(); err != nil {
		h.logger.Warn(ctx, "rows iteration error before defer", map[string]interface{}{"error": err.Error(), "question_id": questionID})
	}
	defer func() {
		if err := rows.Close(); err != nil {
			h.logger.Warn(ctx, "Failed to close report rows", map[string]interface{}{"error": err.Error(), "question_id": questionID})
		}
	}()

	var reporterID int
	var reporterUsername string
	var reporterProvider sql.NullString
	var reporterModel sql.NullString
	var singleReason sql.NullString
	foundProvider := false

	for rows.Next() {
		var uid int
		var uname string
		var prov sql.NullString
		var mod sql.NullString
		var reason sql.NullString
		if err := rows.Scan(&uid, &uname, &prov, &mod, &reason); err != nil {
			h.logger.Warn(ctx, "Failed to scan report row", map[string]interface{}{"error": err.Error(), "question_id": questionID})
			continue
		}
		// Prefer the first reporter that has an AI provider+model configured
		if prov.Valid && prov.String != "" && mod.Valid && mod.String != "" {
			reporterID = uid
			reporterUsername = uname
			reporterProvider = prov
			reporterModel = mod
			singleReason = reason
			foundProvider = true
			break
		}
		// Keep the first reporter as fallback (no provider)
		if reporterID == 0 {
			reporterID = uid
			reporterUsername = uname
			reporterProvider = prov
			reporterModel = mod
			singleReason = reason
		}
	}

	if !foundProvider {
		// If no reporting user has AI configured, fall back to admin user's AI settings or global default provider
		h.logger.Info(ctx, "No reporting user has AI configured; attempting fallback to admin or global provider", map[string]interface{}{"question_id": questionID})

		// Try to get current admin user from context/session
		var adminUserID int
		if uid, err := GetCurrentUserID(c); err == nil {
			adminUserID = uid
		}

		// Try admin user's configured provider/model
		if adminUserID != 0 {
			adminUser, err := h.userService.GetUserByID(ctx, adminUserID)
			if err == nil && adminUser != nil && adminUser.AIProvider.Valid && adminUser.AIProvider.String != "" && adminUser.AIModel.Valid && adminUser.AIModel.String != "" {
				reporterID = adminUser.ID
				reporterUsername = adminUser.Username
				reporterProvider = adminUser.AIProvider
				reporterModel = adminUser.AIModel
				foundProvider = true
				h.logger.Info(ctx, "Falling back to admin user's AI provider", map[string]interface{}{"admin_id": adminUserID, "provider": adminUser.AIProvider.String, "model": adminUser.AIModel.String})
			}
		}

		// If still not found, try global config first provider
		if !foundProvider && h.config != nil && len(h.config.Providers) > 0 {
			p := h.config.Providers[0]
			if len(p.Models) > 0 {
				// Use first provider and model from global config
				reporterProvider = sql.NullString{String: p.Code, Valid: true}
				reporterModel = sql.NullString{String: p.Models[0].Code, Valid: true}
				reporterUsername = "system"
				foundProvider = true
				h.logger.Info(ctx, "Falling back to global configured AI provider", map[string]interface{}{"provider": p.Code, "model": p.Models[0].Code})
			}
		}

		if !foundProvider {
			h.logger.Warn(ctx, "No AI provider configured for reporting users and no fallback available", map[string]interface{}{"question_id": questionID})
			HandleAppError(c, contextutils.ErrAIConfigInvalid)
			return
		}
	}

	// Get saved API key for the reporter's configured provider
	savedKey, apiKeyID, _ := h.userService.GetUserAPIKeyWithID(ctx, reporterID, reporterProvider.String)

	userCfg := &models.UserAIConfig{
		Provider: reporterProvider.String,
		Model:    reporterModel.String,
		APIKey:   savedKey,
		Username: reporterUsername,
	}

	// Build AI chat request with question details and report reasons
	// Use the template manager to render a structured prompt
	// Prepare template data
	questionContentJSON, _ := question.MarshalContentToJSON()
	// Resolve schema for prompt; fail if none
	schema, err := services.GetFixSchema(question.Type)
	if err != nil {
		h.logger.Error(ctx, "No schema available for question type", err, map[string]interface{}{"question_id": questionID, "type": question.Type})
		HandleAppError(c, contextutils.ErrAIConfigInvalid)
		return
	}

	// Read optional additional_context from POST body JSON
	var body struct {
		AdditionalContext string `json:"additional_context"`
	}
	_ = c.BindJSON(&body) // ignore error; body may be empty

	tmplData := services.AITemplateData{
		CurrentQuestionJSON: questionContentJSON,
		ExampleContent:      "", // will be filled below if example available
		SchemaForPrompt:     schema,
		ReportReasons:       []string{},
		AdditionalContext:   body.AdditionalContext,
	}
	if singleReason.Valid {
		tmplData.ReportReasons = []string{singleReason.String}
	}
	// Load example for this question type if available
	if ex, err := h.aiService.TemplateManager().LoadExample(string(question.Type)); err == nil {
		tmplData.ExampleContent = ex
	}

	prompt, err := h.aiService.TemplateManager().RenderTemplate(services.AIFixPromptTemplate, tmplData)
	if err != nil {
		h.logger.Error(ctx, "Failed to render AI fix prompt", err, map[string]interface{}{"question_id": questionID})
		HandleAppError(c, contextutils.WrapError(err, "failed to build AI prompt"))
		return
	}

	// Use schema as grammar for providers that support it
	supportsGrammar := h.aiService.SupportsGrammarField(userCfg.Provider)
	var grammar string
	if supportsGrammar {
		grammar, err = services.GetFixSchema(question.Type)
		if err != nil {
			h.logger.Error(ctx, "No grammar schema available for question type", err, map[string]interface{}{"question_id": questionID, "type": question.Type})
			HandleAppError(c, contextutils.ErrAIConfigInvalid)
			return
		}
	} else {
		grammar = ""
	}

	// Add user ID and API key ID to context for usage tracking
	if reporterID != 0 {
		ctx = contextutils.WithUserID(ctx, reporterID)
	}
	if apiKeyID != nil {
		ctx = contextutils.WithAPIKeyID(ctx, *apiKeyID)
	}

	// Call AI service with constructed prompt and grammar
	respStr, err := h.aiService.CallWithPrompt(ctx, userCfg, prompt, grammar)
	if err != nil {
		h.logger.Error(ctx, "AI service call failed", err, map[string]interface{}{"question_id": questionID, "provider": userCfg.Provider})
		HandleAppError(c, contextutils.WrapError(err, "AI service error"))
		return
	}

	// Attempt to parse AI response as JSON (and try to recover JSON substring if necessary)
	var aiResp map[string]interface{}
	if err := json.Unmarshal([]byte(respStr), &aiResp); err != nil {
		start := strings.Index(respStr, "{")
		end := strings.LastIndex(respStr, "}")
		if start >= 0 && end > start {
			candidate := respStr[start : end+1]
			if err2 := json.Unmarshal([]byte(candidate), &aiResp); err2 != nil {
				h.logger.Error(ctx, "Failed to parse AI response as JSON", err2, map[string]interface{}{"question_id": questionID})
				HandleAppError(c, contextutils.ErrAIResponseInvalid)
				return
			}
		} else {
			h.logger.Error(ctx, "AI did not return JSON", nil, map[string]interface{}{"question_id": questionID})
			HandleAppError(c, contextutils.ErrAIResponseInvalid)
			return
		}
	}

	// Start from the original question map so required top-level fields are preserved
	originalMap := map[string]interface{}{}
	if b, err := json.Marshal(question); err == nil {
		_ = json.Unmarshal(b, &originalMap)
	}

	// Use helper to merge and normalize AI suggestion into original map
	suggestion := MergeAISuggestion(originalMap, aiResp)
	// Attach admin-provided additional context into suggestion metadata so frontend can display it
	if body.AdditionalContext != "" {
		suggestion["additional_context"] = body.AdditionalContext
	}

	// If query param apply=true present, apply suggestion directly and mark fixed
	if strings.ToLower(c.Query("apply")) == "true" {
		// Build update payload: use merged content and read answer/explanation from TOP LEVEL
		updateContent := suggestion["content"].(map[string]interface{})

		// Extract correct_answer from top level (support float64 from JSON)
		correctAnswer := 0
		if ca, ok := suggestion["correct_answer"]; ok {
			switch v := ca.(type) {
			case float64:
				correctAnswer = int(v)
			case int:
				correctAnswer = v
			}
		}

		// Extract explanation from top level
		explanation := ""
		if ex, ok := suggestion["explanation"].(string); ok {
			explanation = ex
		}

		if err := h.questionService.UpdateQuestion(c.Request.Context(), questionID, updateContent, correctAnswer, explanation); err != nil {
			h.logger.Error(c.Request.Context(), "Failed to update question with AI suggestion", err, map[string]interface{}{"question_id": questionID})
			HandleAppError(c, contextutils.WrapError(err, "failed to apply suggestion"))
			return
		}

		if err := h.questionService.MarkQuestionAsFixed(c.Request.Context(), questionID); err != nil {
			h.logger.Warn(c.Request.Context(), "Failed to mark question as fixed after applying suggestion", map[string]interface{}{"question_id": questionID, "error": err.Error()})
		}
		db := h.questionService.DB()
		if _, err := db.ExecContext(c.Request.Context(), `DELETE FROM question_reports WHERE question_id = $1`, questionID); err != nil {
			h.logger.Warn(c.Request.Context(), "Failed to clear question reports after applying suggestion", map[string]interface{}{"question_id": questionID, "error": err.Error()})
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Suggestion applied"})
		return
	}

	// Return original question and merged AI suggestion for frontend review
	c.JSON(http.StatusOK, gin.H{
		"original":   question,
		"suggestion": suggestion,
	})
}

// ServeDatazJS - Removed: Use frontend admin interface instead

// GetAIConcurrencyStats returns AI service concurrency metrics
func (h *AdminHandler) GetAIConcurrencyStats(c *gin.Context) {
	// Get stats from the local AI service instance
	stats := h.aiService.GetConcurrencyStats()
	c.JSON(http.StatusOK, gin.H{
		"ai_concurrency": stats,
	})
}

// --- Story Explorer (Admin) ---

// GetStoriesPaginated returns paginated stories with filters
func (h *AdminHandler) GetStoriesPaginated(c *gin.Context) {
	if h.storyService == nil {
		HandleAppError(c, contextutils.ErrInternalError)
		return
	}
	page, pageSize := ParsePagination(c, 1, 20, 100)
	f := ParseFilters(c, "search", "language", "status")
	search := f["search"]
	language := f["language"]
	status := f["status"]

	var userID *uint
	if u := c.Query("user_id"); u != "" {
		if parsed, err := strconv.Atoi(u); err == nil && parsed > 0 {
			tmp := uint(parsed)
			userID = &tmp
		} else {
			HandleAppError(c, contextutils.ErrInvalidFormat)
			return
		}
	}

	stories, total, err := h.storyService.GetStoriesPaginated(c.Request.Context(), page, pageSize, search, language, status, userID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get stories", err, map[string]interface{}{"page": page, "size": pageSize})
		HandleAppError(c, contextutils.WrapError(err, "failed to get stories"))
		return
	}

	// Map directly; convert to API struct for consistency
	storyMaps := make([]map[string]interface{}, 0, len(stories))
	for _, s := range stories {
		apiS := convertStoryToAPI(&s)
		m := map[string]interface{}{}
		if b, err := json.Marshal(apiS); err == nil {
			_ = json.Unmarshal(b, &m)
		}
		storyMaps = append(storyMaps, m)
	}

	c.JSON(http.StatusOK, gin.H{
		"stories": storyMaps,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": int(math.Ceil(float64(total) / float64(pageSize))),
		},
	})
}

// GetStoryAdmin returns a full story with sections by ID
func (h *AdminHandler) GetStoryAdmin(c *gin.Context) {
	if h.storyService == nil {
		HandleAppError(c, contextutils.ErrInternalError)
		return
	}
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}
	story, err := h.storyService.GetStoryAdmin(c.Request.Context(), uint(id))
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get story", err, map[string]interface{}{"story_id": id})
		if strings.Contains(err.Error(), "story not found") {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to get story"))
		return
	}
	c.JSON(http.StatusOK, convertStoryWithSectionsToAPI(story))
}

// GetSectionAdmin returns a section with questions by ID
func (h *AdminHandler) GetSectionAdmin(c *gin.Context) {
	if h.storyService == nil {
		HandleAppError(c, contextutils.ErrInternalError)
		return
	}
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}
	section, err := h.storyService.GetSectionAdmin(c.Request.Context(), uint(id))
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get section", err, map[string]interface{}{"section_id": id})
		if strings.Contains(err.Error(), "section not found") {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to get section"))
		return
	}
	c.JSON(http.StatusOK, convertStorySectionWithQuestionsToAPI(section))
}

// DeleteStoryAdmin deletes a story by ID (admin only). Only archived or completed stories can be deleted.
func (h *AdminHandler) DeleteStoryAdmin(c *gin.Context) {
	if h.storyService == nil {
		HandleAppError(c, contextutils.ErrInternalError)
		return
	}
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	if err := h.storyService.DeleteStoryAdmin(c.Request.Context(), uint(id)); err != nil {
		h.logger.Error(c.Request.Context(), "Failed to delete story (admin)", err, map[string]interface{}{"story_id": id})

		if strings.Contains(err.Error(), "not found") {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}
		if strings.Contains(err.Error(), "cannot delete active story") {
			HandleAppError(c, contextutils.ErrConflict)
			return
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to delete story"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Story deleted successfully"})
}

// ClearUserData removes all user activity data but keeps the users themselves
func (h *AdminHandler) ClearUserData(c *gin.Context) {
	err := h.userService.ClearUserData(c.Request.Context())
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to clear user data", err, map[string]interface{}{})
		HandleAppError(c, contextutils.WrapError(err, "failed to clear user data"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "User data cleared successfully (users preserved)"})
}

// ClearDatabase completely resets the database to an empty state
func (h *AdminHandler) ClearDatabase(c *gin.Context) {
	err := h.userService.ResetDatabase(c.Request.Context())
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to clear database", err, map[string]interface{}{})
		HandleAppError(c, contextutils.WrapError(err, "failed to clear database"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Database cleared successfully"})
}

// GetQuestion returns a single question by ID for editing
func (h *AdminHandler) GetQuestion(c *gin.Context) {
	questionIDStr := c.Param("id")
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	question, err := h.questionService.GetQuestionByID(c.Request.Context(), questionID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get question", err, map[string]interface{}{"question_id": questionID})
		HandleAppError(c, contextutils.ErrQuestionNotFound)
		return
	}

	c.JSON(http.StatusOK, question)
}

// GetUsersForQuestion returns the users assigned to a question
func (h *AdminHandler) GetUsersForQuestion(c *gin.Context) {
	questionIDStr := c.Param("id")
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	users, totalCount, err := h.questionService.GetUsersForQuestion(c.Request.Context(), questionID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get users for question", err, map[string]interface{}{"question_id": questionID})
		HandleAppError(c, contextutils.WrapError(err, "failed to get users for question"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":       users,
		"total_count": totalCount,
	})
}

// AssignUsersToQuestion assigns multiple users to a question
func (h *AdminHandler) AssignUsersToQuestion(c *gin.Context) {
	questionIDStr := c.Param("id")
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	var request struct {
		UserIDs []int `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		HandleAppError(c, contextutils.ErrInvalidInput)
		return
	}

	// Validate non-empty user list
	if len(request.UserIDs) == 0 {
		HandleAppError(c, contextutils.ErrInvalidInput)
		return
	}

	// Check if the question exists first
	_, err = h.questionService.GetQuestionByID(c.Request.Context(), questionID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get question", err, map[string]interface{}{"question_id": questionID})

		// Check if the error is due to question not found
		if errors.Is(err, sql.ErrNoRows) {
			HandleAppError(c, contextutils.ErrQuestionNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to get question"))
		return
	}

	err = h.questionService.AssignUsersToQuestion(c.Request.Context(), questionID, request.UserIDs)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to assign users to question", err, map[string]interface{}{
			"question_id": questionID,
			"user_ids":    request.UserIDs,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to assign users to question"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Users assigned to question successfully"})
}

// UnassignUsersFromQuestion removes multiple users from a question
func (h *AdminHandler) UnassignUsersFromQuestion(c *gin.Context) {
	questionIDStr := c.Param("id")
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	var request struct {
		UserIDs []int `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(contextutils.ErrorCodeInvalidInput, contextutils.SeverityWarn, "Invalid request body", "", err))
		return
	}

	// Validate non-empty user list
	if len(request.UserIDs) == 0 {
		HandleAppError(c, contextutils.ErrInvalidInput)
		return
	}

	// Check if the question exists first
	_, err = h.questionService.GetQuestionByID(c.Request.Context(), questionID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get question", err, map[string]interface{}{"question_id": questionID})

		// Check if the error is due to question not found
		if errors.Is(err, sql.ErrNoRows) {
			HandleAppError(c, contextutils.ErrQuestionNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to get question"))
		return
	}

	err = h.questionService.UnassignUsersFromQuestion(c.Request.Context(), questionID, request.UserIDs)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to unassign users from question", err, map[string]interface{}{
			"question_id": questionID,
			"user_ids":    request.UserIDs,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to unassign users from question"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Users unassigned from question successfully"})
}

// DeleteQuestion deletes a question by ID
func (h *AdminHandler) DeleteQuestion(c *gin.Context) {
	questionIDStr := c.Param("id")
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	err = h.questionService.DeleteQuestion(c.Request.Context(), questionID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to delete question", err, map[string]interface{}{"question_id": questionID})

		// Check if the error is due to question not found
		if contextutils.IsError(err, contextutils.ErrRecordNotFound) {
			HandleAppError(c, contextutils.ErrQuestionNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to delete question"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Question deleted successfully"})
}

// GetQuestionsPaginated returns paginated questions with response statistics
func (h *AdminHandler) GetQuestionsPaginated(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Parse pagination and filters
	page, pageSize := ParsePagination(c, 1, 10, 100)
	filters := ParseFilters(c, "search", "type", "status")
	search := filters["search"]
	typeFilter := filters["type"]
	statusFilter := filters["status"]

	// Get questions with filters
	questions, total, err := h.questionService.GetQuestionsPaginated(
		c.Request.Context(),
		userID,
		page,
		pageSize,
		search,
		typeFilter,
		statusFilter,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get paginated questions", err, map[string]interface{}{
			"user_id": userID,
			"page":    page,
			"size":    pageSize,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get questions"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"questions": func() []map[string]interface{} {
			out := make([]map[string]interface{}, 0, len(questions))
			for _, q := range questions {
				out = append(out, convertQuestionWithStatsToAPIMap(q))
			}
			return out
		}(),
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": int(math.Ceil(float64(total) / float64(pageSize))),
		},
	})
}

// GetAllQuestions returns all questions with pagination and filtering
func (h *AdminHandler) GetAllQuestions(c *gin.Context) {
	// Parse pagination and filters
	page, pageSize := ParsePagination(c, 1, 20, 100)
	f := ParseFilters(c, "search", "type", "status", "language", "level")
	search := f["search"]
	typeFilter := f["type"]
	statusFilter := f["status"]
	languageFilter := f["language"]
	levelFilter := f["level"]
	userIDStr := c.Query("user_id")

	// Parse user_id if provided
	var userID *int
	if userIDStr != "" {
		uid, err := strconv.Atoi(userIDStr)
		if err != nil {
			HandleAppError(c, contextutils.ErrInvalidFormat)
			return
		}
		userID = &uid
	}

	// Get questions with filters
	questions, total, err := h.questionService.GetAllQuestionsPaginated(
		c.Request.Context(),
		page,
		pageSize,
		search,
		typeFilter,
		statusFilter,
		languageFilter,
		levelFilter,
		userID,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get all questions", err, map[string]interface{}{
			"page":   page,
			"size":   pageSize,
			"search": search,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get questions"))
		return
	}

	// Get stats
	stats, err := h.questionService.GetQuestionStats(c.Request.Context())
	if err != nil {
		h.logger.Warn(c.Request.Context(), "Failed to get question stats", map[string]interface{}{"error": err.Error()})
		stats = map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{
		"questions": func() []map[string]interface{} {
			out := make([]map[string]interface{}, 0, len(questions))
			for _, q := range questions {
				out = append(out, convertQuestionWithStatsToAPIMap(q))
			}
			return out
		}(),
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": int(math.Ceil(float64(total) / float64(pageSize))),
		},
		"stats": stats,
	})
}

// GetReportedQuestionsPaginated returns reported questions with pagination and filtering
func (h *AdminHandler) GetReportedQuestionsPaginated(c *gin.Context) {
	// Parse pagination and filters
	page, pageSize := ParsePagination(c, 1, 20, 100)
	f := ParseFilters(c, "search", "type", "language", "level")
	search := f["search"]
	typeFilter := f["type"]
	languageFilter := f["language"]
	levelFilter := f["level"]

	// Get reported questions with filters
	questions, total, err := h.questionService.GetReportedQuestionsPaginated(
		c.Request.Context(),
		page,
		pageSize,
		search,
		typeFilter,
		languageFilter,
		levelFilter,
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to get reported questions", err, map[string]interface{}{
			"page":   page,
			"size":   pageSize,
			"search": search,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get reported questions"))
		return
	}

	// Get reported questions stats
	stats, err := h.questionService.GetReportedQuestionsStats(c.Request.Context())
	if err != nil {
		h.logger.Warn(c.Request.Context(), "Failed to get reported questions stats", map[string]interface{}{"error": err.Error()})
		stats = map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{
		"questions": func() []map[string]interface{} {
			out := make([]map[string]interface{}, 0, len(questions))
			for _, q := range questions {
				out = append(out, convertQuestionWithStatsToAPIMap(q))
			}
			return out
		}(),
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": int(math.Ceil(float64(total) / float64(pageSize))),
		},
		"stats": stats,
	})
}

// ClearUserDataForUser removes all user activity data for a specific user but keeps the user record
func (h *AdminHandler) ClearUserDataForUser(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "clear_user_data_for_user")
	defer observability.FinishSpan(span, nil)
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Check if user exists before attempting to clear data
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user for clear data operation", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to get user"))
		return
	}
	if user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	err = h.userService.ClearUserDataForUser(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to clear user data for user", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to clear user data for user"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "User data cleared successfully (user preserved)"})
}

// GetConfigz returns the merged config as pretty-printed JSON
func (h *AdminHandler) GetConfigz(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_configz")
	defer observability.FinishSpan(span, nil)
	c.IndentedJSON(http.StatusOK, h.config)
}

// GetRoles returns all available roles in the system
func (h *AdminHandler) GetRoles(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_roles")
	defer observability.FinishSpan(span, nil)

	// For now, return hardcoded roles since we don't have a role service
	// In a real implementation, you'd query the database
	roles := []models.Role{
		{ID: 1, Name: "user", Description: "Normal site access", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: 2, Name: "admin", Description: "Administrative access to all features", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

// GetUserRoles returns all roles for a specific user
func (h *AdminHandler) GetUserRoles(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_user_roles")
	defer observability.FinishSpan(span, nil)

	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Check if user exists before getting roles
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user for roles operation", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to get user"))
		return
	}
	if user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	roles, err := h.userService.GetUserRoles(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user roles", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to get user roles"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

// AssignRole assigns a role to a user
func (h *AdminHandler) AssignRole(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "assign_role")
	defer observability.FinishSpan(span, nil)

	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Check if user exists before assigning role
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user for role assignment", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to get user"))
		return
	}
	if user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	var req struct {
		RoleID int `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(contextutils.ErrorCodeInvalidInput, contextutils.SeverityWarn, "Invalid request body", "", err))
		return
	}

	// Ensure the requester is allowed (self or admin). Route is admin-only, but keep explicit check.
	currentUserID, err := GetCurrentUserID(c)
	if err == nil {
		if err := RequireSelfOrAdmin(ctx, h.userService, currentUserID, userID); err != nil {
			if errors.Is(err, ErrForbidden) {
				HandleAppError(c, contextutils.ErrForbidden)
				return
			}
			h.logger.Error(ctx, "Failed to check authorization", err, map[string]interface{}{"user_id": currentUserID})
			HandleAppError(c, contextutils.WrapError(err, "failed to check authorization"))
			return
		}
	}

	err = h.userService.AssignRole(ctx, userID, req.RoleID)
	if err != nil {
		h.logger.Error(ctx, "Failed to assign role to user", err, map[string]interface{}{"user_id": userID, "role_id": req.RoleID})
		HandleAppError(c, contextutils.WrapError(err, "failed to assign role"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role assigned successfully"})
}

// RemoveRole removes a role from a user
func (h *AdminHandler) RemoveRole(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "remove_role")
	defer observability.FinishSpan(span, nil)

	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Check if user exists before removing role
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user for role removal", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to get user"))
		return
	}
	if user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	roleIDStr := c.Param("roleId")
	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Ensure the requester is allowed (self or admin). Route is admin-only, but keep explicit check.
	currentUserID, err := GetCurrentUserID(c)
	if err == nil {
		if err := RequireSelfOrAdmin(ctx, h.userService, currentUserID, userID); err != nil {
			if errors.Is(err, ErrForbidden) {
				HandleAppError(c, contextutils.ErrForbidden)
				return
			}
			h.logger.Error(ctx, "Failed to check authorization", err, map[string]interface{}{"user_id": currentUserID})
			HandleAppError(c, contextutils.WrapError(err, "failed to check authorization"))
			return
		}
	}

	err = h.userService.RemoveRole(ctx, userID, roleID)
	if err != nil {
		h.logger.Error(ctx, "Failed to remove role", err, map[string]interface{}{"user_id": userID, "role_id": roleID})

		// Check if it's a "user does not have role" error
		if strings.Contains(err.Error(), "does not have role") {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}

		// Check if it's a "user not found" or "role not found" error
		if contextutils.IsError(err, contextutils.ErrRecordNotFound) {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to remove role"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role removed successfully"})
}

// GetUsageStats returns usage statistics for the admin interface
func (h *AdminHandler) GetUsageStats(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_usage_stats")
	defer observability.FinishSpan(span, nil)

	if h.usageStatsSvc == nil {
		HandleAppError(c, contextutils.ErrInternalError)
		return
	}

	// Get all usage stats
	stats, err := h.usageStatsSvc.GetAllUsageStats(ctx)
	if err != nil {
		h.logger.Error(ctx, "Failed to get usage stats", err, map[string]interface{}{})
		HandleAppError(c, contextutils.WrapError(err, "failed to get usage stats"))
		return
	}

	// Group stats by service and month for easier frontend consumption
	serviceStats := make(map[string]map[string]map[string]interface{})
	monthlyTotals := make(map[string]map[string]interface{})

	// Track cache statistics across all services
	var totalCacheHitsRequests, totalCacheHitsCharacters, totalCacheMissesRequests int

	for _, stat := range stats {
		serviceName := stat.ServiceName
		usageType := stat.UsageType
		month := stat.UsageMonth.Format("2006-01")

		if serviceStats[serviceName] == nil {
			serviceStats[serviceName] = make(map[string]map[string]interface{})
		}
		if serviceStats[serviceName][month] == nil {
			serviceStats[serviceName][month] = make(map[string]interface{})
		}

		serviceStats[serviceName][month][usageType] = map[string]interface{}{
			"characters_used": stat.CharactersUsed,
			"requests_made":   stat.RequestsMade,
			"quota":           h.usageStatsSvc.GetMonthlyQuota(serviceName),
		}

		// Accumulate cache statistics
		switch usageType {
		case "translation_cache_hit":
			totalCacheHitsRequests += stat.RequestsMade
			totalCacheHitsCharacters += stat.CharactersUsed
		case "translation_cache_miss":
			totalCacheMissesRequests += stat.RequestsMade
		}

		// Accumulate monthly totals (only for actual translations, not cache)
		if usageType == "translation" {
			if monthlyTotals[month] == nil {
				monthlyTotals[month] = make(map[string]interface{})
			}
			if monthlyTotals[month][serviceName] == nil {
				monthlyTotals[month][serviceName] = map[string]interface{}{
					"total_characters": 0,
					"total_requests":   0,
				}
			}

			totalChars := monthlyTotals[month][serviceName].(map[string]interface{})["total_characters"].(int) + stat.CharactersUsed
			totalReqs := monthlyTotals[month][serviceName].(map[string]interface{})["total_requests"].(int) + stat.RequestsMade

			monthlyTotals[month][serviceName].(map[string]interface{})["total_characters"] = totalChars
			monthlyTotals[month][serviceName].(map[string]interface{})["total_requests"] = totalReqs
		}
	}

	// Calculate cache hit rate
	totalCacheRequests := totalCacheHitsRequests + totalCacheMissesRequests
	var cacheHitRate float64
	if totalCacheRequests > 0 {
		cacheHitRate = (float64(totalCacheHitsRequests) / float64(totalCacheRequests)) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"usage_stats":    serviceStats,
		"monthly_totals": monthlyTotals,
		"services":       []string{"google"}, // Currently only Google Translate
		"cache_stats": gin.H{
			"total_cache_hits_requests":   totalCacheHitsRequests,
			"total_cache_hits_characters": totalCacheHitsCharacters,
			"total_cache_misses_requests": totalCacheMissesRequests,
			"cache_hit_rate":              cacheHitRate,
		},
	})
}

// GetUsageStatsByService returns usage statistics for a specific service
func (h *AdminHandler) GetUsageStatsByService(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_usage_stats_by_service")
	defer observability.FinishSpan(span, nil)

	serviceName := c.Param("service")
	if serviceName == "" {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Validate service name against configured translation providers
	if !h.config.Translation.Enabled {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	isValidService := false
	for providerCode := range h.config.Translation.Providers {
		if providerCode == serviceName {
			isValidService = true
			break
		}
	}

	if !isValidService {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	if h.usageStatsSvc == nil {
		HandleAppError(c, contextutils.ErrInternalError)
		return
	}

	stats, err := h.usageStatsSvc.GetUsageStatsByService(ctx, serviceName)
	if err != nil {
		h.logger.Error(ctx, "Failed to get usage stats by service", err, map[string]interface{}{"service": serviceName})
		HandleAppError(c, contextutils.WrapError(err, "failed to get usage stats"))
		return
	}

	// Format for frontend consumption
	monthlyData := make([]map[string]interface{}, 0)
	for _, stat := range stats {
		// Only show quota for actual translation usage, not for cache hits/misses
		var quota interface{}
		if stat.UsageType == "translation" {
			quota = h.usageStatsSvc.GetMonthlyQuota(serviceName)
		} else {
			quota = nil
		}

		monthlyData = append(monthlyData, map[string]interface{}{
			"month":           stat.UsageMonth.Format("2006-01"),
			"usage_type":      stat.UsageType,
			"characters_used": stat.CharactersUsed,
			"requests_made":   stat.RequestsMade,
			"quota":           quota,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"service": serviceName,
		"data":    monthlyData,
	})
}

// calculateUserAggregateStats calculates aggregate statistics for all users
func calculateUserAggregateStats(ctx context.Context, users []models.User, learningService services.LearningServiceInterface, logger *observability.Logger) map[string]interface{} {
	stats := map[string]interface{}{
		"total_users":              len(users),
		"by_language":              make(map[string]int),
		"by_level":                 make(map[string]int),
		"by_ai_provider":           make(map[string]int),
		"by_ai_model":              make(map[string]int),
		"ai_enabled":               0,
		"ai_disabled":              0,
		"active_users":             0,
		"inactive_users":           0,
		"total_questions_answered": 0,
		"total_correct_answers":    0,
		"average_accuracy":         0.0,
	}

	activeThreshold := time.Now().AddDate(0, 0, -7)

	for _, user := range users {
		lang := "unknown"
		if user.PreferredLanguage.Valid {
			lang = user.PreferredLanguage.String
		}
		stats["by_language"].(map[string]int)[lang]++

		level := "unknown"
		if user.CurrentLevel.Valid {
			level = user.CurrentLevel.String
		}
		stats["by_level"].(map[string]int)[level]++

		provider := "none"
		if user.AIProvider.Valid {
			provider = user.AIProvider.String
		}
		stats["by_ai_provider"].(map[string]int)[provider]++

		model := "none"
		if user.AIModel.Valid {
			model = user.AIModel.String
		}
		stats["by_ai_model"].(map[string]int)[model]++

		if user.AIEnabled.Valid && user.AIEnabled.Bool {
			aiEnabled := stats["ai_enabled"].(int)
			stats["ai_enabled"] = aiEnabled + 1
		} else {
			aiDisabled := stats["ai_disabled"].(int)
			stats["ai_disabled"] = aiDisabled + 1
		}

		if user.LastActive.Valid {
			lastActive := user.LastActive.Time
			if lastActive.After(activeThreshold) {
				activeUsers := stats["active_users"].(int)
				stats["active_users"] = activeUsers + 1
			} else {
				inactiveUsers := stats["inactive_users"].(int)
				stats["inactive_users"] = inactiveUsers + 1
			}
		} else {
			inactiveUsers := stats["inactive_users"].(int)
			stats["inactive_users"] = inactiveUsers + 1
		}

		progress, err := learningService.GetUserProgress(ctx, user.ID)
		if err != nil {
			logger.Warn(ctx, "Failed to get progress for user", map[string]interface{}{"user_id": user.ID, "error": err.Error()})
			continue
		}

		if progress != nil {
			totalAnswered := stats["total_questions_answered"].(int)
			stats["total_questions_answered"] = totalAnswered + progress.TotalQuestions

			totalCorrect := stats["total_correct_answers"].(int)
			stats["total_correct_answers"] = totalCorrect + progress.CorrectAnswers
		}
	}

	totalAnswered := stats["total_questions_answered"].(int)
	if totalAnswered > 0 {
		stats["average_accuracy"] = float64(stats["total_correct_answers"].(int)) / float64(totalAnswered) * 100.0
	}

	return stats
}
