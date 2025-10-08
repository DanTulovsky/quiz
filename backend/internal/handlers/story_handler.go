package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
)

// StoryHandler handles story-related HTTP requests
type StoryHandler struct {
	storyService services.StoryServiceInterface
	userService  services.UserServiceInterface
	aiService    services.AIServiceInterface
	cfg          *config.Config
	logger       *observability.Logger
}

// NewStoryHandler creates a new StoryHandler
func NewStoryHandler(
	storyService services.StoryServiceInterface,
	userService services.UserServiceInterface,
	aiService services.AIServiceInterface,
	cfg *config.Config,
	logger *observability.Logger,
) *StoryHandler {
	return &StoryHandler{
		storyService: storyService,
		userService:  userService,
		aiService:    aiService,
		cfg:          cfg,
		logger:       logger,
	}
}

// CreateStory handles POST /v1/story
func (h *StoryHandler) CreateStory(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "create_story")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		StandardizeHTTPError(c, http.StatusUnauthorized, "Unauthorized", "User session not found or invalid")
		return
	}

	// userID is already int from GetUserIDFromSession

	var req models.CreateStoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Failed to bind story creation request", err, nil)
		StandardizeHTTPError(c, http.StatusBadRequest, "Invalid request format", err.Error())
		return
	}

	// Get user's language preference
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user", err, map[string]interface{}{
			"user_id": userID,
		})
		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to get user information", err.Error())
		return
	}

	// Get the user's preferred language (handle sql.NullString)
	language := "en" // default
	if user.PreferredLanguage.Valid {
		language = user.PreferredLanguage.String
	}

	story, err := h.storyService.CreateStory(ctx, uint(userID), language, &req)
	if err != nil {
		h.logger.Error(ctx, "Failed to create story", err, map[string]interface{}{
			"user_id": userID,
			"title":   req.Title,
		})

		// Handle specific error cases
		if strings.Contains(err.Error(), "maximum archived stories limit reached") {
			StandardizeHTTPError(c, http.StatusForbidden, "Maximum archived stories limit reached", err.Error())
			return
		}

		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to create story", err.Error())
		return
	}

	span.SetAttributes(
		attribute.String("story.title", story.Title),
		attribute.Int("story.id", int(story.ID)),
		attribute.String("user.language", language),
	)

	// Convert to API types to ensure proper serialization
	apiStory := convertStoryToAPI(story)
	c.JSON(http.StatusCreated, apiStory)
}

// GetUserStories handles GET /v1/story
func (h *StoryHandler) GetUserStories(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_user_stories")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		StandardizeHTTPError(c, http.StatusUnauthorized, "Unauthorized", "User session not found or invalid")
		return
	}

	includeArchivedStr := c.Query("include_archived")
	includeArchived := includeArchivedStr == "true"

	stories, err := h.storyService.GetUserStories(ctx, uint(userID), includeArchived)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user stories", err, map[string]interface{}{
			"user_id":          uint(userID),
			"include_archived": includeArchived,
		})
		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to get stories", err.Error())
		return
	}

	c.JSON(http.StatusOK, stories)
}

// GetCurrentStory handles GET /v1/story/current
func (h *StoryHandler) GetCurrentStory(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_current_story")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		StandardizeHTTPError(c, http.StatusUnauthorized, "Unauthorized", "User session not found or invalid")
		return
	}

	story, err := h.storyService.GetCurrentStory(ctx, uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to get current story", err, map[string]interface{}{
			"user_id": uint(userID),
		})
		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to get current story", err.Error())
		return
	}

	if story == nil {
		StandardizeHTTPError(c, http.StatusNotFound, "No current story found", "User has no active story")
		return
	}

	// If story exists but has no sections, it's generating the first section
	if len(story.Sections) == 0 {
		c.JSON(http.StatusAccepted, api.GeneratingResponse{
			Status:  stringPtr("generating"),
			Message: stringPtr("Story created successfully. The first section is being generated. Please check back shortly."),
		})
		return
	}

	// If story exists and has sections, check if a section is currently being generated today
	today := time.Now().Truncate(24 * time.Hour)
	sectionsToday := 0
	for _, section := range story.Sections {
		if section.GenerationDate.Truncate(24 * time.Hour).Equal(today) {
			sectionsToday++
		}
	}

	if sectionsToday == 0 {
		c.JSON(http.StatusAccepted, api.GeneratingResponse{
			Status:  stringPtr("generating"),
			Message: stringPtr("The next section is being generated. Please check back shortly."),
		})
		return
	}

	// Convert to API types to ensure proper serialization
	apiStory := convertStoryWithSectionsToAPI(story)
	c.JSON(http.StatusOK, apiStory)
}

