//go:build integration

package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	"quizapp/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test constants to replace the moved ones
const (
	testUserID                 = 123
	highPerformanceThreshold   = 10
	mediumPerformanceThreshold = 20
	lowPerformanceThreshold    = 30
	highPriorityScore          = 250.0
	mediumPriorityScore        = 220.0
	lowPriorityScore           = 150.5
	mockTotalAttempts          = 100
	mockCorrectAttempts        = 75
	mockTotalAttempts2         = 50
	mockCorrectAttempts2       = 30
	mockAvailable1             = 10
	mockDemand1                = 15
	mockAvailable2             = 5
	mockDemand2                = 8
	mockCount                  = 3
	mockCalculationsPerSecond  = 50
	mockAvgCalculationTime     = 0.5
	mockAvgQueryTime           = 0.2
	mockMemoryUsage            = 512
	mockAvgScore               = 85.5
	mockPriorityUpdates        = 10
)

func TestWorkerAdminHandler_PauseWorker(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockWorkerService := createMockWorkerService()
	mockUserService := createMockUserService()
	mockQuestionService := createMockQuestionService()
	mockAIService := createMockAIService()
	mockConfig := &config.Config{}
	mockWorker := createMockWorker()
	mockLearningService := createMockLearningService()
	mockDailyQuestionService := createMockDailyQuestionService()

	mockWorkerService.On("SetGlobalPause", mock.Anything, true).Return(nil)

	handler := NewWorkerAdminHandlerWithLogger(mockUserService, mockQuestionService, mockAIService, mockConfig, mockWorker, mockWorkerService, mockLearningService, mockDailyQuestionService, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	router := gin.New()
	router.POST("/v1/worker/pause", handler.PauseWorker)

	req, _ := http.NewRequest("POST", "/v1/worker/pause", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Worker paused globally", response["message"])

	mockWorkerService.AssertExpectations(t)
}

func TestWorkerAdminHandler_ResumeWorker(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockWorkerService := createMockWorkerService()
	mockUserService := createMockUserService()
	mockQuestionService := createMockQuestionService()
	mockAIService := createMockAIService()
	mockConfig := &config.Config{}
	mockWorker := createMockWorker()
	mockLearningService := createMockLearningService()
	mockDailyQuestionService := createMockDailyQuestionService()

	mockWorkerService.On("SetGlobalPause", mock.Anything, false).Return(nil)

	handler := NewWorkerAdminHandlerWithLogger(mockUserService, mockQuestionService, mockAIService, mockConfig, mockWorker, mockWorkerService, mockLearningService, mockDailyQuestionService, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	router := gin.New()
	router.POST("/v1/worker/resume", handler.ResumeWorker)

	req, _ := http.NewRequest("POST", "/v1/worker/resume", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Worker resumed globally", response["message"])

	mockWorkerService.AssertExpectations(t)
}

func TestWorkerAdminHandler_PauseUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockWorkerService := createMockWorkerService()
	mockUserService := createMockUserService()
	mockQuestionService := createMockQuestionService()
	mockAIService := createMockAIService()
	mockConfig := &config.Config{}
	mockWorker := createMockWorker()
	mockLearningService := createMockLearningService()
	mockDailyQuestionService := createMockDailyQuestionService()

	mockWorkerService.On("SetUserPause", mock.Anything, testUserID, true).Return(nil)

	handler := NewWorkerAdminHandlerWithLogger(mockUserService, mockQuestionService, mockAIService, mockConfig, mockWorker, mockWorkerService, mockLearningService, mockDailyQuestionService, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	router := gin.New()
	router.POST("/v1/worker/pause-user", handler.PauseWorkerUser)

	requestBody := map[string]interface{}{
		"user_id": testUserID,
	}
	jsonBody, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/v1/worker/pause-user", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "User paused successfully", response["message"])

	mockWorkerService.AssertExpectations(t)
}

func TestWorkerAdminHandler_GetWorkerHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockWorkerService := createMockWorkerService()
	mockUserService := createMockUserService()
	mockQuestionService := createMockQuestionService()
	mockAIService := createMockAIService()
	mockConfig := &config.Config{}
	mockWorker := createMockWorker()
	mockLearningService := createMockLearningService()
	mockDailyQuestionService := createMockDailyQuestionService()

	// This is the data returned by the worker service
	healthData := map[string]interface{}{
		"global_pause":     false,
		"healthy_count":    1,
		"total_count":      1,
		"worker_instances": []interface{}{},
	}

	mockWorkerService.On("GetWorkerHealth", mock.Anything).Return(healthData, nil)

	handler := NewWorkerAdminHandlerWithLogger(mockUserService, mockQuestionService, mockAIService, mockConfig, mockWorker, mockWorkerService, mockLearningService, mockDailyQuestionService, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call the correct handler function that returns JSON
	handler.GetSystemHealth(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, false, response["global_pause"])
	assert.Equal(t, float64(1), response["healthy_count"])

	mockWorkerService.AssertExpectations(t)
}

func TestWorkerAdminHandler_GetPriorityAnalytics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockWorkerService := createMockWorkerService()
	mockUserService := createMockUserService()
	mockQuestionService := createMockQuestionService()
	mockAIService := createMockAIService()
	mockConfig := &config.Config{}
	mockWorker := createMockWorker()
	mockLearningService := createMockLearningService()
	mockDailyQuestionService := createMockDailyQuestionService()

	expectedDistribution := map[string]interface{}{
		"high":    highPerformanceThreshold,
		"medium":  mediumPerformanceThreshold,
		"low":     lowPerformanceThreshold,
		"average": lowPriorityScore,
	}

	expectedQuestions := []map[string]interface{}{
		{
			"question_type":  "vocabulary",
			"level":          "intermediate",
			"topic":          "business",
			"priority_score": highPriorityScore,
		},
		{
			"question_type":  "qa",
			"level":          "advanced",
			"topic":          "technology",
			"priority_score": mediumPriorityScore,
		},
	}

	mockLearningService.On("GetPriorityScoreDistribution", mock.Anything).Return(expectedDistribution, nil)
	mockLearningService.On("GetHighPriorityQuestions", mock.Anything, 5).Return(expectedQuestions, nil)

	handler := NewWorkerAdminHandlerWithLogger(mockUserService, mockQuestionService, mockAIService, mockConfig, mockWorker, mockWorkerService, mockLearningService, mockDailyQuestionService, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	router := gin.New()
	router.GET("/v1/analytics/priority-scores", handler.GetPriorityAnalytics)

	req, _ := http.NewRequest("GET", "/v1/analytics/priority-scores", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check distribution data
	distribution := response["distribution"].(map[string]interface{})
	assert.Equal(t, float64(highPerformanceThreshold), distribution["high"])
	assert.Equal(t, float64(mediumPerformanceThreshold), distribution["medium"])
	assert.Equal(t, float64(lowPerformanceThreshold), distribution["low"])
	assert.Equal(t, lowPriorityScore, distribution["average"])

	// Check high priority questions
	questions := response["highPriorityQuestions"].([]interface{})
	assert.Len(t, questions, 2)

	question1 := questions[0].(map[string]interface{})
	assert.Equal(t, "vocabulary", question1["question_type"])
	assert.Equal(t, "intermediate", question1["level"])
	assert.Equal(t, "business", question1["topic"])
	assert.Equal(t, highPriorityScore, question1["priority_score"])

	mockLearningService.AssertExpectations(t)
}

func TestWorkerAdminHandler_GetUserPerformanceAnalytics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockWorkerService := createMockWorkerService()
	mockUserService := createMockUserService()
	mockQuestionService := createMockQuestionService()
	mockAIService := createMockAIService()
	mockConfig := &config.Config{}
	mockWorker := createMockWorker()
	mockLearningService := createMockLearningService()
	mockDailyQuestionService := createMockDailyQuestionService()

	expectedWeakAreas := []map[string]interface{}{
		{
			"topic":            "grammar",
			"total_attempts":   mockTotalAttempts,
			"correct_attempts": mockCorrectAttempts,
		},
		{
			"topic":            "vocabulary",
			"total_attempts":   mockTotalAttempts2,
			"correct_attempts": mockCorrectAttempts2,
		},
	}

	expectedPreferences := map[string]interface{}{
		"total_users":          5,
		"focusOnWeakAreas":     true,
		"freshQuestionRatio":   0.3,
		"weakAreaBoost":        2.0,
		"knownQuestionPenalty": 0.1,
	}

	mockLearningService.On("GetWeakAreasByTopic", mock.Anything, 5).Return(expectedWeakAreas, nil)
	mockLearningService.On("GetLearningPreferencesUsage", mock.Anything).Return(expectedPreferences, nil)

	handler := NewWorkerAdminHandlerWithLogger(mockUserService, mockQuestionService, mockAIService, mockConfig, mockWorker, mockWorkerService, mockLearningService, mockDailyQuestionService, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	router := gin.New()
	router.GET("/v1/analytics/user-performance", handler.GetUserPerformanceAnalytics)

	req, _ := http.NewRequest("GET", "/v1/analytics/user-performance", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check weak areas data
	weakAreas := response["weakAreas"].([]interface{})
	assert.Len(t, weakAreas, 2)

	area1 := weakAreas[0].(map[string]interface{})
	assert.Equal(t, "grammar", area1["topic"])
	assert.Equal(t, float64(mockTotalAttempts), area1["total_attempts"])
	assert.Equal(t, float64(mockCorrectAttempts), area1["correct_attempts"])

	// Check learning preferences data
	preferences := response["learningPreferences"].(map[string]interface{})
	assert.Equal(t, float64(5), preferences["total_users"])
	assert.Equal(t, true, preferences["focusOnWeakAreas"])
	assert.Equal(t, 0.3, preferences["freshQuestionRatio"])

	mockLearningService.AssertExpectations(t)
}

func TestWorkerAdminHandler_GetGenerationIntelligence(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockWorkerService := createMockWorkerService()
	mockUserService := createMockUserService()
	mockQuestionService := createMockQuestionService()
	mockAIService := createMockAIService()
	mockConfig := &config.Config{}
	mockWorker := createMockWorker()
	mockLearningService := createMockLearningService()

	expectedGapAnalysis := []map[string]interface{}{
		{
			"question_type": "vocabulary",
			"level":         "beginner",
			"available":     mockAvailable1,
			"demand":        mockDemand1,
		},
		{
			"question_type": "grammar",
			"level":         "intermediate",
			"available":     mockAvailable2,
			"demand":        mockDemand2,
		},
	}

	expectedSuggestions := []map[string]interface{}{
		{
			"question_type": "reading",
			"level":         "advanced",
			"count":         mockCount,
			"priority":      1,
		},
	}

	mockLearningService.On("GetQuestionTypeGaps", mock.Anything).Return(expectedGapAnalysis, nil)
	mockLearningService.On("GetGenerationSuggestions", mock.Anything).Return(expectedSuggestions, nil)

	mockDailyQuestionService := createMockDailyQuestionService()
	handler := NewWorkerAdminHandlerWithLogger(mockUserService, mockQuestionService, mockAIService, mockConfig, mockWorker, mockWorkerService, mockLearningService, mockDailyQuestionService, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	router := gin.New()
	router.GET("/v1/analytics/generation-intelligence", handler.GetGenerationIntelligence)

	req, _ := http.NewRequest("GET", "/v1/analytics/generation-intelligence", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check gap analysis data
	gapAnalysis := response["gapAnalysis"].([]interface{})
	assert.Len(t, gapAnalysis, 2)

	gap1 := gapAnalysis[0].(map[string]interface{})
	assert.Equal(t, "vocabulary", gap1["question_type"])
	assert.Equal(t, "beginner", gap1["level"])
	assert.Equal(t, float64(mockAvailable1), gap1["available"])
	assert.Equal(t, float64(mockDemand1), gap1["demand"])

	// Check generation suggestions data
	suggestions := response["generationSuggestions"].([]interface{})
	assert.Len(t, suggestions, 1)

	suggestion1 := suggestions[0].(map[string]interface{})
	assert.Equal(t, "reading", suggestion1["question_type"])
	assert.Equal(t, "advanced", suggestion1["level"])
	assert.Equal(t, float64(mockCount), suggestion1["count"])
	assert.Equal(t, float64(1), suggestion1["priority"])

	mockLearningService.AssertExpectations(t)
}

func TestWorkerAdminHandler_GetSystemHealthAnalytics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockWorkerService := createMockWorkerService()
	mockUserService := createMockUserService()
	mockQuestionService := createMockQuestionService()
	mockAIService := createMockAIService()
	mockConfig := &config.Config{}
	mockWorker := createMockWorker()
	mockLearningService := createMockLearningService()

	expectedPerformance := map[string]interface{}{
		"calculationsPerSecond": mockCalculationsPerSecond,
		"avgCalculationTime":    mockAvgCalculationTime,
		"avgQueryTime":          mockAvgQueryTime,
		"memoryUsage":           mockMemoryUsage,
		"avgScore":              mockAvgScore,
		"lastCalculation":       "2023-01-01T12:00:00Z",
	}

	expectedBackgroundJobs := map[string]interface{}{
		"priorityUpdates": mockPriorityUpdates,
		"lastUpdate":      "2023-01-01T12:00:00Z",
		"queueSize":       5,
		"status":          "healthy",
	}

	mockLearningService.On("GetPrioritySystemPerformance", mock.Anything).Return(expectedPerformance, nil)
	mockLearningService.On("GetBackgroundJobsStatus", mock.Anything).Return(expectedBackgroundJobs, nil)

	mockDailyQuestionService := createMockDailyQuestionService()
	handler := NewWorkerAdminHandlerWithLogger(mockUserService, mockQuestionService, mockAIService, mockConfig, mockWorker, mockWorkerService, mockLearningService, mockDailyQuestionService, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	router := gin.New()
	router.GET("/v1/analytics/system-health", handler.GetSystemHealthAnalytics)

	req, _ := http.NewRequest("GET", "/v1/analytics/system-health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check performance data
	performance := response["performance"].(map[string]interface{})
	assert.Equal(t, float64(mockCalculationsPerSecond), performance["calculationsPerSecond"])
	assert.Equal(t, mockAvgCalculationTime, performance["avgCalculationTime"])
	assert.Equal(t, mockAvgQueryTime, performance["avgQueryTime"])
	assert.Equal(t, float64(mockMemoryUsage), performance["memoryUsage"])
	assert.Equal(t, mockAvgScore, performance["avgScore"])
	assert.Equal(t, "2023-01-01T12:00:00Z", performance["lastCalculation"])

	// Check background jobs data
	backgroundJobs := response["backgroundJobs"].(map[string]interface{})
	assert.Equal(t, float64(mockPriorityUpdates), backgroundJobs["priorityUpdates"])
	assert.Equal(t, "2023-01-01T12:00:00Z", backgroundJobs["lastUpdate"])
	assert.Equal(t, float64(5), backgroundJobs["queueSize"])
	assert.Equal(t, "healthy", backgroundJobs["status"])

	mockLearningService.AssertExpectations(t)
}

func TestWorkerAdminHandler_GetPriorityAnalytics_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockWorkerService := createMockWorkerService()
	mockUserService := createMockUserService()
	mockQuestionService := createMockQuestionService()
	mockAIService := createMockAIService()
	mockConfig := &config.Config{}
	mockWorker := createMockWorker()
	mockLearningService := createMockLearningService()
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Mock service to return error
	mockLearningService.On("GetPriorityScoreDistribution", mock.Anything).Return(nil, assert.AnError)
	mockLearningService.On("GetHighPriorityQuestions", mock.Anything, 5).Return([]map[string]interface{}{}, nil)

	mockDailyQuestionService := createMockDailyQuestionService()
	handler := NewWorkerAdminHandlerWithLogger(mockUserService, mockQuestionService, mockAIService, mockConfig, mockWorker, mockWorkerService, mockLearningService, mockDailyQuestionService, logger)

	router := gin.New()
	router.GET("/v1/analytics/priority-scores", handler.GetPriorityAnalytics)

	req, _ := http.NewRequest("GET", "/v1/analytics/priority-scores", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Should return default values when service fails
	distribution := response["distribution"].(map[string]interface{})
	assert.Equal(t, float64(0), distribution["high"])
	assert.Equal(t, float64(0), distribution["medium"])
	assert.Equal(t, float64(0), distribution["low"])
	assert.Equal(t, 0.0, distribution["average"])

	mockLearningService.AssertExpectations(t)
}

// Helper functions to create mock services
func createMockWorkerService() *MockWorkerServiceForHandler {
	return &MockWorkerServiceForHandler{}
}

func createMockUserService() *MockUserServiceForHandler {
	return &MockUserServiceForHandler{}
}

func createMockQuestionService() *MockQuestionServiceForHandler {
	return &MockQuestionServiceForHandler{}
}

func createMockAIService() *MockAIServiceForHandler {
	return &MockAIServiceForHandler{}
}

func createMockLearningService() *MockLearningServiceForHandler {
	return &MockLearningServiceForHandler{}
}

func createMockDailyQuestionService() *MockDailyQuestionServiceForHandler {
	return &MockDailyQuestionServiceForHandler{}
}

func createMockWorker() *worker.Worker {
	return nil // For these tests, we can use nil since we're testing handler logic
}

// Mock services for handler tests
type MockWorkerServiceForHandler struct {
	mock.Mock
}

func (m *MockWorkerServiceForHandler) GetSetting(ctx context.Context, key string) (result0 string, err error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockWorkerServiceForHandler) SetSetting(ctx context.Context, key, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockWorkerServiceForHandler) IsGlobalPaused(ctx context.Context) (result0 bool, err error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

func (m *MockWorkerServiceForHandler) SetGlobalPause(ctx context.Context, paused bool) error {
	args := m.Called(ctx, paused)
	return args.Error(0)
}

func (m *MockWorkerServiceForHandler) IsUserPaused(ctx context.Context, userID int) (result0 bool, err error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockWorkerServiceForHandler) SetUserPause(ctx context.Context, userID int, paused bool) error {
	args := m.Called(ctx, userID, paused)
	return args.Error(0)
}

func (m *MockWorkerServiceForHandler) UpdateWorkerStatus(ctx context.Context, instance string, status *models.WorkerStatus) error {
	args := m.Called(ctx, instance, status)
	return args.Error(0)
}

func (m *MockWorkerServiceForHandler) GetWorkerStatus(ctx context.Context, instance string) (result0 *models.WorkerStatus, err error) {
	args := m.Called(ctx, instance)
	return args.Get(0).(*models.WorkerStatus), args.Error(1)
}

func (m *MockWorkerServiceForHandler) GetAllWorkerStatuses(ctx context.Context) (result0 []models.WorkerStatus, err error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.WorkerStatus), args.Error(1)
}

func (m *MockWorkerServiceForHandler) UpdateHeartbeat(ctx context.Context, instance string) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *MockWorkerServiceForHandler) IsWorkerHealthy(ctx context.Context, instance string) (result0 bool, err error) {
	args := m.Called(ctx, instance)
	return args.Bool(0), args.Error(1)
}

func (m *MockWorkerServiceForHandler) PauseWorker(ctx context.Context, instance string) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *MockWorkerServiceForHandler) ResumeWorker(ctx context.Context, instance string) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *MockWorkerServiceForHandler) GetWorkerHealth(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockWorkerServiceForHandler) GetHighPriorityTopics(ctx context.Context, userID int, language, level, questionType string) (result0 []string, err error) {
	args := m.Called(ctx, userID, language, level, questionType)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockWorkerServiceForHandler) GetGapAnalysis(ctx context.Context, userID int, language, level, questionType string) (result0 map[string]int, err error) {
	args := m.Called(ctx, userID, language, level, questionType)
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *MockWorkerServiceForHandler) GetPriorityDistribution(ctx context.Context, userID int, language, level, questionType string) (result0 map[string]int, err error) {
	args := m.Called(ctx, userID, language, level, questionType)
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *MockWorkerServiceForHandler) GetNotificationStats(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockWorkerServiceForHandler) GetNotificationErrors(ctx context.Context, page, pageSize int, errorType, notificationType, resolved string) (result0 []map[string]interface{}, result1, result2 map[string]interface{}, err error) {
	args := m.Called(ctx, page, pageSize, errorType, notificationType, resolved)
	return args.Get(0).([]map[string]interface{}), args.Get(1).(map[string]interface{}), args.Get(2).(map[string]interface{}), args.Error(3)
}

func (m *MockWorkerServiceForHandler) GetUpcomingNotifications(ctx context.Context, page, pageSize int, notificationType, status, scheduledAfter, scheduledBefore string) (result0 []map[string]interface{}, result1, result2 map[string]interface{}, err error) {
	args := m.Called(ctx, page, pageSize, notificationType, status, scheduledAfter, scheduledBefore)
	return args.Get(0).([]map[string]interface{}), args.Get(1).(map[string]interface{}), args.Get(2).(map[string]interface{}), args.Error(3)
}

func (m *MockWorkerServiceForHandler) GetSentNotifications(ctx context.Context, page, pageSize int, notificationType, status, sentAfter, sentBefore string) (result0 []map[string]interface{}, result1, result2 map[string]interface{}, err error) {
	args := m.Called(ctx, page, pageSize, notificationType, status, sentAfter, sentBefore)
	return args.Get(0).([]map[string]interface{}), args.Get(1).(map[string]interface{}), args.Get(2).(map[string]interface{}), args.Error(3)
}

func (m *MockWorkerServiceForHandler) CreateTestSentNotification(ctx context.Context, userID int, notificationType, subject, templateName, status, errorMessage string) error {
	args := m.Called(ctx, userID, notificationType, subject, templateName, status, errorMessage)
	return args.Error(0)
}

type MockUserServiceForHandler struct {
	mock.Mock
}

func (m *MockUserServiceForHandler) GetAllUsers(ctx context.Context) (result0 []models.User, err error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserServiceForHandler) CreateUser(ctx context.Context, username, language, level string) (result0 *models.User, err error) {
	args := m.Called(ctx, username, language, level)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserServiceForHandler) CreateUserWithPassword(ctx context.Context, username, password, language, level string) (result0 *models.User, err error) {
	args := m.Called(ctx, username, password, language, level)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserServiceForHandler) GetUserByID(ctx context.Context, id int) (result0 *models.User, err error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserServiceForHandler) GetUserByUsername(ctx context.Context, username string) (result0 *models.User, err error) {
	args := m.Called(ctx, username)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserServiceForHandler) AuthenticateUser(ctx context.Context, username, password string) (result0 *models.User, err error) {
	args := m.Called(ctx, username, password)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserServiceForHandler) UpdateUserSettings(ctx context.Context, userID int, settings *models.UserSettings) error {
	args := m.Called(ctx, userID, settings)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) UpdateLastActive(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) DeleteUser(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) DeleteAllUsers(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) EnsureAdminUserExists(ctx context.Context, username, password string) error {
	args := m.Called(ctx, username, password)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) ResetDatabase(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) UpdateWordOfDayEmailEnabled(ctx context.Context, userID int, enabled bool) error {
	args := m.Called(ctx, userID, enabled)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) ClearUserData(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) GetUserAPIKey(ctx context.Context, userID int, provider string) (result0 string, err error) {
	args := m.Called(ctx, userID, provider)
	return args.String(0), args.Error(1)
}

func (m *MockUserServiceForHandler) GetUserAPIKeyWithID(ctx context.Context, userID int, provider string) (string, *int, error) {
	args := m.Called(ctx, userID, provider)
	return args.String(0), args.Get(1).(*int), args.Error(2)
}

func (m *MockUserServiceForHandler) SetUserAPIKey(ctx context.Context, userID int, provider, apiKey string) error {
	args := m.Called(ctx, userID, provider, apiKey)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) HasUserAPIKey(ctx context.Context, userID int, provider string) (result0 bool, err error) {
	args := m.Called(ctx, userID, provider)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserServiceForHandler) CreateUserWithEmailAndTimezone(ctx context.Context, username, email, timezone, language, level string) (result0 *models.User, err error) {
	args := m.Called(ctx, username, email, timezone, language, level)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserServiceForHandler) GetUserByEmail(ctx context.Context, email string) (result0 *models.User, err error) {
	args := m.Called(ctx, email)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserServiceForHandler) UpdateUserProfile(ctx context.Context, userID int, username, email, timezone string) error {
	args := m.Called(ctx, userID, username, email, timezone)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) ResetPassword(ctx context.Context, userID int, newPassword string) error {
	args := m.Called(ctx, userID, newPassword)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) UpdateUserPassword(ctx context.Context, userID int, newPassword string) error {
	args := m.Called(ctx, userID, newPassword)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) ClearUserDataForUser(ctx context.Context, userID int) error {
	return nil
}

// Role management methods
func (m *MockUserServiceForHandler) GetUserRoles(ctx context.Context, userID int) (result0 []models.Role, err error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *MockUserServiceForHandler) AssignRole(ctx context.Context, userID, roleID int) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) RemoveRole(ctx context.Context, userID, roleID int) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) HasRole(ctx context.Context, userID int, roleName string) (result0 bool, err error) {
	args := m.Called(ctx, userID, roleName)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserServiceForHandler) AssignRoleByName(ctx context.Context, userID int, roleName string) error {
	args := m.Called(ctx, userID, roleName)
	return args.Error(0)
}

func (m *MockUserServiceForHandler) IsAdmin(ctx context.Context, userID int) (result0 bool, err error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserServiceForHandler) GetAllRoles(ctx context.Context) (result0 []models.Role, err error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *MockUserServiceForHandler) GetUsersPaginated(ctx context.Context, page, pageSize int, search, language, level, aiProvider, aiModel, aiEnabled, active string) (result0 []models.User, result1 int, err error) {
	args := m.Called(ctx, page, pageSize, search, language, level, aiProvider, aiModel, aiEnabled, active)
	return args.Get(0).([]models.User), args.Int(1), args.Error(2)
}

func (m *MockUserServiceForHandler) GetDB() *sql.DB {
	args := m.Called()
	return args.Get(0).(*sql.DB)
}

type MockQuestionServiceForHandler struct {
	mock.Mock
}

func (m *MockQuestionServiceForHandler) SaveQuestion(ctx context.Context, question *models.Question) error {
	args := m.Called(ctx, question)
	return args.Error(0)
}

func (m *MockQuestionServiceForHandler) GetQuestionByID(ctx context.Context, id int) (result0 *models.Question, err error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Question), args.Error(1)
}

func (m *MockQuestionServiceForHandler) GetQuestionsByFilter(ctx context.Context, userID int, language, level string, qType models.QuestionType, limit int) (result0 []models.Question, err error) {
	args := m.Called(ctx, userID, language, level, qType, limit)
	return args.Get(0).([]models.Question), args.Error(1)
}

func (m *MockQuestionServiceForHandler) IncrementUsageCount(ctx context.Context, questionID int) error {
	args := m.Called(ctx, questionID)
	return args.Error(0)
}

func (m *MockQuestionServiceForHandler) GetNextQuestion(ctx context.Context, userID int, language, level string, qType models.QuestionType) (result0 *services.QuestionWithStats, err error) {
	args := m.Called(ctx, userID, language, level, qType)
	return args.Get(0).(*services.QuestionWithStats), args.Error(1)
}

func (m *MockQuestionServiceForHandler) ReportQuestion(ctx context.Context, questionID, userID int, reportReason string) error {
	args := m.Called(ctx, questionID, userID, reportReason)
	return args.Error(0)
}

func (m *MockQuestionServiceForHandler) GetQuestionStats(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockQuestionServiceForHandler) GetDetailedQuestionStats(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockQuestionServiceForHandler) GetRecentQuestionContentsForUser(ctx context.Context, userID, limit int) (result0 []string, err error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockQuestionServiceForHandler) GetReportedQuestions(ctx context.Context) (result0 []*services.ReportedQuestionWithUser, err error) {
	args := m.Called(ctx)
	return args.Get(0).([]*services.ReportedQuestionWithUser), args.Error(1)
}

func (m *MockQuestionServiceForHandler) MarkQuestionAsFixed(ctx context.Context, questionID int) error {
	args := m.Called(ctx, questionID)
	return args.Error(0)
}

func (m *MockQuestionServiceForHandler) UpdateQuestion(ctx context.Context, questionID int, content map[string]interface{}, correctAnswerIndex int, explanation string) error {
	args := m.Called(ctx, questionID, content, correctAnswerIndex, explanation)
	return args.Error(0)
}

func (m *MockQuestionServiceForHandler) GetUserQuestions(ctx context.Context, userID, limit int) (result0 []*models.Question, err error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]*models.Question), args.Error(1)
}

func (m *MockQuestionServiceForHandler) DeleteQuestion(ctx context.Context, questionID int) error {
	args := m.Called(ctx, questionID)
	return args.Error(0)
}

func (m *MockQuestionServiceForHandler) GetQuestionWithStats(ctx context.Context, id int) (result0 *services.QuestionWithStats, err error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*services.QuestionWithStats), args.Error(1)
}

