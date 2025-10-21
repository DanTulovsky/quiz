package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLearningService is a mock implementation of LearningServiceInterface
type MockLearningService struct {
	mock.Mock
}

type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendDailyReminder(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockEmailService) SendEmail(ctx context.Context, to, subject, templateName string, data map[string]interface{}) error {
	args := m.Called(ctx, to, subject, templateName, data)
	return args.Error(0)
}

func (m *MockEmailService) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockEmailService) RecordSentNotification(ctx context.Context, userID int, notificationType, subject, templateName, status, errorMessage string) error {
	args := m.Called(ctx, userID, notificationType, subject, templateName, status, errorMessage)
	return args.Error(0)
}

func (m *MockLearningService) GetUserLearningPreferences(ctx context.Context, userID int) (result0 *models.UserLearningPreferences, err error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserLearningPreferences), args.Error(1)
}

func (m *MockLearningService) UpdateUserLearningPreferences(ctx context.Context, userID int, prefs *models.UserLearningPreferences) (result0 *models.UserLearningPreferences, err error) {
	args := m.Called(ctx, userID, prefs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserLearningPreferences), args.Error(1)
}

func (m *MockLearningService) UpdateLastDailyReminderSent(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockLearningService) CalculatePriorityScore(ctx context.Context, userID, questionID int) (result0 float64, err error) {
	args := m.Called(ctx, userID, questionID)
	return args.Get(0).(float64), args.Error(1)
}

// Add stub methods for other interface methods that aren't used in these tests
func (m *MockLearningService) RecordUserResponse(ctx context.Context, response *models.UserResponse) error {
	args := m.Called(ctx, response)
	return args.Error(0)
}

func (m *MockLearningService) GetUserProgress(ctx context.Context, userID int) (result0 *models.UserProgress, err error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProgress), args.Error(1)
}

func (m *MockLearningService) GetWeakestTopics(ctx context.Context, userID, limit int) (result0 []*models.PerformanceMetrics, err error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PerformanceMetrics), args.Error(1)
}

func (m *MockLearningService) ShouldAvoidQuestion(ctx context.Context, userID, questionID int) (result0 bool, err error) {
	args := m.Called(ctx, userID, questionID)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockLearningService) GetUserQuestionStats(ctx context.Context, userID int) (result0 *services.UserQuestionStats, err error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.UserQuestionStats), args.Error(1)
}

func (m *MockLearningService) RecordAnswerWithPriority(ctx context.Context, userID, questionID, answerIndex int, isCorrect bool, responseTime int) error {
	args := m.Called(ctx, userID, questionID, answerIndex, isCorrect, responseTime)
	return args.Error(0)
}

// Mock implementation that returns the created user_responses id for tests
func (m *MockLearningService) RecordAnswerWithPriorityReturningID(ctx context.Context, userID, questionID, answerIndex int, isCorrect bool, responseTime int) (int, error) {
	args := m.Called(ctx, userID, questionID, answerIndex, isCorrect, responseTime)
	if args.Get(0) == nil {
		return 0, args.Error(1)
	}
	return args.Int(0), args.Error(1)
}

func (m *MockLearningService) MarkQuestionAsKnown(ctx context.Context, userID, questionID int, confidenceLevel *int) error {
	args := m.Called(ctx, userID, questionID, confidenceLevel)
	return args.Error(0)
}

func (m *MockLearningService) GetPriorityScoreDistribution(_ context.Context) (result0 map[string]any, err error) {
	return map[string]any{}, nil
}

func (m *MockLearningService) GetHighPriorityQuestions(_ context.Context, _ int) (result0 []map[string]any, err error) {
	return []map[string]any{}, nil
}

func (m *MockLearningService) GetWeakAreasByTopic(_ context.Context, _ int) (result0 []map[string]any, err error) {
	return []map[string]any{}, nil
}

func (m *MockLearningService) GetLearningPreferencesUsage(_ context.Context) (result0 map[string]any, err error) {
	return map[string]any{}, nil
}

func (m *MockLearningService) GetQuestionTypeGaps(_ context.Context) (result0 []map[string]any, err error) {
	return []map[string]any{}, nil
}

func (m *MockLearningService) GetGenerationSuggestions(_ context.Context) (result0 []map[string]any, err error) {
	return []map[string]any{}, nil
}

