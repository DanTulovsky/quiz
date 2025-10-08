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

// StoryHandlerArchivedIntegrationTestSuite provides comprehensive integration tests for archived stories functionality
type StoryHandlerArchivedIntegrationTestSuite struct {
	suite.Suite
	Config    *config.Config
	Logger    *observability.Logger
	Container di.ServiceContainerInterface
}

func TestStoryHandlerArchivedIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(StoryHandlerArchivedIntegrationTestSuite))
}

func (suite *StoryHandlerArchivedIntegrationTestSuite) SetupSuite() {
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

func (suite *StoryHandlerArchivedIntegrationTestSuite) TearDownSuite() {
	if suite.Container != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		suite.Container.Shutdown(ctx)
	}
}

func (suite *StoryHandlerArchivedIntegrationTestSuite) TestStoryHandler_ArchivedStories_Integration() {
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
		Username:          "testuser_archived_stories",
		Email:             sql.NullString{String: "test_archived_stories@example.com", Valid: true},
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

	// Helper function to set user session
	setUserSession := func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("user_id", user.ID)
		c.Next()
	}

	router.Use(setUserSession)

	// Register all routes for this test suite
	router.POST("/v1/story", handler.CreateStory)
	router.GET("/v1/story", handler.GetUserStories)
	router.POST("/v1/story/:id/archive", handler.ArchiveStory)
	router.POST("/v1/story/:id/set-current", handler.SetCurrentStory)
	router.POST("/v1/story/:id/complete", handler.CompleteStory)
	router.GET("/v1/story/current", handler.GetCurrentStory)

	suite.Run("should create and archive a story successfully", func() {
		// Step 1: Create a story
		reqData := models.CreateStoryRequest{
			Title: "Story to Archive",
		}

		body, _ := json.Marshal(reqData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)
		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var createdStory models.Story
		err = json.Unmarshal(w.Body.Bytes(), &createdStory)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Story to Archive", createdStory.Title)
		assert.Equal(suite.T(), "en", createdStory.Language)
		assert.Equal(suite.T(), models.StoryStatusActive, createdStory.Status)
		assert.True(suite.T(), createdStory.Status == models.StoryStatusActive)

		// Step 2: Archive the story
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/v1/story/"+strconv.Itoa(int(createdStory.ID))+"/archive", nil)
		router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		// Step 3: Verify the story is archived
		var archivedStory models.Story
		err = suite.Container.GetDatabase().QueryRowContext(ctx,
			"SELECT id, status FROM stories WHERE id = $1",
			createdStory.ID).Scan(&archivedStory.ID, &archivedStory.Status)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), models.StoryStatusArchived, archivedStory.Status)
		assert.False(suite.T(), archivedStory.Status == models.StoryStatusActive)
	})

	suite.Run("should get archived stories successfully", func() {
		// First, create and archive a story
		reqData := models.CreateStoryRequest{
			Title: "Archived Story 1",
		}

		body, _ := json.Marshal(reqData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var story models.Story
		err = json.Unmarshal(w.Body.Bytes(), &story)
		require.NoError(suite.T(), err)

		// Archive the story
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/v1/story/"+strconv.Itoa(int(story.ID))+"/archive", nil)
		router.ServeHTTP(w, req)
		assert.Equal(suite.T(), http.StatusOK, w.Code)

		// Get all stories including archived
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/v1/story?include_archived=true", nil)
		router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var stories []models.Story
		err = json.Unmarshal(w.Body.Bytes(), &stories)
		require.NoError(suite.T(), err)

		// Should include the archived story
		found := false
		for _, s := range stories {
			if s.ID == story.ID && s.Status == models.StoryStatusArchived {
				found = true
				break
			}
		}
		assert.True(suite.T(), found, "Archived story should be included in response")
	})

	suite.Run("should restore archived story in same language successfully", func() {
		// Create a story in English (same as user's preferred language)
		reqData := models.CreateStoryRequest{
			Title: "Story to Restore",
		}

		body, _ := json.Marshal(reqData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var story models.Story
		err = json.Unmarshal(w.Body.Bytes(), &story)
		require.NoError(suite.T(), err)

		// Archive the story
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/v1/story/"+strconv.Itoa(int(story.ID))+"/archive", nil)
		router.ServeHTTP(w, req)
		assert.Equal(suite.T(), http.StatusOK, w.Code)

		// Verify it's archived
		var archivedStory models.Story
		err = suite.Container.GetDatabase().QueryRowContext(ctx,
			"SELECT id, status FROM stories WHERE id = $1",
			story.ID).Scan(&archivedStory.ID, &archivedStory.Status)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), models.StoryStatusArchived, archivedStory.Status)
		assert.False(suite.T(), archivedStory.Status == models.StoryStatusActive)

		// Restore the story
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/v1/story/"+strconv.Itoa(int(story.ID))+"/set-current", nil)
		router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		// Verify the story is now current
		var restoredStory models.Story
		err = suite.Container.GetDatabase().QueryRowContext(ctx,
			"SELECT id, status FROM stories WHERE id = $1",
			story.ID).Scan(&restoredStory.ID, &restoredStory.Status)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), models.StoryStatusActive, restoredStory.Status)
		assert.True(suite.T(), restoredStory.Status == models.StoryStatusActive)
	})

	suite.Run("should prevent archiving completed stories", func() {
		// Create and complete a story
		reqData := models.CreateStoryRequest{
			Title: "Story to Complete",
		}

		body, _ := json.Marshal(reqData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var story models.Story
		err = json.Unmarshal(w.Body.Bytes(), &story)
		require.NoError(suite.T(), err)

		// Complete the story
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/v1/story/"+strconv.Itoa(int(story.ID))+"/complete", nil)
		router.ServeHTTP(w, req)
		assert.Equal(suite.T(), http.StatusOK, w.Code)

		// Try to archive the completed story (should fail)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/v1/story/"+strconv.Itoa(int(story.ID))+"/archive", nil)
		router.ServeHTTP(w, req)

		// Should return an error
		assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

		// Verify the story is still completed
		var stillCompletedStory models.Story
		err = suite.Container.GetDatabase().QueryRowContext(ctx,
			"SELECT id, status FROM stories WHERE id = $1",
			story.ID).Scan(&stillCompletedStory.ID, &stillCompletedStory.Status)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), models.StoryStatusCompleted, stillCompletedStory.Status)
		assert.False(suite.T(), stillCompletedStory.Status == models.StoryStatusActive)
	})

	suite.Run("should filter archived stories by user's preferred language", func() {
		// Create and archive an English story (user's preferred language)
		reqData := models.CreateStoryRequest{
			Title: "English Archived Story",
		}

		body, _ := json.Marshal(reqData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/v1/story", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var englishStory models.Story
		err = json.Unmarshal(w.Body.Bytes(), &englishStory)
		require.NoError(suite.T(), err)

		// Archive the English story
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/v1/story/"+strconv.Itoa(int(englishStory.ID))+"/archive", nil)
		router.ServeHTTP(w, req)
		assert.Equal(suite.T(), http.StatusOK, w.Code)

		// Get archived stories (should include the English story)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/v1/story?include_archived=true", nil)
		router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var archivedStories []models.Story
		err = json.Unmarshal(w.Body.Bytes(), &archivedStories)
		require.NoError(suite.T(), err)

		// Should include the English archived story
		var englishStoryFound bool
		for _, story := range archivedStories {
			if story.Title == "English Archived Story" && story.Status == models.StoryStatusArchived {
				englishStoryFound = true
				break
			}
		}
		assert.True(suite.T(), englishStoryFound, "English archived story should be included")

		// Verify that only stories in the user's preferred language are included
		// (Since all stories are created in the user's preferred language, they should all be included when archived)
		for _, story := range archivedStories {
			if story.Status == models.StoryStatusArchived {
				assert.Equal(suite.T(), "en", story.Language, "Archived story should be in user's preferred language")
			}
		}
	})
}
