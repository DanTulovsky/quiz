//go:build integration

package main

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// CLIWorkerIntegrationTestSuite provides comprehensive integration tests for the CLI worker tool
type CLIWorkerIntegrationTestSuite struct {
	suite.Suite
	DB           *sql.DB
	UserService  *services.UserService
	Logger       *observability.Logger
	Config       *config.Config
	TestUserID   int
	TestUsername string
}

func TestCLIWorkerIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CLIWorkerIntegrationTestSuite))
}

func (suite *CLIWorkerIntegrationTestSuite) SetupSuite() {
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

	// Initialize database connection (this will run migrations if needed)
	db, err := dbManager.InitDB(testDBURL)
	require.NoError(suite.T(), err)
	suite.DB = db

	// Initialize user service
	suite.UserService = services.NewUserServiceWithLogger(db, cfg, logger)
}

func (suite *CLIWorkerIntegrationTestSuite) TearDownSuite() {
	if suite.DB != nil {
		suite.DB.Close()
	}
}

func (suite *CLIWorkerIntegrationTestSuite) SetupTest() {
	suite.cleanupDatabase()
	suite.setupTestData()
}

func (suite *CLIWorkerIntegrationTestSuite) TearDownTest() {
	suite.cleanupDatabase()
}

func (suite *CLIWorkerIntegrationTestSuite) cleanupDatabase() {
	// Use the shared database cleanup function
	services.CleanupTestDatabase(suite.DB, suite.T())
}

func (suite *CLIWorkerIntegrationTestSuite) setupTestData() {
	// Create test user for CLI worker tests
	suite.TestUsername = "testuser_cli_worker"
	createdUser, err := suite.UserService.CreateUserWithPassword(context.Background(), suite.TestUsername, "testpass", "english", "A1")
	require.NoError(suite.T(), err)
	suite.TestUserID = createdUser.ID

	// Create some test questions
	_, err = suite.DB.Exec(`
		INSERT INTO questions (type, language, level, content, correct_answer, created_at)
		VALUES
			('vocabulary', 'english', 'A1', '{"question_text": "Test question 1", "options": ["a", "b", "c", "d"]}', 0, NOW()),
			('vocabulary', 'english', 'A1', '{"question_text": "Test question 2", "options": ["a", "b", "c", "d"]}', 0, NOW())
	`)
	require.NoError(suite.T(), err)
}

// TestCLIWorkerArgumentValidation_Integration tests argument validation functions
func (suite *CLIWorkerIntegrationTestSuite) TestCLIWorkerArgumentValidation_Integration() {
	// Test isValidLevel function
	validLevels := []string{"A1", "A2", "B1", "B2", "C1", "C2"}

	for _, level := range validLevels {
		assert.True(suite.T(), isValidLevel(level, validLevels), "Level %s should be valid", level)
	}

	assert.False(suite.T(), isValidLevel("INVALID", validLevels), "Invalid level should be rejected")
	assert.False(suite.T(), isValidLevel("", validLevels), "Empty level should be rejected")

	// Test isValidLanguage function
	validLanguages := []string{"english", "spanish", "french", "italian"}

	for _, lang := range validLanguages {
		assert.True(suite.T(), isValidLanguage(lang, validLanguages), "Language %s should be valid", lang)
	}

	assert.False(suite.T(), isValidLanguage("invalid_lang", validLanguages), "Invalid language should be rejected")
	assert.False(suite.T(), isValidLanguage("", validLanguages), "Empty language should be rejected")
}

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Run the tests
	code := m.Run()

	// Clean up
	os.Exit(code)
}
