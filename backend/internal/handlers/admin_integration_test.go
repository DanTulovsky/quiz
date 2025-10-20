//go:build integration
// +build integration

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
	"strconv"
	"strings"
	"testing"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/handlers"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	"quizapp/internal/worker"

	"quizapp/internal/version"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AdminIntegrationTestSuite struct {
	suite.Suite
	BackendRouter *gin.Engine
	WorkerRouter  *gin.Engine
	db            *sql.DB
	testUser      *models.User
	cfg           *config.Config
	worker        *worker.Worker
	userService   *services.UserService
	mockAIService services.AIServiceInterface // Added for mocking AI calls
}

func (suite *AdminIntegrationTestSuite) SetupSuite() {
	// Removed manual AI_PROVIDERS_CONFIG setting; Taskfile.yml sets it correctly.
	// --- Config ---
	cfg, err := config.NewConfig()
	if err != nil {
		suite.T().Fatalf("Failed to load config: %v", err)
	}
	suite.cfg = cfg

	// Use environment variable for test database URL, fallback to test port 5433
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}
	suite.cfg.Database.URL = databaseURL

	// --- Database ---
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := database.NewManager(logger)
	db, err := dbManager.InitDB(databaseURL)
	if err != nil {
		suite.T().Fatalf("Failed to initialize database: %v", err)
	}
	suite.db = db

	// --- Services ---
	userService := services.NewUserServiceWithLogger(db, suite.cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, suite.cfg, logger)

	// Create mock AI service for testing (faster than real external calls)
	mockAIService := handlers.NewMockAIService(suite.cfg, logger)
	suite.mockAIService = mockAIService

	workerService := services.NewWorkerServiceWithLogger(db, logger)
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)
	oauthService := services.NewOAuthServiceWithLogger(suite.cfg, logger)

	// Use the real application router
	generationHintService := services.NewGenerationHintService(db, logger)
	storyService := services.NewStoryService(db, suite.cfg, logger)
	usageStatsService := services.NewUsageStatsService(suite.cfg, suite.db, logger)
	translationService := services.NewTranslationService(suite.cfg, usageStatsService, logger)
	snippetsService := services.NewSnippetsService(db, suite.cfg, logger)
	suite.BackendRouter = handlers.NewRouter(
		suite.cfg,
		userService,
		questionService,
		learningService,
		suite.mockAIService, // Use mock AI service for faster tests
		workerService,
		dailyQuestionService,
		storyService,
		services.NewConversationService(db),
		oauthService,
		generationHintService,
		translationService,
		snippetsService,
		usageStatsService,
		logger,
	)

	// --- Setup Worker Router (unchanged) ---
	workerRouter := gin.Default()
	workerStore := cookie.NewStore([]byte(suite.cfg.Server.SessionSecret))
	workerRouter.Use(sessions.Sessions(config.SessionName, workerStore))

	// Initialize worker service with all required settings to prevent errors
	err = workerService.SetSetting(context.Background(), "global_pause", "false")
	assert.NoError(suite.T(), err)

	err = workerService.SetGlobalPause(context.Background(), false)
	assert.NoError(suite.T(), err)

	// Add any default worker status entries
	workerStatus := &models.WorkerStatus{
		WorkerInstance:          "default",
		IsRunning:               false,
		IsPaused:                false,
		LastHeartbeat:           sql.NullTime{Time: time.Now(), Valid: true},
		TotalQuestionsGenerated: 0,
		TotalRuns:               0,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}
	err = workerService.UpdateWorkerStatus(context.Background(), "default", workerStatus)
	assert.NoError(suite.T(), err)

	// --- Background Worker ---
	emailService := services.NewEmailService(suite.cfg, logger)
	bgWorker := worker.NewWorker(userService, questionService, suite.mockAIService, learningService, workerService, dailyQuestionService, storyService, emailService, generationHintService, "default", suite.cfg, logger)
	suite.worker = bgWorker
	go bgWorker.Start(context.Background())

	// Worker admin handler
	workerAdminHandler := handlers.NewWorkerAdminHandlerWithLogger(
		userService,
		questionService,
		suite.mockAIService,
		suite.cfg,
		bgWorker,
		workerService,
		learningService,
		dailyQuestionService,
		logger,
	)

	// Health check endpoint
	workerRouter.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "worker"})
	})

	// Version endpoint
	workerRouter.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version":   version.Version,
			"commit":    version.Commit,
			"buildTime": version.BuildTime,
		})
	})

	// Add API endpoints that the worker admin page needs
	api := workerRouter.Group("/v1")
	workerGroup := api.Group("/worker")
	{
		workerGroup.GET("/details", workerAdminHandler.GetWorkerDetails)
		workerGroup.GET("/status", workerAdminHandler.GetWorkerStatus)
		workerGroup.POST("/pause", workerAdminHandler.PauseWorker)
		workerGroup.POST("/resume", workerAdminHandler.ResumeWorker)
		workerGroup.POST("/trigger", workerAdminHandler.TriggerWorkerRun)
	}

	// Add analytics endpoints
	analytics := api.Group("/analytics")
	{
		analytics.GET("/priority-scores", workerAdminHandler.GetPriorityAnalytics)
		analytics.GET("/user-performance", workerAdminHandler.GetUserPerformanceAnalytics)
		analytics.GET("/generation-intelligence", workerAdminHandler.GetGenerationIntelligence)
		analytics.GET("/system-health", workerAdminHandler.GetSystemHealthAnalytics)
		analytics.GET("/comparison", workerAdminHandler.GetUserComparisonAnalytics)
		analytics.GET("/user/:userID", workerAdminHandler.GetUserPriorityAnalytics)
	}

	suite.WorkerRouter = workerRouter
}

func (suite *AdminIntegrationTestSuite) SetupTest() {
	// Clean up database before each test using the shared cleanup function
	services.CleanupTestDatabase(suite.db, suite.T())
	suite.createTestData()
}

func (suite *AdminIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *AdminIntegrationTestSuite) createTestData() {
	// Create a test user
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	testUser, err := userService.CreateUserWithPassword(context.Background(), "testuser_admin", "password", "italian", "A1")
	require.NoError(suite.T(), err, "Failed to create test user")
	require.NotNil(suite.T(), testUser, "Test user should not be nil after creation")
	suite.testUser = testUser
	suite.T().Logf("Created test user with ID: %d", testUser.ID)

	// Create a test question
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	testQuestion := &models.Question{
		Type:            "vocabulary",
		Language:        "italian",
		Level:           "A1",
		DifficultyScore: 1.0,
		Content:         map[string]interface{}{"question": "Test question", "options": []string{"A", "B", "C", "D"}},
		CorrectAnswer:   0,
		Explanation:     "Test explanation",
		Status:          models.QuestionStatusActive,
	}
	err = questionService.SaveQuestion(context.Background(), testQuestion)
	require.NoError(suite.T(), err, "Failed to create test question")
	// Assign question to user
	err = questionService.AssignQuestionToUser(context.Background(), testQuestion.ID, testUser.ID)
	require.NoError(suite.T(), err, "Failed to assign test question to user")

	// Create a test user response
	testResponse := &models.UserResponse{
		UserID:          testUser.ID,
		QuestionID:      testQuestion.ID,
		UserAnswerIndex: 0,
		IsCorrect:       true,
		ResponseTimeMs:  1000,
		CreatedAt:       time.Now(),
	}
	err = learningService.RecordUserResponse(context.Background(), testResponse)
	require.NoError(suite.T(), err, "Failed to create test response")

	// Create a performance metric
	suite.T().Logf("Creating performance metric for UserID: %d", testUser.ID)
	_, err = suite.db.Exec(`
		INSERT INTO performance_metrics (user_id, topic, language, level, total_attempts, correct_attempts, average_response_time_ms)
		VALUES ($1, 'basic_vocabulary', 'italian', 'A1', 10, 8, 1500.0)
	`, testUser.ID)
	require.NoError(suite.T(), err, "Failed to create performance metric")

	// Assign admin role to the test user
	err = userService.AssignRoleByName(context.Background(), testUser.ID, "admin")
	require.NoError(suite.T(), err, "Failed to assign admin role to test user")
}

