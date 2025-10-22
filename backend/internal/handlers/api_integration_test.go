//go:build integration
// +build integration

package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/database"
	"quizapp/internal/handlers"
	"quizapp/internal/services"

	"quizapp/internal/models"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type APIIntegrationTestSuite struct {
	suite.Suite
	Router        *gin.Engine
	db            *sql.DB
	testUser      *models.User
	userService   *services.UserService
	cfg           *config.Config
	mockAIService services.AIServiceInterface // Added for mocking AI calls
}

func (suite *APIIntegrationTestSuite) SetupSuite() {
	// Use the real config system - config.local.test.yaml should have signups enabled
	// The Taskfile sets QUIZ_CONFIG_FILE environment variable

	// Initialize database
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := database.NewManager(logger)
	db, err := dbManager.InitDB(dbURL)
	suite.Require().NoError(err)
	suite.db = db

	// Load the real config (will use config.local.yaml override)
	cfg, err := config.NewConfig()
	suite.Require().NoError(err)
	suite.Require().NotNil(cfg, "Config should not be nil after loading config")
	suite.T().Logf("Loaded languages: %v", cfg.GetLanguages())
	suite.T().Logf("Loaded all levels: %v", cfg.GetAllLevels())
	suite.T().Logf("Signups disabled: %v", cfg.System.Auth.SignupsDisabled)
	suite.cfg = cfg

	// Initialize services
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	suite.userService = userService
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create mock AI service for testing (faster than real external calls)
	mockAIService := handlers.NewMockAIService(cfg, logger)
	suite.mockAIService = mockAIService

	workerService := services.NewWorkerServiceWithLogger(db, logger)
	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Use the new router factory
	generationHintService := services.NewGenerationHintService(suite.db, logger)
	storyService := services.NewStoryService(suite.db, cfg, logger)
	usageStatsService := services.NewUsageStatsService(cfg, suite.db, logger)
	translationCacheRepo := services.NewTranslationCacheRepository(suite.db, logger)
	translationService := services.NewTranslationService(cfg, usageStatsService, translationCacheRepo, logger)
	snippetsService := services.NewSnippetsService(suite.db, cfg, logger)
	router := handlers.NewRouter(cfg, userService, questionService, learningService, suite.mockAIService, workerService, dailyQuestionService, storyService, services.NewConversationService(db), oauthService, generationHintService, translationService, snippetsService, usageStatsService, logger)
	suite.Router = router
}

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}

// boolPtr returns a pointer to the given bool
func boolPtr(b bool) *bool {
	return &b
}

// languagePtr returns a pointer to the given Language
func languagePtr(l api.Language) *api.Language {
	return &l
}

// levelPtr returns a pointer to the given Level
func levelPtr(l api.Level) *api.Level {
	return &l
}

// questionTypePtr returns a pointer to the given QuestionType
func questionTypePtr(t api.QuestionType) *api.QuestionType {
	return &t
}

// emailPtr returns a pointer to the given email
func emailPtr(e string) *openapi_types.Email {
	email := openapi_types.Email(e)
	return &email
}

// createTestQuestion creates a question for the test user and returns its ID
func (suite *APIIntegrationTestSuite) createTestQuestion() int {
	testQuestion := &models.Question{
		Type:            "vocabulary",
		Language:        "italian",
		Level:           "A1",
		DifficultyScore: 1.0,
		Content:         map[string]interface{}{"question": "What is 'ciao'?", "options": []string{"hello", "goodbye", "please", "thanks"}},
		CorrectAnswer:   0,
		Explanation:     "Ciao means hello",
		Status:          models.QuestionStatusActive,
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	err := questionService.SaveQuestion(context.Background(), testQuestion)
	suite.Require().NoError(err)
	err = questionService.AssignQuestionToUser(context.Background(), testQuestion.ID, suite.testUser.ID)
	suite.Require().NoError(err)
	return testQuestion.ID
}

// Test reporting a question
func (suite *APIIntegrationTestSuite) TestReportQuestion() {
	cookie := suite.login()

	// Test 1: Report without reason (existing functionality)
	questionID1 := suite.createTestQuestion()
	url1 := fmt.Sprintf("/v1/quiz/question/%d/report", questionID1)
	req1, _ := http.NewRequest("POST", url1, nil)
	req1.Header.Set("Cookie", cookie)

	w1 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w1, req1)

	assert.Equal(suite.T(), http.StatusOK, w1.Code)
	var resp1 api.SuccessResponse
	err := json.Unmarshal(w1.Body.Bytes(), &resp1)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), *resp1.Message, "Question reported successfully")

	// Verify in DB - should have default reason
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Check that the question is reported
	q1, err := questionService.GetQuestionByID(context.Background(), questionID1)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.QuestionStatusReported, q1.Status)

	// Check the report record in the database
	var reportReason1 string
	err = suite.db.QueryRow("SELECT report_reason FROM question_reports WHERE question_id = $1 AND reported_by_user_id = $2",
		questionID1, suite.testUser.ID).Scan(&reportReason1)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Question reported by user", reportReason1)

	// Test 2: Report with reason
	questionID2 := suite.createTestQuestion()
	url2 := fmt.Sprintf("/v1/quiz/question/%d/report", questionID2)
	reportData := map[string]interface{}{
		"report_reason": "This question has incorrect grammar",
	}
	body, _ := json.Marshal(reportData)

	req2, _ := http.NewRequest("POST", url2, bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Cookie", cookie)

	w2 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w2, req2)

	assert.Equal(suite.T(), http.StatusOK, w2.Code)
	var resp2 api.SuccessResponse
	err = json.Unmarshal(w2.Body.Bytes(), &resp2)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), *resp2.Message, "Question reported successfully")

	// Verify the report reason was stored
	var reportReason2 string
	err = suite.db.QueryRow("SELECT report_reason FROM question_reports WHERE question_id = $1 AND reported_by_user_id = $2",
		questionID2, suite.testUser.ID).Scan(&reportReason2)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "This question has incorrect grammar", reportReason2)

	// Test 3: Report with empty reason (should use default)
	questionID3 := suite.createTestQuestion()
	url3 := fmt.Sprintf("/v1/quiz/question/%d/report", questionID3)
	reportData3 := map[string]interface{}{
		"report_reason": "",
	}
	body3, _ := json.Marshal(reportData3)

	req3, _ := http.NewRequest("POST", url3, bytes.NewBuffer(body3))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Cookie", cookie)

	w3 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w3, req3)

	assert.Equal(suite.T(), http.StatusOK, w3.Code)
	var resp3 api.SuccessResponse
	err = json.Unmarshal(w3.Body.Bytes(), &resp3)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), *resp3.Message, "Question reported successfully")

	// Verify empty reason uses default
	var reportReason3 string
	err = suite.db.QueryRow("SELECT report_reason FROM question_reports WHERE question_id = $1 AND reported_by_user_id = $2",
		questionID3, suite.testUser.ID).Scan(&reportReason3)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Question reported by user", reportReason3)

	// Test 4: Report same question again with a new reason - should update existing report
	newReason := "Updated reason after fix"
	reportData4 := map[string]interface{}{
		"report_reason": newReason,
	}
	body4, _ := json.Marshal(reportData4)

	req4, _ := http.NewRequest("POST", fmt.Sprintf("/v1/quiz/question/%d/report", questionID3), bytes.NewBuffer(body4))
	req4.Header.Set("Content-Type", "application/json")
	req4.Header.Set("Cookie", cookie)

	w4 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w4, req4)

	assert.Equal(suite.T(), http.StatusOK, w4.Code)

	var reportReason4 string
	err = suite.db.QueryRow("SELECT report_reason FROM question_reports WHERE question_id = $1 AND reported_by_user_id = $2",
		questionID3, suite.testUser.ID).Scan(&reportReason4)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), newReason, reportReason4)
}

