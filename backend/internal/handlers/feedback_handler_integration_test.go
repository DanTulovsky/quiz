//go:build integration

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
	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/handlers"
	"quizapp/internal/middleware"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
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
	authAPIKeyService := services.NewAuthAPIKeyService(db, logger)

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
		services.NewWordOfTheDayService(db, logger),
		authAPIKeyService,
		services.NewTranslationPracticeService(db, storyService, questionService, suite.cfg, logger),
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
	sessionCookie := w.Result().Header.Get("Set-Cookie")
	require.NotEmpty(suite.T(), sessionCookie, "Session cookie should be set")

	return sessionCookie
}

func TestFeedbackIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(FeedbackIntegrationTestSuite))
}

func (suite *FeedbackIntegrationTestSuite) TestSubmitFeedback_Success() {
	// Login to get session cookie
	sessionCookie := suite.login()

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
	req.Header.Set("Cookie", sessionCookie)

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
	sessionCookie := suite.login()
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
	req.Header.Set("Cookie", sessionCookie)

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
	sessionCookie := suite.login()
	w1 := httptest.NewRecorder()
	reqBody := map[string]interface{}{
		"feedback_text": "Test feedback 1",
		"feedback_type": "bug",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req1, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Cookie", sessionCookie)
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
	sessionCookie := suite.login()

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
		req.Header.Set("Cookie", sessionCookie)
		suite.Router.ServeHTTP(w, req)
	}

	// Create admin user and login
	admin, err := suite.userService.CreateUserWithPassword(context.Background(), "admin_filter", "adminpass", "english", "A1")
	require.NoError(suite.T(), err)

	// Assign admin role
	err = suite.userService.AssignRoleByName(context.Background(), admin.ID, "admin")
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
	sessionCookie := suite.login()
	w1 := httptest.NewRecorder()
	reqBody := map[string]interface{}{
		"feedback_text": "Test feedback for update",
		"feedback_type": "bug",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req1, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Cookie", sessionCookie)
	suite.Router.ServeHTTP(w1, req1)

	var createResponse map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &createResponse)
	require.NoError(suite.T(), err)
	feedbackID := int(createResponse["id"].(float64))

	// Create admin user and login
	admin, err := suite.userService.CreateUserWithPassword(context.Background(), "admin_update", "adminpass", "english", "A1")
	require.NoError(suite.T(), err)

	// Assign admin role
	err = suite.userService.AssignRoleByName(context.Background(), admin.ID, "admin")
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
	sessionCookie := suite.login()
	w1 := httptest.NewRecorder()
	reqBody := map[string]interface{}{
		"feedback_text": "Test feedback",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req1, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Cookie", sessionCookie)
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
	req.Header.Set("Cookie", sessionCookie)

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

func (suite *FeedbackIntegrationTestSuite) TestSubmitFeedback_InvalidJSON_Returns400() {
	// Login to get session cookie
	sessionCookie := suite.login()

	// Test with invalid JSON - number instead of string for feedback_text
	w := httptest.NewRecorder()
	reqBody := map[string]interface{}{
		"feedback_text": 789, // Should be a string, not a number
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", sessionCookie)

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code, "Should return 400 for invalid JSON type")
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "INVALID_INPUT", response["code"])
}

func (suite *FeedbackIntegrationTestSuite) TestSubmitFeedback_MissingRequiredField_Returns400() {
	// Login to get session cookie
	sessionCookie := suite.login()

	// Test with missing required feedback_text field
	w := httptest.NewRecorder()
	reqBody := map[string]interface{}{
		"feedback_type": "bug",
		// feedback_text is missing
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", sessionCookie)

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code, "Should return 400 for missing required field")
}

func (suite *FeedbackIntegrationTestSuite) TestDeleteFeedbackByStatus_MissingStatus_Returns400() {
	// Create admin user and login
	admin, err := suite.userService.CreateUserWithPassword(context.Background(), "admin_delete", "adminpass", "english", "A1")
	require.NoError(suite.T(), err)

	// Assign admin role
	err = suite.userService.AssignRoleByName(context.Background(), admin.ID, "admin")
	require.NoError(suite.T(), err)

	// Login as admin
	loginReq := api.LoginRequest{
		Username: "admin_delete",
		Password: "adminpass",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginW := httptest.NewRecorder()
	loginReqHTTP, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqHTTP.Header.Set("Content-Type", "application/json")
	suite.Router.ServeHTTP(loginW, loginReqHTTP)
	adminCookie := loginW.Result().Header.Get("Set-Cookie")

	// Try to delete without status parameter
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/v1/admin/backend/feedback", nil)
	req.Header.Set("Cookie", adminCookie)

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code, "Should return 400 for missing status parameter")
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "MISSING_REQUIRED_FIELD", response["code"])
}