func (m *MockQuestionServiceForHandler) GetUserQuestionsWithStats(ctx context.Context, userID, limit int) (result0 []*services.QuestionWithStats, err error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]*services.QuestionWithStats), args.Error(1)
}

func (m *MockQuestionServiceForHandler) GetQuestionsPaginated(ctx context.Context, userID, page, pageSize int, search, typeFilter, statusFilter string) (result0 []*services.QuestionWithStats, result1 int, err error) {
	args := m.Called(ctx, userID, page, pageSize, search, typeFilter, statusFilter)
	return args.Get(0).([]*services.QuestionWithStats), args.Int(1), args.Error(2)
}

func (m *MockQuestionServiceForHandler) AssignQuestionToUser(ctx context.Context, questionID, userID int) error {
	args := m.Called(ctx, questionID, userID)
	return args.Error(0)
}

func (m *MockQuestionServiceForHandler) GetUserQuestionCount(ctx context.Context, userID int) (result0 int, err error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockQuestionServiceForHandler) GetUserResponseCount(ctx context.Context, userID int) (result0 int, err error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockQuestionServiceForHandler) GetRandomGlobalQuestionForUser(ctx context.Context, userID int, language, level string, questionType models.QuestionType) (result0 *services.QuestionWithStats, err error) {
	args := m.Called(ctx, userID, language, level, questionType)
	return args.Get(0).(*services.QuestionWithStats), args.Error(1)
}

