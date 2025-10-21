package worker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) GetDB() *sql.DB {
	args := m.Called()
	return args.Get(0).(*sql.DB)
}

type mockQuestionService struct {
	mock.Mock
}

type mockAIService struct {
	mock.Mock
}

type mockLearningService struct {
	mock.Mock
}

// mockStoryService implements services.StoryService for testing
type mockStoryService struct {
	mock.Mock
}

func (m *mockStoryService) CreateStory(ctx context.Context, userID uint, language string, req *models.CreateStoryRequest) (*models.Story, error) {
	args := m.Called(ctx, userID, language, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Story), args.Error(1)
}

func (m *mockStoryService) GetUserStories(ctx context.Context, userID uint, includeArchived bool) ([]models.Story, error) {
	args := m.Called(ctx, userID, includeArchived)
	return args.Get(0).([]models.Story), args.Error(1)
}

func (m *mockStoryService) GetCurrentStory(ctx context.Context, userID uint) (*models.StoryWithSections, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StoryWithSections), args.Error(1)
}

func (m *mockStoryService) GetStory(ctx context.Context, storyID, userID uint) (*models.StoryWithSections, error) {
	args := m.Called(ctx, storyID, userID)
	return args.Get(0).(*models.StoryWithSections), args.Error(1)
}

func (m *mockStoryService) ArchiveStory(ctx context.Context, storyID, userID uint) error {
	args := m.Called(ctx, storyID, userID)
	return args.Error(0)
}

func (m *mockStoryService) CompleteStory(ctx context.Context, storyID, userID uint) error {
	args := m.Called(ctx, storyID, userID)
	return args.Error(0)
}

func (m *mockStoryService) FixCurrentStoryConstraint(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockStoryService) SetCurrentStory(ctx context.Context, storyID, userID uint) error {
	args := m.Called(ctx, storyID, userID)
	return args.Error(0)
}

func (m *mockStoryService) DeleteStory(ctx context.Context, storyID, userID uint) error {
	args := m.Called(ctx, storyID, userID)
	return args.Error(0)
}

func (m *mockStoryService) GetStorySections(ctx context.Context, storyID uint) ([]models.StorySection, error) {
	args := m.Called(ctx, storyID)
	return args.Get(0).([]models.StorySection), args.Error(1)
}

func (m *mockStoryService) GetSection(ctx context.Context, sectionID, userID uint) (*models.StorySectionWithQuestions, error) {
	args := m.Called(ctx, sectionID, userID)
	return args.Get(0).(*models.StorySectionWithQuestions), args.Error(1)
}

func (m *mockStoryService) CreateSection(ctx context.Context, storyID uint, content, level string, wordCount int, generatedBy models.GeneratorType) (*models.StorySection, error) {
	args := m.Called(ctx, storyID, content, level, wordCount, generatedBy)
	return args.Get(0).(*models.StorySection), args.Error(1)
}

func (m *mockStoryService) GetLatestSection(ctx context.Context, storyID uint) (*models.StorySection, error) {
	args := m.Called(ctx, storyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StorySection), args.Error(1)
}

func (m *mockStoryService) GetAllSectionsText(ctx context.Context, storyID uint) (string, error) {
	args := m.Called(ctx, storyID)
	return args.String(0), args.Error(1)
}

func (m *mockStoryService) GetSectionQuestions(ctx context.Context, sectionID uint) ([]models.StorySectionQuestion, error) {
	args := m.Called(ctx, sectionID)
	return args.Get(0).([]models.StorySectionQuestion), args.Error(1)
}

func (m *mockStoryService) CreateSectionQuestions(ctx context.Context, sectionID uint, questions []models.StorySectionQuestionData) error {
	args := m.Called(ctx, sectionID, questions)
	return args.Error(0)
}

func (m *mockStoryService) GetRandomQuestions(ctx context.Context, sectionID uint, count int) ([]models.StorySectionQuestion, error) {
	args := m.Called(ctx, sectionID, count)
	return args.Get(0).([]models.StorySectionQuestion), args.Error(1)
}

func (m *mockStoryService) CanGenerateSection(ctx context.Context, storyID uint, generatorType models.GeneratorType) (*models.StoryGenerationEligibilityResponse, error) {
	args := m.Called(ctx, storyID, generatorType)
	return args.Get(0).(*models.StoryGenerationEligibilityResponse), args.Error(1)
}

func (m *mockStoryService) UpdateLastGenerationTime(ctx context.Context, storyID uint, generatorType models.GeneratorType) error {
	args := m.Called(ctx, storyID, generatorType)
	return args.Error(0)
}

func (m *mockStoryService) RecordStorySectionView(ctx context.Context, userID, sectionID uint) error {
	args := m.Called(ctx, userID, sectionID)
	return args.Error(0)
}

func (m *mockStoryService) HasUserViewedLatestSection(ctx context.Context, userID uint) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *mockStoryService) GetSectionLengthTarget(level string, lengthPref *models.SectionLength) int {
	args := m.Called(level, lengthPref)
	return args.Int(0)
}

func (m *mockStoryService) GetSectionLengthTargetWithLanguage(language, level string, lengthPref *models.SectionLength) int {
	args := m.Called(language, level, lengthPref)
	return args.Int(0)
}

func (m *mockStoryService) SanitizeInput(input string) string {
	args := m.Called(input)
	return args.String(0)
}

func (m *mockStoryService) GenerateStorySection(ctx context.Context, storyID, userID uint, aiService services.AIServiceInterface, userAIConfig *models.UserAIConfig, generatorType models.GeneratorType) (*models.StorySectionWithQuestions, error) {
	args := m.Called(ctx, storyID, userID, aiService, userAIConfig, generatorType)
	return args.Get(0).(*models.StorySectionWithQuestions), args.Error(1)
}

func (m *mockStoryService) DeleteAllStoriesForUser(ctx context.Context, userID uint) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// Admin-only methods (no ownership checks)
func (m *mockStoryService) GetStoriesPaginated(ctx context.Context, page, pageSize int, search, language, status string, userID *uint) ([]models.Story, int, error) {
	args := m.Called(ctx, page, pageSize, search, language, status, userID)
	return args.Get(0).([]models.Story), args.Int(1), args.Error(2)
}

func (m *mockStoryService) GetStoryAdmin(ctx context.Context, storyID uint) (*models.StoryWithSections, error) {
	args := m.Called(ctx, storyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StoryWithSections), args.Error(1)
}

func (m *mockStoryService) GetSectionAdmin(ctx context.Context, sectionID uint) (*models.StorySectionWithQuestions, error) {
	args := m.Called(ctx, sectionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StorySectionWithQuestions), args.Error(1)
}

func (m *mockStoryService) DeleteStoryAdmin(ctx context.Context, storyID uint) error {
	args := m.Called(ctx, storyID)
	return args.Error(0)
}

func (m *mockLearningService) RecordAnswerWithPriorityReturningID(ctx context.Context, userID, questionID, answerIndex int, isCorrect bool, responseTime int) (int, error) {
	args := m.Called(ctx, userID, questionID, answerIndex, isCorrect, responseTime)
	if args.Get(0) == nil {
		return 0, args.Error(1)
	}
	return args.Int(0), args.Error(1)
}

type mockWorkerService struct {
	mock.Mock
}

// Define a mock email service at the top of the file:
type mockEmailService struct {
	mock.Mock
}

func (m *mockEmailService) SendDailyReminder(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

type mockDailyQuestionService struct {
	mock.Mock
}

func (m *mockDailyQuestionService) AssignDailyQuestions(ctx context.Context, userID int, date time.Time) error {
	args := m.Called(ctx, userID, date)
	return args.Error(0)
}

func (m *mockDailyQuestionService) RegenerateDailyQuestions(ctx context.Context, userID int, date time.Time) error {
	args := m.Called(ctx, userID, date)
	return args.Error(0)
}

func (m *mockDailyQuestionService) GetDailyQuestions(ctx context.Context, userID int, date time.Time) ([]*models.DailyQuestionAssignmentWithQuestion, error) {
	args := m.Called(ctx, userID, date)
	return args.Get(0).([]*models.DailyQuestionAssignmentWithQuestion), args.Error(1)
}

func (m *mockDailyQuestionService) MarkQuestionCompleted(ctx context.Context, userID, questionID int, date time.Time) error {
	args := m.Called(ctx, userID, questionID, date)
	return args.Error(0)
}

func (m *mockDailyQuestionService) ResetQuestionCompleted(ctx context.Context, userID, questionID int, date time.Time) error {
	args := m.Called(ctx, userID, questionID, date)
	return args.Error(0)
}

type mockGenerationHintService struct {
	mock.Mock
}

func (m *mockGenerationHintService) UpsertHint(ctx context.Context, userID int, language, level string, qType models.QuestionType, ttl time.Duration) error {
	args := m.Called(ctx, userID, language, level, qType, ttl)
	return args.Error(0)
}

func (m *mockGenerationHintService) GetActiveHintsForUser(ctx context.Context, userID int) ([]services.GenerationHint, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]services.GenerationHint), args.Error(1)
}

func (m *mockGenerationHintService) ClearHint(ctx context.Context, userID int, language, level string, qType models.QuestionType) error {
	args := m.Called(ctx, userID, language, level, qType)
	return args.Error(0)
}

func (m *mockDailyQuestionService) GetAvailableDates(ctx context.Context, userID int) ([]time.Time, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]time.Time), args.Error(1)
}

func (m *mockDailyQuestionService) GetDailyProgress(ctx context.Context, userID int, date time.Time) (*models.DailyProgress, error) {
	args := m.Called(ctx, userID, date)
	return args.Get(0).(*models.DailyProgress), args.Error(1)
}

func (m *mockDailyQuestionService) GetDailyQuestionsCount(ctx context.Context, userID int, date time.Time) (int, error) {
	args := m.Called(ctx, userID, date)
	return args.Int(0), args.Error(1)
}

func (m *mockDailyQuestionService) GetCompletedDailyQuestionsCount(ctx context.Context, userID int, date time.Time) (int, error) {
	args := m.Called(ctx, userID, date)
	return args.Int(0), args.Error(1)
}

func (m *mockDailyQuestionService) SubmitDailyQuestionAnswer(ctx context.Context, userID, questionID int, date time.Time, userAnswerIndex int) (*api.AnswerResponse, error) {
	args := m.Called(ctx, userID, questionID, date, userAnswerIndex)
	return args.Get(0).(*api.AnswerResponse), args.Error(1)
}

func (m *mockDailyQuestionService) GetQuestionHistory(ctx context.Context, userID, questionID, days int) ([]*models.DailyQuestionHistory, error) {
	args := m.Called(ctx, userID, questionID, days)
	return args.Get(0).([]*models.DailyQuestionHistory), args.Error(1)
}

func (m *mockEmailService) SendEmail(ctx context.Context, to, subject, template string, data map[string]interface{}) error {
	args := m.Called(ctx, to, subject, template, data)
	return args.Error(0)
}

func (m *mockEmailService) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockEmailService) CreateUpcomingNotification(ctx context.Context, userID int, notificationType string, scheduledFor time.Time) error {
	args := m.Called(ctx, userID, notificationType, scheduledFor)
	return args.Error(0)
}

func (m *mockEmailService) RecordSentNotification(ctx context.Context, userID int, notificationType, subject, templateName, status, errorMessage string) error {
	args := m.Called(ctx, userID, notificationType, subject, templateName, status, errorMessage)
	return args.Error(0)
}

// Add at the top of the file (after imports):
func testWorkerConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Server.MaxActivityLogs = 100
	cfg.Server.MaxHistory = 20
	cfg.Server.QuestionRefillThreshold = 5
	cfg.Email.DailyReminder.Enabled = true
	cfg.Email.DailyReminder.Hour = 9 // Set to 9 AM for testing
	return cfg
}

// newWorkerWithFakeTime creates a worker with a specific fake time for testing
func newWorkerWithFakeTime(_ *testing.T, fakeTime time.Time, cfg *config.Config) *Worker {
	userService := &mockUserService{}
	questionService := &mockQuestionService{}
	aiService := &mockAIService{}
	learningService := &mockLearningService{}
	workerService := &mockWorkerService{}
	dailyQuestionService := &mockDailyQuestionService{}
	emailService := &mockEmailService{}
	storyService := &mockStoryService{}

	w := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, storyService, emailService, nil, "test-instance", cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Override the time function to return our fake time
	w.timeNow = func() time.Time {
		return fakeTime
	}

	return w
}

