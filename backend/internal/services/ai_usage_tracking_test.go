package services

import (
	"context"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUsageStatsService is a mock implementation of UsageStatsServiceInterface
type MockUsageStatsService struct {
	mock.Mock
}

func (m *MockUsageStatsService) RecordUsage(ctx context.Context, serviceName, usageType string, characters, requests int) error {
	args := m.Called(ctx, serviceName, usageType, characters, requests)
	return args.Error(0)
}

func (m *MockUsageStatsService) RecordUserAITokenUsage(ctx context.Context, userID int, apiKeyID *int, provider, model, usageType string, promptTokens, completionTokens, totalTokens, requests int) error {
	args := m.Called(ctx, userID, apiKeyID, provider, model, usageType, promptTokens, completionTokens, totalTokens, requests)
	return args.Error(0)
}

func (m *MockUsageStatsService) GetCurrentMonthUsage(ctx context.Context, serviceName, usageType string) (*UsageStats, error) {
	args := m.Called(ctx, serviceName, usageType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UsageStats), args.Error(1)
}

func (m *MockUsageStatsService) GetUsageByMonth(ctx context.Context, serviceName, usageType, month string) (*UsageStats, error) {
	args := m.Called(ctx, serviceName, usageType, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UsageStats), args.Error(1)
}

func (m *MockUsageStatsService) CheckQuota(ctx context.Context, serviceName, usageType string, characters int) error {
	args := m.Called(ctx, serviceName, usageType, characters)
	return args.Error(0)
}

func (m *MockUsageStatsService) GetMonthlyQuota(serviceName string) int64 {
	args := m.Called(serviceName)
	return args.Get(0).(int64)
}

func (m *MockUsageStatsService) GetAllUsageStats(ctx context.Context) ([]*UsageStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*UsageStats), args.Error(1)
}

func (m *MockUsageStatsService) GetUserAITokenUsageStats(ctx context.Context, userID int, startDate, endDate time.Time) ([]*UserUsageStats, error) {
	args := m.Called(ctx, userID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*UserUsageStats), args.Error(1)
}

func (m *MockUsageStatsService) GetUserAITokenUsageStatsByDay(ctx context.Context, userID int, startDate, endDate time.Time) ([]*UserUsageStatsDaily, error) {
	args := m.Called(ctx, userID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*UserUsageStatsDaily), args.Error(1)
}

func (m *MockUsageStatsService) GetUserAITokenUsageStatsByHour(ctx context.Context, userID int, date time.Time) ([]*UserUsageStatsHourly, error) {
	args := m.Called(ctx, userID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*UserUsageStatsHourly), args.Error(1)
}

func (m *MockUsageStatsService) GetUsageStatsByService(ctx context.Context, serviceName string) ([]*UsageStats, error) {
	args := m.Called(ctx, serviceName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*UsageStats), args.Error(1)
}

func (m *MockUsageStatsService) GetUsageStatsByMonth(ctx context.Context, year, month int) ([]*UsageStats, error) {
	args := m.Called(ctx, year, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*UsageStats), args.Error(1)
}

// TestTrackAIUsage_WithValidUserID tests that usage is recorded when a valid user ID is in context
func TestTrackAIUsage_WithValidUserID(t *testing.T) {
	mockUsageStats := new(MockUsageStatsService)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	aiService := &AIService{
		usageStatsSvc: mockUsageStats,
		logger:        logger,
	}

	userID := 42
	apiKeyID := 123
	ctx := context.Background()
	ctx = contextutils.WithUserID(ctx, userID)
	ctx = contextutils.WithAPIKeyID(ctx, apiKeyID)

	userConfig := &models.UserAIConfig{
		Provider: "openai",
		Model:    "gpt-4",
	}

	usage := Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	// Expect the usage stats service to be called with the correct parameters
	mockUsageStats.On("RecordUserAITokenUsage",
		mock.Anything,
		userID,
		&apiKeyID,
		"openai",
		"gpt-4",
		"generic",
		100,
		50,
		150,
		1,
	).Return(nil)

	// Call trackAIUsage
	aiService.trackAIUsage(ctx, userConfig, usage, userID, &apiKeyID)

	// Assert expectations were met
	mockUsageStats.AssertExpectations(t)
}

// TestTrackAIUsage_WithZeroUserID tests that usage is NOT recorded when user ID is 0
func TestTrackAIUsage_WithZeroUserID(t *testing.T) {
	mockUsageStats := new(MockUsageStatsService)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	aiService := &AIService{
		usageStatsSvc: mockUsageStats,
		logger:        logger,
	}

	ctx := context.Background()
	// Don't set user ID in context, so GetUserIDFromContext will return 0

	userConfig := &models.UserAIConfig{
		Provider: "openai",
		Model:    "gpt-4",
	}

	usage := Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	// Usage stats service should NOT be called
	// We don't set any expectations on mockUsageStats

	// Call trackAIUsage - should return early without recording
	aiService.trackAIUsage(ctx, userConfig, usage, 0, nil)

	// Assert no methods were called on the mock
	mockUsageStats.AssertNotCalled(t, "RecordUserAITokenUsage")
}

