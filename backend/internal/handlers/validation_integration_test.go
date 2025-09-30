//go:build integration
// +build integration

package handlers_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"quizapp/internal/database"
	"quizapp/internal/handlers"
	"quizapp/internal/services"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ValidationIntegrationTestSuite struct {
	suite.Suite
	Router      *gin.Engine
	db          *sql.DB
	userService *services.UserService
	cfg         *config.Config
}

func (suite *ValidationIntegrationTestSuite) SetupSuite() {
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

	// Load the real config
	cfg, err := config.NewConfig()
	suite.Require().NoError(err)
	suite.cfg = cfg

	// Initialize services
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	suite.userService = userService
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	aiService := services.NewAIService(cfg, logger)
	workerService := services.NewWorkerServiceWithLogger(db, logger)
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)
	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Use the new router factory
	generationHintService := services.NewGenerationHintService(db, logger)
	router := handlers.NewRouter(cfg, userService, questionService, learningService, aiService, workerService, dailyQuestionService, oauthService, generationHintService, logger)
	suite.Router = router
}

func (suite *ValidationIntegrationTestSuite) SetupTest() {
	// Use shared database setup for clean state
	suite.db = services.SharedTestDBSetup(suite.T())
}

func (suite *ValidationIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *ValidationIntegrationTestSuite) TestUndocumentedAPI_Returns404() {
	// Test various undocumented endpoints
	testCases := []struct {
		name     string
		method   string
		path     string
		expected int
	}{
		{
			name:     "Undocumented GET endpoint",
			method:   "GET",
			path:     "/v1/undocumented-endpoint",
			expected: http.StatusNotFound,
		},
		{
			name:     "Undocumented POST endpoint",
			method:   "POST",
			path:     "/v1/undocumented-endpoint",
			expected: http.StatusNotFound,
		},
		{
			name:     "Undocumented PUT endpoint",
			method:   "PUT",
			path:     "/v1/undocumented-endpoint",
			expected: http.StatusNotFound,
		},
		{
			name:     "Undocumented DELETE endpoint",
			method:   "DELETE",
			path:     "/v1/undocumented-endpoint",
			expected: http.StatusNotFound,
		},
		{
			name:     "Undocumented admin endpoint",
			method:   "GET",
			path:     "/v1/admin/backend/undocumented",
			expected: http.StatusNotFound,
		},
		{
			name:     "Undocumented nested endpoint",
			method:   "POST",
			path:     "/v1/quiz/undocumented/nested/endpoint",
			expected: http.StatusNotFound,
		},
		{
			name:     "Undocumented endpoint with parameters",
			method:   "GET",
			path:     "/v1/admin/backend/userz/999/undocumented",
			expected: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			req, err := http.NewRequest(tc.method, tc.path, nil)
			suite.Require().NoError(err)

			w := httptest.NewRecorder()
			suite.Router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tc.expected, w.Code)

			// Check that the response contains the expected error message
			responseBody := w.Body.String()
			assert.Contains(suite.T(), responseBody, "Not found")
		})
	}
}

func (suite *ValidationIntegrationTestSuite) TestDocumentedAPI_Returns200() {
	// Test that documented endpoints still work
	testCases := []struct {
		name     string
		method   string
		path     string
		expected int
	}{
		{
			name:     "Documented version endpoint",
			method:   "GET",
			path:     "/v1/version",
			expected: http.StatusOK,
		},
		{
			name:     "Documented health endpoint",
			method:   "GET",
			path:     "/health",
			expected: http.StatusOK,
		},
		{
			name:     "Documented languages endpoint",
			method:   "GET",
			path:     "/v1/settings/languages",
			expected: http.StatusOK,
		},
		{
			name:     "Documented levels endpoint",
			method:   "GET",
			path:     "/v1/settings/levels",
			expected: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			req, err := http.NewRequest(tc.method, tc.path, nil)
			suite.Require().NoError(err)

			w := httptest.NewRecorder()
			suite.Router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tc.expected, w.Code)
		})
	}
}

func (suite *ValidationIntegrationTestSuite) TestProtectedEndpoints_StillRequireAuth() {
	// Test that protected endpoints still require authentication
	testCases := []struct {
		name     string
		method   string
		path     string
		expected int
	}{
		{
			name:     "Protected admin endpoint without auth",
			method:   "GET",
			path:     "/v1/admin/backend/userz",
			expected: http.StatusUnauthorized, // Should be 401, not 404
		},

		{
			name:     "Protected settings endpoint without auth",
			method:   "GET",
			path:     "/v1/settings/ai-providers",
			expected: http.StatusUnauthorized, // Should be 401, not 404
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			req, err := http.NewRequest(tc.method, tc.path, nil)
			suite.Require().NoError(err)

			w := httptest.NewRecorder()
			suite.Router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tc.expected, w.Code)

			// These should not return "Endpoint not found" since they are documented
			responseBody := w.Body.String()
			assert.NotContains(suite.T(), responseBody, "Endpoint not found")
		})
	}
}