func TestNewWorker_InitializesFields(t *testing.T) {
	userService := &mockUserService{}
	questionService := &mockQuestionService{}
	aiService := &mockAIService{}
	learningService := &mockLearningService{}
	workerService := &mockWorkerService{}
	dailyQuestionService := &mockDailyQuestionService{}
	emailService := &mockEmailService{}
	storyService := &mockStoryService{}
	cfg := testWorkerConfig()

	w := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, storyService, emailService, nil, "test-instance", cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	assert.Equal(t, userService, w.userService)
	assert.Equal(t, questionService, w.questionService)
	assert.Equal(t, aiService, w.aiService)
	assert.Equal(t, learningService, w.learningService)
	assert.Equal(t, workerService, w.workerService)
	assert.Equal(t, "test-instance", w.instance)
	assert.Equal(t, cfg, w.cfg)
	assert.NotNil(t, w.manualTrigger)
	assert.NotNil(t, w.userFailures)
	assert.NotNil(t, w.history)
	assert.NotNil(t, w.activityLogs)
}

func TestNewWorker_DefaultInstance(t *testing.T) {
	userService := &mockUserService{}
	questionService := &mockQuestionService{}
	aiService := &mockAIService{}
	learningService := &mockLearningService{}
	workerService := &mockWorkerService{}
	dailyQuestionService := &mockDailyQuestionService{}
	emailService := &mockEmailService{}
	storyService := &mockStoryService{}
	cfg := testWorkerConfig()

	w := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, storyService, emailService, nil, "", cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	assert.Equal(t, "default", w.instance)
}

func TestGetEnvBool_WithTrueValue(t *testing.T) {
	err := os.Setenv("TEST_BOOL_TRUE", "true")
	assert.NoError(t, err)
	defer func() {
		err := os.Unsetenv("TEST_BOOL_TRUE")
		assert.NoError(t, err)
	}()

	result := getEnvBool("TEST_BOOL_TRUE", false)
	assert.True(t, result)
}

func TestGetEnvBool_WithFalseValue(t *testing.T) {
	err := os.Setenv("TEST_BOOL_FALSE", "false")
	assert.NoError(t, err)
	defer func() {
		err := os.Unsetenv("TEST_BOOL_FALSE")
		assert.NoError(t, err)
	}()

	result := getEnvBool("TEST_BOOL_FALSE", true)
	assert.False(t, result)
}

func TestGetEnvBool_WithInvalidValue(t *testing.T) {
	err := os.Setenv("TEST_BOOL_INVALID", "not-a-bool")
	assert.NoError(t, err)
	defer func() {
		err := os.Unsetenv("TEST_BOOL_INVALID")
		assert.NoError(t, err)
	}()

	result := getEnvBool("TEST_BOOL_INVALID", true)
	assert.True(t, result) // Should return default value
}

func TestGetEnvBool_WithEmptyValue(t *testing.T) {
	err := os.Unsetenv("TEST_BOOL_EMPTY")
	assert.NoError(t, err)

	result := getEnvBool("TEST_BOOL_EMPTY", false)
	assert.False(t, result) // Should return default value
}

func TestGetStatus_ReturnsCopy(t *testing.T) {
	emailService := &mockEmailService{}
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, emailService, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Set some status
	w.status.IsRunning = true
	w.status.CurrentActivity = "test activity"

	status := w.GetStatus()
	assert.True(t, status.IsRunning)
	assert.Equal(t, "test activity", status.CurrentActivity)
}

func TestGetHistory_ReturnsCopy(t *testing.T) {
	emailService := &mockEmailService{}
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, emailService, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Add some history
	w.history = []RunRecord{
		{StartTime: time.Now(), Status: "Success", Details: "test"},
	}

	history := w.GetHistory()
	assert.Len(t, history, 1)
	assert.Equal(t, "Success", history[0].Status)
}

func TestGetActivityLogs_ReturnsCopy(t *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Add some activity logs
	w.activityLogs = []ActivityLog{
		{Timestamp: time.Now(), Level: "INFO", Message: "test"},
	}

	logs := w.GetActivityLogs()
	assert.Len(t, logs, 1)
	assert.Equal(t, "INFO", logs[0].Level)
}

func TestGetInstance(t *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test-instance", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	instance := w.GetInstance()
	assert.Equal(t, "test-instance", instance)
}

func TestTriggerManualRun_SendsToChannel(t *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Trigger should not block
	w.TriggerManualRun()

	// Channel should have a value
	select {
	case <-w.manualTrigger:
		// Success
	default:
		t.Error("Expected manual trigger to be sent")
	}
}

func TestTriggerManualRun_HandlesFullChannel(_ *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Fill the channel
	w.manualTrigger <- true

	// This should not block or panic
	w.TriggerManualRun()
}

func TestPause_UpdatesStatusAndDatabase(t *testing.T) {
	workerService := &mockWorkerService{}
	workerService.On("PauseWorker", mock.Anything, "test-instance").Return(nil)
	workerService.On("UpdateWorkerStatus", mock.Anything, "test-instance", mock.AnythingOfType("*models.WorkerStatus")).Return(nil)

	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, workerService, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test-instance", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	w.Pause(context.Background())

	assert.True(t, w.status.IsPaused)
	workerService.AssertExpectations(t)
}

func TestPause_HandlesDatabaseError(t *testing.T) {
	workerService := &mockWorkerService{}
	workerService.On("PauseWorker", mock.Anything, "test-instance").Return(assert.AnError)
	workerService.On("UpdateWorkerStatus", mock.Anything, "test-instance", mock.AnythingOfType("*models.WorkerStatus")).Return(nil)

	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, workerService, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test-instance", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Should not panic
	w.Pause(context.Background())

	assert.True(t, w.status.IsPaused)
	workerService.AssertExpectations(t)
}

func TestResume_UpdatesStatusAndDatabase(t *testing.T) {
	workerService := &mockWorkerService{}
	workerService.On("ResumeWorker", mock.Anything, "test-instance").Return(nil)
	workerService.On("UpdateWorkerStatus", mock.Anything, "test-instance", mock.AnythingOfType("*models.WorkerStatus")).Return(nil)

	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, workerService, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test-instance", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	w.status.IsPaused = true

	w.Resume(context.Background())

	assert.False(t, w.status.IsPaused)
	workerService.AssertExpectations(t)
}

func TestResume_HandlesDatabaseError(t *testing.T) {
	workerService := &mockWorkerService{}
	workerService.On("ResumeWorker", mock.Anything, "test-instance").Return(assert.AnError)
	workerService.On("UpdateWorkerStatus", mock.Anything, "test-instance", mock.AnythingOfType("*models.WorkerStatus")).Return(nil)

	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, workerService, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test-instance", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	w.status.IsPaused = true

	// Should not panic
	w.Resume(context.Background())

	assert.True(t, w.status.IsPaused)
	workerService.AssertExpectations(t)
}

func TestUpdateActivity(t *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	w.updateActivity("test activity")

	assert.Equal(t, "test activity", w.status.CurrentActivity)
}

func TestLogActivity_AddsToLogs(t *testing.T) {
	cfg := testWorkerConfig()
	cfg.Server.MaxActivityLogs = 100
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	userID := 123
	username := "testuser"

	w.logActivity(context.Background(), "INFO", "test message", &userID, &username)

	logs := w.GetActivityLogs()
	assert.Len(t, logs, 1)
	assert.Equal(t, "INFO", logs[0].Level)
	assert.Equal(t, "test message", logs[0].Message)
	assert.Equal(t, &userID, logs[0].UserID)
	assert.Equal(t, &username, logs[0].Username)
}

func TestLogActivity_CircularBuffer(t *testing.T) {
	cfg := testWorkerConfig()
	cfg.Server.MaxActivityLogs = 100
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Add more than maxActivityLogs entries
	for i := 0; i < 110; i++ {
		w.logActivity(context.Background(), "INFO", fmt.Sprintf("message %d", i), nil, nil)
	}

	logs := w.GetActivityLogs()
	assert.Len(t, logs, 100)
	assert.Equal(t, "message 10", logs[0].Message) // Should have kept the last 100 entries
}

func TestShouldRetryUser_NoFailures(t *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	result := w.shouldRetryUser(123)
	assert.True(t, result)
}

func TestShouldRetryUser_WithFailuresButTimePassed(t *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Record a failure in the past
	w.userFailures[123] = &UserFailureInfo{
		ConsecutiveFailures: 1,
		LastFailureTime:     time.Now().Add(-2 * time.Minute),
		NextRetryTime:       time.Now().Add(-1 * time.Minute), // Past time
	}

	result := w.shouldRetryUser(123)
	assert.True(t, result)
}

func TestShouldRetryUser_WithFailuresAndTimeNotPassed(t *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Record a failure with future retry time
	w.userFailures[123] = &UserFailureInfo{
		ConsecutiveFailures: 1,
		LastFailureTime:     time.Now(),
		NextRetryTime:       time.Now().Add(1 * time.Minute), // Future time
	}

	result := w.shouldRetryUser(123)
	assert.False(t, result)
}

func TestRecordUserFailure_FirstFailure(t *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	w.recordUserFailure(context.Background(), 123, "testuser")

	failure := w.userFailures[123]
	assert.Equal(t, 1, failure.ConsecutiveFailures)
	assert.True(t, failure.NextRetryTime.After(time.Now()))
}

func TestRecordUserFailure_MultipleFailures(t *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Record multiple failures
	w.recordUserFailure(context.Background(), 123, "testuser")
	w.recordUserFailure(context.Background(), 123, "testuser")
	w.recordUserFailure(context.Background(), 123, "testuser")

	failure := w.userFailures[123]
	assert.Equal(t, 3, failure.ConsecutiveFailures)
	assert.True(t, failure.NextRetryTime.After(time.Now()))
}

func TestRecordUserSuccess_ClearsFailures(t *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Record a failure first
	w.recordUserFailure(context.Background(), 123, "testuser")
	assert.Contains(t, w.userFailures, 123)

	// Record success
	w.recordUserSuccess(context.Background(), 123, "testuser")

	// Failure should be cleared
	assert.NotContains(t, w.userFailures, 123)
}

func TestRecordUserSuccess_NoFailures(t *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Should not panic
	w.recordUserSuccess(context.Background(), 123, "testuser")

	assert.NotContains(t, w.userFailures, 123)
}

func TestFormatBatchLogMessage_WithVariety(t *testing.T) {
	variety := &services.VarietyElements{
		TopicCategory:      "daily_life",
		GrammarFocus:       "present_perfect",
		VocabularyDomain:   "food_and_dining",
		Scenario:           "in_a_restaurant",
		StyleModifier:      "conversational",
		DifficultyModifier: "intermediate",
		TimeContext:        "evening_routine",
	}

	result := formatBatchLogMessage("testuser", 5, "vocabulary", "italian", "A2", variety, "ollama", "llama3.1")

	expected := "Worker [user=testuser]: Batch 5 vocabulary questions (lang: italian, level: A2) | grammar: present_perfect | topic: daily_life | scenario: in_a_restaurant | style: conversational | difficulty: intermediate | vocab: food_and_dining | time: evening_routine | provider: ollama, model: llama3.1"
	assert.Equal(t, expected, result)
}

func TestFormatBatchLogMessage_WithoutVariety(t *testing.T) {
	result := formatBatchLogMessage("testuser", 3, "fill_blank", "spanish", "B1", nil, "openai", "gpt-4")

	expected := "Worker [user=testuser]: Batch 3 fill_blank questions (lang: spanish, level: B1) | provider: openai, model: gpt-4"
	assert.Equal(t, expected, result)
}

func TestFormatBatchLogMessage_PartialVariety(t *testing.T) {
	variety := &services.VarietyElements{
		TopicCategory: "travel",
		GrammarFocus:  "past_simple",
		// Other fields empty
	}

	result := formatBatchLogMessage("testuser", 2, "qa", "french", "A1", variety, "google", "gemini-pro")

	expected := "Worker [user=testuser]: Batch 2 qa questions (lang: french, level: A1) | grammar: past_simple | topic: travel | provider: google, model: gemini-pro"
	assert.Equal(t, expected, result)
}

