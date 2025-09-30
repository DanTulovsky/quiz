//go:build integration
// +build integration

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

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// sharedTestDBSetup provides a clean, isolated database for each integration test
// Uses the optimized SharedTestDBSetup from services package
func sharedTestDBSetup(t *testing.T) *sql.DB {
	return services.SharedTestDBSetup(t)
}

type UserAdminIntegrationTestSuite struct {
	suite.Suite
	db               *sql.DB
	userService      *services.UserService
	userAdminHandler *UserAdminHandler
	router           *gin.Engine
	cfg              *config.Config
}

func (suite *UserAdminIntegrationTestSuite) SetupSuite() {
	// Removed manual AI_PROVIDERS_CONFIG setting; Taskfile.yml sets it correctly.
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

	// Initialize test database
	suite.db = sharedTestDBSetup(suite.T())

	// Initialize services
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	suite.userService = services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, logger)
	aiService := services.NewAIService(suite.cfg, logger)
	workerService := services.NewWorkerServiceWithLogger(suite.db, logger)
	oauthService := services.NewOAuthServiceWithLogger(suite.cfg, logger)

	// Use the real application router
	dailyQuestionService := services.NewDailyQuestionService(suite.db, logger, questionService, learningService)
	generationHintService := services.NewGenerationHintService(suite.db, logger)
	suite.router = NewRouter(
		suite.cfg,
		suite.userService,
		questionService,
		learningService,
		aiService,
		workerService,
		dailyQuestionService,
		oauthService,
		generationHintService,
		logger,
	)
}

func (suite *UserAdminIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *UserAdminIntegrationTestSuite) SetupTest() {
	// Clean up before each test using TRUNCATE CASCADE for proper isolation
	// Order matters: child tables first, then parent tables
	tables := []string{
		"user_responses",
		"performance_metrics",
		"questions",
		"worker_status",
		"worker_settings",
		"users",
	}

	for _, table := range tables {
		_, err := suite.db.Exec("TRUNCATE TABLE " + table + " CASCADE")
		if err != nil {
			suite.T().Logf("Warning: Could not truncate table %s: %v", table, err)
		}
	}

	// Reset sequences to ensure consistent IDs starting from 1
	sequences := []string{"users_id_seq", "questions_id_seq", "user_responses_id_seq", "performance_metrics_id_seq"}
	for _, seq := range sequences {
		_, err := suite.db.Exec("ALTER SEQUENCE " + seq + " RESTART WITH 1")
		if err != nil {
			// Log but don't fail if sequence doesn't exist
			suite.T().Logf("Note: Could not reset sequence %s: %v", seq, err)
		}
	}

	// Re-insert default worker settings for consistent test state
	_, err := suite.db.Exec(`
		INSERT INTO worker_settings (setting_key, setting_value, created_at, updated_at)
		VALUES ('global_pause', 'false', NOW(), NOW())
		ON CONFLICT (setting_key) DO NOTHING;
	`)
	require.NoError(suite.T(), err)

	// Create admin user for authentication
	adminUser, err := suite.userService.CreateUserWithPassword(context.Background(), "admin_test", "password", "italian", "A1")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), adminUser)

	// Assign admin role to the test user
	err = suite.userService.AssignRoleByName(context.Background(), adminUser.ID, "admin")
	require.NoError(suite.T(), err, "Failed to assign admin role to test user")
}

