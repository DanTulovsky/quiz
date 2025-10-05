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
	"strconv"
	"testing"
	"time"

	"quizapp/internal/api"
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
		`INSERT INTO users (username, email, preferred_language, current_level, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id`,
		user.Username, user.Email, user.PreferredLanguage, "B1").Scan(&user.ID)
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
		assert.Equal(suite.T(), uint(user.ID), response.UserID)
	})

	suite.Run("should get current story successfully", func() {
		// The story from the first test should already be the current story
		// If not, we need to ensure there's a current story for this user
		var storyCount int
		err := suite.Container.GetDatabase().QueryRowContext(ctx,
			"SELECT COUNT(*) FROM stories WHERE user_id = $1 AND is_current = true",
			user.ID).Scan(&storyCount)
		require.NoError(suite.T(), err)

		if storyCount == 0 {
			// Create a current story if none exists
			_, err = suite.Container.GetDatabase().ExecContext(ctx,
				`INSERT INTO stories (user_id, title, language, status, is_current, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`,
				user.ID, "Test Story Current", "en", "active", true)
			require.NoError(suite.T(), err)
		}

		// Create handler for GET request
		router.GET("/v1/story/current", handler.GetCurrentStory)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/v1/story/current", nil)

		router.ServeHTTP(w, req)

		// Should return 202 Accepted since story exists but has no sections yet
		assert.Equal(suite.T(), http.StatusAccepted, w.Code)

		// Verify response structure for generating status
		var generatingResponse api.GeneratingResponse
		err = json.Unmarshal(w.Body.Bytes(), &generatingResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "generating", *generatingResponse.Status)
		assert.Contains(suite.T(), *generatingResponse.Message, "Story created successfully")
	})

	suite.Run("should archive story successfully", func() {
		// First, ensure there's a current story for this user
		var storyID uint
		err := suite.Container.GetDatabase().QueryRowContext(ctx,
			"SELECT id FROM stories WHERE user_id = $1 AND is_current = true LIMIT 1",
			user.ID).Scan(&storyID)
		if err != nil || storyID == 0 {
			// Create a current story if none exists
			err = suite.Container.GetDatabase().QueryRowContext(ctx,
				`INSERT INTO stories (user_id, title, language, status, is_current, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id`,
				user.ID, "Story to Archive", "en", "active", true).Scan(&storyID)
			require.NoError(suite.T(), err)
		}

		// Create handler for archive request
		router.POST("/v1/story/:id/archive", handler.ArchiveStory)

		// Archive the story
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story/"+strconv.Itoa(int(storyID))+"/archive", nil)

		router.ServeHTTP(w, req)

		// Should succeed
		assert.Equal(suite.T(), http.StatusOK, w.Code)

		// Verify the story is now archived and not current
		var archivedStory models.Story
		err = suite.Container.GetDatabase().QueryRowContext(ctx,
			"SELECT id, status, is_current FROM stories WHERE id = $1",
			storyID).Scan(&archivedStory.ID, &archivedStory.Status, &archivedStory.IsCurrent)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), models.StoryStatusArchived, archivedStory.Status)
		assert.False(suite.T(), archivedStory.IsCurrent)
	})

	suite.Run("should handle language-based story filtering", func() {
		// Get services from DI container
		userService, err := suite.Container.GetUserService()
		require.NoError(suite.T(), err)

		storyService, err := suite.Container.GetStoryService()
		require.NoError(suite.T(), err)

		aiService, err := suite.Container.GetAIService()
		require.NoError(suite.T(), err)

		// Create a test user with Italian as initial language
		ctx := context.Background()
		user := models.User{
			Username:          "testuser_language_switch",
			Email:             sql.NullString{String: "test_language_switch@example.com", Valid: true},
			PreferredLanguage: sql.NullString{String: "it", Valid: true}, // Italian
		}

		err = suite.Container.GetDatabase().QueryRowContext(ctx,
			`INSERT INTO users (username, email, preferred_language, current_level, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id`,
			user.Username, user.Email, user.PreferredLanguage, "B1").Scan(&user.ID)
		require.NoError(suite.T(), err)

		handler := NewStoryHandler(storyService, userService, aiService, suite.Config, suite.Logger)

		router := gin.New()
		store := cookie.NewStore([]byte("secret"))
		router.Use(sessions.Sessions("test_session", store))

		// Helper function to set user session
		setUserSession := func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set("user_id", user.ID)
			c.Next()
		}

		router.Use(setUserSession)
		router.POST("/v1/story", handler.CreateStory)
		router.GET("/v1/story/current", handler.GetCurrentStory)

		// Step 1: Create a story in Italian
		reqData := models.CreateStoryRequest{
			Title: "Storia Italiana",
		}

		body, _ := json.Marshal(reqData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)
		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var italianStory models.Story
		err = json.Unmarshal(w.Body.Bytes(), &italianStory)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Storia Italiana", italianStory.Title)
		assert.Equal(suite.T(), "it", italianStory.Language)
		assert.True(suite.T(), italianStory.IsCurrent)

		// Step 2: Verify current story returns the Italian story
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/v1/story/current", nil)
		router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)
		var currentStory models.Story
		err = json.Unmarshal(w.Body.Bytes(), &currentStory)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Storia Italiana", currentStory.Title)

		// Step 3: Change user's language to Russian
		_, err = suite.Container.GetDatabase().ExecContext(ctx,
			"UPDATE users SET preferred_language = $1, updated_at = NOW() WHERE id = $2",
			"ru", user.ID)
		require.NoError(suite.T(), err)

		// Step 4: Verify current story now returns null (no Russian story exists)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/v1/story/current", nil)
		router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusNotFound, w.Code)

		// Step 5: Create a story in Russian
		reqData = models.CreateStoryRequest{
			Title: "Русская История",
		}

		body, _ = json.Marshal(reqData)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/v1/story", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)
		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var russianStory models.Story
		err = json.Unmarshal(w.Body.Bytes(), &russianStory)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Русская История", russianStory.Title)
		assert.Equal(suite.T(), "ru", russianStory.Language)
		assert.True(suite.T(), russianStory.IsCurrent)

		// Step 6: Verify current story returns the Russian story
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/v1/story/current", nil)
		router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)
		err = json.Unmarshal(w.Body.Bytes(), &currentStory)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Русская История", currentStory.Title)

		// Step 7: Switch back to Italian
		_, err = suite.Container.GetDatabase().ExecContext(ctx,
			"UPDATE users SET preferred_language = $1, updated_at = NOW() WHERE id = $2",
			"it", user.ID)
		require.NoError(suite.T(), err)

		// Step 8: Verify current story returns the Italian story again
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/v1/story/current", nil)
		router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)
		err = json.Unmarshal(w.Body.Bytes(), &currentStory)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Storia Italiana", currentStory.Title)

		// Step 9: Verify both stories exist in the database
		var storyCount int
		err = suite.Container.GetDatabase().QueryRowContext(ctx,
			"SELECT COUNT(*) FROM stories WHERE user_id = $1", user.ID).Scan(&storyCount)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 2, storyCount)

		// Step 10: Verify both stories are current (one per language)
		var stories []struct {
			ID        uint
			Title     string
			Language  string
			IsCurrent bool
		}
		rows, err := suite.Container.GetDatabase().QueryContext(ctx,
			"SELECT id, title, language, is_current FROM stories WHERE user_id = $1", user.ID)
		require.NoError(suite.T(), err)
		defer rows.Close()

		for rows.Next() {
			var story struct {
				ID        uint
				Title     string
				Language  string
				IsCurrent bool
			}
			err = rows.Scan(&story.ID, &story.Title, &story.Language, &story.IsCurrent)
			require.NoError(suite.T(), err)
			stories = append(stories, story)
		}

		// Debug: Print the stories
		for _, story := range stories {
			suite.T().Logf("Story: ID=%d, Title=%s, Language=%s, IsCurrent=%v", story.ID, story.Title, story.Language, story.IsCurrent)
		}

		var currentCount int
		err = suite.Container.GetDatabase().QueryRowContext(ctx,
			"SELECT COUNT(*) FROM stories WHERE user_id = $1 AND is_current = true", user.ID).Scan(&currentCount)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 2, currentCount) // Both Italian and Russian stories should be current in their respective languages

		// Verify we have current stories in both languages
		var italianCurrent, russianCurrent bool
		for _, story := range stories {
			if story.Language == "it" && story.IsCurrent {
				italianCurrent = true
			}
			if story.Language == "ru" && story.IsCurrent {
				russianCurrent = true
			}
		}
		assert.True(suite.T(), italianCurrent, "Italian story should be current")
		assert.True(suite.T(), russianCurrent, "Russian story should be current")
	})
}