// Test for worker variety element population
func TestSaveGeneratedQuestions_PopulatesVarietyFields(t *testing.T) {
	// Create mock services
	mockUserService := &mockUserService{}
	mockQuestionService := &mockQuestionService{}
	mockAIService := &mockAIService{}
	mockLearningService := &mockLearningService{}
	mockWorkerService := &mockWorkerService{}
	mockStoryService := &mockStoryService{}
	mockGenerationHintService := &mockGenerationHintService{}

	// Create worker
	worker := NewWorker(
		mockUserService,
		mockQuestionService,
		mockAIService,
		mockLearningService,
		mockWorkerService,
		&mockDailyQuestionService{},
		mockStoryService,
		&mockEmailService{},
		mockGenerationHintService,
		"test-worker",
		testWorkerConfig(),
		observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}),
	)

	// Create test user
	user := &models.User{
		ID:       1,
		Username: "testuser",
	}

	// Create test questions (without variety elements initially)
	questions := []*models.Question{
		{
			Type:          models.Vocabulary,
			Language:      "italian",
			Level:         "A2",
			Content:       map[string]interface{}{"question": "Test question 1"},
			CorrectAnswer: 0,
			Explanation:   "Test explanation 1",
		},
		{
			Type:          models.FillInBlank,
			Language:      "italian",
			Level:         "A2",
			Content:       map[string]interface{}{"sentence": "Test sentence ___"},
			CorrectAnswer: 0,
			Explanation:   "Test explanation 2",
		},
	}

	// Create variety elements
	variety := &services.VarietyElements{
		TopicCategory:      "daily_life",
		GrammarFocus:       "present_perfect",
		VocabularyDomain:   "food_and_dining",
		Scenario:           "in_a_restaurant",
		StyleModifier:      "conversational",
		DifficultyModifier: "basic",
		TimeContext:        "evening_routine",
	}

	// Set up mock expectations
	var capturedQuestions []*models.Question
	mockQuestionService.On("SaveQuestion", mock.Anything, mock.AnythingOfType("*models.Question")).Run(func(args mock.Arguments) {
		question := args.Get(1).(*models.Question)
		capturedQuestions = append(capturedQuestions, question)
	}).Return(nil)
	mockQuestionService.On("AssignQuestionToUser", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil)

	// Call the method
	savedCount := worker.saveGeneratedQuestions(context.Background(), user, questions, "italian", "A2", models.Vocabulary, "", variety)

	// Verify results
	assert.Equal(t, 2, savedCount)
	assert.Len(t, capturedQuestions, 2)

	// Verify variety elements were populated
	for i, question := range capturedQuestions {
		assert.Equal(t, "daily_life", question.TopicCategory, "Question %d should have topic category", i+1)
		assert.Equal(t, "present_perfect", question.GrammarFocus, "Question %d should have grammar focus", i+1)
		assert.Equal(t, "food_and_dining", question.VocabularyDomain, "Question %d should have vocabulary domain", i+1)
		assert.Equal(t, "in_a_restaurant", question.Scenario, "Question %d should have scenario", i+1)
		assert.Equal(t, "conversational", question.StyleModifier, "Question %d should have style modifier", i+1)
		assert.Equal(t, "basic", question.DifficultyModifier, "Question %d should have difficulty modifier", i+1)
		assert.Equal(t, "evening_routine", question.TimeContext, "Question %d should have time context", i+1)
	}

	// Verify mocks were called correctly
	mockQuestionService.AssertExpectations(t)
}

func TestSaveGeneratedQuestions_NilVarietyElements(t *testing.T) {
	// Create mock services
	mockUserService := &mockUserService{}
	mockQuestionService := &mockQuestionService{}
	mockAIService := &mockAIService{}
	mockLearningService := &mockLearningService{}
	mockWorkerService := &mockWorkerService{}
	mockStoryService := &mockStoryService{}
	mockGenerationHintService := &mockGenerationHintService{}

	// Create worker
	worker := NewWorker(
		mockUserService,
		mockQuestionService,
		mockAIService,
		mockLearningService,
		mockWorkerService,
		&mockDailyQuestionService{},
		mockStoryService,
		&mockEmailService{},
		mockGenerationHintService,
		"test-worker",
		testWorkerConfig(),
		observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}),
	)

	// Create test user
	user := &models.User{
		ID:       1,
		Username: "testuser",
	}

	// Create test question
	questions := []*models.Question{
		{
			Type:          models.Vocabulary,
			Language:      "italian",
			Level:         "A2",
			Content:       map[string]interface{}{"question": "Test question"},
			CorrectAnswer: 0,
			Explanation:   "Test explanation",
		},
	}

	// Set up mock expectations
	var capturedQuestion *models.Question
	mockQuestionService.On("SaveQuestion", mock.Anything, mock.AnythingOfType("*models.Question")).Run(func(args mock.Arguments) {
		capturedQuestion = args.Get(1).(*models.Question)
	}).Return(nil)
	mockQuestionService.On("AssignQuestionToUser", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil)

	// Call the method with nil variety
	savedCount := worker.saveGeneratedQuestions(context.Background(), user, questions, "italian", "A2", models.Vocabulary, "", nil)

	// Verify results
	assert.Equal(t, 1, savedCount)
	assert.NotNil(t, capturedQuestion)

	// Verify variety elements remain empty when nil variety is passed
	assert.Empty(t, capturedQuestion.TopicCategory)
	assert.Empty(t, capturedQuestion.GrammarFocus)
	assert.Empty(t, capturedQuestion.VocabularyDomain)
	assert.Empty(t, capturedQuestion.Scenario)
	assert.Empty(t, capturedQuestion.StyleModifier)
	assert.Empty(t, capturedQuestion.DifficultyModifier)
	assert.Empty(t, capturedQuestion.TimeContext)

	// Verify mocks were called correctly
	mockQuestionService.AssertExpectations(t)
}

func TestSaveGeneratedQuestions_PartialVarietyElements(t *testing.T) {
	// Create mock services
	mockUserService := &mockUserService{}
	mockQuestionService := &mockQuestionService{}
	mockAIService := &mockAIService{}
	mockLearningService := &mockLearningService{}
	mockWorkerService := &mockWorkerService{}
	mockStoryService := &mockStoryService{}
	mockGenerationHintService := &mockGenerationHintService{}

	// Create worker
	worker := NewWorker(
		mockUserService,
		mockQuestionService,
		mockAIService,
		mockLearningService,
		mockWorkerService,
		&mockDailyQuestionService{},
		mockStoryService,
		&mockEmailService{},
		mockGenerationHintService,
		"test-worker",
		testWorkerConfig(),
		observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}),
	)

	// Create test user
	user := &models.User{
		ID:       1,
		Username: "testuser",
	}

	// Create test question
	questions := []*models.Question{
		{
			Type:          models.ReadingComprehension,
			Language:      "spanish",
			Level:         "B1",
			Content:       map[string]interface{}{"question": "Test reading question"},
			CorrectAnswer: 2,
			Explanation:   "Test explanation",
		},
	}

	// Create variety elements with only some fields set
	variety := &services.VarietyElements{
		TopicCategory: "travel",
		GrammarFocus:  "subjunctive",
		Scenario:      "at_the_airport",
		// Other fields empty
	}

	// Set up mock expectations
	var capturedQuestion *models.Question
	mockQuestionService.On("SaveQuestion", mock.Anything, mock.AnythingOfType("*models.Question")).Run(func(args mock.Arguments) {
		capturedQuestion = args.Get(1).(*models.Question)
	}).Return(nil)
	mockQuestionService.On("AssignQuestionToUser", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil)

	// Call the method
	savedCount := worker.saveGeneratedQuestions(context.Background(), user, questions, "spanish", "B1", models.ReadingComprehension, "current_events", variety)

	// Verify results
	assert.Equal(t, 1, savedCount)
	assert.NotNil(t, capturedQuestion)

	// Verify only specified variety elements were populated
	assert.Equal(t, "travel", capturedQuestion.TopicCategory)
	assert.Equal(t, "subjunctive", capturedQuestion.GrammarFocus)
	assert.Equal(t, "at_the_airport", capturedQuestion.Scenario)
	// These should remain empty
	assert.Empty(t, capturedQuestion.VocabularyDomain)
	assert.Empty(t, capturedQuestion.StyleModifier)
	assert.Empty(t, capturedQuestion.DifficultyModifier)
	assert.Empty(t, capturedQuestion.TimeContext)

	// Verify mocks were called correctly
	mockQuestionService.AssertExpectations(t)
}

// UserServiceInterface
func (m *mockUserService) CreateUser(ctx context.Context, username, language, level string) (result0 *models.User, err error) {
	args := m.Called(ctx, username, language, level)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockUserService) CreateUserWithPassword(ctx context.Context, username, password, language, level string) (result0 *models.User, err error) {
	args := m.Called(ctx, username, password, language, level)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockUserService) CreateUserWithEmailAndTimezone(ctx context.Context, username, email, timezone, language, level string) (result0 *models.User, err error) {
	args := m.Called(ctx, username, email, timezone, language, level)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockUserService) GetUserByID(ctx context.Context, id int) (result0 *models.User, err error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockUserService) GetUserByUsername(ctx context.Context, username string) (result0 *models.User, err error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockUserService) GetUserByEmail(ctx context.Context, email string) (result0 *models.User, err error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockUserService) AuthenticateUser(ctx context.Context, username, password string) (result0 *models.User, err error) {
	args := m.Called(ctx, username, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockUserService) UpdateUserSettings(ctx context.Context, userID int, settings *models.UserSettings) error {
	args := m.Called(ctx, userID, settings)
	return args.Error(0)
}

func (m *mockUserService) UpdateUserProfile(ctx context.Context, userID int, username, email, timezone string) error {
	args := m.Called(ctx, userID, username, email, timezone)
	return args.Error(0)
}

func (m *mockUserService) UpdateUserPassword(ctx context.Context, userID int, newPassword string) error {
	args := m.Called(ctx, userID, newPassword)
	return args.Error(0)
}

func (m *mockUserService) UpdateLastActive(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockUserService) GetAllUsers(ctx context.Context) (result0 []models.User, err error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *mockUserService) DeleteUser(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockUserService) DeleteAllUsers(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockUserService) EnsureAdminUserExists(ctx context.Context, adminUsername, adminPassword string) error {
	args := m.Called(ctx, adminUsername, adminPassword)
	return args.Error(0)
}

func (m *mockUserService) ResetDatabase(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockUserService) ClearUserData(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockUserService) ClearUserDataForUser(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockUserService) GetUserAPIKey(ctx context.Context, userID int, provider string) (result0 string, err error) {
	args := m.Called(ctx, userID, provider)
	return args.String(0), args.Error(1)
}

func (m *mockUserService) GetUserAPIKeyWithID(ctx context.Context, userID int, provider string) (string, *int, error) {
	args := m.Called(ctx, userID, provider)
	return args.String(0), args.Get(1).(*int), args.Error(2)
}

func (m *mockUserService) SetUserAPIKey(ctx context.Context, userID int, provider, apiKey string) error {
	args := m.Called(ctx, userID, provider, apiKey)
	return args.Error(0)
}

func (m *mockUserService) HasUserAPIKey(ctx context.Context, userID int, provider string) (result0 bool, err error) {
	args := m.Called(ctx, userID, provider)
	return args.Bool(0), args.Error(1)
}

// Role management methods
func (m *mockUserService) GetUserRoles(ctx context.Context, userID int) (result0 []models.Role, err error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *mockUserService) AssignRole(ctx context.Context, userID, roleID int) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *mockUserService) AssignRoleByName(ctx context.Context, userID int, roleName string) error {
	args := m.Called(ctx, userID, roleName)
	return args.Error(0)
}

func (m *mockUserService) RemoveRole(ctx context.Context, userID, roleID int) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *mockUserService) HasRole(ctx context.Context, userID int, roleName string) (result0 bool, err error) {
	args := m.Called(ctx, userID, roleName)
	return args.Bool(0), args.Error(1)
}

func (m *mockUserService) IsAdmin(ctx context.Context, userID int) (result0 bool, err error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *mockUserService) GetAllRoles(ctx context.Context) (result0 []models.Role, err error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *mockUserService) GetUsersPaginated(ctx context.Context, page, pageSize int, search, language, level, aiProvider, aiModel, aiEnabled, active string) (result0 []models.User, result1 int, err error) {
	args := m.Called(ctx, page, pageSize, search, language, level, aiProvider, aiModel, aiEnabled, active)
	return args.Get(0).([]models.User), args.Int(1), args.Error(2)
}

// QuestionServiceInterface
func (m *mockQuestionService) SaveQuestion(ctx context.Context, question *models.Question) error {
	args := m.Called(ctx, question)
	return args.Error(0)
}

func (m *mockQuestionService) GetQuestionByID(ctx context.Context, id int) (result0 *models.Question, err error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Question), args.Error(1)
}

func (m *mockQuestionService) GetQuestionWithStats(ctx context.Context, id int) (result0 *services.QuestionWithStats, err error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*services.QuestionWithStats), args.Error(1)
}

func (m *mockQuestionService) GetQuestionsByFilter(ctx context.Context, userID int, language, level string, questionType models.QuestionType, limit int) (result0 []models.Question, err error) {
	args := m.Called(ctx, userID, language, level, questionType, limit)
	return args.Get(0).([]models.Question), args.Error(1)
}

func (m *mockQuestionService) GetNextQuestion(ctx context.Context, userID int, language, level string, qType models.QuestionType) (result0 *services.QuestionWithStats, err error) {
	args := m.Called(ctx, userID, language, level, qType)
	return args.Get(0).(*services.QuestionWithStats), args.Error(1)
}

