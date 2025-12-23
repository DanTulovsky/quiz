package handlers

import (
	"fmt"
	"net/http"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/middleware"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	serviceinterfaces "quizapp/internal/serviceinterfaces"
	"quizapp/internal/services"
	"quizapp/internal/services/mailer"
	contextutils "quizapp/internal/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// SettingsHandler handles user settings related HTTP requests
type SettingsHandler struct {
	userService                services.UserServiceInterface
	storyService               services.StoryServiceInterface
	conversationService        services.ConversationServiceInterface
	translationPracticeService services.TranslationPracticeServiceInterface
	aiService                  services.AIServiceInterface
	learningService            services.LearningServiceInterface
	usageStatsSvc              services.UsageStatsServiceInterface
	emailService               mailer.Mailer
	apnsService                serviceinterfaces.APNSService
	wordOfTheDayService        services.WordOfTheDayServiceInterface
	cfg                        *config.Config
	logger                     *observability.Logger
}

// NewSettingsHandler creates a new SettingsHandler instance
func NewSettingsHandler(userService services.UserServiceInterface, storyService services.StoryServiceInterface, conversationService services.ConversationServiceInterface, translationPracticeService services.TranslationPracticeServiceInterface, aiService services.AIServiceInterface, learningService services.LearningServiceInterface, emailService mailer.Mailer, usageStatsSvc services.UsageStatsServiceInterface, apnsService serviceinterfaces.APNSService, wordOfTheDayService services.WordOfTheDayServiceInterface, cfg *config.Config, logger *observability.Logger) *SettingsHandler {
	return &SettingsHandler{
		userService:                userService,
		storyService:               storyService,
		conversationService:        conversationService,
		translationPracticeService: translationPracticeService,
		aiService:                  aiService,
		learningService:            learningService,
		usageStatsSvc:              usageStatsSvc,
		emailService:               emailService,
		apnsService:                apnsService,
		wordOfTheDayService:        wordOfTheDayService,
		cfg:                        cfg,
		logger:                     logger,
	}
}

// UpdateWordOfDayEmailPreference updates the user's word-of-day email preference
func (h *SettingsHandler) UpdateWordOfDayEmailPreference(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "update_word_of_day_email_preference")
	defer observability.FinishSpan(span, nil)

	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request body",
			"",
			err,
		))
		return
	}

	if err := h.userService.UpdateWordOfDayEmailEnabled(ctx, userID, body.Enabled); err != nil {
		HandleAppError(c, contextutils.WrapError(err, "failed to update word of day email preference"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateUserSettings handles updating user settings
func (h *SettingsHandler) UpdateUserSettings(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "update_user_settings")
	defer observability.FinishSpan(span, nil)
	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	var settings api.UserSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid request body",
			"",
			err,
		))
		return
	}

	// Validate that at least one meaningful field is provided
	// Avoid relying on generated union/raw fields that may be non-nil for an empty JSON body
	hasAnyField := settings.Language != nil ||
		settings.Level != nil ||
		settings.AiProvider != nil ||
		settings.AiModel != nil ||
		settings.ApiKey != nil ||
		settings.AiEnabled != nil

	if !hasAnyField {
		HandleAppError(c, contextutils.ErrInvalidInput)
		return
	}

	// Convert api.UserSettings to models.UserSettings
	modelSettings := models.UserSettings{}
	if settings.Language != nil {
		modelSettings.Language = string(*settings.Language)
		span.SetAttributes(attribute.String("settings.language", modelSettings.Language))
	}
	if settings.Level != nil {
		modelSettings.Level = string(*settings.Level)
		span.SetAttributes(attribute.String("settings.level", modelSettings.Level))
	}
	if settings.AiProvider != nil {
		modelSettings.AIProvider = *settings.AiProvider
		span.SetAttributes(attribute.String("settings.ai_provider", modelSettings.AIProvider))
	}
	if settings.AiModel != nil {
		modelSettings.AIModel = *settings.AiModel
		span.SetAttributes(attribute.String("settings.ai_model", modelSettings.AIModel))
	}
	if settings.ApiKey != nil {
		modelSettings.AIAPIKey = *settings.ApiKey
		span.SetAttributes(attribute.Bool("settings.api_key_provided", true))
	}
	if settings.AiEnabled != nil {
		modelSettings.AIEnabled = *settings.AiEnabled
		span.SetAttributes(attribute.Bool("settings.ai_enabled", modelSettings.AIEnabled))
	}

	// Validate level if provided (including empty string)
	if settings.Level != nil {
		validLevels := h.cfg.GetAllLevels()
		isValidLevel := false
		for _, level := range validLevels {
			if modelSettings.Level == level {
				isValidLevel = true
				break
			}
		}

		if !isValidLevel {
			HandleAppError(c, contextutils.ErrInvalidFormat)
			return
		}
	}

	// Validate language if provided (including empty string)
	if settings.Language != nil {
		validLanguages := h.cfg.GetLanguages()
		isValidLanguage := false
		for _, language := range validLanguages {
			if modelSettings.Language == language {
				isValidLanguage = true
				break
			}
		}

		if !isValidLanguage {
			HandleAppError(c, contextutils.ErrInvalidFormat)
			return
		}
	}

	if err := h.userService.UpdateUserSettings(c.Request.Context(), userID, &modelSettings); err != nil {
		// Check if the error is due to user not found
		if contextutils.IsError(err, contextutils.ErrRecordNotFound) {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}
		HandleAppError(c, contextutils.WrapError(err, "failed to update settings"))
		return
	}

	c.JSON(http.StatusOK, api.SuccessResponse{Success: true})
}