func (m *MockQuestionServiceForHandler) AssignUsersToQuestion(ctx context.Context, questionID int, userIDs []int) error {
	args := m.Called(ctx, questionID, userIDs)
	return args.Error(0)
}

func (m *MockQuestionServiceForHandler) GetAllQuestionsPaginated(ctx context.Context, page, pageSize int, search, typeFilter, statusFilter, language, level string, userID *int) (result0 []*services.QuestionWithStats, result1 int, err error) {
	args := m.Called(ctx, page, pageSize, search, typeFilter, statusFilter, language, level, userID)
	return args.Get(0).([]*services.QuestionWithStats), args.Int(1), args.Error(2)
}

func (m *MockQuestionServiceForHandler) GetReportedQuestionsPaginated(ctx context.Context, page, pageSize int, search, statusFilter, language, level string) (result0 []*services.QuestionWithStats, result1 int, err error) {
	args := m.Called(ctx, page, pageSize, search, statusFilter, language, level)
	return args.Get(0).([]*services.QuestionWithStats), args.Int(1), args.Error(2)
}

func (m *MockQuestionServiceForHandler) GetReportedQuestionsStats(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockQuestionServiceForHandler) GetUsersForQuestion(ctx context.Context, questionID int) (result0 []*models.User, result1 int, err error) {
	args := m.Called(ctx, questionID)
	return args.Get(0).([]*models.User), args.Int(1), args.Error(2)
}

