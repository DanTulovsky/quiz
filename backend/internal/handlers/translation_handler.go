package handlers

import (
	"context"
	"net/http"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/middleware"
	"quizapp/internal/observability"
	"quizapp/internal/serviceinterfaces"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// stringPtrOrEmpty returns the string value if not nil, otherwise returns empty string
func stringPtrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// TranslationHandler handles translation related HTTP requests
type TranslationHandler struct {
	translationService services.TranslationServiceInterface
	cfg                *config.Config
	logger             *observability.Logger
}

// NewTranslationHandler creates a new TranslationHandler instance
func NewTranslationHandler(translationService services.TranslationServiceInterface, cfg *config.Config, logger *observability.Logger) *TranslationHandler {
	return &TranslationHandler{
		translationService: translationService,
		cfg:                cfg,
		logger:             logger,
	}
}

// TranslateText handles text translation requests
func (h *TranslationHandler) TranslateText(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "translate_text")
	defer observability.FinishSpan(span, nil)

	var req api.TranslateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "Invalid translation request format", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Code:    stringPtr("INVALID_REQUEST"),
			Message: stringPtr("Invalid request format"),
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Validate input
	if err := h.validateTranslationRequest(ctx, req); err != nil {
		h.logger.Warn(ctx, "Translation request validation failed", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Code:    stringPtr("VALIDATION_ERROR"),
			Message: stringPtr("Request validation failed"),
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Set span attributes for observability
	span.SetAttributes(
		attribute.String("translation.target_language", req.TargetLanguage),
		attribute.String("translation.source_language", stringPtrOrEmpty(req.SourceLanguage)),
		attribute.Int("translation.text_length", len(req.Text)),
	)

	// Perform translation
	response, err := h.translationService.Translate(ctx, serviceinterfaces.TranslateRequest{
		Text:           req.Text,
		TargetLanguage: req.TargetLanguage,
		SourceLanguage: stringPtrOrEmpty(req.SourceLanguage),
	})
	if err != nil {
		h.logger.Error(ctx, "Translation failed", err)

		// Check if it's a service unavailable error
		if contextutils.GetErrorCode(err) == contextutils.ErrorCodeServiceUnavailable {
			c.JSON(http.StatusServiceUnavailable, api.ErrorResponse{
				Code:    stringPtr("TRANSLATION_SERVICE_UNAVAILABLE"),
				Message: stringPtr("Translation service is currently unavailable"),
				Error:   stringPtr(err.Error()),
			})
			return
		}

		// Default to bad request for other errors
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Code:    stringPtr("TRANSLATION_FAILED"),
			Message: stringPtr("Translation failed"),
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Return successful response
	var confidencePtr *float32
	if response.Confidence > 0 {
		conf := float32(response.Confidence)
		confidencePtr = &conf
	}
	c.JSON(http.StatusOK, api.TranslateResponse{
		TranslatedText: response.TranslatedText,
		SourceLanguage: response.SourceLanguage,
		TargetLanguage: response.TargetLanguage,
		Confidence:     confidencePtr,
	})
}

// validateTranslationRequest validates the translation request
func (h *TranslationHandler) validateTranslationRequest(_ context.Context, req api.TranslateRequest) error {
	// Validate text length
	if len(req.Text) == 0 {
		return contextutils.NewAppError(contextutils.ErrorCodeInvalidInput, contextutils.SeverityError, "Text cannot be empty", "")
	}

	if len(req.Text) > 5000 {
		return contextutils.NewAppError(contextutils.ErrorCodeInvalidInput, contextutils.SeverityError, "Text cannot exceed 5000 characters", "")
	}

	// Validate target language
	if err := h.translationService.ValidateLanguageCode(req.TargetLanguage); err != nil {
		return contextutils.WrapError(err, "Invalid target language")
	}

	// Validate source language if provided
	if req.SourceLanguage != nil && *req.SourceLanguage != "" {
		if err := h.translationService.ValidateLanguageCode(*req.SourceLanguage); err != nil {
			return contextutils.WrapError(err, "Invalid source language")
		}
	}

	return nil
}

// RegisterRoutes registers the translation routes with the router
func (h *TranslationHandler) RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/v1")
	{
		v1.POST("/translate", middleware.RequireAuth(), h.TranslateText)
	}
}