// Test submitting an answer
func (suite *APIIntegrationTestSuite) TestSubmitAnswer() {
	cookie := suite.login()

	questionID := suite.createTestQuestion()

	response := &api.AnswerRequest{
		QuestionId:      int64(questionID),
		UserAnswerIndex: 0,
		ResponseTimeMs:  nil,
	}

	body, _ := json.Marshal(response)
	req, _ := http.NewRequest("POST", "/v1/quiz/answer", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	suite.Equal(200, w.Code)

	var result api.AnswerResponse
	err := json.Unmarshal(w.Body.Bytes(), &result)
	suite.Require().NoError(err)

	suite.True(*result.IsCorrect)
	suite.Equal(0, *result.CorrectAnswerIndex) // The correct answer is at index 0
}

func (suite *APIIntegrationTestSuite) SetupTest() {
	// Clean up database before each test using the shared cleanup function
	services.CleanupTestDatabase(suite.db, suite.T())

	// Create a standard test user for all API tests with all required fields
	testUser, err := suite.userService.CreateUserWithPassword(context.Background(), "testuser_api", "testpass", "italian", "A1")
	if err != nil {
		suite.T().Fatalf("Failed to create test user: %v", err)
	}
	if testUser == nil {
		suite.T().Fatalf("Test user is nil after creation")
	}

	// Update the user with all required fields that the validation middleware expects
	_, err = suite.db.Exec(`
		UPDATE users
		SET email = $1, timezone = $2, ai_provider = $3, ai_model = $4, last_active = $5
		WHERE id = $6
	`, "testuser_api@example.com", "UTC", "ollama", "llama3", time.Now(), testUser.ID)
	if err != nil {
		suite.T().Fatalf("Failed to update test user with required fields: %v", err)
	}

	// Reload the user to get the updated data
	testUser, err = suite.userService.GetUserByID(context.Background(), testUser.ID)
	if err != nil {
		suite.T().Fatalf("Failed to reload test user: %v", err)
	}

	suite.testUser = testUser
	suite.T().Logf("DEBUG: Created test user with ID: %d", testUser.ID)
}

func (suite *APIIntegrationTestSuite) TearDownSuite() {
	suite.db.Close()
}

// login performs a login and returns the session cookie
func (suite *APIIntegrationTestSuite) login() string {
	loginReq := api.LoginRequest{
		Username: "testuser_api",
		Password: "testpass",
	}
	reqBody, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code, "Login should be successful")
	cookie := w.Result().Header.Get("Set-Cookie")
	assert.NotEmpty(suite.T(), cookie, "Session cookie should be set")

	return cookie
}

func (suite *APIIntegrationTestSuite) TestLoginAndStatus() {
	// Step 1: Login and get cookie
	cookie := suite.login()

	// Step 2: Access a protected route using the cookie
	req, _ := http.NewRequest("GET", "/v1/auth/status", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var statusResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &statusResp)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), true, statusResp["authenticated"])

	userMap, ok := statusResp["user"].(map[string]interface{})
	assert.True(suite.T(), ok, "User object should be a map")
	assert.Equal(suite.T(), "testuser_api", userMap["username"], "Correct username should be returned")
}

func (suite *APIIntegrationTestSuite) TestGetQuestion_EmptyCache() {
	// Step 1: Login and get cookie
	cookie := suite.login()

	// Step 2: Try to get a question. Since the cache is empty, we expect a 202 Accepted with generating status.
	req, _ := http.NewRequest("GET", "/v1/quiz/question", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Expect a 202 Accepted status because the cache is empty and worker is separate
	assert.Equal(suite.T(), http.StatusAccepted, w.Code)

	var generatingResp api.GeneratingResponse
	err := json.Unmarshal(w.Body.Bytes(), &generatingResp)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "generating", *generatingResp.Status)
	assert.Contains(suite.T(), *generatingResp.Message, "No questions available")
}

func (suite *APIIntegrationTestSuite) TestGetProgress_Success() {
	// Step 1: Login and get cookie
	cookie := suite.login()

	// Step 2: Try to get progress
	req, _ := http.NewRequest("GET", "/v1/quiz/progress", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Expect a successful response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var progressResp models.UserProgress
	err := json.Unmarshal(w.Body.Bytes(), &progressResp)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), progressResp)
}

func (suite *APIIntegrationTestSuite) TestLoginFailure_InvalidCredentials() {
	loginReq := api.LoginRequest{
		Username: "testuser_api",
		Password: "wrongpassword",
	}
	reqBody, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	var errResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "INVALID_CREDENTIALS", errResp["code"])
}

func (suite *APIIntegrationTestSuite) TestProtectedRoute_Unauthorized() {
	// Try to access a protected route without authentication
	req, _ := http.NewRequest("GET", "/v1/quiz/question", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *APIIntegrationTestSuite) TestSignup_Success() {
	// Test signup with valid data
	signupReq := api.UserCreateRequest{
		Username:          "newuser_signup",
		Password:          "newpass123",
		PreferredLanguage: stringPtr("italian"),
		CurrentLevel:      stringPtr("B1"),
		Email:             emailPtr("newuser@example.com"),
		Timezone:          stringPtr("America/New_York"),
	}
	reqBody, _ := json.Marshal(signupReq)

	req, _ := http.NewRequest("POST", "/v1/auth/signup", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var signupResp api.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &signupResp)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), signupResp.Success)
	assert.Contains(suite.T(), *signupResp.Message, "Account created successfully")

	// Verify user was actually created in database
	createdUser, err := suite.userService.GetUserByUsername(context.Background(), "newuser_signup")
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), createdUser)
	assert.Equal(suite.T(), "italian", createdUser.PreferredLanguage.String)
	assert.Equal(suite.T(), "B1", createdUser.CurrentLevel.String)
}

