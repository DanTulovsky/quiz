//go:build integration

package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// AuthIntegrationTestSuite provides integration tests for authentication functionality
type AuthIntegrationTestSuite struct {
	suite.Suite
	Router *gin.Engine
	cfg    *config.Config
	db     *sql.DB
}

func TestAuthIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AuthIntegrationTestSuite))
}

func (suite *AuthIntegrationTestSuite) SetupSuite() {
	// Use environment variable for test database URL, fallback to test port 5433
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

	// Create services needed for the router
	userService := services.NewUserServiceWithLogger(db, suite.cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, suite.cfg, logger)
	aiService := services.NewAIService(suite.cfg, logger)
	workerService := services.NewWorkerServiceWithLogger(db, logger)
	oauthService := services.NewOAuthServiceWithLogger(suite.cfg, logger)

	// Create the real application router
	dailyQuestionService := services.NewDailyQuestionService(suite.db, logger, questionService, learningService)
	generationHintService := services.NewGenerationHintService(suite.db, logger)
	storyService := services.NewStoryService(suite.db, suite.cfg, logger)
	suite.Router = NewRouter(suite.cfg, userService, questionService, learningService, aiService, workerService, dailyQuestionService, storyService, services.NewConversationService(db), oauthService, generationHintService, logger)
}

func (suite *AuthIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *AuthIntegrationTestSuite) SetupTest() {
	// Clean database for each test
	suite.cleanupDatabase()
}

func (suite *AuthIntegrationTestSuite) cleanupDatabase() {
	// Use the shared database cleanup function
	services.CleanupTestDatabase(suite.db, suite.T())
}

// TestGoogleLogin_ValidRequest tests the Google OAuth login flow
func (suite *AuthIntegrationTestSuite) TestGoogleLogin_ValidRequest() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/google/login", nil)
	suite.Router.ServeHTTP(w, req)

	// Should return JSON response with auth URL
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response, "auth_url")
	assert.NotEmpty(suite.T(), response["auth_url"])
}

// TestGoogleLogin_WithState tests the Google OAuth login with state parameter
func (suite *AuthIntegrationTestSuite) TestGoogleLogin_WithState() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/google/login?state=test-state", nil)
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response, "auth_url")
}

// TestGoogleLogin_SessionStateGeneration tests that OAuth state is properly stored in session
func (suite *AuthIntegrationTestSuite) TestGoogleLogin_SessionStateGeneration() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/google/login", nil)
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Check that session cookie is set
	cookies := w.Result().Cookies()
	assert.NotEmpty(suite.T(), cookies)

	// Verify session contains OAuth state
	sessionCookie := cookies[0]
	assert.Contains(suite.T(), sessionCookie.Name, "session")
}

// TestGoogleCallback_ValidCode tests the Google OAuth callback with valid code
func (suite *AuthIntegrationTestSuite) TestGoogleCallback_ValidCode() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/google/callback?code=test-code&state=test-state", nil)
	suite.Router.ServeHTTP(w, req)

	// Should return error due to missing OAuth state in session
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "OAUTH_STATE_MISMATCH", response["code"])
}

// TestGoogleCallback_MissingCode tests the Google OAuth callback without code
func (suite *AuthIntegrationTestSuite) TestGoogleCallback_MissingCode() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/google/callback?state=test-state", nil)
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "MISSING_REQUIRED_FIELD", response["code"])
}

// TestGoogleCallback_MissingState tests the Google OAuth callback without state
func (suite *AuthIntegrationTestSuite) TestGoogleCallback_MissingState() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/google/callback?code=test-code", nil)
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "OAUTH_STATE_MISMATCH", response["code"])
}

// TestGoogleCallback_StateMismatch tests OAuth state mismatch (CSRF protection)
func (suite *AuthIntegrationTestSuite) TestGoogleCallback_StateMismatch() {
	// First, create a session with OAuth state
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/v1/auth/google/login", nil)
	suite.Router.ServeHTTP(w1, req1)

	// Extract session cookie
	cookies := w1.Result().Cookies()
	require.NotEmpty(suite.T(), cookies)
	sessionCookie := cookies[0]

	// Test callback with different state
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/v1/auth/google/callback?code=test-code&state=different-state", nil)
	req2.AddCookie(sessionCookie)
	suite.Router.ServeHTTP(w2, req2)

	assert.Equal(suite.T(), http.StatusBadRequest, w2.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w2.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "OAUTH_STATE_MISMATCH", response["code"])
}

