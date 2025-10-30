package handlers

import (
	"net/http"
	"strconv"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// SnippetsHandler handles snippets related HTTP requests
type SnippetsHandler struct {
	snippetsService services.SnippetsServiceInterface
	cfg             *config.Config
	logger          *observability.Logger
}

// NewSnippetsHandler creates a new SnippetsHandler instance
func NewSnippetsHandler(snippetsService services.SnippetsServiceInterface, cfg *config.Config, logger *observability.Logger) *SnippetsHandler {
	return &SnippetsHandler{
		snippetsService: snippetsService,
		cfg:             cfg,
		logger:          logger,
	}
}

// CreateSnippet handles POST /v1/snippets
func (h *SnippetsHandler) CreateSnippet(c *gin.Context) {
	ctx, span := observability.TraceSnippetFunction(c.Request.Context(), "create_snippet")
	defer observability.FinishSpan(span, nil)

	// Get user ID from context (set by auth middleware)
	userID, exists := GetUserIDFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "User ID not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	username, exists := GetUsernameFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "Username not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	span.SetAttributes(attribute.Int64("user.id", int64(userID)))
	span.SetAttributes(attribute.String("user.username", username))

	var req api.CreateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "Invalid create snippet request format", map[string]interface{}{
			"error": err.Error(),
		})
		HandleAppError(c, contextutils.ErrInvalidInput)
		return
	}

	snippet, err := h.snippetsService.CreateSnippet(ctx, int64(userID), req)
	if err != nil {
		h.logger.Error(ctx, "Failed to create snippet", err, map[string]interface{}{
			"user_id": userID,
		})

		HandleAppError(c, err)
		return
	}

	// Convert to API response format
	response := api.Snippet{
		Id:              &snippet.ID,
		UserId:          &snippet.UserID,
		OriginalText:    &snippet.OriginalText,
		TranslatedText:  &snippet.TranslatedText,
		SourceLanguage:  &snippet.SourceLanguage,
		TargetLanguage:  &snippet.TargetLanguage,
		QuestionId:      snippet.QuestionID,
		SectionId:       snippet.SectionID,
		StoryId:         snippet.StoryID,
		Context:         snippet.Context,
		DifficultyLevel: snippet.DifficultyLevel,
		CreatedAt:       &snippet.CreatedAt,
		UpdatedAt:       &snippet.UpdatedAt,
	}

	span.SetAttributes(
		attribute.Int64("snippet.id", snippet.ID),
		attribute.Int64("user.id", int64(userID)),
		attribute.String("snippet.original_text", snippet.OriginalText),
		attribute.String("snippet.translated_text", snippet.TranslatedText),
		attribute.String("snippet.source_language", snippet.SourceLanguage),
		attribute.String("snippet.target_language", snippet.TargetLanguage),
	)
	if snippet.QuestionID != nil {
		span.SetAttributes(attribute.Int64("snippet.question_id", *snippet.QuestionID))
	}
	if snippet.Context != nil {
		span.SetAttributes(attribute.String("snippet.context", *snippet.Context))
	}
	if snippet.DifficultyLevel != nil {
		span.SetAttributes(attribute.String("snippet.difficulty_level", *snippet.DifficultyLevel))
	}

	c.JSON(http.StatusCreated, response)
}