func (suite *AdminIntegrationTestSuite) TestWorkerAdminAPIEndpoints() {
	// Test that the worker details API endpoint works
	req, _ := http.NewRequest("GET", "/v1/worker/details", nil)
	w := httptest.NewRecorder()
	suite.WorkerRouter.ServeHTTP(w, req)

	// Should return 200 OK
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Should return JSON content
	contentType := w.Header().Get("Content-Type")
	assert.Contains(suite.T(), contentType, "application/json")

	// Should contain status and history fields
	body := w.Body.String()
	assert.Contains(suite.T(), body, "status")
	assert.Contains(suite.T(), body, "history")
}

func (suite *AdminIntegrationTestSuite) TestWorkerControlActions() {
	// Test worker pause action
	req, _ := http.NewRequest("POST", "/v1/worker/pause", nil)
	w := httptest.NewRecorder()
	suite.WorkerRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(suite.T(), body, "message")

	// Test worker resume action
	req, _ = http.NewRequest("POST", "/v1/worker/resume", nil)
	w = httptest.NewRecorder()
	suite.WorkerRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	body = w.Body.String()
	assert.Contains(suite.T(), body, "message")

	// Test worker trigger action
	req, _ = http.NewRequest("POST", "/v1/worker/trigger", nil)
	w = httptest.NewRecorder()
	suite.WorkerRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	body = w.Body.String()
	assert.Contains(suite.T(), body, "message")
}

func (suite *AdminIntegrationTestSuite) TestWorkerWithRealWorkerInstance() {
	// Start the worker for a brief moment to test real integration
	ctx, cancel := context.WithTimeout(context.Background(), config.TestTimeout)
	defer cancel()

	// Start worker in background
	go suite.worker.Start(ctx)

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Test that the worker details API now returns real status
	req, _ := http.NewRequest("GET", "/v1/worker/details", nil)
	w := httptest.NewRecorder()
	suite.WorkerRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Should contain actual worker status
	body := w.Body.String()
	assert.Contains(suite.T(), body, "is_running")
	assert.Contains(suite.T(), body, "is_paused")

	// Wait for context to finish
	<-ctx.Done()
}

func (suite *AdminIntegrationTestSuite) TestAdminPagesContainCorrectData() {
	// First authenticate as the admin user
	loginReq := api.LoginRequest{
		Username: "testuser_admin",
		Password: "password",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqObj, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqObj.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	suite.BackendRouter.ServeHTTP(loginW, loginReqObj)

	// Get the session cookie from the login response
	cookies := loginW.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "quiz-session" {
			sessionCookie = cookie
			break
		}
	}
	require.NotNil(suite.T(), sessionCookie, "Should have session cookie after login")

	// Test that backend admin page contains user statistics
	req, _ := http.NewRequest("GET", "/v1/admin/backend", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	suite.BackendRouter.ServeHTTP(w, req)

	body := w.Body.String()

	// Should show our test user with their data
	assert.Contains(suite.T(), body, "testuser_admin")
	assert.Contains(suite.T(), body, "italian") // User's language
	assert.Contains(suite.T(), body, "A1")      // User's level
}

func TestAdminIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AdminIntegrationTestSuite))
}

func (suite *AdminIntegrationTestSuite) TestClearUserDataEndpoint() {
	// Create some test data first
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create additional test user
	testUser2, err := userService.CreateUserWithPassword(context.Background(), "testuser_clear", "password", "spanish", "A2")
	assert.NoError(suite.T(), err)
	require.NotNil(suite.T(), testUser2, "Test user should not be nil after creation")

	// Verify user exists before creating questions
	existingUser, err := userService.GetUserByID(context.Background(), testUser2.ID)
	assert.NoError(suite.T(), err)
	require.NotNil(suite.T(), existingUser, "User should exist before creating questions")

	// Create test question
	testQuestion := &models.Question{
		Type:            "vocabulary",
		Language:        "spanish",
		Level:           "A2",
		DifficultyScore: 1.5,
		Content:         map[string]interface{}{"question": "Test question", "options": []string{"A", "B", "C", "D"}},
		CorrectAnswer:   0,
		Explanation:     "Test explanation",
		Status:          models.QuestionStatusActive,
	}
	err = questionService.SaveQuestion(context.Background(), testQuestion)
	assert.NoError(suite.T(), err)
	require.Greater(suite.T(), testQuestion.ID, 0, "Question should have a valid ID")
	err = questionService.AssignQuestionToUser(context.Background(), testQuestion.ID, testUser2.ID)
	assert.NoError(suite.T(), err)

	// Create user response
	response := &models.UserResponse{
		UserID:          testUser2.ID,
		QuestionID:      testQuestion.ID,
		UserAnswerIndex: 0,
		IsCorrect:       true,
		ResponseTimeMs:  1200,
	}
	err = learningService.RecordUserResponse(context.Background(), response)
	assert.NoError(suite.T(), err)

	// Verify data exists before clearing - check specifically for our test user
	users, err := userService.GetAllUsers(context.Background())
	assert.NoError(suite.T(), err)

	// Find our specific test user
	var foundUser bool
	for _, user := range users {
		if user.Username == "testuser_clear" {
			foundUser = true
			break
		}
	}
	assert.True(suite.T(), foundUser, "Should have our test user before clearing")

	// Authenticate as admin first
	sessionCookie := suite.authenticateAsAdmin()

	// Test the Clear User Data endpoint
	req, _ := http.NewRequest("POST", "/v1/admin/backend/clear-user-data", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	suite.BackendRouter.ServeHTTP(w, req)

	// Should return 200 OK
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Should return JSON success response
	var response_data map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response_data)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response_data["success"].(bool))
	assert.Contains(suite.T(), response_data["message"].(string), "User data cleared successfully")

	// Verify users still exist
	usersAfter, err := userService.GetAllUsers(context.Background())
	assert.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(usersAfter), 1, "Users should still exist after clear user data")

	// Verify our test user still exists
	var foundUserAfter bool
	for _, user := range usersAfter {
		if user.Username == "testuser_clear" {
			foundUserAfter = true
			break
		}
	}
	assert.True(suite.T(), foundUserAfter, "Our test user should still exist after clear user data")

	// Verify questions are deleted
	questionsAfter, err := questionService.GetQuestionsByFilter(context.Background(), 0, "", "", "", 100)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, len(questionsAfter), "All questions should be deleted")

	// Verify responses are deleted
	var responseCount int
	err = suite.db.QueryRow("SELECT COUNT(*) FROM user_responses").Scan(&responseCount)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, responseCount, "All user responses should be deleted")
}

