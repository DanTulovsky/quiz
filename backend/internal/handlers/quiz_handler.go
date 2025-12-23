package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"quizapp/internal/config"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// QuizHandler handles quiz-related HTTP requests including questions and answers
type QuizHandler struct {
	userService     services.UserServiceInterface
	questionService services.QuestionServiceInterface
	aiService       services.AIServiceInterface
	learningService services.LearningServiceInterface
	workerService   services.WorkerServiceInterface
	hintService     services.GenerationHintServiceInterface
	usageStatsSvc   services.UsageStatsServiceInterface
	cfg             *config.Config
	logger          *observability.Logger
}

// NewQuizHandler creates a new QuizHandler
func NewQuizHandler(
	userService services.UserServiceInterface,
	questionService services.QuestionServiceInterface,
	aiService services.AIServiceInterface,
	learningService services.LearningServiceInterface,
	workerService services.WorkerServiceInterface,
	hintService services.GenerationHintServiceInterface,
	usageStatsSvc services.UsageStatsServiceInterface,
	config *config.Config,
	logger *observability.Logger,
) *QuizHandler {
	return &QuizHandler{
		userService:     userService,
		questionService: questionService,
		aiService:       aiService,
		learningService: learningService,
		workerService:   workerService,
		hintService:     hintService,
		usageStatsSvc:   usageStatsSvc,
		cfg:             config,
		logger:          logger,
	}
}

// Deprecated: use GetUserIDFromSession in session.go
func (h *QuizHandler) getUserIDFromSession(c *gin.Context) (int, bool) {
	return GetUserIDFromSession(c)
}

// GetQuestion handles requests for quiz questions
func (h *QuizHandler) GetQuestion(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_question")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Add span attributes for observability
	span.SetAttributes(observability.AttributeUserID(userID))

	// Check if a specific question ID is requested
	questionIDStr := c.Param("id")
	if questionIDStr != "" {
		span.SetAttributes(attribute.String("question.id", questionIDStr))
		h.getSpecificQuestion(c, userID, questionIDStr)
		return
	}

	h.getNextQuestion(c, userID)
}

// getSpecificQuestion improves error handling with centralized utilities
func (h *QuizHandler) getSpecificQuestion(c *gin.Context, userID int, questionIDStr string) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_specific_question",
		observability.AttributeUserID(userID),
		attribute.String("question.id_str", questionIDStr),
	)
	defer observability.FinishSpan(span, nil)

	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid question ID format",
			"Question ID must be a valid integer",
			err,
		))
		return
	}

	questionWithStats, err := h.questionService.GetQuestionWithStats(ctx, questionID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get question with stats", err, map[string]interface{}{
			"question_id": questionID,
			"user_id":     userID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get question with stats"))
		return
	}

	// Convert and hide sensitive information
	apiQuestion, err := convertQuestionToAPI(ctx, questionWithStats.Question)
	if err != nil {
		h.logger.Error(ctx, "Failed to convert question to API", err, map[string]interface{}{
			"question_id": questionID,
			"user_id":     userID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to convert question"))
		return
	}
	apiQuestion.Explanation = nil // Hide explanation

	// Add response statistics to the API question
	apiQuestion.CorrectCount = &questionWithStats.CorrectCount
	apiQuestion.IncorrectCount = &questionWithStats.IncorrectCount
	apiQuestion.TotalResponses = &questionWithStats.TotalResponses

	// Get user-specific confidence level if available
	confidenceLevel, err := h.learningService.GetUserQuestionConfidenceLevel(ctx, userID, questionID)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get user confidence level", map[string]interface{}{
			"error":       err.Error(),
			"question_id": questionID,
			"user_id":     userID,
		})
		// Don't fail the request, just continue without confidence level
	} else if confidenceLevel != nil {
		apiQuestion.ConfidenceLevel = confidenceLevel
	}

	c.JSON(http.StatusOK, apiQuestion)
}

