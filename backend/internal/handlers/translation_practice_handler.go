package handlers

import (
	"context"
	"fmt"
	"net/http"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/middleware"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
)

// TranslationPracticeHandler handles translation practice related HTTP requests
type TranslationPracticeHandler struct {
	translationPracticeService services.TranslationPracticeServiceInterface
	aiService                  services.AIServiceInterface
	userService                services.UserServiceInterface
	cfg                        *config.Config
	logger                     *observability.Logger
}

// convertToServicesAIConfig creates AI config for the user in services format,
// reusing the same approach as other handlers (e.g., story/quiz) including
// fetching the saved per-provider API key.
func (h *TranslationPracticeHandler) convertToServicesAIConfig(ctx context.Context, user *models.User) (*models.UserAIConfig, *int) {
	aiProvider := ""
	if user.AIProvider.Valid {
		aiProvider = user.AIProvider.String
	}
	aiModel := ""
	if user.AIModel.Valid {
		aiModel = user.AIModel.String
	}
	apiKey := ""
	var apiKeyID *int
	if aiProvider != "" {
		savedKey, keyID, err := h.userService.GetUserAPIKeyWithID(ctx, user.ID, aiProvider)
		if err == nil && savedKey != "" {
			apiKey = savedKey
			apiKeyID = keyID
		}
	}
	return &models.UserAIConfig{
		Provider: aiProvider,
		Model:    aiModel,
		APIKey:   apiKey,
		Username: user.Username,
	}, apiKeyID
}

// NewTranslationPracticeHandler creates a new TranslationPracticeHandler instance
func NewTranslationPracticeHandler(
	translationPracticeService services.TranslationPracticeServiceInterface,
	aiService services.AIServiceInterface,
	userService services.UserServiceInterface,
	cfg *config.Config,
	logger *observability.Logger,
) *TranslationPracticeHandler {
	return &TranslationPracticeHandler{
		translationPracticeService: translationPracticeService,
		aiService:                  aiService,
		userService:                userService,
		cfg:                        cfg,
		logger:                     logger,
	}
}

// GenerateSentence handles requests to generate a new sentence for translation practice
func (h *TranslationPracticeHandler) GenerateSentence(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "generate_translation_sentence")
	defer observability.FinishSpan(span, nil)

	userIDInt, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	userID := uint(userIDInt)

	var req api.TranslationPracticeGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "Invalid generate sentence request format", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Code:    stringPtr("INVALID_REQUEST"),
			Message: stringPtr("Invalid request format"),
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Get user for AI config
	user, err := h.userService.GetUserByID(ctx, int(userID))
	if err != nil || user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	userAIConfig, _ := h.convertToServicesAIConfig(ctx, user)

	// Convert API request to service request
	serviceReq := &models.GenerateSentenceRequest{
		Language:  req.Language,
		Level:     req.Level,
		Direction: models.TranslationDirection(req.Direction),
		Topic:     req.Topic,
	}

	sentence, err := h.translationPracticeService.GenerateSentence(ctx, userID, serviceReq, h.aiService, userAIConfig)
	if err != nil {
		h.logger.Error(ctx, "Failed to generate sentence", err)
		HandleAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, api.TranslationPracticeSentenceResponse{
		Id:             int(sentence.ID),
		SentenceText:   sentence.SentenceText,
		SourceLanguage: sentence.SourceLanguage,
		TargetLanguage: sentence.TargetLanguage,
		LanguageLevel:  sentence.LanguageLevel,
		SourceType:     string(sentence.SourceType),
		SourceId:       intPtrFromUintPtr(sentence.SourceID),
		Topic:          sentence.Topic,
		CreatedAt:      sentence.CreatedAt,
	})
}

// GetSentence handles requests to get a sentence from existing content
func (h *TranslationPracticeHandler) GetSentence(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_translation_sentence")
	defer observability.FinishSpan(span, nil)

	userIDInt, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	userID := uint(userIDInt)

	// Get query parameters
	language := c.Query("language")
	level := c.Query("level")
	direction := c.Query("direction")

	if language == "" || level == "" || direction == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Code:    stringPtr("INVALID_REQUEST"),
			Message: stringPtr("Missing required parameters: language, level, direction"),
		})
		return
	}

	sentence, err := h.translationPracticeService.GetSentenceFromExistingContent(
		ctx,
		userID,
		language,
		level,
		models.TranslationDirection(direction),
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to get sentence from existing content", err)
		HandleAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, api.TranslationPracticeSentenceResponse{
		Id:             int(sentence.ID),
		SentenceText:   sentence.SentenceText,
		SourceLanguage: sentence.SourceLanguage,
		TargetLanguage: sentence.TargetLanguage,
		LanguageLevel:  sentence.LanguageLevel,
		SourceType:     string(sentence.SourceType),
		SourceId:       intPtrFromUintPtr(sentence.SourceID),
		Topic:          sentence.Topic,
		CreatedAt:      sentence.CreatedAt,
	})
}

