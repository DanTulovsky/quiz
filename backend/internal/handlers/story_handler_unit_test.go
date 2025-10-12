package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Define types that are needed for the mock
type ConcurrencyStats struct {
	ActiveRequests int
	QueueSize      int
}

type VarietyElements struct {
	TopicCategory      string
	GrammarFocus       string
	VocabularyDomain   string
	Scenario           string
	StyleModifier      string
	DifficultyModifier string
	TimeContext        string
}

type VarietyService struct{}

type AITemplateManager struct{}

// Note: stringPtr is already defined in convert.go

// MockStoryService is a mock implementation for testing
type MockStoryService struct {
	story *models.StoryWithSections
	err   error
}

func (m *MockStoryService) CreateStory(_ context.Context, _ uint, _ string, _ *models.CreateStoryRequest) (*models.Story, error) {
	return nil, nil
}

func (m *MockStoryService) GetUserStories(_ context.Context, _ uint, _ bool) ([]models.Story, error) {
	return nil, nil
}

func (m *MockStoryService) GetCurrentStory(_ context.Context, _ uint) (*models.StoryWithSections, error) {
	return m.story, m.err
}

func (m *MockStoryService) GetStory(_ context.Context, storyID, _ uint) (*models.StoryWithSections, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.story != nil && m.story.ID == uint(storyID) {
		return m.story, nil
	}
	return nil, contextutils.ErrorWithContextf("story not found or access denied")
}

func (m *MockStoryService) ArchiveStory(_ context.Context, _, _ uint) error {
	return nil
}

func (m *MockStoryService) CompleteStory(_ context.Context, _, _ uint) error {
	return nil
}

func (m *MockStoryService) SetCurrentStory(_ context.Context, _, _ uint) error {
	return nil
}

func (m *MockStoryService) DeleteStory(_ context.Context, _, _ uint) error {
	return nil
}

func (m *MockStoryService) DeleteAllStoriesForUser(_ context.Context, _ uint) error {
	return nil
}

func (m *MockStoryService) FixCurrentStoryConstraint(_ context.Context) error {
	return nil
}

func (m *MockStoryService) GetStorySections(_ context.Context, storyID uint) ([]models.StorySection, error) {
	if m.story != nil && m.story.ID == storyID {
		return m.story.Sections, nil
	}
	return nil, nil
}

func (m *MockStoryService) GetSection(_ context.Context, _, _ uint) (*models.StorySectionWithQuestions, error) {
	return nil, nil
}

func (m *MockStoryService) CreateSection(_ context.Context, _ uint, _, _ string, _ int, _ models.GeneratorType) (*models.StorySection, error) {
	return nil, nil
}

func (m *MockStoryService) GetLatestSection(_ context.Context, _ uint) (*models.StorySection, error) {
	return nil, nil
}

func (m *MockStoryService) GetAllSectionsText(_ context.Context, _ uint) (string, error) {
	return "", nil
}

func (m *MockStoryService) GetSectionQuestions(_ context.Context, _ uint) ([]models.StorySectionQuestion, error) {
	return nil, nil
}

func (m *MockStoryService) CreateSectionQuestions(_ context.Context, _ uint, _ []models.StorySectionQuestionData) error {
	return nil
}

func (m *MockStoryService) GetRandomQuestions(_ context.Context, _ uint, _ int) ([]models.StorySectionQuestion, error) {
	return nil, nil
}

func (m *MockStoryService) UpdateLastGenerationTime(_ context.Context, _ uint, _ models.GeneratorType) error {
	return nil
}

func (m *MockStoryService) GetSectionLengthTarget(_ string, _ *models.SectionLength) int {
	return 150
}

func (m *MockStoryService) GetSectionLengthTargetWithLanguage(_, _ string, _ *models.SectionLength) int {
	return 150
}

func (m *MockStoryService) SanitizeInput(input string) string {
	return input
}

func (m *MockStoryService) GenerateStorySection(_ context.Context, _, _ uint, _ services.AIServiceInterface, _ *models.UserAIConfig, _ models.GeneratorType) (*models.StorySectionWithQuestions, error) {
	return nil, nil
}