func (m *MockQuestionServiceForHandler) UnassignUsersFromQuestion(ctx context.Context, questionID int, userIDs []int) error {
	args := m.Called(ctx, questionID, userIDs)
	return args.Error(0)
}

func (m *MockQuestionServiceForHandler) GetAdaptiveQuestionsForDaily(ctx context.Context, userID int, language, level string, limit int) (result0 []*services.QuestionWithStats, err error) {
	args := m.Called(ctx, userID, language, level, limit)
	return args.Get(0).([]*services.QuestionWithStats), args.Error(1)
}

func (m *MockQuestionServiceForHandler) DB() *sql.DB {
	return nil
}

type MockLearningServiceForHandler struct {
	mock.Mock
}

func (m *MockLearningServiceForHandler) RecordUserResponse(ctx context.Context, response *models.UserResponse) error {
	args := m.Called(ctx, response)
	return args.Error(0)
}

func (m *MockLearningServiceForHandler) GetUserProgress(ctx context.Context, userID int) (result0 *models.UserProgress, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*models.UserProgress), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetWeakestTopics(ctx context.Context, userID, limit int) (result0 []*models.PerformanceMetrics, err error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]*models.PerformanceMetrics), args.Error(1)
}

func (m *MockLearningServiceForHandler) ShouldAvoidQuestion(ctx context.Context, userID, questionID int) (result0 bool, err error) {
	args := m.Called(ctx, userID, questionID)
	return args.Bool(0), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetUserQuestionStats(ctx context.Context, userID int) (result0 *services.UserQuestionStats, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*services.UserQuestionStats), args.Error(1)
}