// getNextQuestion improves error handling with centralized utilities
func (h *QuizHandler) getNextQuestion(c *gin.Context, userID int) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_next_question",
		observability.AttributeUserID(userID),
	)
	defer observability.FinishSpan(span, nil)

	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user by ID", err, map[string]interface{}{
			"user_id": userID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get user by ID"))
		return
	}
	if user == nil {
		span.SetAttributes(attribute.String("error.type", "user_nil"))
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	// Check if user has required preferences set
	if !user.PreferredLanguage.Valid || user.PreferredLanguage.String == "" {
		span.SetAttributes(attribute.String("error.type", "missing_language_preference"))
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeMissingRequired,
			contextutils.SeverityWarn,
			"Language preference not set",
			"Please set your preferred language in settings",
			nil,
		))
		return
	}

	if !user.CurrentLevel.Valid || user.CurrentLevel.String == "" {
		span.SetAttributes(attribute.String("error.type", "missing_level_preference"))
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeMissingRequired,
			contextutils.SeverityWarn,
			"Level preference not set",
			"Please set your current level in settings",
			nil,
		))
		return
	}

	language := c.DefaultQuery("language", user.PreferredLanguage.String)
	level := c.DefaultQuery("level", user.CurrentLevel.String)

	// Handle question type selection based on query parameters
	var qType models.QuestionType
	requestedTypes := c.Query("type")
	strictTypeRequested := false

	if requestedTypes != "" {
		strictTypeRequested = true
		types := strings.Split(requestedTypes, ",")
		// Use the first valid type from the list
		for _, t := range types {
			if t = strings.TrimSpace(t); t != "" {
				qType = models.QuestionType(t)
				break
			}
		}
	} else {
		// Check if we need to exclude certain types (comma-separated list)
		excludeTypes := c.Query("exclude_type")
		if excludeTypes != "" {
			excludeList := strings.Split(excludeTypes, ",")
			var excludeSet []models.QuestionType
			for _, t := range excludeList {
				if t = strings.TrimSpace(t); t != "" {
					excludeSet = append(excludeSet, models.QuestionType(t))
				}
			}
			qType = h.selectRandomQuestionTypeExcluding(excludeSet...)
		} else {
			// Default random selection
			qType = h.selectRandomQuestionType()
		}
	}

	// Add span attributes for observability
	span.SetAttributes(
		attribute.String("language", language),
		attribute.String("level", level),
		attribute.String("question.type", string(qType)),
		attribute.Bool("strict.type.requested", strictTypeRequested),
	)

	// Get next question with fallback logic
	questionWithStats, err := h.questionService.GetNextQuestion(ctx, userID, language, level, qType)
	if err != nil {
		h.logger.Error(ctx, "Failed to get next question", err, map[string]interface{}{
			"user_id":       userID,
			"language":      language,
			"level":         level,
			"question_type": string(qType),
		})

		// Fallback: try without question type if strict type was requested
		if strictTypeRequested {
			h.logger.Info(ctx, "Attempting fallback without question type", map[string]interface{}{
				"user_id":  userID,
				"language": language,
				"level":    level,
			})
			questionWithStats, err = h.questionService.GetNextQuestion(ctx, userID, language, level, "")
			if err != nil {
				h.logger.Error(ctx, "Fallback also failed", err, map[string]interface{}{
					"user_id":  userID,
					"language": language,
					"level":    level,
				})
				HandleAppError(c, contextutils.ErrNoQuestionsAvailable)
				return
			}
		} else {
			HandleAppError(c, contextutils.ErrNoQuestionsAvailable)
			return
		}
	}

	// Check if we got a valid question
	if questionWithStats == nil || questionWithStats.Question == nil {
		h.logger.Error(ctx, "GetNextQuestion returned nil question", nil, map[string]interface{}{
			"user_id":       userID,
			"language":      language,
			"level":         level,
			"question_type": string(qType),
		})
		// If the user strictly requested a type, record a generation hint with short TTL
		if strictTypeRequested && h.hintService != nil && qType != "" {
			// Best-effort; do not fail the request if hint upsert fails
			_ = h.hintService.UpsertHint(ctx, userID, language, level, qType, 10*time.Minute)
		}
		c.JSON(http.StatusAccepted, api.GeneratingResponse{
			Status:  stringPtr("generating"),
			Message: stringPtr("No questions available. Prioritizing your requested question type. Please try again shortly."),
		})
		return
	}

	// Convert to API format and hide sensitive information
	apiQuestion, err := convertQuestionToAPI(ctx, questionWithStats.Question)
	if err != nil {
		h.logger.Error(ctx, "Failed to convert question to API", err, map[string]interface{}{
			"user_id":       userID,
			"language":      language,
			"level":         level,
			"question_type": string(qType),
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to convert question"))
		return
	}
	apiQuestion.Explanation = nil // Hide explanation

	// Add response statistics to the API question
	apiQuestion.CorrectCount = &questionWithStats.CorrectCount
	apiQuestion.IncorrectCount = &questionWithStats.IncorrectCount
	apiQuestion.TotalResponses = &questionWithStats.TotalResponses

	// Add confidence level if available
	if questionWithStats.ConfidenceLevel != nil {
		apiQuestion.ConfidenceLevel = questionWithStats.ConfidenceLevel
	}

	c.JSON(http.StatusOK, apiQuestion)
}

// SubmitAnswer improves error handling with centralized utilities
func (h *QuizHandler) SubmitAnswer(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "submit_answer")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	var req api.AnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid answer request format", err, map[string]interface{}{
			"user_id": userID,
		})
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request format",
			"",
			err,
		))
		return
	}

	// Get the question
	question, err := h.questionService.GetQuestionByID(ctx, int(req.QuestionId))
	if err != nil {
		h.logger.Error(ctx, "Failed to get question by ID", err, map[string]interface{}{
			"question_id": req.QuestionId,
			"user_id":     userID,
		})
		HandleAppError(c, contextutils.ErrQuestionNotFound)
		return
	}

	// Check if answer is correct
	isCorrect := int(req.UserAnswerIndex) == question.CorrectAnswer

	// Record user response
	responseTimeMs := 0
	if req.ResponseTimeMs != nil {
		responseTimeMs = int(*req.ResponseTimeMs)
	}

	// Use priority-aware recording to ensure priority scores are updated
	// Store the user's answer index for future reference
	if err := h.learningService.RecordAnswerWithPriority(ctx, userID, int(req.QuestionId), int(req.UserAnswerIndex), isCorrect, responseTimeMs); err != nil {
		h.logger.Error(ctx, "Failed to record user response", err, map[string]interface{}{
			"user_id":     userID,
			"question_id": req.QuestionId,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to record response"))
		return
	}

	// Prepare response
	// Get the user's answer text from the question options
	userAnswerText := ""
	if optionsRaw, ok := question.Content["options"]; ok {
		if options, ok := optionsRaw.([]interface{}); ok {
			if int(req.UserAnswerIndex) >= 0 && int(req.UserAnswerIndex) < len(options) {
				if optStr, ok := options[int(req.UserAnswerIndex)].(string); ok {
					userAnswerText = optStr
				}
			}
		}
	}

	answerResponse := &api.AnswerResponse{
		IsCorrect:          &isCorrect,
		UserAnswer:         &userAnswerText,
		UserAnswerIndex:    &req.UserAnswerIndex,
		Explanation:        &question.Explanation,
		CorrectAnswerIndex: &question.CorrectAnswer,
	}

	c.JSON(http.StatusOK, answerResponse)

	// Add span attributes for observability
	span.SetAttributes(
		attribute.Int("user.id", userID),
		attribute.Int("question.id", int(req.QuestionId)),
		attribute.Bool("answer.is_correct", isCorrect),
		attribute.Int("response.time_ms", responseTimeMs),
	)
}

// GetProgress improves error handling with centralized utilities
func (h *QuizHandler) GetProgress(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_progress")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	span.SetAttributes(observability.AttributeUserID(userID))

	progress, err := h.learningService.GetUserProgress(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user progress", err, map[string]interface{}{
			"user_id": userID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get progress"))
		return
	}

	// Get worker status information
	workerStatus, err := h.getWorkerStatusForUser(ctx, userID)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get worker status for user", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		// Don't fail the entire request, just log the warning
	}

	// Get learning preferences
	learningPrefs, err := h.learningService.GetUserLearningPreferences(ctx, userID)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get learning preferences for user", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		// Don't fail the entire request, just log the warning
	}

	// Get priority insights
	priorityInsights, err := h.getPriorityInsightsForUser(ctx, userID)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get priority insights for user", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		// Don't fail the entire request, just log the warning
	}

	// Get generation focus information
	generationFocus, err := h.getGenerationFocusForUser(ctx, userID)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get generation focus for user", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		// Don't fail the entire request, just log the warning
	}

	// Get high priority topics
	highPriorityTopics, err := h.getHighPriorityTopicsForUser(ctx, userID)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get high priority topics for user", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		// Don't fail the entire request, just log the warning
	}

	// Get gap analysis
	gapAnalysis, err := h.getGapAnalysisForUser(ctx, userID)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get gap analysis for user", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		// Don't fail the entire request, just log the warning
	}

	// Get priority distribution
	priorityDistribution, err := h.getPriorityDistributionForUser(ctx, userID)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get priority distribution for user", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		// Don't fail the entire request, just log the warning
	}

	// Convert models.UserProgress to api.UserProgress
	apiProgress := convertUserProgressToAPI(ctx, progress, userID, h.userService.GetUserByID)

	// Add worker-related information
	if workerStatus != nil {
		apiProgress.WorkerStatus = workerStatus
	}
	if learningPrefs != nil {
		apiProgress.LearningPreferences = convertLearningPreferencesToAPI(learningPrefs)
	}
	if priorityInsights != nil {
		apiProgress.PriorityInsights = priorityInsights
	}
	if generationFocus != nil {
		apiProgress.GenerationFocus = generationFocus
	}
	if highPriorityTopics != nil {
		apiProgress.HighPriorityTopics = &highPriorityTopics
	}
	if gapAnalysis != nil {
		apiProgress.GapAnalysis = &gapAnalysis
	}
	if priorityDistribution != nil {
		apiProgress.PriorityDistribution = &priorityDistribution
	}

	c.JSON(http.StatusOK, apiProgress)
}