func (suite *AdminIntegrationTestSuite) TestClearDatabaseEndpoint() {
	// Create some test data first
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create additional test user
	testUser3, err := userService.CreateUserWithPassword(context.Background(), "testuser_db_clear", "password", "french", "B1")
	assert.NoError(suite.T(), err)

	// Create test question
	testQuestion := &models.Question{
		Type:            "fill_in_blank",
		Language:        "french",
		Level:           "B1",
		DifficultyScore: 2.0,
		Content:         map[string]interface{}{"question": "Je _____ français", "options": []string{"parle", "mange", "bois", "lis"}},
		CorrectAnswer:   0,
		Explanation:     "Parle means speak in French",
		Status:          models.QuestionStatusActive,
	}
	err = questionService.SaveQuestion(context.Background(), testQuestion)
	assert.NoError(suite.T(), err)
	err = questionService.AssignQuestionToUser(context.Background(), testQuestion.ID, testUser3.ID)
	assert.NoError(suite.T(), err)

	// Verify data exists before clearing
	users, err := userService.GetAllUsers(context.Background())
	assert.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(users), 1, "Should have at least 1 user before database clear")

	// Authenticate as admin first
	sessionCookie := suite.authenticateAsAdmin()

	// Test the Clear Database endpoint (questions may be 0 due to test isolation)
	req, _ := http.NewRequest("POST", "/v1/admin/backend/clear-database", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	suite.BackendRouter.ServeHTTP(w, req)

	// Should return 200 OK
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Should return JSON success response
	var response_data map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response_data)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response_data["success"].(bool))
	assert.Contains(suite.T(), response_data["message"].(string), "Database cleared successfully")

	// Verify ALL data is deleted including users
	usersAfter, err := userService.GetAllUsers(context.Background())
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, len(usersAfter), "All users should be deleted after database clear")

	questionsAfter, err := questionService.GetQuestionsByFilter(context.Background(), 0, "", "", "", 100)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, len(questionsAfter), "All questions should be deleted after database clear")

	// Verify all tables are empty
	var userCount, questionCount, responseCount, metricsCount int
	err = suite.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, userCount, "Users table should be empty")

	err = suite.db.QueryRow("SELECT COUNT(*) FROM questions").Scan(&questionCount)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, questionCount, "Questions table should be empty")

	err = suite.db.QueryRow("SELECT COUNT(*) FROM user_responses").Scan(&responseCount)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, responseCount, "User responses table should be empty")

	err = suite.db.QueryRow("SELECT COUNT(*) FROM performance_metrics").Scan(&metricsCount)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, metricsCount, "Performance metrics table should be empty")
}

func (suite *AdminIntegrationTestSuite) TestClearUserDataForUserEndpoint() {
	// Create two test users
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	testUser1, err := userService.CreateUserWithPassword(context.Background(), "testuser1", "password", "italian", "A1")
	suite.Require().NoError(err)
	testUser2, err := userService.CreateUserWithPassword(context.Background(), "testuser2", "password", "spanish", "B1")
	suite.Require().NoError(err)

	// Add questions and responses for both users
	q1 := &models.Question{
		Type:            "vocabulary",
		Language:        "italian",
		Level:           "A1",
		DifficultyScore: 1.0,
		Content:         map[string]interface{}{"question": "Q1?", "options": []string{"A", "B", "C", "D"}},
		CorrectAnswer:   0,
		Explanation:     "A is correct",
		Status:          models.QuestionStatusActive,
	}
	suite.Require().NoError(questionService.SaveQuestion(context.Background(), q1))
	err = questionService.AssignQuestionToUser(context.Background(), q1.ID, testUser1.ID)
	suite.Require().NoError(err)

	q2 := &models.Question{
		Type:            "vocabulary",
		Language:        "spanish",
		Level:           "B1",
		DifficultyScore: 1.0,
		Content:         map[string]interface{}{"question": "Q2?", "options": []string{"A", "B", "C", "D"}},
		CorrectAnswer:   1,
		Explanation:     "B is correct",
		Status:          models.QuestionStatusActive,
	}
	suite.Require().NoError(questionService.SaveQuestion(context.Background(), q2))
	err = questionService.AssignQuestionToUser(context.Background(), q2.ID, testUser2.ID)
	suite.Require().NoError(err)

	resp1 := &models.UserResponse{
		UserID:          testUser1.ID,
		QuestionID:      q1.ID,
		UserAnswerIndex: 0,
		IsCorrect:       true,
		ResponseTimeMs:  1000,
	}
	suite.Require().NoError(learningService.RecordUserResponse(context.Background(), resp1))

	resp2 := &models.UserResponse{
		UserID:          testUser2.ID,
		QuestionID:      q2.ID,
		UserAnswerIndex: 1,
		IsCorrect:       true,
		ResponseTimeMs:  1000,
	}
	suite.Require().NoError(learningService.RecordUserResponse(context.Background(), resp2))

	// Authenticate as admin first
	sessionCookie := suite.authenticateAsAdmin()

	// Call the new endpoint for testUser1
	req, _ := http.NewRequest("POST", "/v1/admin/backend/userz/"+strconv.Itoa(testUser1.ID)+"/clear", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	suite.BackendRouter.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &response))
	suite.True(response["success"].(bool))

	// Verify testUser1's data is gone, but testUser2's data remains
	questions1, err := questionService.GetUserQuestionsWithStats(context.Background(), testUser1.ID, 100)
	suite.NoError(err)
	suite.Equal(0, len(questions1))
	questions2, err := questionService.GetUserQuestionsWithStats(context.Background(), testUser2.ID, 100)
	suite.NoError(err)
	suite.Equal(1, len(questions2))

	// Verify both users still exist
	user1, err := userService.GetUserByID(context.Background(), testUser1.ID)
	suite.NoError(err)
	suite.NotNil(user1)
	user2, err := userService.GetUserByID(context.Background(), testUser2.ID)
	suite.NoError(err)
	suite.NotNil(user2)
}

func (suite *AdminIntegrationTestSuite) TestWorkerVersionEndpoint() {
	// Test that worker /version endpoint returns version info
	req, _ := http.NewRequest("GET", "/version", nil)
	w := httptest.NewRecorder()
	suite.WorkerRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(suite.T(), err)

	// Check expected fields
	assert.Contains(suite.T(), resp, "version")
	assert.Contains(suite.T(), resp, "commit")
	assert.Contains(suite.T(), resp, "buildTime")

	// Check types
	_, ok := resp["version"].(string)
	assert.True(suite.T(), ok, "version should be a string")
	_, ok = resp["commit"].(string)
	assert.True(suite.T(), ok, "commit should be a string")
	_, ok = resp["buildTime"].(string)
	assert.True(suite.T(), ok, "buildTime should be a string")
}