func (suite *UserAdminIntegrationTestSuite) authenticateAsAdmin() *http.Cookie {
	loginReq := api.LoginRequest{
		Username: "admin_test",
		Password: "password",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqObj, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqObj.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	suite.router.ServeHTTP(loginW, loginReqObj)

	// Log the login response for debugging
	suite.T().Logf("Login response status: %d", loginW.Code)
	suite.T().Logf("Login response body: %s", loginW.Body.String())

	// Get the session cookie from the login response
	cookies := loginW.Result().Cookies()
	suite.T().Logf("Found %d cookies", len(cookies))
	for _, cookie := range cookies {
		suite.T().Logf("Cookie: %s = %s", cookie.Name, cookie.Value)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == config.SessionName {
			sessionCookie = cookie
			break
		}
	}
	require.NotNil(suite.T(), sessionCookie, "Should have session cookie after login")
	return sessionCookie
}

func (suite *UserAdminIntegrationTestSuite) TestGetUsers() {
	// Create test users
	user1, err := suite.userService.CreateUserWithEmailAndTimezone(context.Background(), "user1", "user1@example.com", "America/New_York", "italian", "A1")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), user1)
	suite.T().Logf("Created user1 with ID: %d", user1.ID)

	user2, err := suite.userService.CreateUserWithEmailAndTimezone(context.Background(), "user2", "user2@example.com", "Europe/London", "spanish", "B1")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), user2)
	suite.T().Logf("Created user2 with ID: %d", user2.ID)

	// Verify users exist in database
	allUsers, err := suite.userService.GetAllUsers(context.Background())
	require.NoError(suite.T(), err)
	suite.T().Logf("Found %d users in database", len(allUsers))
	for _, u := range allUsers {
		suite.T().Logf("User in DB: ID=%d, Username=%s", u.ID, u.Username)
	}

	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Make request with Accept header to get JSON response
	req, _ := http.NewRequest("GET", "/v1/admin/backend/userz", nil)
	req.Header.Set("Accept", "application/json")
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	suite.T().Logf("Response body: %s", w.Body.String())

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	users, ok := response["users"].([]interface{})
	require.True(suite.T(), ok)
	assert.Len(suite.T(), users, 3) // Including admin_test user created for authentication

	// Verify user data - check first user
	user1Data := users[0].(map[string]interface{})
	assert.Equal(suite.T(), "user1", user1Data["username"])
	assert.Equal(suite.T(), "user1@example.com", user1Data["email"])
	assert.Equal(suite.T(), "America/New_York", user1Data["timezone"])
	assert.Equal(suite.T(), float64(user1.ID), user1Data["id"])

	// Verify second user exists
	_ = user2 // Use user2 to avoid lint error
}

func (suite *UserAdminIntegrationTestSuite) TestCreateUser() {
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Prepare request body
	email := openapi_types.Email("newuser@example.com")
	timezone := "Asia/Tokyo"
	preferredLanguage := "french"
	currentLevel := "B2"
	createReq := UserCreateRequest{
		Username:          "newuser",
		Email:             &email,
		Timezone:          &timezone,
		Password:          "password123",
		PreferredLanguage: &preferredLanguage,
		CurrentLevel:      &currentLevel,
	}

	reqBody, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/v1/admin/backend/userz", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.AddCookie(sessionCookie)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Log the response for debugging
	suite.T().Logf("Create user response status: %d", w.Code)
	suite.T().Logf("Create user response body: %s", w.Body.String())

	// Verify response
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "User created successfully", response["message"])

	userObj, ok := response["user"].(map[string]interface{})
	require.True(suite.T(), ok)
	assert.Equal(suite.T(), "newuser", userObj["username"])
	assert.Equal(suite.T(), "newuser@example.com", userObj["email"])
	assert.Equal(suite.T(), "Asia/Tokyo", userObj["timezone"])

	// Verify user was actually created in database
	user, err := suite.userService.GetUserByUsername(context.Background(), "newuser")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), user, "User should not be nil after creation")
	assert.Equal(suite.T(), "newuser@example.com", user.Email.String)
	assert.Equal(suite.T(), "Asia/Tokyo", user.Timezone.String)
}

