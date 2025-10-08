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
	"testing"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SettingsIntegrationTestSuite tests the settings handler with real database interactions
type SettingsIntegrationTestSuite struct {
	suite.Suite
	Router          *gin.Engine
	UserService     *services.UserService
	LearningService *services.LearningService
	Config          *config.Config
	TestUserID      int
	DB              *sql.DB // Add DB field
}

func (suite *SettingsIntegrationTestSuite) SetupSuite() {
	// Use shared test database setup
	db := services.SharedTestDBSetup(suite.T())

	// Load config
	cfg, err := config.NewConfig()
	require.NoError(suite.T(), err)
	suite.Config = cfg

	// Create services
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger)
	workerService := services.NewWorkerServiceWithLogger(db, logger)
	oauthService := services.NewOAuthServiceWithLogger(cfg, logger)

	// Create test user
	createdUser, err := userService.CreateUserWithPassword(context.Background(), "testuser_settings", "testpass", "english", "A1")
	suite.Require().NoError(err)
	suite.TestUserID = createdUser.ID

	// Use the real application router
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)
	generationHintService := services.NewGenerationHintService(suite.DB, logger)
	storyService := services.NewStoryService(db, cfg, logger)
	suite.Router = NewRouter(
		cfg,
		userService,
		questionService,
		learningService,
		aiService,
		workerService,
		dailyQuestionService,
		storyService,
		oauthService,
		generationHintService,
		logger,
	)

	suite.UserService = userService
	suite.LearningService = learningService
	suite.DB = db // Store DB in suite
}

func (suite *SettingsIntegrationTestSuite) TearDownSuite() {
	// Cleanup test data
	suite.UserService.DeleteUser(context.Background(), suite.TestUserID)
	// Close database connection
	if suite.DB != nil {
		suite.DB.Close()
	}
}

func (suite *SettingsIntegrationTestSuite) SetupTest() {
	// Clean database before each test
	suite.cleanupDatabase()
}

func (suite *SettingsIntegrationTestSuite) cleanupDatabase() {
	// Use the shared database cleanup function
	services.CleanupTestDatabase(suite.DB, suite.T())

	// Recreate test user
	createdUser, err := suite.UserService.CreateUserWithPassword(context.Background(), "testuser_settings", "testpass", "english", "A1")
	suite.Require().NoError(err)
	suite.TestUserID = createdUser.ID
}

func (suite *SettingsIntegrationTestSuite) login() string {
	loginReq := api.LoginRequest{
		Username: "testuser_settings",
		Password: "testpass",
	}
	reqBody, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Extract cookie from response
	cookies := w.Result().Cookies()
	var sessionCookie string
	for _, cookie := range cookies {
		if cookie.Name == config.SessionName {
			sessionCookie = cookie.String()
			break
		}
	}
	return sessionCookie
}

func (suite *SettingsIntegrationTestSuite) TestUpdateUserSettings_Success() {
	sessionCookie := suite.login()

	settings := api.UserSettings{
		Language:   langPtr("french"),
		Level:      levelPtr("B1"),
		AiProvider: stringPtr("test"),
		AiModel:    stringPtr("test-model"),
		ApiKey:     stringPtr("test-api-key"),
		AiEnabled:  boolPtr(true),
	}

	body, _ := json.Marshal(settings)
	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response api.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	// Verify settings were saved
	user, err := suite.UserService.GetUserByID(context.Background(), suite.TestUserID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "french", user.PreferredLanguage.String)
	assert.Equal(suite.T(), "B1", user.CurrentLevel.String)
	assert.Equal(suite.T(), "test", user.AIProvider.String)
	assert.Equal(suite.T(), "test-model", user.AIModel.String)
	assert.True(suite.T(), user.AIEnabled.Bool)
}

func (suite *SettingsIntegrationTestSuite) TestUpdateUserSettings_InvalidLevel() {
	sessionCookie := suite.login()

	settings := api.UserSettings{
		Language: langPtr("english"),
		Level:    levelPtr("INVALID_LEVEL"),
	}

	body, _ := json.Marshal(settings)
	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "INVALID_FORMAT", response["code"])
}