func (suite *ValidationIntegrationTestSuite) TestNonV1Endpoints_NotBlocked() {
	// Test that non-v1 endpoints are not blocked by the middleware
	testCases := []struct {
		name     string
		method   string
		path     string
		expected int
	}{
		{
			name:     "Configz endpoint",
			method:   "GET",
			path:     "/configz",
			expected: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			req, err := http.NewRequest(tc.method, tc.path, nil)
			suite.Require().NoError(err)

			w := httptest.NewRecorder()
			suite.Router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tc.expected, w.Code)
		})
	}
}

func (suite *ValidationIntegrationTestSuite) TestMiddleware_LogsUndocumentedCalls() {
	// This test verifies that the middleware logs undocumented calls
	// We can't easily test the logging in integration tests, but we can verify
	// that the middleware doesn't interfere with normal operation

	req, err := http.NewRequest("GET", "/v1/undocumented-endpoint", nil)
	suite.Require().NoError(err)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Should return 404 with proper error message
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	responseBody := w.Body.String()
	assert.Contains(suite.T(), responseBody, "Not found")
}

func (suite *ValidationIntegrationTestSuite) TestRequestBodyValidation() {
	// Test request body validation for various endpoints
	testCases := []struct {
		name           string
		method         string
		path           string
		requestBody    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid login request",
			method:         "POST",
			path:           "/v1/auth/login",
			requestBody:    `{"username": "testuser", "password": "password123"}`,
			expectedStatus: http.StatusUnauthorized, // 401 because user doesn't exist, but validation should pass
		},
		{
			name:           "Invalid login request - missing username",
			method:         "POST",
			path:           "/v1/auth/login",
			requestBody:    `{"password": "password123"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request data",
		},
		{
			name:           "Invalid login request - missing password",
			method:         "POST",
			path:           "/v1/auth/login",
			requestBody:    `{"username": "testuser"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request data",
		},
		{
			name:           "Invalid login request - username too short",
			method:         "POST",
			path:           "/v1/auth/login",
			requestBody:    `{"username": "", "password": "password123"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request data",
		},
		{
			name:           "Invalid login request - password too short",
			method:         "POST",
			path:           "/v1/auth/login",
			requestBody:    `{"username": "testuser", "password": "123"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request data",
		},
		{
			name:           "Invalid login request - invalid username pattern",
			method:         "POST",
			path:           "/v1/auth/login",
			requestBody:    `{"username": "test-user", "password": "password123"}`,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid credentials",
		},
		{
			name:           "Invalid login request - null username",
			method:         "POST",
			path:           "/v1/auth/login",
			requestBody:    `{"username": null, "password": "123"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request data",
		},
		{
			name:           "Valid signup request",
			method:         "POST",
			path:           "/v1/auth/signup",
			requestBody:    `{"username": "newuser", "email": "test@example.com", "password": "password123"}`,
			expectedStatus: http.StatusCreated, // 201 for successful signup
		},
		{
			name:           "Invalid signup request - missing required fields",
			method:         "POST",
			path:           "/v1/auth/signup",
			requestBody:    `{"username": "newuser"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request data",
		},
		{
			name:           "Valid settings update request",
			method:         "PUT",
			path:           "/v1/settings",
			requestBody:    `{"language": "italian", "level": "A1"}`,
			expectedStatus: http.StatusUnauthorized, // 401 because not authenticated, but validation should pass
		},
		{
			name:           "Invalid settings update request - invalid language",
			method:         "PUT",
			path:           "/v1/settings",
			requestBody:    `{"language": "invalid_language", "level": "A1"}`,
			expectedStatus: http.StatusUnauthorized, // 401 because not authenticated, validation happens after auth
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			req, err := http.NewRequest(tc.method, tc.path, strings.NewReader(tc.requestBody))
			suite.Require().NoError(err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			suite.Router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tc.expectedStatus, w.Code, "Expected status code %d, got %d", tc.expectedStatus, w.Code)

			responseBody := w.Body.String()
			if tc.expectedError != "" {
				assert.Contains(suite.T(), responseBody, tc.expectedError, "Response should contain error message: %s", tc.expectedError)
			}
		})
	}
}

func (suite *ValidationIntegrationTestSuite) TestRequestBodyValidation_NonJSONRequests() {
	// Test that non-JSON requests are handled properly
	testCases := []struct {
		name           string
		method         string
		path           string
		contentType    string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "Non-JSON request to POST endpoint",
			method:         "POST",
			path:           "/v1/auth/login",
			contentType:    "text/plain",
			requestBody:    "not json",
			expectedStatus: http.StatusBadRequest, // Should fail JSON parsing
		},
		{
			name:           "GET request with no body",
			method:         "GET",
			path:           "/v1/auth/status",
			contentType:    "",
			requestBody:    "",
			expectedStatus: http.StatusOK, // Should work fine
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			req, err := http.NewRequest(tc.method, tc.path, strings.NewReader(tc.requestBody))
			suite.Require().NoError(err)
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}

			w := httptest.NewRecorder()
			suite.Router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tc.expectedStatus, w.Code, "Expected status code %d, got %d", tc.expectedStatus, w.Code)
		})
	}
}

func TestValidationIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ValidationIntegrationTestSuite))
}