// GetSnippets handles GET /v1/snippets
func (h *SnippetsHandler) GetSnippets(c *gin.Context) {
	ctx, span := observability.TraceSnippetFunction(c.Request.Context(), "get_snippets")
	defer observability.FinishSpan(span, nil)

	// Get user ID from context (set by auth middleware)
	userID, exists := GetUserIDFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "User ID not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	username, exists := GetUsernameFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "Username not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	span.SetAttributes(attribute.Int64("user.id", int64(userID)))
	span.SetAttributes(attribute.String("user.username", username))
	// Parse query parameters
	params := api.GetV1SnippetsParams{}

	if q := c.Query("q"); q != "" {
		params.Q = &q
	}
	if sourceLang := c.Query("source_lang"); sourceLang != "" {
		params.SourceLang = &sourceLang
	}
	if targetLang := c.Query("target_lang"); targetLang != "" {
		params.TargetLang = &targetLang
	}
	if storyIDStr := c.Query("story_id"); storyIDStr != "" {
		if storyID, err := strconv.ParseInt(storyIDStr, 10, 64); err == nil {
			params.StoryId = &storyID
		}
	}
	if level := c.Query("level"); level != "" {
		params.Level = (*api.GetV1SnippetsParamsLevel)(&level)
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			params.Limit = &limit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			params.Offset = &offset
		}
	}
	if params.Limit != nil {
		span.SetAttributes(attribute.Int("params.limit", *params.Limit))
	}
	if params.Offset != nil {
		span.SetAttributes(attribute.Int("params.offset", *params.Offset))
	}
	if q := params.Q; q != nil {
		span.SetAttributes(attribute.String("params.q", *q))
	}
	if sourceLang := params.SourceLang; sourceLang != nil {
		span.SetAttributes(attribute.String("params.source_lang", *sourceLang))
	}
	if targetLang := params.TargetLang; targetLang != nil {
		span.SetAttributes(attribute.String("params.target_lang", *targetLang))
	}
	if storyID := params.StoryId; storyID != nil {
		span.SetAttributes(attribute.Int64("params.story_id", *storyID))
	}
	if level := params.Level; level != nil {
		span.SetAttributes(attribute.String("params.level", string(*level)))
	}
	snippetList, err := h.snippetsService.GetSnippets(ctx, int64(userID), params)
	if err != nil {
		h.logger.Error(ctx, "Failed to get snippets", err, map[string]any{
			"user_id": userID,
		})
		HandleAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, snippetList)
}

// GetSnippetsByQuestion handles GET /v1/snippets/by-question/:question_id
func (h *SnippetsHandler) GetSnippetsByQuestion(c *gin.Context) {
	ctx, span := observability.TraceSnippetFunction(c.Request.Context(), "get_snippets_by_question")
	defer observability.FinishSpan(span, nil)

	// Get user ID from context (set by auth middleware)
	userID, exists := GetUserIDFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "User ID not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	username, exists := GetUsernameFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "Username not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	span.SetAttributes(attribute.Int64("user.id", int64(userID)))
	span.SetAttributes(attribute.String("user.username", username))

	// Parse question_id from path parameter
	questionIDStr := c.Param("question_id")
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		h.logger.Warn(ctx, "Invalid question_id parameter", map[string]any{
			"question_id": questionIDStr,
			"error":       err.Error(),
		})
		HandleAppError(c, contextutils.ErrInvalidInput)
		return
	}

	span.SetAttributes(attribute.Int64("question.id", questionID))

	// Get snippets for this question
	snippets, err := h.snippetsService.GetSnippetsByQuestion(ctx, int64(userID), questionID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get snippets by question", err, map[string]any{
			"user_id":     userID,
			"question_id": questionID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get snippets by question"))
		return
	}

	// Return response with snippets array
	response := gin.H{
		"snippets": snippets,
	}

	c.JSON(http.StatusOK, response)
}

// GetSnippetsBySection handles GET /v1/snippets/by-section/:section_id
func (h *SnippetsHandler) GetSnippetsBySection(c *gin.Context) {
	ctx, span := observability.TraceSnippetFunction(c.Request.Context(), "get_snippets_by_section")
	defer observability.FinishSpan(span, nil)

	// Get user ID from context (set by auth middleware)
	userID, exists := GetUserIDFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "User ID not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	username, exists := GetUsernameFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "Username not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	span.SetAttributes(attribute.Int64("user.id", int64(userID)))
	span.SetAttributes(attribute.String("user.username", username))

	// Parse section_id from path parameter
	sectionIDStr := c.Param("section_id")
	sectionID, err := strconv.ParseInt(sectionIDStr, 10, 64)
	if err != nil {
		h.logger.Warn(ctx, "Invalid section_id parameter", map[string]any{
			"section_id": sectionIDStr,
			"error":      err.Error(),
		})
		HandleAppError(c, contextutils.ErrInvalidInput)
		return
	}

	span.SetAttributes(attribute.Int64("section.id", sectionID))

	// Get snippets for this section
	snippets, err := h.snippetsService.GetSnippetsBySection(ctx, int64(userID), sectionID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get snippets by section", err, map[string]any{
			"user_id":    userID,
			"section_id": sectionID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get snippets by section"))
		return
	}

	// Return response with snippets array
	response := gin.H{
		"snippets": snippets,
	}

	c.JSON(http.StatusOK, response)
}