func (m *MockLearningServiceForHandler) RecordAnswerWithPriority(ctx context.Context, userID, questionID, answerIndex int, isCorrect bool, responseTime int) error {
	args := m.Called(ctx, userID, questionID, answerIndex, isCorrect, responseTime)
	return args.Error(0)
}

func (m *MockLearningServiceForHandler) RecordAnswerWithPriorityReturningID(ctx context.Context, userID, questionID, answerIndex int, isCorrect bool, responseTime int) (int, error) {
	args := m.Called(ctx, userID, questionID, answerIndex, isCorrect, responseTime)
	if args.Get(0) == nil {
		return 0, args.Error(1)
	}
	return args.Int(0), args.Error(1)
}

func (m *MockLearningServiceForHandler) MarkQuestionAsKnown(ctx context.Context, userID, questionID int, confidenceLevel *int) error {
	args := m.Called(ctx, userID, questionID, confidenceLevel)
	return args.Error(0)
}

func (m *MockLearningServiceForHandler) GetUserLearningPreferences(ctx context.Context, userID int) (result0 *models.UserLearningPreferences, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*models.UserLearningPreferences), args.Error(1)
}

func (m *MockLearningServiceForHandler) CalculatePriorityScore(ctx context.Context, userID, questionID int) (result0 float64, err error) {
	args := m.Called(ctx, userID, questionID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockLearningServiceForHandler) UpdateUserLearningPreferences(ctx context.Context, userID int, prefs *models.UserLearningPreferences) (result0 *models.UserLearningPreferences, err error) {
	args := m.Called(ctx, userID, prefs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserLearningPreferences), args.Error(1)
}

func (m *MockLearningServiceForHandler) UpdateLastDailyReminderSent(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// Analytics methods
func (m *MockLearningServiceForHandler) GetPriorityScoreDistribution(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	result := args.Get(0)
	if result == nil {
		return map[string]interface{}{}, args.Error(1)
	}
	return result.(map[string]interface{}), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetHighPriorityQuestions(ctx context.Context, limit int) (result0 []map[string]interface{}, err error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetWeakAreasByTopic(ctx context.Context, limit int) (result0 []map[string]interface{}, err error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetLearningPreferencesUsage(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetQuestionTypeGaps(ctx context.Context) (result0 []map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetGenerationSuggestions(ctx context.Context) (result0 []map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetPrioritySystemPerformance(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetBackgroundJobsStatus(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetUserWeakAreas(ctx context.Context, userID, limit int) (result0 []map[string]interface{}, err error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetUserHighPriorityQuestions(ctx context.Context, userID, limit int) (result0 []map[string]interface{}, err error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetUserPriorityScoreDistribution(ctx context.Context, userID int) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetHighPriorityTopics(ctx context.Context, userID int) (result0 []string, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetGapAnalysis(ctx context.Context, userID int) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetPriorityDistribution(ctx context.Context, userID int) (result0 map[string]int, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *MockLearningServiceForHandler) GetUserQuestionConfidenceLevel(ctx context.Context, userID, questionID int) (*int, error) {
	args := m.Called(ctx, userID, questionID)
	return args.Get(0).(*int), args.Error(1)
}

// Mock AI service for handler tests
type MockAIServiceForHandler struct {
	mock.Mock
}

func (m *MockAIServiceForHandler) GenerateQuestion(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest) (result0 *models.Question, err error) {
	args := m.Called(ctx, userConfig, req)
	return args.Get(0).(*models.Question), args.Error(1)
}

func (m *MockAIServiceForHandler) GenerateQuestions(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest) (result0 []*models.Question, err error) {
	args := m.Called(ctx, userConfig, req)
	return args.Get(0).([]*models.Question), args.Error(1)
}

func (m *MockAIServiceForHandler) GenerateQuestionsStream(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest, progress chan<- *models.Question, variety *services.VarietyElements) error {
	args := m.Called(ctx, userConfig, req, progress, variety)
	return args.Error(0)
}

func (m *MockAIServiceForHandler) GenerateChatResponse(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIChatRequest) (result0 string, err error) {
	args := m.Called(ctx, userConfig, req)
	return args.String(0), args.Error(1)
}

func (m *MockAIServiceForHandler) GenerateChatResponseStream(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIChatRequest, chunks chan<- string) error {
	args := m.Called(ctx, userConfig, req, chunks)
	return args.Error(0)
}

func (m *MockAIServiceForHandler) GenerateStoryQuestions(ctx context.Context, userConfig *models.UserAIConfig, req *models.StoryQuestionsRequest) ([]*models.StorySectionQuestionData, error) {
	args := m.Called(ctx, userConfig, req)
	return args.Get(0).([]*models.StorySectionQuestionData), args.Error(1)
}

func (m *MockAIServiceForHandler) GenerateStorySection(ctx context.Context, userConfig *models.UserAIConfig, req *models.StoryGenerationRequest) (string, error) {
	args := m.Called(ctx, userConfig, req)
	return args.String(0), args.Error(1)
}

func (m *MockAIServiceForHandler) TestConnection(ctx context.Context, provider, model, apiKey string) error {
	args := m.Called(ctx, provider, model, apiKey)
	return args.Error(0)
}

func (m *MockAIServiceForHandler) GetConcurrencyStats() services.ConcurrencyStats {
	args := m.Called()
	return args.Get(0).(services.ConcurrencyStats)
}

func (m *MockAIServiceForHandler) GetQuestionBatchSize(provider string) int {
	args := m.Called(provider)
	return args.Int(0)
}

func (m *MockAIServiceForHandler) VarietyService() *services.VarietyService {
	args := m.Called()
	return args.Get(0).(*services.VarietyService)
}

func (m *MockAIServiceForHandler) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAIServiceForHandler) TemplateManager() *services.AITemplateManager {
	args := m.Called()
	return args.Get(0).(*services.AITemplateManager)
}

func (m *MockAIServiceForHandler) SupportsGrammarField(provider string) bool {
	args := m.Called(provider)
	return args.Bool(0)
}

func (m *MockAIServiceForHandler) CallWithPrompt(ctx context.Context, userConfig *models.UserAIConfig, prompt, grammar string) (string, error) {
	args := m.Called(ctx, userConfig, prompt, grammar)
	return args.String(0), args.Error(1)
}

type MockDailyQuestionServiceForHandler struct {
	mock.Mock
}

func (m *MockDailyQuestionServiceForHandler) AssignDailyQuestions(ctx context.Context, userID int, date time.Time) error {
	args := m.Called(ctx, userID, date)
	return args.Error(0)
}

func (m *MockDailyQuestionServiceForHandler) GetDailyQuestions(ctx context.Context, userID int, date time.Time) ([]*models.DailyQuestionAssignmentWithQuestion, error) {
	args := m.Called(ctx, userID, date)
	return args.Get(0).([]*models.DailyQuestionAssignmentWithQuestion), args.Error(1)
}

func (m *MockDailyQuestionServiceForHandler) MarkQuestionCompleted(ctx context.Context, userID, questionID int, date time.Time) error {
	args := m.Called(ctx, userID, questionID, date)
	return args.Error(0)
}

func (m *MockDailyQuestionServiceForHandler) ResetQuestionCompleted(ctx context.Context, userID, questionID int, date time.Time) error {
	args := m.Called(ctx, userID, questionID, date)
	return args.Error(0)
}

func (m *MockDailyQuestionServiceForHandler) SubmitDailyQuestionAnswer(ctx context.Context, userID, questionID int, date time.Time, userAnswerIndex int) (*api.AnswerResponse, error) {
	args := m.Called(ctx, userID, questionID, date, userAnswerIndex)
	return args.Get(0).(*api.AnswerResponse), args.Error(1)
}

func (m *MockDailyQuestionServiceForHandler) GetAvailableDates(ctx context.Context, userID int) ([]time.Time, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]time.Time), args.Error(1)
}

func (m *MockDailyQuestionServiceForHandler) GetDailyProgress(ctx context.Context, userID int, date time.Time) (*models.DailyProgress, error) {
	args := m.Called(ctx, userID, date)
	return args.Get(0).(*models.DailyProgress), args.Error(1)
}

func (m *MockDailyQuestionServiceForHandler) GetDailyQuestionsCount(ctx context.Context, userID int, date time.Time) (int, error) {
	args := m.Called(ctx, userID, date)
	return args.Int(0), args.Error(1)
}

func (m *MockDailyQuestionServiceForHandler) GetCompletedDailyQuestionsCount(ctx context.Context, userID int, date time.Time) (int, error) {
	args := m.Called(ctx, userID, date)
	return args.Int(0), args.Error(1)
}

func (m *MockDailyQuestionServiceForHandler) RegenerateDailyQuestions(ctx context.Context, userID int, date time.Time) error {
	args := m.Called(ctx, userID, date)
	return args.Error(0)
}

func (m *MockDailyQuestionServiceForHandler) GetQuestionHistory(ctx context.Context, userID, questionID, days int) ([]*models.DailyQuestionHistory, error) {
	args := m.Called(ctx, userID, questionID, days)
	return args.Get(0).([]*models.DailyQuestionHistory), args.Error(1)
}

// mockResponseWriter implements http.ResponseWriter for testing
type mockResponseWriter struct {
	*bytes.Buffer
}

func (m *mockResponseWriter) Header() http.Header {
	return make(http.Header)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	// No-op for testing
}