func (suite *SettingsIntegrationTestSuite) TestUpdateUserSettings_Unauthorized() {
	settings := api.UserSettings{
		Language: langPtr("english"),
		Level:    levelPtr("A1"),
	}

	body, _ := json.Marshal(settings)
	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *SettingsIntegrationTestSuite) TestUpdateUserSettings_InvalidJSON() {
	sessionCookie := suite.login()

	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBufferString("invalid json"))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *SettingsIntegrationTestSuite) TestUpdateUserSettings_EmptyPayload() {
	sessionCookie := suite.login()

	// Send empty JSON object
	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBufferString("{}"))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "INVALID_INPUT", response["code"])
}

func (suite *SettingsIntegrationTestSuite) TestUpdateUserSettings_PartialUpdate() {
	sessionCookie := suite.login()

	// Update only language
	settings := api.UserSettings{
		Language: langPtr("french"),
	}

	body, _ := json.Marshal(settings)
	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Print response for debugging
	if w.Code != http.StatusOK {
		suite.T().Logf("Response body: %s", w.Body.String())
	}

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify only language was updated
	user, err := suite.UserService.GetUserByID(context.Background(), suite.TestUserID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "french", user.PreferredLanguage.String)
	// Other fields should remain unchanged (user starts with A1)
	assert.Equal(suite.T(), "A1", user.CurrentLevel.String)
}