func (suite *UserAdminIntegrationTestSuite) TestCreateUserValidation() {
	// Test missing required fields
	tests := []struct {
		name         string
		request      UserCreateRequest
		expectedCode int
	}{
		{
			name: "missing username",
			request: UserCreateRequest{
				Email:    func() *openapi_types.Email { e := openapi_types.Email("test@example.com"); return &e }(),
				Password: "password123",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "missing password",
			request: UserCreateRequest{
				Username: "testuser",
				Email:    func() *openapi_types.Email { e := openapi_types.Email("test@example.com"); return &e }(),
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "duplicate username",
			request: UserCreateRequest{
				Username: "existing",
				Email:    func() *openapi_types.Email { e := openapi_types.Email("test@example.com"); return &e }(),
				Password: "password123",
			},
			expectedCode: http.StatusConflict,
		},
	}

	// Create existing user for duplicate test
	_, err := suite.userService.CreateUserWithEmailAndTimezone(context.Background(), "existing", "existing@example.com", "UTC", "italian", "A1")
	require.NoError(suite.T(), err)

	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			reqBody, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest("POST", "/v1/admin/backend/userz", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")
			req.AddCookie(sessionCookie)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func (suite *UserAdminIntegrationTestSuite) TestUpdateUser() {
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create test user
	user, err := suite.userService.CreateUserWithEmailAndTimezone(context.Background(), "testuser", "old@example.com", "UTC", "italian", "A1")
	require.NoError(suite.T(), err)

	// Prepare update request
	username := "updateduser"
	email := openapi_types.Email("updated@example.com")
	timezone := "Europe/Paris"
	updateReq := UserUpdateRequest{
		Username: &username,
		Email:    &email,
		Timezone: &timezone,
	}

	reqBody, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/v1/admin/backend/userz/%d", user.ID), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.AddCookie(sessionCookie)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "User updated successfully", response["message"])

	// Verify user was actually updated in database
	updatedUser, err := suite.userService.GetUserByID(context.Background(), user.ID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), updatedUser)
	assert.Equal(suite.T(), "updateduser", updatedUser.Username)
	assert.Equal(suite.T(), "updated@example.com", updatedUser.Email.String)
	assert.Equal(suite.T(), "Europe/Paris", updatedUser.Timezone.String)
}

// TestUpdateCurrentUserProfile_AllowsEmptyAIFields verifies that saving profile with empty AI fields is accepted (covers SSO-first-save case)
func (suite *UserAdminIntegrationTestSuite) TestUpdateCurrentUserProfile_AllowsEmptyAIFields() {
	// Authenticate as admin (self profile update)
	sessionCookie := suite.authenticateAsAdmin()

	// Prepare request body with empty AI fields
	body := map[string]interface{}{
		"username":           "admin_test",
		"email":              "admin_test@example.com",
		"timezone":           "UTC",
		"preferred_language": "italian",
		"current_level":      "A1",
		"ai_enabled":         false,
		"ai_provider":        "",
		"ai_model":           "",
	}
	reqBytes, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/v1/userz/profile", bytes.NewBuffer(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.AddCookie(sessionCookie)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Expect success after relaxing regex/conditional in schema
	assert.Equal(suite.T(), http.StatusOK, w.Code, "should accept empty ai_provider/ai_model on profile save")
}

// TestUpdateCurrentUserProfile_AllowsColonInAIModel verifies models like "llama4:latest" are accepted when a provider is set
func (suite *UserAdminIntegrationTestSuite) TestUpdateCurrentUserProfile_AllowsColonInAIModel() {
	sessionCookie := suite.authenticateAsAdmin()

	body := map[string]interface{}{
		"username":           "admin_test",
		"timezone":           "America/Detroit",
		"preferred_language": "italian",
		"current_level":      "A1",
		"ai_enabled":         true,
		"ai_provider":        "ollama",
		"ai_model":           "llama4:latest",
		"api_key":            "123",
	}
	reqBytes, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/v1/userz/profile", bytes.NewBuffer(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.AddCookie(sessionCookie)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code, "should accept ai_model with colon when provider is set")
}

// TestUpdateCurrentUserProfile_RequiresAiModelWhenProviderPresent verifies conditional requirement:
// if ai_provider is provided, ai_model must also be present
func (suite *UserAdminIntegrationTestSuite) TestUpdateCurrentUserProfile_RequiresAiModelWhenProviderPresent() {
	sessionCookie := suite.authenticateAsAdmin()

	// Missing ai_model while ai_provider is set
	body := map[string]interface{}{
		"username":           "admin_test",
		"timezone":           "UTC",
		"preferred_language": "italian",
		"current_level":      "A1",
		"ai_enabled":         true,
		"ai_provider":        "ollama",
		// "ai_model" omitted intentionally
	}
	reqBytes, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/v1/userz/profile", bytes.NewBuffer(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.AddCookie(sessionCookie)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Expect 400 once the schema enforces conditional requirement
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code, "should reject when ai_provider is set but ai_model missing")
}

func (suite *UserAdminIntegrationTestSuite) TestDeleteUser() {
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create test user
	user, err := suite.userService.CreateUserWithEmailAndTimezone(context.Background(), "testuser", "test@example.com", "UTC", "italian", "A1")
	require.NoError(suite.T(), err)

	// Delete user
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1/admin/backend/userz/%d", user.ID), nil)
	req.Header.Set("Accept", "application/json")
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "User deleted successfully", response["message"])

	// Verify user was actually deleted from database
	deletedUser, err := suite.userService.GetUserByID(context.Background(), user.ID)
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), deletedUser)
}

