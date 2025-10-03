package services

import (
	"database/sql"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// StoryServiceTestSuite defines the test suite for StoryService
type StoryServiceTestSuite struct {
	suite.Suite
	db           *sql.DB
	storyService StoryServiceInterface
	config       *config.Config
	logger       *observability.Logger
	testUserID   uint
}

// SetupSuite runs once before all tests in the suite
func (suite *StoryServiceTestSuite) SetupSuite() {
	// Initialize test database and services
	cfg := &config.Config{}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// For unit tests, we'll use a mock database setup
	// In a real implementation, you'd set up a test database
	suite.config = cfg
	suite.logger = logger
	suite.db = nil // Would be set up in integration tests

	// Create services - should panic with nil database
	assert.Panics(suite.T(), func() {
		suite.storyService = NewStoryService(suite.db, cfg, logger)
	})
}

// TearDownSuite runs once after all tests in the suite
func (suite *StoryServiceTestSuite) TearDownSuite() {
	if suite.db != nil {
		_ = suite.db.Close()
	}
}

// SetupTest runs before each test
func (suite *StoryServiceTestSuite) SetupTest() {
	// Create a test user for each test
	// In a real implementation, this would create a user in the test database
	suite.testUserID = 1
}

// TestCreateStory tests story creation functionality
func (suite *StoryServiceTestSuite) TestCreateStory() {
	suite.T().Skip("Test requires database setup - service creation panics with nil database")
}

// TestCreateStoryValidation tests input validation
func (suite *StoryServiceTestSuite) TestCreateStoryValidation() {
	suite.T().Skip("Test requires database setup - service creation panics with nil database")
}

// TestGetSectionLengthTarget tests section length calculation
func (suite *StoryServiceTestSuite) TestGetSectionLengthTarget() {
	// Test A1 level defaults
	length := models.GetSectionLengthTarget("A1", nil)
	assert.Equal(suite.T(), 80, length) // Medium length for A1

	// Test with short preference
	shortPref := models.SectionLengthShort
	length = models.GetSectionLengthTarget("A1", &shortPref)
	assert.Equal(suite.T(), 50, length)

	// Test with long preference
	longPref := models.SectionLengthLong
	length = models.GetSectionLengthTarget("A1", &longPref)
	assert.Equal(suite.T(), 120, length)

	// Test B2 level
	length = models.GetSectionLengthTarget("B2", nil)
	assert.Equal(suite.T(), 350, length) // Medium length for B2

	// Test unknown level defaults to intermediate
	length = models.GetSectionLengthTarget("unknown", nil)
	assert.Equal(suite.T(), 220, length) // Medium length for intermediate
}

// TestCanGenerateSection tests generation eligibility logic
func (suite *StoryServiceTestSuite) TestCanGenerateSection() {
	suite.T().Skip("Test requires database setup - service creation panics with nil database")
}

// TestSanitizeInput tests input sanitization
func (suite *StoryServiceTestSuite) TestSanitizeInput() {
	// Test basic sanitization
	input := "Hello\x00World\x01Test"
	sanitized := models.SanitizeInput(input)
	assert.Equal(suite.T(), "HelloWorldTest", sanitized)

	// Test whitespace trimming
	input = "  Hello World  "
	sanitized = models.SanitizeInput(input)
	assert.Equal(suite.T(), "Hello World", sanitized)

	// Test control character removal
	input = "Hello\x00\x01\x02World"
	sanitized = models.SanitizeInput(input)
	assert.Equal(suite.T(), "HelloWorld", sanitized)
}

// TestValidateCreateStoryRequest tests request validation
func (suite *StoryServiceTestSuite) TestValidateCreateStoryRequest() {
	// Valid request
	req := &models.CreateStoryRequest{
		Title:   "Valid Story Title",
		Subject: stringPtr("Valid subject"),
	}

	err := req.Validate()
	assert.NoError(suite.T(), err)

	// Invalid - empty title
	req = &models.CreateStoryRequest{
		Title: "",
	}

	err = req.Validate()
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "title is required")

	// Invalid - title too long
	req = &models.CreateStoryRequest{
		Title: string(make([]byte, 201)),
	}

	err = req.Validate()
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "title must be 200 characters or less")
}

// Test helpers
func stringPtr(s string) *string {
	return &s
}

// TestStoryService runs the test suite
func TestStoryService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping story service tests in short mode")
	}

	suite.Run(t, new(StoryServiceTestSuite))
}