func (suite *SettingsIntegrationTestSuite) TestGetProviders_Success() {
	sessionCookie := suite.login()

	req, _ := http.NewRequest("GET", "/v1/settings/ai-providers", nil)
	req.Header.Set("Cookie", sessionCookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	providers, ok := response["providers"].([]interface{})
	assert.True(suite.T(), ok)
	assert.NotEmpty(suite.T(), providers)

	levels, ok := response["levels"].([]interface{})
	assert.True(suite.T(), ok)
	assert.NotEmpty(suite.T(), levels)

	languages, ok := response["languages"].([]interface{})
	assert.True(suite.T(), ok)
	assert.NotEmpty(suite.T(), languages)
}

func (suite *SettingsIntegrationTestSuite) TestGetLevels_AllLevels() {
	req, _ := http.NewRequest("GET", "/v1/settings/levels", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	levels, ok := response["levels"].([]interface{})
	assert.True(suite.T(), ok)
	assert.NotEmpty(suite.T(), levels)

	descriptions, ok := response["level_descriptions"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.NotEmpty(suite.T(), descriptions)
}

func (suite *SettingsIntegrationTestSuite) TestGetLevels_ForLanguage() {
	req, _ := http.NewRequest("GET", "/v1/settings/levels?language=italian", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	levels, ok := response["levels"].([]interface{})
	assert.True(suite.T(), ok)
	assert.NotEmpty(suite.T(), levels)

	descriptions, ok := response["level_descriptions"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.NotEmpty(suite.T(), descriptions)
}

// Verify all configured languages return expected levels
func (suite *SettingsIntegrationTestSuite) TestGetLevels_AllLanguagesMatchConfig() {
	languages := suite.Config.GetLanguages()
	require.NotEmpty(suite.T(), languages)

	for _, lang := range languages {
		expected := suite.Config.GetLevelsForLanguage(lang)
		req, _ := http.NewRequest("GET", "/v1/settings/levels?language="+lang, nil)
		w := httptest.NewRecorder()
		suite.Router.ServeHTTP(w, req)
		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)

		levels, ok := response["levels"].([]interface{})
		assert.True(suite.T(), ok)
		got := make([]string, len(levels))
		for i, v := range levels {
			got[i] = v.(string)
		}
		assert.ElementsMatch(suite.T(), expected, got)
	}
}

func (suite *SettingsIntegrationTestSuite) TestGetLanguages_Success() {
	req, _ := http.NewRequest("GET", "/v1/settings/languages", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var languages []string
	err := json.Unmarshal(w.Body.Bytes(), &languages)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), languages)
}

func (suite *SettingsIntegrationTestSuite) TestClearAllStoriesEndpoint() {
	sessionCookie := suite.login()

	// Create a story for the test user
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	storyService := services.NewStoryService(suite.DB, suite.Config, logger)

	req := models.CreateStoryRequest{
		Title: "ClearStories Test",
	}
	story, err := storyService.CreateStory(context.Background(), uint(suite.TestUserID), "english", &req)
	suite.Require().NoError(err)

	// Add a section and a question to ensure related rows exist
	section, err := storyService.CreateSection(context.Background(), story.ID, "Section content", "A1", 10)
	suite.Require().NoError(err)
	questions := []models.StorySectionQuestionData{{QuestionText: "Q1", Options: []string{"a", "b"}, CorrectAnswerIndex: 0, Explanation: stringPtr("")}}
	err = storyService.CreateSectionQuestions(context.Background(), section.ID, questions)
	suite.Require().NoError(err)

	// Verify story exists
	var cnt int
	err = suite.DB.QueryRow("SELECT COUNT(*) FROM stories WHERE user_id = $1", suite.TestUserID).Scan(&cnt)
	suite.Require().NoError(err)
	suite.Require().Equal(1, cnt)

	// Call clear-stories endpoint
	reqHTTP, _ := http.NewRequest("POST", "/v1/settings/clear-stories", nil)
	reqHTTP.Header.Set("Cookie", sessionCookie)
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, reqHTTP)

	suite.Require().Equal(http.StatusOK, w.Code)

	// Verify stories deleted
	err = suite.DB.QueryRow("SELECT COUNT(*) FROM stories WHERE user_id = $1", suite.TestUserID).Scan(&cnt)
	suite.Require().NoError(err)
	suite.Require().Equal(0, cnt)
}

func (suite *SettingsIntegrationTestSuite) TestResetAccountEndpoint() {
	sessionCookie := suite.login()

	// Create a story for the test user
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	storyService := services.NewStoryService(suite.DB, suite.Config, logger)

	req := models.CreateStoryRequest{Title: "ResetAccount Test"}
	_, err := storyService.CreateStory(context.Background(), uint(suite.TestUserID), "english", &req)
	suite.Require().NoError(err)

	// Verify story exists
	var cnt int
	err = suite.DB.QueryRow("SELECT COUNT(*) FROM stories WHERE user_id = $1", suite.TestUserID).Scan(&cnt)
	suite.Require().NoError(err)
	suite.Require().Equal(1, cnt)

	// Call reset-account endpoint
	reqHTTP, _ := http.NewRequest("POST", "/v1/settings/reset-account", nil)
	reqHTTP.Header.Set("Cookie", sessionCookie)
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, reqHTTP)

	suite.Require().Equal(http.StatusOK, w.Code)

	// Verify stories deleted and user data cleared (at least stories)
	err = suite.DB.QueryRow("SELECT COUNT(*) FROM stories WHERE user_id = $1", suite.TestUserID).Scan(&cnt)
	suite.Require().NoError(err)
	suite.Require().Equal(0, cnt)
}

func (suite *SettingsIntegrationTestSuite) TestCheckAPIKeyAvailability_NoKey() {
	sessionCookie := suite.login()

	req, _ := http.NewRequest("GET", "/v1/settings/api-key/test-provider", nil)
	req.Header.Set("Cookie", sessionCookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["has_api_key"].(bool))
}

func (suite *SettingsIntegrationTestSuite) TestCheckAPIKeyAvailability_WithKey() {
	sessionCookie := suite.login()

	// First, save an API key
	err := suite.UserService.SetUserAPIKey(context.Background(), suite.TestUserID, "test-provider", "test-api-key")
	suite.Require().NoError(err)

	req, _ := http.NewRequest("GET", "/v1/settings/api-key/test-provider", nil)
	req.Header.Set("Cookie", sessionCookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["has_api_key"].(bool))
}

func (suite *SettingsIntegrationTestSuite) TestCheckAPIKeyAvailability_Unauthorized() {
	req, _ := http.NewRequest("GET", "/v1/settings/api-key/test-provider", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *SettingsIntegrationTestSuite) TestGetLearningPreferences_Success() {
	sessionCookie := suite.login()

	req, _ := http.NewRequest("GET", "/v1/preferences/learning", nil)
	req.Header.Set("Cookie", sessionCookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.UserLearningPreferences
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
}

func (suite *SettingsIntegrationTestSuite) TestGetLearningPreferences_Unauthorized() {
	req, _ := http.NewRequest("GET", "/v1/preferences/learning", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *SettingsIntegrationTestSuite) TestUpdateLearningPreferences_Success() {
	sessionCookie := suite.login()

	preferences := map[string]interface{}{
		"focus_on_weak_areas":    true,
		"fresh_question_ratio":   0.3,
		"known_question_penalty": 0.5,
		"review_interval_days":   7,
		"weak_area_boost":        2.0,
		"daily_reminder_enabled": false,
		"tts_voice":              "it-IT-IsabellaNeural",
	}

	body, _ := json.Marshal(preferences)
	req, _ := http.NewRequest("PUT", "/v1/preferences/learning", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.UserLearningPreferences
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), "it-IT-IsabellaNeural", response.TTSVoice)
}

func (suite *SettingsIntegrationTestSuite) TestUpdateLearningPreferences_InvalidJSON() {
	sessionCookie := suite.login()

	req, _ := http.NewRequest("PUT", "/v1/preferences/learning", bytes.NewBufferString("invalid json"))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *SettingsIntegrationTestSuite) TestUpdateLearningPreferences_Unauthorized() {
	preferences := map[string]interface{}{
		"focus_on_weak_areas": true,
	}

	body, _ := json.Marshal(preferences)
	req, _ := http.NewRequest("PUT", "/v1/preferences/learning", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// Edge case tests
func (suite *SettingsIntegrationTestSuite) TestUpdateUserSettings_EmptyValues() {
	sessionCookie := suite.login()

	settings := api.UserSettings{
		Language: langPtr(""),
		Level:    levelPtr(""),
	}

	body, _ := json.Marshal(settings)
	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *SettingsIntegrationTestSuite) TestUpdateUserSettings_ExtremelyLongValues() {
	sessionCookie := suite.login()

	longString := string(make([]byte, 1000)) // 1000 character string
	settings := api.UserSettings{
		Language: langPtr(longString),
		Level:    levelPtr("A1"),
	}

	body, _ := json.Marshal(settings)
	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Should either succeed or fail gracefully, not crash
	assert.Contains(suite.T(), []int{http.StatusOK, http.StatusBadRequest, http.StatusServiceUnavailable}, w.Code)
}

func (suite *SettingsIntegrationTestSuite) TestUpdateLearningPreferences_InvalidValues() {
	sessionCookie := suite.login()

	preferences := map[string]interface{}{
		"fresh_question_ratio": 1.5,             // Invalid: should be 0-1
		"focus_on_weak_areas":  "not a boolean", // Invalid: should be boolean
	}

	body, _ := json.Marshal(preferences)
	req, _ := http.NewRequest("PUT", "/v1/preferences/learning", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Should handle invalid values gracefully
	assert.Contains(suite.T(), []int{http.StatusOK, http.StatusBadRequest, http.StatusServiceUnavailable}, w.Code)
}

// Comprehensive API key handling tests
func (suite *SettingsIntegrationTestSuite) TestUpdateUserSettings_APIKeyHandling() {
	sessionCookie := suite.login()

	// Test 1: Save API key with AI enabled
	settings := api.UserSettings{
		Language:   langPtr("italian"),
		Level:      levelPtr("B1"),
		AiProvider: stringPtr("google"),
		AiModel:    stringPtr("gemini-2.5-flash"),
		ApiKey:     stringPtr("test-api-key-123"),
		AiEnabled:  boolPtr(true),
	}

	body, _ := json.Marshal(settings)
	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify API key was saved in user_api_keys table
	var apiKey string
	err := suite.DB.QueryRow("SELECT api_key FROM user_api_keys WHERE user_id = $1 AND provider = $2",
		suite.TestUserID, "google").Scan(&apiKey)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test-api-key-123", apiKey)

	// Test 2: Update API key for same provider
	settings.ApiKey = stringPtr("updated-api-key-456")
	body, _ = json.Marshal(settings)
	req, _ = http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify API key was updated
	err = suite.DB.QueryRow("SELECT api_key FROM user_api_keys WHERE user_id = $1 AND provider = $2",
		suite.TestUserID, "google").Scan(&apiKey)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "updated-api-key-456", apiKey)

	// Test 3: Save API key for different provider
	settings.AiProvider = stringPtr("openai")
	settings.AiModel = stringPtr("gpt-4")
	settings.ApiKey = stringPtr("openai-api-key-789")
	body, _ = json.Marshal(settings)
	req, _ = http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify both API keys exist
	var openaiKey string
	err = suite.DB.QueryRow("SELECT api_key FROM user_api_keys WHERE user_id = $1 AND provider = $2",
		suite.TestUserID, "openai").Scan(&openaiKey)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "openai-api-key-789", openaiKey)

	// Verify google key still exists
	err = suite.DB.QueryRow("SELECT api_key FROM user_api_keys WHERE user_id = $1 AND provider = $2",
		suite.TestUserID, "google").Scan(&apiKey)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "updated-api-key-456", apiKey)

	// Test 4: Disable AI and verify API key is not cleared (should remain in user_api_keys)
	// Send the actual values - backend will clear them when AI is disabled
	settings.AiEnabled = boolPtr(false)
	settings.AiProvider = stringPtr("google")
	settings.AiModel = stringPtr("gemini-2.5-flash")
	settings.ApiKey = stringPtr("test-api-key")
	body, _ = json.Marshal(settings)
	req, _ = http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify API keys still exist in user_api_keys table
	err = suite.DB.QueryRow("SELECT api_key FROM user_api_keys WHERE user_id = $1 AND provider = $2",
		suite.TestUserID, "google").Scan(&apiKey)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "updated-api-key-456", apiKey)

	err = suite.DB.QueryRow("SELECT api_key FROM user_api_keys WHERE user_id = $1 AND provider = $2",
		suite.TestUserID, "openai").Scan(&openaiKey)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "openai-api-key-789", openaiKey)
}

func (suite *SettingsIntegrationTestSuite) TestUpdateUserSettings_APIKeyWithEmptyString() {
	sessionCookie := suite.login()

	// Test saving API key with empty string (should not save)
	settings := api.UserSettings{
		Language:   langPtr("italian"),
		Level:      levelPtr("B1"),
		AiProvider: stringPtr("google"),
		AiModel:    stringPtr("gemini-2.5-flash"),
		ApiKey:     stringPtr(""), // Send empty string - backend should not save it
		AiEnabled:  boolPtr(true),
	}

	body, _ := json.Marshal(settings)
	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify no API key was saved for empty string
	var count int
	err := suite.DB.QueryRow("SELECT COUNT(*) FROM user_api_keys WHERE user_id = $1 AND provider = $2",
		suite.TestUserID, "google").Scan(&count)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, count)
}

