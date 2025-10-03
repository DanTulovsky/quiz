package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockStoryService for testing
type MockStoryService struct {
	mock.Mock
}

func (m *MockStoryService) CreateStory(ctx context.Context, userID uint, language string, req *models.CreateStoryRequest) (result0 *models.Story, err error) {
	args := m.Called(ctx, userID, language, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Story), args.Error(1)
}

func (m *MockStoryService) GetUserStories(ctx context.Context, userID uint, includeArchived bool) (result0 []models.Story, err error) {
	args := m.Called(ctx, userID, includeArchived)
	return args.Get(0).([]models.Story), args.Error(1)
}

func (m *MockStoryService) GetCurrentStory(ctx context.Context, userID uint) (result0 *models.StoryWithSections, err error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StoryWithSections), args.Error(1)
}

func (m *MockStoryService) GetStory(ctx context.Context, storyID, userID uint) (result0 *models.StoryWithSections, err error) {
	args := m.Called(ctx, storyID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StoryWithSections), args.Error(1)
}

func (m *MockStoryService) GetSection(ctx context.Context, sectionID, userID uint) (result0 *models.StorySection, err error) {
	args := m.Called(ctx, sectionID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StorySection), args.Error(1)
}

func (m *MockStoryService) GetAllSectionsText(ctx context.Context, storyID uint) (result0 string, err error) {
	args := m.Called(ctx, storyID)
	return args.String(0), args.Error(1)
}

func (m *MockStoryService) CreateSection(ctx context.Context, storyID uint, content, language string, wordCount int) (result0 *models.StorySection, err error) {
	args := m.Called(ctx, storyID, content, language, wordCount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StorySection), args.Error(1)
}

func (m *MockStoryService) CreateSectionQuestions(ctx context.Context, sectionID uint, questions []models.StorySectionQuestionData) error {
	args := m.Called(ctx, sectionID, questions)
	return args.Error(0)
}

func (m *MockStoryService) UpdateLastGenerationTime(ctx context.Context, storyID uint) error {
	args := m.Called(ctx, storyID)
	return args.Error(0)
}

func (m *MockStoryService) CanGenerateSection(ctx context.Context, storyID uint) (result0 bool, err error) {
	args := m.Called(ctx, storyID)
	return args.Bool(0), args.Error(1)
}

func (m *MockStoryService) GetSectionLengthTarget(language string, override *models.SectionLength) int {
	args := m.Called(language, override)
	return args.Int(0)
}

func (m *MockStoryService) ArchiveStory(ctx context.Context, storyID, userID uint) error {
	args := m.Called(ctx, storyID, userID)
	return args.Error(0)
}

func (m *MockStoryService) CompleteStory(ctx context.Context, storyID, userID uint) error {
	args := m.Called(ctx, storyID, userID)
	return args.Error(0)
}

func (m *MockStoryService) SetCurrentStory(ctx context.Context, storyID, userID uint) error {
	args := m.Called(ctx, storyID, userID)
	return args.Error(0)
}

func (m *MockStoryService) DeleteStory(ctx context.Context, storyID, userID uint) error {
	args := m.Called(ctx, storyID, userID)
	return args.Error(0)
}

// MockAIService for testing
type MockAIService struct {
	mock.Mock
}

func (m *MockAIService) GenerateStorySection(ctx context.Context, config *services.UserAIConfig, req *models.StoryGenerationRequest) (result0 string, err error) {
	args := m.Called(ctx, config, req)
	return args.String(0), args.Error(1)
}

func (m *MockAIService) GenerateStoryQuestions(ctx context.Context, config *services.UserAIConfig, req *models.StoryQuestionsRequest) (result0 []*models.StorySectionQuestionData, err error) {
	args := m.Called(ctx, config, req)
	return args.Get(0).([]*models.StorySectionQuestionData), args.Error(1)
}

func (m *MockAIService) TestConnection(ctx context.Context, provider, model, apiKey string) error {
	args := m.Called(ctx, provider, model, apiKey)
	return args.Error(0)
}

func (m *MockAIService) GetConcurrencyStats() services.ConcurrencyStats {
	args := m.Called()
	return args.Get(0).(services.ConcurrencyStats)
}