// GetAITokenUsage returns AI token usage statistics for the authenticated user
func (h *QuizHandler) GetAITokenUsage(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_ai_token_usage")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		span.SetAttributes(attribute.String("error", "no_user_session"))
		HandleAppError(c, contextutils.WrapError(contextutils.ErrUnauthorized, "user not authenticated"))
		return
	}
	span.SetAttributes(observability.AttributeUserID(userID))

	startDateStr := c.Query("startDate")
	if startDateStr == "" {
		span.SetAttributes(attribute.String("error", "missing_start_date"))
		HandleAppError(c, contextutils.WrapError(contextutils.ErrInvalidInput, "startDate parameter is required"))
		return
	}

	endDateStr := c.Query("endDate")
	if endDateStr == "" {
		span.SetAttributes(attribute.String("error", "missing_end_date"))
		HandleAppError(c, contextutils.WrapError(contextutils.ErrInvalidInput, "endDate parameter is required"))
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_start_date"))
		HandleAppError(c, contextutils.WrapErrorf(contextutils.ErrInvalidInput, "invalid startDate format: %v", err))
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_end_date"))
		HandleAppError(c, contextutils.WrapErrorf(contextutils.ErrInvalidInput, "invalid endDate format: %v", err))
		return
	}

	// Get usage stats
	stats, err := h.usageStatsSvc.GetUserAITokenUsageStats(ctx, userID, startDate, endDate)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user AI token usage stats", err, map[string]any{
			"user_id":    userID,
			"start_date": startDateStr,
			"end_date":   endDateStr,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get AI token usage stats"))
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetAITokenUsageDaily returns daily aggregated AI token usage for the authenticated user
func (h *QuizHandler) GetAITokenUsageDaily(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_ai_token_usage_daily")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		span.SetAttributes(attribute.String("error", "no_user_session"))
		HandleAppError(c, contextutils.WrapError(contextutils.ErrUnauthorized, "user not authenticated"))
		return
	}
	span.SetAttributes(observability.AttributeUserID(userID))

	startDateStr := c.Query("startDate")
	if startDateStr == "" {
		span.SetAttributes(attribute.String("error", "missing_start_date"))
		HandleAppError(c, contextutils.WrapError(contextutils.ErrInvalidInput, "startDate parameter is required"))
		return
	}

	endDateStr := c.Query("endDate")
	if endDateStr == "" {
		span.SetAttributes(attribute.String("error", "missing_end_date"))
		HandleAppError(c, contextutils.WrapError(contextutils.ErrInvalidInput, "endDate parameter is required"))
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_start_date"))
		HandleAppError(c, contextutils.WrapErrorf(contextutils.ErrInvalidInput, "invalid startDate format: %v", err))
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_end_date"))
		HandleAppError(c, contextutils.WrapErrorf(contextutils.ErrInvalidInput, "invalid endDate format: %v", err))
		return
	}

	// Get daily usage stats
	stats, err := h.usageStatsSvc.GetUserAITokenUsageStatsByDay(ctx, userID, startDate, endDate)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user AI token usage stats by day", err, map[string]interface{}{
			"user_id":    userID,
			"start_date": startDateStr,
			"end_date":   endDateStr,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get daily AI token usage stats"))
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetAITokenUsageHourly returns hourly aggregated AI token usage for the authenticated user on a specific day
func (h *QuizHandler) GetAITokenUsageHourly(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_ai_token_usage_hourly")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		span.SetAttributes(attribute.String("error", "no_user_session"))
		HandleAppError(c, contextutils.WrapError(contextutils.ErrUnauthorized, "user not authenticated"))
		return
	}
	span.SetAttributes(observability.AttributeUserID(userID))

	dateStr := c.Query("date")
	if dateStr == "" {
		span.SetAttributes(attribute.String("error", "missing_date"))
		HandleAppError(c, contextutils.WrapError(contextutils.ErrInvalidInput, "date parameter is required"))
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_date"))
		HandleAppError(c, contextutils.WrapErrorf(contextutils.ErrInvalidInput, "invalid date format: %v", err))
		return
	}

	// Get hourly usage stats
	stats, err := h.usageStatsSvc.GetUserAITokenUsageStatsByHour(ctx, userID, date)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user AI token usage stats by hour", err, map[string]interface{}{
			"user_id": userID,
			"date":    dateStr,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get hourly AI token usage stats"))
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ReportQuestion improves error handling with centralized utilities
func (h *QuizHandler) ReportQuestion(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "report_question")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	questionIDStr := c.Param("id")
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleValidationError(c, "question_id", questionIDStr, "must be a valid integer")
		return
	}

	// Parse request body for report reason
	var req struct {
		ReportReason *string `json:"report_reason"`
	}

	// Bind JSON if present (optional)
	if err := c.ShouldBindJSON(&req); err != nil {
		// Ignore binding errors for optional request body
		req.ReportReason = nil
	}

	// Get report reason, default to empty string if not provided
	reportReason := ""
	if req.ReportReason != nil {
		reportReason = *req.ReportReason
	}

	span.SetAttributes(
		observability.AttributeUserID(userID),
		observability.AttributeQuestionID(questionID),
	)

	err = h.questionService.ReportQuestion(ctx, questionID, userID, reportReason)
	if err != nil {
		h.logger.Error(ctx, "Failed to report question", err, map[string]interface{}{
			"question_id": questionID,
			"user_id":     userID,
		})
		if contextutils.IsError(err, contextutils.ErrRecordNotFound) {
			HandleAppError(c, contextutils.ErrQuestionNotFound)
			return
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to report question"))
		return
	}

	c.JSON(http.StatusOK, api.SuccessResponse{Success: true, Message: stringPtr("Question reported successfully")})
}