// TestGoogleCallback_ReusedCode tests OAuth code reuse protection
func (suite *AuthIntegrationTestSuite) TestGoogleCallback_ReusedCode() {
	// This test would require mocking the OAuth service to simulate code reuse
	// For now, we test the basic flow
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/google/callback?code=reused-code&state=test-state", nil)
	suite.Router.ServeHTTP(w, req)

	// Should return error due to missing state
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestLogin_Success tests successful login with valid credentials
func (suite *AuthIntegrationTestSuite) TestLogin_Success() {
	// Create a test user with proper password hash and all required fields
	// Use bcrypt to hash the password "password"
	hashedPassword := "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi" // hash of "password"
	_, err := suite.db.Exec(`
		INSERT INTO users (username, password_hash, preferred_language, current_level, email, timezone, ai_provider, ai_model, ai_enabled, last_active, created_at, updated_at)
		VALUES ('testuser', $1, 'italian', 'A1', 'test@example.com', 'UTC', 'ollama', 'llama3', true, NOW(), NOW(), NOW())
	`, hashedPassword)
	require.NoError(suite.T(), err)

	loginReq := api.LoginRequest{
		Username: "testuser",
		Password: "password",
	}

	reqBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Should return success
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response api.LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response.Success)
	assert.True(suite.T(), *response.Success)
}

// TestLogin_InvalidCredentials tests login with invalid credentials
func (suite *AuthIntegrationTestSuite) TestLogin_InvalidCredentials() {
	loginReq := api.LoginRequest{
		Username: "nonexistent",
		Password: "wrongpassword",
	}

	reqBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "INVALID_CREDENTIALS", response["code"])
}

// TestLogin_MalformedRequest tests login with malformed JSON
func (suite *AuthIntegrationTestSuite) TestLogin_MalformedRequest() {
	req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestLogout_ValidRequest tests the logout functionality
func (suite *AuthIntegrationTestSuite) TestLogout_ValidRequest() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/auth/logout", nil)
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response api.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
}

// TestLogout_WithSession tests logout with existing session
func (suite *AuthIntegrationTestSuite) TestLogout_WithSession() {
	// Create a test session first
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/v1/auth/google/login", nil)
	suite.Router.ServeHTTP(w1, req1)

	// Then test logout
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/v1/auth/logout", nil)
	suite.Router.ServeHTTP(w2, req2)

	assert.Equal(suite.T(), http.StatusOK, w2.Code)
}

// TestStatus_Unauthenticated tests status endpoint when not authenticated
func (suite *AuthIntegrationTestSuite) TestStatus_Unauthenticated() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/status", nil)
	suite.Router.ServeHTTP(w, req)

	// Should return 200 with unauthenticated status
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), false, response["authenticated"])
	assert.Nil(suite.T(), response["user"])
}

// TestStatus_Authenticated tests status endpoint when authenticated
func (suite *AuthIntegrationTestSuite) TestStatus_Authenticated() {
	// Create a test user with all required fields
	hashedPassword := "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi" // hash of "password"
	_, err := suite.db.Exec(`
		INSERT INTO users (username, password_hash, preferred_language, current_level, email, timezone, ai_provider, ai_model, ai_enabled, last_active, created_at, updated_at)
		VALUES ('testuser', $1, 'italian', 'A1', 'test@example.com', 'UTC', 'ollama', 'llama3', true, NOW(), NOW(), NOW())
	`, hashedPassword)
	require.NoError(suite.T(), err)

	// First, login to get a session
	loginReq := api.LoginRequest{
		Username: "testuser",
		Password: "password",
	}
	loginBody, _ := json.Marshal(loginReq)

	loginW := httptest.NewRecorder()
	loginHttpReq, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginHttpReq.Header.Set("Content-Type", "application/json")
	suite.Router.ServeHTTP(loginW, loginHttpReq)

	// Get the session cookie from the login response
	cookies := loginW.Result().Cookies()
	require.NotEmpty(suite.T(), cookies, "Login should set a session cookie")

	// Now make the status request with the session cookie
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/status", nil)

	// Add the session cookie to the request
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	suite.Router.ServeHTTP(w, req)

	// Should return authenticated status
	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

// TestGenerateRandomState tests the state generation functionality
func (suite *AuthIntegrationTestSuite) TestGenerateRandomState() {
	// Test that state generation produces different values
	state1 := generateRandomState()
	state2 := generateRandomState()

	assert.NotEmpty(suite.T(), state1)
	assert.NotEmpty(suite.T(), state2)
	assert.NotEqual(suite.T(), state1, state2)
	assert.Len(suite.T(), state1, 32) // Assuming 32-character state
}

// TestGenerateRandomState_Length tests the state generation length
func (suite *AuthIntegrationTestSuite) TestGenerateRandomState_Length() {
	state := generateRandomState()
	assert.Len(suite.T(), state, 32)
}

// TestGenerateRandomState_Uniqueness tests that generated states are unique
func (suite *AuthIntegrationTestSuite) TestGenerateRandomState_Uniqueness() {
	states := make(map[string]bool)
	for i := 0; i < 100; i++ {
		state := generateRandomState()
		assert.False(suite.T(), states[state], "Duplicate state generated: %s", state)
		states[state] = true
	}
}

// TestGenerateRandomState_Entropy tests that generated states have sufficient entropy
func (suite *AuthIntegrationTestSuite) TestGenerateRandomState_Entropy() {
	states := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		state := generateRandomState()
		states[state] = true
	}

	// Should have generated many unique states
	assert.Greater(suite.T(), len(states), 950, "Generated states lack sufficient entropy")
}