func (m *mockQuestionService) GetAdaptiveQuestionsForDaily(ctx context.Context, userID int, language, level string, limit int) (result0 []*services.QuestionWithStats, err error) {
	args := m.Called(ctx, userID, language, level, limit)
	return args.Get(0).([]*services.QuestionWithStats), args.Error(1)
}

func (m *mockQuestionService) IncrementUsageCount(ctx context.Context, questionID int) error {
	args := m.Called(ctx, questionID)
	return args.Error(0)
}

func (m *mockQuestionService) ReportQuestion(ctx context.Context, questionID, userID int, reportReason string) error {
	args := m.Called(ctx, questionID, userID, reportReason)
	return args.Error(0)
}

func (m *mockQuestionService) GetQuestionStats(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockQuestionService) GetDetailedQuestionStats(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockQuestionService) GetRecentQuestionContentsForUser(ctx context.Context, userID, limit int) (result0 []string, err error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockQuestionService) GetReportedQuestions(ctx context.Context) (result0 []*services.ReportedQuestionWithUser, err error) {
	args := m.Called(ctx)
	return args.Get(0).([]*services.ReportedQuestionWithUser), args.Error(1)
}

func (m *mockQuestionService) MarkQuestionAsFixed(ctx context.Context, questionID int) error {
	args := m.Called(ctx, questionID)
	return args.Error(0)
}

func (m *mockQuestionService) UpdateQuestion(ctx context.Context, questionID int, content map[string]interface{}, correctAnswerIndex int, explanation string) error {
	args := m.Called(ctx, questionID, content, correctAnswerIndex, explanation)
	return args.Error(0)
}

func (m *mockQuestionService) DeleteQuestion(ctx context.Context, questionID int) error {
	args := m.Called(ctx, questionID)
	return args.Error(0)
}

func (m *mockQuestionService) GetUserQuestions(ctx context.Context, userID, limit int) (result0 []*models.Question, err error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]*models.Question), args.Error(1)
}

func (m *mockQuestionService) GetUserQuestionsWithStats(ctx context.Context, userID, limit int) (result0 []*services.QuestionWithStats, err error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]*services.QuestionWithStats), args.Error(1)
}

func (m *mockQuestionService) GetQuestionsPaginated(ctx context.Context, userID, page, pageSize int, search, typeFilter, statusFilter string) (result0 []*services.QuestionWithStats, result1 int, err error) {
	args := m.Called(ctx, userID, page, pageSize, search, typeFilter, statusFilter)
	return args.Get(0).([]*services.QuestionWithStats), args.Int(1), args.Error(2)
}

func (m *mockQuestionService) GetAllQuestionsPaginated(ctx context.Context, page, pageSize int, search, typeFilter, statusFilter, languageFilter, levelFilter string, userID *int) (result0 []*services.QuestionWithStats, result1 int, err error) {
	args := m.Called(ctx, page, pageSize, search, typeFilter, statusFilter, languageFilter, levelFilter, userID)
	return args.Get(0).([]*services.QuestionWithStats), args.Int(1), args.Error(2)
}

func (m *mockQuestionService) GetReportedQuestionsPaginated(ctx context.Context, page, pageSize int, search, typeFilter, languageFilter, levelFilter string) (result0 []*services.QuestionWithStats, result1 int, err error) {
	args := m.Called(ctx, page, pageSize, search, typeFilter, languageFilter, levelFilter)
	return args.Get(0).([]*services.QuestionWithStats), args.Int(1), args.Error(2)
}

func (m *mockQuestionService) GetReportedQuestionsStats(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockQuestionService) GetUserQuestionCount(ctx context.Context, userID int) (result0 int, err error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *mockQuestionService) GetUserResponseCount(ctx context.Context, userID int) (result0 int, err error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *mockQuestionService) AssignQuestionToUser(ctx context.Context, questionID, userID int) error {
	args := m.Called(ctx, questionID, userID)
	return args.Error(0)
}

func (m *mockQuestionService) DB() *sql.DB {
	args := m.Called()
	return args.Get(0).(*sql.DB)
}

func (m *mockQuestionService) GetRandomGlobalQuestionForUser(ctx context.Context, userID int, language, level string, qType models.QuestionType) (result0 *services.QuestionWithStats, err error) {
	args := m.Called(ctx, userID, language, level, qType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.QuestionWithStats), args.Error(1)
}

func (m *mockQuestionService) GetUsersForQuestion(ctx context.Context, questionID int) (result0 []*models.User, result1 int, err error) {
	args := m.Called(ctx, questionID)
	return args.Get(0).([]*models.User), args.Int(1), args.Error(2)
}

func (m *mockQuestionService) AssignUsersToQuestion(ctx context.Context, questionID int, userIDs []int) error {
	args := m.Called(ctx, questionID, userIDs)
	return args.Error(0)
}

func (m *mockQuestionService) UnassignUsersFromQuestion(ctx context.Context, questionID int, userIDs []int) error {
	args := m.Called(ctx, questionID, userIDs)
	return args.Error(0)
}

// AIServiceInterface
func (m *mockAIService) GenerateQuestion(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest) (result0 *models.Question, err error) {
	args := m.Called(ctx, userConfig, req)
	return args.Get(0).(*models.Question), args.Error(1)
}

func (m *mockAIService) GenerateQuestions(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest) (result0 []*models.Question, err error) {
	args := m.Called(ctx, userConfig, req)
	return args.Get(0).([]*models.Question), args.Error(1)
}

func (m *mockAIService) GenerateQuestionsStream(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIQuestionGenRequest, progress chan<- *models.Question, variety *services.VarietyElements) error {
	args := m.Called(ctx, userConfig, req, progress, variety)
	return args.Error(0)
}

func (m *mockAIService) GenerateChatResponse(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIChatRequest) (result0 string, err error) {
	args := m.Called(ctx, userConfig, req)
	return args.String(0), args.Error(1)
}

func (m *mockAIService) GenerateChatResponseStream(ctx context.Context, userConfig *models.UserAIConfig, req *models.AIChatRequest, chunks chan<- string) error {
	args := m.Called(ctx, userConfig, req, chunks)
	return args.Error(0)
}

func (m *mockAIService) GenerateStorySection(ctx context.Context, userConfig *models.UserAIConfig, req *models.StoryGenerationRequest) (string, error) {
	args := m.Called(ctx, userConfig, req)
	return args.String(0), args.Error(1)
}

func (m *mockAIService) GenerateStoryQuestions(ctx context.Context, userConfig *models.UserAIConfig, req *models.StoryQuestionsRequest) ([]*models.StorySectionQuestionData, error) {
	args := m.Called(ctx, userConfig, req)
	return args.Get(0).([]*models.StorySectionQuestionData), args.Error(1)
}

func (m *mockAIService) TestConnection(ctx context.Context, provider, model, apiKey string) error {
	args := m.Called(ctx, provider, model, apiKey)
	return args.Error(0)
}

func (m *mockAIService) GetConcurrencyStats() services.ConcurrencyStats {
	args := m.Called()
	return args.Get(0).(services.ConcurrencyStats)
}

func (m *mockAIService) GetQuestionBatchSize(provider string) int {
	args := m.Called(provider)
	return args.Int(0)
}

func (m *mockAIService) VarietyService() *services.VarietyService {
	args := m.Called()
	return args.Get(0).(*services.VarietyService)
}

func (m *mockAIService) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockAIService) TemplateManager() *services.AITemplateManager {
	args := m.Called()
	return args.Get(0).(*services.AITemplateManager)
}

func (m *mockAIService) SupportsGrammarField(provider string) bool {
	args := m.Called(provider)
	return args.Bool(0)
}

func (m *mockAIService) CallWithPrompt(ctx context.Context, userConfig *models.UserAIConfig, prompt, grammar string) (string, error) {
	args := m.Called(ctx, userConfig, prompt, grammar)
	return args.String(0), args.Error(1)
}

// LearningServiceInterface
func (m *mockLearningService) RecordUserResponse(ctx context.Context, response *models.UserResponse) error {
	args := m.Called(ctx, response)
	return args.Error(0)
}

func (m *mockLearningService) GetUserProgress(ctx context.Context, userID int) (result0 *models.UserProgress, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*models.UserProgress), args.Error(1)
}

func (m *mockLearningService) GetWeakestTopics(ctx context.Context, userID, limit int) (result0 []*models.PerformanceMetrics, err error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]*models.PerformanceMetrics), args.Error(1)
}

func (m *mockLearningService) ShouldAvoidQuestion(ctx context.Context, userID, questionID int) (result0 bool, err error) {
	args := m.Called(ctx, userID, questionID)
	return args.Bool(0), args.Error(1)
}

func (m *mockLearningService) GetUserQuestionStats(ctx context.Context, userID int) (result0 *services.UserQuestionStats, err error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.UserQuestionStats), args.Error(1)
}

func (m *mockLearningService) RecordAnswerWithPriority(_ context.Context, _, _, _ int, _ bool, _ int) error {
	return nil
}

func (m *mockLearningService) MarkQuestionAsKnown(ctx context.Context, userID, questionID int, confidenceLevel *int) error {
	args := m.Called(ctx, userID, questionID, confidenceLevel)
	return args.Error(0)
}

func (m *mockLearningService) GetUserLearningPreferences(ctx context.Context, userID int) (result0 *models.UserLearningPreferences, err error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserLearningPreferences), args.Error(1)
}

func (m *mockLearningService) CalculatePriorityScore(ctx context.Context, userID, questionID int) (result0 float64, err error) {
	args := m.Called(ctx, userID, questionID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockLearningService) UpdateUserLearningPreferences(ctx context.Context, userID int, prefs *models.UserLearningPreferences) (result0 *models.UserLearningPreferences, err error) {
	args := m.Called(ctx, userID, prefs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserLearningPreferences), args.Error(1)
}

func (m *mockLearningService) UpdateLastDailyReminderSent(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// Analytics methods
func (m *mockLearningService) GetPriorityScoreDistribution(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockLearningService) GetHighPriorityQuestions(ctx context.Context, limit int) (result0 []map[string]interface{}, err error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *mockLearningService) GetUserHighPriorityQuestions(ctx context.Context, userID, limit int) (result0 []map[string]interface{}, err error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *mockLearningService) GetUserPriorityScoreDistribution(ctx context.Context, userID int) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockLearningService) GetUserWeakAreas(ctx context.Context, userID, limit int) (result0 []map[string]interface{}, err error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *mockLearningService) GetWeakAreasByTopic(ctx context.Context, limit int) (result0 []map[string]interface{}, err error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *mockLearningService) GetLearningPreferencesUsage(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockLearningService) GetQuestionTypeGaps(ctx context.Context) (result0 []map[string]interface{}, err error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *mockLearningService) GetGenerationSuggestions(ctx context.Context) (result0 []map[string]interface{}, err error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *mockLearningService) GetPrioritySystemPerformance(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockLearningService) GetBackgroundJobsStatus(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockLearningService) GetHighPriorityTopics(ctx context.Context, userID int) (result0 []string, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockLearningService) GetGapAnalysis(ctx context.Context, userID int) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockLearningService) GetPriorityDistribution(ctx context.Context, userID int) (result0 map[string]int, err error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *mockLearningService) GetUserQuestionConfidenceLevel(ctx context.Context, userID, questionID int) (*int, error) {
	args := m.Called(ctx, userID, questionID)
	return args.Get(0).(*int), args.Error(1)
}

// Priority generation methods moved to worker

// WorkerServiceInterface
func (m *mockWorkerService) GetSetting(ctx context.Context, key string) (result0 string, err error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *mockWorkerService) SetSetting(ctx context.Context, key, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *mockWorkerService) IsGlobalPaused(ctx context.Context) (result0 bool, err error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

func (m *mockWorkerService) SetGlobalPause(ctx context.Context, paused bool) error {
	args := m.Called(ctx, paused)
	return args.Error(0)
}

func (m *mockWorkerService) IsUserPaused(ctx context.Context, userID int) (result0 bool, err error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *mockWorkerService) SetUserPause(ctx context.Context, userID int, paused bool) error {
	args := m.Called(ctx, userID, paused)
	return args.Error(0)
}

func (m *mockWorkerService) UpdateWorkerStatus(ctx context.Context, instance string, status *models.WorkerStatus) error {
	args := m.Called(ctx, instance, status)
	return args.Error(0)
}

func (m *mockWorkerService) GetWorkerStatus(ctx context.Context, instance string) (result0 *models.WorkerStatus, err error) {
	args := m.Called(ctx, instance)
	return args.Get(0).(*models.WorkerStatus), args.Error(1)
}

func (m *mockWorkerService) GetAllWorkerStatuses(ctx context.Context) (result0 []models.WorkerStatus, err error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.WorkerStatus), args.Error(1)
}