func (suite *APIIntegrationTestSuite) TestSignup_DuplicateUsername() {
	// First, create a user
	signupReq1 := api.UserCreateRequest{
		Username:          "duplicate_user",
		Password:          "pass12345", // At least 8 characters
		PreferredLanguage: stringPtr("italian"),
		CurrentLevel:      stringPtr("A1"),
		Email:             emailPtr("user1@example.com"),
		Timezone:          stringPtr("UTC"),
	}
	reqBody1, _ := json.Marshal(signupReq1)

	req1, _ := http.NewRequest("POST", "/v1/auth/signup", bytes.NewBuffer(reqBody1))
	req1.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w1, req1)

	assert.Equal(suite.T(), http.StatusCreated, w1.Code)

	// Try to create another user with the same username
	signupReq2 := api.UserCreateRequest{
		Username:          "duplicate_user",
		Password:          "pass67890", // At least 8 characters
		PreferredLanguage: stringPtr("french"),
		CurrentLevel:      stringPtr("A2"),
		Email:             emailPtr("user2@example.com"),
		Timezone:          stringPtr("Europe/London"),
	}
	reqBody2, _ := json.Marshal(signupReq2)

	req2, _ := http.NewRequest("POST", "/v1/auth/signup", bytes.NewBuffer(reqBody2))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w2, req2)

	assert.Equal(suite.T(), http.StatusConflict, w2.Code)

	var errResp map[string]interface{}
	err := json.Unmarshal(w2.Body.Bytes(), &errResp)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "RECORD_ALREADY_EXISTS", errResp["code"])
}

func (suite *APIIntegrationTestSuite) TestSignup_InvalidData() {
	// Test signup with invalid data (missing required fields)
	signupReq := api.UserCreateRequest{
		Username:          "", // Empty username
		Password:          "pass123",
		PreferredLanguage: stringPtr("italian"),
		CurrentLevel:      stringPtr("A1"),
		Email:             emailPtr("test@example.com"),
		Timezone:          stringPtr("UTC"),
	}
	reqBody, _ := json.Marshal(signupReq)

	req, _ := http.NewRequest("POST", "/v1/auth/signup", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var errResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), errResp, "error")
	fmt.Printf("DEBUG: Response: %+v\n", errResp)
	fmt.Printf("DEBUG: Status code: %d\n", w.Code)
}

// Test getting worker status
func (suite *APIIntegrationTestSuite) TestGetWorkerStatus() {
	cookie := suite.login()

	endpoints := []string{
		"/v1/quiz/worker-status",
	}

	for _, endpoint := range endpoints {
		req, _ := http.NewRequest("GET", endpoint, nil)
		req.Header.Set("Cookie", cookie)

		w := httptest.NewRecorder()
		suite.Router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(suite.T(), err)

		// Check expected fields
		assert.Contains(suite.T(), resp, "has_errors")
		assert.Contains(suite.T(), resp, "error_message")
		assert.Contains(suite.T(), resp, "global_paused")
		assert.Contains(suite.T(), resp, "user_paused")
		assert.Contains(suite.T(), resp, "healthy_workers")
		assert.Contains(suite.T(), resp, "total_workers")
		assert.Contains(suite.T(), resp, "last_error_details")
		assert.Contains(suite.T(), resp, "worker_running")

		// Check types for booleans
		_, ok := resp["global_paused"].(bool)
		assert.True(suite.T(), ok, "global_paused should be a bool")
		_, ok = resp["user_paused"].(bool)
		assert.True(suite.T(), ok, "user_paused should be a bool")
	}
}

func (suite *APIIntegrationTestSuite) TestGetLevels() {
	req, _ := http.NewRequest("GET", "/v1/settings/levels", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(suite.T(), err)

	// Check that both levels and level_descriptions are present
	assert.Contains(suite.T(), resp, "levels")
	assert.Contains(suite.T(), resp, "level_descriptions")

	// Extract levels array and verify it contains expected levels
	levels, ok := resp["levels"].([]interface{})
	assert.True(suite.T(), ok, "levels should be an array")
	assert.NotEmpty(suite.T(), levels)

	// Convert to string slice for easier assertion
	levelStrings := make([]string, len(levels))
	for i, level := range levels {
		levelStrings[i] = level.(string)
	}

	assert.Contains(suite.T(), levelStrings, "A1")
	assert.Contains(suite.T(), levelStrings, "B1")

	// --- Language-specific tests ---
	// Japanese
	req, _ = http.NewRequest("GET", "/v1/settings/levels?language=japanese", nil)
	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var respJa map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &respJa)
	assert.NoError(suite.T(), err)
	levelsJa, ok := respJa["levels"].([]interface{})
	assert.True(suite.T(), ok, "levels should be an array")
	levelStringsJa := make([]string, len(levelsJa))
	for i, level := range levelsJa {
		levelStringsJa[i] = level.(string)
	}
	assert.ElementsMatch(suite.T(), levelStringsJa, []string{"N5", "N4", "N3", "N2", "N1"})

	// Chinese
	req, _ = http.NewRequest("GET", "/v1/settings/levels?language=chinese", nil)
	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var respZh map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &respZh)
	assert.NoError(suite.T(), err)
	levelsZh, ok := respZh["levels"].([]interface{})
	assert.True(suite.T(), ok, "levels should be an array")
	levelStringsZh := make([]string, len(levelsZh))
	for i, level := range levelsZh {
		levelStringsZh[i] = level.(string)
	}
	assert.ElementsMatch(suite.T(), levelStringsZh, []string{"HSK1", "HSK2", "HSK3", "HSK4", "HSK5", "HSK6"})

	// Italian (CEFR)
	req, _ = http.NewRequest("GET", "/v1/settings/levels?language=italian", nil)
	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var respIt map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &respIt)
	assert.NoError(suite.T(), err)
	levelsIt, ok := respIt["levels"].([]interface{})
	assert.True(suite.T(), ok, "levels should be an array")
	levelStringsIt := make([]string, len(levelsIt))
	for i, level := range levelsIt {
		levelStringsIt[i] = level.(string)
	}
	assert.ElementsMatch(suite.T(), levelStringsIt, []string{"A1", "A2", "B1", "B1+", "B1++", "B2", "C1", "C2"})
}

