//go:build integration
// +build integration

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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

// StoryAutoGenerationSimpleIntegrationTestSuite provides focused integration tests for auto-generation toggle functionality
type StoryAutoGenerationSimpleIntegrationTestSuite struct {
	suite.Suite
	Config    *config.Config
	Logger    *observability.Logger
	Container di.ServiceContainerInterface
}

func TestStoryAutoGenerationSimpleIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(StoryAutoGenerationSimpleIntegrationTestSuite))
}

func (suite *StoryAutoGenerationSimpleIntegrationTestSuite) SetupSuite() {
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

func (suite *StoryAutoGenerationSimpleIntegrationTestSuite) TearDownSuite() {
	if suite.Container != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		suite.Container.Shutdown(ctx)
	}
}

func (suite *StoryAutoGenerationSimpleIntegrationTestSuite) setupRouter(userID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add session middleware
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("test-session", store))

	// Add auth middleware that sets user ID in session
	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("user_id", userID)
		session.Save()
		c.Set("user_id", userID)
		c.Next()
	})

	// Get services from DI container
	userService, err := suite.Container.GetUserService()
	require.NoError(suite.T(), err)
	storyService, err := suite.Container.GetStoryService()
	require.NoError(suite.T(), err)
	aiService, err := suite.Container.GetAIService()
	require.NoError(suite.T(), err)

	// Setup story handler
	storyHandler := NewStoryHandler(
		storyService,
		userService,
		aiService,
		suite.Config,
		suite.Logger,
	)

	// Setup routes
	storyGroup := router.Group("/v1/story")
	storyGroup.POST("/:id/toggle-auto-generation", storyHandler.ToggleAutoGeneration)
	storyGroup.GET("/:id", storyHandler.GetStory)

	return router
}

func (suite *StoryAutoGenerationSimpleIntegrationTestSuite) createTestUser() uint {
	userService, err := suite.Container.GetUserService()
	require.NoError(suite.T(), err)
	
	// Use a unique username based on timestamp to avoid conflicts
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	email := fmt.Sprintf("test_%d@example.com", time.Now().UnixNano())
	
	user, err := userService.CreateUserWithEmailAndTimezone(
		context.Background(),
		username,
		email,
		"UTC",
		"en",
		"A1",
	)
	require.NoError(suite.T(), err)
	return uint(user.ID)
}

func (suite *StoryAutoGenerationSimpleIntegrationTestSuite) createTestStory(userID uint) uint {
	storyService, err := suite.Container.GetStoryService()
	require.NoError(suite.T(), err)

	subject := "Test subject"
	authorStyle := "Test style"
	timePeriod := "Test period"
	genre := "Test genre"
	tone := "Test tone"
	characterNames := "Test characters"
	customInstructions := "Test instructions"
	sectionLength := models.SectionLengthMedium

	req := &models.CreateStoryRequest{
		Title:                 "Test Story",
		Subject:               &subject,
		AuthorStyle:           &authorStyle,
		TimePeriod:            &timePeriod,
		Genre:                 &genre,
		Tone:                  &tone,
		CharacterNames:        &characterNames,
		CustomInstructions:    &customInstructions,
		SectionLengthOverride: &sectionLength,
	}

	story, err := storyService.CreateStory(
		context.Background(),
		userID,
		"en",
		req,
	)
	require.NoError(suite.T(), err)
	return story.ID
}

func (suite *StoryAutoGenerationSimpleIntegrationTestSuite) TestToggleAutoGeneration_BasicFunctionality() {
	// Create test user and story
	userID := suite.createTestUser()
	storyID := suite.createTestStory(userID)

	router := suite.setupRouter(userID)

	// Test 1: Toggle to paused (true)
	toggleReq := api.ToggleAutoGenerationRequest{
		Paused: true,
	}
	reqBody, _ := json.Marshal(toggleReq)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/story/%d/toggle-auto-generation", storyID), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var toggleResp api.ToggleAutoGenerationResponse
	err := json.Unmarshal(w.Body.Bytes(), &toggleResp)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), *toggleResp.AutoGenerationPaused)
	assert.Equal(suite.T(), "Auto-generation paused", *toggleResp.Message)

	// Test 2: Toggle to resumed (false)
	toggleReq2 := api.ToggleAutoGenerationRequest{
		Paused: false,
	}
	reqBody2, _ := json.Marshal(toggleReq2)

	req2, _ := http.NewRequest("POST", fmt.Sprintf("/v1/story/%d/toggle-auto-generation", storyID), bytes.NewBuffer(reqBody2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		suite.T().Logf("Response body: %s", w2.Body.String())
	}
	assert.Equal(suite.T(), http.StatusOK, w2.Code)

	var toggleResp2 api.ToggleAutoGenerationResponse
	err = json.Unmarshal(w2.Body.Bytes(), &toggleResp2)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), *toggleResp2.AutoGenerationPaused)
	assert.Equal(suite.T(), "Auto-generation resumed", *toggleResp2.Message)

	// Test 3: Verify the state persisted by checking the story directly
	req, _ = http.NewRequest("GET", fmt.Sprintf("/v1/story/%d", storyID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var story api.StoryWithSections
	err = json.Unmarshal(w.Body.Bytes(), &story)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), *story.AutoGenerationPaused, "State should be false after toggle")
}

func (suite *StoryAutoGenerationSimpleIntegrationTestSuite) TestToggleAutoGeneration_ErrorCases() {
	userID := suite.createTestUser()
	router := suite.setupRouter(userID)

	// Test invalid story ID
	toggleReq := api.ToggleAutoGenerationRequest{
		Paused: true,
	}
	reqBody, _ := json.Marshal(toggleReq)

	req, _ := http.NewRequest("POST", "/v1/story/invalid/toggle-auto-generation", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	// Test non-existent story ID
	req, _ = http.NewRequest("POST", "/v1/story/99999/toggle-auto-generation", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	// Test invalid JSON
	req, _ = http.NewRequest("POST", "/v1/story/1/toggle-auto-generation", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	// Test missing paused field
	req, _ = http.NewRequest("POST", "/v1/story/1/toggle-auto-generation", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *StoryAutoGenerationSimpleIntegrationTestSuite) TestToggleAutoGeneration_UnauthorizedAccess() {
	// Create two test users
	user1ID := suite.createTestUser()
	user2ID := suite.createTestUser()

	// Create story for user1
	storyID := suite.createTestStory(user1ID)

	// Try to toggle with user2 (different user)
	router := suite.setupRouter(user2ID)

	toggleReq := api.ToggleAutoGenerationRequest{
		Paused: true,
	}
	reqBody, _ := json.Marshal(toggleReq)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/story/%d/toggle-auto-generation", storyID), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}