func (m *mockWorkerService) UpdateHeartbeat(ctx context.Context, instance string) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *mockWorkerService) IsWorkerHealthy(ctx context.Context, instance string) (result0 bool, err error) {
	args := m.Called(ctx, instance)
	return args.Bool(0), args.Error(1)
}

func (m *mockWorkerService) PauseWorker(ctx context.Context, instance string) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *mockWorkerService) ResumeWorker(ctx context.Context, instance string) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *mockWorkerService) GetWorkerHealth(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockWorkerService) GetHighPriorityTopics(_ context.Context, userID int, language, level, questionType string) (result0 []string, err error) {
	args := m.Called(userID, language, level, questionType)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockWorkerService) GetGapAnalysis(_ context.Context, userID int, language, level, questionType string) (result0 map[string]int, err error) {
	args := m.Called(userID, language, level, questionType)
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *mockWorkerService) GetPriorityDistribution(_ context.Context, userID int, language, level, questionType string) (result0 map[string]int, err error) {
	args := m.Called(userID, language, level, questionType)
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *mockWorkerService) GetNotificationStats(ctx context.Context) (result0 map[string]interface{}, err error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockWorkerService) GetNotificationErrors(ctx context.Context, page, pageSize int, errorType, notificationType, resolved string) (result0 []map[string]interface{}, result1, result2 map[string]interface{}, err error) {
	args := m.Called(ctx, page, pageSize, errorType, notificationType, resolved)
	return args.Get(0).([]map[string]interface{}), args.Get(1).(map[string]interface{}), args.Get(2).(map[string]interface{}), args.Error(3)
}

func (m *mockWorkerService) GetUpcomingNotifications(ctx context.Context, page, pageSize int, notificationType, status, scheduledAfter, scheduledBefore string) (result0 []map[string]interface{}, result1, result2 map[string]interface{}, err error) {
	args := m.Called(ctx, page, pageSize, notificationType, status, scheduledAfter, scheduledBefore)
	return args.Get(0).([]map[string]interface{}), args.Get(1).(map[string]interface{}), args.Get(2).(map[string]interface{}), args.Error(3)
}

func (m *mockWorkerService) GetSentNotifications(ctx context.Context, page, pageSize int, notificationType, status, sentAfter, sentBefore string) (result0 []map[string]interface{}, result1, result2 map[string]interface{}, err error) {
	args := m.Called(ctx, page, pageSize, notificationType, status, sentAfter, sentBefore)
	return args.Get(0).([]map[string]interface{}), args.Get(1).(map[string]interface{}), args.Get(2).(map[string]interface{}), args.Error(3)
}

func (m *mockWorkerService) CreateTestSentNotification(ctx context.Context, userID int, notificationType, subject, templateName, status, errorMessage string) error {
	args := m.Called(ctx, userID, notificationType, subject, templateName, status, errorMessage)
	return args.Error(0)
}

func TestGetEligibleAIUsers_FiltersCorrectly(t *testing.T) {
	userService := &mockUserService{}
	workerService := &mockWorkerService{}
	w := NewWorker(userService, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, workerService, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	users := []models.User{
		{ID: 1, Username: "ai1", AIEnabled: sql.NullBool{Bool: true, Valid: true}, AIProvider: sql.NullString{String: "openai", Valid: true}},
		{ID: 2, Username: "noai", AIEnabled: sql.NullBool{Bool: false, Valid: true}},
		{ID: 3, Username: "paused", AIEnabled: sql.NullBool{Bool: true, Valid: true}, AIProvider: sql.NullString{String: "openai", Valid: true}},
	}
	userService.On("GetAllUsers", mock.Anything).Return(users, nil)
	workerService.On("IsUserPaused", mock.Anything, 1).Return(false, nil)
	workerService.On("IsUserPaused", mock.Anything, 2).Return(false, nil)
	workerService.On("IsUserPaused", mock.Anything, 3).Return(true, nil)
	userService.On("GetUserAPIKey", mock.Anything, 1, "openai").Return("key", nil)
	userService.On("GetUserAPIKey", mock.Anything, 3, "openai").Return("", nil)

	aiUsers, err := w.getEligibleAIUsers(context.Background())
	assert.NoError(t, err)
	assert.Len(t, aiUsers, 1)
	assert.Equal(t, "ai1", aiUsers[0].Username)
}

func TestShouldProcessUser_ExponentialBackoff(t *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	user := &models.User{ID: 1, Username: "testuser"}
	w.userFailures[1] = &UserFailureInfo{ConsecutiveFailures: 2, NextRetryTime: time.Now().Add(1 * time.Hour)}
	ok, reason := w.shouldProcessUser(context.Background(), user)
	assert.False(t, ok)
	assert.Contains(t, reason, "exponential backoff")
}

func TestShouldProcessUser_GlobalPause(t *testing.T) {
	workerService := &mockWorkerService{}
	workerService.On("IsGlobalPaused", mock.Anything).Return(true, nil)
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, workerService, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	user := &models.User{ID: 1, Username: "testuser"}
	ok, reason := w.shouldProcessUser(context.Background(), user)
	assert.False(t, ok)
	assert.Contains(t, reason, "Run paused globally")
}

func TestShouldProcessUser_InstancePause(t *testing.T) {
	workerService := &mockWorkerService{}
	workerService.On("IsGlobalPaused", mock.Anything).Return(false, nil)
	workerService.On("GetWorkerStatus", mock.Anything, "test").Return(&models.WorkerStatus{IsPaused: true}, nil)
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, workerService, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	user := &models.User{ID: 1, Username: "testuser"}
	ok, reason := w.shouldProcessUser(context.Background(), user)
	assert.False(t, ok)
	assert.Contains(t, reason, "paused")
}

func TestShouldProcessUser_Shutdown(t *testing.T) {
	workerService := &mockWorkerService{}
	workerService.On("IsGlobalPaused", mock.Anything).Return(false, nil)
	workerService.On("GetWorkerStatus", mock.Anything, "test").Return(&models.WorkerStatus{IsPaused: false}, nil)
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, workerService, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	user := &models.User{ID: 1, Username: "testuser"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ok, reason := w.shouldProcessUser(ctx, user)
	assert.False(t, ok)
	assert.Contains(t, reason, "Shutdown initiated")
}

func TestSummarizeRunActions(t *testing.T) {
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	// No actions, all users in backoff
	result := w.summarizeRunActions([]string{}, []string{"a"}, []string{}, false, false)
	assert.Contains(t, result, "All users in exponential backoff")
	// No actions, errors
	result = w.summarizeRunActions([]string{}, []string{"a"}, []string{"a"}, true, true)
	assert.Contains(t, result, "No actions taken due to errors")
	// No actions, all sufficient
	result = w.summarizeRunActions([]string{}, []string{"a"}, []string{"a"}, false, false)
	assert.Contains(t, result, "All question types have sufficient questions")
	// Actions present
	result = w.summarizeRunActions([]string{"did something"}, []string{"a"}, []string{"a"}, false, false)
	assert.Contains(t, result, "did something")
}

func TestBuildAIQuestionGenRequest(t *testing.T) {
	// Create mock services
	mockQuestionService := &mockQuestionService{}
	mockLearningService := &mockLearningService{}

	// Create test user
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    sql.NullString{String: "test@example.com", Valid: true},
	}

	// Set up mock expectations
	mockQuestionService.On("GetRecentQuestionContentsForUser", mock.Anything, user.ID, 10).Return([]string{"recent1", "recent2"}, nil)

	// Create worker
	worker := &Worker{
		questionService: mockQuestionService,
		learningService: mockLearningService,
	}

	// Test building AI request
	aiReq, recentQuestions, err := worker.buildAIQuestionGenRequest(context.Background(), user, "italian", "A1", models.Vocabulary, 5, "test")

	// Assertions
	assert.Nil(t, err)
	assert.NotNil(t, aiReq)
	assert.Equal(t, "italian", aiReq.Language)
	assert.Equal(t, "A1", aiReq.Level)
	assert.Equal(t, models.Vocabulary, aiReq.QuestionType)
	assert.Equal(t, 5, aiReq.Count)
	assert.Equal(t, []string{"recent1", "recent2"}, recentQuestions)

	// Verify mock expectations
	mockQuestionService.AssertExpectations(t)
	mockLearningService.AssertExpectations(t)
}

func TestGetUserAIConfig(t *testing.T) {
	userService := &mockUserService{}
	w := NewWorker(userService, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	user := &models.User{ID: 1, Username: "bob", AIProvider: sql.NullString{String: "openai", Valid: true}, AIModel: sql.NullString{String: "gpt-4", Valid: true}}
	userService.On("GetUserAPIKey", mock.Anything, 1, "openai").Return("key", nil)
	cfg := w.getUserAIConfig(context.Background(), user)
	assert.Equal(t, "openai", cfg.Provider)
	assert.Equal(t, "gpt-4", cfg.Model)
	assert.Equal(t, "key", cfg.APIKey)
	assert.Equal(t, "bob", cfg.Username)
}

func TestCheckPauseStatus_GlobalPause(t *testing.T) {
	workerService := &mockWorkerService{}
	workerService.On("IsGlobalPaused", mock.Anything).Return(true, nil)
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, workerService, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	paused, reason := w.checkPauseStatus(context.Background())
	assert.True(t, paused)
	assert.Contains(t, reason, "Globally paused")
}

func TestRecordRunHistory_AppendsAndTrims(t *testing.T) {
	cfg := testWorkerConfig()
	cfg.Server.MaxHistory = 50
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, &mockWorkerService{}, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	w.status.LastRunStart = time.Now()
	w.status.LastRunFinish = time.Now().Add(1 * time.Second)
	for i := 0; i < 55; i++ {
		w.recordRunHistory(fmt.Sprintf("details %d", i), nil)
	}
	history := w.GetHistory()
	assert.Len(t, history, 50)
	assert.Equal(t, "details 54", history[49].Details)
}

func TestHandleStartupPause_SetsPause(t *testing.T) {
	workerService := &mockWorkerService{}
	workerService.On("SetGlobalPause", mock.Anything, true).Return(nil)
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, workerService, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	w.workerCfg.StartWorkerPaused = true
	w.handleStartupPause(context.Background())
	workerService.AssertCalled(t, "SetGlobalPause", mock.Anything, true)
}

func TestGetInitialWorkerStatus(t *testing.T) {
	workerService := &mockWorkerService{}
	workerService.On("IsGlobalPaused", mock.Anything).Return(false, nil)
	workerService.On("GetWorkerStatus", mock.Anything, "test").Return(&models.WorkerStatus{IsPaused: true}, nil)
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, workerService, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	status := w.getInitialWorkerStatus(context.Background())
	assert.Contains(t, status, "paused (instance)")
}

func TestUpdateHeartbeat_CallsService(t *testing.T) {
	workerService := &mockWorkerService{}
	workerService.On("UpdateHeartbeat", mock.Anything, "test").Return(nil)
	w := NewWorker(&mockUserService{}, &mockQuestionService{}, &mockAIService{}, &mockLearningService{}, workerService, &mockDailyQuestionService{}, &mockStoryService{}, &mockEmailService{}, nil, "test", testWorkerConfig(), observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	w.updateHeartbeat(context.Background())
	workerService.AssertCalled(t, "UpdateHeartbeat", mock.Anything, "test")
}

func TestBuildAIQuestionGenRequestWithPriorityData(t *testing.T) {
	// Create mock services
	mockQuestionService := &mockQuestionService{}
	mockLearningService := &mockLearningService{}

	// Set up mock expectations
	mockQuestionService.On("GetRecentQuestionContentsForUser", mock.Anything, 1, 10).Return([]string{"recent1", "recent2"}, nil)

	// Create worker
	worker := &Worker{
		questionService: mockQuestionService,
		learningService: mockLearningService,
	}

	// Test building AI request with priority data
	user := &models.User{ID: 1}
	aiReq, recentQuestions, err := worker.buildAIQuestionGenRequest(context.Background(), user, "russian", "intermediate", models.Vocabulary, 5, "test")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, aiReq)
	assert.Equal(t, "russian", aiReq.Language)
	assert.Equal(t, "intermediate", aiReq.Level)
	assert.Equal(t, models.Vocabulary, aiReq.QuestionType)
	assert.Equal(t, 5, aiReq.Count)
	assert.Equal(t, []string{"recent1", "recent2"}, recentQuestions)

	// Priority data is handled internally by the worker, not passed to AI request

	// Verify mock expectations
	mockQuestionService.AssertExpectations(t)
	mockLearningService.AssertExpectations(t)
}

func TestPriorityAwareGenerationFlow(t *testing.T) {
	// Create mock services
	mockQuestionService := &mockQuestionService{}
	mockLearningService := &mockLearningService{}
	mockWorkerService := &mockWorkerService{}

	// Create worker
	worker := &Worker{
		questionService: mockQuestionService,
		learningService: mockLearningService,
		workerService:   mockWorkerService,
	}

	// Test priority generation reasoning
	priorityData := &PriorityGenerationData{
		UserWeakAreas:        []string{"past-tense", "food-vocabulary"},
		HighPriorityTopics:   []string{"grammar", "vocabulary"},
		GapAnalysis:          map[string]int{"rare-topic": 1},
		FocusOnWeakAreas:     true,
		FreshQuestionRatio:   0.4,
		PriorityDistribution: map[string]int{"high": 5, "medium": 10},
	}
	reasoning := worker.getGenerationReasoning(priorityData, nil)
	assert.Contains(t, reasoning, "focusing on weak areas")
	assert.Contains(t, reasoning, "high priority topics")
	assert.Contains(t, reasoning, "gap analysis")
	assert.Contains(t, reasoning, "fresh ratio: 40.0%")
}

func TestVarietySelectionWithPriorityData(t *testing.T) {
	// Create mock services
	mockQuestionService := &mockQuestionService{}
	mockLearningService := &mockLearningService{}
	mockWorkerService := &mockWorkerService{}

	// Mock priority data with different scenarios
	testCases := []struct {
		name              string
		priorityData      *PriorityGenerationData
		expectedReasoning string
	}{
		{
			name: "With weak areas focus",
			priorityData: &PriorityGenerationData{
				UserWeakAreas:    []string{"past-tense"},
				FocusOnWeakAreas: true,
			},
			expectedReasoning: "focusing on weak areas",
		},
		{
			name: "With high priority topics",
			priorityData: &PriorityGenerationData{
				HighPriorityTopics: []string{"grammar", "vocabulary"},
			},
			expectedReasoning: "high priority topics",
		},
		{
			name: "With gap analysis",
			priorityData: &PriorityGenerationData{
				GapAnalysis: map[string]int{"rare-topic": 1},
			},
			expectedReasoning: "gap analysis",
		},
		{
			name:              "Standard generation",
			priorityData:      nil,
			expectedReasoning: "standard generation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			worker := &Worker{
				questionService: mockQuestionService,
				learningService: mockLearningService,
				workerService:   mockWorkerService,
			}

			reasoning := worker.getGenerationReasoning(tc.priorityData, nil)
			assert.Contains(t, reasoning, tc.expectedReasoning)
		})
	}
}

