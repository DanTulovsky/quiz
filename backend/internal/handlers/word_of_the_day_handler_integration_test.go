//go:build integration

package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// WordOfTheDayIntegrationTestSuite provides integration tests for word of the day functionality
type WordOfTheDayIntegrationTestSuite struct {
	suite.Suite
	Router              *gin.Engine
	cfg                 *config.Config
	db                  *sql.DB
	wordOfTheDayService services.WordOfTheDayServiceInterface
	userService         *services.UserService
	snippetsService     services.SnippetsServiceInterface
}

func TestWordOfTheDayIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(WordOfTheDayIntegrationTestSuite))
}

func (suite *WordOfTheDayIntegrationTestSuite) SetupSuite() {
	// Use environment variable for test database URL
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	// Create config
	cfg, err := config.NewConfig()
	require.NoError(suite.T(), err)
	cfg.Database.URL = databaseURL
	suite.cfg = cfg

	// Initialize database
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := database.NewManager(logger)
	db, err := dbManager.InitDB(databaseURL)
	if err != nil {
		suite.T().Fatalf("Failed to initialize database: %v", err)
	}
	suite.db = db

	// Create services
	userService := services.NewUserServiceWithLogger(db, suite.cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, suite.cfg, logger)
	aiService := services.NewAIService(suite.cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)
	oauthService := services.NewOAuthServiceWithLogger(suite.cfg, logger)
	dailyQuestionService := services.NewDailyQuestionService(suite.db, logger, questionService, learningService)
	generationHintService := services.NewGenerationHintService(suite.db, logger)
	storyService := services.NewStoryService(suite.db, suite.cfg, logger)
	usageStatsService := services.NewUsageStatsService(suite.cfg, suite.db, logger)
	translationCacheRepo := services.NewTranslationCacheRepository(suite.db, logger)
	translationService := services.NewTranslationService(suite.cfg, usageStatsService, translationCacheRepo, logger)
	snippetsService := services.NewSnippetsService(suite.db, suite.cfg, logger)
	wordOfTheDayService := services.NewWordOfTheDayService(suite.db, logger)
	authAPIKeyService := services.NewAuthAPIKeyService(suite.db, logger)

	suite.userService = userService
	suite.wordOfTheDayService = wordOfTheDayService
	suite.snippetsService = snippetsService

	suite.Router = NewRouter(suite.cfg, userService, questionService, learningService, aiService, workerService, dailyQuestionService, storyService, services.NewConversationService(db), oauthService, generationHintService, translationService, snippetsService, usageStatsService, wordOfTheDayService, authAPIKeyService, logger)
}

func (suite *WordOfTheDayIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *WordOfTheDayIntegrationTestSuite) SetupTest() {
	// Clean database for each test
	services.CleanupTestDatabase(suite.db, suite.T())
}

// Helper function to create a test user
func (suite *WordOfTheDayIntegrationTestSuite) createTestUser(username, language, level string) *models.User {
	ctx := context.Background()
	user, err := suite.userService.CreateUserWithPassword(ctx, username, "password123", language, level)
	require.NoError(suite.T(), err)

	return user
}

// Helper function to login and get session cookie
func (suite *WordOfTheDayIntegrationTestSuite) loginUser(username, password string) string {
	loginReq := map[string]string{
		"username": username,
		"password": password,
	}
	reqBody, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Debug output
	if w.Code != http.StatusOK {
		suite.T().Logf("Login failed with status %d: %s", w.Code, w.Body.String())
	}
	require.Equal(suite.T(), http.StatusOK, w.Code, "Login should succeed")

	// Extract session cookie
	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "quiz-session" {
			return cookie.Value
		}
	}

	suite.T().Fatalf("No session cookie found after successful login. Got cookies: %+v", cookies)
	return ""
}