func (m *MockLearningService) GetPrioritySystemPerformance(_ context.Context) (result0 map[string]any, err error) {
	return map[string]any{}, nil
}

func (m *MockLearningService) GetBackgroundJobsStatus(_ context.Context) (result0 map[string]any, err error) {
	return map[string]any{}, nil
}

func (m *MockLearningService) GetUserHighPriorityQuestions(ctx context.Context, userID, limit int) (result0 []map[string]any, err error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]map[string]any), args.Error(1)
}

func (m *MockLearningService) GetUserPriorityScoreDistribution(ctx context.Context, userID int) (result0 map[string]any, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[string]any), args.Error(1)
}

func (m *MockLearningService) GetUserWeakAreas(ctx context.Context, userID, limit int) (result0 []map[string]any, err error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]map[string]any), args.Error(1)
}

func (m *MockLearningService) GetHighPriorityTopics(ctx context.Context, userID int) (result0 []string, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockLearningService) GetGapAnalysis(ctx context.Context, userID int) (result0 map[string]any, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[string]any), args.Error(1)
}

func (m *MockLearningService) GetPriorityDistribution(ctx context.Context, userID int) (result0 map[string]int, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *MockLearningService) GetUserQuestionConfidenceLevel(ctx context.Context, userID, questionID int) (*int, error) {
	args := m.Called(ctx, userID, questionID)
	return args.Get(0).(*int), args.Error(1)
}

func setupSettingsTestRouter(learningService services.LearningServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup session middleware
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("test-session", store))

	mockConfig := &config.Config{}

	// Create a mock email service
	mockEmailService := &MockEmailService{}

	handler := NewSettingsHandler(nil, nil, nil, nil, learningService, mockEmailService, nil, mockConfig, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	router.GET("/preferences/learning", handler.GetLearningPreferences)
	router.PUT("/preferences/learning", handler.UpdateLearningPreferences)

	return router
}

func TestGetProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockConfig := &config.Config{
		Providers: []config.ProviderConfig{
			{
				Name: "Test Provider",
				Code: "test",
				Models: []config.AIModel{
					{Name: "Test Model", Code: "test-model"},
				},
			},
		},
	}

	// aiService and userService can be nil for this handler test
	mockEmailService := &MockEmailService{}
	handler := NewSettingsHandler(nil, nil, nil, nil, nil, mockEmailService, nil, mockConfig, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	router := gin.New()
	router.GET("/ai-providers", handler.GetProviders)

	req, _ := http.NewRequest(http.MethodGet, "/ai-providers", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	providers, ok := response["providers"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, providers, 1)

	provider := providers[0].(map[string]interface{})
	assert.Equal(t, "Test Provider", provider["name"])
}

func TestGetLearningPreferences(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		mockLearningService := &MockLearningService{}
		router := setupSettingsTestRouter(mockLearningService)

		expectedPrefs := &models.UserLearningPreferences{
			UserID:               1,
			FocusOnWeakAreas:     true,
			FreshQuestionRatio:   0.3,
			KnownQuestionPenalty: 2.0,
			ReviewIntervalDays:   7,
			WeakAreaBoost:        1.5,
		}

		mockLearningService.On("GetUserLearningPreferences", mock.Anything, 1).Return(expectedPrefs, nil)

		req, _ := http.NewRequest(http.MethodGet, "/preferences/learning", nil)
		rr := httptest.NewRecorder()

		// Set up session with user ID before making the request
		store := cookie.NewStore([]byte("test-secret"))
		session, _ := store.Get(req, "test-session")
		session.Values["user_id"] = 1
		saveErr := session.Save(req, rr)
		assert.NoError(t, saveErr)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.UserLearningPreferences
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Equal(t, expectedPrefs.FocusOnWeakAreas, response.FocusOnWeakAreas)
		assert.Equal(t, expectedPrefs.FreshQuestionRatio, response.FreshQuestionRatio)
		assert.Equal(t, expectedPrefs.KnownQuestionPenalty, response.KnownQuestionPenalty)
		assert.Equal(t, expectedPrefs.ReviewIntervalDays, response.ReviewIntervalDays)
		assert.Equal(t, expectedPrefs.WeakAreaBoost, response.WeakAreaBoost)

		mockLearningService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockLearningService := &MockLearningService{}
		router := setupSettingsTestRouter(mockLearningService)

		mockLearningService.On("GetUserLearningPreferences", mock.Anything, 1).Return(nil, assert.AnError)

		req, _ := http.NewRequest(http.MethodGet, "/preferences/learning", nil)
		rr := httptest.NewRecorder()

		// Set up session with user ID before making the request
		store := cookie.NewStore([]byte("test-secret"))
		session, _ := store.Get(req, "test-session")
		session.Values["user_id"] = 1
		saveErr := session.Save(req, rr)
		assert.NoError(t, saveErr)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		var response map[string]any
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "code")
		assert.Contains(t, response, "message")

		mockLearningService.AssertExpectations(t)
	})
}

