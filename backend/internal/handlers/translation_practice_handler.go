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
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// TranslationPracticeHandler handles translation practice related HTTP requests
type TranslationPracticeHandler struct {
	translationPracticeService services.TranslationPracticeServiceInterface
	aiService                  services.AIServiceInterface
	userService                services.UserServiceInterface
	cfg                        *config.Config
	logger                     *observability.Logger
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
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	userAIConfig := &models.UserAIConfig{
		Provider: user.AIProvider.String,
		Model:    user.AIModel.String,
		APIKey:   user.AIAPIKey.String,
		Username: user.Username,
	}

	// Convert API request to service request
	serviceReq := &models.GenerateSentenceRequest{
		Language:  req.Language,
		Level:      req.Level,
		Direction:  models.TranslationDirection(req.Direction),
		Topic:      req.Topic,
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
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		HandleAppError(c, contextutils.ErrRecordNotFound)
		return
	}

	userAIConfig := &models.UserAIConfig{
		Provider: user.AIProvider.String,
		Model:    user.AIModel.String,
		APIKey:   user.AIAPIKey.String,
		Username: user.Username,
	}

	// Convert API request to service request
	serviceReq := &models.SubmitTranslationRequest{
		SentenceID:          uint(req.SentenceId),
		OriginalSentence:    req.OriginalSentence,
		UserTranslation:     req.UserTranslation,
		TranslationDirection: models.TranslationDirection(req.TranslationDirection),
	}

	session, err := h.translationPracticeService.SubmitTranslation(ctx, userID, serviceReq, h.aiService, userAIConfig)
	if err != nil {
		h.logger.Error(ctx, "Failed to submit translation", err)
		HandleAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, api.TranslationPracticeSessionResponse{
		Id:                 int(session.ID),
		SentenceId:         int(session.SentenceID),
		OriginalSentence:   session.OriginalSentence,
		UserTranslation:    session.UserTranslation,
		TranslationDirection: string(session.TranslationDirection),
		AiFeedback:         session.AIFeedback,
		AiScore:            session.AIScore,
		CreatedAt:          session.CreatedAt,
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

	sessions, err := h.translationPracticeService.GetPracticeHistory(ctx, userID, limit)
	if err != nil {
		h.logger.Error(ctx, "Failed to get practice history", err)
		HandleAppError(c, err)
		return
	}

	response := make([]api.TranslationPracticeSessionResponse, len(sessions))
	for i, session := range sessions {
		response[i] = api.TranslationPracticeSessionResponse{
			Id:                 int(session.ID),
			SentenceId:         int(session.SentenceID),
			OriginalSentence:   session.OriginalSentence,
			UserTranslation:    session.UserTranslation,
			TranslationDirection: string(session.TranslationDirection),
			AiFeedback:         session.AIFeedback,
			AiScore:            session.AIScore,
			CreatedAt:          session.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, api.TranslationPracticeHistoryResponse{
		Sessions: response,
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

	c.JSON(http.StatusOK, stats)
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