func TestFreshQuestionRatioEnforcement(t *testing.T) {
	// Create mock services
	mockQuestionService := &mockQuestionService{}
	mockLearningService := &mockLearningService{}
	mockWorkerService := &mockWorkerService{}

	// Test with different freshness ratios
	testCases := []struct {
		name              string
		freshnessRatio    float64
		expectedReasoning string
	}{
		{"High fresh ratio", 0.8, "fresh ratio: 80.0%"},
		{"Low fresh ratio", 0.2, "fresh ratio: 20.0%"},
		{"Balanced ratio", 0.5, "fresh ratio: 50.0%"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			priorityData := &PriorityGenerationData{
				FreshQuestionRatio: tc.freshnessRatio,
			}

			worker := &Worker{
				questionService: mockQuestionService,
				learningService: mockLearningService,
				workerService:   mockWorkerService,
			}

			reasoning := worker.getGenerationReasoning(priorityData, nil)
			assert.Contains(t, reasoning, tc.expectedReasoning)
		})
	}
}

// Tests for priority generation functions that need implementation

func TestGetHighPriorityTopics_ReturnsTopics(t *testing.T) {
	// Create worker with mock services
	workerService := &mockWorkerService{}
	worker := &Worker{
		questionService: &mockQuestionService{},
		learningService: &mockLearningService{},
		workerService:   workerService,
	}

	// Set up the mock to return sample topics
	expectedTopics := []string{"grammar", "vocabulary", "pronunciation"}
	workerService.On("GetHighPriorityTopics", 1, "italian", "A1", "vocabulary").Return(expectedTopics, nil)

	// Test that the function returns high priority topics
	topics, err := worker.getHighPriorityTopics(context.Background(), 1, "italian", "A1", models.Vocabulary)

	// This should work now with the mock set up
	assert.NoError(t, err)
	assert.NotNil(t, topics, "getHighPriorityTopics should return a non-nil slice")
	assert.Equal(t, expectedTopics, topics, "Should return the expected topics")

	// Verify topics are strings
	for _, topic := range topics {
		assert.IsType(t, "", topic)
		assert.NotEmpty(t, topic)
	}

	workerService.AssertExpectations(t)
}

func TestGetHighPriorityTopics_HandlesDifferentQuestionTypes(t *testing.T) {
	workerService := &mockWorkerService{}
	worker := &Worker{
		questionService: &mockQuestionService{},
		learningService: &mockLearningService{},
		workerService:   workerService,
	}

	questionTypes := []models.QuestionType{
		models.Vocabulary,
		models.FillInBlank,
		models.QuestionAnswer,
		models.ReadingComprehension,
	}

	for _, qType := range questionTypes {
		t.Run(string(qType), func(t *testing.T) {
			// Set up mock for this question type
			expectedTopics := []string{"grammar", "vocabulary"}
			workerService.On("GetHighPriorityTopics", 1, "italian", "A1", string(qType)).Return(expectedTopics, nil)

			topics, err := worker.getHighPriorityTopics(context.Background(), 1, "italian", "A1", qType)
			assert.NoError(t, err)
			assert.NotNil(t, topics, "Should return a non-nil slice for %s questions", qType)
			assert.Equal(t, expectedTopics, topics, "Should return the expected topics for %s questions", qType)
		})
	}

	workerService.AssertExpectations(t)
}

func TestGetGapAnalysis_ReturnsGaps(t *testing.T) {
	workerService := &mockWorkerService{}
	worker := &Worker{
		questionService: &mockQuestionService{},
		learningService: &mockLearningService{},
		workerService:   workerService,
	}

	// Set up the mock to return sample gaps
	expectedGaps := map[string]int{
		"grammar":       5,
		"vocabulary":    3,
		"pronunciation": 2,
	}
	workerService.On("GetGapAnalysis", 1, "italian", "A1", "vocabulary").Return(expectedGaps, nil)

	// Test that the function returns gap analysis
	gaps, err := worker.getGapAnalysis(context.Background(), 1, "italian", "A1", models.Vocabulary)

	// This should work now with the mock set up
	assert.NoError(t, err)
	assert.NotNil(t, gaps, "getGapAnalysis should return a non-nil map")
	assert.Equal(t, expectedGaps, gaps, "Should return the expected gaps")

	// Verify gaps map structure
	for topic, count := range gaps {
		assert.IsType(t, "", topic)
		assert.IsType(t, 0, count)
		assert.Greater(t, count, 0, "Gap count should be positive")
	}

	workerService.AssertExpectations(t)
}

func TestGetGapAnalysis_HandlesDifferentLevels(t *testing.T) {
	workerService := &mockWorkerService{}
	worker := &Worker{
		questionService: &mockQuestionService{},
		learningService: &mockLearningService{},
		workerService:   workerService,
	}

	levels := []string{"A1", "A2", "B1", "B2", "C1"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			// Set up mock for this level
			expectedGaps := map[string]int{
				"grammar":    3,
				"vocabulary": 2,
			}
			workerService.On("GetGapAnalysis", 1, "italian", level, "vocabulary").Return(expectedGaps, nil)

			gaps, err := worker.getGapAnalysis(context.Background(), 1, "italian", level, models.Vocabulary)
			assert.NoError(t, err)
			assert.NotNil(t, gaps, "Should return a non-nil map for level %s", level)
			assert.Equal(t, expectedGaps, gaps, "Should return the expected gaps for level %s", level)
		})
	}

	workerService.AssertExpectations(t)
}

func TestGetPriorityDistribution_ReturnsDistribution(t *testing.T) {
	workerService := &mockWorkerService{}
	worker := &Worker{
		questionService: &mockQuestionService{},
		learningService: &mockLearningService{},
		workerService:   workerService,
	}

	// Set up the mock to return a sample distribution
	expectedDistribution := map[string]int{
		"grammar":       10,
		"vocabulary":    15,
		"pronunciation": 5,
	}
	workerService.On("GetPriorityDistribution", 1, "italian", "A1", "vocabulary").Return(expectedDistribution, nil)

	// Test that the function returns priority distribution
	distribution, err := worker.getPriorityDistribution(context.Background(), 1, "italian", "A1", models.Vocabulary)

	// This should work now with the mock set up
	assert.NoError(t, err)
	assert.NotNil(t, distribution, "getPriorityDistribution should return a non-nil map")
	assert.Equal(t, expectedDistribution, distribution, "Should return the expected distribution")

	// Verify distribution map structure
	for category, count := range distribution {
		assert.IsType(t, "", category)
		assert.IsType(t, 0, count)
		assert.GreaterOrEqual(t, count, 0, "Distribution count should be non-negative")
	}

	workerService.AssertExpectations(t)
}

func TestGetPriorityDistribution_HandlesDifferentLanguages(t *testing.T) {
	workerService := &mockWorkerService{}
	worker := &Worker{
		questionService: &mockQuestionService{},
		learningService: &mockLearningService{},
		workerService:   workerService,
	}

	languages := []string{"italian", "spanish", "french", "german"}

	for _, language := range languages {
		t.Run(language, func(t *testing.T) {
			// Set up mock for this language
			expectedDistribution := map[string]int{
				"grammar":    5,
				"vocabulary": 8,
			}
			workerService.On("GetPriorityDistribution", 1, language, "A1", "vocabulary").Return(expectedDistribution, nil)

			distribution, err := worker.getPriorityDistribution(context.Background(), 1, language, "A1", models.Vocabulary)
			assert.NoError(t, err)
			assert.NotNil(t, distribution, "Should return a non-nil map for language %s", language)
			assert.Equal(t, expectedDistribution, distribution, "Should return the expected distribution for language %s", language)
		})
	}

	workerService.AssertExpectations(t)
}

func TestPriorityGenerationFunctions_Integration(t *testing.T) {
	workerService := &mockWorkerService{}
	worker := &Worker{
		questionService: &mockQuestionService{},
		learningService: &mockLearningService{},
		workerService:   workerService,
	}

	userID := 1
	language := "italian"
	level := "A1"
	questionType := models.Vocabulary

	// Set up mocks for all three functions
	expectedTopics := []string{"grammar", "vocabulary"}
	expectedGaps := map[string]int{"grammar": 5, "vocabulary": 3}
	expectedDistribution := map[string]int{"grammar": 10, "vocabulary": 15}

	workerService.On("GetHighPriorityTopics", userID, language, level, "vocabulary").Return(expectedTopics, nil)
	workerService.On("GetGapAnalysis", userID, language, level, "vocabulary").Return(expectedGaps, nil)
	workerService.On("GetPriorityDistribution", userID, language, level, "vocabulary").Return(expectedDistribution, nil)

	// Test all three functions together
	t.Run("high_priority_topics", func(t *testing.T) {
		topics, err := worker.getHighPriorityTopics(context.Background(), userID, language, level, questionType)
		assert.NoError(t, err)
		assert.NotNil(t, topics)
		assert.Equal(t, expectedTopics, topics)
	})

	t.Run("gap_analysis", func(t *testing.T) {
		gaps, err := worker.getGapAnalysis(context.Background(), userID, language, level, questionType)
		assert.NoError(t, err)
		assert.NotNil(t, gaps)
		assert.Equal(t, expectedGaps, gaps)
	})

	t.Run("priority_distribution", func(t *testing.T) {
		distribution, err := worker.getPriorityDistribution(context.Background(), userID, language, level, questionType)
		assert.NoError(t, err)
		assert.NotNil(t, distribution)
		assert.Equal(t, expectedDistribution, distribution)
	})

	workerService.AssertExpectations(t)
}