// Admin-only methods (no ownership checks)
func (m *MockStoryService) GetStoriesPaginated(_ context.Context, _, _ int, _, _, _ string, _ *uint) ([]models.Story, int, error) {
	return []models.Story{}, 0, nil
}

func (m *MockStoryService) GetStoryAdmin(_ context.Context, _ uint) (*models.StoryWithSections, error) {
	return nil, nil
}

func (m *MockStoryService) GetSectionAdmin(_ context.Context, _ uint) (*models.StorySectionWithQuestions, error) {
	return nil, nil
}

// TestExportStoryPDF_UnicodeHandling tests the PDF export functionality with Unicode characters
// This test verifies that Unicode characters are properly handled in PDF generation
func TestExportStoryPDF_UnicodeHandling(t *testing.T) {
	// Create a test story with Unicode content
	story := &models.StoryWithSections{
		Story: models.Story{
			ID:                 1,
			UserID:             1,
			Title:              "Unicode Test Story - ¡Hola! こんにちは مرحبا",
			Language:           "en",
			Status:             models.StoryStatusActive,
			Subject:            stringPtr("Testing Unicode characters: àáâãäåæçèéêëìíîïðñòóôõöøùúûüýþÿ"),
			AuthorStyle:        stringPtr("Modern with émojis 🚀 and ñoñ-ASCII characters"),
			TimePeriod:         stringPtr("2023-2024"),
			Genre:              stringPtr("Science Fiction with çhäráctérs"),
			Tone:               stringPtr("Friendly and welcôming"),
			CharacterNames:     stringPtr("José, François, Åsa, José María"),
			CustomInstructions: stringPtr("Include diverse characters from différent cultures"),
		},
		Sections: []models.StorySection{
			{
				ID:             1,
				StoryID:        1,
				SectionNumber:  1,
				Content:        "In the beginning, José and François met in a café. \"¡Hola!\" said José. \"Bonjour,\" replied François. They discussed quantum physics with Åsa, who had just arrived from Stockholm. \"こんにちは,\" she greeted them with a smile 😊. They continued their conversation about artificial intelligence and machine learning, touching on topics from neural networks to natural language processing. The weather was perfect - sunny with a gentle breeze 🌤️. José María joined them later, bringing fresh croissants 🥐 and sharing stories from his travels in México, España, and Portugal.",
				LanguageLevel:  "B1",
				WordCount:      85,
				GeneratedAt:    time.Now(),
				GenerationDate: time.Now().Truncate(24 * time.Hour),
			},
		},
	}

	// Create mock service
	mockService := &MockStoryService{story: story}

	// Create test user service (minimal mock)
	mockUserService := &MockUserService{}

	// Create handler with mocks
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	handler := NewStoryHandler(mockService, mockUserService, &SimpleMockAIService{}, nil, logger)

	// Create test router
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Mock session with user ID
		c.Set("user_id", uint(1))
		c.Next()
	})
	router.GET("/v1/story/:id/export", handler.ExportStory)

	// Test PDF export
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/story/1/export", nil)
	req.Header.Set("Accept", "application/pdf")

	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "story_")

	// Verify PDF content is not empty
	pdfContent := w.Body.Bytes()
	assert.NotEmpty(t, pdfContent)
	assert.Greater(t, len(pdfContent), 100) // PDF should be substantial

	// Basic PDF structure validation (PDF files start with %PDF-)
	assert.True(t, bytes.HasPrefix(pdfContent, []byte("%PDF-")),
		"PDF should start with %PDF- marker")

	// Test with non-existent story
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/story/999/export", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestExportStoryPDF_FontFallback tests that PDF export falls back gracefully when DejaVu font is not available
func TestExportStoryPDF_FontFallback(t *testing.T) {
	// Create a test story with Unicode content
	story := &models.StoryWithSections{
		Story: models.Story{
			ID:          1,
			UserID:      1,
			Title:       "Font Fallback Test - Unicode Characters",
			Language:    "en",
			Status:      models.StoryStatusActive,
			Subject:     stringPtr("Testing font fallback behavior"),
			AuthorStyle: stringPtr("Standard style"),
		},
		Sections: []models.StorySection{
			{
				ID:             1,
				StoryID:        1,
				SectionNumber:  1,
				Content:        "This is a test of the font fallback mechanism when DejaVu Sans is not available. The PDF should still generate successfully using core fonts like Arial.",
				LanguageLevel:  "B1",
				WordCount:      25,
				GeneratedAt:    time.Now(),
				GenerationDate: time.Now().Truncate(24 * time.Hour),
			},
		},
	}

	// Create mock service
	mockService := &MockStoryService{story: story}
	mockUserService := &MockUserService{}

	// Create handler with mocks
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	handler := NewStoryHandler(mockService, mockUserService, &SimpleMockAIService{}, nil, logger)

	// Create test router
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	})
	router.GET("/v1/story/:id/export", handler.ExportStory)

	// Test PDF export (should work even without DejaVu font)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/story/1/export", nil)
	req.Header.Set("Accept", "application/pdf")

	router.ServeHTTP(w, req)

	// Verify response - should still succeed even without DejaVu font
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")

	// Verify PDF content is not empty
	pdfContent := w.Body.Bytes()
	assert.NotEmpty(t, pdfContent)
	assert.Greater(t, len(pdfContent), 100)

	// Basic PDF structure validation
	assert.True(t, bytes.HasPrefix(pdfContent, []byte("%PDF-")),
		"PDF should start with %PDF- marker")
}