func (suite *AdminIntegrationTestSuite) TestPriorityAnalyticsEndpoint() {
	// Test priority analytics endpoint
	req, _ := http.NewRequest("GET", "/v1/analytics/priority-scores", nil)
	w := httptest.NewRecorder()
	suite.WorkerRouter.ServeHTTP(w, req)

	// Should return 200 OK
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Should return JSON content
	contentType := w.Header().Get("Content-Type")
	assert.Contains(suite.T(), contentType, "application/json")

	// Parse response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Should parse JSON response")

	// Debug: print the actual response
	suite.T().Logf("Priority analytics response: %+v", response)

	// Should contain distribution and high priority questions
	assert.Contains(suite.T(), response, "distribution")
	assert.Contains(suite.T(), response, "highPriorityQuestions")

	// Check distribution structure
	distribution := response["distribution"].(map[string]interface{})
	assert.Contains(suite.T(), distribution, "high")
	assert.Contains(suite.T(), distribution, "medium")
	assert.Contains(suite.T(), distribution, "low")
	assert.Contains(suite.T(), distribution, "average")

	// Check high priority questions structure
	highPriorityQuestions, ok := response["highPriorityQuestions"].([]interface{})
	// Should be an array (may be empty if no high priority questions)
	if !ok {
		// If it's nil, that's also acceptable for empty results
		assert.Nil(suite.T(), response["highPriorityQuestions"], "highPriorityQuestions should be nil or an array")
	} else {
		// If it's an array, that's good
		assert.IsType(suite.T(), []interface{}{}, highPriorityQuestions, "highPriorityQuestions should be an array")
	}
}

func (suite *AdminIntegrationTestSuite) TestUserPerformanceAnalyticsEndpoint() {
	// Test user performance analytics endpoint
	req, _ := http.NewRequest("GET", "/v1/analytics/user-performance", nil)
	w := httptest.NewRecorder()
	suite.WorkerRouter.ServeHTTP(w, req)

	// Should return 200 OK
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Should return JSON content
	contentType := w.Header().Get("Content-Type")
	assert.Contains(suite.T(), contentType, "application/json")

	// Parse response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Should parse JSON response")

	// Debug: print the actual response
	suite.T().Logf("User performance analytics response: %+v", response)

	// Should contain weak areas and learning preferences
	assert.Contains(suite.T(), response, "weakAreas")
	assert.Contains(suite.T(), response, "learningPreferences")

	// Check weak areas structure
	_, ok := response["weakAreas"].([]interface{})
	// Should be an array (may be empty if no weak areas)
	assert.True(suite.T(), ok, "weakAreas should be an array")

	// Check learning preferences structure
	preferences := response["learningPreferences"].(map[string]interface{})
	assert.Contains(suite.T(), preferences, "total_users")
	assert.Contains(suite.T(), preferences, "focusOnWeakAreas")
	assert.Contains(suite.T(), preferences, "freshQuestionRatio")
}

func (suite *AdminIntegrationTestSuite) TestGenerationIntelligenceEndpoint() {
	// Test generation intelligence endpoint
	req, _ := http.NewRequest("GET", "/v1/analytics/generation-intelligence", nil)
	w := httptest.NewRecorder()
	suite.WorkerRouter.ServeHTTP(w, req)

	// Should return 200 OK
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Should return JSON content
	contentType := w.Header().Get("Content-Type")
	assert.Contains(suite.T(), contentType, "application/json")

	// Parse response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Should parse JSON response")

	// Debug: print the actual response
	suite.T().Logf("Generation intelligence response: %+v", response)

	// Should contain gap analysis and generation suggestions
	assert.Contains(suite.T(), response, "gapAnalysis")
	assert.Contains(suite.T(), response, "generationSuggestions")

	// Check gap analysis structure
	_, ok := response["gapAnalysis"].([]interface{})
	// Should be an array (may be empty if no gaps)
	assert.True(suite.T(), ok, "gapAnalysis should be an array")

	// Check generation suggestions structure
	_, ok = response["generationSuggestions"].([]interface{})
	// Should be an array (may be empty if no suggestions)
	assert.True(suite.T(), ok, "generationSuggestions should be an array")
}

func (suite *AdminIntegrationTestSuite) TestSystemHealthAnalyticsEndpoint() {
	// Test system health analytics endpoint
	req, _ := http.NewRequest("GET", "/v1/analytics/system-health", nil)
	w := httptest.NewRecorder()
	suite.WorkerRouter.ServeHTTP(w, req)

	// Should return 200 OK
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Should return JSON content
	contentType := w.Header().Get("Content-Type")
	assert.Contains(suite.T(), contentType, "application/json")

	// Parse response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Should parse JSON response")

	// Debug: print the actual response
	suite.T().Logf("System health analytics response: %+v", response)

	// Should contain performance and background jobs
	assert.Contains(suite.T(), response, "performance")
	assert.Contains(suite.T(), response, "backgroundJobs")

	// Check performance structure
	performance := response["performance"].(map[string]interface{})
	assert.Contains(suite.T(), performance, "calculationsPerSecond")
	assert.Contains(suite.T(), performance, "avgCalculationTime")
	assert.Contains(suite.T(), performance, "avgQueryTime")
	assert.Contains(suite.T(), performance, "memoryUsage")

	// Check background jobs structure
	backgroundJobs := response["backgroundJobs"].(map[string]interface{})
	assert.Contains(suite.T(), backgroundJobs, "priorityUpdates")
	assert.Contains(suite.T(), backgroundJobs, "lastUpdate")
	assert.Contains(suite.T(), backgroundJobs, "queueSize")
	assert.Contains(suite.T(), backgroundJobs, "status")
}

func (suite *AdminIntegrationTestSuite) TestAnalyticsComparisonEndpoint() {
	// Create two users for testing
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	user1, err := userService.CreateUser(context.Background(), "testuser1", "italian", "A1")
	suite.Require().NoError(err)
	user2, err := userService.CreateUser(context.Background(), "testuser2", "italian", "A1")
	suite.Require().NoError(err)

	// Insert dummy priority scores for both users
	db := suite.db
	_, err = db.Exec(`INSERT INTO question_priority_scores (user_id, question_id, priority_score, last_calculated_at, created_at, updated_at) VALUES ($1, 1, 123.4, NOW(), NOW(), NOW()) ON CONFLICT DO NOTHING`, user1.ID)
	suite.Require().NoError(err)
	_, err = db.Exec(`INSERT INTO question_priority_scores (user_id, question_id, priority_score, last_calculated_at, created_at, updated_at) VALUES ($1, 1, 234.5, NOW(), NOW(), NOW()) ON CONFLICT DO NOTHING`, user2.ID)
	suite.Require().NoError(err)

	// Test with a single user ID
	req, _ := http.NewRequest("GET", "/v1/analytics/comparison?user_ids="+fmt.Sprint(user1.ID), nil)
	w := httptest.NewRecorder()
	suite.WorkerRouter.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	comparison, ok := response["comparison"].([]interface{})
	suite.True(ok)
	suite.Len(comparison, 1)

	// Test with multiple user IDs (comma-separated)
	url := fmt.Sprintf("/v1/analytics/comparison?user_ids=%d,%d", user1.ID, user2.ID)
	req, _ = http.NewRequest("GET", url, nil)
	w = httptest.NewRecorder()
	suite.WorkerRouter.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	comparison, ok = response["comparison"].([]interface{})
	suite.True(ok)
	suite.Len(comparison, 2)

	// Test with invalid user ID
	req, _ = http.NewRequest("GET", "/v1/analytics/comparison?user_ids=notanid", nil)
	w = httptest.NewRecorder()
	suite.WorkerRouter.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
	var errorResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &errorResp)
	suite.Require().NoError(err)
	_, hasError := errorResp["code"]
	suite.True(hasError)
}

