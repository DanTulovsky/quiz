//go:build integration

package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/middleware"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDatabase(t *testing.T) *sql.DB {
	// Use environment variable for test database URL, fallback to test port 5433
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := database.NewManager(logger)
	db, err := dbManager.InitDB(databaseURL)
	require.NoError(t, err)
	return db
}

func setupSettingsIntegrationTest(t *testing.T) (*gin.Engine, *services.UserService, *services.LearningService, *services.EmailService, func()) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup session middleware
	store := cookie.NewStore([]byte("test-secret"))
	// Configure session options for security
	sessionOpts := sessions.Options{
		Path:     config.SessionPath,
		MaxAge:   int(config.SessionMaxAge.Seconds()),
		HttpOnly: config.SessionHTTPOnly,
		Secure:   config.SessionSecure,
	}
	store.Options(sessionOpts)
	router.Use(sessions.Sessions(config.SessionName, store))

	// Load test configuration
	cfg, err := config.NewConfig()
	require.NoError(t, err)

	// Create test database
	db := setupTestDatabase(t)

	// Create services
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	aiService := services.NewAIService(cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	emailService := services.NewEmailService(cfg, logger)
	storyService := services.NewStoryService(db, cfg, logger)
	conversationService := services.NewConversationService(db)

	// Create settings handler
	settingsHandler := NewSettingsHandler(userService, storyService, conversationService, aiService, learningService, emailService, cfg, logger)

	// Setup routes
	v1 := router.Group("/v1")
	settings := v1.Group("/settings")
	{
		settings.POST("/test-email", middleware.RequireAuth(), settingsHandler.SendTestEmail)
		settings.GET("/api-key/:provider", middleware.RequireAuth(), settingsHandler.CheckAPIKeyAvailability)
	}

	// Add a setup endpoint for creating sessions in tests
	router.GET("/setup-session", func(c *gin.Context) {
		userIDStr := c.Query("user_id")
		username := c.Query("username")
		if userIDStr != "" && username != "" {
			// Convert userID to integer
			var userID int
			fmt.Sscanf(userIDStr, "%d", &userID)
			session := sessions.Default(c)
			session.Set("user_id", userID)
			session.Set("username", username)
			session.Save()
		}
		c.JSON(http.StatusOK, gin.H{"message": "session set"})
	})

	cleanup := func() {
		db.Close()
	}

	return router, userService, learningService, emailService, cleanup
}

func createTestUserWithEmail(t *testing.T, userService *services.UserService) *models.User {
	user, err := userService.CreateUserWithEmailAndTimezone(
		context.Background(),
		"testuser_email",
		"test@example.com",
		"UTC",
		"english",
		"A1",
	)
	require.NoError(t, err)
	require.NotNil(t, user)
	return user
}

func TestSettingsHandler_SendTestEmail_Integration(t *testing.T) {
	_, userService, _, _, cleanup := setupSettingsIntegrationTest(t)
	defer cleanup()

	// Create test user with email
	user := createTestUserWithEmail(t, userService)

	// Test cases
	tests := []struct {
		name          string
		emailEnabled  bool
		emailConfig   *config.EmailConfig
		expectedError bool
	}{
		{
			name:         "email service disabled",
			emailEnabled: false,
			emailConfig: &config.EmailConfig{
				Enabled: false,
			},
			expectedError: false,
		},
		{
			name:         "email service enabled but no SMTP host",
			emailEnabled: true,
			emailConfig: &config.EmailConfig{
				Enabled: true,
				SMTP: config.SMTPConfig{
					Host: "", // Empty host
				},
			},
			expectedError: false,
		},
		{
			name:         "email service properly configured",
			emailEnabled: true,
			emailConfig: &config.EmailConfig{
				Enabled: true,
				SMTP: config.SMTPConfig{
					Host:        "smtp.example.com",
					Port:        587,
					Username:    "test@example.com",
					Password:    "password",
					FromAddress: "noreply@example.com",
					FromName:    "Test App",
				},
			},
			expectedError: true, // Will fail due to invalid SMTP config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new email service with the test configuration
			testCfg := &config.Config{
				Email: *tt.emailConfig,
			}
			testEmailService := services.NewEmailService(testCfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

			// Test IsEnabled
			enabled := testEmailService.IsEnabled()
			if tt.emailConfig.Enabled && tt.emailConfig.SMTP.Host != "" {
				assert.True(t, enabled)
			} else {
				assert.False(t, enabled)
			}

			// Test SendEmail directly
			err := testEmailService.SendEmail(
				context.Background(),
				user.Email.String,
				"Test Email from Quiz App",
				"test_email",
				map[string]interface{}{
					"Username": user.Username,
					"TestTime": "now",
					"Message":  "This is a test email to verify your email settings are working correctly.",
				},
			)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				// If email is disabled, it should not error
				if !testEmailService.IsEnabled() {
					assert.NoError(t, err)
				} else {
					// If enabled but with invalid config, it should error
					assert.Error(t, err)
				}
			}
		})
	}
}