// GetStory handles GET /v1/story/:id
func (h *StoryHandler) GetStory(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_story")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		StandardizeHTTPError(c, http.StatusUnauthorized, "Unauthorized", "User session not found or invalid")
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		StandardizeHTTPError(c, http.StatusBadRequest, "Invalid story ID", "Story ID must be a valid number")
		return
	}

	story, err := h.storyService.GetStory(ctx, uint(storyID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to get story", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			StandardizeHTTPError(c, http.StatusNotFound, "Story not found", "The requested story does not exist or you don't have access to it")
			return
		}

		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to get story", err.Error())
		return
	}

	// Convert to API types to ensure proper serialization
	apiStory := convertStoryWithSectionsToAPI(story)
	c.JSON(http.StatusOK, apiStory)
}

// GetSection handles GET /v1/story/section/:id
func (h *StoryHandler) GetSection(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "get_section")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		StandardizeHTTPError(c, http.StatusUnauthorized, "Unauthorized", "User session not found or invalid")
		return
	}

	sectionIDStr := c.Param("id")
	sectionID, err := strconv.ParseUint(sectionIDStr, 10, 32)
	if err != nil {
		StandardizeHTTPError(c, http.StatusBadRequest, "Invalid section ID", "Section ID must be a valid number")
		return
	}

	section, err := h.storyService.GetSection(ctx, uint(sectionID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to get section", err, map[string]interface{}{
			"section_id": sectionID,
			"user_id":    uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			StandardizeHTTPError(c, http.StatusNotFound, "Section not found", "The requested section does not exist or you don't have access to it")
			return
		}

		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to get section", err.Error())
		return
	}

	// Convert to API types to ensure proper serialization
	apiSection := convertStorySectionWithQuestionsToAPI(section)
	c.JSON(http.StatusOK, apiSection)
}

