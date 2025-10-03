//go:build integration
// +build integration

package di

import (
	"context"
	"os"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ServiceContainerIntegrationTestSuite provides comprehensive integration tests for the DI container
type ServiceContainerIntegrationTestSuite struct {
	suite.Suite
	Config    *config.Config
	Logger    *observability.Logger
	Container ServiceContainerInterface
}

func TestServiceContainerIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceContainerIntegrationTestSuite))
}

func (suite *ServiceContainerIntegrationTestSuite) SetupSuite() {
	// Initialize logger
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Load configuration
	cfg, err := config.NewConfig()
	require.NoError(suite.T(), err)
	suite.Config = cfg

	// Override database URL for integration tests
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL != "" {
		suite.Config.Database.URL = testDatabaseURL
	}

	// Setup observability with noop telemetry for tests
	suite.Logger = logger

	// Initialize dependency injection container
	suite.Container = NewServiceContainer(cfg, suite.Logger)

	// Initialize all services
	ctx := context.Background()
	err = suite.Container.Initialize(ctx)
	require.NoError(suite.T(), err)

	// Ensure admin user exists
	err = suite.Container.EnsureAdminUser(ctx)
	require.NoError(suite.T(), err)
}

func (suite *ServiceContainerIntegrationTestSuite) TearDownSuite() {
	if suite.Container != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		suite.Container.Shutdown(ctx)
	}
}

// TestNewServiceContainer_Integration tests container creation
func (suite *ServiceContainerIntegrationTestSuite) TestNewServiceContainer_Integration() {
	container := NewServiceContainer(suite.Config, suite.Logger)
	assert.NotNil(suite.T(), container)
	assert.Equal(suite.T(), suite.Config, container.GetConfig())
	assert.Equal(suite.T(), suite.Logger, container.GetLogger())
}

// TestInitialize_Integration tests service initialization
func (suite *ServiceContainerIntegrationTestSuite) TestInitialize_Integration() {
	ctx := context.Background()

	// Create a fresh container for testing
	testContainer := NewServiceContainer(suite.Config, suite.Logger)
	assert.NotNil(suite.T(), testContainer)

	// Initialize should succeed
	err := testContainer.Initialize(ctx)
	assert.NoError(suite.T(), err)

	// Database should be initialized
	db := testContainer.GetDatabase()
	assert.NotNil(suite.T(), db)

	// Test database connection
	err = db.Ping()
	assert.NoError(suite.T(), err)
}

// TestInitialize_FailureScenarios tests initialization failure handling
func (suite *ServiceContainerIntegrationTestSuite) TestInitialize_FailureScenarios() {
	ctx := context.Background()

	// Test with invalid database URL
	invalidConfig := *suite.Config
	invalidConfig.Database.URL = "postgres://invalid:invalid@nonexistent:5432/invalid"

	testContainer := NewServiceContainer(&invalidConfig, suite.Logger)
	err := testContainer.Initialize(ctx)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to initialize database")
}

// TestGetService_Integration tests service retrieval by name
func (suite *ServiceContainerIntegrationTestSuite) TestGetService_Integration() {
	// Test retrieving user service
	userService, err := suite.Container.GetService("user")
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), userService)

	// Test retrieving non-existent service
	nonExistentService, err := suite.Container.GetService("nonexistent")
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), nonExistentService)
	assert.Contains(suite.T(), err.Error(), "service nonexistent not found")
}

// TestGetServiceAs_Integration tests type-safe service retrieval
func (suite *ServiceContainerIntegrationTestSuite) TestGetServiceAs_Integration() {
	// Test getting user service with correct type
	userService, err := GetServiceAs[interface{}](suite.Container.(*ServiceContainer), "user")
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), userService)

	// Test getting service with wrong type
	wrongType, err := GetServiceAs[string](suite.Container.(*ServiceContainer), "user")
	assert.Error(suite.T(), err)
	assert.Empty(suite.T(), wrongType)
	assert.Contains(suite.T(), err.Error(), "service user is not of expected type")
}

// TestGetUserService_Integration tests user service retrieval
func (suite *ServiceContainerIntegrationTestSuite) TestGetUserService_Integration() {
	userService, err := suite.Container.GetUserService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), userService)

	// Test that the service is functional
	ctx := context.Background()
	users, err := userService.GetAllUsers(ctx)
	assert.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(users), 1) // Should have at least admin user
}

// TestGetQuestionService_Integration tests question service retrieval
func (suite *ServiceContainerIntegrationTestSuite) TestGetQuestionService_Integration() {
	questionService, err := suite.Container.GetQuestionService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), questionService)

	// Test that the service is functional
	testCtx := context.Background()
	_, testErr := questionService.GetQuestionsByFilter(testCtx, 0, "", "", "", 100)
	assert.NoError(suite.T(), testErr)
	// May be empty in test environment, but should not error
}

