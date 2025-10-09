package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// DailyQuestionHandler handles daily question-related HTTP requests
type DailyQuestionHandler struct {
	userService          services.UserServiceInterface
	dailyQuestionService services.DailyQuestionServiceInterface
	cfg                  *config.Config
	logger               *observability.Logger
}

// NewDailyQuestionHandler creates a new DailyQuestionHandler
func NewDailyQuestionHandler(
	userService services.UserServiceInterface,
	dailyQuestionService services.DailyQuestionServiceInterface,
	cfg *config.Config,
	logger *observability.Logger,
) *DailyQuestionHandler {
	return &DailyQuestionHandler{
		userService:          userService,
		dailyQuestionService: dailyQuestionService,
		cfg:                  cfg,
		logger:               logger,
	}
}

// ParseDateInUserTimezone parses a date string in the user's timezone
func (h *DailyQuestionHandler) ParseDateInUserTimezone(ctx context.Context, userID int, dateStr string) (time.Time, string, error) {
	// Delegate to shared util with injected user lookup
	return contextutils.ParseDateInUserTimezone(ctx, userID, dateStr, h.userService.GetUserByID)
}

// GetDailyQuestions handles GET /v1/daily/questions/{date}
func (h *DailyQuestionHandler) GetDailyQuestions(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_daily_questions")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse date parameter
	dateStr := c.Param("date")
	if dateStr == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Parse date in user's timezone
	date, timezone, err := h.ParseDateInUserTimezone(ctx, userID, dateStr)
	if err != nil {
		// Check if it's an invalid date format error
		if strings.Contains(err.Error(), "invalid date format") {
			HandleAppError(c, contextutils.ErrInvalidFormat)
			return
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to get user information"))
		return
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("date", dateStr),
		attribute.String("timezone", timezone),
	)

	// Get user to check current language preferences
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user for language preference check", err, map[string]interface{}{
			"user_id": userID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get user information"))
		return
	}

	// Check if user has valid language and level preferences
	if !user.PreferredLanguage.Valid || !user.CurrentLevel.Valid {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	currentLanguage := user.PreferredLanguage.String
	currentLevel := user.CurrentLevel.String

	// Get daily questions for the date
	questions, err := h.dailyQuestionService.GetDailyQuestions(ctx, userID, date)
	if err != nil {
		h.logger.Error(ctx, "Failed to get daily questions", err, map[string]interface{}{
			"user_id":  userID,
			"date":     dateStr,
			"timezone": timezone,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get daily questions"))
		return
	}

	// Check if existing questions match current language preferences
	needsRegeneration := false
	var oldLanguage, oldLevel string

	if len(questions) == 0 {
		// No questions exist, need to generate them
		needsRegeneration = true
	} else {
		// Check if existing questions match current preferences
		oldLanguage = questions[0].Question.Language
		oldLevel = questions[0].Question.Level

		for _, assignment := range questions {
			if assignment.Question.Language != currentLanguage || assignment.Question.Level != currentLevel {
				needsRegeneration = true
				break
			}
		}
	}

	// If questions don't match current preferences, regenerate them
	if needsRegeneration {
		h.logger.Info(ctx, "Regenerating daily questions due to language preference change", map[string]interface{}{
			"user_id":      userID,
			"date":         dateStr,
			"old_language": oldLanguage,
			"old_level":    oldLevel,
			"new_language": currentLanguage,
			"new_level":    currentLevel,
		})

		// Regenerate daily questions with current preferences
		err = h.dailyQuestionService.RegenerateDailyQuestions(ctx, userID, date)
		if err != nil {
			// Check if this is a "no questions available" error
			if contextutils.IsError(err, contextutils.ErrNoQuestionsAvailable) {
				h.logger.Warn(ctx, "No questions available in preferred language, keeping existing questions", map[string]interface{}{
					"user_id":  userID,
					"date":     dateStr,
					"language": currentLanguage,
					"level":    currentLevel,
					"error":    err.Error(),
				})
				// Continue with existing questions rather than failing completely
			} else {
				h.logger.Error(ctx, "Failed to regenerate daily questions", err, map[string]interface{}{
					"user_id": userID,
					"date":    dateStr,
				})
				// Continue with existing questions rather than failing completely
				h.logger.Warn(ctx, "Continuing with existing questions due to regeneration failure", map[string]interface{}{
					"user_id": userID,
					"date":    dateStr,
				})
			}
		} else {
			// Get the regenerated questions
			questions, err = h.dailyQuestionService.GetDailyQuestions(ctx, userID, date)
			if err != nil {
				h.logger.Error(ctx, "Failed to get regenerated daily questions", err, map[string]interface{}{
					"user_id": userID,
					"date":    dateStr,
				})
				HandleAppError(c, contextutils.WrapError(err, "failed to get daily questions"))
				return
			}
		}
	}

	// Convert to API types using shared converter
	apiQuestions := convertDailyAssignmentsToAPI(ctx, questions, userID, h.userService.GetUserByID)

	c.JSON(http.StatusOK, gin.H{
		"questions": apiQuestions,
		"date":      dateStr,
	})
}

// MarkQuestionCompleted handles POST /v1/daily/questions/{date}/complete/{questionId}
func (h *DailyQuestionHandler) MarkQuestionCompleted(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "mark_daily_question_completed")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse parameters
	dateStr := c.Param("date")
	questionIDStr := c.Param("questionId")

	if dateStr == "" || questionIDStr == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Parse date in user's timezone
	date, timezone, err := h.ParseDateInUserTimezone(ctx, userID, dateStr)
	if err != nil {
		// Check if it's an invalid date format error
		if strings.Contains(err.Error(), "invalid date format") {
			HandleAppError(c, contextutils.ErrInvalidFormat)
			return
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to get user information"))
		return
	}

	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("date", dateStr),
		attribute.Int("question_id", questionID),
		attribute.String("timezone", timezone),
	)

	// Mark question as completed
	err = h.dailyQuestionService.MarkQuestionCompleted(ctx, userID, questionID, date)
	if err != nil {
		h.logger.Error(ctx, "Failed to mark daily question as completed", err, map[string]interface{}{
			"user_id":     userID,
			"question_id": questionID,
			"date":        dateStr,
			"timezone":    timezone,
		})

		// Check if the error indicates no assignment was found
		if contextutils.IsError(err, contextutils.ErrAssignmentNotFound) {
			HandleAppError(c, contextutils.ErrAssignmentNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to mark question as completed"))
		return
	}

	c.JSON(http.StatusOK, api.SuccessResponse{
		Message: stringPtr("Question marked as completed"),
	})
}

// ResetQuestionCompleted handles DELETE /v1/daily/questions/{date}/complete/{questionId}
func (h *DailyQuestionHandler) ResetQuestionCompleted(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "reset_daily_question_completed")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse parameters
	dateStr := c.Param("date")
	questionIDStr := c.Param("questionId")

	if dateStr == "" || questionIDStr == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Parse date in user's timezone
	date, timezone, err := h.ParseDateInUserTimezone(ctx, userID, dateStr)
	if err != nil {
		// Check if it's an invalid date format error
		if strings.Contains(err.Error(), "invalid date format") {
			HandleAppError(c, contextutils.ErrInvalidFormat)
			return
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to get user information"))
		return
	}

	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("date", dateStr),
		attribute.Int("question_id", questionID),
		attribute.String("timezone", timezone),
	)

	// Reset question completion status
	err = h.dailyQuestionService.ResetQuestionCompleted(ctx, userID, questionID, date)
	if err != nil {
		h.logger.Error(ctx, "Failed to reset daily question completion", err, map[string]interface{}{
			"user_id":     userID,
			"question_id": questionID,
			"date":        dateStr,
			"timezone":    timezone,
		})

		// Check if the error indicates no assignment was found
		if contextutils.IsError(err, contextutils.ErrAssignmentNotFound) {
			HandleAppError(c, contextutils.ErrAssignmentNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to reset question completion"))
		return
	}

	c.JSON(http.StatusOK, api.SuccessResponse{
		Message: stringPtr("Question completion reset"),
	})
}

// GetAvailableDates handles GET /v1/daily/dates
func (h *DailyQuestionHandler) GetAvailableDates(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_daily_available_dates")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Add span attributes for observability
	span.SetAttributes(observability.AttributeUserID(userID))

	// Get available dates with assignments
	dates, err := h.dailyQuestionService.GetAvailableDates(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get available dates", err, map[string]interface{}{
			"user_id": userID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get available dates"))
		return
	}

	// Exclude future dates based on the user's timezone (clients expect local calendar days only)
	user, _ := h.userService.GetUserByID(ctx, userID)
	tz := "UTC"
	if user != nil && user.Timezone.Valid && user.Timezone.String != "" {
		tz = user.Timezone.String
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	// Filter out dates that are after today in the user's timezone
	var filtered []time.Time
	for _, d := range dates {
		// Treat the date value as a date-only value (time component ignored)
		if !d.After(today) {
			filtered = append(filtered, d)
		}
	}

	// Convert dates to string format for JSON response
	dateStrings := make([]string, len(filtered))
	for i, date := range filtered {
		dateStrings[i] = date.Format("2006-01-02")
	}

	c.JSON(http.StatusOK, gin.H{
		"dates": dateStrings,
	})
}

// Note: Daily question assignment is now handled automatically by the worker
// when sending daily reminder emails. No manual assignment endpoint needed.

// GetDailyProgress handles GET /v1/daily/progress/{date}
func (h *DailyQuestionHandler) GetDailyProgress(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_daily_progress")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse date parameter
	dateStr := c.Param("date")
	if dateStr == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Parse date in user's timezone
	date, timezone, err := h.ParseDateInUserTimezone(ctx, userID, dateStr)
	if err != nil {
		// Check if it's an invalid date format error
		if strings.Contains(err.Error(), "invalid date format") {
			HandleAppError(c, contextutils.ErrInvalidFormat)
			return
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to get user information"))
		return
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("date", dateStr),
		attribute.String("timezone", timezone),
	)

	// Get daily progress for the date
	progress, err := h.dailyQuestionService.GetDailyProgress(ctx, userID, date)
	if err != nil {
		h.logger.Error(ctx, "Failed to get daily progress", err, map[string]interface{}{
			"user_id":  userID,
			"date":     dateStr,
			"timezone": timezone,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get daily progress"))
		return
	}

	// Convert to API type using shared converter
	apiProgress := convertDailyProgressToAPI(progress)

	c.JSON(http.StatusOK, apiProgress)
}

// SubmitDailyQuestionAnswer handles POST /v1/daily/questions/{date}/answer/{questionId}
func (h *DailyQuestionHandler) SubmitDailyQuestionAnswer(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "submit_daily_question_answer")
	defer observability.FinishSpan(span, nil)

	h.logger.Info(ctx, "SubmitDailyQuestionAnswer handler called", map[string]interface{}{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
		"params": c.Params,
	})

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse parameters
	dateStr := c.Param("date")
	questionIDStr := c.Param("questionId")

	if dateStr == "" || questionIDStr == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Parse date in user's timezone
	date, timezone, err := h.ParseDateInUserTimezone(ctx, userID, dateStr)
	if err != nil {
		// Check if it's an invalid date format error
		if strings.Contains(err.Error(), "invalid date format") {
			HandleAppError(c, contextutils.ErrInvalidFormat)
			return
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to get user information"))
		return
	}

	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Parse request body
	var requestBody api.PostV1DailyQuestionsDateAnswerQuestionIdJSONBody

	h.logger.Info(ctx, "Parsing request body", map[string]interface{}{
		"user_id":     userID,
		"question_id": questionID,
		"date":        dateStr,
		"timezone":    timezone,
	})

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		h.logger.Error(ctx, "Failed to parse request body", err, map[string]interface{}{
			"user_id":     userID,
			"question_id": questionID,
			"date":        dateStr,
			"timezone":    timezone,
			"error":       err.Error(),
		})
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request body",
			"",
			err,
		))
		return
	}

	h.logger.Info(ctx, "Request body parsed successfully",
		map[string]interface{}{
			"user_id":           userID,
			"question_id":       questionID,
			"date":              dateStr,
			"timezone":          timezone,
			"user_answer_index": requestBody.UserAnswerIndex,
		})

	// Validate user answer index
	if requestBody.UserAnswerIndex < 0 {
		h.logger.Warn(ctx, "Invalid user answer index in SubmitDailyQuestionAnswer", map[string]interface{}{"user_answer_index": requestBody.UserAnswerIndex})
		HandleAppError(c, contextutils.ErrInvalidAnswerIndex)
		return
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("date", dateStr),
		attribute.Int("question_id", questionID),
		attribute.String("timezone", timezone),
		attribute.Int("user_answer_index", requestBody.UserAnswerIndex),
	)

	// Submit the answer
	response, err := h.dailyQuestionService.SubmitDailyQuestionAnswer(
		ctx,
		userID,
		questionID,
		date,
		requestBody.UserAnswerIndex,
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to submit daily question answer", err, map[string]interface{}{
			"user_id":           userID,
			"question_id":       questionID,
			"date":              dateStr,
			"timezone":          timezone,
			"user_answer_index": requestBody.UserAnswerIndex,
		})

		// Check for specific error types
		if contextutils.IsError(err, contextutils.ErrQuestionAlreadyAnswered) {
			HandleAppError(c, contextutils.ErrQuestionAlreadyAnswered)
			return
		}
		if contextutils.IsError(err, contextutils.ErrAssignmentNotFound) {
			HandleAppError(c, contextutils.ErrAssignmentNotFound)
			return
		}
		if contextutils.IsError(err, contextutils.ErrInvalidAnswerIndex) {
			HandleAppError(c, contextutils.ErrInvalidAnswerIndex)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to submit answer"))
		return
	}

	// Add completion status to response
	responseWithCompletion := gin.H{
		"user_answer_index":    response.UserAnswerIndex,
		"user_answer":          response.UserAnswer,
		"is_correct":           response.IsCorrect,
		"correct_answer_index": response.CorrectAnswerIndex,
		"explanation":          response.Explanation,
		"is_completed":         true,
	}

	c.JSON(http.StatusOK, responseWithCompletion)
}

// GetQuestionHistory handles GET /v1/daily/questions/{questionId}/history
func (h *DailyQuestionHandler) GetQuestionHistory(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_question_history")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse question ID parameter
	questionIDStr := c.Param("questionId")
	if questionIDStr == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.Int("question_id", questionID),
	)

	// Get question history for the last 14 days
	history, err := h.dailyQuestionService.GetQuestionHistory(ctx, userID, questionID, 14)
	if err != nil {
		h.logger.Error(ctx, "Failed to get question history", err, map[string]interface{}{
			"user_id":     userID,
			"question_id": questionID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get question history"))
		return
	}

	// Determine user's timezone/location once, then filter out any future-dated assignments
	user, _ := h.userService.GetUserByID(ctx, userID)
	tz := "UTC"
	if user != nil && user.Timezone.Valid && user.Timezone.String != "" {
		tz = user.Timezone.String
	}
	loc, locErr := time.LoadLocation(tz)
	if locErr != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	// Format times in user's timezone using helper, skipping future dates
	resp := make([]map[string]interface{}, 0, len(history))
	for _, he := range history {
		// Skip future assignments in user's local date
		ad := he.AssignmentDate.In(loc)
		adDate := time.Date(ad.Year(), ad.Month(), ad.Day(), 0, 0, 0, 0, loc)
		if adDate.After(today) {
			continue
		}

		// Return assignment_date as date-only string (YYYY-MM-DD) using the stored UTC
		// date to avoid timezone ambiguity for clients.
		assignDateStr := he.AssignmentDate.UTC().Format("2006-01-02")
		span.SetAttributes(attribute.String("assignment_date.formatted_with", "date_only"))

		entry := map[string]interface{}{
			"assignment_date": assignDateStr,
			"is_completed":    he.IsCompleted,
			"is_correct":      nil,
			"submitted_at":    nil,
		}
		if he.IsCorrect != nil {
			entry["is_correct"] = *he.IsCorrect
		}
		if he.SubmittedAt != nil {
			submittedStr, _, submittedErr := contextutils.FormatTimeInUserTimezone(ctx, userID, *he.SubmittedAt, time.RFC3339, h.userService.GetUserByID)
			if submittedErr != nil || submittedStr == "" {
				h.logger.Error(ctx, "Failed to format submitted_at in user's timezone", submittedErr, map[string]interface{}{
					"user_id":         userID,
					"question_id":     questionID,
					"submitted_at_db": he.SubmittedAt,
				})
				span.RecordError(submittedErr, trace.WithStackTrace(true))
				span.SetStatus(codes.Error, "failed to format submitted_at")
				HandleAppError(c, contextutils.WrapError(submittedErr, "failed to format submitted_at"))
				return
			}
			span.SetAttributes(attribute.String("submitted_at.formatted_with", "user_timezone"))
			entry["submitted_at"] = submittedStr
		}
		resp = append(resp, entry)
	}

	c.JSON(http.StatusOK, gin.H{"history": resp})
}