func (suite *FeedbackIntegrationTestSuite) TestDeleteFeedbackByStatus_WithStatus_Returns200() {
	// Submit some feedback with resolved status
	sessionCookie := suite.login()

	// Create feedback service to directly set status
	feedbackService := services.NewFeedbackService(suite.db, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	w1 := httptest.NewRecorder()
	reqBody := map[string]interface{}{
		"feedback_text": "Test feedback for deletion",
		"feedback_type": "bug",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req1, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Cookie", sessionCookie)
	suite.Router.ServeHTTP(w1, req1)

	var createResponse map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &createResponse)
	require.NoError(suite.T(), err)
	feedbackID := int(createResponse["id"].(float64))

	// Update status to resolved
	_, err = feedbackService.UpdateFeedback(context.Background(), feedbackID, map[string]interface{}{
		"status": "resolved",
	})
	require.NoError(suite.T(), err)

	// Create admin user and login
	admin, err := suite.userService.CreateUserWithPassword(context.Background(), "admin_delete2", "adminpass", "english", "A1")
	require.NoError(suite.T(), err)

	// Assign admin role
	err = suite.userService.AssignRoleByName(context.Background(), admin.ID, "admin")
	require.NoError(suite.T(), err)

	// Login as admin
	loginReq := api.LoginRequest{
		Username: "admin_delete2",
		Password: "adminpass",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginW := httptest.NewRecorder()
	loginReqHTTP, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqHTTP.Header.Set("Content-Type", "application/json")
	suite.Router.ServeHTTP(loginW, loginReqHTTP)
	adminCookie := loginW.Result().Header.Get("Set-Cookie")

	// Delete feedback by status
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/v1/admin/backend/feedback?status=resolved", nil)
	req.Header.Set("Cookie", adminCookie)

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code, "Should return 200 when deleting feedback by status")
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), response, "deleted_count")
	assert.GreaterOrEqual(suite.T(), response["deleted_count"], float64(1))
}

func (suite *FeedbackIntegrationTestSuite) TestGetFeedback_ByID_Success() {
	// Submit feedback
	sessionCookie := suite.login()
	w1 := httptest.NewRecorder()
	reqBody := map[string]interface{}{
		"feedback_text": "Test feedback for get",
		"feedback_type": "bug",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req1, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Cookie", sessionCookie)
	suite.Router.ServeHTTP(w1, req1)

	var createResponse map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &createResponse)
	require.NoError(suite.T(), err)
	feedbackID := int(createResponse["id"].(float64))

	// Create admin user and login
	admin, err := suite.userService.CreateUserWithPassword(context.Background(), "admin_get", "adminpass", "english", "A1")
	require.NoError(suite.T(), err)

	// Assign admin role
	err = suite.userService.AssignRoleByName(context.Background(), admin.ID, "admin")
	require.NoError(suite.T(), err)

	// Login as admin
	loginReq := api.LoginRequest{
		Username: "admin_get",
		Password: "adminpass",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginW := httptest.NewRecorder()
	loginReqHTTP, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqHTTP.Header.Set("Content-Type", "application/json")
	suite.Router.ServeHTTP(loginW, loginReqHTTP)
	adminCookie := loginW.Result().Header.Get("Set-Cookie")

	// Get feedback by ID
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/admin/backend/feedback/%d", feedbackID), nil)
	req.Header.Set("Cookie", adminCookie)

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code, "Should return 200 for valid feedback ID")
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), float64(feedbackID), response["id"])
	assert.Equal(suite.T(), "Test feedback for get", response["feedback_text"])
}