// TestAIConnection tests the AI service connection with provided settings
func (h *SettingsHandler) TestAIConnection(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "test_ai_connection")
	defer observability.FinishSpan(span, nil)
	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	var req api.TestAIRequest
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

	// Extract values from API request
	provider := req.Provider
	model := req.Model
	apiKey := ""
	if req.ApiKey != nil {
		apiKey = *req.ApiKey
	}

	// If API key is empty, try to use the saved one from the new user_api_keys table
	if apiKey == "" {
		savedKey, err := h.userService.GetUserAPIKey(c.Request.Context(), userID, provider)
		if err != nil {
			HandleAppError(c, contextutils.WrapError(err, "failed to get saved API key"))
			return
		}
		apiKey = savedKey
	}

	err := h.aiService.TestConnection(c.Request.Context(), provider, model, apiKey)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("Model '%s': %s", model, err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Connection successful"})
}

// GetProviders returns the available AI provider configurations
func (h *SettingsHandler) GetProviders(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_providers")
	defer observability.FinishSpan(span, nil)

	response := gin.H{
		"providers": h.cfg.Providers,
		"levels":    h.cfg.GetAllLevels(),
		"languages": h.cfg.GetLanguages(),
	}
	c.JSON(http.StatusOK, response)
}

// GetLevels returns the available levels and their descriptions.
func (h *SettingsHandler) GetLevels(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_levels")
	defer observability.FinishSpan(span, nil)
	language := c.Query("language")
	if language != "" {
		levels := h.cfg.GetLevelsForLanguage(language)
		descriptions := h.cfg.GetLevelDescriptionsForLanguage(language)
		c.JSON(http.StatusOK, gin.H{
			"levels":             levels,
			"level_descriptions": descriptions,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"levels":             h.cfg.GetAllLevels(),
		"level_descriptions": h.cfg.GetAllLevelDescriptions(),
	})
}

// GetLanguages returns the available languages.
func (h *SettingsHandler) GetLanguages(c *gin.Context) {
	_, span := observability.TraceHandlerFunction(c.Request.Context(), "get_languages")
	defer observability.FinishSpan(span, nil)
	c.JSON(http.StatusOK, h.cfg.GetLanguageInfoList())
}

// CheckAPIKeyAvailability checks if the user has a saved API key for the specified provider
func (h *SettingsHandler) CheckAPIKeyAvailability(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "check_api_key_availability")
	defer observability.FinishSpan(span, nil)
	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	provider := c.Param("provider")
	if provider == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Check if user has a saved API key for this provider
	hasAPIKey, err := h.userService.HasUserAPIKey(ctx, userID, provider)
	if err != nil {
		h.logger.Error(ctx, "Failed to check API key availability", err, map[string]interface{}{
			"user_id":  userID,
			"provider": provider,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to check API key availability"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"has_api_key": hasAPIKey})
}

// GetLearningPreferences retrieves user learning preferences
func (h *SettingsHandler) GetLearningPreferences(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_learning_preferences")
	defer observability.FinishSpan(span, nil)
	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	preferences, err := h.learningService.GetUserLearningPreferences(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get learning preferences", err, map[string]interface{}{
			"user_id": userID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get learning preferences"))
		return
	}

	// Convert backend model to API schema
	apiPreferences := convertLearningPreferencesToAPI(preferences)
	c.JSON(http.StatusOK, apiPreferences)
}

// UpdateLearningPreferences updates user learning preferences
func (h *SettingsHandler) UpdateLearningPreferences(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "update_learning_preferences")
	defer observability.FinishSpan(span, nil)
	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	var req models.UserLearningPreferences
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

	// Set the user ID
	req.UserID = userID

	// Set span attributes for updated preferences
	span.SetAttributes(
		attribute.Bool("learning.focus_on_weak_areas", req.FocusOnWeakAreas),
		attribute.Bool("learning.include_review_questions", req.IncludeReviewQuestions),
		attribute.Float64("learning.fresh_question_ratio", req.FreshQuestionRatio),
		attribute.Float64("learning.known_question_penalty", req.KnownQuestionPenalty),
		attribute.Int("learning.review_interval_days", req.ReviewIntervalDays),
		attribute.Float64("learning.weak_area_boost", req.WeakAreaBoost),
	)

	// Update preferences in database
	updatedPrefs, err := h.learningService.UpdateUserLearningPreferences(ctx, userID, &req)
	if err != nil {
		h.logger.Error(ctx, "Failed to update learning preferences", err, map[string]interface{}{
			"user_id": userID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to update learning preferences"))
		return
	}

	// Convert backend model to API schema and return
	apiPreferences := convertLearningPreferencesToAPI(updatedPrefs)
	c.JSON(http.StatusOK, apiPreferences)
}

// SendTestEmail sends a test email to the current user
func (h *SettingsHandler) SendTestEmail(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "send_test_email")
	defer observability.FinishSpan(span, nil)

	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Get the current user
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user for test email", err, map[string]interface{}{
			"user_id": userID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get user information"))
		return
	}

	// Check if user has an email address
	if !user.Email.Valid || user.Email.String == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Check if email service is enabled
	if !h.emailService.IsEnabled() {
		HandleAppError(c, contextutils.ErrServiceUnavailable)
		return
	}

	// Send test email
	err = h.emailService.SendEmail(ctx, user.Email.String, "Test Email from Quiz App", "test_email", map[string]interface{}{
		"Username": user.Username,
		"TestTime": "now",
		"Message":  "This is a test email to verify your email settings are working correctly.",
	})
	if err != nil {
		h.logger.Error(ctx, "Failed to send test email", err, map[string]interface{}{
			"user_id": userID,
			"email":   user.Email.String,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to send test email"))
		return
	}

	h.logger.Info(ctx, "Test email sent successfully", map[string]interface{}{
		"user_id": userID,
		"email":   user.Email.String,
	})

	c.JSON(http.StatusOK, api.SuccessResponse{Success: true, Message: stringPtr("Test email sent successfully")})
}

// SendTestIOSNotificationRequest represents the request body for test iOS notification
type SendTestIOSNotificationRequest struct {
	NotificationType string `json:"notification_type" binding:"required,oneof=daily_reminder word_of_day"`
}

// SendTestIOSNotification sends a test iOS notification to the current user
func (h *SettingsHandler) SendTestIOSNotification(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "send_test_ios_notification")
	defer observability.FinishSpan(span, nil)

	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse request body
	var req SendTestIOSNotificationRequest
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

	// Get the current user
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user for test iOS notification", err, map[string]interface{}{
			"user_id": userID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get user information"))
		return
	}

	// Check if APNS service is enabled
	if !h.apnsService.IsEnabled() {
		HandleAppError(c, contextutils.ErrServiceUnavailable)
		return
	}

	// Check if user has device tokens
	deviceTokens, err := h.userService.GetUserDeviceTokens(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get device tokens for test iOS notification", err, map[string]interface{}{
			"user_id": userID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get device tokens"))
		return
	}

	if len(deviceTokens) == 0 {
		HandleAppError(c, contextutils.NewAppError(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"No device tokens registered",
			"Please register a device token from your iOS app first",
		))
		return
	}

	// Send test notification based on type
	var payload map[string]interface{}
	var notificationType string
	var successMessage string

	switch req.NotificationType {
	case "daily_reminder":
		payload = map[string]interface{}{
			"aps": map[string]interface{}{
				"alert": map[string]interface{}{
					"title": "Time for your daily quiz! 🧠",
					"body":  "Tap to continue your learning journey.",
				},
				"sound": "default",
			},
			"deep_link": "daily",
		}
		notificationType = "daily_reminder_ios"
		successMessage = "Test daily reminder iOS notification sent successfully"

	case "word_of_day":
		// Get word of the day for the user (use today's date)
		// Get user timezone for proper date calculation
		userTimezone := "UTC"
		if user.Timezone.Valid && user.Timezone.String != "" {
			userTimezone = user.Timezone.String
		}
		loc, err := time.LoadLocation(userTimezone)
		if err != nil {
			loc = time.UTC
		}
		today := time.Now().In(loc)

		wordOfTheDay, err := h.wordOfTheDayService.GetWordOfTheDay(ctx, userID, today)
		if err != nil {
			h.logger.Error(ctx, "Failed to get word of the day for test iOS notification", err, map[string]interface{}{
				"user_id": userID,
			})
			HandleAppError(c, contextutils.WrapError(err, "failed to get word of the day"))
			return
		}

		if wordOfTheDay == nil {
			HandleAppError(c, contextutils.NewAppError(
				contextutils.ErrorCodeInvalidInput,
				contextutils.SeverityWarn,
				"No word of the day available",
				"Please try again later",
			))
			return
		}

		payload = map[string]interface{}{
			"aps": map[string]interface{}{
				"alert": map[string]interface{}{
					"title": "Word of the Day: " + wordOfTheDay.Word,
					"body":  wordOfTheDay.Translation,
				},
				"sound": "default",
			},
			"word":        wordOfTheDay.Word,
			"translation": wordOfTheDay.Translation,
		}
		notificationType = "word_of_the_day_ios"
		successMessage = "Test word of the day iOS notification sent successfully"

	default:
		HandleAppError(c, contextutils.NewAppError(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid notification type",
			"Notification type must be 'daily_reminder' or 'word_of_day'",
		))
		return
	}

	// Send notification to all device tokens
	var sentCount int
	var failedCount int
	for _, deviceToken := range deviceTokens {
		if err := h.apnsService.SendNotification(ctx, deviceToken, payload); err != nil {
			failedCount++
			h.logger.Error(ctx, "Failed to send test iOS notification to device", err, map[string]interface{}{
				"user_id":      userID,
				"device_token": deviceToken[:20] + "...",
			})
		} else {
			sentCount++
		}
	}

	if sentCount == 0 {
		HandleAppError(c, contextutils.NewAppError(
			contextutils.ErrorCodeInternalError,
			contextutils.SeverityError,
			"Failed to send test iOS notification",
			"All device tokens failed",
		))
		return
	}

	// Record notification
	if err := h.emailService.RecordSentNotification(ctx, userID, notificationType, "Test Notification", notificationType, "sent", ""); err != nil {
		h.logger.Warn(ctx, "Failed to record test iOS notification", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
	}

	h.logger.Info(ctx, "Test iOS notification sent successfully", map[string]interface{}{
		"user_id":      userID,
		"sent_count":   sentCount,
		"failed_count": failedCount,
	})

	c.JSON(http.StatusOK, api.SuccessResponse{Success: true, Message: stringPtr(successMessage)})
}

// ClearAllStories deletes all stories belonging to the current user
func (h *SettingsHandler) ClearAllStories(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "clear_all_stories")
	defer observability.FinishSpan(span, nil)
	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	// Use the story service to delete all stories for this user
	if h.storyService == nil {
		h.logger.Warn(ctx, "Story service not available for ClearAllStories")
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Clear all stories not available",
			"",
			nil,
		))
		return
	}

	if err := h.storyService.DeleteAllStoriesForUser(ctx, uint(userID)); err != nil {
		h.logger.Error(ctx, "Failed to delete all stories for user", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to delete all stories for user"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "All stories deleted successfully"})
}

// ResetAccount deletes all stories and clears user-specific data (questions, stats)
func (h *SettingsHandler) ResetAccount(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "reset_account")
	defer observability.FinishSpan(span, nil)
	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	// Reset account: clear user data (questions, responses, metrics) and delete stories
	// First, clear user data (uses userService)
	if err := h.userService.ClearUserDataForUser(ctx, userID); err != nil {
		h.logger.Error(ctx, "Failed to clear user data for user during reset", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to clear user data"))
		return
	}

	// Then delete all stories
	if h.storyService == nil {
		h.logger.Warn(ctx, "Story service not available for ResetAccount")
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Reset account not available",
			"",
			nil,
		))
		return
	}

	if err := h.storyService.DeleteAllStoriesForUser(ctx, uint(userID)); err != nil {
		h.logger.Error(ctx, "Failed to delete stories during reset account", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to delete stories during reset"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Account reset successfully"})
}