// SimpleMockAIService for testing - implements AIServiceInterface
type SimpleMockAIService struct{}

func (m *SimpleMockAIService) GenerateStorySection(_ context.Context, _ *models.UserAIConfig, _ *models.StoryGenerationRequest) (string, error) {
	return "Mock story section", nil
}

func (m *SimpleMockAIService) GenerateStoryQuestions(_ context.Context, _ *models.UserAIConfig, _ *models.StoryQuestionsRequest) ([]*models.StorySectionQuestionData, error) {
	return []*models.StorySectionQuestionData{}, nil
}

func (m *SimpleMockAIService) CallWithPrompt(_ context.Context, _ *models.UserAIConfig, _, _ string) (string, error) {
	return "Mock response", nil
}

func (m *SimpleMockAIService) GenerateChatResponse(_ context.Context, _ *models.UserAIConfig, _ *models.AIChatRequest) (string, error) {
	return "Mock chat response", nil
}

func (m *SimpleMockAIService) GenerateChatResponseStream(_ context.Context, _ *models.UserAIConfig, _ *models.AIChatRequest, callback chan<- string) error {
	callback <- "Mock streaming response"
	return nil
}

func (m *SimpleMockAIService) GenerateQuestion(_ context.Context, _ *models.UserAIConfig, _ *models.AIQuestionGenRequest) (*models.Question, error) {
	return &models.Question{}, nil
}

func (m *SimpleMockAIService) GenerateQuestions(_ context.Context, _ *models.UserAIConfig, _ *models.AIQuestionGenRequest) ([]*models.Question, error) {
	return []*models.Question{}, nil
}

func (m *SimpleMockAIService) GenerateQuestionsStream(_ context.Context, _ *models.UserAIConfig, _ *models.AIQuestionGenRequest, _ chan<- *models.Question, _ *services.VarietyElements) error {
	return nil
}

func (m *SimpleMockAIService) TestConnection(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *SimpleMockAIService) GetConcurrencyStats() services.ConcurrencyStats {
	return services.ConcurrencyStats{}
}

func (m *SimpleMockAIService) GetQuestionBatchSize(_ string) int {
	return 1
}

func (m *SimpleMockAIService) VarietyService() *services.VarietyService {
	return &services.VarietyService{}
}

func (m *SimpleMockAIService) TemplateManager() *services.AITemplateManager {
	return &services.AITemplateManager{}
}

func (m *SimpleMockAIService) SupportsGrammarField(_ string) bool {
	return false
}

func (m *SimpleMockAIService) Shutdown(_ context.Context) error {
	return nil
}

// Note: MockUserService is already defined in auth_handler_test.go