// GetSnippetsByStory handles GET /v1/snippets/by-story/:story_id
func (h *SnippetsHandler) GetSnippetsByStory(c *gin.Context) {
	ctx, span := observability.TraceSnippetFunction(c.Request.Context(), "get_snippets_by_story")
	defer observability.FinishSpan(span, nil)

	// Get user ID from context (set by auth middleware)
	userID, exists := GetUserIDFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "User ID not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	username, exists := GetUsernameFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "Username not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	span.SetAttributes(attribute.Int64("user.id", int64(userID)))
	span.SetAttributes(attribute.String("user.username", username))

	// Parse story_id from path parameter
	storyIDStr := c.Param("story_id")
	storyID, err := strconv.ParseInt(storyIDStr, 10, 64)
	if err != nil {
		h.logger.Warn(ctx, "Invalid story_id parameter", map[string]any{
			"story_id": storyIDStr,
			"error":    err.Error(),
		})
		HandleAppError(c, contextutils.ErrInvalidInput)
		return
	}

	span.SetAttributes(attribute.Int64("story.id", storyID))

	// Get snippets for this story
	snippets, err := h.snippetsService.GetSnippetsByStory(ctx, int64(userID), storyID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get snippets by story", err, map[string]any{
			"user_id":  userID,
			"story_id": storyID,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to get snippets by story"))
		return
	}

	// Return response with snippets array
	response := gin.H{
		"snippets": snippets,
	}

	c.JSON(http.StatusOK, response)
}

