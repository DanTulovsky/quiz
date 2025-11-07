//go:build integration

package main

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ResetDBIntegrationTestSuite provides comprehensive integration tests for the reset-db CLI tool
type ResetDBIntegrationTestSuite struct {
	suite.Suite
	DB          *sql.DB
	UserService *services.UserService
	Logger      *observability.Logger
	Config      *config.Config
}

func TestResetDBIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ResetDBIntegrationTestSuite))
}

func (suite *ResetDBIntegrationTestSuite) SetupSuite() {
	// Load configuration
	cfg, err := config.NewConfig()
	require.NoError(suite.T(), err)
	suite.Config = cfg

	// Setup observability with noop telemetry for tests
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	suite.Logger = logger

	// Initialize database manager
	dbManager := database.NewManager(logger)

	// Use environment variable for test database URL, fallback to default
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	// Initialize database connection
	db, err := dbManager.InitDB(testDBURL)
	require.NoError(suite.T(), err)
	suite.DB = db

	// Initialize user service
	suite.UserService = services.NewUserServiceWithLogger(db, cfg, logger)
}

func (suite *ResetDBIntegrationTestSuite) TearDownSuite() {
	if suite.DB != nil {
		suite.DB.Close()
	}
}

func (suite *ResetDBIntegrationTestSuite) SetupTest() {
	suite.cleanupDatabase()
	suite.setupTestData()
}

func (suite *ResetDBIntegrationTestSuite) TearDownTest() {
	suite.cleanupDatabase()
}

func (suite *ResetDBIntegrationTestSuite) cleanupDatabase() {
	// Use the shared database cleanup function
	services.CleanupTestDatabase(suite.DB, suite.T())
}

func (suite *ResetDBIntegrationTestSuite) setupTestData() {
	// Create test users
	_, err := suite.DB.Exec(`
		INSERT INTO users (username, email, password_hash, preferred_language, current_level, created_at, updated_at)
		VALUES
			('testuser1', 'test1@example.com', '$2a$10$test', 'english', 'A1', NOW(), NOW()),
			('testuser2', 'test2@example.com', '$2a$10$test', 'spanish', 'B1', NOW(), NOW()),
			('admin', 'admin@example.com', '$2a$10$test', 'english', 'A1', NOW(), NOW())
	`)
	require.NoError(suite.T(), err)

	// Create test questions
	_, err = suite.DB.Exec(`
		INSERT INTO questions (type, language, level, content, correct_answer, created_at)
		VALUES
			('vocabulary', 'english', 'A1', '{"question_text": "Test question 1", "options": ["a", "b", "c", "d"]}', 0, NOW()),
			('vocabulary', 'english', 'A1', '{"question_text": "Test question 2", "options": ["a", "b", "c", "d"]}', 0, NOW())
	`)
	require.NoError(suite.T(), err)

	// Create test user responses
	_, err = suite.DB.Exec(`
		INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, response_time_ms, created_at)
		VALUES (1, 1, 0, true, 1000, NOW())
	`)
	require.NoError(suite.T(), err)
}

// TestResetDatabase_Integration tests the database reset functionality
func (suite *ResetDBIntegrationTestSuite) TestResetDatabase_Integration() {
	ctx := context.Background()

	// Verify test data exists
	var userCount, questionCount, responseCount int64
	err := suite.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	require.NoError(suite.T(), err)
	err = suite.DB.QueryRow("SELECT COUNT(*) FROM questions").Scan(&questionCount)
	require.NoError(suite.T(), err)
	err = suite.DB.QueryRow("SELECT COUNT(*) FROM user_responses").Scan(&responseCount)
	require.NoError(suite.T(), err)

	assert.Greater(suite.T(), userCount, int64(0), "Should have test users")
	assert.Greater(suite.T(), questionCount, int64(0), "Should have test questions")
	assert.Greater(suite.T(), responseCount, int64(0), "Should have test responses")

	// Reset the database
	err = suite.UserService.ResetDatabase(ctx)
	require.NoError(suite.T(), err)

	// Verify all data was cleared
	err = suite.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	require.NoError(suite.T(), err)
	err = suite.DB.QueryRow("SELECT COUNT(*) FROM questions").Scan(&questionCount)
	require.NoError(suite.T(), err)
	err = suite.DB.QueryRow("SELECT COUNT(*) FROM user_responses").Scan(&responseCount)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), int64(0), userCount, "All users should be deleted")
	assert.Equal(suite.T(), int64(0), questionCount, "All questions should be deleted")
	assert.Equal(suite.T(), int64(0), responseCount, "All responses should be deleted")
}

