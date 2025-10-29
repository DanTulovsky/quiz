package worker

import (
	"context"
	"database/sql"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"
	contextutils "quizapp/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestWorkerSetsUserContextForAI tests that the worker sets user context before calling AI service
func TestWorkerSetsUserContextForAI(t *testing.T) {
	userService := &mockUserService{}
	aiService := &mockAIService{}
	questionService := &mockQuestionService{}
	learningService := &mockLearningService{}
	workerService := &mockWorkerService{}
	dailyQuestionService := &mockDailyQuestionService{}
	storyService := &mockStoryService{}
	emailService := &mockEmailService{}

	w := NewWorker(
		userService,
		questionService,
		aiService,
		learningService,
		workerService,
		dailyQuestionService,
		&mockWordOfTheDayService{},
		storyService,
		emailService,
		nil,
		services.NewInMemoryTranslationCacheRepository(),
		"test",
		testWorkerConfig(),
		observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}),
	)

	user := &models.User{
		ID:         42,
		Username:   "testuser",
		AIProvider: sql.NullString{String: "openai", Valid: true},
		AIModel:    sql.NullString{String: "gpt-4", Valid: true},
	}

	apiKeyID := 123
	userService.On("GetUserAPIKeyWithID", mock.Anything, 42, "openai").Return("test-key", &apiKeyID, nil)

	userConfig, returnedAPIKeyID := w.getUserAIConfig(context.Background(), user)

	assert.Equal(t, "openai", userConfig.Provider)
	assert.Equal(t, "gpt-4", userConfig.Model)
	assert.Equal(t, "test-key", userConfig.APIKey)
	assert.Equal(t, "testuser", userConfig.Username)
	assert.NotNil(t, returnedAPIKeyID)
	assert.Equal(t, 123, *returnedAPIKeyID)

	userService.AssertExpectations(t)
}

// TestWorkerHandleAIQuestionStreamSetsContext tests that handleAIQuestionStream sets user context
func TestWorkerHandleAIQuestionStreamSetsContext(t *testing.T) {
	userService := &mockUserService{}
	aiService := &mockAIService{}
	questionService := &mockQuestionService{}
	learningService := &mockLearningService{}
	workerService := &mockWorkerService{}
	dailyQuestionService := &mockDailyQuestionService{}
	storyService := &mockStoryService{}
	emailService := &mockEmailService{}

	w := NewWorker(
		userService,
		questionService,
		aiService,
		learningService,
		workerService,
		dailyQuestionService,
		&mockWordOfTheDayService{},
		storyService,
		emailService,
		nil,
		services.NewInMemoryTranslationCacheRepository(),
		"test",
		testWorkerConfig(),
		observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}),
	)

	user := &models.User{
		ID:         42,
		Username:   "testuser",
		AIProvider: sql.NullString{String: "openai", Valid: true},
		AIModel:    sql.NullString{String: "gpt-4", Valid: true},
	}

	apiKeyID := 123
	userConfig := &models.UserAIConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Username: "testuser",
	}

	req := &models.AIQuestionGenRequest{
		Language:     "it",
		Level:        "A1",
		QuestionType: models.Vocabulary,
		Count:        1,
	}

	// Variable to capture the context from the call
	var capturedCtx context.Context

	// Mock the AI service and capture the context
	aiService.On("GenerateQuestionsStream",
		mock.Anything, // ctx
		mock.Anything, // userConfig
		mock.Anything, // req
		mock.Anything, // progress channel
		mock.Anything, // variety
	).Run(func(args mock.Arguments) {
		// Capture the context to verify later
		capturedCtx = args.Get(0).(context.Context)
		// Close the channel immediately to end the test
		progressChan := args.Get(3).(chan<- *models.Question)
		close(progressChan)
	}).Return(nil)

	ctx := context.Background()
	_, _, err := w.handleAIQuestionStream(ctx, userConfig, &apiKeyID, req, nil, 1, "it", "A1", models.Vocabulary, "", user)

	assert.NoError(t, err)
	aiService.AssertExpectations(t)

	// Verify context was set correctly
	assert.NotNil(t, capturedCtx)
	assert.Equal(t, 42, contextutils.GetUserIDFromContext(capturedCtx))
	apiKeyIDFromCtx := contextutils.GetAPIKeyIDFromContext(capturedCtx)
	assert.NotNil(t, apiKeyIDFromCtx)
	assert.Equal(t, 123, *apiKeyIDFromCtx)
}

