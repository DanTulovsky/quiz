package handlers

import (
	"net/http"
	"strconv"

	"quizapp/internal/middleware"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// AuthAPIKeyHandler handles authentication API key related HTTP requests
type AuthAPIKeyHandler struct {
	apiKeyService services.AuthAPIKeyServiceInterface
	logger        *observability.Logger
}

// NewAuthAPIKeyHandler creates a new AuthAPIKeyHandler instance
func NewAuthAPIKeyHandler(apiKeyService services.AuthAPIKeyServiceInterface, logger *observability.Logger) *AuthAPIKeyHandler {
	return &AuthAPIKeyHandler{
		apiKeyService: apiKeyService,
		logger:        logger,
	}
}

// CreateAPIKey handles POST /v1/api-keys
func (h *AuthAPIKeyHandler) CreateAPIKey(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "CreateAPIKey")
	defer observability.FinishSpan(span, nil)

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	userIDInt, ok := userID.(int)
	if !ok {
		HandleAppError(c, contextutils.ErrInternalError)
		return
	}

	span.SetAttributes(attribute.Int("user_id", userIDInt))

	// Parse request body
	var req struct {
		KeyName         string `json:"key_name" binding:"required"`
		PermissionLevel string `json:"permission_level" binding:"required"`
	}

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

	span.SetAttributes(
		attribute.String("key_name", req.KeyName),
		attribute.String("permission_level", req.PermissionLevel),
	)

	// Create API key
	apiKey, rawKey, err := h.apiKeyService.CreateAPIKey(ctx, userIDInt, req.KeyName, req.PermissionLevel)
	if err != nil {
		h.logger.Error(ctx, "Failed to create API key", err, map[string]interface{}{
			"user_id":          userIDInt,
			"key_name":         req.KeyName,
			"permission_level": req.PermissionLevel,
		})
		HandleAppError(c, err)
		return
	}

	span.SetAttributes(attribute.Int("api_key_id", apiKey.ID))

	// Return the full key ONCE (this is the only time it will be shown)
	c.JSON(http.StatusCreated, gin.H{
		"id":               apiKey.ID,
		"key_name":         apiKey.KeyName,
		"key":              rawKey, // Full key - only shown once!
		"key_prefix":       apiKey.KeyPrefix,
		"permission_level": apiKey.PermissionLevel,
		"created_at":       apiKey.CreatedAt,
		"message":          "Save this API key now. You won't be able to see it again!",
	})
}

// ListAPIKeys handles GET /v1/api-keys
func (h *AuthAPIKeyHandler) ListAPIKeys(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "ListAPIKeys")
	defer observability.FinishSpan(span, nil)

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	userIDInt, ok := userID.(int)
	if !ok {
		HandleAppError(c, contextutils.ErrInternalError)
		return
	}

	span.SetAttributes(attribute.Int("user_id", userIDInt))

	// List API keys
	apiKeys, err := h.apiKeyService.ListAPIKeys(ctx, userIDInt)
	if err != nil {
		h.logger.Error(ctx, "Failed to list API keys", err, map[string]interface{}{"user_id": userIDInt})
		HandleAppError(c, err)
		return
	}

	span.SetAttributes(attribute.Int("count", len(apiKeys)))

	// Return list of API keys (without full keys or hashes)
	c.JSON(http.StatusOK, gin.H{
		"api_keys": apiKeys,
		"count":    len(apiKeys),
	})
}

// DeleteAPIKey handles DELETE /v1/api-keys/:id
func (h *AuthAPIKeyHandler) DeleteAPIKey(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "DeleteAPIKey")
	defer observability.FinishSpan(span, nil)

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	userIDInt, ok := userID.(int)
	if !ok {
		HandleAppError(c, contextutils.ErrInternalError)
		return
	}

	// Get key ID from URL parameter
	keyIDStr := c.Param("id")
	keyID, err := strconv.Atoi(keyIDStr)
	if err != nil {
		HandleAppError(c, contextutils.NewAppErrorWithCause(
			contextutils.ErrorCodeInvalidInput,
			contextutils.SeverityWarn,
			"Invalid API key ID",
			"",
			err,
		))
		return
	}

	span.SetAttributes(
		attribute.Int("user_id", userIDInt),
		attribute.Int("key_id", keyID),
	)

	// Delete API key
	err = h.apiKeyService.DeleteAPIKey(ctx, userIDInt, keyID)
	if err != nil {
		h.logger.Error(ctx, "Failed to delete API key", err, map[string]interface{}{
			"user_id": userIDInt,
			"key_id":  keyID,
		})
		HandleAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "API key deleted successfully",
	})
}
