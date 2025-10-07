package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

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
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: stringPtr("unauthorized")})
		return
	}

	// userID is already int from GetUserIDFromSession

	var req models.CreateStoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Failed to bind story creation request", err, nil)
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: stringPtr("invalid request format")})
		return
	}

	// Get user's language preference
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user", err, map[string]interface{}{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to get user information")})
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
			c.JSON(http.StatusForbidden, api.ErrorResponse{Error: stringPtr(err.Error())})
			return
		}

		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to create story")})
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
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: stringPtr("unauthorized")})
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
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to get stories")})
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
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: stringPtr("unauthorized")})
		return
	}

	story, err := h.storyService.GetCurrentStory(ctx, uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to get current story", err, map[string]interface{}{
			"user_id": uint(userID),
		})
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to get current story")})
		return
	}

	if story == nil {
		c.JSON(http.StatusNotFound, api.ErrorResponse{Error: stringPtr("no current story found")})
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
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: stringPtr("unauthorized")})
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: stringPtr("invalid story ID")})
		return
	}

	story, err := h.storyService.GetStory(ctx, uint(storyID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to get story", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: stringPtr("story not found")})
			return
		}

		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to get story")})
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
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: stringPtr("unauthorized")})
		return
	}

	sectionIDStr := c.Param("id")
	sectionID, err := strconv.ParseUint(sectionIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: stringPtr("invalid section ID")})
		return
	}

	section, err := h.storyService.GetSection(ctx, uint(sectionID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to get section", err, map[string]interface{}{
			"section_id": sectionID,
			"user_id":    uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: stringPtr("section not found")})
			return
		}

		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to get section")})
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
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: stringPtr("unauthorized")})
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: stringPtr("invalid story ID")})
		return
	}

	// Get the story to verify ownership and get details
	story, err := h.storyService.GetStory(ctx, uint(storyID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to get story for generation", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: stringPtr("story not found")})
			return
		}

		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to get story")})
		return
	}

	// Check if generation is allowed today
	canGenerate, err := h.storyService.CanGenerateSection(ctx, uint(storyID))
	if err != nil {
		h.logger.Error(ctx, "Failed to check generation eligibility", err, map[string]interface{}{
			"story_id": storyID,
		})
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to check generation eligibility")})
		return
	}

	if !canGenerate {
		// Provide more specific error messages
		story, err := h.storyService.GetStory(ctx, uint(storyID), uint(userID))
		if err != nil {
			c.JSON(http.StatusConflict, api.ErrorResponse{Error: stringPtr("cannot generate section: story is not active or you have reached the daily generation limit")})
			return
		}

		// Check if story is not active
		if story.Status != models.StoryStatusActive || !story.IsCurrent {
			c.JSON(http.StatusConflict, api.ErrorResponse{Error: stringPtr("cannot generate section: story is not active")})
			return
		}

		// Check if daily generation limit is reached (no extra generations available)
		if story.ExtraGenerationsToday < 1 {
			c.JSON(http.StatusConflict, api.ErrorResponse{Error: stringPtr("daily generation limit reached: you have already generated a section today for this story. Please try again tomorrow.")})
			return
		}

		c.JSON(http.StatusConflict, api.ErrorResponse{Error: stringPtr("cannot generate section: please try again tomorrow")})
		return
	}

	// Get user for AI config
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get user for generation", err, map[string]interface{}{
			"user_id": uint(userID),
		})
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to get user information")})
		return
	}

	// Get all previous sections for context
	previousSections, err := h.storyService.GetAllSectionsText(ctx, uint(storyID))
	if err != nil {
		h.logger.Error(ctx, "Failed to get previous sections", err, map[string]interface{}{
			"story_id": storyID,
		})
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to get story context")})
		return
	}

	// Get the user's preferred language (handle sql.NullString)
	userLanguage := "en" // default
	if user.PreferredLanguage.Valid {
		userLanguage = user.PreferredLanguage.String
	}

	// Get the user's current language level (handle sql.NullString)
	userLevel := "B1" // default
	if user.CurrentLevel.Valid {
		userLevel = user.CurrentLevel.String
	}

	// Determine target length
	targetWords := h.storyService.GetSectionLengthTarget(userLanguage, story.SectionLengthOverride)

	// Build generation request
	genReq := &models.StoryGenerationRequest{
		UserID:             uint(userID),
		StoryID:            uint(storyID),
		Language:           story.Language,
		Level:              userLevel,
		Title:              story.Title,
		Subject:            story.Subject,
		AuthorStyle:        story.AuthorStyle,
		TimePeriod:         story.TimePeriod,
		Genre:              story.Genre,
		Tone:               story.Tone,
		CharacterNames:     story.CharacterNames,
		CustomInstructions: story.CustomInstructions,
		SectionLength:      models.SectionLengthMedium,
		PreviousSections:   previousSections,
		IsFirstSection:     len(story.Sections) == 0,
		TargetWords:        targetWords,
		TargetSentences:    targetWords / 15,
	}

	// Generate the section
	sectionContent, err := h.aiService.GenerateStorySection(ctx, h.convertToServicesAIConfig(ctx, user), genReq)
	if err != nil {
		h.logger.Error(ctx, "Failed to generate story section", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to generate story section")})
		return
	}

	// Count words in generated content
	wordCount := len(strings.Fields(sectionContent))

	// Create the section
	section, err := h.storyService.CreateSection(ctx, uint(storyID), sectionContent, userLevel, wordCount)
	if err != nil {
		// Check if this is a constraint violation (duplicate generation today)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			h.logger.Warn(ctx, "Attempted to generate duplicate story section today", map[string]interface{}{
				"story_id":   storyID,
				"word_count": wordCount,
			})
			c.JSON(http.StatusConflict, api.ErrorResponse{Error: stringPtr("you have already generated a story section today. Please try again tomorrow.")})
			return
		}

		h.logger.Error(ctx, "Failed to create story section", err, map[string]interface{}{
			"story_id":   storyID,
			"word_count": wordCount,
		})
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to save story section")})
		return
	}

	// Generate questions for the section
	questionsReq := &models.StoryQuestionsRequest{
		UserID:        uint(userID),
		SectionID:     section.ID,
		Language:      story.Language,
		Level:         userLevel,
		SectionText:   sectionContent,
		QuestionCount: h.cfg.Story.QuestionsPerSection,
	}

	questions, err := h.aiService.GenerateStoryQuestions(ctx, h.convertToServicesAIConfig(ctx, user), questionsReq)
	if err != nil {
		h.logger.Warn(ctx, "Failed to generate questions for story section", map[string]interface{}{
			"section_id": section.ID,
			"story_id":   storyID,
			"error":      err.Error(),
		})
		// Continue anyway - questions are nice to have but not critical
	} else {
		// Convert to database model slice
		dbQuestions := make([]models.StorySectionQuestionData, len(questions))
		for i, q := range questions {
			dbQuestions[i] = *q
		}

		// Save the questions
		if err := h.storyService.CreateSectionQuestions(ctx, section.ID, dbQuestions); err != nil {
			h.logger.Warn(ctx, "Failed to save story questions", map[string]interface{}{
				"section_id": section.ID,
				"story_id":   storyID,
				"error":      err.Error(),
			})
		}
	}

	// Update the story's last generation time (user generation)
	if err := h.storyService.UpdateLastGenerationTime(ctx, uint(storyID), true); err != nil {
		h.logger.Warn(ctx, "Failed to update story generation time", map[string]interface{}{
			"story_id": storyID,
			"error":    err.Error(),
		})
	}

	span.SetAttributes(
		attribute.Int("story.id", int(storyID)),
		attribute.Int("section.id", int(section.ID)),
		attribute.Int("word_count", wordCount),
	)

	// Convert to API types to ensure proper serialization
	apiSection := convertStorySectionToAPI(section)
	c.JSON(http.StatusCreated, apiSection)
}