// GenerateNextSection handles POST /v1/story/:id/generate
func (h *StoryHandler) GenerateNextSection(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "generate_next_section")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		StandardizeHTTPError(c, http.StatusUnauthorized, "Unauthorized", "User session not found or invalid")
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		StandardizeHTTPError(c, http.StatusBadRequest, "Invalid story ID", "Story ID must be a valid number")
		return
	}

	// Verify story ownership (this is done inside GenerateStorySection, but we need to check beforehand for early return)

	// Check if generation is allowed today
	canGenerate, err := h.storyService.CanGenerateSection(ctx, uint(storyID))
	if err != nil {
		h.logger.Error(ctx, "Failed to check generation eligibility", err, map[string]interface{}{
			"story_id": storyID,
		})
		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to check generation eligibility", err.Error())
		return
	}

	if !canGenerate {
		// Provide more specific error messages
		story, err := h.storyService.GetStory(ctx, uint(storyID), uint(userID))
		if err != nil {
			StandardizeHTTPError(c, http.StatusConflict, "Cannot generate section", "Story is not active or you have reached the daily generation limit")
			return
		}

		// Check if story is not active
		if story.Status != models.StoryStatusActive || !story.IsCurrent {
			StandardizeHTTPError(c, http.StatusConflict, "Cannot generate section", "Story is not active")
			return
		}

		// Check if daily generation limit is reached (no extra generations available)
		if story.ExtraGenerationsToday >= h.cfg.Story.MaxExtraGenerationsPerDay {
			StandardizeHTTPError(c, http.StatusConflict, "Daily generation limit reached", "You have already generated a section today for this story. Please try again tomorrow.")
			return
		}

		StandardizeHTTPError(c, http.StatusConflict, "Cannot generate section", "Please try again tomorrow")
		return
	}

	// Get user for AI config
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user for generation", err, map[string]interface{}{
			"user_id": uint(userID),
		})
		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to get user information", err.Error())
		return
	}

	// Get user's AI configuration
	userAIConfig := h.convertToServicesAIConfig(ctx, user)

	// Generate the story section using the shared service method (user generation)
	sectionWithQuestions, err := h.storyService.GenerateStorySection(ctx, uint(storyID), uint(userID), h.aiService, userAIConfig)
	if err != nil {
		// Check if this is a generation limit reached error (normal business case)
		if errors.Is(err, contextutils.ErrGenerationLimitReached) {
			h.logger.Info(ctx, "User reached daily generation limit", map[string]interface{}{
				"story_id": storyID,
				"user_id":  uint(userID),
			})
			StandardizeHTTPError(c, http.StatusConflict, "Daily generation limit reached", "You have already generated a section today for this story. Please try again tomorrow.")
			return
		}

		h.logger.Error(ctx, "Failed to generate story section", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})

		// Check if this is a constraint violation (duplicate generation today)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			StandardizeHTTPError(c, http.StatusConflict, "Cannot generate section", "You have already generated a section today for this story. Please try again tomorrow.")
			return
		}

		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to generate story section", err.Error())
		return
	}

	// Return success response with the generated section
	apiSection := convertStorySectionWithQuestionsToAPI(sectionWithQuestions)
	c.JSON(http.StatusCreated, apiSection)
}

// ArchiveStory handles POST /v1/story/:id/archive
func (h *StoryHandler) ArchiveStory(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "archive_story")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		StandardizeHTTPError(c, http.StatusUnauthorized, "Unauthorized", "User session not found or invalid")
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		StandardizeHTTPError(c, http.StatusBadRequest, "Invalid story ID", "Story ID must be a valid number")
		return
	}

	err = h.storyService.ArchiveStory(ctx, uint(storyID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to archive story", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			StandardizeHTTPError(c, http.StatusNotFound, "Story not found", "The requested story does not exist or you don't have access to it")
			return
		}

		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to archive story", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "story archived successfully"})
}

// CompleteStory handles POST /v1/story/:id/complete
func (h *StoryHandler) CompleteStory(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "complete_story")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		StandardizeHTTPError(c, http.StatusUnauthorized, "Unauthorized", "User session not found or invalid")
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		StandardizeHTTPError(c, http.StatusBadRequest, "Invalid story ID", "Story ID must be a valid number")
		return
	}

	err = h.storyService.CompleteStory(ctx, uint(storyID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to complete story", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			StandardizeHTTPError(c, http.StatusNotFound, "Story not found", "The requested story does not exist or you don't have access to it")
			return
		}

		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to complete story", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "story completed successfully"})
}

// SetCurrentStory handles POST /v1/story/:id/set-current
func (h *StoryHandler) SetCurrentStory(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "set_current_story")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		StandardizeHTTPError(c, http.StatusUnauthorized, "Unauthorized", "User session not found or invalid")
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		StandardizeHTTPError(c, http.StatusBadRequest, "Invalid story ID", "Story ID must be a valid number")
		return
	}

	err = h.storyService.SetCurrentStory(ctx, uint(storyID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to set current story", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			StandardizeHTTPError(c, http.StatusNotFound, "Story not found", "The requested story does not exist or you don't have access to it")
			return
		}

		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to set current story", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "story set as current successfully"})
}