func TestUpdateLearningPreferences(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		mockLearningService := &MockLearningService{}
		router := setupSettingsTestRouter(mockLearningService)

		prefs := models.UserLearningPreferences{
			FocusOnWeakAreas:     true,
			FreshQuestionRatio:   0.4,
			KnownQuestionPenalty: 2.5,
			ReviewIntervalDays:   10,
			WeakAreaBoost:        2.0,
		}

		expectedPrefs := &models.UserLearningPreferences{
			UserID:               1,
			FocusOnWeakAreas:     prefs.FocusOnWeakAreas,
			FreshQuestionRatio:   prefs.FreshQuestionRatio,
			KnownQuestionPenalty: prefs.KnownQuestionPenalty,
			ReviewIntervalDays:   prefs.ReviewIntervalDays,
			WeakAreaBoost:        prefs.WeakAreaBoost,
		}

		mockLearningService.On("UpdateUserLearningPreferences", mock.Anything, 1, mock.AnythingOfType("*models.UserLearningPreferences")).Return(expectedPrefs, nil)

		jsonData, _ := json.Marshal(prefs)
		req, _ := http.NewRequest(http.MethodPut, "/preferences/learning", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Set up session with user ID before making the request
		store := cookie.NewStore([]byte("test-secret"))
		session, _ := store.Get(req, "test-session")
		session.Values["user_id"] = 1
		saveErr := session.Save(req, rr)
		assert.NoError(t, saveErr)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response api.UserLearningPreferences
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, prefs.FocusOnWeakAreas, response.FocusOnWeakAreas)
		assert.Equal(t, float32(prefs.FreshQuestionRatio), response.FreshQuestionRatio)
		assert.Equal(t, float32(prefs.KnownQuestionPenalty), response.KnownQuestionPenalty)
		assert.Equal(t, prefs.ReviewIntervalDays, response.ReviewIntervalDays)
		assert.Equal(t, float32(prefs.WeakAreaBoost), response.WeakAreaBoost)

		mockLearningService.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockLearningService := &MockLearningService{}
		router := setupSettingsTestRouter(mockLearningService)

		req, _ := http.NewRequest(http.MethodPut, "/preferences/learning", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Set up session with user ID before making the request
		store := cookie.NewStore([]byte("test-secret"))
		session, _ := store.Get(req, "test-session")
		session.Values["user_id"] = 1
		saveErr := session.Save(req, rr)
		assert.NoError(t, saveErr)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var response map[string]any
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "code")
		assert.Contains(t, response, "message")

		mockLearningService.AssertNotCalled(t, "UpdateUserLearningPreferences")
	})

	t.Run("service error", func(t *testing.T) {
		mockLearningService := &MockLearningService{}
		router := setupSettingsTestRouter(mockLearningService)

		prefs := models.UserLearningPreferences{
			FocusOnWeakAreas:     false,
			FreshQuestionRatio:   0.2,
			KnownQuestionPenalty: 1.5,
			ReviewIntervalDays:   5,
			WeakAreaBoost:        1.2,
		}

		mockLearningService.On("UpdateUserLearningPreferences", mock.Anything, 1, mock.AnythingOfType("*models.UserLearningPreferences")).Return(nil, assert.AnError)

		jsonData, _ := json.Marshal(prefs)
		req, _ := http.NewRequest(http.MethodPut, "/preferences/learning", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Set up session with user ID before making the request
		store := cookie.NewStore([]byte("test-secret"))
		session, _ := store.Get(req, "test-session")
		session.Values["user_id"] = 1
		saveErr := session.Save(req, rr)
		assert.NoError(t, saveErr)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		var response map[string]any
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "code")
		assert.Contains(t, response, "message")

		mockLearningService.AssertExpectations(t)
	})
}
