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
	"testing"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/handlers"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type FeedbackIntegrationTestSuite struct {
	suite.Suite
	Router      *gin.Engine
	db          *sql.DB
	testUser    *models.User
	testUserID  int
	cfg         *config.Config
	userService *services.UserService
}

func (suite *FeedbackIntegrationTestSuite) SetupSuite() {
	// Config
	cfg, err := config.NewConfig()
	require.NoError(suite.T(), err)
	suite.cfg = cfg

	// Database
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}
	suite.cfg.Database.URL = databaseURL

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := database.NewManager(logger)
	db, err := dbManager.InitDB(databaseURL)
	require.NoError(suite.T(), err)
	suite.db = db

	// Services
	userService := services.NewUserServiceWithLogger(db, suite.cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, suite.cfg, logger)

	mockAIService := handlers.NewMockAIService(suite.cfg, logger)
	workerService := services.NewWorkerServiceWithLogger(db, logger)
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)
	oauthService := services.NewOAuthServiceWithLogger(suite.cfg, logger)

	generationHintService := services.NewGenerationHintService(db, logger)
	storyService := services.NewStoryService(db, suite.cfg, logger)
	usageStatsService := services.NewUsageStatsService(suite.cfg, suite.db, logger)
	translationCacheRepo := services.NewTranslationCacheRepository(suite.db, logger)
	translationService := services.NewTranslationService(suite.cfg, usageStatsService, translationCacheRepo, logger)
	snippetsService := services.NewSnippetsService(db, suite.cfg, logger)
	conversationService := services.NewConversationService(db)

	suite.Router = handlers.NewRouter(
		suite.cfg,
		userService,
		questionService,
		learningService,
		mockAIService,
		workerService,
		dailyQuestionService,
		storyService,
		conversationService,
		oauthService,
		generationHintService,
		translationService,
		snippetsService,
		usageStatsService,
		logger,
	)

	suite.userService = userService
}

func (suite *FeedbackIntegrationTestSuite) TearDownSuite() {
	if suite.testUser != nil && suite.testUserID != 0 {
		suite.userService.DeleteUser(context.Background(), suite.testUserID)
	}
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *FeedbackIntegrationTestSuite) SetupTest() {
	// Clean database before each test
	services.CleanupTestDatabase(suite.db, suite.T())

	// Recreate test user
	createdUser, err := suite.userService.CreateUserWithPassword(context.Background(), "testuser_feedback", "testpass", "english", "A1")
	suite.Require().NoError(err)
	suite.testUserID = createdUser.ID
	suite.testUser = createdUser

	// Update user with required fields
	_, err = suite.db.Exec(`
		UPDATE users
		SET email = $1, timezone = $2, ai_provider = $3, ai_model = $4, last_active = $5
		WHERE id = $6
	`, "testuser_feedback@example.com", "UTC", "ollama", "llama3", time.Now(), suite.testUserID)
	suite.Require().NoError(err)
}

func (suite *FeedbackIntegrationTestSuite) login() string {
	loginReq := api.LoginRequest{
		Username: "testuser_feedback",
		Password: "testpass",
	}
	reqBody, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	require.Equal(suite.T(), http.StatusOK, w.Code, "Login should be successful")
	cookie := w.Result().Header.Get("Set-Cookie")
	require.NotEmpty(suite.T(), cookie, "Session cookie should be set")

	return cookie
}

func TestFeedbackIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(FeedbackIntegrationTestSuite))
}

func (suite *FeedbackIntegrationTestSuite) TestSubmitFeedback_Success() {
	// Login to get session cookie
	cookie := suite.login()

	// Create authenticated request
	w := httptest.NewRecorder()

	reqBody := map[string]interface{}{
		"feedback_text": "This is a test feedback",
		"feedback_type": "bug",
		"context_data": map[string]interface{}{
			"page_url":       "/quiz",
			"viewport_width": 1920,
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), response["id"])
	assert.Equal(suite.T(), "new", response["status"])
	assert.Equal(suite.T(), "This is a test feedback", response["feedback_text"])
}

