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

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockUserService for testing
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, username, language, level string) (result0 *models.User, err error) {
	args := m.Called(ctx, username, language, level)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(ctx context.Context, id int) (result0 *models.User, err error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetUserByUsername(ctx context.Context, username string) (result0 *models.User, err error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) UpdateUserSettings(ctx context.Context, userID int, settings *models.UserSettings) error {
	args := m.Called(ctx, userID, settings)
	return args.Error(0)
}

func (m *MockUserService) UpdateLastActive(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) GetAllUsers(ctx context.Context) (result0 []models.User, err error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserService) DeleteUser(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) DeleteAllUsers(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockUserService) CreateUserWithPassword(ctx context.Context, username, password, language, level string) (result0 *models.User, err error) {
	args := m.Called(ctx, username, password, language, level)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) AuthenticateUser(ctx context.Context, username, password string) (result0 *models.User, err error) {
	args := m.Called(ctx, username, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) EnsureAdminUserExists(ctx context.Context, adminUsername, adminPassword string) error {
	args := m.Called(ctx, adminUsername, adminPassword)
	return args.Error(0)
}

func (m *MockUserService) ResetDatabase(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockUserService) ClearUserData(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockUserService) GetUserAPIKey(ctx context.Context, userID int, provider string) (result0 string, err error) {
	args := m.Called(ctx, userID, provider)
	return args.String(0), args.Error(1)
}

func (m *MockUserService) SetUserAPIKey(ctx context.Context, userID int, provider, apiKey string) error {
	args := m.Called(ctx, userID, provider, apiKey)
	return args.Error(0)
}

func (m *MockUserService) HasUserAPIKey(ctx context.Context, userID int, provider string) (result0 bool, err error) {
	args := m.Called(ctx, userID, provider)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserService) CreateUserWithEmailAndTimezone(ctx context.Context, username, email, timezone, language, level string) (result0 *models.User, err error) {
	args := m.Called(ctx, username, email, timezone, language, level)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetUserByEmail(ctx context.Context, email string) (result0 *models.User, err error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) UpdateUserProfile(ctx context.Context, userID int, username, email, timezone string) error {
	args := m.Called(ctx, userID, username, email, timezone)
	return args.Error(0)
}

func (m *MockUserService) ResetPassword(ctx context.Context, userID int, newPassword string) error {
	args := m.Called(ctx, userID, newPassword)
	return args.Error(0)
}

func (m *MockUserService) UpdateUserPassword(ctx context.Context, userID int, newPassword string) error {
	args := m.Called(ctx, userID, newPassword)
	return args.Error(0)
}

func (m *MockUserService) ClearUserDataForUser(_ context.Context, _ int) error { return nil }

// Role management methods
func (m *MockUserService) GetUserRoles(ctx context.Context, userID int) (result0 []models.Role, err error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *MockUserService) AssignRole(ctx context.Context, userID, roleID int) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockUserService) AssignRoleByName(ctx context.Context, userID int, roleName string) error {
	args := m.Called(ctx, userID, roleName)
	return args.Error(0)
}

func (m *MockUserService) RemoveRole(ctx context.Context, userID, roleID int) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockUserService) HasRole(ctx context.Context, userID int, roleName string) (result0 bool, err error) {
	args := m.Called(ctx, userID, roleName)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserService) IsAdmin(ctx context.Context, userID int) (result0 bool, err error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserService) GetAllRoles(ctx context.Context) (result0 []models.Role, err error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *MockUserService) GetUsersPaginated(ctx context.Context, page, pageSize int, search, language, level, aiProvider, aiModel, aiEnabled, active string) (result0 []models.User, result1 int, err error) {
	args := m.Called(ctx, page, pageSize, search, language, level, aiProvider, aiModel, aiEnabled, active)
	return args.Get(0).([]models.User), args.Int(1), args.Error(2)
}

func (m *MockUserService) GetDB() *sql.DB {
	args := m.Called()
	return args.Get(0).(*sql.DB)
}

func setupAuthTestRouter(userService services.UserServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup session middleware
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("test-session", store))

	cfg := &config.Config{
		Server: config.ServerConfig{
			SessionSecret: "test-secret",
			AdminUsername: "admin",
			AdminPassword: "password",
		},
	}

	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	handler := NewAuthHandler(userService, oauthService, cfg, logger)

	router.POST("/login", handler.Login)
	router.POST("/logout", handler.Logout)
	router.GET("/status", handler.Status)

	return router
}

// Helper to create *openapi_types.Email from string
func emailPtr(s string) *openapi_types.Email {
	e := openapi_types.Email(s)
	return &e
}

func TestAuthHandler_Login_Success_NewUser(t *testing.T) {
	mockUserService := new(MockUserService)
	router := setupAuthTestRouter(mockUserService)

	testUser := &models.User{
		ID:                1,
		Username:          "admin",
		PreferredLanguage: sql.NullString{String: "italian", Valid: true},
		CurrentLevel:      sql.NullString{String: "A1", Valid: true},
	}

	// Mock user service calls - try authentication first, then fallback to old behavior if needed
	mockUserService.On("AuthenticateUser", mock.Anything, "admin", "password").Return(testUser, nil)
	mockUserService.On("UpdateLastActive", mock.Anything, 1).Return(nil)
	// No API key checking needed since user doesn't have an AI provider set

	loginReq := api.LoginRequest{
		Username: "admin",
		Password: "password",
	}

	reqBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.True(t, *response.Success)
	assert.Equal(t, "Login successful", *response.Message)
	assert.NotNil(t, response.User)
	assert.Equal(t, "admin", *response.User.Username)

	mockUserService.AssertExpectations(t)
}

func TestAuthHandler_Login_Success_ExistingUser(t *testing.T) {
	mockUserService := new(MockUserService)
	router := setupAuthTestRouter(mockUserService)

	testUser := &models.User{
		ID:                1,
		Username:          "admin",
		PreferredLanguage: sql.NullString{String: "italian", Valid: true},
		CurrentLevel:      sql.NullString{String: "A1", Valid: true},
	}

	// Mock user service calls
	mockUserService.On("AuthenticateUser", mock.Anything, "admin", "password").Return(testUser, nil)
	mockUserService.On("UpdateLastActive", mock.Anything, 1).Return(nil)
	// No API key checking needed since user doesn't have an AI provider set

	loginReq := api.LoginRequest{
		Username: "admin",
		Password: "password",
	}

	reqBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.True(t, *response.Success)
	assert.NotNil(t, response.User)
	assert.Equal(t, "admin", *response.User.Username)

	mockUserService.AssertExpectations(t)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	mockUserService := new(MockUserService)
	router := setupAuthTestRouter(mockUserService)

	// Mock authentication failure
	mockUserService.On("AuthenticateUser", mock.Anything, "admin", "wrong-password").Return(nil, assert.AnError)

	loginReq := api.LoginRequest{
		Username: "admin",
		Password: "wrong-password",
	}

	reqBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.Equal(t, "INVALID_CREDENTIALS", response["code"])
	assert.Contains(t, response, "message")
}

func TestAuthHandler_Login_MissingFields(t *testing.T) {
	mockUserService := new(MockUserService)
	router := setupAuthTestRouter(mockUserService)

	tests := []struct {
		name string
		req  api.LoginRequest
	}{
		{
			name: "missing username",
			req: api.LoginRequest{
				Password: "password",
			},
		},
		{
			name: "missing password",
			req: api.LoginRequest{
				Username: "admin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock authentication failure for empty fields
			mockUserService.On("AuthenticateUser", mock.Anything, tt.req.Username, tt.req.Password).Return(nil, assert.AnError)

			reqBody, _ := json.Marshal(tt.req)
			req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Empty fields result in invalid credentials (401), not bad request (400)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	mockUserService := new(MockUserService)
	router := setupAuthTestRouter(mockUserService)

	// To test logout, we first need to simulate a login to create a session
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/status", nil) // Any authenticated endpoint

	// Manually create a session
	session, _ := cookie.NewStore([]byte("test-secret")).Get(req, "test-session")
	session.Values["user_id"] = 1
	_ = session.Save(req, w)

	// Now perform the logout
	req, _ = http.NewRequest("POST", "/logout", nil)
	req.Header.Set("Cookie", w.Header().Get("Set-Cookie"))

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that the session is cleared
	// A new response recorder for the status check
	statusRecorder := httptest.NewRecorder()
	statusReq, _ := http.NewRequest("GET", "/status", nil)
	statusReq.Header.Set("Cookie", w.Header().Get("Set-Cookie"))
	router.ServeHTTP(statusRecorder, statusReq)

	statusResponse := make(map[string]interface{})
	require.NoError(t, json.Unmarshal(statusRecorder.Body.Bytes(), &statusResponse))
	assert.False(t, statusResponse["authenticated"].(bool))
}

func TestAuthHandler_Status_NotAuthenticated(t *testing.T) {
	mockUserService := new(MockUserService)
	router := setupAuthTestRouter(mockUserService)

	req, _ := http.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.False(t, response["authenticated"].(bool))
}

func TestAuthHandler_Status_Authenticated(t *testing.T) {
	mockUserService := new(MockUserService)
	router := setupAuthTestRouter(mockUserService)

	testUser := &models.User{
		ID:       1,
		Username: "admin",
	}

	mockUserService.On("GetUserByID", mock.Anything, 1).Return(testUser, nil)
	// Mock API key checking (no API key for this test)
	mockUserService.On("GetUserAPIKey", mock.Anything, 1, mock.Anything).Return("", nil)
	// Mock UpdateLastActive call
	mockUserService.On("UpdateLastActive", mock.Anything, 1).Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/status", nil)

	// Manually create a session
	session, _ := cookie.NewStore([]byte("test-secret")).Get(req, "test-session")
	session.Values["user_id"] = 1
	_ = session.Save(req, w)

	req.Header.Set("Cookie", w.Header().Get("Set-Cookie"))

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var statusResponse map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &statusResponse))

	assert.True(t, statusResponse["authenticated"].(bool))

	userMap, ok := statusResponse["user"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "admin", userMap["username"])
}

func TestAuthHandler_Signup_Success(t *testing.T) {
	// Mock user service
	mockUserService := new(MockUserService)

	user := &models.User{
		ID:       1,
		Username: "newuser",
		Email:    sql.NullString{String: "newuser@example.com", Valid: true},
	}

	// Mock expectations
	mockUserService.On("GetUserByUsername", mock.Anything, "newuser").Return(nil, nil)
	mockUserService.On("GetUserByEmail", mock.Anything, "newuser@example.com").Return(nil, nil)
	// Use a more flexible mock that accepts any language and expect canonical level code (A1)
	mockUserService.On("CreateUserWithEmailAndTimezone", mock.Anything, "newuser", "newuser@example.com", "UTC", mock.AnythingOfType("string"), "A1").Return(user, nil)
	mockUserService.On("UpdateUserPassword", mock.Anything, 1, "password123").Return(nil)

	// Create config with signups enabled for testing
	cfg := &config.Config{
		LanguageLevels: map[string]config.LanguageLevelConfig{
			"italian": {
				Levels: []string{"A1", "A2", "B1", "B2", "C1", "C2"},
			},
		},
	}

	// Create OAuth service
	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create auth handler
	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Setup gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/signup", handler.Signup)

	// Test signup
	requestBody := map[string]string{
		"username": "newuser",
		"password": "password123",
		"email":    "newuser@example.com",
	}
	jsonBody, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/signup", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	assert.Equal(t, "Account created successfully. Please log in.", response["message"])

	mockUserService.AssertExpectations(t)
}

func TestAuthHandler_Signup_Validation(t *testing.T) {
	// Mock user service
	mockUserService := new(MockUserService)

	// Create config
	cfg := &config.Config{}

	// Create OAuth service
	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create auth handler
	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Setup gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/signup", handler.Signup)

	tests := []struct {
		name             string
		requestBody      map[string]interface{}
		expectedHTTPCode int
		expectedErrCode  string
	}{
		{
			name:             "missing username",
			requestBody:      map[string]interface{}{"password": "password123", "email": "test@example.com"},
			expectedHTTPCode: http.StatusBadRequest,
			expectedErrCode:  "MISSING_REQUIRED_FIELD",
		},
		{
			name:             "missing password",
			requestBody:      map[string]interface{}{"username": "testuser", "email": "test@example.com"},
			expectedHTTPCode: http.StatusBadRequest,
			expectedErrCode:  "MISSING_REQUIRED_FIELD",
		},
		{
			name:             "missing email",
			requestBody:      map[string]interface{}{"username": "testuser", "password": "password123"},
			expectedHTTPCode: http.StatusBadRequest,
			expectedErrCode:  "MISSING_REQUIRED_FIELD",
		},
		{
			name:             "short username",
			requestBody:      map[string]interface{}{"username": "ab", "password": "password123", "email": "test@example.com"},
			expectedHTTPCode: http.StatusBadRequest,
			expectedErrCode:  "INVALID_FORMAT",
		},
		{
			name:             "invalid username characters",
			requestBody:      map[string]interface{}{"username": "test-user", "password": "password123", "email": "test@example.com"},
			expectedHTTPCode: http.StatusBadRequest,
			expectedErrCode:  "INVALID_FORMAT",
		},
		{
			name:             "short password",
			requestBody:      map[string]interface{}{"username": "testuser", "password": "123", "email": "test@example.com"},
			expectedHTTPCode: http.StatusBadRequest,
			expectedErrCode:  "INVALID_FORMAT",
		},
		{
			name:             "invalid email",
			requestBody:      map[string]interface{}{"username": "testuser", "password": "password123", "email": "invalid-email"},
			expectedHTTPCode: http.StatusBadRequest,
			expectedErrCode:  "INVALID_INPUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.requestBody)

			req, _ := http.NewRequest("POST", "/signup", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedHTTPCode, w.Code)

			var response map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			assert.Equal(t, tt.expectedErrCode, response["code"])
		})
	}
}

func TestAuthHandler_Signup_DuplicateUsername(t *testing.T) {
	// Mock user service
	mockUserService := new(MockUserService)

	existingUser := &models.User{
		ID:       1,
		Username: "existinguser",
	}

	// Mock expectations
	mockUserService.On("GetUserByUsername", mock.Anything, "existinguser").Return(existingUser, nil)

	// Create config
	cfg := &config.Config{}

	// Create OAuth service
	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create auth handler
	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Setup gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/signup", handler.Signup)

	// Test signup with duplicate username
	requestBody := map[string]string{
		"username": "existinguser",
		"password": "password123",
		"email":    "newuser@example.com",
	}
	jsonBody, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/signup", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, "RECORD_ALREADY_EXISTS", response["code"])

	mockUserService.AssertExpectations(t)
}

func TestAuthHandler_Signup_DuplicateEmail(t *testing.T) {
	// Mock user service
	mockUserService := new(MockUserService)

	existingUser := &models.User{
		ID:    1,
		Email: sql.NullString{String: "existing@example.com", Valid: true},
	}

	// Mock expectations
	mockUserService.On("GetUserByUsername", mock.Anything, "newuser").Return(nil, nil)
	mockUserService.On("GetUserByEmail", mock.Anything, "existing@example.com").Return(existingUser, nil)

	// Create config
	cfg := &config.Config{}

	// Create OAuth service
	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create auth handler
	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Setup gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/signup", handler.Signup)

	// Test signup with duplicate email
	requestBody := map[string]string{
		"username": "newuser",
		"password": "password123",
		"email":    "existing@example.com",
	}
	jsonBody, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/signup", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, "RECORD_ALREADY_EXISTS", response["code"])

	mockUserService.AssertExpectations(t)
}

func TestAuthHandler_SignupStatus_Enabled(t *testing.T) {
	mockUserService := new(MockUserService)

	// Create config with signups enabled
	cfg := &config.Config{
		Server: config.ServerConfig{
			SessionSecret: "test-secret",
			AdminUsername: "admin",
			AdminPassword: "password",
		},
		System: &config.SystemConfig{
			Auth: config.AuthConfig{
				SignupsDisabled: false,
			},
		},
	}

	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a new router for this test
	gin.SetMode(gin.TestMode)
	testRouter := gin.New()
	testRouter.GET("/signup/status", handler.SignupStatus)

	req, _ := http.NewRequest("GET", "/signup/status", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.Equal(t, false, response["signups_disabled"])
}

func TestAuthHandler_SignupStatus_Disabled(t *testing.T) {
	mockUserService := new(MockUserService)

	// Create config with signups disabled
	cfg := &config.Config{
		Server: config.ServerConfig{
			SessionSecret: "test-secret",
			AdminUsername: "admin",
			AdminPassword: "password",
		},
		System: &config.SystemConfig{
			Auth: config.AuthConfig{
				SignupsDisabled: true,
			},
		},
	}

	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a new router for this test
	gin.SetMode(gin.TestMode)
	testRouter := gin.New()
	testRouter.GET("/signup/status", handler.SignupStatus)

	req, _ := http.NewRequest("GET", "/signup/status", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.Equal(t, true, response["signups_disabled"])
}

func TestAuthHandler_SignupStatus_NoConfig(t *testing.T) {
	mockUserService := new(MockUserService)

	// Create config with no system config
	cfg := &config.Config{
		Server: config.ServerConfig{
			SessionSecret: "test-secret",
			AdminUsername: "admin",
			AdminPassword: "password",
		},
	}

	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a new router for this test
	gin.SetMode(gin.TestMode)
	testRouter := gin.New()
	testRouter.GET("/signup/status", handler.SignupStatus)

	req, _ := http.NewRequest("GET", "/signup/status", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.Equal(t, false, response["signups_disabled"])
	assert.Equal(t, false, response["oauth_whitelist_enabled"])
	assert.Nil(t, response["allowed_domains"])
	assert.Nil(t, response["allowed_emails"])
}

func TestAuthHandler_SignupStatus_WithWhitelist(t *testing.T) {
	mockUserService := new(MockUserService)

	// Create config with signups disabled but whitelist enabled
	cfg := &config.Config{
		Server: config.ServerConfig{
			SessionSecret: "test-secret",
			AdminUsername: "admin",
			AdminPassword: "password",
		},
		System: &config.SystemConfig{
			Auth: config.AuthConfig{
				SignupsDisabled: true,
				AllowedDomains:  []string{"company.com", "trusted-partner.org"},
				AllowedEmails:   []string{"admin@example.com", "support@quizapp.com"},
			},
		},
	}

	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a new router for this test
	gin.SetMode(gin.TestMode)
	testRouter := gin.New()
	testRouter.GET("/signup/status", handler.SignupStatus)

	req, _ := http.NewRequest("GET", "/signup/status", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.Equal(t, true, response["signups_disabled"])
	assert.Equal(t, true, response["oauth_whitelist_enabled"])

	// Check allowed domains
	allowedDomains, ok := response["allowed_domains"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, allowedDomains, 2)
	assert.Contains(t, allowedDomains, "company.com")
	assert.Contains(t, allowedDomains, "trusted-partner.org")

	// Check allowed emails
	allowedEmails, ok := response["allowed_emails"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, allowedEmails, 2)
	assert.Contains(t, allowedEmails, "admin@example.com")
	assert.Contains(t, allowedEmails, "support@quizapp.com")
}

func TestAuthHandler_Signup_Disabled(t *testing.T) {
	mockUserService := new(MockUserService)

	// Create config with signups disabled
	cfg := &config.Config{
		Server: config.ServerConfig{
			SessionSecret: "test-secret",
			AdminUsername: "admin",
			AdminPassword: "password",
		},
		System: &config.SystemConfig{
			Auth: config.AuthConfig{
				SignupsDisabled: true,
			},
		},
	}

	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a new router for this test
	gin.SetMode(gin.TestMode)
	testRouter := gin.New()
	testRouter.POST("/signup", handler.Signup)

	signupReq := api.UserCreateRequest{
		Username: "testuser",
		Password: "password123",
		Email:    emailPtr("test@example.com"),
	}

	reqBody, _ := json.Marshal(signupReq)
	req, _ := http.NewRequest("POST", "/signup", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.Equal(t, "FORBIDDEN", response["code"])
}

func TestAuthHandler_GoogleCallback_SignupDisabled(t *testing.T) {
	mockUserService := new(MockUserService)

	// Expect GetUserByEmail to be called and return nil (user does not exist)
	mockUserService.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, nil)

	// Create config with signups disabled
	cfg := &config.Config{
		Server: config.ServerConfig{
			SessionSecret: "test-secret",
			AdminUsername: "admin",
			AdminPassword: "password",
		},
		System: &config.SystemConfig{
			Auth: config.AuthConfig{
				SignupsDisabled: true,
			},
		},
		GoogleOAuthClientID:     "test-client-id",
		GoogleOAuthClientSecret: "test-client-secret",
		GoogleOAuthRedirectURL:  "http://localhost:3000/oauth-callback",
	}

	// Mock Google OAuth endpoints
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"fake-access-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	userinfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","email":"test@example.com","verified_email":true}`))
	}))
	defer userinfoServer.Close()

	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	oauthService.TokenEndpoint = tokenServer.URL
	oauthService.UserInfoEndpoint = userinfoServer.URL

	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a new router for this test with session middleware
	gin.SetMode(gin.TestMode)
	testRouter := gin.New()

	// Setup session middleware
	store := cookie.NewStore([]byte("test-secret"))
	testRouter.Use(sessions.Sessions("test-session", store))

	testRouter.GET("/oauth/callback", handler.GoogleCallback)

	// First, set up a session with a valid OAuth state
	testRouter.GET("/setup-oauth-state", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("oauth_state", "test-state")
		_ = session.Save()
		c.JSON(http.StatusOK, gin.H{"message": "oauth state set"})
	})

	setupReq, _ := http.NewRequest("GET", "/setup-oauth-state", nil)
	setupW := httptest.NewRecorder()
	testRouter.ServeHTTP(setupW, setupReq)
	assert.Equal(t, http.StatusOK, setupW.Code)

	// Extract session cookie
	cookies := setupW.Result().Cookies()
	require.NotEmpty(t, cookies)
	sessionCookie := cookies[0]

	// Test Google OAuth callback with signups disabled
	req, _ := http.NewRequest("GET", "/oauth/callback?code=fake-code&state=test-state", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	// Should return 403 Forbidden when signups are disabled
	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.Equal(t, "FORBIDDEN", response["code"])
}

func TestAuthHandler_GoogleLogin_WithRedirectURI(t *testing.T) {
	mockUserService := new(MockUserService)

	cfg := &config.Config{
		Server: config.ServerConfig{
			SessionSecret: "test-secret",
		},
		GoogleOAuthClientID:     "test-client-id",
		GoogleOAuthClientSecret: "test-client-secret",
		GoogleOAuthRedirectURL:  "http://localhost:3000/oauth-callback",
	}

	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a new router for this test with session middleware
	gin.SetMode(gin.TestMode)
	testRouter := gin.New()

	// Setup session middleware
	store := cookie.NewStore([]byte("test-secret"))
	testRouter.Use(sessions.Sessions("test-session", store))

	testRouter.GET("/auth/google/login", handler.GoogleLogin)

	// Test Google OAuth login with redirect_uri parameter
	req, _ := http.NewRequest("GET", "/auth/google/login?redirect_uri=%2Fdaily", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.Contains(t, response, "auth_url")
	assert.True(t, len(response["auth_url"].(string)) > 0)

	// Verify that the redirect_uri was stored in session
	// We can't directly access the session in the test, but we can verify the response
	// and test the callback functionality separately
}

func TestAuthHandler_GoogleLogin_WithoutRedirectURI(t *testing.T) {
	mockUserService := new(MockUserService)

	cfg := &config.Config{
		Server: config.ServerConfig{
			SessionSecret: "test-secret",
		},
		GoogleOAuthClientID:     "test-client-id",
		GoogleOAuthClientSecret: "test-client-secret",
		GoogleOAuthRedirectURL:  "http://localhost:3000/oauth-callback",
	}

	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a new router for this test with session middleware
	gin.SetMode(gin.TestMode)
	testRouter := gin.New()

	// Setup session middleware
	store := cookie.NewStore([]byte("test-secret"))
	testRouter.Use(sessions.Sessions("test-session", store))

	testRouter.GET("/auth/google/login", handler.GoogleLogin)

	// Test Google OAuth login without redirect_uri parameter
	req, _ := http.NewRequest("GET", "/auth/google/login", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.Contains(t, response, "auth_url")
	assert.True(t, len(response["auth_url"].(string)) > 0)
}

func TestAuthHandler_GoogleCallback_WithRedirectURI(t *testing.T) {
	mockUserService := new(MockUserService)

	// Create a test user
	testUser := &models.User{
		ID:                1,
		Username:          "testuser",
		Email:             sql.NullString{String: "test@example.com", Valid: true},
		PreferredLanguage: sql.NullString{String: "italian", Valid: true},
		CurrentLevel:      sql.NullString{String: "B1", Valid: true},
	}

	// Expect GetUserByEmail to be called and return the test user
	mockUserService.On("GetUserByEmail", mock.Anything, "test@example.com").Return(testUser, nil)
	mockUserService.On("UpdateLastActive", mock.Anything, 1).Return(nil)

	cfg := &config.Config{
		Server: config.ServerConfig{
			SessionSecret: "test-secret",
		},
		GoogleOAuthClientID:     "test-client-id",
		GoogleOAuthClientSecret: "test-client-secret",
		GoogleOAuthRedirectURL:  "http://localhost:3000/oauth-callback",
	}

	// Mock Google OAuth endpoints
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"fake-access-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	userinfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","email":"test@example.com","verified_email":true}`))
	}))
	defer userinfoServer.Close()

	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	oauthService.TokenEndpoint = tokenServer.URL
	oauthService.UserInfoEndpoint = userinfoServer.URL

	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a new router for this test with session middleware
	gin.SetMode(gin.TestMode)
	testRouter := gin.New()

	// Setup session middleware
	store := cookie.NewStore([]byte("test-secret"))
	testRouter.Use(sessions.Sessions("test-session", store))

	testRouter.GET("/oauth/callback", handler.GoogleCallback)

	// First, set up a session with a valid OAuth state and redirect_uri
	testRouter.GET("/setup-oauth-state", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("oauth_state", "test-state")
		session.Set("oauth_redirect_uri", "/daily")
		_ = session.Save()
		c.JSON(http.StatusOK, gin.H{"message": "oauth state set"})
	})

	setupReq, _ := http.NewRequest("GET", "/setup-oauth-state", nil)
	setupW := httptest.NewRecorder()
	testRouter.ServeHTTP(setupW, setupReq)
	assert.Equal(t, http.StatusOK, setupW.Code)

	// Extract session cookie
	cookies := setupW.Result().Cookies()
	require.NotEmpty(t, cookies)
	sessionCookie := cookies[0]

	// Test Google OAuth callback with redirect_uri stored in session
	req, _ := http.NewRequest("GET", "/oauth/callback?code=fake-code&state=test-state", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.Equal(t, true, response["success"])
	assert.Equal(t, "Google authentication successful", response["message"])
	assert.Contains(t, response, "user")
	assert.Equal(t, "/daily", response["redirect_uri"])

	mockUserService.AssertExpectations(t)
}

