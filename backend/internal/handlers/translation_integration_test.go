//go:build integration
// +build integration

package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TranslationIntegrationTestSuite tests the translation handler with real database interactions
type TranslationIntegrationTestSuite struct {
	suite.Suite
	Router          *gin.Engine
	UserService     *services.UserService
	LearningService *services.LearningService
	Config          *config.Config
	TestUserID      int
	DB              *sql.DB
}

func (suite *TranslationIntegrationTestSuite) SetupSuite() {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}
	// Load config
	cfg, err := config.NewConfig()
	cfg.Database.URL = databaseURL
	require.NoError(suite.T(), err)
	suite.Config = cfg

	// Use shared test database setup
	suite.DB = services.SharedTestDBSetup(suite.T())

	// Create services
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.DB, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(suite.DB, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.DB, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(suite.DB, logger)
	dailyQuestionService := services.NewDailyQuestionService(suite.DB, logger, questionService, learningService)
	storyService := services.NewStoryService(suite.DB, cfg, logger)
	oauthService := services.NewOAuthServiceWithLogger(cfg, logger)
	generationHintService := services.NewGenerationHintService(suite.DB, logger)
	usageStatsService := services.NewUsageStatsService(cfg, suite.DB, logger)
	translationCacheRepo := services.NewTranslationCacheRepository(suite.DB, logger)
	translationService := services.NewTranslationService(cfg, usageStatsService, translationCacheRepo, logger)
	snippetsService := services.NewSnippetsService(suite.DB, cfg, logger)
    authAPIKeyService := services.NewAuthAPIKeyService(suite.DB, logger)

	suite.UserService = userService
	suite.LearningService = learningService

	// Create test user
	createdUser, err := userService.CreateUserWithPassword(context.Background(), "testuser_translate", "testpass", "english", "A1")
	require.NoError(suite.T(), err)
	suite.TestUserID = createdUser.ID

	// Use the real application router
	suite.Router = NewRouter(
		cfg,
		userService,
		questionService,
		learningService,
		aiService,
		workerService,
		dailyQuestionService,
		storyService,
		services.NewConversationService(suite.DB),
		oauthService,
		generationHintService,
		translationService,
		snippetsService,
		usageStatsService,
		services.NewWordOfTheDayService(suite.DB, logger),
		authAPIKeyService,
		logger,
	)
}

func (suite *TranslationIntegrationTestSuite) TearDownSuite() {
	if suite.DB != nil {
		suite.DB.Close()
	}
}

func (suite *TranslationIntegrationTestSuite) SetupTest() {
	// Database cleanup is handled by SharedTestDBSetup, no need for separate cleanup
}

func (suite *TranslationIntegrationTestSuite) TestTranslationQuotaExceeded() {
	// This test verifies that when monthly usage exceeds the quota, translation requests are rejected

	// First, record some usage to get close to the quota limit
	// Set a low quota for testing (100 characters)
	suite.Config.Translation.Quota.GoogleMonthlyQuota = 100

	// Record 90 characters of usage (leaving 10 characters remaining)
	err := suite.recordUsage("google", "translation", 90, 1)
	require.NoError(suite.T(), err)

	// Now try to translate text that would use 20 characters (exceeding the remaining 10)
	translateReq := api.TranslateRequest{
		Text:           "This is a test translation that should exceed the quota limit when combined with existing usage.",
		TargetLanguage: "es",
		SourceLanguage: stringPtr("en"),
	}

	// Create authenticated request
	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(translateReq)
	req := httptest.NewRequest("POST", "/v1/translate", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Add authentication cookie (this would normally be set by middleware)
	ctx := context.WithValue(req.Context(), "user_id", suite.TestUserID)
	req = req.WithContext(ctx)

	// Perform the request
	suite.Router.ServeHTTP(w, req)

	// Should return 503 Service Unavailable due to quota exceeded
	assert.Equal(suite.T(), http.StatusServiceUnavailable, w.Code)

	var response api.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	// Verify the error message mentions quota exceeded
	assert.Contains(suite.T(), *response.Message, "quota exceeded")
	assert.Contains(suite.T(), *response.Message, "google")
}

func (suite *TranslationIntegrationTestSuite) TestTranslationWithinQuota() {
	// This test verifies that translation works when within quota limits

	// Set a reasonable quota for testing (1000 characters)
	suite.Config.Translation.Quota.GoogleMonthlyQuota = 1000

	// Record minimal usage (10 characters)
	err := suite.recordUsage("google", "translation", 10, 1)
	require.NoError(suite.T(), err)

	// Try to translate text that uses 50 characters (well within the remaining 990)
	translateReq := api.TranslateRequest{
		Text:           "Short translation test.",
		TargetLanguage: "es",
		SourceLanguage: stringPtr("en"),
	}

	// Create authenticated request
	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(translateReq)
	req := httptest.NewRequest("POST", "/v1/translate", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Add authentication cookie (this would normally be set by middleware)
	ctx := context.WithValue(req.Context(), "user_id", suite.TestUserID)
	req = req.WithContext(ctx)

	// Perform the request
	suite.Router.ServeHTTP(w, req)

	// Should succeed (but will fail due to no actual Google API key in test environment)
	// The important thing is that it doesn't fail due to quota exceeded
	assert.NotEqual(suite.T(), http.StatusServiceUnavailable, w.Code)
}

func (suite *TranslationIntegrationTestSuite) TestUsageRecording() {
	// This test verifies that usage is properly recorded after successful translations

	// Set a reasonable quota
	suite.Config.Translation.Quota.GoogleMonthlyQuota = 1000

	// Get initial usage count
	initialUsage, err := suite.getCurrentUsage("google", "translation")
	require.NoError(suite.T(), err)

	// Record some usage manually
	err = suite.recordUsage("google", "translation", 50, 2)
	require.NoError(suite.T(), err)

	// Verify usage was recorded
	finalUsage, err := suite.getCurrentUsage("google", "translation")
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), initialUsage.CharactersUsed+50, finalUsage.CharactersUsed)
	assert.Equal(suite.T(), initialUsage.RequestsMade+2, finalUsage.RequestsMade)
}

// Helper method to record usage for testing
func (suite *TranslationIntegrationTestSuite) recordUsage(serviceName, usageType string, characters, requests int) error {
	usageStatsService := services.NewUsageStatsService(suite.Config, suite.DB, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	return usageStatsService.RecordUsage(context.Background(), serviceName, usageType, characters, requests)
}

// Helper method to get current usage for testing
func (suite *TranslationIntegrationTestSuite) getCurrentUsage(serviceName, usageType string) (*services.UsageStats, error) {
	usageStatsService := services.NewUsageStatsService(suite.Config, suite.DB, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	return usageStatsService.GetCurrentMonthUsage(context.Background(), serviceName, usageType)
}

// Note: stringPtr function is defined in convert.go