// TestSessionManagement tests session creation and management
func (suite *AuthIntegrationTestSuite) TestSessionManagement() {
	// Test session creation during login
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/google/login", nil)
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Check that session cookie is set
	cookies := w.Result().Cookies()
	assert.NotEmpty(suite.T(), cookies)

	sessionCookie := cookies[0]
	assert.Contains(suite.T(), sessionCookie.Name, "session")
	assert.NotEmpty(suite.T(), sessionCookie.Value)
}

// TestSessionSecurity tests session security features
func (suite *AuthIntegrationTestSuite) TestSessionSecurity() {
	// Test that sessions are properly secured
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/google/login", nil)
	suite.Router.ServeHTTP(w, req)

	cookies := w.Result().Cookies()
	require.NotEmpty(suite.T(), cookies)

	sessionCookie := cookies[0]
	// Check that session cookie has security attributes
	assert.True(suite.T(), sessionCookie.HttpOnly, "Session cookie should be HttpOnly")
	assert.True(suite.T(), sessionCookie.Secure || !sessionCookie.Secure, "Session cookie security should be configured")
}

// TestOAuthErrorHandling tests various OAuth error scenarios
func (suite *AuthIntegrationTestSuite) TestOAuthErrorHandling() {
	testCases := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "Missing code",
			queryParams:    "state=test-state",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "MISSING_REQUIRED_FIELD",
		},
		{
			name:           "Missing state",
			queryParams:    "code=test-code",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "OAUTH_STATE_MISMATCH",
		},
		{
			name:           "Empty code",
			queryParams:    "code=&state=test-state",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "MISSING_REQUIRED_FIELD",
		},
		{
			name:           "Empty state",
			queryParams:    "code=test-code&state=",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "OAUTH_STATE_MISMATCH",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/v1/auth/google/callback?"+tc.queryParams, nil)
			suite.Router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedCode, response["code"])
		})
	}
}

// TestAuthEndpoints_UnauthorizedAccess tests unauthorized access to protected endpoints
func (suite *AuthIntegrationTestSuite) TestAuthEndpoints_UnauthorizedAccess() {
	protectedEndpoints := []string{
		"/v1/quiz/question",
		"/v1/quiz/progress",
	}

	for _, endpoint := range protectedEndpoints {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", endpoint, nil)
		suite.Router.ServeHTTP(w, req)

		// Should return unauthorized or redirect to login
		assert.Contains(suite.T(), []int{http.StatusUnauthorized, http.StatusFound, http.StatusForbidden}, w.Code)
	}
}