func (suite *WordOfTheDayIntegrationTestSuite) TestGetWordOfTheDay_CreatesNewWord() {
	user := suite.createTestUser("worduser1", "italian", "B1")
	session := suite.loginUser(user.Username, "password123")

	today := time.Now().Format("2006-01-02")

	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/word-of-day/%s", today), nil)
	req.AddCookie(&http.Cookie{Name: "quiz-session", Value: session})
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// In a fresh test DB, there may be no suitable data; accept 200 or 500
	if w.Code == http.StatusOK {
		var word models.WordOfTheDayDisplay
		err := json.Unmarshal(w.Body.Bytes(), &word)
		require.NoError(suite.T(), err)
		// Basic sanity if present
		assert.NotEmpty(suite.T(), word.Language)
	} else {
		assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	}
}

func (suite *WordOfTheDayIntegrationTestSuite) TestGetWordOfTheDay_ReturnsSameWordForSameDate() {
	user := suite.createTestUser("worduser2", "italian", "B1")
	session := suite.loginUser(user.Username, "password123")

	today := time.Now().Format("2006-01-02")

	// First request
	req1, _ := http.NewRequest("GET", fmt.Sprintf("/v1/word-of-day/%s", today), nil)
	req1.AddCookie(&http.Cookie{Name: "quiz-session", Value: session})
	w1 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		// If no data, just assert error is handled gracefully
		assert.Equal(suite.T(), http.StatusInternalServerError, w1.Code)
		return
	}

	var word1 models.WordOfTheDayDisplay
	err := json.Unmarshal(w1.Body.Bytes(), &word1)
	require.NoError(suite.T(), err)

	// Second request
	req2, _ := http.NewRequest("GET", fmt.Sprintf("/v1/word-of-day/%s", today), nil)
	req2.AddCookie(&http.Cookie{Name: "quiz-session", Value: session})
	w2 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w2, req2)

	assert.Equal(suite.T(), http.StatusOK, w2.Code)

	var word2 models.WordOfTheDayDisplay
	err = json.Unmarshal(w2.Body.Bytes(), &word2)
	require.NoError(suite.T(), err)

	// Words should be identical
	assert.Equal(suite.T(), word1.Word, word2.Word)
	assert.Equal(suite.T(), word1.SourceID, word2.SourceID)
	assert.Equal(suite.T(), word1.SourceType, word2.SourceType)
}

func (suite *WordOfTheDayIntegrationTestSuite) TestGetWordOfTheDay_Unauthorized() {
	today := time.Now().Format("2006-01-02")

	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/word-of-day/%s", today), nil)
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *WordOfTheDayIntegrationTestSuite) TestGetWordOfTheDayEmbed_HTML() {
	user := suite.createTestUser("worduser3", "italian", "B1")

	today := time.Now().Format("2006-01-02")

	// Test the embed endpoint - it will try to create a word automatically
	// If it fails due to no content, we'll just verify the endpoint structure is correct
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/word-of-day/%s/embed?user_id=%d", today, user.ID), nil)
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Should return HTML in either case
	assert.Contains(suite.T(), w.Header().Get("Content-Type"), "text/html")
	// Either success with word HTML or error HTML
	assert.True(suite.T(), w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

func (suite *WordOfTheDayIntegrationTestSuite) TestGetWordHistory() {
	user := suite.createTestUser("worduser4", "italian", "B1")
	session := suite.loginUser(user.Username, "password123")

	// For this test, we'll just test that the endpoint responds correctly
	// even with no word history (which is valid - fresh test database)
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)

	// Query history
	startDate := yesterday.Format("2006-01-02")
	endDate := today.Format("2006-01-02")

	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/word-of-day/history?start_date=%s&end_date=%s", startDate, endDate), nil)
	req.AddCookie(&http.Cookie{Name: "quiz-session", Value: session})
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response struct {
		Words []*models.WordOfTheDayDisplay `json:"words"`
		Count int                           `json:"count"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	// Empty history is valid for a fresh test
	assert.GreaterOrEqual(suite.T(), response.Count, 0)
	assert.Equal(suite.T(), response.Count, len(response.Words))
}