func (suite *FeedbackIntegrationTestSuite) TestGetFeedback_ByID_NotFound_Returns404() {
	// Create admin user and login
	admin, err := suite.userService.CreateUserWithPassword(context.Background(), "admin_get2", "adminpass", "english", "A1")
	require.NoError(suite.T(), err)

	// Assign admin role
	err = suite.userService.AssignRoleByName(context.Background(), admin.ID, "admin")
	require.NoError(suite.T(), err)

	// Login as admin
	loginReq := api.LoginRequest{
		Username: "admin_get2",
		Password: "adminpass",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginW := httptest.NewRecorder()
	loginReqHTTP, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqHTTP.Header.Set("Content-Type", "application/json")
	suite.Router.ServeHTTP(loginW, loginReqHTTP)
	adminCookie := loginW.Result().Header.Get("Set-Cookie")

	// Try to get non-existent feedback
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/feedback/999999", nil)
	req.Header.Set("Cookie", adminCookie)

	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code, "Should return 404 for non-existent feedback ID")
}

func (suite *FeedbackIntegrationTestSuite) TestCreateLinearIssue_Success() {
	// Submit feedback first
	sessionCookie := suite.login()
	w1 := httptest.NewRecorder()
	reqBody := map[string]interface{}{
		"feedback_text": "Test feedback for Linear issue",
		"feedback_type": "bug",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req1, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Cookie", sessionCookie)
	suite.Router.ServeHTTP(w1, req1)

	var createResponse map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &createResponse)
	require.NoError(suite.T(), err)
	feedbackID := int(createResponse["id"].(float64))

	// Create admin user and login
	admin, err := suite.userService.CreateUserWithPassword(context.Background(), "admin_linear", "adminpass", "english", "A1")
	require.NoError(suite.T(), err)

	// Assign admin role
	err = suite.userService.AssignRoleByName(context.Background(), admin.ID, "admin")
	require.NoError(suite.T(), err)

	// Login as admin
	loginReq := api.LoginRequest{
		Username: "admin_linear",
		Password: "adminpass",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginW := httptest.NewRecorder()
	loginReqHTTP, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqHTTP.Header.Set("Content-Type", "application/json")
	suite.Router.ServeHTTP(loginW, loginReqHTTP)
	adminCookie := loginW.Result().Header.Get("Set-Cookie")

	// Mock Linear API server - handles team lookup, project lookup, and issue creation
	requestCount := 0
	mockLinearServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(suite.T(), "POST", r.Method)
		assert.Equal(suite.T(), "application/json", r.Header.Get("Content-Type"))
		assert.NotEmpty(suite.T(), r.Header.Get("Authorization"))

		// Parse request body
		var graphQLReq map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&graphQLReq)
		require.NoError(suite.T(), err)

		assert.Contains(suite.T(), graphQLReq, "query")
		query := graphQLReq["query"].(string)

		// Handle different GraphQL queries/mutations
		if strings.Contains(query, "query Teams") {
			// Team lookup query
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"teams": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id":   "test-team-uuid-123",
								"name": "test-team-id",
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		} else if strings.Contains(query, "query Projects") {
			// Project lookup query
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"team": map[string]interface{}{
						"projects": map[string]interface{}{
							"nodes": []map[string]interface{}{
								{
									"id":   "test-project-uuid-123",
									"name": "test-project-id",
								},
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		} else if strings.Contains(query, "query Labels") || strings.Contains(query, "query TeamLabels") || strings.Contains(query, "query ProjectLabels") {
			// Label lookup query (organization, team, or project labels)
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"organization": map[string]interface{}{
						"labels": map[string]interface{}{
							"nodes": []map[string]interface{}{
								{
									"id":   "label-bug-uuid-123",
									"name": "Bug",
								},
								{
									"id":   "label-feature-uuid-123",
									"name": "Feature",
								},
								{
									"id":   "label-improvement-uuid-123",
									"name": "Improvement",
								},
							},
						},
					},
				},
			}
			// Handle team labels response structure
			if strings.Contains(query, "query TeamLabels") {
				response = map[string]interface{}{
					"data": map[string]interface{}{
						"team": map[string]interface{}{
							"labels": map[string]interface{}{
								"nodes": []map[string]interface{}{
									{
										"id":   "label-bug-uuid-123",
										"name": "Bug",
									},
								},
							},
						},
					},
				}
			}
			// Handle project labels response structure
			if strings.Contains(query, "query ProjectLabels") {
				response = map[string]interface{}{
					"data": map[string]interface{}{
						"project": map[string]interface{}{
							"labels": map[string]interface{}{
								"nodes": []map[string]interface{}{
									{
										"id":   "label-bug-uuid-123",
										"name": "Bug",
									},
								},
							},
						},
					},
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		} else if strings.Contains(query, "mutation IssueCreate") {
			// Issue creation mutation
			assert.Contains(suite.T(), graphQLReq, "variables")
			variables := graphQLReq["variables"].(map[string]interface{})
			input := variables["input"].(map[string]interface{})

			// Verify required fields
			assert.NotEmpty(suite.T(), input["title"])
			assert.NotEmpty(suite.T(), input["teamId"])

			// Return successful Linear response
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"issueCreate": map[string]interface{}{
						"success": true,
						"issue": map[string]interface{}{
							"id":    "linear-issue-123",
							"title": input["title"].(string),
							"url":   "https://linear.app/issue/linear-issue-123",
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		} else {
			// Unknown query type
			suite.T().Fatalf("Unexpected GraphQL query: %s", query)
		}
		requestCount++
	}))
	defer mockLinearServer.Close()

	// Update config to enable Linear and point to mock server
	suite.cfg.Linear.Enabled = true
	suite.cfg.Linear.APIKey = "test-api-key"
	suite.cfg.Linear.TeamID = "test-team-id"
	suite.cfg.Linear.ProjectID = "test-project-id"

	// Create Linear service with mock URL
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	linearService := services.NewLinearServiceWithURL(suite.cfg, logger, mockLinearServer.URL)

	// Create feedback handler with mock Linear service
	feedbackService := services.NewFeedbackService(suite.db, logger)
	feedbackHandler := handlers.NewFeedbackHandler(feedbackService, linearService, suite.userService, suite.cfg, logger)

	// Create test router with proper middleware
	testRouter := gin.New()
	// Setup session middleware (same as in NewRouter)
	store := cookie.NewStore([]byte(suite.cfg.Server.SessionSecret))
	testRouter.Use(sessions.Sessions(config.SessionName, store))

	adminGroup := testRouter.Group("/v1/admin/backend")
	adminGroup.Use(middleware.RequireAdmin(suite.userService))
	adminGroup.POST("/feedback/:id/linear-issue", feedbackHandler.CreateLinearIssue)

	// Test the endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/admin/backend/feedback/%d/linear-issue", feedbackID), nil)
	req.Header.Set("Cookie", adminCookie)

	testRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code, "Should return 200 for successful Linear issue creation")
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "linear-issue-123", response["issue_id"])
	assert.Contains(suite.T(), response["issue_url"], "linear.app")
	assert.NotEmpty(suite.T(), response["title"])
}