// TestWorkerHandleAIQuestionStreamWithoutAPIKeyID tests context setup when API key ID is nil
func TestWorkerHandleAIQuestionStreamWithoutAPIKeyID(t *testing.T) {
	userService := &mockUserService{}
	aiService := &mockAIService{}
	questionService := &mockQuestionService{}
	learningService := &mockLearningService{}
	workerService := &mockWorkerService{}
	dailyQuestionService := &mockDailyQuestionService{}
	storyService := &mockStoryService{}
	emailService := &mockEmailService{}

	w := NewWorker(
		userService,
		questionService,
		aiService,
		learningService,
		workerService,
		dailyQuestionService,
		&mockWordOfTheDayService{},
		storyService,
		emailService,
		nil,
		services.NewInMemoryTranslationCacheRepository(),
		"test",
		testWorkerConfig(),
		observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}),
	)

	user := &models.User{
		ID:         42,
		Username:   "testuser",
		AIProvider: sql.NullString{String: "openai", Valid: true},
		AIModel:    sql.NullString{String: "gpt-4", Valid: true},
	}

	userConfig := &models.UserAIConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Username: "testuser",
	}

	req := &models.AIQuestionGenRequest{
		Language:     "it",
		Level:        "A1",
		QuestionType: models.Vocabulary,
		Count:        1,
	}

	// Variable to capture the context from the call
	var capturedCtx context.Context

	// Mock the AI service and capture the context
	aiService.On("GenerateQuestionsStream",
		mock.Anything, // ctx
		mock.Anything, // userConfig
		mock.Anything, // req
		mock.Anything, // progress channel
		mock.Anything, // variety
	).Run(func(args mock.Arguments) {
		// Capture the context to verify later
		capturedCtx = args.Get(0).(context.Context)
		// Close the channel immediately to end the test
		progressChan := args.Get(3).(chan<- *models.Question)
		close(progressChan)
	}).Return(nil)

	ctx := context.Background()
	_, _, err := w.handleAIQuestionStream(ctx, userConfig, nil, req, nil, 1, "it", "A1", models.Vocabulary, "", user)

	assert.NoError(t, err)
	aiService.AssertExpectations(t)

	// Verify context was set correctly
	assert.NotNil(t, capturedCtx)
	assert.Equal(t, 42, contextutils.GetUserIDFromContext(capturedCtx))
	apiKeyIDFromCtx := contextutils.GetAPIKeyIDFromContext(capturedCtx)
	assert.Nil(t, apiKeyIDFromCtx, "Expected nil API key ID in context")
}

// TestGetUserAIConfig_WithAPIKeyID tests that getUserAIConfig returns API key ID
func TestGetUserAIConfig_WithAPIKeyID(t *testing.T) {
	userService := &mockUserService{}
	w := NewWorker(
		userService,
		&mockQuestionService{},
		&mockAIService{},
		&mockLearningService{},
		&mockWorkerService{},
		&mockDailyQuestionService{},
		&mockStoryService{},
		&mockEmailService{},
		nil,
		services.NewInMemoryTranslationCacheRepository(),
		"test",
		testWorkerConfig(),
		observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}),
	)

	user := &models.User{
		ID:         1,
		Username:   "alice",
		AIProvider: sql.NullString{String: "anthropic", Valid: true},
		AIModel:    sql.NullString{String: "claude-3", Valid: true},
	}

	keyID := 999
	userService.On("GetUserAPIKeyWithID", mock.Anything, 1, "anthropic").Return("secret-key", &keyID, nil)

	cfg, apiKeyID := w.getUserAIConfig(context.Background(), user)

	assert.Equal(t, "anthropic", cfg.Provider)
	assert.Equal(t, "claude-3", cfg.Model)
	assert.Equal(t, "secret-key", cfg.APIKey)
	assert.Equal(t, "alice", cfg.Username)
	assert.NotNil(t, apiKeyID)
	assert.Equal(t, 999, *apiKeyID)

	userService.AssertExpectations(t)
}

// TestGetUserAIConfig_NoProvider tests behavior when user has no AI provider configured
func TestGetUserAIConfig_NoProvider(t *testing.T) {
	userService := &mockUserService{}
	w := NewWorker(
		userService,
		&mockQuestionService{},
		&mockAIService{},
		&mockLearningService{},
		&mockWorkerService{},
		&mockDailyQuestionService{},
		&mockStoryService{},
		&mockEmailService{},
		nil,
		services.NewInMemoryTranslationCacheRepository(),
		"test",
		testWorkerConfig(),
		observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}),
	)

	user := &models.User{
		ID:         1,
		Username:   "bob",
		AIProvider: sql.NullString{Valid: false}, // No provider configured
		AIModel:    sql.NullString{Valid: false},
	}

	cfg, apiKeyID := w.getUserAIConfig(context.Background(), user)

	assert.Equal(t, "", cfg.Provider)
	assert.Equal(t, "", cfg.Model)
	assert.Equal(t, "", cfg.APIKey)
	assert.Equal(t, "bob", cfg.Username)
	assert.Nil(t, apiKeyID)

	// GetUserAPIKeyWithID should NOT be called when there's no provider
	userService.AssertNotCalled(t, "GetUserAPIKeyWithID")
}
