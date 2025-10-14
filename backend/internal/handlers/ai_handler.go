package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// AIConversationHandler handles AI conversation-related HTTP requests
type AIConversationHandler struct {
	conversationService services.ConversationServiceInterface
	cfg                 *config.Config
	logger              *observability.Logger
}

// NewAIConversationHandler creates a new AIConversationHandler
func NewAIConversationHandler(
	conversationService services.ConversationServiceInterface,
	cfg *config.Config,
	logger *observability.Logger,
) *AIConversationHandler {
	return &AIConversationHandler{
		conversationService: conversationService,
		cfg:                 cfg,
		logger:              logger,
	}
}

// GetConversations handles GET /v1/ai/conversations
func (h *AIConversationHandler) GetConversations(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_ai_conversations")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.Int("limit", limit),
		attribute.Int("offset", offset),
	)

	// Get conversations for the user
	conversations, total, err := h.conversationService.GetUserConversations(ctx, uint(userID), limit, offset)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user conversations", err, map[string]interface{}{
			"user_id": userID,
			"limit":   limit,
			"offset":  offset,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get conversations"))
		return
	}

	// Add total count to response
	response := gin.H{
		"conversations": conversations,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	}

	c.JSON(http.StatusOK, response)
}

// CreateConversation handles POST /v1/ai/conversations
func (h *AIConversationHandler) CreateConversation(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "create_ai_conversation")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse request body
	var req api.CreateConversationRequest
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

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("conversation_title", req.Title),
	)

	// Create conversation
	conversation, err := h.conversationService.CreateConversation(ctx, uint(userID), &req)
	if err != nil {
		h.logger.Error(ctx, "Failed to create conversation", err, map[string]interface{}{
			"user_id": userID,
			"title":   req.Title,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to create conversation"))
		return
	}

	c.JSON(http.StatusCreated, conversation)
}

// GetConversation handles GET /v1/ai/conversations/{id}
func (h *AIConversationHandler) GetConversation(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_ai_conversation")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse conversation ID parameter
	conversationID := c.Param("id")
	if conversationID == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(conversationID); err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("conversation_id", conversationID),
	)

	// Get conversation with messages
	conversation, err := h.conversationService.GetConversation(ctx, conversationID, uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to get conversation", err, map[string]interface{}{
			"user_id":         userID,
			"conversation_id": conversationID,
		})

		// Check if it's a conversation not found error
		if strings.Contains(err.Error(), "conversation not found") {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to get conversation"))
		return
	}

	c.JSON(http.StatusOK, conversation)
}

// UpdateConversation handles PUT /v1/ai/conversations/{id}
func (h *AIConversationHandler) UpdateConversation(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "update_ai_conversation")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse conversation ID parameter
	conversationID := c.Param("id")
	if conversationID == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(conversationID); err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Parse request body
	var req api.UpdateConversationRequest
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

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("conversation_id", conversationID),
		attribute.String("new_title", req.Title),
	)

	// Update conversation
	conversation, err := h.conversationService.UpdateConversation(ctx, conversationID, uint(userID), &req)
	if err != nil {
		h.logger.Error(ctx, "Failed to update conversation", err, map[string]interface{}{
			"user_id":         userID,
			"conversation_id": conversationID,
			"new_title":       req.Title,
		})

		// Check if it's a conversation not found error
		if strings.Contains(err.Error(), "conversation not found") {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to update conversation"))
		return
	}

	c.JSON(http.StatusOK, conversation)
}

// DeleteConversation handles DELETE /v1/ai/conversations/{id}
func (h *AIConversationHandler) DeleteConversation(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "delete_ai_conversation")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse conversation ID parameter
	conversationID := c.Param("id")
	if conversationID == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(conversationID); err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("conversation_id", conversationID),
	)

	// Delete conversation and all its messages
	err := h.conversationService.DeleteConversation(ctx, conversationID, uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to delete conversation", err, map[string]interface{}{
			"user_id":         userID,
			"conversation_id": conversationID,
		})

		// Check if it's a conversation not found error
		if strings.Contains(err.Error(), "conversation not found") {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to delete conversation"))
		return
	}

	c.Status(http.StatusNoContent)
}

// AddMessage handles POST /v1/ai/conversations/{conversationId}/messages
func (h *AIConversationHandler) AddMessage(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "add_ai_message")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse conversation ID parameter
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		HandleAppError(c, contextutils.ErrMissingRequired)
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(conversationID); err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Parse request body
	var req api.CreateMessageRequest
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

	// Calculate content length for observability
	contentLength := 0
	if req.Content.Text != nil {
		contentLength = len(*req.Content.Text)
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("conversation_id", conversationID),
		attribute.String("message_role", string(req.Role)),
		attribute.Int("message_content_length", contentLength),
	)

	// Add message to conversation
	createdMessage, err := h.conversationService.AddMessage(ctx, conversationID, uint(userID), &req)
	if err != nil {
		h.logger.Error(ctx, "Failed to add message to conversation", err, map[string]interface{}{
			"user_id":         userID,
			"conversation_id": conversationID,
			"message_role":    req.Role,
		})

		// Check if it's a conversation not found error
		if strings.Contains(err.Error(), "conversation not found") {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to add message"))
		return
	}

	c.JSON(http.StatusCreated, createdMessage)
}

// SearchConversations handles GET /v1/ai/search
func (h *AIConversationHandler) SearchConversations(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "search_ai_conversations")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse query parameters
	query := c.Query("q")
	if query == "" {
		HandleAppError(c, contextutils.ErrInvalidInput)
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("search_query", query),
		attribute.Int("limit", limit),
		attribute.Int("offset", offset),
	)

	// Search conversations
	conversations, total, err := h.conversationService.SearchConversations(ctx, uint(userID), query, limit, offset)
	if err != nil {
		h.logger.Error(ctx, "Failed to search conversations", err, map[string]interface{}{
			"user_id": userID,
			"query":   query,
			"limit":   limit,
			"offset":  offset,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to search conversations"))
		return
	}

	// Add total count to response
	response := gin.H{
		"conversations": conversations,
		"query":         query,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	}

	c.JSON(http.StatusOK, response)
}

// ToggleMessageBookmark handles PUT /v1/ai/conversations/bookmark
func (h *AIConversationHandler) ToggleMessageBookmark(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "toggle_message_bookmark")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}

	// Parse request body
	var req struct {
		ConversationID string `json:"conversation_id" binding:"required"`
		MessageID      string `json:"message_id" binding:"required"`
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

	// Validate UUID formats
	if _, err := uuid.Parse(req.ConversationID); err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}
	if _, err := uuid.Parse(req.MessageID); err != nil {
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	// Add span attributes for observability
	span.SetAttributes(
		observability.AttributeUserID(userID),
		attribute.String("conversation_id", req.ConversationID),
		attribute.String("message_id", req.MessageID),
	)

	// Toggle message bookmark
	newBookmarkedStatus, err := h.conversationService.ToggleMessageBookmark(ctx, req.ConversationID, req.MessageID, uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to toggle message bookmark", err, map[string]interface{}{
			"user_id":         userID,
			"conversation_id": req.ConversationID,
			"message_id":      req.MessageID,
		})

		// Check if it's a conversation or message not found error
		if strings.Contains(err.Error(), "not found") {
			HandleAppError(c, contextutils.ErrRecordNotFound)
			return
		}

		HandleAppError(c, contextutils.WrapError(err, "failed to toggle message bookmark"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bookmarked": newBookmarkedStatus,
	})
}