func (m *MockAIService) GetQuestionBatchSize(provider string) int {
	args := m.Called(provider)
	return args.Int(0)
}

func (m *MockAIService) VarietyService() *services.VarietyService {
	args := m.Called()
	return args.Get(0).(*services.VarietyService)
}

func (m *MockAIService) TemplateManager() *services.AITemplateManager {
	args := m.Called()
	return args.Get(0).(*services.AITemplateManager)
}

func (m *MockAIService) SupportsGrammarField(provider string) bool {
	args := m.Called(provider)
	return args.Bool(0)
}

func setupStoryHandlerTest(t *testing.T) (*StoryHandler, *MockStoryService, *MockUserService, *MockAIService, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	mockStoryService := &MockStoryService{}
	mockUserService := &MockUserService{}

	cfg := &config.Config{}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{})

	handler := NewStoryHandler(mockStoryService, mockUserService, nil, cfg, logger)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		store := cookie.NewStore([]byte("secret"))
		session := sessions.NewSession(store, "test_session")
		session.Set("user_id", 1)
		c.Set("session", session)
		c.Next()
	})

	return handler, mockStoryService, mockUserService, nil, router
}

func TestStoryHandler_CreateStory(t *testing.T) {
	handler, mockStoryService, mockUserService, _, router := setupStoryHandlerTest(t)

	t.Run("should create story successfully", func(t *testing.T) {
		req := models.CreateStoryRequest{
			Title:   "Test Story",
			Subject: openapi_types.PtrString("A test story"),
		}

		expectedUser := &models.User{
			ID:                1,
			Username:          "testuser",
			Email:             sql.NullString{String: "test@example.com", Valid: true},
			PreferredLanguage: sql.NullString{String: "en", Valid: true},
		}

		expectedStory := &models.Story{
			ID:      1,
			Title:   "Test Story",
			Subject: openapi_types.PtrString("A test story"),
			UserID:  1,
		}

		mockUserService.On("GetUserByID", mock.Anything, 1).Return(expectedUser, nil)
		mockStoryService.On("CreateStory", mock.Anything, uint(1), "en", &req).Return(expectedStory, nil)

		router.POST("/v1/story", handler.CreateStory)

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		req2, _ := http.NewRequest("POST", "/v1/story", bytes.NewBuffer(body))
		req2.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req2)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockUserService.AssertExpectations(t)
		mockStoryService.AssertExpectations(t)
	})

	t.Run("should return unauthorized when no user session", func(t *testing.T) {
		routerNoAuth := gin.New()
		routerNoAuth.POST("/v1/story", handler.CreateStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story", nil)

		routerNoAuth.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("should return bad request for invalid JSON", func(t *testing.T) {
		router.POST("/v1/story", handler.CreateStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestStoryHandler_GetUserStories(t *testing.T) {
	handler, mockStoryService, mockUserService, _, router := setupStoryHandlerTest(t)

	t.Run("should get user stories successfully", func(t *testing.T) {
		expectedStories := []models.Story{
			{ID: 1, Title: "Story 1"},
			{ID: 2, Title: "Story 2"},
		}

		mockStoryService.On("GetUserStories", mock.Anything, uint(1), false).Return(expectedStories, nil)

		router.GET("/v1/story", handler.GetUserStories)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/story", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []models.Story
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
		assert.Equal(t, "Story 1", response[0].Title)

		mockStoryService.AssertExpectations(t)
	})

	t.Run("should handle include archived parameter", func(t *testing.T) {
		mockStoryService.On("GetUserStories", mock.Anything, uint(1), true).Return([]models.Story{}, nil)

		router.GET("/v1/story", handler.GetUserStories)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/story?include_archived=true", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockStoryService.AssertExpectations(t)
	})
}

func TestStoryHandler_GetCurrentStory(t *testing.T) {
	handler, mockStoryService, mockUserService, _, router := setupStoryHandlerTest(t)

	t.Run("should get current story successfully", func(t *testing.T) {
		expectedStory := &models.StoryWithSections{
			Story: models.Story{ID: 1, Title: "Current Story"},
		}

		mockStoryService.On("GetCurrentStory", mock.Anything, uint(1)).Return(expectedStory, nil)

		router.GET("/v1/story/current", handler.GetCurrentStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/story/current", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockStoryService.AssertExpectations(t)
	})

	t.Run("should return not found when no current story", func(t *testing.T) {
		mockStoryService.On("GetCurrentStory", mock.Anything, uint(1)).Return(nil, fmt.Errorf("no current story"))

		router.GET("/v1/story/current", handler.GetCurrentStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/story/current", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockStoryService.AssertExpectations(t)
	})
}

func TestStoryHandler_GetStory(t *testing.T) {
	handler, mockStoryService, mockUserService, mockAIService, router := setupStoryHandlerTest(t)

	t.Run("should get story successfully", func(t *testing.T) {
		expectedStory := &models.StoryWithSections{
			Story: models.Story{ID: 1, Title: "Test Story"},
		}

		mockStoryService.On("GetStory", mock.Anything, uint(1), uint(1)).Return(expectedStory, nil)

		router.GET("/v1/story/1", handler.GetStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/story/1", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockStoryService.AssertExpectations(t)
	})

	t.Run("should return not found for non-existent story", func(t *testing.T) {
		mockStoryService.On("GetStory", mock.Anything, uint(1), uint(1)).Return(nil, fmt.Errorf("story not found"))

		router.GET("/v1/story/1", handler.GetStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/story/1", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockStoryService.AssertExpectations(t)
	})

	t.Run("should return bad request for invalid story ID", func(t *testing.T) {
		router.GET("/v1/story/invalid", handler.GetStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/story/invalid", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestStoryHandler_GetSection(t *testing.T) {
	handler, mockStoryService, mockUserService, mockAIService, router := setupStoryHandlerTest(t)

	t.Run("should get section successfully", func(t *testing.T) {
		expectedSection := &models.StorySection{
			ID:            1,
			StoryID:       1,
			SectionNumber: 1,
			Content:       "Test section content",
		}

		mockStoryService.On("GetSection", mock.Anything, uint(1), uint(1)).Return(expectedSection, nil)

		router.GET("/v1/story/section/1", handler.GetSection)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/story/section/1", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockStoryService.AssertExpectations(t)
	})

	t.Run("should return not found for non-existent section", func(t *testing.T) {
		mockStoryService.On("GetSection", mock.Anything, uint(1), uint(1)).Return(nil, fmt.Errorf("section not found"))

		router.GET("/v1/story/section/1", handler.GetSection)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/story/section/1", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockStoryService.AssertExpectations(t)
	})
}

func TestStoryHandler_GenerateNextSection(t *testing.T) {
	handler, mockStoryService, mockUserService, mockAIService, router := setupStoryHandlerTest(t)

	t.Run("should generate next section successfully", func(t *testing.T) {
		existingStory := &models.StoryWithSections{
			Story: models.Story{
				ID:       1,
				Title:    "Test Story",
				Language: "en",
				UserID:   1,
				Subject:  openapi_types.PtrString("Test subject"),
			},
		}

		expectedSection := &models.StorySection{
			ID:            1,
			StoryID:       1,
			SectionNumber: 1,
			Content:       "Generated section content",
		}

		mockStoryService.On("GetStory", mock.Anything, uint(1), uint(1)).Return(existingStory, nil)
		mockStoryService.On("CanGenerateSection", mock.Anything, uint(1)).Return(true, nil)
		mockUserService.On("GetUserByID", mock.Anything, 1).Return(&models.User{
			ID:                1,
			Username:          "testuser",
			Email:             sql.NullString{String: "test@example.com", Valid: true},
			PreferredLanguage: sql.NullString{String: "en", Valid: true},
		}, nil)
		mockStoryService.On("GetAllSectionsText", mock.Anything, uint(1)).Return([]string{}, nil)
		mockStoryService.On("GetSectionLengthTarget", "en", (*models.SectionLength)(nil)).Return(200)
		mockAIService.On("GenerateStorySection", mock.Anything, mock.Anything, mock.Anything).Return("Generated content", nil)
		mockStoryService.On("CreateSection", mock.Anything, uint(1), "Generated content", "en", 5).Return(expectedSection, nil)
		mockStoryService.On("UpdateLastGenerationTime", mock.Anything, uint(1)).Return(nil)

		router.POST("/v1/story/1/generate", handler.GenerateNextSection)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story/1/generate", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockStoryService.AssertExpectations(t)
		mockAIService.AssertExpectations(t)
	})

	t.Run("should return conflict when generation not allowed", func(t *testing.T) {
		existingStory := &models.StoryWithSections{
			Story: models.Story{ID: 1, Title: "Test Story"},
		}

		mockStoryService.On("GetStory", mock.Anything, uint(1), uint(1)).Return(existingStory, nil)
		mockStoryService.On("CanGenerateSection", mock.Anything, uint(1)).Return(false, nil)

		router.POST("/v1/story/1/generate", handler.GenerateNextSection)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story/1/generate", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		mockStoryService.AssertExpectations(t)
	})
}

func TestStoryHandler_ArchiveStory(t *testing.T) {
	handler, mockStoryService, mockUserService, mockAIService, router := setupStoryHandlerTest(t)

	t.Run("should archive story successfully", func(t *testing.T) {
		mockStoryService.On("ArchiveStory", mock.Anything, uint(1), uint(1)).Return(nil)

		router.POST("/v1/story/1/archive", handler.ArchiveStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story/1/archive", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockStoryService.AssertExpectations(t)
	})

	t.Run("should return not found for non-existent story", func(t *testing.T) {
		mockStoryService.On("ArchiveStory", mock.Anything, uint(1), uint(1)).Return(fmt.Errorf("story not found"))

		router.POST("/v1/story/1/archive", handler.ArchiveStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story/1/archive", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockStoryService.AssertExpectations(t)
	})
}

func TestStoryHandler_CompleteStory(t *testing.T) {
	handler, mockStoryService, mockUserService, mockAIService, router := setupStoryHandlerTest(t)

	t.Run("should complete story successfully", func(t *testing.T) {
		mockStoryService.On("CompleteStory", mock.Anything, uint(1), uint(1)).Return(nil)

		router.POST("/v1/story/1/complete", handler.CompleteStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story/1/complete", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockStoryService.AssertExpectations(t)
	})
}

func TestStoryHandler_SetCurrentStory(t *testing.T) {
	handler, mockStoryService, mockUserService, mockAIService, router := setupStoryHandlerTest(t)

	t.Run("should set current story successfully", func(t *testing.T) {
		mockStoryService.On("SetCurrentStory", mock.Anything, uint(1), uint(1)).Return(nil)

		router.POST("/v1/story/1/set-current", handler.SetCurrentStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story/1/set-current", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockStoryService.AssertExpectations(t)
	})
}

func TestStoryHandler_DeleteStory(t *testing.T) {
	handler, mockStoryService, mockUserService, mockAIService, router := setupStoryHandlerTest(t)

	t.Run("should delete story successfully", func(t *testing.T) {
		mockStoryService.On("DeleteStory", mock.Anything, uint(1), uint(1)).Return(nil)

		router.DELETE("/v1/story/1", handler.DeleteStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/v1/story/1", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		mockStoryService.AssertExpectations(t)
	})

	t.Run("should return conflict when trying to delete current story", func(t *testing.T) {
		mockStoryService.On("DeleteStory", mock.Anything, uint(1), uint(1)).Return(fmt.Errorf("cannot delete current story"))

		router.DELETE("/v1/story/1", handler.DeleteStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/v1/story/1", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		mockStoryService.AssertExpectations(t)
	})
}

func TestStoryHandler_ExportStory(t *testing.T) {
	handler, mockStoryService, mockUserService, mockAIService, router := setupStoryHandlerTest(t)

	t.Run("should export story successfully", func(t *testing.T) {
		expectedStory := &models.StoryWithSections{
			Story: models.Story{
				ID:    1,
				Title: "Test Story",
			},
		}

		mockStoryService.On("GetStory", mock.Anything, uint(1), uint(1)).Return(expectedStory, nil)

		router.GET("/v1/story/1/export", handler.ExportStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/story/1/export", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Header().Get("Content-Disposition"), "story_Test_Story.pdf")

		mockStoryService.AssertExpectations(t)
	})
}