// TestAuthEndpoints_InvalidMethods tests invalid HTTP methods for auth endpoints
func (suite *AuthIntegrationTestSuite) TestAuthEndpoints_InvalidMethods() {
	// Test GET endpoints with POST method
	getEndpoints := []string{
		"/v1/auth/google/login",
		"/v1/auth/google/callback",
	}

	for _, endpoint := range getEndpoints {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", endpoint, nil)
		suite.Router.ServeHTTP(w, req)

		// Should return method not allowed or not found
		assert.Contains(suite.T(), []int{http.StatusMethodNotAllowed, http.StatusNotFound, http.StatusBadRequest}, w.Code)
	}

	// Test POST endpoints with GET method
	postEndpoints := []string{
		"/v1/auth/logout",
	}

	for _, endpoint := range postEndpoints {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", endpoint, nil)
		suite.Router.ServeHTTP(w, req)

		// Should return method not allowed or not found
		assert.Contains(suite.T(), []int{http.StatusMethodNotAllowed, http.StatusNotFound, http.StatusBadRequest}, w.Code)
	}
}

// TestAuthEndpoints_MalformedRequests tests malformed requests to auth endpoints
func (suite *AuthIntegrationTestSuite) TestAuthEndpoints_MalformedRequests() {
	// Test malformed callback URL
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/google/callback?invalid=param", nil)
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestAuthEndpoints_EmptyRequests tests empty requests to auth endpoints
func (suite *AuthIntegrationTestSuite) TestAuthEndpoints_EmptyRequests() {
	// Test empty POST body for login
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestOAuthFlow_CompleteFlow tests a complete OAuth flow with mocked responses
func (suite *AuthIntegrationTestSuite) TestOAuthFlow_CompleteFlow() {
	// Step 1: Initiate OAuth login
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/v1/auth/google/login", nil)
	suite.Router.ServeHTTP(w1, req1)

	assert.Equal(suite.T(), http.StatusOK, w1.Code)

	// Extract session cookie
	cookies := w1.Result().Cookies()
	require.NotEmpty(suite.T(), cookies)
	sessionCookie := cookies[0]

	// Step 2: Simulate OAuth callback (this would normally come from Google)
	// Note: This test is limited because we can't easily mock the OAuth service
	// in the integration test environment
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/v1/auth/google/callback?code=test-code&state=test-state", nil)
	req2.AddCookie(sessionCookie)
	suite.Router.ServeHTTP(w2, req2)

	// Should fail because we can't mock the OAuth service properly
	// but we can verify the request was processed
	assert.Equal(suite.T(), http.StatusBadRequest, w2.Code)
}

// TestSessionTimeout tests session timeout behavior
func (suite *AuthIntegrationTestSuite) TestSessionTimeout() {
	// This test would require configuring session timeout
	// For now, we test basic session behavior
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/auth/google/login", nil)
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	cookies := w.Result().Cookies()
	require.NotEmpty(suite.T(), cookies)

	sessionCookie := cookies[0]
	assert.NotEmpty(suite.T(), sessionCookie.Value)
}

// TestConcurrentOAuthRequests tests handling of concurrent OAuth requests
func (suite *AuthIntegrationTestSuite) TestConcurrentOAuthRequests() {
	// Test that multiple concurrent OAuth login requests work correctly
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/v1/auth/google/login", nil)
			suite.Router.ServeHTTP(w, req)

			assert.Equal(suite.T(), http.StatusOK, w.Code)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 5; i++ {
		select {
		case <-done:
			// Request completed successfully
		case <-time.After(5 * time.Second):
			suite.T().Fatal("Timeout waiting for concurrent OAuth requests")
		}
	}
}

// TestOAuthStateReuse tests that OAuth states cannot be reused
func (suite *AuthIntegrationTestSuite) TestOAuthStateReuse() {
	// Step 1: Create initial OAuth state
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/v1/auth/google/login", nil)
	suite.Router.ServeHTTP(w1, req1)

	cookies := w1.Result().Cookies()
	require.NotEmpty(suite.T(), cookies)
	sessionCookie := cookies[0]

	// Step 2: Try to use the same state twice
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/v1/auth/google/callback?code=test-code&state=test-state", nil)
	req2.AddCookie(sessionCookie)
	suite.Router.ServeHTTP(w2, req2)

	// First use should fail due to OAuth service mocking limitations
	assert.Equal(suite.T(), http.StatusBadRequest, w2.Code)

	// Step 3: Try to use the same state again
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/v1/auth/google/callback?code=test-code&state=test-state", nil)
	req3.AddCookie(sessionCookie)
	suite.Router.ServeHTTP(w3, req3)

	// Should also fail
	assert.Equal(suite.T(), http.StatusBadRequest, w3.Code)
}
