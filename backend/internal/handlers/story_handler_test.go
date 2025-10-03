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
	"os"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/di"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// StoryHandlerIntegrationTestSuite provides comprehensive integration tests for the StoryHandler
type StoryHandlerIntegrationTestSuite struct {
	suite.Suite
	Config    *config.Config
	Logger    *observability.Logger
	Container di.ServiceContainerInterface
}

func TestStoryHandlerIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(StoryHandlerIntegrationTestSuite))
}

func (suite *StoryHandlerIntegrationTestSuite) SetupSuite() {
	// Set up test database URL
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		testDatabaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}
	os.Setenv("DATABASE_URL", testDatabaseURL)

	// Set config file path to project root
	os.Setenv("QUIZ_CONFIG_FILE", "../../../config.yaml")

	// Initialize logger
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Load configuration
	cfg, err := config.NewConfig()
	require.NoError(suite.T(), err)
	suite.Config = cfg

	// Setup observability with noop telemetry for tests
	suite.Logger = logger

	// Initialize dependency injection container
	suite.Container = di.NewServiceContainer(cfg, suite.Logger)

	// Initialize all services
	ctx := context.Background()
	err = suite.Container.Initialize(ctx)
	require.NoError(suite.T(), err)

	// Ensure admin user exists
	err = suite.Container.EnsureAdminUser(ctx)
	require.NoError(suite.T(), err)
}

func (suite *StoryHandlerIntegrationTestSuite) TearDownSuite() {
	if suite.Container != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		suite.Container.Shutdown(ctx)
	}
}

func (suite *StoryHandlerIntegrationTestSuite) TestStoryHandler_CreateStory_Integration() {
	// Get services from DI container
	userService, err := suite.Container.GetUserService()
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), userService)

	storyService, err := suite.Container.GetStoryService()
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), storyService)

	aiService, err := suite.Container.GetAIService()
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), aiService)

	// Create a test user
	ctx := context.Background()
	user := models.User{
		Username:          "testuser_story",
		Email:             sql.NullString{String: "test_story@example.com", Valid: true},
		PreferredLanguage: sql.NullString{String: "en", Valid: true},
	}

	err = suite.Container.GetDatabase().QueryRowContext(ctx,
		`INSERT INTO users (username, email, preferred_language, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id`,
		user.Username, user.Email, user.PreferredLanguage).Scan(&user.ID)
	require.NoError(suite.T(), err)

	// Create handler with real services
	handler := NewStoryHandler(storyService, userService, aiService, suite.Config, suite.Logger)

	router := gin.New()
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("test_session", store))

	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("user_id", user.ID)
		c.Next()
	})

	router.POST("/v1/story", handler.CreateStory)

	suite.Run("should create story successfully", func() {
		reqData := models.CreateStoryRequest{
			Title: "Test Story",
		}

		body, _ := json.Marshal(reqData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		// Should succeed with real services
		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		// Verify response structure
		var response models.Story
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Test Story", response.Title)
		assert.Equal(suite.T(), user.ID, response.UserID)
	})

	suite.Run("should get current story successfully", func() {
		// Create a story in the database first
		_, err := suite.Container.GetDatabase().ExecContext(ctx,
			`INSERT INTO stories (user_id, title, language, status, is_current, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`,
			user.ID, "Test Story Current", "en", "active", true)
		require.NoError(suite.T(), err)

		// Create handler for GET request
		router.GET("/v1/story/current", handler.GetCurrentStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/story/current", nil)

		router.ServeHTTP(w, req)

		// Should succeed with real services
		assert.Equal(suite.T(), http.StatusOK, w.Code)

		// Verify response structure
		var response models.Story
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Test Story Current", response.Title)
		assert.Equal(suite.T(), user.ID, response.UserID)
	})
}