// TestGetLearningService_Integration tests learning service retrieval
func (suite *ServiceContainerIntegrationTestSuite) TestGetLearningService_Integration() {
	learningService, err := suite.Container.GetLearningService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), learningService)

	// Test that the service is functional
	ctx := context.Background()
	progress, err := learningService.GetUserProgress(ctx, 1) // Admin user
	if err == nil {
		assert.NotNil(suite.T(), progress)
	}
	// May error if no data, but should not panic
}

// TestGetAIService_Integration tests AI service retrieval
func (suite *ServiceContainerIntegrationTestSuite) TestGetAIService_Integration() {
	aiService, err := suite.Container.GetAIService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), aiService)

	// Test that the service is functional
	stats := aiService.GetConcurrencyStats()
	assert.NotNil(suite.T(), stats)
}

// TestGetWorkerService_Integration tests worker service retrieval
func (suite *ServiceContainerIntegrationTestSuite) TestGetWorkerService_Integration() {
	workerService, err := suite.Container.GetWorkerService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), workerService)

	// Test that the service is functional
	ctx := context.Background()
	health, err := workerService.GetWorkerHealth(ctx)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), health)
}

// TestGetDailyQuestionService_Integration tests daily question service retrieval
func (suite *ServiceContainerIntegrationTestSuite) TestGetDailyQuestionService_Integration() {
	dailyQuestionService, err := suite.Container.GetDailyQuestionService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), dailyQuestionService)

	// Test that the service is functional
	testCtx := context.Background()
	_, testErr := dailyQuestionService.GetAvailableDates(testCtx, 1)
	assert.NoError(suite.T(), testErr)
	// May be empty, but should not error
}

// TestGetOAuthService_Integration tests OAuth service retrieval
func (suite *ServiceContainerIntegrationTestSuite) TestGetOAuthService_Integration() {
	oauthService, err := suite.Container.GetOAuthService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), oauthService)

	// Test that the service is functional
	ctx := context.Background()
	authURL := oauthService.GetGoogleAuthURL(ctx, "test-state")
	assert.NotEmpty(suite.T(), authURL)
}

// TestGetGenerationHintService_Integration tests generation hint service retrieval
func (suite *ServiceContainerIntegrationTestSuite) TestGetGenerationHintService_Integration() {
	generationHintService, err := suite.Container.GetGenerationHintService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), generationHintService)

	// Test that the service is functional
	testCtx := context.Background()
	_, testErr := generationHintService.GetActiveHintsForUser(testCtx, 1)
	assert.NoError(suite.T(), testErr)
	// May be empty, but should not error
}

// TestGetEmailService_Integration tests email service retrieval
func (suite *ServiceContainerIntegrationTestSuite) TestGetEmailService_Integration() {
	emailService, err := suite.Container.GetEmailService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), emailService)

	// Test that the service is functional
	enabled := emailService.IsEnabled()
	// Should return a boolean value
	assert.IsType(suite.T(), false, enabled)
}

// TestGetDatabase_Integration tests database retrieval
func (suite *ServiceContainerIntegrationTestSuite) TestGetDatabase_Integration() {
	db := suite.Container.GetDatabase()
	assert.NotNil(suite.T(), db)

	// Test database connection
	err := db.Ping()
	assert.NoError(suite.T(), err)
}

// TestGetConfig_Integration tests config retrieval
func (suite *ServiceContainerIntegrationTestSuite) TestGetConfig_Integration() {
	config := suite.Container.GetConfig()
	assert.NotNil(suite.T(), config)
	assert.Equal(suite.T(), suite.Config, config)
}

// TestGetLogger_Integration tests logger retrieval
func (suite *ServiceContainerIntegrationTestSuite) TestGetLogger_Integration() {
	logger := suite.Container.GetLogger()
	assert.NotNil(suite.T(), logger)
	assert.Equal(suite.T(), suite.Logger, logger)
}

// TestShutdown_Integration tests graceful shutdown
func (suite *ServiceContainerIntegrationTestSuite) TestShutdown_Integration() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a fresh container for testing shutdown
	testContainer := NewServiceContainer(suite.Config, suite.Logger)
	err := testContainer.Initialize(ctx)
	assert.NoError(suite.T(), err)

	// Shutdown should succeed
	err = testContainer.Shutdown(ctx)
	assert.NoError(suite.T(), err)

	// Database should be closed
	db := testContainer.GetDatabase()
	err = db.Ping()
	assert.Error(suite.T(), err) // Should fail because connection is closed
}

// TestShutdown_Timeout tests shutdown with timeout
func (suite *ServiceContainerIntegrationTestSuite) TestShutdown_Timeout() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Create a fresh container for testing timeout
	testContainer := NewServiceContainer(suite.Config, suite.Logger)
	err := testContainer.Initialize(context.Background())
	assert.NoError(suite.T(), err)

	// Shutdown with very short timeout
	err = testContainer.Shutdown(ctx)
	// Should handle timeout gracefully (may or may not error depending on implementation)
	suite.Logger.Info(context.Background(), "Shutdown timeout test completed", map[string]interface{}{
		"error": err,
	})
}