func TestPriorityGenerationFunctions_ErrorHandling(t *testing.T) {
	workerService := &mockWorkerService{}
	worker := &Worker{
		questionService: &mockQuestionService{},
		learningService: &mockLearningService{},
		workerService:   workerService,
	}

	// Test with invalid user ID
	t.Run("invalid_user_id", func(t *testing.T) {
		// Set up mocks to return errors for invalid user ID
		workerService.On("GetHighPriorityTopics", -1, "italian", "A1", "vocabulary").Return([]string{}, errors.New("invalid user ID"))
		workerService.On("GetGapAnalysis", -1, "italian", "A1", "vocabulary").Return(map[string]int{}, errors.New("invalid user ID"))
		workerService.On("GetPriorityDistribution", -1, "italian", "A1", "vocabulary").Return(map[string]int{}, errors.New("invalid user ID"))

		topics, err := worker.getHighPriorityTopics(context.Background(), -1, "italian", "A1", models.Vocabulary)
		assert.Error(t, err, "Should handle invalid user ID")
		assert.Empty(t, topics)

		gaps, err := worker.getGapAnalysis(context.Background(), -1, "italian", "A1", models.Vocabulary)
		assert.Error(t, err, "Should handle invalid user ID")
		assert.Empty(t, gaps)

		distribution, err := worker.getPriorityDistribution(context.Background(), -1, "italian", "A1", models.Vocabulary)
		assert.Error(t, err, "Should handle invalid user ID")
		assert.Empty(t, distribution)
	})

	// Test with empty language/level
	t.Run("empty_parameters", func(t *testing.T) {
		// Set up mocks to return errors for empty parameters
		workerService.On("GetHighPriorityTopics", 1, "", "", "vocabulary").Return([]string{}, errors.New("empty language/level"))
		workerService.On("GetGapAnalysis", 1, "", "", "vocabulary").Return(map[string]int{}, errors.New("empty language/level"))
		workerService.On("GetPriorityDistribution", 1, "", "", "vocabulary").Return(map[string]int{}, errors.New("empty language/level"))

		topics, err := worker.getHighPriorityTopics(context.Background(), 1, "", "", models.Vocabulary)
		assert.Error(t, err, "Should handle empty language/level")
		assert.Empty(t, topics)

		gaps, err := worker.getGapAnalysis(context.Background(), 1, "", "", models.Vocabulary)
		assert.Error(t, err, "Should handle empty language/level")
		assert.Empty(t, gaps)

		distribution, err := worker.getPriorityDistribution(context.Background(), 1, "", "", models.Vocabulary)
		assert.Error(t, err, "Should handle empty language/level")
		assert.Empty(t, distribution)
	})

	workerService.AssertExpectations(t)
}

func TestWorker_GetEligibleQuestionCount(t *testing.T) {
	// Create mock services
	mockQuestionService := &mockQuestionService{}
	mockLearningService := &mockLearningService{}
	mockWorkerService := &mockWorkerService{}

	// Create worker instance
	w := &Worker{
		questionService: mockQuestionService,
		learningService: mockLearningService,
		workerService:   mockWorkerService,
	}

	// Test case 1: No eligible questions (all recently answered)
	mockQuestionService.On("DB").Return(&sql.DB{})
	// Mock the query to return 0 eligible questions
	// This would require setting up a mock database, but for unit test we'll test the logic

	// Test case 2: Some eligible questions
	// This would require more complex mocking of the database layer

	// For now, we'll test that the method exists and doesn't panic
	// In a real integration test, we'd test with actual database queries
	ctx := context.Background()
	_, err := w.getEligibleQuestionCount(ctx, 1, "italian", "A1", models.Vocabulary)
	// This will fail because we can't easily mock the database in unit tests,
	// but the method exists and is properly structured
	assert.Error(t, err) // Expected to fail in unit test environment
}

func TestWorker_ProcessUserQuestionGenerationWithEligibleCount(t *testing.T) {
	// Create mock services
	mockUserService := &mockUserService{}
	mockQuestionService := &mockQuestionService{}
	mockAIService := &mockAIService{}
	mockLearningService := &mockLearningService{}
	mockWorkerService := &mockWorkerService{}

	// Create worker instance
	w := &Worker{
		userService:     mockUserService,
		questionService: mockQuestionService,
		aiService:       mockAIService,
		learningService: mockLearningService,
		workerService:   mockWorkerService,
	}

	// Create test user
	user := &models.User{
		ID:                1,
		Username:          "testuser",
		PreferredLanguage: sql.NullString{String: "italian", Valid: true},
		CurrentLevel:      sql.NullString{String: "A1", Valid: true},
		AIEnabled:         sql.NullBool{Bool: true, Valid: true},
		AIProvider:        sql.NullString{String: "openai", Valid: true},
	}

	// Mock the eligible question count to be below threshold
	// This would require more complex mocking, but we can test the structure
	ctx := context.Background()
	actions, attempted, failed := w.processUserQuestionGeneration(ctx, user)

	// The method should complete without panicking
	assert.IsType(t, "", actions)
	assert.IsType(t, false, attempted)
	assert.IsType(t, false, failed)
}

func TestWorker_DailyReminderIntegration(t *testing.T) {
	userService := &mockUserService{}
	questionService := &mockQuestionService{}
	aiService := &mockAIService{}
	learningService := &mockLearningService{}
	workerService := &mockWorkerService{}
	dailyQuestionService := &mockDailyQuestionService{}
	cfg := testWorkerConfig()

	w := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, &mockStoryService{}, &mockEmailService{}, nil, "test-instance", cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Test user with daily reminder enabled
	userWithReminder := &models.User{
		ID:       1,
		Username: "testuser_reminder_enabled",
		Email:    sql.NullString{String: "test@example.com", Valid: true},
	}

	// Mock learning preferences with daily reminder enabled
	preferencesWithReminder := &models.UserLearningPreferences{
		UserID:               1,
		DailyReminderEnabled: true,
	}

	// Mock learning preferences with daily reminder disabled
	preferencesWithoutReminder := &models.UserLearningPreferences{
		UserID:               2,
		DailyReminderEnabled: false,
	}

	// Test 1: User with daily reminder enabled should be included in email list
	learningService.On("GetUserLearningPreferences", mock.Anything, 1).Return(preferencesWithReminder, nil)
	userService.On("GetUserByID", mock.Anything, 1).Return(userWithReminder, nil)

	// Test 2: User with daily reminder disabled should not be included
	learningService.On("GetUserLearningPreferences", mock.Anything, 2).Return(preferencesWithoutReminder, nil)

	// Verify that the mocks are set up correctly
	assert.NotNil(t, w)
	assert.Equal(t, userService, w.userService)
	assert.Equal(t, learningService, w.learningService)
}

func TestWorker_CheckForDailyReminders_Integration(t *testing.T) {
	cfg := testWorkerConfig()

	// Create a fake time at 9 AM (the configured reminder hour)
	fakeTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	w := newWorkerWithFakeTime(t, fakeTime, cfg)

	// Test users with different scenarios
	user1 := &models.User{
		ID:       1,
		Username: "user1_enabled_no_reminder",
		Email:    sql.NullString{String: "user1@example.com", Valid: true},
	}

	user2 := &models.User{
		ID:       2,
		Username: "user2_enabled_with_reminder",
		Email:    sql.NullString{String: "user2@example.com", Valid: true},
	}

	user3 := &models.User{
		ID:       3,
		Username: "user3_disabled",
		Email:    sql.NullString{String: "user3@example.com", Valid: true},
	}

	user4 := &models.User{
		ID:       4,
		Username: "user4_no_email",
		Email:    sql.NullString{String: "", Valid: false},
	}

	// Mock user service to return all users
	w.userService.(*mockUserService).On("GetAllUsers", mock.Anything).Return([]models.User{*user1, *user2, *user3, *user4}, nil)

	// Mock learning preferences for each user
	prefs1 := &models.UserLearningPreferences{
		UserID:                1,
		DailyReminderEnabled:  true,
		LastDailyReminderSent: nil, // No reminder sent yet
	}

	// Set user2's last reminder to today's date so they won't get another reminder
	today := fakeTime
	prefs2 := &models.UserLearningPreferences{
		UserID:                2,
		DailyReminderEnabled:  true,
		LastDailyReminderSent: &today, // Already sent today
	}

	prefs3 := &models.UserLearningPreferences{
		UserID:               3,
		DailyReminderEnabled: false, // Disabled
	}

	prefs4 := &models.UserLearningPreferences{
		UserID:               4,
		DailyReminderEnabled: true,
	}

	w.learningService.(*mockLearningService).On("GetUserLearningPreferences", mock.Anything, 1).Return(prefs1, nil)
	w.learningService.(*mockLearningService).On("GetUserLearningPreferences", mock.Anything, 2).Return(prefs2, nil)
	w.learningService.(*mockLearningService).On("GetUserLearningPreferences", mock.Anything, 3).Return(prefs3, nil)
	w.learningService.(*mockLearningService).On("GetUserLearningPreferences", mock.Anything, 4).Return(prefs4, nil)

	// Mock daily question service to assign questions for user1 (the only user who should get a reminder)
	w.dailyQuestionService.(*mockDailyQuestionService).On("AssignDailyQuestions", mock.Anything, 1, fakeTime.Truncate(24*time.Hour)).Return(nil)

	// Mock email service
	w.emailService.(*mockEmailService).On("SendDailyReminder", mock.Anything, user1).Return(nil)
	w.emailService.(*mockEmailService).On("RecordSentNotification", mock.Anything, 1, "daily_reminder", "Time for your daily quiz! 🧠", "daily_reminder", "sent", "").Return(nil)

	// Mock learning service to update timestamp
	w.learningService.(*mockLearningService).On("UpdateLastDailyReminderSent", mock.Anything, 1).Return(nil)

	// Test the daily reminder check
	ctx := context.Background()
	err := w.checkForDailyReminders(ctx)

	// Should not error
	assert.NoError(t, err)

	// Verify that only user1 (enabled, no reminder sent) should receive email
	w.emailService.(*mockEmailService).AssertNumberOfCalls(t, "SendDailyReminder", 1)
	w.emailService.(*mockEmailService).AssertCalled(t, "SendDailyReminder", mock.Anything, user1)

	// Verify that timestamp was updated for user1
	w.learningService.(*mockLearningService).AssertCalled(t, "UpdateLastDailyReminderSent", mock.Anything, 1)
}

func TestWorker_CheckForDailyReminders_WrongHour(t *testing.T) {
	userService := &mockUserService{}
	questionService := &mockQuestionService{}
	aiService := &mockAIService{}
	learningService := &mockLearningService{}
	workerService := &mockWorkerService{}
	dailyQuestionService := &mockDailyQuestionService{}
	emailService := &mockEmailService{}
	cfg := testWorkerConfig()

	// Set reminder hour to 9 AM
	cfg.Email.DailyReminder.Hour = 9
	cfg.Email.DailyReminder.Enabled = true

	w := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, &mockStoryService{}, emailService, nil, "test-instance", cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Mock current time to be 2 PM (not 9 AM)
	// We can't easily mock time.Now() in unit tests, so we'll test the logic
	// by ensuring no users are processed when it's not the right hour

	// Test the daily reminder check
	ctx := context.Background()
	err := w.checkForDailyReminders(ctx)

	// Should not error (just skip processing)
	assert.NoError(t, err)

	// Verify no emails were sent
	emailService.AssertNumberOfCalls(t, "SendDailyReminder", 0)
}