// MarkQuestionAsKnown improves error handling with centralized utilities
func (h *QuizHandler) MarkQuestionAsKnown(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "mark_question_as_known")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	questionIDStr := c.Param("id")
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleValidationError(c, "question_id", questionIDStr, "must be a valid integer")
		return
	}

	// Optional: Parse confidence level from request body
	var req struct {
		ConfidenceLevel *int `json:"confidence_level"`
	}

	// Bind JSON if present (optional)
	if err := c.ShouldBindJSON(&req); err != nil {
		// Ignore binding errors for optional request body
		req.ConfidenceLevel = nil
	}

	span.SetAttributes(
		observability.AttributeUserID(userID),
		observability.AttributeQuestionID(questionID),
	)

	// Mark question as known with confidence level
	err = h.learningService.MarkQuestionAsKnown(ctx, userID, questionID, req.ConfidenceLevel)
	if err != nil {
		h.logger.Error(ctx, "Failed to mark question as known for user", err, map[string]interface{}{
			"question_id": questionID,
			"user_id":     userID,
		})
		if contextutils.IsError(err, contextutils.ErrQuestionNotFound) {
			HandleAppError(c, contextutils.ErrQuestionNotFound)
			return
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to mark question as known"))
		return
	}

	c.JSON(http.StatusOK, api.SuccessResponse{Success: true, Message: stringPtr("Question marked as known successfully")})
}