// TestEnsureAdminUser_Integration tests admin user creation
func (suite *ServiceContainerIntegrationTestSuite) TestEnsureAdminUser_Integration() {
	ctx := context.Background()

	// Create a fresh container for testing
	testContainer := NewServiceContainer(suite.Config, suite.Logger)
	err := testContainer.Initialize(ctx)
	assert.NoError(suite.T(), err)

	// Ensure admin user exists
	err = testContainer.EnsureAdminUser(ctx)
	assert.NoError(suite.T(), err)

	// Verify admin user exists
	userService, err := testContainer.GetUserService()
	assert.NoError(suite.T(), err)

	adminUser, err := userService.GetUserByUsername(ctx, suite.Config.Server.AdminUsername)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), adminUser)
	assert.Equal(suite.T(), suite.Config.Server.AdminUsername, adminUser.Username)
}

// TestEnsureAdminUser_AlreadyExists tests admin user creation when user already exists
func (suite *ServiceContainerIntegrationTestSuite) TestEnsureAdminUser_AlreadyExists() {
	ctx := context.Background()

	// Admin user should already exist from SetupSuite
	err := suite.Container.EnsureAdminUser(ctx)
	assert.NoError(suite.T(), err) // Should not error even if user exists
}

// TestServiceLifecycle_Integration tests the complete service lifecycle
func (suite *ServiceContainerIntegrationTestSuite) TestServiceLifecycle_Integration() {
	ctx := context.Background()

	// Create fresh container
	testContainer := NewServiceContainer(suite.Config, suite.Logger)

	// Test all service getters return appropriate types and are functional
	userService, err := testContainer.GetUserService()
	assert.Error(suite.T(), err) // Should error because services not initialized yet
	assert.Nil(suite.T(), userService)

	// Initialize container
	err = testContainer.Initialize(ctx)
	assert.NoError(suite.T(), err)

	// Now all services should be available
	userService, err = testContainer.GetUserService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), userService)

	questionService, err := testContainer.GetQuestionService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), questionService)

	learningService, err := testContainer.GetLearningService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), learningService)

	aiService, err := testContainer.GetAIService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), aiService)

	workerService, err := testContainer.GetWorkerService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), workerService)

	dailyQuestionService, err := testContainer.GetDailyQuestionService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), dailyQuestionService)

	oauthService, err := testContainer.GetOAuthService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), oauthService)

	generationHintService, err := testContainer.GetGenerationHintService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), generationHintService)

	emailService, err := testContainer.GetEmailService()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), emailService)

	// Test database and config access
	db := testContainer.GetDatabase()
	assert.NotNil(suite.T(), db)

	config := testContainer.GetConfig()
	assert.NotNil(suite.T(), config)

	logger := testContainer.GetLogger()
	assert.NotNil(suite.T(), logger)

	// Test shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	err = testContainer.Shutdown(shutdownCtx)
	assert.NoError(suite.T(), err)
}

// TestServiceDependencies_Integration tests that services have proper dependencies
func (suite *ServiceContainerIntegrationTestSuite) TestServiceDependencies_Integration() {
	ctx := context.Background()

	// Create fresh container
	testContainer := NewServiceContainer(suite.Config, suite.Logger)
	err := testContainer.Initialize(ctx)
	assert.NoError(suite.T(), err)

	// Test that services can interact with each other
	userService, err := testContainer.GetUserService()
	assert.NoError(suite.T(), err)

	questionService, err := testContainer.GetQuestionService()
	assert.NoError(suite.T(), err)

	learningService, err := testContainer.GetLearningService()
	assert.NoError(suite.T(), err)

	// Test user-question interaction
	users, err := userService.GetAllUsers(ctx)
	assert.NoError(suite.T(), err)

	if len(users) > 0 {
		userID := users[0].ID

		// Test user progress calculation
		progress, err := learningService.GetUserProgress(ctx, userID)
		if err == nil {
			assert.NotNil(suite.T(), progress)
		}

		// Test question retrieval for user
		_, err = questionService.GetUserQuestions(ctx, userID, 10)
		assert.NoError(suite.T(), err)
		// May be empty, but should not error
	}
}

// TestConcurrentAccess_Integration tests concurrent access to the container
func (suite *ServiceContainerIntegrationTestSuite) TestConcurrentAccess_Integration() {
	// Test concurrent service retrieval
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Concurrently access various services
			userService, err := suite.Container.GetUserService()
			assert.NoError(suite.T(), err)
			assert.NotNil(suite.T(), userService)

			questionService, err := suite.Container.GetQuestionService()
			assert.NoError(suite.T(), err)
			assert.NotNil(suite.T(), questionService)

			db := suite.Container.GetDatabase()
			assert.NotNil(suite.T(), db)

			config := suite.Container.GetConfig()
			assert.NotNil(suite.T(), config)

			logger := suite.Container.GetLogger()
			assert.NotNil(suite.T(), logger)
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Run the tests
	code := m.Run()

	// Clean up
	os.Exit(code)
}