// SubmitTranslation handles requests to submit a translation for evaluation
func (h *TranslationPracticeHandler) SubmitTranslation(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "submit_translation")
	defer observability.FinishSpan(span, nil)

	userIDInt, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	userID := uint(userIDInt)

	var req api.TranslationPracticeSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "Invalid submit translation request format", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Code:    stringPtr("INVALID_REQUEST"),
			Message: stringPtr("Invalid request format"),
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Get user for AI config
	user, err := h.userService.GetUserByID(ctx, int(userID))
	if err != nil || user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	userAIConfig, _ := h.convertToServicesAIConfig(ctx, user)

	// Convert API request to service request
	serviceReq := &models.SubmitTranslationRequest{
		SentenceID:           uint(req.SentenceId),
		OriginalSentence:     req.OriginalSentence,
		UserTranslation:      req.UserTranslation,
		TranslationDirection: models.TranslationDirection(req.TranslationDirection),
	}

	session, err := h.translationPracticeService.SubmitTranslation(ctx, userID, serviceReq, h.aiService, userAIConfig)
	if err != nil {
		h.logger.Error(ctx, "Failed to submit translation", err)
		HandleAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, api.TranslationPracticeSessionResponse{
		Id:                   int(session.ID),
		SentenceId:           int(session.SentenceID),
		OriginalSentence:     session.OriginalSentence,
		UserTranslation:      session.UserTranslation,
		TranslationDirection: string(session.TranslationDirection),
		AiFeedback:           session.AIFeedback,
		AiScore:              float64PtrTo32(session.AIScore),
		CreatedAt:            session.CreatedAt,
	})
}

// GetHistory handles requests to get translation practice history
func (h *TranslationPracticeHandler) GetHistory(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_translation_practice_history")
	defer observability.FinishSpan(span, nil)

	userIDInt, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	userID := uint(userIDInt)

	limit := 50 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := parseInt(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offset := 0 // Default offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := parseInt(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	search := c.Query("search")

	sessions, total, err := h.translationPracticeService.GetPracticeHistory(ctx, userID, limit, offset, search)
	if err != nil {
		h.logger.Error(ctx, "Failed to get practice history", err)
		HandleAppError(c, err)
		return
	}

	response := make([]api.TranslationPracticeSessionResponse, len(sessions))
	for i, session := range sessions {
		response[i] = api.TranslationPracticeSessionResponse{
			Id:                   int(session.ID),
			SentenceId:           int(session.SentenceID),
			OriginalSentence:     session.OriginalSentence,
			UserTranslation:      session.UserTranslation,
			TranslationDirection: string(session.TranslationDirection),
			AiFeedback:           session.AIFeedback,
			AiScore:              float64PtrTo32(session.AIScore),
			CreatedAt:            session.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, api.TranslationPracticeHistoryResponse{
		Sessions: response,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	})
}

// GetStats handles requests to get translation practice statistics
func (h *TranslationPracticeHandler) GetStats(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_translation_practice_stats")
	defer observability.FinishSpan(span, nil)

	userIDInt, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	userID := uint(userIDInt)

	stats, err := h.translationPracticeService.GetPracticeStats(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get practice stats", err)
		HandleAppError(c, err)
		return
	}

	// Coerce nullable numeric fields to numbers to satisfy response schema
	response := map[string]interface{}{}
	// counts (always integers)
	if v, ok := stats["total_sessions"]; ok {
		response["total_sessions"] = v
	} else {
		response["total_sessions"] = 0
	}
	if v, ok := stats["excellent_count"]; ok {
		response["excellent_count"] = v
	} else {
		response["excellent_count"] = 0
	}
	if v, ok := stats["good_count"]; ok {
		response["good_count"] = v
	} else {
		response["good_count"] = 0
	}
	if v, ok := stats["needs_improvement_count"]; ok {
		response["needs_improvement_count"] = v
	} else {
		response["needs_improvement_count"] = 0
	}
	// numeric (float) values; convert nil to 0.0
	if v, ok := stats["average_score"]; ok && v != nil {
		response["average_score"] = v
	} else {
		response["average_score"] = 0.0
	}
	if v, ok := stats["min_score"]; ok && v != nil {
		response["min_score"] = v
	} else {
		response["min_score"] = 0.0
	}
	if v, ok := stats["max_score"]; ok && v != nil {
		response["max_score"] = v
	} else {
		response["max_score"] = 0.0
	}

	c.JSON(http.StatusOK, response)
}

// RegisterRoutes registers the translation practice routes with the router
func (h *TranslationPracticeHandler) RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/v1")
	{
		v1.POST("/translation-practice/generate", middleware.RequireAuth(), h.GenerateSentence)
		v1.GET("/translation-practice/sentence", middleware.RequireAuth(), h.GetSentence)
		v1.POST("/translation-practice/submit", middleware.RequireAuth(), h.SubmitTranslation)
		v1.GET("/translation-practice/history", middleware.RequireAuth(), h.GetHistory)
		v1.GET("/translation-practice/stats", middleware.RequireAuth(), h.GetStats)
	}
}

// Helper functions

func intPtrFromUintPtr(ptr *uint) *int {
	if ptr == nil {
		return nil
	}
	val := int(*ptr)
	return &val
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

func float64PtrTo32(p *float64) *float32 {
	if p == nil {
		return nil
	}
	v := float32(*p)
	return &v
}