// TestTrackAIUsage_WithoutAPIKeyID tests that usage is recorded even without API key ID
func TestTrackAIUsage_WithoutAPIKeyID(t *testing.T) {
	mockUsageStats := new(MockUsageStatsService)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	aiService := &AIService{
		usageStatsSvc: mockUsageStats,
		logger:        logger,
	}

	userID := 42
	ctx := context.Background()
	ctx = contextutils.WithUserID(ctx, userID)
	// Don't set API key ID in context

	userConfig := &models.UserAIConfig{
		Provider: "anthropic",
		Model:    "claude-3",
	}

	usage := Usage{
		PromptTokens:     200,
		CompletionTokens: 100,
		TotalTokens:      300,
	}

	// Expect the usage stats service to be called with nil API key ID
	mockUsageStats.On("RecordUserAITokenUsage",
		mock.Anything,
		userID,
		(*int)(nil),
		"anthropic",
		"claude-3",
		"generic",
		200,
		100,
		300,
		1,
	).Return(nil)

	// Call trackAIUsage
	aiService.trackAIUsage(ctx, userConfig, usage, userID, nil)

	// Assert expectations were met
	mockUsageStats.AssertExpectations(t)
}

// TestGetUserIDFromContext_Valid tests extracting valid user ID from context
func TestGetUserIDFromContext_Valid(t *testing.T) {
	ctx := context.Background()
	expectedUserID := 123

	ctx = contextutils.WithUserID(ctx, expectedUserID)

	actualUserID := contextutils.GetUserIDFromContext(ctx)

	assert.Equal(t, expectedUserID, actualUserID)
}

// TestGetUserIDFromContext_Missing tests that missing user ID returns 0
func TestGetUserIDFromContext_Missing(t *testing.T) {
	ctx := context.Background()

	actualUserID := contextutils.GetUserIDFromContext(ctx)

	assert.Equal(t, 0, actualUserID)
}

// TestGetAPIKeyIDFromContext_Valid tests extracting valid API key ID from context
func TestGetAPIKeyIDFromContext_Valid(t *testing.T) {
	ctx := context.Background()
	expectedAPIKeyID := 456

	ctx = contextutils.WithAPIKeyID(ctx, expectedAPIKeyID)

	actualAPIKeyID := contextutils.GetAPIKeyIDFromContext(ctx)

	assert.NotNil(t, actualAPIKeyID)
	assert.Equal(t, expectedAPIKeyID, *actualAPIKeyID)
}

// TestGetAPIKeyIDFromContext_Missing tests that missing API key ID returns nil
func TestGetAPIKeyIDFromContext_Missing(t *testing.T) {
	ctx := context.Background()

	actualAPIKeyID := contextutils.GetAPIKeyIDFromContext(ctx)

	assert.Nil(t, actualAPIKeyID)
}

// TestWithUserID tests setting user ID in context
func TestWithUserID(t *testing.T) {
	ctx := context.Background()
	userID := 789

	ctx = contextutils.WithUserID(ctx, userID)

	retrievedUserID := contextutils.GetUserIDFromContext(ctx)

	assert.Equal(t, userID, retrievedUserID)
}

// TestWithAPIKeyID tests setting API key ID in context
func TestWithAPIKeyID(t *testing.T) {
	ctx := context.Background()
	apiKeyID := 101112

	ctx = contextutils.WithAPIKeyID(ctx, apiKeyID)

	retrievedAPIKeyID := contextutils.GetAPIKeyIDFromContext(ctx)

	assert.NotNil(t, retrievedAPIKeyID)
	assert.Equal(t, apiKeyID, *retrievedAPIKeyID)
}

// TestTrackAIUsage_HandlesError tests that errors from usage stats service are logged but don't crash
func TestTrackAIUsage_HandlesError(t *testing.T) {
	mockUsageStats := new(MockUsageStatsService)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	aiService := &AIService{
		usageStatsSvc: mockUsageStats,
		logger:        logger,
	}

	userID := 42
	ctx := context.Background()
	ctx = contextutils.WithUserID(ctx, userID)

	userConfig := &models.UserAIConfig{
		Provider: "openai",
		Model:    "gpt-4",
	}

	usage := Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	// Simulate an error from the usage stats service
	mockUsageStats.On("RecordUserAITokenUsage",
		mock.Anything,
		userID,
		(*int)(nil),
		"openai",
		"gpt-4",
		"generic",
		100,
		50,
		150,
		1,
	).Return(contextutils.ErrDatabaseQuery)

	// Call trackAIUsage - should not panic despite error
	aiService.trackAIUsage(ctx, userConfig, usage, userID, nil)

	// Assert expectations were met
	mockUsageStats.AssertExpectations(t)
}
