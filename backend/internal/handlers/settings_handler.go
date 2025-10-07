package handlers

import (
	"fmt"
	"net/http"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/middleware"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	"quizapp/internal/services/mailer"
	contextutils "quizapp/internal/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// SettingsHandler handles user settings related HTTP requests
type SettingsHandler struct {
	userService     services.UserServiceInterface
    storyService    services.StoryServiceInterface
	aiService       services.AIServiceInterface
	learningService services.LearningServiceInterface
	emailService    mailer.Mailer
	cfg             *config.Config
	logger          *observability.Logger
}

// NewSettingsHandler creates a new SettingsHandler instance
func NewSettingsHandler(userService services.UserServiceInterface, storyService services.StoryServiceInterface, aiService services.AIServiceInterface, learningService services.LearningServiceInterface, emailService mailer.Mailer, cfg *config.Config, logger *observability.Logger) *SettingsHandler {
    return &SettingsHandler{
        userService:     userService,
        storyService:    storyService,
        aiService:       aiService,
        learningService: learningService,
        emailService:    emailService,
        cfg:             cfg,
        logger:          logger,
    }
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
	c.JSON(http.StatusOK, h.cfg.GetLanguages())
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