func TestSettingsHandler_SendTestEmail_UserWithoutEmail(t *testing.T) {
	router, userService, _, _, cleanup := setupSettingsIntegrationTest(t)
	defer cleanup()

	// Create test user without email
	user, err := userService.CreateUserWithEmailAndTimezone(
		context.Background(),
		"testuser_no_email",
		"", // No email
		"UTC",
		"english",
		"A1",
	)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Setup session with user
	setupReq, _ := http.NewRequest("GET", fmt.Sprintf("/setup-session?user_id=%d&username=%s", user.ID, user.Username), nil)
	setupW := httptest.NewRecorder()
	router.ServeHTTP(setupW, setupReq)
	assert.Equal(t, http.StatusOK, setupW.Code)

	// Extract session cookie
	cookies := setupW.Result().Cookies()
	require.NotEmpty(t, cookies)
	sessionCookie := cookies[0]

	// Create test request with session
	req, _ := http.NewRequest("POST", "/v1/settings/test-email", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "MISSING_REQUIRED_FIELD", response["code"])
}

func TestSettingsHandler_SendTestEmail_Unauthenticated(t *testing.T) {
	router, _, _, _, cleanup := setupSettingsIntegrationTest(t)
	defer cleanup()

	// Create test request without authentication
	req, _ := http.NewRequest("POST", "/v1/settings/test-email", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestEmailService_Integration(t *testing.T) {
	// Test email service with various configurations
	tests := []struct {
		name          string
		cfg           *config.Config
		expectedError bool
	}{
		{
			name: "email disabled",
			cfg: &config.Config{
				Email: config.EmailConfig{
					Enabled: false,
				},
			},
			expectedError: false,
		},
		{
			name: "email enabled but no SMTP host",
			cfg: &config.Config{
				Email: config.EmailConfig{
					Enabled: true,
					SMTP: config.SMTPConfig{
						Host: "", // Empty host
					},
				},
			},
			expectedError: false, // Should not error, just not be enabled
		},
		{
			name: "email enabled with invalid SMTP config",
			cfg: &config.Config{
				Email: config.EmailConfig{
					Enabled: true,
					SMTP: config.SMTPConfig{
						Host:        "invalid-host",
						Port:        587,
						Username:    "test@example.com",
						Password:    "password",
						FromAddress: "noreply@example.com",
						FromName:    "Test App",
					},
				},
			},
			expectedError: true, // Should error due to invalid SMTP config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
			emailService := services.NewEmailService(tt.cfg, logger)

			// Test IsEnabled
			enabled := emailService.IsEnabled()
			if tt.cfg.Email.Enabled && tt.cfg.Email.SMTP.Host != "" {
				assert.True(t, enabled)
			} else {
				assert.False(t, enabled)
			}

			// Test SendEmail with test template
			user := &models.User{
				ID:       1,
				Username: "testuser",
				Email:    sql.NullString{String: "test@example.com", Valid: true},
			}

			err := emailService.SendEmail(
				context.Background(),
				"test@example.com",
				"Test Email",
				"test_email",
				map[string]interface{}{
					"Username": user.Username,
					"TestTime": "now",
					"Message":  "This is a test email",
				},
			)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				// If email is disabled, it should not error
				if !emailService.IsEnabled() {
					assert.NoError(t, err)
				} else {
					// If enabled but with invalid config, it should error
					assert.Error(t, err)
				}
			}
		})
	}
}