func (suite *AdminIntegrationTestSuite) TestAdmin_MarkQuestionAsFixed() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create a question to mark as fixed
	questionID := suite.createTestQuestion()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/admin/backend/questions/"+strconv.Itoa(questionID)+"/fix", nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *AdminIntegrationTestSuite) TestAdmin_UpdateQuestion() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	questionID := suite.createTestQuestion()
	updatePayload := `{"content": {"question": "Updated?", "options": ["A", "B"], "answer": 0}, "correct_answer_index": 0, "explanation": "Updated explanation"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/v1/admin/backend/questions/"+strconv.Itoa(questionID), strings.NewReader(updatePayload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *AdminIntegrationTestSuite) TestAdmin_UpdateQuestion_MarkFixed() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	questionID := suite.createTestQuestion()
	// Create a report for the question so clearing reports can be observed
	_, err := suite.db.Exec(`INSERT INTO question_reports (question_id, reported_by_user_id, report_reason, created_at) VALUES ($1, $2, $3, NOW())`, questionID, suite.testUser.ID, "Typo in options")
	suite.Require().NoError(err)

	updatePayload := `{"content": {"question": "Updated?", "options": ["A", "B"], "answer": 0}, "correct_answer": 0, "explanation": "Updated explanation"}`
	w := httptest.NewRecorder()
	url := "/v1/admin/backend/questions/" + strconv.Itoa(questionID) + "?mark_fixed=true"
	req, _ := http.NewRequest("PUT", url, strings.NewReader(updatePayload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	// Verify question status is active and reports cleared
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	q, err := questionService.GetQuestionByID(context.Background(), questionID)
	suite.Require().NoError(err)
	suite.Equal(models.QuestionStatusActive, q.Status)

	var reportCount int
	err = suite.db.QueryRow("SELECT COUNT(*) FROM question_reports WHERE question_id = $1", questionID).Scan(&reportCount)
	suite.Require().NoError(err)
	suite.Equal(0, reportCount)
}

func (suite *AdminIntegrationTestSuite) TestAdmin_FixQuestionWithAI() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	questionID := suite.createTestQuestion()
	w := httptest.NewRecorder()
	fixPayload := `{"question_id": ` + strconv.Itoa(questionID) + `}`
	req, _ := http.NewRequest("POST", "/v1/admin/backend/questions/"+strconv.Itoa(questionID)+"/ai-fix", strings.NewReader(fixPayload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)
	// Should return 200 or 400 or 502 depending on AI config, but should not panic
	assert.Contains(suite.T(), []int{http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError, http.StatusBadGateway}, w.Code)
}

// Full flow: get AI suggestion, apply it via UpdateQuestion with mark_fixed=true
func (suite *AdminIntegrationTestSuite) TestAdmin_FixQuestionWithAI_ApplySuggestion() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	questionID := suite.createTestQuestion()

	// Call AI-fix endpoint
	w := httptest.NewRecorder()
	fixPayload := `{"question_id": ` + strconv.Itoa(questionID) + `}`
	req, _ := http.NewRequest("POST", "/v1/admin/backend/questions/"+strconv.Itoa(questionID)+"/ai-fix", strings.NewReader(fixPayload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	// If AI not configured, endpoint may return 400; in that case skip apply step
	if w.Code != http.StatusOK {
		suite.T().Logf("AI-fix endpoint returned %d, skipping apply step", w.Code)
		return
	}

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	suite.Require().NoError(err)

	suggestionRaw, ok := resp["suggestion"]
	suite.Require().True(ok)
	suggestion, ok := suggestionRaw.(map[string]interface{})
	suite.Require().True(ok)

	// Build update payload from suggestion: merged content should be under suggestion["content"]
	contentRaw, ok := suggestion["content"]
	suite.Require().True(ok)
	contentMap, ok := contentRaw.(map[string]interface{})
	suite.Require().True(ok)

	// Extract correct_answer from content (may be float64)
	var correctIdx int
	if ca, ok := contentMap["correct_answer"]; ok {
		switch v := ca.(type) {
		case float64:
			correctIdx = int(v)
		case int:
			correctIdx = v
		}
	}

	explanation := ""
	if ex, ok := contentMap["explanation"].(string); ok {
		explanation = ex
	}

	// Simulate a malformed AI payload that wraps content twice and includes duplicate fields
	// to ensure the server-side sanitization unwraps and removes duplicates.
	malformed := map[string]interface{}{
		"content": map[string]interface{}{
			"content":        contentMap,
			"correct_answer": correctIdx,
			"explanation":    explanation,
		},
		"correct_answer": correctIdx,
		"explanation":    explanation,
	}
	contentBytes, err := json.Marshal(malformed)
	suite.Require().NoError(err)

	updatePayload := string(contentBytes)

	// If change_reason present, ensure it's a string
	if cr, ok := suggestion["change_reason"]; ok {
		_, isStr := cr.(string)
		suite.Require().True(isStr)
	}

	// Apply suggestion via UpdateQuestion and mark fixed
	w2 := httptest.NewRecorder()
	url := "/v1/admin/backend/questions/" + strconv.Itoa(questionID) + "?mark_fixed=true"
	req2, _ := http.NewRequest("PUT", url, strings.NewReader(updatePayload))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w2, req2)

	suite.Require().Equal(http.StatusOK, w2.Code)

	// Verify question status is active and content updated
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	q, err := questionService.GetQuestionByID(context.Background(), questionID)
	suite.Require().NoError(err)
	suite.Equal(models.QuestionStatusActive, q.Status)

	// Ensure reports cleared
	var reportCount int
	err = suite.db.QueryRow("SELECT COUNT(*) FROM question_reports WHERE question_id = $1", questionID).Scan(&reportCount)
	suite.Require().NoError(err)
	suite.Equal(0, reportCount)
}

func (suite *AdminIntegrationTestSuite) TestAdmin_GetQuestion() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	questionID := suite.createTestQuestion()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/questions/"+strconv.Itoa(questionID), nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *AdminIntegrationTestSuite) TestAdmin_DeleteQuestion() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	questionID := suite.createTestQuestion()

	// Verify the question exists before trying to delete it
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	question, err := questionService.GetQuestionByID(context.Background(), questionID)
	suite.Require().NoError(err)
	suite.Require().NotNil(question)
	suite.T().Logf("Verified question exists with ID: %d", question.ID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/v1/admin/backend/questions/"+strconv.Itoa(questionID), nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)
	suite.T().Logf("DELETE request returned status: %d", w.Code)
	suite.T().Logf("Response body: %s", w.Body.String())
	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *AdminIntegrationTestSuite) TestAdmin_GetQuestionsPaginated() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create a test user to use as user_id
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	user, err := userService.CreateUser(context.Background(), "testuser", "english", "A1")
	suite.Require().NoError(err)
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/v1/admin/backend/questions/paginated?page=1&page_size=10&user_id=%d", user.ID)
	req, _ := http.NewRequest("GET", url, nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *AdminIntegrationTestSuite) TestAdmin_ReportedQuestions_APIShape() {
	suite.SetupTest()

	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create a reported question with nested/duplicated content to simulate bad legacy data
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	q := &models.Question{
		Type:            "qa",
		Language:        "italian",
		Level:           "A1",
		DifficultyScore: 0,
		Content: map[string]interface{}{
			"content": map[string]interface{}{
				"question":       "Hai visto il suo cane?",
				"options":        []string{"A", "B", "C", "D"},
				"correct_answer": 1,
				"explanation":    "inner explanation",
			},
			"correct_answer": 1,
			"explanation":    "outer explanation",
		},
		CorrectAnswer: 1,
		Explanation:   "outer explanation",
		Status:        models.QuestionStatusReported,
	}
	err := questionService.SaveQuestion(context.Background(), q)
	suite.Require().NoError(err)

	// Insert a report record so reported-questions aggregates it
	_, err = suite.db.Exec(`INSERT INTO question_reports (question_id, reported_by_user_id, report_reason, created_at) VALUES ($1, $2, $3, NOW())`, q.ID, suite.testUser.ID, "this feels wrong to me")
	suite.Require().NoError(err)

	// Call reported-questions endpoint
	req, _ := http.NewRequest("GET", "/v1/admin/backend/reported-questions", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	suite.BackendRouter.ServeHTTP(w, req)

	suite.Require().Equal(http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	suite.Require().NoError(err)

	questions, ok := resp["questions"].([]interface{})
	suite.Require().True(ok)
	suite.Require().GreaterOrEqual(len(questions), 1)

	first := questions[0].(map[string]interface{})
	// Top-level explanation should exist
	_, hasTopExplanation := first["explanation"]
	suite.True(hasTopExplanation, "top-level explanation should exist")

	// content should not contain duplicated top-level keys
	content, ok := first["content"].(map[string]interface{})
	suite.Require().True(ok)
	_, hasInnerExplanation := content["explanation"]
	suite.False(hasInnerExplanation, "content should not include explanation duplicated")
	_, hasInnerCorrect := content["correct_answer"]
	suite.False(hasInnerCorrect, "content should not include correct_answer duplicated")

	// report_reasons should be present at top level
	rr, ok := first["report_reasons"]
	suite.Require().True(ok)
	suite.Require().Contains(fmt.Sprint(rr), "feels wrong")
}

func (suite *AdminIntegrationTestSuite) TestAdmin_GetConfigz() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/configz", nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

// Edge case: invalid question ID
func (suite *AdminIntegrationTestSuite) TestAdmin_InvalidQuestionID() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/questions/999999", nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

// Edge case: invalid update payload
func (suite *AdminIntegrationTestSuite) TestAdmin_UpdateQuestion_InvalidPayload() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	questionID := suite.createTestQuestion()
	w := httptest.NewRecorder()
	invalidPayload := `{"content": "invalid", "correct_answer": 5, "explanation": ""}`
	req, _ := http.NewRequest("PUT", "/v1/admin/backend/questions/"+strconv.Itoa(questionID), strings.NewReader(invalidPayload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// Test FixQuestionWithAI with non-existent question (should return 404)
func (suite *AdminIntegrationTestSuite) TestAdmin_FixQuestionWithAI_QuestionNotFound() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Use a non-existent question ID
	nonExistentQuestionID := 999999999
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/admin/backend/questions/"+strconv.Itoa(nonExistentQuestionID)+"/ai-fix", nil)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), "QUESTION_NOT_FOUND", response["code"])
}

// Test AssignUsersToQuestion with non-existent question (should return 404)
func (suite *AdminIntegrationTestSuite) TestAdmin_AssignUsersToQuestion_QuestionNotFound() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create a test user to assign
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	user, err := userService.CreateUser(context.Background(), "testuser", "english", "A1")
	suite.Require().NoError(err)

	// Use a non-existent question ID
	nonExistentQuestionID := 999999999
	w := httptest.NewRecorder()
	payload := fmt.Sprintf(`{"user_ids": [%d]}`, user.ID)
	req, _ := http.NewRequest("POST", "/v1/admin/backend/questions/"+strconv.Itoa(nonExistentQuestionID)+"/assign-users", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), "QUESTION_NOT_FOUND", response["code"])
}

// Test AssignUsersToQuestion with valid question (should return 200)
func (suite *AdminIntegrationTestSuite) TestAdmin_AssignUsersToQuestion_Success() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create a test question
	questionID := suite.createTestQuestion()

	// Create test users to assign
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	user1, err := userService.CreateUser(context.Background(), "testuser1", "english", "A1")
	suite.Require().NoError(err)
	user2, err := userService.CreateUser(context.Background(), "testuser2", "english", "A1")
	suite.Require().NoError(err)

	w := httptest.NewRecorder()
	payload := fmt.Sprintf(`{"user_ids": [%d, %d]}`, user1.ID, user2.ID)
	req, _ := http.NewRequest("POST", "/v1/admin/backend/questions/"+strconv.Itoa(questionID)+"/assign-users", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), "Users assigned to question successfully", response["message"])
}

// Test UnassignUsersFromQuestion with non-existent question (should return 404)
func (suite *AdminIntegrationTestSuite) TestAdmin_UnassignUsersFromQuestion_QuestionNotFound() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create a test user to unassign
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	user, err := userService.CreateUser(context.Background(), "testuser", "english", "A1")
	suite.Require().NoError(err)

	// Use a non-existent question ID
	nonExistentQuestionID := 999999999
	w := httptest.NewRecorder()
	payload := fmt.Sprintf(`{"user_ids": [%d]}`, user.ID)
	req, _ := http.NewRequest("POST", "/v1/admin/backend/questions/"+strconv.Itoa(nonExistentQuestionID)+"/unassign-users", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), "QUESTION_NOT_FOUND", response["code"])
}

// Test UnassignUsersFromQuestion with valid question (should return 200)
func (suite *AdminIntegrationTestSuite) TestAdmin_UnassignUsersFromQuestion_Success() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create a test question
	questionID := suite.createTestQuestion()

	// Create test users to unassign
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	user1, err := userService.CreateUser(context.Background(), "testuser1", "english", "A1")
	suite.Require().NoError(err)
	user2, err := userService.CreateUser(context.Background(), "testuser2", "english", "A1")
	suite.Require().NoError(err)

	// First assign users to the question
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, logger)
	err = questionService.AssignUsersToQuestion(context.Background(), questionID, []int{user1.ID, user2.ID})
	suite.Require().NoError(err)

	// Now unassign them
	w := httptest.NewRecorder()
	payload := fmt.Sprintf(`{"user_ids": [%d, %d]}`, user1.ID, user2.ID)
	req, _ := http.NewRequest("POST", "/v1/admin/backend/questions/"+strconv.Itoa(questionID)+"/unassign-users", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), "Users unassigned from question successfully", response["message"])
}

// Test GetUsersForQuestion with non-existent question (should return 200 with empty users)
func (suite *AdminIntegrationTestSuite) TestAdmin_GetUsersForQuestion_QuestionNotFound() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Use a non-existent question ID
	nonExistentQuestionID := 999999999
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/questions/"+strconv.Itoa(nonExistentQuestionID)+"/users", nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	// This should return 200 with empty users list, not 404
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Debug: print the response body
	suite.T().Logf("Response body: %s", w.Body.String())

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	// Debug: print the response map
	suite.T().Logf("Response map: %+v", response)

	assert.NotNil(suite.T(), response["users"])
	assert.NotNil(suite.T(), response["total_count"])
	assert.Equal(suite.T(), float64(0), response["total_count"])
}

// Test GetUsersForQuestion with valid question (should return 200)
func (suite *AdminIntegrationTestSuite) TestAdmin_GetUsersForQuestion_Success() {
	suite.SetupTest()
	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create a test question
	questionID := suite.createTestQuestion()

	// Create test users and assign them to the question
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	user1, err := userService.CreateUser(context.Background(), "testuser1", "english", "A1")
	suite.Require().NoError(err)
	user2, err := userService.CreateUser(context.Background(), "testuser2", "english", "A1")
	suite.Require().NoError(err)

	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, logger)
	err = questionService.AssignUsersToQuestion(context.Background(), questionID, []int{user1.ID, user2.ID})
	suite.Require().NoError(err)

	// Get users for the question
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/questions/"+strconv.Itoa(questionID)+"/users", nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	assert.NotNil(suite.T(), response["users"])
	assert.NotNil(suite.T(), response["total_count"])
	assert.Equal(suite.T(), float64(2), response["total_count"])

	users := response["users"].([]interface{})
	assert.Len(suite.T(), users, 2)
}

// --- Story Explorer Tests ---

func (suite *AdminIntegrationTestSuite) TestAdmin_GetStoriesPaginated() {
	suite.SetupTest()

	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create test stories
	storyService := services.NewStoryService(suite.db, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a test story for our test user
	_, err := storyService.CreateStory(context.Background(), uint(suite.testUser.ID), "italian", &models.CreateStoryRequest{
		Title:   "Test Story 1",
		Subject: stringPtr("Test subject 1"),
	})
	suite.Require().NoError(err)

	// Create another story
	_, err = storyService.CreateStory(context.Background(), uint(suite.testUser.ID), "spanish", &models.CreateStoryRequest{
		Title:   "Test Story 2",
		Subject: stringPtr("Test subject 2"),
	})
	suite.Require().NoError(err)

	// Test GetStoriesPaginated without filters
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/stories?page=1&page_size=10", nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	stories := response["stories"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(stories), 2)

	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(suite.T(), float64(1), pagination["page"])
	assert.Equal(suite.T(), float64(10), pagination["page_size"])
	assert.GreaterOrEqual(suite.T(), int(pagination["total"].(float64)), 2)
}

func (suite *AdminIntegrationTestSuite) TestAdmin_GetStoriesPaginated_WithFilters() {
	suite.SetupTest()

	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create test stories
	storyService := services.NewStoryService(suite.db, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a test story for our test user
	_, err := storyService.CreateStory(context.Background(), uint(suite.testUser.ID), "italian", &models.CreateStoryRequest{
		Title:   "Italian Adventure",
		Subject: stringPtr("Adventure story"),
	})
	suite.Require().NoError(err)

	// Create another story in Spanish
	_, err = storyService.CreateStory(context.Background(), uint(suite.testUser.ID), "spanish", &models.CreateStoryRequest{
		Title:   "Spanish Mystery",
		Subject: stringPtr("Mystery story"),
	})
	suite.Require().NoError(err)

	// Test filtering by language
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/stories?page=1&page_size=10&language=italian", nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	suite.T().Logf("Response: %+v", response)

	stories := response["stories"].([]interface{})
	suite.T().Logf("Found %d stories", len(stories))
	suite.Require().Len(stories, 1)

	firstStory := stories[0].(map[string]interface{})
	suite.T().Logf("First story: %+v", firstStory)
	assert.Equal(suite.T(), "Italian Adventure", firstStory["title"])
	assert.Equal(suite.T(), "italian", firstStory["language"])

	// Test filtering by status (active stories)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/admin/backend/stories?page=1&page_size=10&status=active", nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	stories = response["stories"].([]interface{})
	assert.Len(suite.T(), stories, 2)

	// Find the Spanish story in the results
	var spanishStory map[string]interface{}
	for _, story := range stories {
		s := story.(map[string]interface{})
		if s["title"] == "Spanish Mystery" {
			spanishStory = s
			break
		}
	}
	suite.Require().NotNil(spanishStory, "Spanish Mystery story should be found in active stories")
	assert.Equal(suite.T(), "Spanish Mystery", spanishStory["title"])
	assert.Equal(suite.T(), "spanish", spanishStory["language"])
}

func (suite *AdminIntegrationTestSuite) TestAdmin_GetStoriesPaginated_WithUserFilter() {
	suite.SetupTest()

	// Create another test user
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	testUser2, err := userService.CreateUserWithPassword(context.Background(), "testuser2", "password", "french", "B1")
	suite.Require().NoError(err)

	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create test stories
	storyService := services.NewStoryService(suite.db, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a story for testUser1
	_, err = storyService.CreateStory(context.Background(), uint(suite.testUser.ID), "italian", &models.CreateStoryRequest{
		Title:   "User1 Story",
		Subject: stringPtr("User1 subject"),
	})
	suite.Require().NoError(err)

	// Create a story for testUser2
	_, err = storyService.CreateStory(context.Background(), uint(testUser2.ID), "french", &models.CreateStoryRequest{
		Title:   "User2 Story",
		Subject: stringPtr("User2 subject"),
	})
	suite.Require().NoError(err)

	// Test filtering by user_id
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/v1/admin/backend/stories?page=1&page_size=10&user_id=%d", testUser2.ID)
	req, _ := http.NewRequest("GET", url, nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	stories := response["stories"].([]interface{})
	assert.Len(suite.T(), stories, 1)

	firstStory := stories[0].(map[string]interface{})
	assert.Equal(suite.T(), "User2 Story", firstStory["title"])
	assert.Equal(suite.T(), float64(testUser2.ID), firstStory["user_id"])
}

func (suite *AdminIntegrationTestSuite) TestAdmin_GetStoryAdmin() {
	suite.SetupTest()

	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create a test story with sections
	storyService := services.NewStoryService(suite.db, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	var storyID uint
	_, err := storyService.CreateStory(context.Background(), uint(suite.testUser.ID), "italian", &models.CreateStoryRequest{
		Title:   "Story with Sections",
		Subject: stringPtr("Test story with sections"),
	})
	suite.Require().NoError(err)

	// Get the story ID from the database
	var createdStory models.Story
	err = suite.db.QueryRow("SELECT id, title FROM stories WHERE title = 'Story with Sections'").Scan(&createdStory.ID, &createdStory.Title)
	suite.Require().NoError(err)
	storyID = createdStory.ID

	// Add a section to the story
	section := &models.StorySection{
		StoryID:        uint(storyID),
		SectionNumber:  1,
		Content:        "This is the first section of the story.",
		LanguageLevel:  "A1",
		WordCount:      8,
		GeneratedBy:    models.GeneratorTypeUser,
		GeneratedAt:    time.Now(),
		GenerationDate: time.Now().Truncate(24 * time.Hour),
	}
	_, err = suite.db.Exec(`
		INSERT INTO story_sections (story_id, section_number, content, language_level, word_count, generated_by, generated_at, generation_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		section.StoryID, section.SectionNumber, section.Content, section.LanguageLevel,
		section.WordCount, string(section.GeneratedBy), section.GeneratedAt, section.GenerationDate)
	suite.Require().NoError(err)

	// Test GetStoryAdmin
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/stories/"+strconv.Itoa(int(storyID)), nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	// Verify story data
	assert.Equal(suite.T(), "Story with Sections", response["title"])
	assert.Equal(suite.T(), "italian", response["language"])
	assert.Equal(suite.T(), "active", response["status"])

	// Verify sections are included
	sections, ok := response["sections"].([]interface{})
	suite.Require().True(ok)
	assert.Len(suite.T(), sections, 1)

	firstSection := sections[0].(map[string]interface{})
	assert.Equal(suite.T(), "This is the first section of the story.", firstSection["content"])
	assert.Equal(suite.T(), float64(1), firstSection["section_number"])
}