func TestAuthHandler_GoogleCallback_WithoutRedirectURI(t *testing.T) {
	mockUserService := new(MockUserService)

	// Create a test user
	testUser := &models.User{
		ID:                1,
		Username:          "testuser",
		Email:             sql.NullString{String: "test@example.com", Valid: true},
		PreferredLanguage: sql.NullString{String: "italian", Valid: true},
		CurrentLevel:      sql.NullString{String: "B1", Valid: true},
	}

	// Expect GetUserByEmail to be called and return the test user
	mockUserService.On("GetUserByEmail", mock.Anything, "test@example.com").Return(testUser, nil)
	mockUserService.On("UpdateLastActive", mock.Anything, 1).Return(nil)

	cfg := &config.Config{
		Server: config.ServerConfig{
			SessionSecret: "test-secret",
		},
		GoogleOAuthClientID:     "test-client-id",
		GoogleOAuthClientSecret: "test-client-secret",
		GoogleOAuthRedirectURL:  "http://localhost:3000/oauth-callback",
	}

	// Mock Google OAuth endpoints
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"fake-access-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	userinfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","email":"test@example.com","verified_email":true}`))
	}))
	defer userinfoServer.Close()

	oauthService := services.NewOAuthServiceWithLogger(cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	oauthService.TokenEndpoint = tokenServer.URL
	oauthService.UserInfoEndpoint = userinfoServer.URL

	handler := NewAuthHandler(mockUserService, oauthService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a new router for this test with session middleware
	gin.SetMode(gin.TestMode)
	testRouter := gin.New()

	// Setup session middleware
	store := cookie.NewStore([]byte("test-secret"))
	testRouter.Use(sessions.Sessions("test-session", store))

	testRouter.GET("/oauth/callback", handler.GoogleCallback)

	// First, set up a session with a valid OAuth state but no redirect_uri
	testRouter.GET("/setup-oauth-state", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("oauth_state", "test-state")
		_ = session.Save()
		c.JSON(http.StatusOK, gin.H{"message": "oauth state set"})
	})

	setupReq, _ := http.NewRequest("GET", "/setup-oauth-state", nil)
	setupW := httptest.NewRecorder()
	testRouter.ServeHTTP(setupW, setupReq)
	assert.Equal(t, http.StatusOK, setupW.Code)

	// Extract session cookie
	cookies := setupW.Result().Cookies()
	require.NotEmpty(t, cookies)
	sessionCookie := cookies[0]

	// Test Google OAuth callback without redirect_uri stored in session
	req, _ := http.NewRequest("GET", "/oauth/callback?code=fake-code&state=test-state", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.Equal(t, true, *response.Success)
	assert.Equal(t, "Google authentication successful", *response.Message)
	assert.NotNil(t, response.User)

	mockUserService.AssertExpectations(t)
}