// ArchiveStory handles POST /v1/story/:id/archive
func (h *StoryHandler) ArchiveStory(c *gin.Context) {
	ctx, span := observability.TraceHandlerFunction(c.Request.Context(), "archive_story")
	defer observability.FinishSpan(span, nil)

	userID, exists := GetUserIDFromSession(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: stringPtr("unauthorized")})
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: stringPtr("invalid story ID")})
		return
	}

	err = h.storyService.ArchiveStory(ctx, uint(storyID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to archive story", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: stringPtr("story not found")})
			return
		}

		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to archive story")})
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
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: stringPtr("unauthorized")})
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: stringPtr("invalid story ID")})
		return
	}

	err = h.storyService.CompleteStory(ctx, uint(storyID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to complete story", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: stringPtr("story not found")})
			return
		}

		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to complete story")})
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
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: stringPtr("unauthorized")})
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: stringPtr("invalid story ID")})
		return
	}

	err = h.storyService.SetCurrentStory(ctx, uint(storyID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to set current story", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: stringPtr("story not found")})
			return
		}

		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to set current story")})
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
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: stringPtr("unauthorized")})
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: stringPtr("invalid story ID")})
		return
	}

	err = h.storyService.DeleteStory(ctx, uint(storyID), uint(userID))
	if err != nil {
		h.logger.Error(ctx, "Failed to delete story", err, map[string]interface{}{
			"story_id": storyID,
			"user_id":  uint(userID),
		})

		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: stringPtr("story not found")})
			return
		}

		if strings.Contains(err.Error(), "cannot delete current story") {
			c.JSON(http.StatusConflict, api.ErrorResponse{Error: stringPtr("cannot delete current story")})
			return
		}

		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to delete story")})
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
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: stringPtr("unauthorized")})
		return
	}

	storyIDStr := c.Param("id")
	storyID, err := strconv.ParseUint(storyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: stringPtr("invalid story ID")})
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
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: stringPtr("story not found")})
			return
		}

		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to get story")})
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
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: stringPtr("failed to generate PDF")})
		return
	}

	c.Data(http.StatusOK, "application/pdf", []byte(buf.String()))
}

// convertToServicesAIConfig creates AI config for the user in services format
func (h *StoryHandler) convertToServicesAIConfig(ctx context.Context, user *models.User) *services.UserAIConfig {
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

	return &services.UserAIConfig{
		Provider: aiProvider,
		Model:    aiModel,
		APIKey:   apiKey,
	}
}