// SearchSnippets handles GET /v1/snippets/search
func (h *SnippetsHandler) SearchSnippets(c *gin.Context) {
	ctx, span := observability.TraceSnippetFunction(c.Request.Context(), "search_snippets")
	defer observability.FinishSpan(span, nil)

	// Get user ID from context (set by auth middleware)
	userID, exists := GetUserIDFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "User ID not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	username, exists := GetUsernameFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "Username not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	span.SetAttributes(attribute.Int64("user.id", int64(userID)))
	span.SetAttributes(attribute.String("user.username", username))

	// Parse query parameters
	query := c.Query("q")
	if query == "" {
		HandleAppError(c, contextutils.ErrInvalidInput)
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	// Optional filters
	sourceLang := c.Query("source_lang")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	span.SetAttributes(
		attribute.String("query", query),
		attribute.Int("limit", limit),
		attribute.Int("offset", offset),
	)
	if sourceLang != "" {
		span.SetAttributes(attribute.String("params.source_lang", sourceLang))
	}

	// Search snippets
	var sourceLangPtr *string
	if sourceLang != "" {
		sourceLangPtr = &sourceLang
	}
	snippets, total, err := h.snippetsService.SearchSnippets(ctx, int64(userID), query, limit, offset, sourceLangPtr)
	if err != nil {
		h.logger.Error(ctx, "Failed to search snippets", err, map[string]any{
			"user_id": userID,
			"query":   query,
			"limit":   limit,
			"offset":  offset,
		})
		HandleAppError(c, contextutils.WrapError(err, "failed to search snippets"))
		return
	}

	// Add metadata to response
	response := gin.H{
		"snippets": snippets,
		"query":    query,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	}

	c.JSON(http.StatusOK, response)
}

// GetSnippet handles GET /v1/snippets/{id}
func (h *SnippetsHandler) GetSnippet(c *gin.Context) {
	ctx, span := observability.TraceSnippetFunction(c.Request.Context(), "get_snippet")
	defer observability.FinishSpan(span, nil)

	// Get user ID from context (set by auth middleware)
	userID, exists := GetUserIDFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "User ID not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	username, exists := GetUsernameFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "Username not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	span.SetAttributes(attribute.String("user.username", username))
	span.SetAttributes(attribute.Int64("user.id", int64(userID)))

	// Parse snippet ID from URL parameter
	snippetIDStr := c.Param("id")
	snippetID, err := strconv.ParseInt(snippetIDStr, 10, 64)
	if err != nil {
		h.logger.Warn(ctx, "Invalid snippet ID format", map[string]interface{}{
			"snippet_id": snippetIDStr,
			"error":      err.Error(),
		})
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	snippet, err := h.snippetsService.GetSnippet(ctx, int64(userID), snippetID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get snippet", err, map[string]interface{}{
			"user_id":    userID,
			"snippet_id": snippetID,
		})

		HandleAppError(c, err)
		return
	}

	// Convert to API response format
	response := api.Snippet{
		Id:              &snippet.ID,
		UserId:          &snippet.UserID,
		OriginalText:    &snippet.OriginalText,
		TranslatedText:  &snippet.TranslatedText,
		SourceLanguage:  &snippet.SourceLanguage,
		TargetLanguage:  &snippet.TargetLanguage,
		QuestionId:      snippet.QuestionID,
		Context:         snippet.Context,
		DifficultyLevel: snippet.DifficultyLevel,
		CreatedAt:       &snippet.CreatedAt,
		UpdatedAt:       &snippet.UpdatedAt,
	}

	span.SetAttributes(
		attribute.Int64("snippet.id", snippet.ID),
		attribute.Int64("user.id", int64(userID)),
		attribute.String("user.username", username),
		attribute.String("snippet.original_text", snippet.OriginalText),
		attribute.String("snippet.translated_text", snippet.TranslatedText),
		attribute.String("snippet.source_language", snippet.SourceLanguage),
		attribute.String("snippet.target_language", snippet.TargetLanguage),
	)
	if snippet.QuestionID != nil {
		span.SetAttributes(attribute.Int64("snippet.question_id", *snippet.QuestionID))
	}
	if snippet.Context != nil {
		span.SetAttributes(attribute.String("snippet.context", *snippet.Context))
	}
	if snippet.DifficultyLevel != nil {
		span.SetAttributes(attribute.String("snippet.difficulty_level", *snippet.DifficultyLevel))
	}

	c.JSON(http.StatusOK, response)
}

// UpdateSnippet handles PUT /v1/snippets/{id}
func (h *SnippetsHandler) UpdateSnippet(c *gin.Context) {
	ctx, span := observability.TraceSnippetFunction(c.Request.Context(), "update_snippet")
	defer observability.FinishSpan(span, nil)

	// Get user ID from context (set by auth middleware)
	userID, exists := GetUserIDFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "User ID not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	username, exists := GetUsernameFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "Username not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	span.SetAttributes(attribute.String("user.username", username))
	span.SetAttributes(attribute.Int64("user.id", int64(userID)))

	// Parse snippet ID from URL parameter
	snippetIDStr := c.Param("id")
	snippetID, err := strconv.ParseInt(snippetIDStr, 10, 64)
	if err != nil {
		h.logger.Warn(ctx, "Invalid snippet ID format", map[string]interface{}{
			"snippet_id": snippetIDStr,
			"error":      err.Error(),
		})
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	var req api.UpdateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "Invalid update snippet request format", map[string]interface{}{
			"error": err.Error(),
		})
		HandleAppError(c, contextutils.ErrInvalidInput)
		return
	}

	snippet, err := h.snippetsService.UpdateSnippet(ctx, int64(userID), snippetID, req)
	if err != nil {
		h.logger.Error(ctx, "Failed to update snippet", err, map[string]interface{}{
			"user_id":    userID,
			"snippet_id": snippetID,
		})

		HandleAppError(c, err)
		return
	}

	// Convert to API response format
	response := api.Snippet{
		Id:              &snippet.ID,
		UserId:          &snippet.UserID,
		OriginalText:    &snippet.OriginalText,
		TranslatedText:  &snippet.TranslatedText,
		SourceLanguage:  &snippet.SourceLanguage,
		TargetLanguage:  &snippet.TargetLanguage,
		QuestionId:      snippet.QuestionID,
		Context:         snippet.Context,
		DifficultyLevel: snippet.DifficultyLevel,
		CreatedAt:       &snippet.CreatedAt,
		UpdatedAt:       &snippet.UpdatedAt,
	}

	span.SetAttributes(
		attribute.Int64("snippet.id", snippet.ID),
		attribute.Int64("user.id", int64(userID)),
		attribute.String("user.username", username),
		attribute.String("snippet.original_text", snippet.OriginalText),
		attribute.String("snippet.translated_text", snippet.TranslatedText),
		attribute.String("snippet.source_language", snippet.SourceLanguage),
		attribute.String("snippet.target_language", snippet.TargetLanguage),
	)
	if snippet.QuestionID != nil {
		span.SetAttributes(attribute.Int64("snippet.question_id", *snippet.QuestionID))
	}
	if snippet.Context != nil {
		span.SetAttributes(attribute.String("snippet.context", *snippet.Context))
	}
	if snippet.DifficultyLevel != nil {
		span.SetAttributes(attribute.String("snippet.difficulty_level", *snippet.DifficultyLevel))
	}

	c.JSON(http.StatusOK, response)
}