// TestEnsureAdminUserExists_Integration tests admin user creation
func (suite *ResetDBIntegrationTestSuite) TestEnsureAdminUserExists_Integration() {
	ctx := context.Background()

	// Ensure admin user exists
	err := suite.UserService.EnsureAdminUserExists(ctx, "admin", "adminpass")
	require.NoError(suite.T(), err)

	// Verify admin user was created
	var adminUser *models.User
	adminUser, err = suite.UserService.GetUserByUsername(ctx, "admin")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), adminUser)
	assert.Equal(suite.T(), "admin", adminUser.Username)
	assert.Equal(suite.T(), "admin@example.com", adminUser.Email.String)
}

// TestResetDatabaseWithNoData_Integration tests reset with empty database
func (suite *ResetDBIntegrationTestSuite) TestResetDatabaseWithNoData_Integration() {
	ctx := context.Background()

	// Clean up all data first
	suite.cleanupDatabase()

	// Verify database is empty
	var userCount int64
	err := suite.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(0), userCount, "Database should be empty")

	// Reset empty database - should succeed
	err = suite.UserService.ResetDatabase(ctx)
	require.NoError(suite.T(), err)

	// Verify database is still empty
	err = suite.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(0), userCount, "Database should remain empty")
}

// TestResetDatabaseErrorHandling_Integration tests error handling scenarios
func (suite *ResetDBIntegrationTestSuite) TestResetDatabaseErrorHandling_Integration() {
	ctx := context.Background()

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	// Should handle cancelled context gracefully
	err := suite.UserService.ResetDatabase(cancelledCtx)
	// The error handling depends on the implementation, but it shouldn't panic
	suite.Logger.Info(ctx, "Reset with cancelled context handled", map[string]interface{}{
		"error": err,
	})
}

// TestResetDatabaseTimeout_Integration tests timeout handling
func (suite *ResetDBIntegrationTestSuite) TestResetDatabaseTimeout_Integration() {
	// Test with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Run reset with timeout - should handle gracefully
	err := suite.UserService.ResetDatabase(ctx)
	// Should either complete quickly or handle timeout gracefully
	suite.Logger.Info(context.Background(), "Reset with timeout handled", map[string]interface{}{
		"error": err,
	})
}

// TestResetDBCLI_Integration tests the CLI tool functionality
func (suite *ResetDBIntegrationTestSuite) TestResetDBCLI_Integration() {
	// Test that the main function exists and can be called
	// This tests the CLI tool structure without running the interactive part
	ctx := context.Background()

	// Test that the config can be loaded (this is what the CLI does first)
	cfg, err := config.NewConfig()
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), cfg, "Config should be loaded successfully")

	// Test that the database URL is set
	assert.NotEmpty(suite.T(), cfg.Database.URL, "Database URL should be set")

	suite.Logger.Info(ctx, "CLI functionality test completed", map[string]interface{}{
		"database_url": cfg.Database.URL,
	})
}