// ClearAllAIChats deletes all AI conversations and messages for the current user
func (h *SettingsHandler) ClearAllAIChats(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "clear_all_ai_chats")
	defer observability.FinishSpan(span, nil)
	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Use the conversation service to delete all conversations for this user
	if h.conversationService == nil {
		h.logger.Warn(ctx, "Conversation service not available for ClearAllAIChats")
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Clear all AI chats not available",
			"",
			nil,
		))
		return
	}

	// Get all conversation IDs for this user
	conversations, _, err := h.conversationService.GetUserConversations(ctx, uint(userID), 1000, 0) // Get max 1000 to avoid issues
	if err != nil {
		h.logger.Error(ctx, "Failed to get user conversations for deletion", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to get user conversations for deletion"))
		return
	}

	// Delete each conversation
	deletedCount := 0
	for _, conversation := range conversations {
		err := h.conversationService.DeleteConversation(ctx, conversation.Id.String(), uint(userID))
		if err != nil {
			h.logger.Error(ctx, "Failed to delete conversation", err, map[string]interface{}{
				"user_id":         userID,
				"conversation_id": conversation.Id.String(),
			})
			// Continue with other conversations even if one fails
		} else {
			deletedCount++
		}
	}

	h.logger.Info(ctx, "Deleted AI conversations for user", map[string]interface{}{
		"user_id":       userID,
		"deleted_count": deletedCount,
		"total_count":   len(conversations),
	})

	c.JSON(http.StatusOK, api.SuccessResponse{
		Message: stringPtr(fmt.Sprintf("Deleted %d AI conversations successfully", deletedCount)),
		Success: true,
	})
}

// ClearAllTranslationPracticeHistory deletes all translation practice history for the current user
func (h *SettingsHandler) ClearAllTranslationPracticeHistory(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "clear_all_translation_practice_history")
	defer observability.FinishSpan(span, nil)
	session := sessions.Default(c)
	userID, ok := session.Get(middleware.UserIDKey).(int)
	if !ok {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Use the translation practice service to delete all practice history for this user
	if h.translationPracticeService == nil {
		h.logger.Warn(ctx, "Translation practice service not available for ClearAllTranslationPracticeHistory")
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Clear all translation practice history not available",
			"",
			nil,
		))
		return
	}

	if err := h.translationPracticeService.DeleteAllPracticeHistoryForUser(ctx, uint(userID)); err != nil {
		h.logger.Error(ctx, "Failed to delete all translation practice history for user", err, map[string]interface{}{"user_id": userID})
		HandleAppError(c, contextutils.WrapError(err, "failed to delete all translation practice history for user"))
		return
	}

	c.JSON(http.StatusOK, api.SuccessResponse{
		Message: stringPtr("All translation practice history deleted successfully"),
		Success: true,
	})
}