// ChatStream handles requests for AI-powered streaming chat about a question
func (h *QuizHandler) ChatStream(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "chat_stream")
	defer observability.FinishSpan(span, nil)

	userID, exists := h.getUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	var req api.QuizChatRequest
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

	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("ai.provider", user.AIProvider.String),
		attribute.String("ai.model", user.AIModel.String),
	)

	// Prepare the request for the AI service
	aiReq := &models.AIChatRequest{
		Language:     string(*req.Question.Language),
		Level:        string(*req.Question.Level),
		QuestionType: models.QuestionType(*req.Question.Type),
		UserMessage:  req.UserMessage,
	}

	if req.Question.Content != nil {
		// For fill_blank questions, use sentence if question is not available
		if req.Question.Content.Question != nil {
			aiReq.Question = *req.Question.Content.Question
		} else if req.Question.Content.Sentence != nil {
			aiReq.Question = *req.Question.Content.Sentence
		}
		aiReq.Options = req.Question.Content.Options
		if req.Question.Content.Passage != nil {
			aiReq.Passage = *req.Question.Content.Passage
		}
		// For vocabulary questions, use the sentence field as the passage
		if req.Question.Content.Sentence != nil && req.Question.Type != nil && *req.Question.Type == "vocabulary" {
			aiReq.Passage = *req.Question.Content.Sentence
		}
	}

	if req.AnswerContext != nil {
		if req.AnswerContext.UserAnswer != nil {
			aiReq.UserAnswer = *req.AnswerContext.UserAnswer
		}
		if req.AnswerContext.IsCorrect != nil {
			aiReq.IsCorrect = req.AnswerContext.IsCorrect
		}
	}

	// Include conversation history if provided
	if req.ConversationHistory != nil {
		aiReq.ConversationHistory = make([]models.ChatMessage, len(*req.ConversationHistory))
		for i, msg := range *req.ConversationHistory {
			// Extract text content from the object
			contentText := ""
			if msg.Content.Text != nil {
				contentText = *msg.Content.Text
			}
			aiReq.ConversationHistory[i] = models.ChatMessage{
				Role:    msg.Role,
				Content: contentText,
			}
		}
	}

	// Create user AI configuration
	userConfig := &models.UserAIConfig{
		Provider: "", // will be set from user settings
		Model:    "", // use service default
		APIKey:   "",
		Username: user.Username,
	}
	if user.AIProvider.Valid && user.AIProvider.String != "" {
		userConfig.Provider = user.AIProvider.String
	}
	if user.AIModel.Valid && user.AIModel.String != "" {
		userConfig.Model = user.AIModel.String
	}
	// Use the new per-provider API key system instead of the old user.AIAPIKey field
	var apiKeyID *int
	if userConfig.Provider != "" {
		savedKey, keyID, err := h.userService.GetUserAPIKeyWithID(c.Request.Context(), userID, userConfig.Provider)
		if err == nil && savedKey != "" {
			userConfig.APIKey = savedKey
			apiKeyID = keyID
		}
	}

	// Set up Server-Sent Events headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Create a channel for streaming chunks
	chunks := make(chan string, 10)

	// Use the request context to detect client disconnect
	reqCtx := c.Request.Context()

	// Create a timeout context, but also watch for client disconnect
	timeoutCtx, cancel := context.WithTimeout(reqCtx, config.QuizStreamTimeout)
	defer cancel()

	// Combine both contexts - cancel if either times out or client disconnects
	ctx, combinedCancel := context.WithCancel(timeoutCtx)
	defer combinedCancel()

	// Store userID and apiKeyID in context for usage tracking
	// This context will be used by the AI service for usage tracking
	ctx = contextutils.WithUserID(ctx, userID)
	if apiKeyID != nil {
		ctx = contextutils.WithAPIKeyID(ctx, *apiKeyID)
	}

	// Watch for client disconnect
	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error(ctx, "Panic in client disconnect watcher", nil, map[string]any{
					"panic": r,
				})
			}
		}()
		select {
		case <-reqCtx.Done():
			combinedCancel() // Cancel if client disconnects
		case <-ctx.Done():
			// Context already cancelled
		}
	}()

	// Start the AI streaming in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error(ctx, "Panic in AI streaming goroutine", nil, map[string]interface{}{
					"panic": r,
				})
			}
			close(chunks) // Close the channel when the goroutine completes
		}()
		if err := h.aiService.GenerateChatResponseStream(ctx, userConfig, aiReq, chunks); err != nil {
			h.logger.Error(ctx, "AI chat streaming failed for user", err, map[string]interface{}{
				"user_id": contextutils.GetUserIDFromContext(ctx),
			})
			// Only send error if context is not cancelled (avoid sending to closed channel)
			if ctx.Err() == nil {
				select {
				case chunks <- fmt.Sprintf("ERROR: %v", err):
				default:
					// Channel full, skip sending error
				}
			}
		}
	}()

	// Stream the response chunks
	c.Stream(func(w io.Writer) bool {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				// Channel closed, end streaming
				return false
			}

			// Handle error messages
			if strings.HasPrefix(chunk, "ERROR: ") {
				c.SSEvent("error", chunk[7:]) // Remove "ERROR: " prefix
				return false
			}

			// Marshal the chunk to JSON to ensure newlines and special characters are preserved.
			jsonChunk, err := json.Marshal(chunk)
			if err != nil {
				h.logger.Error(ctx, "Failed to marshal chat stream chunk to JSON", err)
				return true // Continue streaming, skip this chunk
			}

			// Send normal content chunk in proper SSE format
			if _, err := fmt.Fprintf(w, "data: %s\n\n", jsonChunk); err != nil {
				h.logger.Error(ctx, "Failed to write chat stream data", err)
				return false
			}
			c.Writer.Flush()
			return true
		case <-ctx.Done():
			c.SSEvent("error", "Request timeout")
			return false
		}
	})
}

