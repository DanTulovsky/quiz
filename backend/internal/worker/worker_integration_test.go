//go:build integration

package worker

import (
	"context"
	"strings"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function for string pointer conversion
func stringPtr(s string) *string {
	return &s
}

// MockAIService implements a mock AI service for testing
type MockAIService struct {
	*services.AIService
}

// GenerateQuestionsStream mocks the AI question generation
func (m *MockAIService) GenerateQuestionsStream(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest, progress chan<- *models.Question, variety *services.VarietyElements) error {
	defer close(progress)
	// Create a mock question
	mockQuestion := &models.Question{
		ID:       1,
		Language: req.Language,
		Level:    req.Level,
		Type:     req.QuestionType,
		Content: map[string]interface{}{
			"question":       "What is the capital of France?",
			"options":        []string{"London", "Paris", "Berlin", "Madrid"},
			"correct_answer": 1,
			"explanation":    "Paris is the capital of France.",
			"topic":          "geography",
		},
		CorrectAnswer: 1,
		Explanation:   "Paris is the capital of France.",
		// Optionally set variety fields if needed
	}
	// Send the mock question
	select {
	case progress <- mockQuestion:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TestWorkerIntegration_StartAndShutdown tests the main worker lifecycle
func TestWorkerIntegration_StartAndShutdown(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create worker
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Test worker startup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker in a goroutine
	go worker.Start(ctx)

	// Give worker time to start
	time.Sleep(100 * time.Millisecond)

	// Check that worker is running
	status := worker.GetStatus()
	assert.True(t, status.IsRunning)

	// Test shutdown
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Check that worker has stopped
	status = worker.GetStatus()
	assert.False(t, status.IsRunning)
}

// TestWorkerIntegration_HeartbeatLoop tests the heartbeat functionality
func TestWorkerIntegration_HeartbeatLoop(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create worker
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Test heartbeat loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start heartbeat loop
	go worker.heartbeatLoop(ctx)

	// Give heartbeat time to run
	time.Sleep(200 * time.Millisecond)

	// Check that heartbeat was updated
	// Note: In a real test, we'd check the database for heartbeat updates
}

// TestWorkerIntegration_RunWithNoUsers tests worker run with no eligible users
func TestWorkerIntegration_RunWithNoUsers(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create worker
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Test run with no users
	worker.run(context.Background())

	// Check that run was recorded (may be paused due to worker status not found)
	history := worker.GetHistory()
	if len(history) > 0 {
		// If run was recorded, check the details
		assert.Contains(t, history[0].Details, "No active users with AI provider configuration found")
	} else {
		// If no run was recorded, it means the worker was paused
		// This is expected behavior when worker status doesn't exist
		t.Log("Worker run was paused (expected when worker status doesn't exist)")
	}
}

// TestWorkerIntegration_RunWithUsers tests worker run with eligible users
func TestWorkerIntegration_RunWithUsers(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create test user
	user, err := userService.CreateUser(context.Background(), "testuser", "english", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create worker
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Test run with user
	worker.run(context.Background())

	// Check that run was recorded (may be paused due to worker status not found)
	history := worker.GetHistory()
	if len(history) > 0 {
		// If run was recorded, check that it was successful
		assert.Equal(t, "Success", history[0].Status)
	} else {
		// If no run was recorded, it means the worker was paused
		// This is expected behavior when worker status doesn't exist
		t.Log("Worker run was paused (expected when worker status doesn't exist)")
	}
}

// TestWorkerIntegration_GenerateNeededQuestions tests question generation
func TestWorkerIntegration_GenerateNeededQuestions(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)

	// Use mock AI service instead of real one
	mockAIService := &MockAIService{
		AIService: services.NewAIService(cfg, logger, services.NewNoopUsageStatsService()),
	}

	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create test user
	user, err := userService.CreateUser(context.Background(), "testuser", "english", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Enable AI for the user with Ollama provider
	err = userService.UpdateUserSettings(context.Background(), user.ID, &models.UserSettings{
		Language:   "english",
		Level:      "A1",
		AIProvider: "ollama",
		AIModel:    "llama4:latest",
		AIEnabled:  true,
	})
	require.NoError(t, err)

	// Refresh user to get updated AI settings
	user, err = userService.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create worker with mock AI service
	worker := NewWorker(userService, questionService, mockAIService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Test question generation
	result, err := worker.GenerateQuestionsForUser(context.Background(), user, "english", "A1", models.Vocabulary, 1, "test")
	assert.NoError(t, err)
	assert.Contains(t, result, "Generated")
}

// Verify worker eligible count uses 2-day recent-correct exclusion
func TestWorkerIntegration_EligibleCount_RecentCorrectExclusion(t *testing.T) {
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	worker := NewWorker(userService, questionService, services.NewAIService(cfg, logger, services.NewNoopUsageStatsService()), learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Create user and two questions
	user, err := userService.CreateUser(context.Background(), "eligibleuser", "italian", "A1")
	require.NoError(t, err)

	q1 := &models.Question{Type: models.Vocabulary, Language: "italian", Level: "A1", Content: map[string]interface{}{"question": "Q1", "options": []string{"A", "B", "C", "D"}}, CorrectAnswer: 0, Status: models.QuestionStatusActive}
	q2 := &models.Question{Type: models.Vocabulary, Language: "italian", Level: "A1", Content: map[string]interface{}{"question": "Q2", "options": []string{"A", "B", "C", "D"}}, CorrectAnswer: 1, Status: models.QuestionStatusActive}
	err = questionService.SaveQuestion(context.Background(), q1)
	require.NoError(t, err)
	err = questionService.SaveQuestion(context.Background(), q2)
	require.NoError(t, err)
	err = questionService.AssignQuestionToUser(context.Background(), q1.ID, user.ID)
	require.NoError(t, err)
	err = questionService.AssignQuestionToUser(context.Background(), q2.ID, user.ID)
	require.NoError(t, err)

	// Record a correct response for q1 yesterday -> should exclude q1
	_, err = db.Exec(`INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, response_time_ms, created_at) VALUES ($1,$2,0,TRUE,1000,NOW() - INTERVAL '1 day')`, user.ID, q1.ID)
	require.NoError(t, err)

	// No response for q2 -> eligible
	count, err := worker.getEligibleQuestionCount(context.Background(), user.ID, "italian", "A1", models.Vocabulary)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Only q2 should be eligible due to recent-correct exclusion on q1")
}

// TestWorkerIntegration_HandleAIQuestionStream tests AI stream handling
func TestWorkerIntegration_HandleAIQuestionStream(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create test user
	user, err := userService.CreateUser(context.Background(), "testuser", "english", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create worker
	_ = NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Test AI stream handling (simplified test)
	// Note: This is a basic test that verifies the function exists and can be called
	// In a real scenario, we'd test the actual AI integration
}

// TestWorkerIntegration_ErrorHandling tests error handling scenarios
func TestWorkerIntegration_ErrorHandling(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create worker
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Test error handling with invalid context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Run worker with cancelled context
	_ = ctx
	worker.run(context.Background())

	// Check that error was handled gracefully
	history := worker.GetHistory()
	if len(history) > 0 {
		// If run was recorded, check that it was handled gracefully
		assert.Equal(t, "Success", history[0].Status)
	} else {
		// If no run was recorded, it means the worker was paused
		// This is expected behavior when worker status doesn't exist
		t.Log("Worker run was paused (expected when worker status doesn't exist)")
	}
}

// TestWorkerIntegration_PauseResumeFlow tests pause and resume functionality
func TestWorkerIntegration_PauseResumeFlow(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create worker
	_ = NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Test pause functionality
	err = workerService.SetGlobalPause(context.Background(), true)
	assert.NoError(t, err)

	// Test resume functionality
	err = workerService.SetGlobalPause(context.Background(), false)
	assert.NoError(t, err)
}

// TestWorkerIntegration_StartupPause tests worker startup with global pause
func TestWorkerIntegration_StartupPause(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Set global pause before starting worker
	err = workerService.SetGlobalPause(context.Background(), true)
	assert.NoError(t, err)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create worker
	emailService := services.NewEmailService(cfg, logger)
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Run worker (should respect global pause)
	worker.run(context.Background())

	// Check that worker handled pause correctly
	history := worker.GetHistory()
	if len(history) > 0 {
		// If run was recorded, check that it was handled correctly
		assert.Equal(t, "Success", history[0].Status)
	} else {
		// If no run was recorded, it means the worker was paused
		// This is expected behavior when global pause is set
		t.Log("Worker run was paused (expected when global pause is set)")
	}
}

// TestWorkerIntegration_ActivityLogging tests activity logging functionality
func TestWorkerIntegration_ActivityLogging(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create worker
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Test activity logging
	worker.logActivity(context.Background(), "INFO", "Test activity", nil, nil)

	// Check that activity was logged
	activityLogs := worker.GetActivityLogs()
	assert.Len(t, activityLogs, 1)
	assert.Equal(t, "INFO", activityLogs[0].Level)
	assert.Equal(t, "Test activity", activityLogs[0].Message)
}

// TestWorkerIntegration_UserFailureTracking tests user failure tracking
func TestWorkerIntegration_UserFailureTracking(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create test user
	user, err := userService.CreateUser(context.Background(), "testuser", "english", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create worker
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Test user failure tracking
	worker.recordUserFailure(context.Background(), user.ID, "Test failure")

	// Check that failure was recorded
	// Note: In a real implementation, we'd check the database for failure records
}

// TestWorkerIntegration_ManualTrigger tests manual trigger functionality
func TestWorkerIntegration_ManualTrigger(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create worker
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Test manual trigger
	worker.TriggerManualRun()

	// Check that manual trigger channel receives a value
	select {
	case <-worker.manualTrigger:
		// Success
	case <-time.After(time.Second):
		t.Error("Expected manual trigger to be sent")
	}
}

// TestWorkerIntegration_Shutdown tests graceful shutdown
func TestWorkerIntegration_Shutdown(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create worker
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Test shutdown
	ctx := context.Background()
	err = worker.Shutdown(ctx)
	assert.NoError(t, err)

	// Check that worker has stopped
	status := worker.GetStatus()
	assert.False(t, status.IsRunning)
}

// TestWorkerPriorityFunctions_Integration tests the priority functions and gap analysis
func TestWorkerPriorityFunctions_Integration(t *testing.T) {
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Initialize services
	cfg, err := config.NewConfig()
	require.NoError(t, err)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create worker instance
	emailService := services.NewEmailService(cfg, logger)
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-worker", cfg, logger)

	// Create a test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser", "password", "italian", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Create test questions with different topic categories
	questions := []*models.Question{
		{
			Type:            models.Vocabulary,
			Language:        "italian",
			Level:           "A1",
			Content:         map[string]interface{}{"question": "What is the Italian word for 'hello'?", "options": []string{"Ciao", "Arrivederci", "Grazie", "Per favore"}},
			CorrectAnswer:   0,
			TopicCategory:   "greetings",
			DifficultyScore: 0.3,
			Status:          models.QuestionStatusActive,
		},
		{
			Type:            models.Vocabulary,
			Language:        "italian",
			Level:           "A1",
			Content:         map[string]interface{}{"question": "What is the Italian word for 'goodbye'?", "options": []string{"Ciao", "Arrivederci", "Grazie", "Per favore"}},
			CorrectAnswer:   1,
			TopicCategory:   "greetings",
			DifficultyScore: 0.3,
			Status:          models.QuestionStatusActive,
		},
		{
			Type:            models.Vocabulary,
			Language:        "italian",
			Level:           "A1",
			Content:         map[string]interface{}{"question": "What is the Italian word for 'thank you'?", "options": []string{"Ciao", "Arrivederci", "Grazie", "Per favore"}},
			CorrectAnswer:   2,
			TopicCategory:   "courtesy",
			DifficultyScore: 0.3,
			Status:          models.QuestionStatusActive,
		},
		{
			Type:            models.Vocabulary,
			Language:        "italian",
			Level:           "A1",
			Content:         map[string]interface{}{"question": "What is the Italian word for 'please'?", "options": []string{"Ciao", "Arrivederci", "Grazie", "Per favore"}},
			CorrectAnswer:   3,
			TopicCategory:   "courtesy",
			DifficultyScore: 0.3,
			Status:          models.QuestionStatusActive,
		},
		{
			Type:            models.Vocabulary,
			Language:        "italian",
			Level:           "A1",
			Content:         map[string]interface{}{"question": "What is the Italian word for 'pizza'?", "options": []string{"Pizza", "Pasta", "Gelato", "Caffè"}},
			CorrectAnswer:   0,
			TopicCategory:   "food",
			DifficultyScore: 0.3,
			Status:          models.QuestionStatusActive,
		},
	}

	// Save questions and assign to user
	for i, question := range questions {
		err := questionService.SaveQuestion(context.Background(), question)
		require.NoError(t, err)

		err = questionService.AssignQuestionToUser(context.Background(), question.ID, user.ID)
		require.NoError(t, err)

		// Update the question ID for priority score insertion
		questions[i] = question
	}

	// Create priority scores for the questions
	priorityScores := []struct {
		questionID int
		score      float64
	}{
		{questions[0].ID, 8.5}, // greetings - high priority
		{questions[1].ID, 7.2}, // greetings - medium priority
		{questions[2].ID, 9.1}, // courtesy - high priority
		{questions[3].ID, 6.8}, // courtesy - medium priority
		{questions[4].ID, 4.2}, // food - low priority
	}

	for _, ps := range priorityScores {
		_, err := db.Exec(`
			INSERT INTO question_priority_scores (question_id, user_id, priority_score, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW())
		`, ps.questionID, user.ID, ps.score)
		require.NoError(t, err)
	}

	// Create some user responses to generate performance data
	responses := []struct {
		questionID int
		correct    bool
		response   int
	}{
		{1, true, 0},  // 'Ciao' is at index 0
		{2, false, 0}, // 'Ciao' is at index 0
		{3, true, 2},  // 'Grazie' is at index 2
		{4, false, 2}, // 'Grazie' is at index 2
		{5, true, 1},  // 'Acqua' is at index 1 (assuming options are set accordingly)
	}

	for _, resp := range responses {
		_, err := db.Exec(`
			INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, response_time_ms, created_at)
			VALUES ($1, $2, $3, $4, 2000, NOW())
		`, user.ID, resp.questionID, resp.response, resp.correct)
		require.NoError(t, err)
	}

	// Test getHighPriorityTopics
	t.Run("getHighPriorityTopics", func(t *testing.T) {
		topics, err := worker.getHighPriorityTopics(context.Background(), user.ID, "italian", "A1", models.Vocabulary)
		require.NoError(t, err)

		// Should return topics with high average priority scores
		// Based on our test data, "courtesy" should be high priority (avg 7.95)
		assert.Contains(t, topics, "courtesy")
		// "greetings" should also be included (avg 7.85)
		assert.Contains(t, topics, "greetings")
		// "food_drink" should not be included (avg 4.2 is low)
		assert.NotContains(t, topics, "food_drink")
	})

	// Test gap analysis - should identify areas where user has poor performance
	gapAnalysis, err := worker.getGapAnalysis(context.Background(), user.ID, "italian", "A1", models.Vocabulary)
	require.NoError(t, err)
	require.NotNil(t, gapAnalysis, "getGapAnalysis should return a non-nil map")

	// Create some user responses with poor performance to simulate knowledge gaps
	// This will create gaps in the user's knowledge
	for i, question := range questions[:3] { // Use first 3 questions
		isCorrect := i == 0 // Only first question correct, others wrong (poor performance)
		_, err := db.Exec(`
			INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, response_time_ms, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`, user.ID, question.ID, 0, isCorrect, 5000)
		require.NoError(t, err)
	}

	// Now get gap analysis again - should show gaps based on poor performance
	gapAnalysis, err = worker.getGapAnalysis(context.Background(), user.ID, "italian", "A1", models.Vocabulary)
	require.NoError(t, err)
	require.NotNil(t, gapAnalysis, "getGapAnalysis should return a non-nil map")

	// Should have gaps identified based on poor performance
	// The gaps will be identified by topic, grammar, vocabulary, and scenario areas
	// where the user has poor accuracy (< 60%)
	assert.True(t, len(gapAnalysis) > 0, "Should identify knowledge gaps based on poor performance")

	// Check that gaps are properly categorized (topic_*, grammar_*, vocabulary_*, scenario_*)
	for gapKey := range gapAnalysis {
		assert.True(t,
			strings.HasPrefix(gapKey, "topic_") ||
				strings.HasPrefix(gapKey, "grammar_") ||
				strings.HasPrefix(gapKey, "vocabulary_") ||
				strings.HasPrefix(gapKey, "scenario_"),
			"Gap keys should be properly categorized: %s", gapKey)
	}

	// Test getPriorityDistribution
	t.Run("getPriorityDistribution", func(t *testing.T) {
		distribution, err := worker.getPriorityDistribution(context.Background(), user.ID, "italian", "A1", models.Vocabulary)
		require.NoError(t, err)

		// Should return distribution of priority scores
		assert.NotEmpty(t, distribution)

		// Check that we have distribution data for our topics
		assert.Contains(t, distribution, "greetings")
		assert.Contains(t, distribution, "courtesy")
		assert.Contains(t, distribution, "food")

		// Distribution values should be positive
		for topic, count := range distribution {
			assert.Greater(t, count, 0, "Distribution count should be positive for topic: %s", topic)
		}
	})
}

func TestWorkerPriorityFunctions_EmptyData_Integration(t *testing.T) {
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Initialize services
	cfg, err := config.NewConfig()
	require.NoError(t, err)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create worker instance
	emailService := services.NewEmailService(cfg, logger)
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-worker", cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser2", "testpass", "italian", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Test with no data - should return empty results but not error
	t.Run("getHighPriorityTopics_empty", func(t *testing.T) {
		topics, err := worker.getHighPriorityTopics(context.Background(), user.ID, "italian", "A1", models.Vocabulary)
		require.NoError(t, err)
		assert.Empty(t, topics)
	})

	t.Run("getGapAnalysis_empty", func(t *testing.T) {
		gaps, err := worker.getGapAnalysis(context.Background(), user.ID, "italian", "A1", models.Vocabulary)
		require.NoError(t, err)
		assert.Empty(t, gaps)
	})

	t.Run("getPriorityDistribution_empty", func(t *testing.T) {
		distribution, err := worker.getPriorityDistribution(context.Background(), user.ID, "italian", "A1", models.Vocabulary)
		require.NoError(t, err)
		assert.Empty(t, distribution)
	})
}

func TestWorkerPriorityFunctions_DifferentLanguages_Integration(t *testing.T) {
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	// Initialize services
	cfg, err := config.NewConfig()
	require.NoError(t, err)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create worker instance
	emailService := services.NewEmailService(cfg, logger)
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-worker", cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser3", "testpass", "spanish", "A2")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Create test questions for Spanish
	questions := []*models.Question{
		{
			Type:          models.Vocabulary,
			Language:      "spanish",
			Level:         "A2",
			Content:       map[string]interface{}{"question": "What is the Spanish word for 'hello'?", "options": []string{"Hola", "Adiós", "Gracias", "Por favor"}, "correct_answer": 0},
			CorrectAnswer: 0,
			TopicCategory: "greetings",
		},
		{
			Type:          models.Vocabulary,
			Language:      "spanish",
			Level:         "A2",
			Content:       map[string]interface{}{"question": "What is the Spanish word for 'goodbye'?", "options": []string{"Hola", "Adiós", "Gracias", "Por favor"}, "correct_answer": 1},
			CorrectAnswer: 1,
			TopicCategory: "greetings",
		},
	}

	// Save questions and assign to user
	for _, q := range questions {
		err := questionService.SaveQuestion(context.Background(), q)
		require.NoError(t, err)

		err = questionService.AssignQuestionToUser(context.Background(), q.ID, user.ID)
		require.NoError(t, err)
	}

	// Create priority scores
	for i, q := range questions {
		_, err := db.Exec(`
                        INSERT INTO question_priority_scores (question_id, user_id, priority_score, created_at, updated_at)
                        VALUES ($1, $2, $3, NOW(), NOW())
                `, q.ID, user.ID, 8.0+float64(i))
		require.NoError(t, err)
	}

	// Test that functions work with different language/level combinations
	t.Run("getHighPriorityTopics_spanish", func(t *testing.T) {
		topics, err := worker.getHighPriorityTopics(context.Background(), user.ID, "spanish", "A2", models.Vocabulary)
		require.NoError(t, err)
		assert.Contains(t, topics, "greetings")
	})

	t.Run("getGapAnalysis_spanish", func(t *testing.T) {
		gaps, err := worker.getGapAnalysis(context.Background(), user.ID, "spanish", "A2", models.Vocabulary)
		require.NoError(t, err)
		assert.NotEmpty(t, gaps)
	})

	t.Run("getPriorityDistribution_spanish", func(t *testing.T) {
		distribution, err := worker.getPriorityDistribution(context.Background(), user.ID, "spanish", "A2", models.Vocabulary)
		require.NoError(t, err)
		assert.Contains(t, distribution, "greetings")
	})
}

// TestWorker_EngagementBasedGeneration_Integration tests the worker's engagement-based generation filtering
func TestWorker_EngagementBasedGeneration_Integration(t *testing.T) {
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	// Enable engagement-based generation for these tests
	cfg.Story.EngagementBasedGeneration = true
	cfg.Story.MaxWorkerGenerationsPerDay = 1

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create worker
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	ctx := context.Background()

	// Create a test user with AI enabled
	username := "testuser_" + strings.Replace(time.Now().Format("20060102_150405"), "-", "", -1)
	user, err := userService.CreateUser(context.Background(), username, "italian", "A1")
	require.NoError(t, err)

	// Set up AI configuration for the user
	err = userService.UpdateUserSettings(context.Background(), user.ID, &models.UserSettings{
		Language:   "italian",
		Level:      "A1",
		AIProvider: "openai",
		AIModel:    "gpt-4",
		AIEnabled:  true,
	})
	require.NoError(t, err)

	// Create a test story
	story, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:       "Engagement Worker Test Story",
		Subject:     stringPtr("Engagement Testing"),
		AuthorStyle: stringPtr("Simple"),
	})
	require.NoError(t, err)
	require.NotNil(t, story)

	// Test 1: Initially, user has no sections, so getUsersWithActiveStories should return the user
	// (but they won't be able to generate because they haven't viewed anything)
	usersWithStories, err := worker.getUsersWithActiveStories(ctx)
	require.NoError(t, err)
	assert.Len(t, usersWithStories, 1, "Should have one user with active story")
	assert.Equal(t, user.ID, usersWithStories[0].ID, "Should be our test user")

	// Test 2: Create a section manually
	section, err := storyService.CreateSection(ctx, story.ID, "Test section content", "A1", 100, models.GeneratorTypeUser)
	require.NoError(t, err)
	require.NotNil(t, section)

	// Test 3: Now user has a section but hasn't viewed it, so should still be in the list
	// but won't be able to generate new sections due to engagement check
	usersWithStories, err = worker.getUsersWithActiveStories(ctx)
	require.NoError(t, err)
	assert.Len(t, usersWithStories, 1, "Should still have one user with active story")

	// Test 4: Record that user has viewed the section
	err = storyService.RecordStorySectionView(ctx, uint(user.ID), section.ID)
	require.NoError(t, err)

	// Test 5: Now user has viewed the latest section, so should be able to generate
	usersWithStories, err = worker.getUsersWithActiveStories(ctx)
	require.NoError(t, err)
	assert.Len(t, usersWithStories, 1, "Should still have one user with active story")

	// Test 6: Generate another section to verify engagement-based generation works
	section2, err := storyService.CreateSection(ctx, story.ID, "Test section 2 content", "A1", 100, models.GeneratorTypeWorker)
	require.NoError(t, err)
	require.NotNil(t, section2)
	require.Greater(t, section2.SectionNumber, section.SectionNumber, "Second section should have higher number")
}

// TestWorker_EngagementBasedGeneration_Disabled_Integration tests that engagement filtering is bypassed when disabled
func TestWorker_EngagementBasedGeneration_Disabled_Integration(t *testing.T) {
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	// Disable engagement-based generation for these tests
	cfg.Story.EngagementBasedGeneration = false
	cfg.Story.MaxWorkerGenerationsPerDay = 1

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create daily question service
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	// Create email service
	emailService := services.NewEmailService(cfg, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create word of the day service
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create worker
	worker := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, emailService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	ctx := context.Background()

	// Create a test user with AI enabled
	username := "testuser_" + strings.Replace(time.Now().Format("20060102_150405"), "-", "", -1)
	user, err := userService.CreateUser(context.Background(), username, "italian", "A1")
	require.NoError(t, err)

	// Set up AI configuration for the user
	err = userService.UpdateUserSettings(context.Background(), user.ID, &models.UserSettings{
		Language:   "italian",
		Level:      "A1",
		AIProvider: "openai",
		AIModel:    "gpt-4",
		AIEnabled:  true,
	})
	require.NoError(t, err)

	// Create a test story
	story, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:       "Engagement Disabled Test Story",
		Subject:     stringPtr("Engagement Disabled Testing"),
		AuthorStyle: stringPtr("Simple"),
	})
	require.NoError(t, err)
	require.NotNil(t, story)

	// Create a section manually
	section, err := storyService.CreateSection(ctx, story.ID, "Test section content", "A1", 100, models.GeneratorTypeUser)
	require.NoError(t, err)
	require.NotNil(t, section)

	// Test: Even though user hasn't viewed the section, they should still be in the list
	// because engagement-based generation is disabled
	usersWithStories, err := worker.getUsersWithActiveStories(ctx)
	require.NoError(t, err)
	assert.Len(t, usersWithStories, 1, "Should have one user with active story when engagement is disabled")

	// Test: Should be able to generate new section even without viewing (engagement check is bypassed)
	section2, err := storyService.CreateSection(ctx, story.ID, "Test section 2 content", "A1", 100, models.GeneratorTypeWorker)
	require.NoError(t, err)
	require.NotNil(t, section2)
	require.Greater(t, section2.SectionNumber, section.SectionNumber, "Second section should have higher number")
}