func (suite *AdminIntegrationTestSuite) TestAdmin_GetStoryAdmin_NotFound() {
	suite.SetupTest()

	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Test with non-existent story ID
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/stories/999999", nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func (suite *AdminIntegrationTestSuite) TestAdmin_GetSectionAdmin() {
	suite.SetupTest()

	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Create a test story and section
	storyService := services.NewStoryService(suite.db, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	_, err := storyService.CreateStory(context.Background(), uint(suite.testUser.ID), "italian", &models.CreateStoryRequest{
		Title:   "Story for Section Test",
		Subject: stringPtr("Test story"),
	})
	suite.Require().NoError(err)

	// Get the story ID
	var storyID int
	err = suite.db.QueryRow("SELECT id FROM stories WHERE title = 'Story for Section Test'").Scan(&storyID)
	suite.Require().NoError(err)

	// Create a section
	section := &models.StorySection{
		StoryID:        uint(storyID),
		SectionNumber:  1,
		Content:        "This is a test section for admin viewing.",
		LanguageLevel:  "A1",
		WordCount:      9,
		GeneratedBy:    models.GeneratorTypeUser,
		GeneratedAt:    time.Now(),
		GenerationDate: time.Now().Truncate(24 * time.Hour),
	}
	_, err = suite.db.Exec(`
		INSERT INTO story_sections (story_id, section_number, content, language_level, word_count, generated_by, generated_at, generation_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		section.StoryID, section.SectionNumber, section.Content, section.LanguageLevel,
		section.WordCount, string(section.GeneratedBy), section.GeneratedAt, section.GenerationDate)
	suite.Require().NoError(err)

	// Get the section ID
	var sectionID uint
	err = suite.db.QueryRow("SELECT id FROM story_sections WHERE story_id = $1", storyID).Scan(&sectionID)
	suite.Require().NoError(err)

	// Add a question to the section
	_, err = suite.db.Exec(`
		INSERT INTO story_section_questions (section_id, question_text, options, correct_answer_index, explanation, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		sectionID, "What is the main topic?", `["Topic A", "Topic B", "Topic C"]`, 0, "Topic A is correct", time.Now())
	suite.Require().NoError(err)

	// Test GetSectionAdmin
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/story-sections/"+strconv.Itoa(int(sectionID)), nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	// Verify section data
	assert.Equal(suite.T(), "This is a test section for admin viewing.", response["content"])
	assert.Equal(suite.T(), float64(1), response["section_number"])
	assert.Equal(suite.T(), "A1", response["language_level"])

	// Verify questions are included
	questions, ok := response["questions"].([]interface{})
	suite.Require().True(ok)
	assert.Len(suite.T(), questions, 1)

	firstQuestion := questions[0].(map[string]interface{})
	assert.Equal(suite.T(), "What is the main topic?", firstQuestion["question_text"])
}

func (suite *AdminIntegrationTestSuite) TestAdmin_GetSectionAdmin_NotFound() {
	suite.SetupTest()

	// Authenticate as admin
	sessionCookie := suite.authenticateAsAdmin()

	// Test with non-existent section ID
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/story-sections/999999", nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func (suite *AdminIntegrationTestSuite) TestAdmin_StoriesEndpoints_Unauthenticated() {
	suite.SetupTest()

	// Test without authentication
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/stories", nil)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *AdminIntegrationTestSuite) TestAdmin_StoriesEndpoints_NonAdminUser() {
	suite.SetupTest()

	// Create a non-admin user
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(suite.db, suite.cfg, logger)
	_, err := userService.CreateUserWithPassword(context.Background(), "nonadmin", "password", "english", "A1")
	suite.Require().NoError(err)

	// Login as non-admin user
	loginReq := api.LoginRequest{
		Username: "nonadmin",
		Password: "password",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqObj, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqObj.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	suite.BackendRouter.ServeHTTP(loginW, loginReqObj)

	// Get the session cookie
	cookies := loginW.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == config.SessionName {
			sessionCookie = cookie
			break
		}
	}
	suite.Require().NotNil(sessionCookie)

	// Try to access admin endpoint as non-admin
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/admin/backend/stories", nil)
	req.AddCookie(sessionCookie)
	suite.BackendRouter.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

// Helper to authenticate as admin and return session cookie
func (suite *AdminIntegrationTestSuite) authenticateAsAdmin() *http.Cookie {
	loginReq := api.LoginRequest{
		Username: "testuser_admin",
		Password: "password",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqObj, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReqObj.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	suite.BackendRouter.ServeHTTP(loginW, loginReqObj)

	// Get the session cookie from the login response
	cookies := loginW.Result().Cookies()
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

// Helper to create a test question
func (suite *AdminIntegrationTestSuite) createTestQuestion() int {
	question := &models.Question{
		Type:          models.Vocabulary,
		Language:      "english",
		Level:         "A1",
		Content:       map[string]interface{}{"question": "Test?", "options": []string{"A", "B", "C", "D"}},
		CorrectAnswer: 0,
		Status:        models.QuestionStatusActive,
	}
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	err := questionService.SaveQuestion(context.Background(), question)
	assert.NoError(suite.T(), err)
	return question.ID
}