func (suite *FeedbackIntegrationTestSuite) TestSubmitFeedback_WithScreenshot() {
	cookie := suite.login()
	w := httptest.NewRecorder()

	reqBody := map[string]interface{}{
		"feedback_text":   "Test with screenshot",
		"feedback_type":   "bug",
		"screenshot_data": "data:image/jpeg;base64,/9j/4AAQSkZJRg==",
		"context_data": map[string]interface{}{
			"page_url": "/quiz",
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response["screenshot_data"])
}

func (suite *FeedbackIntegrationTestSuite) TestSubmitFeedback_Unauthenticated() {
	w := httptest.NewRecorder()

	reqBody := map[string]interface{}{
		"feedback_text": "Test feedback",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *FeedbackIntegrationTestSuite) TestGetFeedbackList_AsAdmin() {
	// Submit some feedback first
	cookie := suite.login()
	w1 := httptest.NewRecorder()
	reqBody := map[string]interface{}{
		"feedback_text": "Test feedback 1",
		"feedback_type": "bug",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req1, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Cookie", cookie)
	suite.Router.ServeHTTP(w1, req1)

	// Create admin user
	admin, err := suite.userService.CreateUserWithPassword(context.Background(), "admin_feedback", "adminpass", "english", "A1")
	require.NoError(suite.T(), err)

	// Assign admin role
	err = suite.userService.AssignRoleByName(context.Background(), admin.ID, "admin")
	require.NoError(suite.T(), err)

	// Login as admin
	loginReq := api.LoginRequest{
		Username: "admin_feedback",
		Password: "adminpass",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginW := httptest.NewRecorder()
	loginReqHTTP, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqHTTP.Header.Set("Content-Type", "application/json")
	suite.Router.ServeHTTP(loginW, loginReqHTTP)
	adminCookie := loginW.Result().Header.Get("Set-Cookie")

	// Get feedback list as admin
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/feedback?page=1&page_size=20", nil)
	req.Header.Set("Cookie", adminCookie)

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), response, "items")
	assert.Contains(suite.T(), response, "total")
	assert.Greater(suite.T(), response["total"], float64(0))
}

func (suite *FeedbackIntegrationTestSuite) TestGetFeedbackList_WithFilters() {
	// Login to get session cookie
	cookie := suite.login()

	// Submit feedback with different types
	for i, ftype := range []string{"bug", "feature_request", "general"} {
		w := httptest.NewRecorder()
		reqBody := map[string]interface{}{
			"feedback_text": fmt.Sprintf("Test feedback %d", i),
			"feedback_type": ftype,
		}
		jsonBody, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", cookie)
		suite.Router.ServeHTTP(w, req)
	}

	// Create admin user and login
	admin, err := suite.userService.CreateUserWithPassword(context.Background(), "admin_filter", "adminpass", "english", "A1")
	require.NoError(suite.T(), err)

	// Mark as admin
	_, err = suite.db.Exec("UPDATE users SET is_admin = true WHERE id = $1", admin.ID)
	require.NoError(suite.T(), err)

	// Login as admin
	loginReq := api.LoginRequest{
		Username: "admin_filter",
		Password: "adminpass",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginW := httptest.NewRecorder()
	loginReqHTTP, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqHTTP.Header.Set("Content-Type", "application/json")
	suite.Router.ServeHTTP(loginW, loginReqHTTP)
	adminCookie := loginW.Result().Header.Get("Set-Cookie")

	// Get feedback list filtered by type
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/feedback?page=1&page_size=20&feedback_type=bug", nil)
	req.Header.Set("Cookie", adminCookie)

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	items := response["items"].([]interface{})
	// All items should be bugs
	for _, item := range items {
		feedback := item.(map[string]interface{})
		assert.Equal(suite.T(), "bug", feedback["feedback_type"])
	}
}

func (suite *FeedbackIntegrationTestSuite) TestUpdateFeedback_AsAdmin() {
	// Submit feedback
	cookie := suite.login()
	w1 := httptest.NewRecorder()
	reqBody := map[string]interface{}{
		"feedback_text": "Test feedback for update",
		"feedback_type": "bug",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req1, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Cookie", cookie)
	suite.Router.ServeHTTP(w1, req1)

	var createResponse map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &createResponse)
	require.NoError(suite.T(), err)
	feedbackID := int(createResponse["id"].(float64))

	// Create admin user and login
	admin, err := suite.userService.CreateUserWithPassword(context.Background(), "admin_update", "adminpass", "english", "A1")
	require.NoError(suite.T(), err)

	// Mark as admin
	_, err = suite.db.Exec("UPDATE users SET is_admin = true WHERE id = $1", admin.ID)
	require.NoError(suite.T(), err)

	// Login as admin
	loginReq := api.LoginRequest{
		Username: "admin_update",
		Password: "adminpass",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginW := httptest.NewRecorder()
	loginReqHTTP, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqHTTP.Header.Set("Content-Type", "application/json")
	suite.Router.ServeHTTP(loginW, loginReqHTTP)
	adminCookie := loginW.Result().Header.Get("Set-Cookie")

	// Update feedback as admin
	w := httptest.NewRecorder()
	updateBody := map[string]interface{}{
		"status":      "in_progress",
		"admin_notes": "Working on this issue",
	}
	updateJsonBody, _ := json.Marshal(updateBody)
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/v1/admin/backend/feedback/%d", feedbackID), bytes.NewBuffer(updateJsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", adminCookie)

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "in_progress", response["status"])
	assert.Equal(suite.T(), "Working on this issue", response["admin_notes"])
}

func (suite *FeedbackIntegrationTestSuite) TestUpdateFeedback_NonAdmin() {
	// Submit feedback
	cookie := suite.login()
	w1 := httptest.NewRecorder()
	reqBody := map[string]interface{}{
		"feedback_text": "Test feedback",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req1, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Cookie", cookie)
	suite.Router.ServeHTTP(w1, req1)

	var createResponse map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &createResponse)
	require.NoError(suite.T(), err)
	feedbackID := int(createResponse["id"].(float64))

	// Try to update as non-admin
	w := httptest.NewRecorder()
	updateBody := map[string]interface{}{
		"status": "resolved",
	}
	updateJsonBody, _ := json.Marshal(updateBody)
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/v1/admin/backend/feedback/%d", feedbackID), bytes.NewBuffer(updateJsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}