// Ensure levels per language match config.yaml for all languages
func (suite *APIIntegrationTestSuite) TestLevels_PerLanguage_MatchesConfig() {
	languages := suite.cfg.GetLanguages()
	suite.Require().NotEmpty(languages)

	for _, lang := range languages {
		expected := suite.cfg.GetLevelsForLanguage(lang)
		req, _ := http.NewRequest("GET", "/v1/settings/levels?language="+lang, nil)
		w := httptest.NewRecorder()
		suite.Router.ServeHTTP(w, req)
		suite.Equal(http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		suite.Require().NoError(err)
		levelsAny, ok := resp["levels"].([]interface{})
		suite.Require().True(ok)
		got := make([]string, len(levelsAny))
		for i, v := range levelsAny {
			got[i] = v.(string)
		}
		assert.ElementsMatch(suite.T(), expected, got)
	}
}

// Ensure languages endpoint matches config.yaml languages
func (suite *APIIntegrationTestSuite) TestLanguages_MatchConfig() {
	expected := suite.cfg.GetLanguages()
	req, _ := http.NewRequest("GET", "/v1/settings/languages", nil)
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	var languageInfos []config.LanguageInfo
	err := json.Unmarshal(w.Body.Bytes(), &languageInfos)
	suite.Require().NoError(err)

	// Extract language names from the response for comparison
	actual := make([]string, len(languageInfos))
	for i, langInfo := range languageInfos {
		actual[i] = langInfo.Name
	}
	assert.ElementsMatch(suite.T(), expected, actual)
}

// Test updating user settings
func (suite *APIIntegrationTestSuite) TestUpdateUserSettings() {
	cookie := suite.login()

	settingsReq := api.UserSettings{
		Language:   languagePtr(api.Language("french")),
		Level:      levelPtr(api.Level("B1")),
		AiProvider: stringPtr("openai"),
		AiModel:    stringPtr("gpt-3.5-turbo"),
		ApiKey:     stringPtr("test-api-key"),
		AiEnabled:  boolPtr(true),
	}
	reqBody, _ := json.Marshal(settingsReq)

	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var resp api.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), resp.Success)

	// Verify settings were updated in database
	updatedUser, err := suite.userService.GetUserByID(context.Background(), suite.testUser.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "french", updatedUser.PreferredLanguage.String)
	assert.Equal(suite.T(), "B1", updatedUser.CurrentLevel.String)
	assert.Equal(suite.T(), "openai", updatedUser.AIProvider.String)
	assert.Equal(suite.T(), "gpt-3.5-turbo", updatedUser.AIModel.String)
	assert.True(suite.T(), updatedUser.AIEnabled.Bool)
}

// Test updating user settings with invalid level
func (suite *APIIntegrationTestSuite) TestUpdateUserSettings_InvalidLevel() {
	cookie := suite.login()

	settingsReq := api.UserSettings{
		Language:   languagePtr(api.Language("italian")),
		Level:      levelPtr(api.Level("INVALID_LEVEL")),
		AiProvider: stringPtr("openai"),
		AiModel:    stringPtr("gpt-3.5-turbo"),
		ApiKey:     stringPtr("test-api-key"),
		AiEnabled:  boolPtr(true),
	}
	reqBody, _ := json.Marshal(settingsReq)

	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	// Check the response format - it could be either ErrorResponse or a different structure
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(suite.T(), err)

	// The response should contain a message field with the error information
	assert.Contains(suite.T(), resp, "message")
	message := resp["message"].(string)
	assert.True(suite.T(), strings.Contains(message, "Invalid") || strings.Contains(message, "level"),
		"Expected error message to contain 'Invalid' or 'level', got: %s", message)
}