func (suite *SettingsIntegrationTestSuite) TestUpdateUserSettings_APIKeyWithoutProvider() {
	sessionCookie := suite.login()

	// Test saving API key without provider (should not save)
	settings := api.UserSettings{
		Language:   langPtr("italian"),
		Level:      levelPtr("B1"),
		AiProvider: nil, // Don't send provider
		AiModel:    stringPtr("gemini-2.5-flash"),
		ApiKey:     stringPtr("test-api-key"),
		AiEnabled:  boolPtr(true),
	}

	body, _ := json.Marshal(settings)
	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify no API key was saved without provider
	var count int
	err := suite.DB.QueryRow("SELECT COUNT(*) FROM user_api_keys WHERE user_id = $1",
		suite.TestUserID).Scan(&count)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, count)
}

func (suite *SettingsIntegrationTestSuite) TestUpdateUserSettings_APIKeyWithAIDisabled() {
	sessionCookie := suite.login()

	// Test saving API key with AI disabled (should not save)
	settings := api.UserSettings{
		Language:   langPtr("italian"),
		Level:      levelPtr("B1"),
		AiProvider: stringPtr("google"),
		AiModel:    stringPtr("gemini-2.5-flash"),
		ApiKey:     stringPtr("test-api-key"),
		AiEnabled:  boolPtr(false), // AI disabled
	}

	body, _ := json.Marshal(settings)
	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(body))
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify no API key was saved when AI is disabled
	var count int
	err := suite.DB.QueryRow("SELECT COUNT(*) FROM user_api_keys WHERE user_id = $1 AND provider = $2",
		suite.TestUserID, "google").Scan(&count)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, count)
}

func TestSettingsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SettingsIntegrationTestSuite))
}

// Helper functions for pointer conversion
func langPtr(s string) *api.Language {
	l := api.Language(s)
	return &l
}

func levelPtr(s string) *api.Level {
	l := api.Level(s)
	return &l
}