// DeleteSnippet handles DELETE /v1/snippets/{id}
func (h *SnippetsHandler) DeleteSnippet(c *gin.Context) {
	ctx, span := observability.TraceSnippetFunction(c.Request.Context(), "delete_snippet")
	defer observability.FinishSpan(span, nil)

	// Get user ID from context (set by auth middleware)
	userID, exists := GetUserIDFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "User ID not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	username, exists := GetUsernameFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "Username not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	span.SetAttributes(attribute.String("user.username", username))
	span.SetAttributes(attribute.Int64("user.id", int64(userID)))

	// Parse snippet ID from URL parameter
	snippetIDStr := c.Param("id")
	snippetID, err := strconv.ParseInt(snippetIDStr, 10, 64)
	if err != nil {
		h.logger.Warn(ctx, "Invalid snippet ID format", map[string]interface{}{
			"snippet_id": snippetIDStr,
			"error":      err.Error(),
		})
		HandleAppError(c, contextutils.ErrInvalidFormat)
		return
	}

	err = h.snippetsService.DeleteSnippet(ctx, int64(userID), snippetID)
	if err != nil {
		h.logger.Error(ctx, "Failed to delete snippet", err, map[string]interface{}{
			"user_id":    userID,
			"snippet_id": snippetID,
		})

		HandleAppError(c, err)
		return
	}

	span.SetAttributes(
		attribute.Int64("snippet.id", snippetID),
		attribute.Int64("user.id", int64(userID)),
		attribute.String("user.username", username),
	)

	c.Status(http.StatusNoContent)
}

// DeleteAllSnippets handles DELETE /v1/snippets
func (h *SnippetsHandler) DeleteAllSnippets(c *gin.Context) {
	ctx, span := observability.TraceSnippetFunction(c.Request.Context(), "delete_all_snippets")
	defer observability.FinishSpan(span, nil)

	// Get user ID from context (set by auth middleware)
	userID, exists := GetUserIDFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "User ID not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	username, exists := GetUsernameFromSession(c)
	if !exists {
		h.logger.Warn(ctx, "Username not found in context")
		HandleAppError(c, contextutils.ErrUnauthorized)
		return
	}
	span.SetAttributes(attribute.String("user.username", username))
	span.SetAttributes(attribute.Int64("user.id", int64(userID)))

	err := h.snippetsService.DeleteAllSnippets(ctx, int64(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to delete all snippets", err, map[string]interface{}{
			"user_id": userID,
		})

		HandleAppError(c, contextutils.ErrInternalError)
		return
	}

	c.Status(http.StatusNoContent)
}