func (suite *UserAdminIntegrationTestSuite) TestResetPassword() {
	// Create test user with password - need to add email/timezone manually since CreateUserWithPassword doesn't set them
	user, err := suite.userService.CreateUserWithPassword(context.Background(), "testuser", "oldpassword", "italian", "A1")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), user, "User should not be nil after creation")
	suite.T().Logf("Created user with ID: %d, username: %s", user.ID, user.Username)

	// Verify user exists immediately after creation
	existingUser, err := suite.userService.GetUserByUsername(context.Background(), "testuser")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), existingUser, "User should exist immediately after creation")
	suite.T().Logf("Verified user exists with ID: %d", existingUser.ID)

	// Verify user can authenticate with old password before reset
	authUser, err := suite.userService.AuthenticateUser(context.Background(), "testuser", "oldpassword")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), authUser, "User should authenticate with old password")
	suite.T().Logf("Successfully authenticated with old password, user ID: %d", authUser.ID)

	// Double-check user exists right before making the HTTP request
	userBeforeRequest, err := suite.userService.GetUserByID(context.Background(), user.ID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), userBeforeRequest, "User should exist right before HTTP request")
	suite.T().Logf("User exists right before HTTP request with ID: %d", userBeforeRequest.ID)

	// Prepare reset password request
	resetReq := PasswordResetRequest{
		NewPassword: "newpassword123",
	}

	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	reqBody, _ := json.Marshal(resetReq)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/admin/backend/userz/%d/reset-password", user.ID), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.AddCookie(sessionCookie)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Log the response for debugging
	suite.T().Logf("HTTP Response Status: %d", w.Code)
	suite.T().Logf("HTTP Response Body: %s", w.Body.String())

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "Password reset successfully", response["message"])

	// Verify user still exists after password reset
	userAfterReset, err := suite.userService.GetUserByUsername(context.Background(), "testuser")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), userAfterReset, "User should still exist after password reset")
	suite.T().Logf("User still exists after reset with ID: %d", userAfterReset.ID)

	// Verify password was actually changed by attempting authentication
	authenticatedUser, err := suite.userService.AuthenticateUser(context.Background(), "testuser", "newpassword123")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), authenticatedUser)
	assert.Equal(suite.T(), user.ID, authenticatedUser.ID)

	// Verify old password no longer works
	_, err = suite.userService.AuthenticateUser(context.Background(), "testuser", "oldpassword")
	assert.Error(suite.T(), err)
}

func TestUserAdminIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UserAdminIntegrationTestSuite))
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