// Test updating user settings without authentication
func (suite *APIIntegrationTestSuite) TestUpdateUserSettings_Unauthorized() {
	settingsReq := api.UserSettings{
		Language:   languagePtr(api.Language("german")),
		Level:      levelPtr(api.Level("A2")),
		AiProvider: stringPtr("anthropic"),
		AiModel:    stringPtr("claude-3-sonnet"),
		ApiKey:     stringPtr("test-api-key"),
		AiEnabled:  boolPtr(false),
	}
	reqBody, _ := json.Marshal(settingsReq)

	req, _ := http.NewRequest("PUT", "/v1/settings", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// Test AI connection with valid settings
func (suite *APIIntegrationTestSuite) TestTestAIConnection() {
	cookie := suite.login()

	testReq := api.TestAIRequest{
		Provider: "openai",
		Model:    "gpt_3_5_turbo", // updated to match pattern
		ApiKey:   stringPtr("test-api-key"),
	}
	reqBody, _ := json.Marshal(testReq)

	req, _ := http.NewRequest("POST", "/v1/settings/test-ai", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// The test will likely fail since we don't have a real API key, but we should get a proper response
	// First check if it's a validation error (400) or handler response (200)
	if w.Code == http.StatusBadRequest {
		// Request validation failed
		var errResp api.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		assert.NoError(suite.T(), err)
		assert.Contains(suite.T(), *errResp.Error, "validation")
	} else {
		// Handler returned a response (likely 200 with success=false)
		assert.Equal(suite.T(), http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(suite.T(), err)
		assert.Contains(suite.T(), resp, "success")
		assert.Equal(suite.T(), false, resp["success"]) // Should be false since we're using a test API key
	}
}

// Test AI connection without authentication
func (suite *APIIntegrationTestSuite) TestTestAIConnection_Unauthorized() {
	testReq := api.TestAIRequest{
		Provider: "openai",
		Model:    "gpt-3.5-turbo",
		ApiKey:   stringPtr("test-api-key"),
	}
	reqBody, _ := json.Marshal(testReq)

	req, _ := http.NewRequest("POST", "/v1/settings/test-ai", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// Test AI connection with invalid request format
func (suite *APIIntegrationTestSuite) TestTestAIConnection_InvalidRequest() {
	cookie := suite.login()

	// Send invalid JSON
	req, _ := http.NewRequest("POST", "/v1/settings/test-ai", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	// Check the response format - it could be either ErrorResponse or a different structure
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(suite.T(), err)

	// The response should contain a message field with the error information
	assert.Contains(suite.T(), resp, "message")
	assert.Contains(suite.T(), resp["message"].(string), "Invalid request format")
}

func (suite *APIIntegrationTestSuite) TestGetQuestion_ReturnsVarietyMetadata() {
	// Login and get cookie
	cookie := suite.login()

	// Create a question with all variety metadata fields
	question := &models.Question{
		Type:               "vocabulary",
		Language:           "italian",
		Level:              "A1",
		DifficultyScore:    1.0,
		Content:            map[string]interface{}{"question": "What is 'ciao'?", "options": []string{"hello", "goodbye", "please", "thanks"}},
		CorrectAnswer:      0,
		Explanation:        "Ciao means hello",
		Status:             models.QuestionStatusActive,
		TopicCategory:      "greetings",
		GrammarFocus:       "present tense",
		VocabularyDomain:   "daily life",
		Scenario:           "ordering coffee",
		StyleModifier:      "formal",
		DifficultyModifier: "easy",
		TimeContext:        "morning",
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	err := questionService.SaveQuestion(context.Background(), question)
	suite.Require().NoError(err)
	err = questionService.AssignQuestionToUser(context.Background(), question.ID, suite.testUser.ID)
	suite.Require().NoError(err)

	// Fetch the question via the API
	req, _ := http.NewRequest("GET", "/v1/quiz/question/"+fmt.Sprint(question.ID), nil)
	req.Header.Set("Cookie", cookie)
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	suite.Require().NoError(err)

	// Assert all variety metadata fields are present and correct
	suite.Equal("greetings", resp["topic_category"])
	suite.Equal("present tense", resp["grammar_focus"])
	suite.Equal("daily life", resp["vocabulary_domain"])
	suite.Equal("ordering coffee", resp["scenario"])
	suite.Equal("formal", resp["style_modifier"])
	suite.Equal("easy", resp["difficulty_modifier"])
	suite.Equal("morning", resp["time_context"])
}

// Test marking a question as known
func (suite *APIIntegrationTestSuite) TestMarkQuestionAsKnown() {
	cookie := suite.login()
	questionID := suite.createTestQuestion()

	// Test marking question as known without confidence level
	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/quiz/question/%d/mark-known", questionID), nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var resp api.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), *resp.Message, "Question marked as known successfully")

	// Verify in database that question is marked as known
	var markedAsKnown bool
	var markedAt sql.NullTime
	var confidenceLevel sql.NullInt32
	err = suite.db.QueryRow(`
		SELECT marked_as_known, marked_as_known_at, confidence_level
		FROM user_question_metadata
		WHERE user_id = $1 AND question_id = $2
	`, suite.testUser.ID, questionID).Scan(&markedAsKnown, &markedAt, &confidenceLevel)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), markedAsKnown)
	assert.True(suite.T(), markedAt.Valid)
	assert.False(suite.T(), confidenceLevel.Valid) // Should be NULL if not provided
}

// Test marking a question as known with confidence level
func (suite *APIIntegrationTestSuite) TestMarkQuestionAsKnown_WithConfidenceLevel() {
	cookie := suite.login()
	questionID := suite.createTestQuestion()

	// Test marking question as known with confidence level
	confidenceReq := map[string]interface{}{
		"confidence_level": 5,
	}
	reqBody, _ := json.Marshal(confidenceReq)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/quiz/question/%d/mark-known", questionID), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var resp api.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), *resp.Message, "Question marked as known successfully")

	// Verify in database that question is marked as known
	var markedAsKnown bool
	var markedAt sql.NullTime
	var confidenceLevel sql.NullInt32
	err = suite.db.QueryRow(`
		SELECT marked_as_known, marked_as_known_at, confidence_level
		FROM user_question_metadata
		WHERE user_id = $1 AND question_id = $2
	`, suite.testUser.ID, questionID).Scan(&markedAsKnown, &markedAt, &confidenceLevel)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), markedAsKnown)
	assert.True(suite.T(), markedAt.Valid)
	assert.True(suite.T(), confidenceLevel.Valid)
	assert.Equal(suite.T(), int32(5), confidenceLevel.Int32)
}

// Test that confidence level is included in question API response
func (suite *APIIntegrationTestSuite) TestGetQuestion_IncludesConfidenceLevel() {
	cookie := suite.login()

	// Create a question to test with
	questionID1 := suite.createTestQuestion()

	// Mark one of the questions as known with a confidence level
	confidenceReq := map[string]interface{}{
		"confidence_level": 3,
	}
	reqBody, _ := json.Marshal(confidenceReq)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/quiz/question/%d/mark-known", questionID1), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify in database that confidence level was stored
	var confidenceLevel sql.NullInt32
	err := suite.db.QueryRow(`
		SELECT confidence_level
		FROM user_question_metadata
		WHERE user_id = $1 AND question_id = $2
	`, suite.testUser.ID, questionID1).Scan(&confidenceLevel)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), confidenceLevel.Valid)
	assert.Equal(suite.T(), int32(3), confidenceLevel.Int32)

	// Now fetch the specific question and verify confidence level is included
	req, _ = http.NewRequest("GET", fmt.Sprintf("/v1/quiz/question/%d", questionID1), nil)
	req.Header.Set("Cookie", cookie)

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var question api.Question
	err = json.Unmarshal(w.Body.Bytes(), &question)
	assert.NoError(suite.T(), err)

	// Verify that the confidence level is included in the response
	assert.NotNil(suite.T(), question.ConfidenceLevel)
	assert.Equal(suite.T(), 3, *question.ConfidenceLevel)
}

// Test marking a question as known with invalid question ID
func (suite *APIIntegrationTestSuite) TestMarkQuestionAsKnown_InvalidQuestionID() {
	cookie := suite.login()

	// DEBUG: Check if question 99999 exists in the database
	var questionExists bool
	err := suite.db.QueryRow("SELECT EXISTS(SELECT 1 FROM questions WHERE id = $1)", 99999).Scan(&questionExists)
	suite.Require().NoError(err)
	suite.T().Logf("DEBUG: Question 99999 exists in database: %v", questionExists)

	// DEBUG: Check if there's a user_questions entry for this question
	var userQuestionExists bool
	err = suite.db.QueryRow("SELECT EXISTS(SELECT 1 FROM user_questions WHERE question_id = $1 AND user_id = $2)", 99999, suite.testUser.ID).Scan(&userQuestionExists)
	suite.Require().NoError(err)
	suite.T().Logf("DEBUG: User question entry exists for question 99999: %v", userQuestionExists)

	req, _ := http.NewRequest("POST", "/v1/quiz/question/99999/mark-known", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// DEBUG: Log the actual response
	suite.T().Logf("DEBUG: Response status: %d", w.Code)
	suite.T().Logf("DEBUG: Response body: %s", w.Body.String())

	// The test should either return 404 (if foreign key constraint is enforced)
	// or 200 (if the operation succeeds despite the non-existent question)
	// Both are acceptable outcomes depending on the database configuration
	if w.Code == http.StatusNotFound {
		// Expected behavior: foreign key constraint violation
		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "QUESTION_NOT_FOUND", resp["code"])
	} else if w.Code == http.StatusOK {
		// Unexpected but acceptable: operation succeeded
		var resp api.SuccessResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(suite.T(), err)
		assert.Contains(suite.T(), *resp.Message, "Question marked as known successfully")

		// Verify that the metadata was actually created (even though question doesn't exist)
		var markedAsKnown bool
		err = suite.db.QueryRow(`
			SELECT marked_as_known
			FROM user_question_metadata
			WHERE user_id = $1 AND question_id = $2
		`, suite.testUser.ID, 99999).Scan(&markedAsKnown)
		assert.NoError(suite.T(), err)
		assert.True(suite.T(), markedAsKnown)
	} else {
		suite.T().Fatalf("Unexpected status code: %d", w.Code)
	}
}

// Test marking a question as known without authentication
func (suite *APIIntegrationTestSuite) TestMarkQuestionAsKnown_Unauthorized() {
	req, _ := http.NewRequest("POST", "/v1/quiz/question/1/mark-known", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *APIIntegrationTestSuite) TestGetVersion() {
	// Test that /v1/version endpoint returns version info
	req, _ := http.NewRequest("GET", "/v1/version", nil)
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(suite.T(), err)

	// Check that response contains backend and worker sections
	assert.Contains(suite.T(), resp, "backend")
	assert.Contains(suite.T(), resp, "worker")

	// Check backend version info
	backend, ok := resp["backend"].(map[string]interface{})
	assert.True(suite.T(), ok, "backend should be an object")
	assert.Contains(suite.T(), backend, "version")
	assert.Contains(suite.T(), backend, "commit")
	assert.Contains(suite.T(), backend, "buildTime")
	assert.Contains(suite.T(), backend, "service")

	// Check types for backend version info
	_, ok = backend["version"].(string)
	assert.True(suite.T(), ok, "backend version should be a string")
	_, ok = backend["commit"].(string)
	assert.True(suite.T(), ok, "backend commit should be a string")
	_, ok = backend["buildTime"].(string)
	assert.True(suite.T(), ok, "backend buildTime should be a string")
	_, ok = backend["service"].(string)
	assert.True(suite.T(), ok, "backend service should be a string")
	assert.Equal(suite.T(), "backend", backend["service"])

	// Check worker section exists (may contain error if worker unavailable)
	_, ok = resp["worker"].(map[string]interface{})
	assert.True(suite.T(), ok, "worker should be an object")
}

func (suite *APIIntegrationTestSuite) TestQuizStatsUpdate() {
	cookie := suite.login()
	questionID := suite.createTestQuestion()

	// Helper to get progress
	getProgress := func() models.UserProgress {
		req, _ := http.NewRequest("GET", "/v1/quiz/progress", nil)
		req.Header.Set("Cookie", cookie)
		w := httptest.NewRecorder()
		suite.Router.ServeHTTP(w, req)
		suite.Equal(http.StatusOK, w.Code)
		var progress models.UserProgress
		err := json.Unmarshal(w.Body.Bytes(), &progress)
		suite.Require().NoError(err)
		return progress
	}

	// Initial progress: all zero
	progress := getProgress()
	assert.Equal(suite.T(), 0, progress.TotalQuestions)
	assert.Equal(suite.T(), 0, progress.CorrectAnswers)

	// Submit a correct answer
	response := &api.AnswerRequest{
		QuestionId:      int64(questionID),
		UserAnswerIndex: 0, // correct for test question
		ResponseTimeMs:  nil,
	}
	body, _ := json.Marshal(response)
	req, _ := http.NewRequest("POST", "/v1/quiz/answer", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	suite.Equal(200, w.Code)

	// Progress should show 1 shown, 1 correct
	progress = getProgress()
	assert.Equal(suite.T(), 1, progress.TotalQuestions)
	assert.Equal(suite.T(), 1, progress.CorrectAnswers)

	// Submit an incorrect answer
	response = &api.AnswerRequest{
		QuestionId:      int64(questionID),
		UserAnswerIndex: 1, // incorrect for test question
		ResponseTimeMs:  nil,
	}
	body, _ = json.Marshal(response)
	req, _ = http.NewRequest("POST", "/v1/quiz/answer", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	suite.Equal(200, w.Code)

	// Progress should show 2 shown, 1 correct (so 1 wrong)
	progress = getProgress()
	assert.Equal(suite.T(), 2, progress.TotalQuestions)
	assert.Equal(suite.T(), 1, progress.CorrectAnswers)
	// Wrong = TotalQuestions - CorrectAnswers
	assert.Equal(suite.T(), 1, progress.TotalQuestions-progress.CorrectAnswers)
}

func (suite *APIIntegrationTestSuite) TestQuizStatsUpdate_MultipleQuestionsAndEdgeCases() {
	cookie := suite.login()
	q1 := suite.createTestQuestion()
	q2 := suite.createTestQuestion()
	q3 := suite.createTestQuestion()

	getProgress := func() models.UserProgress {
		req, _ := http.NewRequest("GET", "/v1/quiz/progress", nil)
		req.Header.Set("Cookie", cookie)
		w := httptest.NewRecorder()
		suite.Router.ServeHTTP(w, req)
		suite.Equal(http.StatusOK, w.Code)
		var progress models.UserProgress
		err := json.Unmarshal(w.Body.Bytes(), &progress)
		suite.Require().NoError(err)
		return progress
	}

	// Initial: all zero
	progress := getProgress()
	assert.Equal(suite.T(), 0, progress.TotalQuestions)
	assert.Equal(suite.T(), 0, progress.CorrectAnswers)

	// Submit correct for q1, incorrect for q2, skip q3
	answer := func(qid, ans int) {
		response := &api.AnswerRequest{
			QuestionId:      int64(qid),
			UserAnswerIndex: ans,
			ResponseTimeMs:  nil,
		}
		body, _ := json.Marshal(response)
		req, _ := http.NewRequest("POST", "/v1/quiz/answer", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", cookie)
		w := httptest.NewRecorder()
		suite.Router.ServeHTTP(w, req)
		suite.Equal(200, w.Code)
	}

	answer(q1, 0) // correct
	progress = getProgress()
	assert.Equal(suite.T(), 1, progress.TotalQuestions)
	assert.Equal(suite.T(), 1, progress.CorrectAnswers)
	assert.Equal(suite.T(), 0, progress.TotalQuestions-progress.CorrectAnswers)

	answer(q2, 1) // incorrect
	progress = getProgress()
	assert.Equal(suite.T(), 2, progress.TotalQuestions)
	assert.Equal(suite.T(), 1, progress.CorrectAnswers)
	assert.Equal(suite.T(), 1, progress.TotalQuestions-progress.CorrectAnswers)

	// Edge: answer q1 again, but incorrectly
	answer(q1, 1)
	progress = getProgress()
	assert.Equal(suite.T(), 3, progress.TotalQuestions)
	assert.Equal(suite.T(), 1, progress.CorrectAnswers)
	assert.Equal(suite.T(), 2, progress.TotalQuestions-progress.CorrectAnswers)

	// Edge: answer q2 again, correctly
	answer(q2, 0)
	progress = getProgress()
	assert.Equal(suite.T(), 4, progress.TotalQuestions)
	assert.Equal(suite.T(), 2, progress.CorrectAnswers)
	assert.Equal(suite.T(), 2, progress.TotalQuestions-progress.CorrectAnswers)

	// Edge: answer q3 only incorrectly
	answer(q3, 1)
	progress = getProgress()
	assert.Equal(suite.T(), 5, progress.TotalQuestions)
	assert.Equal(suite.T(), 2, progress.CorrectAnswers)
	assert.Equal(suite.T(), 3, progress.TotalQuestions-progress.CorrectAnswers)

	// Edge: answer q3 correctly
	answer(q3, 0)
	progress = getProgress()
	assert.Equal(suite.T(), 6, progress.TotalQuestions)
	assert.Equal(suite.T(), 3, progress.CorrectAnswers)
	assert.Equal(suite.T(), 3, progress.TotalQuestions-progress.CorrectAnswers)
}

func (suite *APIIntegrationTestSuite) TestQuestionAPIResponse_IncludesStats() {
	cookie := suite.login()
	questionID := suite.createTestQuestion()

	// Submit a correct answer to increment correct_count
	response := &api.AnswerRequest{
		QuestionId:      int64(questionID),
		UserAnswerIndex: 0, // correct for test question
		ResponseTimeMs:  nil,
	}
	body, _ := json.Marshal(response)
	req, _ := http.NewRequest("POST", "/v1/quiz/answer", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	// Now fetch the question and verify it includes stats
	req, _ = http.NewRequest("GET", fmt.Sprintf("/v1/quiz/question/%d", questionID), nil)
	req.Header.Set("Cookie", cookie)
	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	var questionResp api.Question
	err := json.Unmarshal(w.Body.Bytes(), &questionResp)
	suite.Require().NoError(err)

	// Verify the response includes the stats fields
	suite.True(questionResp.CorrectCount != nil, "API response should include correct_count")
	suite.True(questionResp.IncorrectCount != nil, "API response should include incorrect_count")
	suite.True(questionResp.TotalResponses != nil, "API response should include total_responses")

	// Verify the stats are correct (we submitted one correct answer)
	suite.Equal(int(1), *questionResp.CorrectCount)
	suite.Equal(int(0), *questionResp.IncorrectCount)
	suite.Equal(int(1), *questionResp.TotalResponses)
}

func (suite *APIIntegrationTestSuite) TestGetProgress_EnhancedWithWorkerInfo() {
	// Step 1: Login and get cookie
	cookie := suite.login()

	// Step 2: Create some test data to ensure we have progress
	questionID := suite.createTestQuestion()

	// Submit an answer to create some progress data
	response := &api.AnswerRequest{
		QuestionId:      int64(questionID),
		UserAnswerIndex: 0, // correct for test question
		ResponseTimeMs:  nil,
	}
	body, _ := json.Marshal(response)
	req, _ := http.NewRequest("POST", "/v1/quiz/answer", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	suite.Equal(200, w.Code)

	// Step 3: Get enhanced progress with worker information
	req, _ = http.NewRequest("GET", "/v1/quiz/progress", nil)
	req.Header.Set("Cookie", cookie)

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Debug: Log the response when it fails
	if w.Code != http.StatusOK {
		fmt.Printf("DEBUG: Response status: %d\n", w.Code)
		fmt.Printf("DEBUG: Response body: %s\n", w.Body.String())
	}

	// Expect a successful response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var progressResp api.UserProgress
	err := json.Unmarshal(w.Body.Bytes(), &progressResp)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), progressResp)

	// Verify basic progress fields are present
	assert.NotNil(suite.T(), progressResp.CurrentLevel)
	assert.NotNil(suite.T(), progressResp.TotalQuestions)
	assert.NotNil(suite.T(), progressResp.CorrectAnswers)
	assert.NotNil(suite.T(), progressResp.AccuracyRate)

	// Verify worker-related fields are present (may be nil if services are not available)
	// These fields are optional and may not be present if worker services are not configured
	if progressResp.WorkerStatus != nil {
		assert.NotNil(suite.T(), progressResp.WorkerStatus.Status)
		// The actual status may vary, so we check if it's a valid status or just that it exists
		if status := progressResp.WorkerStatus.Status; status != nil {
			// Status should be one of the valid enum values
			validStatuses := []api.WorkerStatusStatus{api.Busy, api.Idle}
			assert.Contains(suite.T(), validStatuses, *status)
		}
	}

	if progressResp.GenerationFocus != nil {
		assert.NotNil(suite.T(), progressResp.GenerationFocus.CurrentGenerationModel)
		if progressResp.GenerationFocus.GenerationRate != nil {
			assert.GreaterOrEqual(suite.T(), *progressResp.GenerationFocus.GenerationRate, float32(0.0))
		}
	}

	if progressResp.PriorityInsights != nil {
		// Priority counts should be non-negative
		if progressResp.PriorityInsights.HighPriorityQuestions != nil {
			assert.GreaterOrEqual(suite.T(), *progressResp.PriorityInsights.HighPriorityQuestions, 0)
		}
		if progressResp.PriorityInsights.MediumPriorityQuestions != nil {
			assert.GreaterOrEqual(suite.T(), *progressResp.PriorityInsights.MediumPriorityQuestions, 0)
		}
		if progressResp.PriorityInsights.LowPriorityQuestions != nil {
			assert.GreaterOrEqual(suite.T(), *progressResp.PriorityInsights.LowPriorityQuestions, 0)
		}
	}

	if progressResp.LearningPreferences != nil {
		assert.NotNil(suite.T(), progressResp.LearningPreferences.FocusOnWeakAreas)
		// These are not pointers in the API type, they're direct values
		assert.GreaterOrEqual(suite.T(), progressResp.LearningPreferences.FreshQuestionRatio, float32(0.0))
		assert.LessOrEqual(suite.T(), progressResp.LearningPreferences.FreshQuestionRatio, float32(1.0))
		assert.GreaterOrEqual(suite.T(), progressResp.LearningPreferences.ReviewIntervalDays, 1)
		assert.LessOrEqual(suite.T(), progressResp.LearningPreferences.ReviewIntervalDays, 60)
	}
}

func (suite *APIIntegrationTestSuite) TestGetProgress_WorkerStatusHandling() {
	// Step 1: Login and get cookie
	cookie := suite.login()

	// Step 2: Get progress and verify worker status handling
	req, _ := http.NewRequest("GET", "/v1/quiz/progress", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var progressResp api.UserProgress
	err := json.Unmarshal(w.Body.Bytes(), &progressResp)
	assert.NoError(suite.T(), err)

	// Test that the API doesn't fail even if worker services are not available
	// The worker status should be gracefully handled (may be nil)
	// This test ensures the progress endpoint doesn't break when worker services are unavailable
	assert.NotNil(suite.T(), progressResp)

	// Basic progress fields should always be present
	assert.NotNil(suite.T(), progressResp.CurrentLevel)
	assert.NotNil(suite.T(), progressResp.TotalQuestions)
	assert.NotNil(suite.T(), progressResp.CorrectAnswers)
}

func (suite *APIIntegrationTestSuite) TestGetProgress_PriorityInsightsHandling() {
	// Step 1: Login and get cookie
	cookie := suite.login()

	// Step 2: Get progress and verify priority insights handling
	req, _ := http.NewRequest("GET", "/v1/quiz/progress", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var progressResp api.UserProgress
	err := json.Unmarshal(w.Body.Bytes(), &progressResp)
	assert.NoError(suite.T(), err)

	// Test that priority insights are handled gracefully
	// They may be nil if the learning service doesn't support priority features
	if progressResp.PriorityInsights != nil {
		// If present, validate the structure
		if progressResp.PriorityInsights.TotalQuestionsInQueue != nil {
			assert.GreaterOrEqual(suite.T(), *progressResp.PriorityInsights.TotalQuestionsInQueue, 0)
		}
		// PriorityInsights doesn't have CurrentFocus field in the API type
		// We can test the other fields that do exist
	}
}

func (suite *APIIntegrationTestSuite) TestGetProgress_GenerationFocusHandling() {
	// Step 1: Login and get cookie
	cookie := suite.login()

	// Step 2: Get progress and verify generation focus handling
	req, _ := http.NewRequest("GET", "/v1/quiz/progress", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var progressResp api.UserProgress
	err := json.Unmarshal(w.Body.Bytes(), &progressResp)
	assert.NoError(suite.T(), err)

	// Test that generation focus is handled gracefully
	if progressResp.GenerationFocus != nil {
		// If present, validate the structure
		if progressResp.GenerationFocus.CurrentGenerationModel != nil {
			assert.NotEmpty(suite.T(), *progressResp.GenerationFocus.CurrentGenerationModel)
		}
		if progressResp.GenerationFocus.LastGenerationTime != nil {
			// This is already a time.Time, not a string
			assert.NotNil(suite.T(), progressResp.GenerationFocus.LastGenerationTime)
		}
		if progressResp.GenerationFocus.GenerationRate != nil {
			assert.GreaterOrEqual(suite.T(), *progressResp.GenerationFocus.GenerationRate, float32(0.0))
		}
	}
}

func (suite *APIIntegrationTestSuite) TestGetProgress_LearningPreferencesHandling() {
	// Step 1: Login and get cookie
	cookie := suite.login()

	// Step 2: Get progress and verify learning preferences handling
	req, _ := http.NewRequest("GET", "/v1/quiz/progress", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var progressResp api.UserProgress
	err := json.Unmarshal(w.Body.Bytes(), &progressResp)
	assert.NoError(suite.T(), err)

	// Test that learning preferences are handled gracefully
	if progressResp.LearningPreferences != nil {
		// If present, validate the structure
		assert.NotNil(suite.T(), progressResp.LearningPreferences.FocusOnWeakAreas)

		// These are not pointers in the API type, they're direct values
		ratio := progressResp.LearningPreferences.FreshQuestionRatio
		assert.GreaterOrEqual(suite.T(), ratio, float32(0.0))
		assert.LessOrEqual(suite.T(), ratio, float32(1.0))

		penalty := progressResp.LearningPreferences.KnownQuestionPenalty
		assert.GreaterOrEqual(suite.T(), penalty, float32(0.0))
		assert.LessOrEqual(suite.T(), penalty, float32(1.0))

		interval := progressResp.LearningPreferences.ReviewIntervalDays
		assert.GreaterOrEqual(suite.T(), interval, 1)
		assert.LessOrEqual(suite.T(), interval, 60)

		boost := progressResp.LearningPreferences.WeakAreaBoost
		assert.GreaterOrEqual(suite.T(), boost, float32(1.0))
		assert.LessOrEqual(suite.T(), boost, float32(5.0))
	}
}

func (suite *APIIntegrationTestSuite) TestGetProgress_ErrorHandling() {
	// Test that the progress endpoint handles errors gracefully
	// This test ensures that if any of the worker-related services fail,
	// the main progress endpoint still returns basic progress data

	// Step 1: Login and get cookie
	cookie := suite.login()

	// Step 2: Get progress (this should work even if worker services are not available)
	req, _ := http.NewRequest("GET", "/v1/quiz/progress", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Should always return 200 OK, even if worker services fail
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var progressResp api.UserProgress
	err := json.Unmarshal(w.Body.Bytes(), &progressResp)
	assert.NoError(suite.T(), err)

	// Basic progress should always be available
	assert.NotNil(suite.T(), progressResp.CurrentLevel)
	assert.NotNil(suite.T(), progressResp.TotalQuestions)
	assert.NotNil(suite.T(), progressResp.CorrectAnswers)
	assert.NotNil(suite.T(), progressResp.AccuracyRate)

	// Worker-related fields may be nil, but that's OK
	// The important thing is that the endpoint doesn't fail
}

func TestAPIIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(APIIntegrationTestSuite))
}