// Helper methods

func (h *QuizHandler) selectRandomQuestionType() models.QuestionType {
	// Note: This is a pure function that doesn't need tracing since it doesn't make external calls
	types := []models.QuestionType{
		models.Vocabulary,
		models.FillInBlank,
		models.QuestionAnswer,
		models.ReadingComprehension,
	}
	return types[rand.Intn(len(types))]
}

// selectRandomQuestionTypeExcluding returns a random question type excluding the specified types
func (h *QuizHandler) selectRandomQuestionTypeExcluding(excludeTypes ...models.QuestionType) models.QuestionType {
	availableTypes := []models.QuestionType{
		models.Vocabulary,
		models.FillInBlank,
		models.QuestionAnswer,
		models.ReadingComprehension,
	}

	// Filter out excluded types
	for _, excludeType := range excludeTypes {
		for i, availableType := range availableTypes {
			if availableType == excludeType {
				availableTypes = append(availableTypes[:i], availableTypes[i+1:]...)
				break
			}
		}
	}

	if len(availableTypes) == 0 {
		return models.Vocabulary // Default fallback
	}

	return availableTypes[rand.Intn(len(availableTypes))]
}

// GetWorkerStatus returns worker status and error information for the current user
func (h *QuizHandler) GetWorkerStatus(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_worker_status")
	defer observability.FinishSpan(span, nil)

	userID, exists := h.getUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	span.SetAttributes(observability.AttributeUserID(userID))

	// Get worker health information
	workerHealth, err := h.workerService.GetWorkerHealth(ctx)
	if err != nil {
		h.logger.Error(ctx, "Failed to get worker health", err)
		HandleAppError(c, contextutils.WrapError(err, "failed to get worker status"))
		return
	}

	// Check if user is paused
	userPaused, err := h.workerService.IsUserPaused(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to check user pause status", err, nil)
		userPaused = false // Default to not paused if check fails
	}

	// Check if global pause is active
	globalPaused, err := h.workerService.IsGlobalPaused(ctx)
	if err != nil {
		h.logger.Error(ctx, "Failed to check global pause status", err, nil)
		globalPaused = false // Default to not paused if check fails
	}

	// Extract relevant information for the user
	response := gin.H{
		"has_errors":         false,
		"error_message":      "",
		"global_paused":      globalPaused,
		"user_paused":        userPaused,
		"healthy_workers":    workerHealth["healthy_count"],
		"total_workers":      workerHealth["total_count"],
		"last_error_details": "",
		"worker_running":     false,
	}

	// Check for worker errors
	if workerInstances, ok := workerHealth["worker_instances"].([]map[string]interface{}); ok {
		for _, instance := range workerInstances {
			if lastError, hasError := instance["last_run_error"]; hasError && lastError != nil {
				// Only handle string type
				if errorStr, ok := lastError.(string); ok && errorStr != "" {
					response["has_errors"] = true
					response["error_message"] = "Worker encountered errors during question generation"
					response["last_error_details"] = errorStr
					break
				}
			}
			if isRunning, ok := instance["is_running"].(bool); ok && isRunning {
				response["worker_running"] = true
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions for enhanced progress information

func (h *QuizHandler) getWorkerStatusForUser(ctx context.Context, userID int) (*api.WorkerStatus, error) {
	// Get worker health information
	workerHealth, err := h.workerService.GetWorkerHealth(ctx)
	if err != nil {
		return nil, err
	}

	// Check if user is paused
	userPaused, err := h.workerService.IsUserPaused(ctx, userID)
	if err != nil {
		userPaused = false // Default to not paused if check fails
	}

	// Check if global pause is active
	globalPaused, err := h.workerService.IsGlobalPaused(ctx)
	if err != nil {
		globalPaused = false // Default to not paused if check fails
	}

	// Determine worker status
	var status api.WorkerStatusStatus
	var errorMessage *string

	if globalPaused {
		status = api.Idle // Use idle for paused state
	} else if userPaused {
		status = api.Idle // Use idle for paused state
	} else {
		status = api.Idle // Default to idle
		// Check for worker errors and actual activity
		if workerInstances, ok := workerHealth["worker_instances"].([]map[string]interface{}); ok {
			for _, instance := range workerInstances {
				// Check for errors first - errors take priority
				if lastError, hasError := instance["last_run_error"]; hasError && lastError != nil {
					if errorStr, ok := lastError.(string); ok && errorStr != "" {
						// Set status to error when there are errors
						status = api.Error
						errorMessage = &errorStr
						break
					}
				}

				// Only check for busy status if we haven't found an error
				if status != api.Error {
					// Check if worker is running AND has recent activity
					if isRunning, ok := instance["is_running"].(bool); ok && isRunning {
						// Only set to busy if the worker is actually active (not just running but idle)
						// We'll check if there's recent activity or if the worker is actively generating
						if lastHeartbeat, hasHeartbeat := instance["last_heartbeat"]; hasHeartbeat && lastHeartbeat != nil {
							if heartbeatStr, ok := lastHeartbeat.(string); ok {
								if heartbeat, err := time.Parse(time.RFC3339, heartbeatStr); err == nil {
									// Consider busy if heartbeat is very recent (within last 30 seconds)
									if time.Since(heartbeat) < 30*time.Second {
										status = api.Busy
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Get last heartbeat
	var lastHeartbeat *time.Time
	if workerInstances, ok := workerHealth["worker_instances"].([]map[string]interface{}); ok && len(workerInstances) > 0 {
		if heartbeatStr, ok := workerInstances[0]["last_heartbeat"].(string); ok {
			if heartbeat, err := time.Parse(time.RFC3339, heartbeatStr); err == nil {
				lastHeartbeat = &heartbeat
			}
		}
	}

	return &api.WorkerStatus{
		Status:        &status,
		LastHeartbeat: formatTimePointer(lastHeartbeat),
		ErrorMessage:  errorMessage,
	}, nil
}

func (h *QuizHandler) getPriorityInsightsForUser(ctx context.Context, userID int) (*api.PriorityInsights, error) {
	// Get priority distribution for the user
	priorityDistribution, err := h.learningService.GetUserPriorityScoreDistribution(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Extract counts from distribution
	highCount := 0
	mediumCount := 0
	lowCount := 0
	totalCount := 0

	if high, ok := priorityDistribution["high"].(int); ok {
		highCount = high
		totalCount += high
	}
	if medium, ok := priorityDistribution["medium"].(int); ok {
		mediumCount = medium
		totalCount += medium
	}
	if low, ok := priorityDistribution["low"].(int); ok {
		lowCount = low
		totalCount += low
	}

	return &api.PriorityInsights{
		TotalQuestionsInQueue:   &totalCount,
		HighPriorityQuestions:   &highCount,
		MediumPriorityQuestions: &mediumCount,
		LowPriorityQuestions:    &lowCount,
	}, nil
}

func (h *QuizHandler) getGenerationFocusForUser(ctx context.Context, userID int) (*api.GenerationFocus, error) {
	// Get user's AI configuration
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get current generation model
	model := "default"
	if user.AIModel.Valid && user.AIModel.String != "" {
		model = user.AIModel.String
	}

	// Get last generation time (simplified - could be enhanced with actual generation logs)
	lastGenerationTime := time.Now().Add(-time.Hour) // Placeholder

	// Get generation rate (simplified - could be enhanced with actual metrics)
	generationRate := float32(2.5) // Placeholder: average questions per minute

	return &api.GenerationFocus{
		CurrentGenerationModel: &model,
		LastGenerationTime:     formatTimePtr(lastGenerationTime),
		GenerationRate:         &generationRate,
	}, nil
}

func (h *QuizHandler) getHighPriorityTopicsForUser(ctx context.Context, userID int) ([]string, error) {
	// Get high priority topics from learning service
	topics, err := h.learningService.GetHighPriorityTopics(ctx, userID)
	if err != nil {
		return nil, err
	}
	return topics, nil
}

func (h *QuizHandler) getGapAnalysisForUser(ctx context.Context, userID int) (map[string]interface{}, error) {
	// Get gap analysis from learning service
	gapAnalysis, err := h.learningService.GetGapAnalysis(ctx, userID)
	if err != nil {
		return nil, err
	}
	return gapAnalysis, nil
}

func (h *QuizHandler) getPriorityDistributionForUser(ctx context.Context, userID int) (map[string]int, error) {
	// Get priority distribution from learning service
	distribution, err := h.learningService.GetPriorityDistribution(ctx, userID)
	if err != nil {
		return nil, err
	}
	return distribution, nil
}

func convertLearningPreferencesToAPI(prefs *models.UserLearningPreferences) *api.UserLearningPreferences {
	out := &api.UserLearningPreferences{
		FocusOnWeakAreas:              prefs.FocusOnWeakAreas,
		FreshQuestionRatio:            float32(prefs.FreshQuestionRatio),
		KnownQuestionPenalty:          float32(prefs.KnownQuestionPenalty),
		ReviewIntervalDays:            prefs.ReviewIntervalDays,
		WeakAreaBoost:                 float32(prefs.WeakAreaBoost),
		DailyReminderEnabled:          prefs.DailyReminderEnabled,
		WordOfDayIosNotifyEnabled:     &prefs.WordOfDayIOSNotifyEnabled,
		DailyReminderIosNotifyEnabled: &prefs.DailyReminderIOSNotifyEnabled,
	}
	if prefs.TTSVoice != "" {
		v := prefs.TTSVoice
		out.TtsVoice = &v
	}
	if prefs.DailyGoal > 0 {
		dg := prefs.DailyGoal
		out.DailyGoal = &dg
	}
	return out
}