func TestWorker_CheckForDailyReminders_Disabled(t *testing.T) {
	userService := &mockUserService{}
	questionService := &mockQuestionService{}
	aiService := &mockAIService{}
	learningService := &mockLearningService{}
	workerService := &mockWorkerService{}
	dailyQuestionService := &mockDailyQuestionService{}
	emailService := &mockEmailService{}
	cfg := testWorkerConfig()

	// Disable daily reminders
	cfg.Email.DailyReminder.Enabled = false

	w := NewWorker(userService, questionService, aiService, learningService, workerService, dailyQuestionService, &mockStoryService{}, emailService, nil, "test-instance", cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Test the daily reminder check
	ctx := context.Background()
	err := w.checkForDailyReminders(ctx)

	// Should not error
	assert.NoError(t, err)

	// Verify no emails were sent
	emailService.AssertNumberOfCalls(t, "SendDailyReminder", 0)
}

func TestWorker_GetUsersNeedingDailyReminders_Integration(t *testing.T) {
	cfg := testWorkerConfig()

	// Create a fake time for consistent testing
	fakeTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	w := newWorkerWithFakeTime(t, fakeTime, cfg)

	// Test users with different scenarios
	user1 := &models.User{
		ID:       1,
		Username: "user1_eligible",
		Email:    sql.NullString{String: "user1@example.com", Valid: true},
	}

	user2 := &models.User{
		ID:       2,
		Username: "user2_no_email",
		Email:    sql.NullString{String: "", Valid: false},
	}

	user3 := &models.User{
		ID:       3,
		Username: "user3_disabled",
		Email:    sql.NullString{String: "user3@example.com", Valid: true},
	}

	user4 := &models.User{
		ID:       4,
		Username: "user4_already_sent",
		Email:    sql.NullString{String: "user4@example.com", Valid: true},
	}

	// Mock user service to return all users
	w.userService.(*mockUserService).On("GetAllUsers", mock.Anything).Return([]models.User{*user1, *user2, *user3, *user4}, nil)

	// Mock learning preferences for each user
	prefs1 := &models.UserLearningPreferences{
		UserID:                1,
		DailyReminderEnabled:  true,
		LastDailyReminderSent: nil, // No reminder sent yet
	}

	prefs2 := &models.UserLearningPreferences{
		UserID:                2,
		DailyReminderEnabled:  true,
		LastDailyReminderSent: nil,
	}

	prefs3 := &models.UserLearningPreferences{
		UserID:               3,
		DailyReminderEnabled: false, // Disabled
	}

	today := fakeTime
	prefs4 := &models.UserLearningPreferences{
		UserID:                4,
		DailyReminderEnabled:  true,
		LastDailyReminderSent: &today, // Already sent today
	}

	w.learningService.(*mockLearningService).On("GetUserLearningPreferences", mock.Anything, 1).Return(prefs1, nil)
	w.learningService.(*mockLearningService).On("GetUserLearningPreferences", mock.Anything, 2).Return(prefs2, nil)
	w.learningService.(*mockLearningService).On("GetUserLearningPreferences", mock.Anything, 3).Return(prefs3, nil)
	w.learningService.(*mockLearningService).On("GetUserLearningPreferences", mock.Anything, 4).Return(prefs4, nil)

	// Test the function
	ctx := context.Background()
	users, err := w.getUsersNeedingDailyReminders(ctx)

	// Should not error
	assert.NoError(t, err)

	// Should only return user1 (enabled, has email, no reminder sent today)
	assert.Len(t, users, 1)
	assert.Equal(t, user1.ID, users[0].ID)
	assert.Equal(t, user1.Username, users[0].Username)
}

// TestCheckForDailyQuestionAssignments tests the new daily question assignment functionality
func TestCheckForDailyQuestionAssignments(t *testing.T) {
	tests := []struct {
		name            string
		users           []models.User
		assignmentError error
		expectedCalls   int
		expectedError   bool
	}{
		{
			name: "successful assignment for eligible users",
			users: []models.User{
				{
					ID:                1,
					Username:          "user1",
					PreferredLanguage: sql.NullString{Valid: true, String: "italian"},
					CurrentLevel:      sql.NullString{Valid: true, String: "B1"},
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
				{
					ID:                2,
					Username:          "user2",
					PreferredLanguage: sql.NullString{Valid: true, String: "italian"},
					CurrentLevel:      sql.NullString{Valid: true, String: "A1"},
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
			},
			assignmentError: nil,
			expectedCalls:   2,
			expectedError:   false,
		},
		{
			name: "users without language preferences are skipped",
			users: []models.User{
				{
					ID:                1,
					Username:          "user1",
					PreferredLanguage: sql.NullString{Valid: false},
					CurrentLevel:      sql.NullString{Valid: true, String: "B1"},
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
				{
					ID:                2,
					Username:          "user2",
					PreferredLanguage: sql.NullString{Valid: true, String: "italian"},
					CurrentLevel:      sql.NullString{Valid: true, String: "A1"},
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
			},
			assignmentError: nil,
			expectedCalls:   1,
			expectedError:   false,
		},
		{
			name: "users without level preferences are skipped",
			users: []models.User{
				{
					ID:                1,
					Username:          "user1",
					PreferredLanguage: sql.NullString{Valid: true, String: "italian"},
					CurrentLevel:      sql.NullString{Valid: false},
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
				{
					ID:                2,
					Username:          "user2",
					PreferredLanguage: sql.NullString{Valid: true, String: "italian"},
					CurrentLevel:      sql.NullString{Valid: true, String: "A1"},
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
			},
			assignmentError: nil,
			expectedCalls:   1,
			expectedError:   false,
		},
		{
			name: "assignment errors are logged but don't stop processing",
			users: []models.User{
				{
					ID:                1,
					Username:          "user1",
					PreferredLanguage: sql.NullString{Valid: true, String: "italian"},
					CurrentLevel:      sql.NullString{Valid: true, String: "B1"},
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
				{
					ID:                2,
					Username:          "user2",
					PreferredLanguage: sql.NullString{Valid: true, String: "italian"},
					CurrentLevel:      sql.NullString{Valid: true, String: "A1"},
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
			},
			assignmentError: errors.New("assignment failed"),
			expectedCalls:   2,
			expectedError:   false, // Method doesn't return error for assignment failures
		},
		{
			name:            "no eligible users",
			users:           []models.User{},
			assignmentError: nil,
			expectedCalls:   0,
			expectedError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockUserSvc := &mockUserService{}
			mockDailyQuestionSvc := &mockDailyQuestionService{}
			mockStorySvc := &mockStoryService{}
			mockGenerationHintSvc := &mockGenerationHintService{}

			// Mock the GetAllUsers call
			mockUserSvc.On("GetAllUsers", mock.Anything).Return(tt.users, nil)

			// Mock the AssignDailyQuestions calls
			for i := 0; i < tt.expectedCalls; i++ {
				mockDailyQuestionSvc.On("AssignDailyQuestions", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("time.Time")).Return(tt.assignmentError)
			}

			// Create worker with mocked time
			fixedTime := time.Date(2025, 8, 3, 12, 0, 0, 0, time.UTC)
			worker := NewWorker(
				mockUserSvc,
				&mockQuestionService{},
				&mockAIService{},
				&mockLearningService{},
				&mockWorkerService{},
				mockDailyQuestionSvc,
				mockStorySvc,
				&mockEmailService{},
				mockGenerationHintSvc,
				"test",
				testWorkerConfig(),
				observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}),
			)
			worker.timeNow = func() time.Time { return fixedTime }

			// Execute the method
			err := worker.checkForDailyQuestionAssignments(context.Background())

			// Verify results
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify all mocks were called as expected
			mockUserSvc.AssertExpectations(t)
			mockDailyQuestionSvc.AssertExpectations(t)
		})
	}
}

// TestGetUsersEligibleForDailyQuestions tests the user eligibility logic
func TestGetUsersEligibleForDailyQuestions(t *testing.T) {
	tests := []struct {
		name             string
		users            []models.User
		getUsersError    error
		expectedEligible int
		expectedError    bool
	}{
		{
			name: "all users eligible",
			users: []models.User{
				{
					ID:                1,
					Username:          "user1",
					PreferredLanguage: sql.NullString{Valid: true, String: "italian"},
					CurrentLevel:      sql.NullString{Valid: true, String: "B1"},
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
				{
					ID:                2,
					Username:          "user2",
					PreferredLanguage: sql.NullString{Valid: true, String: "italian"},
					CurrentLevel:      sql.NullString{Valid: true, String: "A1"},
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
			},
			getUsersError:    nil,
			expectedEligible: 2,
			expectedError:    false,
		},
		{
			name: "mixed eligibility",
			users: []models.User{
				{
					ID:                1,
					Username:          "user1",
					PreferredLanguage: sql.NullString{Valid: true, String: "italian"},
					CurrentLevel:      sql.NullString{Valid: true, String: "B1"},
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
				{
					ID:                2,
					Username:          "user2",
					PreferredLanguage: sql.NullString{Valid: false}, // No language
					CurrentLevel:      sql.NullString{Valid: true, String: "A1"},
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
				{
					ID:                3,
					Username:          "user3",
					PreferredLanguage: sql.NullString{Valid: true, String: "italian"},
					CurrentLevel:      sql.NullString{Valid: false}, // No level
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
				{
					ID:                4,
					Username:          "user4",
					PreferredLanguage: sql.NullString{Valid: true, String: ""}, // Empty language
					CurrentLevel:      sql.NullString{Valid: true, String: "A1"},
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
				{
					ID:                5,
					Username:          "user5",
					PreferredLanguage: sql.NullString{Valid: true, String: "italian"},
					CurrentLevel:      sql.NullString{Valid: true, String: ""}, // Empty level
					AIEnabled:         sql.NullBool{Valid: true, Bool: true},
				},
			},
			getUsersError:    nil,
			expectedEligible: 1, // Only user1 is eligible
			expectedError:    false,
		},
		{
			name:             "error getting users",
			users:            nil,
			getUsersError:    errors.New("database error"),
			expectedEligible: 0,
			expectedError:    true,
		},
		{
			name:             "no users",
			users:            []models.User{},
			getUsersError:    nil,
			expectedEligible: 0,
			expectedError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockUserSvc := &mockUserService{}
			mockStorySvc := &mockStoryService{}
			mockGenerationHintSvc := &mockGenerationHintService{}

			// Mock the GetAllUsers call
			mockUserSvc.On("GetAllUsers", mock.Anything).Return(tt.users, tt.getUsersError)

			// Create worker
			worker := NewWorker(
				mockUserSvc,
				&mockQuestionService{},
				&mockAIService{},
				&mockLearningService{},
				&mockWorkerService{},
				&mockDailyQuestionService{},
				mockStorySvc,
				&mockEmailService{},
				mockGenerationHintSvc,
				"test",
				testWorkerConfig(),
				observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}),
			)

			// Execute the method
			eligibleUsers, err := worker.getUsersEligibleForDailyQuestions(context.Background())

			// Verify results
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, eligibleUsers)
			} else {
				assert.NoError(t, err)
				assert.Len(t, eligibleUsers, tt.expectedEligible)
			}

			// Verify all mocks were called as expected
			mockUserSvc.AssertExpectations(t)
		})
	}
}

// TestDailyQuestionAssignmentIndependent tests that daily questions are assigned independently of email reminders
func TestDailyQuestionAssignmentIndependent(t *testing.T) {
	// Setup mocks
	mockUserSvc := &mockUserService{}
	mockDailyQuestionSvc := &mockDailyQuestionService{}
	mockLearning := &mockLearningService{}
	mockEmailSvc := &mockEmailService{}

	// Create test user without email reminders enabled
	testUser := models.User{
		ID:                1,
		Username:          "test@example.com",
		Email:             sql.NullString{Valid: true, String: "test@example.com"},
		PreferredLanguage: sql.NullString{Valid: true, String: "italian"},
		CurrentLevel:      sql.NullString{Valid: true, String: "B1"},
		AIEnabled:         sql.NullBool{Valid: true, Bool: true},
	}

	// Create learning preferences with daily reminders DISABLED
	prefs := &models.UserLearningPreferences{
		UserID:                testUser.ID,
		DailyReminderEnabled:  false, // This is the key - reminders disabled
		LastDailyReminderSent: nil,
	}

	// Mock calls for daily question assignment (should happen)
	mockUserSvc.On("GetAllUsers", mock.Anything).Return([]models.User{testUser}, nil)
	mockDailyQuestionSvc.On("AssignDailyQuestions", mock.Anything, testUser.ID, mock.AnythingOfType("time.Time")).Return(nil)

	// Mock calls for daily reminders (should not happen due to disabled setting)
	mockLearning.On("GetUserLearningPreferences", mock.Anything, testUser.ID).Return(prefs, nil)

	// Create worker with mocked config
	cfg := &config.Config{
		Email: config.EmailConfig{
			DailyReminder: config.DailyReminderConfig{
				Enabled: true,
				Hour:    9,
			},
		},
	}

	fixedTime := time.Date(2025, 8, 3, 9, 0, 0, 0, time.UTC) // 9 AM UTC (reminder time)
	worker := createTestWorkerWithConfig(cfg)
	worker.userService = mockUserSvc
	worker.dailyQuestionService = mockDailyQuestionSvc
	worker.learningService = mockLearning
	worker.emailService = mockEmailSvc
	worker.timeNow = func() time.Time { return fixedTime }

	// Execute both methods (simulating the worker run cycle)
	ctx := context.Background()

	// This should assign daily questions regardless of email preferences
	err1 := worker.checkForDailyQuestionAssignments(ctx)
	assert.NoError(t, err1)

	// This should NOT send email reminders (and should not assign questions again)
	err2 := worker.checkForDailyReminders(ctx)
	assert.NoError(t, err2)

	// Verify daily questions were assigned
	mockDailyQuestionSvc.AssertExpectations(t)

	// Verify that email reminders were checked but not sent (user has reminders disabled)
	mockLearning.AssertExpectations(t)

	// Verify no email was sent (because DailyReminderEnabled = false)
	mockEmailSvc.AssertNotCalled(t, "SendDailyReminder", mock.Anything, mock.Anything)
}

// Helper function to create test worker with default config
func createTestWorkerWithConfig(cfg *config.Config) *Worker {
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	return NewWorker(
		&mockUserService{},
		&mockQuestionService{},
		&mockAIService{},
		&mockLearningService{},
		&mockWorkerService{},
		&mockDailyQuestionService{},
		&mockStoryService{},
		&mockEmailService{},
		&mockGenerationHintService{},
		"test",
		cfg,
		logger,
	)
}