func (suite *FeedbackIntegrationTestSuite) TestCreateLinearIssue_LinearDisabled() {
	// Submit feedback first
	sessionCookie := suite.login()
	w1 := httptest.NewRecorder()
	reqBody := map[string]interface{}{
		"feedback_text": "Test feedback",
		"feedback_type": "bug",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req1, _ := http.NewRequest("POST", "/v1/feedback", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Cookie", sessionCookie)
	suite.Router.ServeHTTP(w1, req1)

	var createResponse map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &createResponse)
	require.NoError(suite.T(), err)
	feedbackID := int(createResponse["id"].(float64))

	// Create admin user and login
	admin, err := suite.userService.CreateUserWithPassword(context.Background(), "admin_linear2", "adminpass", "english", "A1")
	require.NoError(suite.T(), err)

	// Assign admin role
	err = suite.userService.AssignRoleByName(context.Background(), admin.ID, "admin")
	require.NoError(suite.T(), err)

	// Login as admin
	loginReq := api.LoginRequest{
		Username: "admin_linear2",
		Password: "adminpass",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginW := httptest.NewRecorder()
	loginReqHTTP, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqHTTP.Header.Set("Content-Type", "application/json")
	suite.Router.ServeHTTP(loginW, loginReqHTTP)
	adminCookie := loginW.Result().Header.Get("Set-Cookie")

	// Disable Linear in config
	suite.cfg.Linear.Enabled = false

	// Create Linear service (disabled)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	linearService := services.NewLinearService(suite.cfg, logger)

	// Create feedback handler
	feedbackService := services.NewFeedbackService(suite.db, logger)
	feedbackHandler := handlers.NewFeedbackHandler(feedbackService, linearService, suite.userService, suite.cfg, logger)

	// Create test router
	testRouter := gin.New()
	adminGroup := testRouter.Group("/v1/admin/backend")
	adminGroup.Use(func(c *gin.Context) {
		c.Set("user_id", admin.ID)
		c.Set("username", admin.Username)
		c.Set("is_admin", true)
	})
	adminGroup.POST("/feedback/:id/linear-issue", feedbackHandler.CreateLinearIssue)

	// Test the endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/admin/backend/feedback/%d/linear-issue", feedbackID), nil)
	req.Header.Set("Cookie", adminCookie)

	testRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusServiceUnavailable, w.Code, "Should return 503 when Linear is disabled")
}

func (suite *FeedbackIntegrationTestSuite) TestCreateLinearIssue_NotFound() {
	// Create admin user and login
	admin, err := suite.userService.CreateUserWithPassword(context.Background(), "admin_linear3", "adminpass", "english", "A1")
	require.NoError(suite.T(), err)

	// Assign admin role
	err = suite.userService.AssignRoleByName(context.Background(), admin.ID, "admin")
	require.NoError(suite.T(), err)

	// Login as admin
	loginReq := api.LoginRequest{
		Username: "admin_linear3",
		Password: "adminpass",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginW := httptest.NewRecorder()
	loginReqHTTP, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqHTTP.Header.Set("Content-Type", "application/json")
	suite.Router.ServeHTTP(loginW, loginReqHTTP)
	adminCookie := loginW.Result().Header.Get("Set-Cookie")

	// Mock Linear API server
	mockLinearServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"issueCreate": map[string]interface{}{
					"success": true,
					"issue": map[string]interface{}{
						"id":    "linear-issue-123",
						"title": "Test Issue",
						"url":   "https://linear.app/issue/linear-issue-123",
					},
				},
			},
		})
	}))
	defer mockLinearServer.Close()

	// Update config
	suite.cfg.Linear.Enabled = true
	suite.cfg.Linear.APIKey = "test-api-key"
	suite.cfg.Linear.TeamID = "test-team-id"

	// Create Linear service with mock URL
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	linearService := services.NewLinearServiceWithURL(suite.cfg, logger, mockLinearServer.URL)

	// Create feedback handler
	feedbackService := services.NewFeedbackService(suite.db, logger)
	feedbackHandler := handlers.NewFeedbackHandler(feedbackService, linearService, suite.userService, suite.cfg, logger)

	// Create test router with proper middleware
	testRouter := gin.New()
	// Setup session middleware (same as in NewRouter)
	store := cookie.NewStore([]byte(suite.cfg.Server.SessionSecret))
	testRouter.Use(sessions.Sessions(config.SessionName, store))

	adminGroup := testRouter.Group("/v1/admin/backend")
	adminGroup.Use(middleware.RequireAdmin(suite.userService))
	adminGroup.POST("/feedback/:id/linear-issue", feedbackHandler.CreateLinearIssue)

	// Test with non-existent feedback ID
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/admin/backend/feedback/999999/linear-issue", nil)
	req.Header.Set("Cookie", adminCookie)

	testRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code, "Should return 404 for non-existent feedback ID")
}