// TestResetDBCLIErrorHandling_Integration tests CLI error handling
func (suite *ResetDBIntegrationTestSuite) TestResetDBCLIErrorHandling_Integration() {
	// Test with invalid database URL
	originalDBURL := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", originalDBURL)

	// Set invalid database URL
	os.Setenv("DATABASE_URL", "postgres://invalid:invalid@localhost:9999/invalid?sslmode=disable")

	// Test that the config can be loaded (config loading doesn't validate DB connection)
	// The actual database connection test happens when trying to use the database
	cfg, err := config.NewConfig()
	// Config loading should succeed even with invalid DB URL
	assert.NoError(suite.T(), err, "Config should load successfully even with invalid database URL")
	assert.NotNil(suite.T(), cfg, "Config should not be nil")

	// Test that the database URL is set to the invalid value
	assert.Equal(suite.T(), "postgres://invalid:invalid@localhost:9999/invalid?sslmode=disable", cfg.Database.URL)

	suite.Logger.Info(context.Background(), "CLI error handling test completed", map[string]interface{}{
		"database_url": cfg.Database.URL,
	})
}

// TestResetDBCLIConfigError_Integration tests CLI configuration error handling
func (suite *ResetDBIntegrationTestSuite) TestResetDBCLIConfigError_Integration() {
	// Test with invalid config file
	originalConfigFile := os.Getenv("QUIZ_CONFIG_FILE")
	defer os.Setenv("QUIZ_CONFIG_FILE", originalConfigFile)

	// Set invalid config file
	os.Setenv("QUIZ_CONFIG_FILE", "/nonexistent/config.yaml")

	// Test that config loading fails with invalid config file
	// This tests the error handling without running the interactive CLI
	_, err := config.NewConfig()
	// Should fail due to invalid config file
	assert.Error(suite.T(), err, "Config should fail with invalid config file")

	suite.Logger.Info(context.Background(), "CLI config error handling test completed", map[string]interface{}{
		"error": err,
	})
}

// TestResetDBCLIDatabaseConnection_Integration tests database connection handling
func (suite *ResetDBIntegrationTestSuite) TestResetDBCLIDatabaseConnection_Integration() {
	ctx := context.Background()

	// Test that reset works with valid database connection
	// This test verifies that the reset service can handle database operations properly

	// Create some test data
	_, err := suite.DB.Exec(`
		INSERT INTO users (username, email, password_hash, preferred_language, current_level, created_at, updated_at)
		VALUES ('testuser_reset', 'test@example.com', '$2a$10$test', 'english', 'A1', NOW(), NOW())
	`)
	require.NoError(suite.T(), err)

	// Verify test data exists
	var userCount int64
	err = suite.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'testuser_reset'").Scan(&userCount)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1), userCount, "Test user should exist")

	// Reset the database
	err = suite.UserService.ResetDatabase(ctx)
	require.NoError(suite.T(), err)

	// Verify test data was removed
	err = suite.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'testuser_reset'").Scan(&userCount)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(0), userCount, "Test user should be removed")

	suite.Logger.Info(ctx, "Database connection reset test completed", map[string]interface{}{})
}

// TestResetDBCLIAdminUserRecreation_Integration tests admin user recreation
func (suite *ResetDBIntegrationTestSuite) TestResetDBCLIAdminUserRecreation_Integration() {
	ctx := context.Background()

	// Reset the database
	err := suite.UserService.ResetDatabase(ctx)
	require.NoError(suite.T(), err)

	// Ensure admin user exists after reset
	err = suite.UserService.EnsureAdminUserExists(ctx, "admin", "adminpass")
	require.NoError(suite.T(), err)

	// Verify admin user was created
	adminUser, err := suite.UserService.GetUserByUsername(ctx, "admin")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), adminUser)
	assert.Equal(suite.T(), "admin", adminUser.Username)
	assert.Equal(suite.T(), "admin@example.com", adminUser.Email.String)

	suite.Logger.Info(ctx, "Admin user recreation test completed", map[string]interface{}{})
}

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Run the tests
	code := m.Run()

	// Clean up
	os.Exit(code)
}