// DeleteStory handles DELETE /v1/story/:id
func (h *StoryHandler) DeleteStory(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "delete_story")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		StandardizeHTTPError(c, http.StatusUnauthorized, "Unauthorized", "User session not found or invalid")
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		StandardizeHTTPError(c, http.StatusBadRequest, "Invalid story ID", "Story ID must be a valid number")
		return
	}

	err = h.storyService.DeleteStory(ctx, uint(storyID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to delete story", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			StandardizeHTTPError(c, http.StatusNotFound, "Story not found", "The requested story does not exist or you don't have access to it")
			return
		}

		if strings.Contains(err.Error(), "cannot delete current story") {
			StandardizeHTTPError(c, http.StatusConflict, "Cannot delete current story", "You cannot delete a story that is currently set as your active story")
			return
		}

		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to delete story", err.Error())
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ExportStory handles GET /v1/story/:id/export
func (h *StoryHandler) ExportStory(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "export_story")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		StandardizeHTTPError(c, http.StatusUnauthorized, "Unauthorized", "User session not found or invalid")
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		StandardizeHTTPError(c, http.StatusBadRequest, "Invalid story ID", "Story ID must be a valid number")
		return
	}

	// Get the story with all sections
	story, err := h.storyService.GetStory(ctx, uint(storyID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to get story for export", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			StandardizeHTTPError(c, http.StatusNotFound, "Story not found", "The requested story does not exist or you don't have access to it")
			return
		}

		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to get story", err.Error())
		return
	}

	// Create PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	// Add title
	pdf.Cell(40, 10, story.Title)
	pdf.Ln(12)

	// Add story metadata if present
	pdf.SetFont("Arial", "", 10)
	if story.Subject != nil && *story.Subject != "" {
		pdf.Cell(40, 8, fmt.Sprintf("Subject: %s", *story.Subject))
		pdf.Ln(6)
	}
	if story.AuthorStyle != nil && *story.AuthorStyle != "" {
		pdf.Cell(40, 8, fmt.Sprintf("Style: %s", *story.AuthorStyle))
		pdf.Ln(6)
	}
	if story.Genre != nil && *story.Genre != "" {
		pdf.Cell(40, 8, fmt.Sprintf("Genre: %s", *story.Genre))
		pdf.Ln(6)
	}
	pdf.Ln(5)

	// Add sections
	pdf.SetFont("Arial", "", 11)
	for _, section := range story.Sections {
		// Section header
		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(40, 8, fmt.Sprintf("Section %d", section.SectionNumber))
		pdf.Ln(8)

		// Section content
		pdf.SetFont("Arial", "", 11)

		// Split content into paragraphs (double line breaks)
		paragraphs := strings.Split(section.Content, "\n\n")
		for _, paragraph := range paragraphs {
			if paragraph != "" {
				// MultiCell for text wrapping
				pdf.MultiCell(0, 6, paragraph, "", "L", false)
				pdf.Ln(3)
			}
		}
		pdf.Ln(5)
	}

	// Set headers for PDF download
	filename := fmt.Sprintf("story_%s.pdf", strings.ReplaceAll(strings.ToLower(story.Title), " ", "_"))
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Output PDF to response
	var buf strings.Builder
	err = pdf.Output(&buf)
	if err != nil {
		h.logger.Error(ctx, "Failed to generate PDF", err, map[string]interface{}{
			"story_id": storyID,
		})
		StandardizeHTTPError(c, http.StatusInternalServerError, "Failed to generate PDF", err.Error())
		return
	}

	c.Data(http.StatusOK, "application/pdf", []byte(buf.String()))
}

// convertToServicesAIConfig creates AI config for the user in services format
func (h *StoryHandler) convertToServicesAIConfig(ctx context.Context, user *models.User) *models.UserAIConfig {
	// Handle sql.NullString fields
	aiProvider := ""
	if user.AIProvider.Valid {
		aiProvider = user.AIProvider.String
	}

	aiModel := ""
	if user.AIModel.Valid {
		aiModel = user.AIModel.String
	}

	apiKey := ""
	if aiProvider != "" {
		savedKey, err := h.userService.GetUserAPIKey(ctx, user.ID, aiProvider)
		if err == nil && savedKey != "" {
			apiKey = savedKey
		}
	}

	return &models.UserAIConfig{
		Provider: aiProvider,
		Model:    aiModel,
		APIKey:   apiKey,
		Username: user.Username,
	}
}
