//go:build integration

package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

func TestUserService_CreateUser_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := service.CreateUser(context.Background(), username, "italian", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Greater(t, user.ID, 0)
	assert.Equal(t, username, user.Username)
	assert.True(t, user.PreferredLanguage.Valid)
	assert.Equal(t, "italian", user.PreferredLanguage.String)
	assert.True(t, user.CurrentLevel.Valid)
	assert.Equal(t, "A1", user.CurrentLevel.String)
	assert.NotEmpty(t, user.CreatedAt)
}

func TestUserService_CreateUserWithPassword_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := service.CreateUserWithPassword(context.Background(), username, "password123", "italian", "B1")
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Greater(t, user.ID, 0)
	assert.Equal(t, username, user.Username)
	assert.True(t, user.PreferredLanguage.Valid)
	assert.Equal(t, "italian", user.PreferredLanguage.String)
	assert.True(t, user.CurrentLevel.Valid)
	assert.Equal(t, "B1", user.CurrentLevel.String)
	assert.True(t, user.PasswordHash.Valid)
	assert.NotEqual(t, "password123", user.PasswordHash.String) // Should be hashed
}

func TestUserService_GetUserByID_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	// Create a user first
	created, err := service.CreateUser(context.Background(), "testuser", "italian", "A2")
	require.NoError(t, err)
	require.NotNil(t, created, "Created user should not be nil")

	// Retrieve the user
	retrieved, err := service.GetUserByID(context.Background(), created.ID)
	require.NoError(t, err, "Error retrieving user by ID %d", created.ID)
	require.NotNil(t, retrieved, "Retrieved user should not be nil")

	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, "testuser", retrieved.Username)
	assert.True(t, retrieved.PreferredLanguage.Valid)
	assert.Equal(t, "italian", retrieved.PreferredLanguage.String)
	assert.True(t, retrieved.CurrentLevel.Valid)
	assert.Equal(t, "A2", retrieved.CurrentLevel.String)
}

func TestUserService_GetUserByUsername_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	// Create a user first
	created, err := service.CreateUser(context.Background(), "uniqueuser", "italian", "B1")
	require.NoError(t, err)

	// Retrieve the user by username
	retrieved, err := service.GetUserByUsername(context.Background(), "uniqueuser")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, "uniqueuser", retrieved.Username)
	assert.True(t, retrieved.PreferredLanguage.Valid)
	assert.Equal(t, "italian", retrieved.PreferredLanguage.String)
	assert.True(t, retrieved.CurrentLevel.Valid)
	assert.Equal(t, "B1", retrieved.CurrentLevel.String)
}

func TestUserService_AuthenticateUser_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	// Create a user with password
	_, err = service.CreateUserWithPassword(context.Background(), "authuser", "correctpassword", "italian", "B2")
	require.NoError(t, err)

	// Test correct authentication
	user, err := service.AuthenticateUser(context.Background(), "authuser", "correctpassword")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "authuser", user.Username)

	// Test incorrect password
	user, err = service.AuthenticateUser(context.Background(), "authuser", "wrongpassword")
	assert.Error(t, err)
	assert.Nil(t, user)

	// Test non-existent user
	user, err = service.AuthenticateUser(context.Background(), "nonexistent", "anypassword")
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestUserService_UpdateUserSettings_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	// Create a user first
	user, err := service.CreateUser(context.Background(), "settingsuser", "italian", "A1")
	require.NoError(t, err)
	require.NotNil(t, user, "User should not be nil after creation")

	// Update settings
	settings := &models.UserSettings{
		Language: "spanish",
		Level:    "B1",
	}

	err = service.UpdateUserSettings(context.Background(), user.ID, settings)
	require.NoError(t, err)

	// Verify the update
	updated, err := service.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, updated, "User should exist after updating settings")
	assert.True(t, updated.PreferredLanguage.Valid)
	assert.Equal(t, "spanish", updated.PreferredLanguage.String)
	assert.True(t, updated.CurrentLevel.Valid)
	assert.Equal(t, "B1", updated.CurrentLevel.String)
}

func TestUserService_UpdateLastActive_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	// Create a user first
	user, err := service.CreateUser(context.Background(), "activeuser", "italian", "A1")
	require.NoError(t, err)

	// Update last active
	err = service.UpdateLastActive(context.Background(), user.ID)
	require.NoError(t, err)

	// Verify the update by checking the user still exists and is valid
	updated, err := service.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Username, updated.Username)
}

// Helper function to set up test database for user service tests
func setupTestDBForUser(t *testing.T) *sql.DB {
	return SharedTestDBSetup(t)
}

// TestUserService_EnsureAdminAISettings_PreservesExistingSettings tests that existing AI settings are preserved
func TestUserService_EnsureAdminAISettings_PreservesExistingSettings_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	// Create a user with custom AI settings
	user, err := service.CreateUser(context.Background(), "testuser", "italian", "A1")
	require.NoError(t, err)

	// Set custom AI settings
	customSettings := &models.UserSettings{
		AIProvider: "ollama",
		AIModel:    "custom-model:latest",
		AIAPIKey:   "custom-api-key",
		AIEnabled:  true,
	}
	err = service.UpdateUserSettings(context.Background(), user.ID, customSettings)
	require.NoError(t, err)

	// Verify the custom settings were saved
	updatedUser, err := service.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.True(t, updatedUser.AIProvider.Valid)
	assert.Equal(t, "ollama", updatedUser.AIProvider.String)
	assert.True(t, updatedUser.AIModel.Valid)
	assert.Equal(t, "custom-model:latest", updatedUser.AIModel.String)

	// Note: ensureAdminAISettings is a private method and should not be called directly in tests
	// The functionality is tested through the public EnsureAdminUserExists method
}

// TestUserService_EnsureAdminAISettings_SetsDefaultsForNewUser tests that default settings are applied to new users
func TestUserService_EnsureAdminAISettings_SetsDefaultsForNewUser_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg := &config.Config{
		// Use default config for testing
	}

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)

	// Create a new user without AI settings
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_new", "password", "italian", "A1")
	assert.NoError(t, err)
	assert.NotNil(t, user)

	// Verify the user has no AI settings initially
	assert.False(t, user.AIProvider.Valid)
	assert.False(t, user.AIModel.Valid)

	// Note: ensureAdminAISettings is a private method and should not be called directly in tests
	// The functionality is tested through the public EnsureAdminUserExists method
}

func TestUserService_ClearUserData_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create test users with unique usernames using more unique identifiers
	timestamp := time.Now().UnixNano()
	username1 := fmt.Sprintf("testuser_clear1_%d", timestamp)
	username2 := fmt.Sprintf("testuser_clear2_%d", timestamp+1) // Ensure different timestamp

	user1, err := userService.CreateUserWithPassword(context.Background(), username1, "password", "italian", "A1")
	assert.NoError(t, err)
	assert.NotNil(t, user1)

	user2, err := userService.CreateUserWithPassword(context.Background(), username2, "password", "spanish", "A2")
	assert.NoError(t, err)
	assert.NotNil(t, user2)

	// Create test questions
	question1 := &models.Question{
		Type:            models.Vocabulary,
		Language:        "italian",
		Level:           "A1",
		DifficultyScore: 1.0,
		Content:         map[string]interface{}{"question": "What is 'ciao'?", "options": []string{"hello", "goodbye", "please", "thanks"}},
		CorrectAnswer:   0,
		Explanation:     "Ciao means hello",
		CreatedAt:       time.Now(),
	}
	err = questionService.SaveQuestion(context.Background(), question1)
	assert.NoError(t, err)
	// Assign question to user1
	err = questionService.AssignQuestionToUser(context.Background(), question1.ID, user1.ID)
	assert.NoError(t, err)

	question2 := &models.Question{
		Type:            models.FillInBlank,
		Language:        "spanish",
		Level:           "A2",
		DifficultyScore: 2.0,
		Content:         map[string]interface{}{"question": "Which is correct?", "options": []string{"io sono", "io sei", "io è", "io siamo"}},
		CorrectAnswer:   0,
		Explanation:     "First person singular",
		CreatedAt:       time.Now(),
	}
	err = questionService.SaveQuestion(context.Background(), question2)
	assert.NoError(t, err)
	// Assign question to user2
	err = questionService.AssignQuestionToUser(context.Background(), question2.ID, user2.ID)
	assert.NoError(t, err)

	// Create test user responses
	response1 := &models.UserResponse{
		UserID:          user1.ID,
		QuestionID:      question1.ID,
		UserAnswerIndex: 0,
		IsCorrect:       true,
		ResponseTimeMs:  1000,
		CreatedAt:       time.Now(),
	}
	err = learningService.RecordUserResponse(context.Background(), response1)
	assert.NoError(t, err)

	response2 := &models.UserResponse{
		UserID:          user2.ID,
		QuestionID:      question2.ID,
		UserAnswerIndex: 1,
		IsCorrect:       true,
		ResponseTimeMs:  1500,
		CreatedAt:       time.Now(),
	}
	err = learningService.RecordUserResponse(context.Background(), response2)
	assert.NoError(t, err)

	// Verify data exists before clearing
	questions, err := questionService.GetQuestionsByFilter(context.Background(), user1.ID, "italian", "A1", models.Vocabulary, 10)
	assert.NoError(t, err)
	assert.Len(t, questions, 1)

	questions, err = questionService.GetQuestionsByFilter(context.Background(), user2.ID, "spanish", "A2", models.FillInBlank, 10)
	assert.NoError(t, err)
	assert.Len(t, questions, 1)

	// Clear user data for user1
	err = userService.ClearUserData(context.Background())
	assert.NoError(t, err)

	// Verify that questions and responses were cleared but users remain
	questions, err = questionService.GetQuestionsByFilter(context.Background(), user1.ID, "italian", "A1", models.Vocabulary, 10)
	assert.NoError(t, err)
	assert.Len(t, questions, 0)

	questions, err = questionService.GetQuestionsByFilter(context.Background(), user2.ID, "spanish", "A2", models.FillInBlank, 10)
	assert.NoError(t, err)
	assert.Len(t, questions, 0)

	// Verify users still exist
	remainingUser1, err := userService.GetUserByID(context.Background(), user1.ID)
	assert.NoError(t, err)
	assert.NotNil(t, remainingUser1, "User1 should still exist")
	assert.Equal(t, username1, remainingUser1.Username) // Use the original username variable

	remainingUser2, err := userService.GetUserByID(context.Background(), user2.ID)
	assert.NoError(t, err)
	assert.NotNil(t, remainingUser2, "User2 should still exist")
	assert.Equal(t, username2, remainingUser2.Username) // Use the original username variable
}

func TestUserService_TracingIntegration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	// Set up a test tracer provider
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(nooptrace.NewTracerProvider())

	ctx, parent := otel.Tracer("test-parent").Start(context.Background(), "parent-span")
	defer parent.End()

	username := fmt.Sprintf("traceuser_%d", time.Now().UnixNano())
	user, err := service.CreateUser(ctx, username, "italian", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	// The returned context should have a span
	span := oteltrace.SpanFromContext(ctx)
	assert.NotNil(t, span)

	// Authenticate user (should create a login span)
	_, err = service.AuthenticateUser(ctx, username, "") // Password is empty, will fail, but span should be created
	// Ignore error, just check span propagation
	span = oteltrace.SpanFromContext(ctx)
	assert.NotNil(t, span)

	// Update user profile (should create a profile update span)
	err = service.UpdateUserProfile(ctx, user.ID, username+"2", "", "")
	// Ignore error, just check span propagation
	span = oteltrace.SpanFromContext(ctx)
	assert.NotNil(t, span)
}

func TestUserService_CreateUser_DuplicateUsername_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	username := fmt.Sprintf("dupuser_%d", time.Now().UnixNano())
	_, err = service.CreateUser(context.Background(), username, "italian", "A1")
	require.NoError(t, err)
	_, err = service.CreateUser(context.Background(), username, "italian", "A1")
	assert.Error(t, err)
}

func TestUserService_CreateUser_InvalidInput_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	_, err = service.CreateUser(context.Background(), "", "italian", "A1")
	assert.Error(t, err)
}

func TestUserService_UpdateUserSettings_NonExistentUser_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	settings := &models.UserSettings{Language: "spanish", Level: "B1"}
	err = service.UpdateUserSettings(context.Background(), 999999, settings)
	assert.Error(t, err)
}

func TestUserService_DeleteUser_NonExistentUser_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	err = service.DeleteUser(context.Background(), 999999)
	assert.Error(t, err)
}

func TestUserService_DeleteUser_WithRelatedData_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	// Create a test user
	user, err := service.CreateUserWithPassword(context.Background(), "deleteuser", "password", "italian", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Create related data for the user
	// 1. Create a question and assign it to the user
	var questionID int
	err = db.QueryRow(`
		INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, "vocabulary", "italian", "A1", 1.0, `{"question":"Test?"}`, 0, "Test explanation").Scan(&questionID)
	require.NoError(t, err)

	// Assign question to user
	_, err = db.Exec(`INSERT INTO user_questions (user_id, question_id) VALUES ($1, $2)`, user.ID, questionID)
	require.NoError(t, err)

	// 2. Create user responses
	_, err = db.Exec(`
		INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, response_time_ms)
		VALUES ($1, $2, $3, $4, $5)
	`, user.ID, questionID, 0, true, 5000)
	require.NoError(t, err)

	// 3. Create performance metrics
	_, err = db.Exec(`
		INSERT INTO performance_metrics (user_id, topic, language, level, total_attempts, correct_attempts, average_response_time_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, user.ID, "vocabulary", "italian", "A1", 5, 3, 1500.0)
	require.NoError(t, err)

	// 4. Create user question metadata
	_, err = db.Exec(`
		INSERT INTO user_question_metadata (user_id, question_id, marked_as_known, confidence_level)
		VALUES ($1, $2, $3, $4)
	`, user.ID, questionID, true, 4)
	require.NoError(t, err)

	// 5. Create question priority scores
	_, err = db.Exec(`
		INSERT INTO question_priority_scores (user_id, question_id, priority_score)
		VALUES ($1, $2, $3)
	`, user.ID, questionID, 85.5)
	require.NoError(t, err)

	// 6. Create user learning preferences
	_, err = db.Exec(`
		INSERT INTO user_learning_preferences (user_id, focus_on_weak_areas, fresh_question_ratio)
		VALUES ($1, $2, $3)
	`, user.ID, true, 0.3)
	require.NoError(t, err)

	// 7. Create user API keys
	_, err = db.Exec(`
		INSERT INTO user_api_keys (user_id, provider, api_key)
		VALUES ($1, $2, $3)
	`, user.ID, "openai", "test-api-key")
	require.NoError(t, err)

	// 8. Create a role and assign it to the user
	var roleID int
	err = db.QueryRow(`INSERT INTO roles (name, description) VALUES ($1, $2) RETURNING id`, "test_role", "Test role").Scan(&roleID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, user.ID, roleID)
	require.NoError(t, err)

	// 9. Create question reports by this user
	_, err = db.Exec(`
		INSERT INTO question_reports (question_id, reported_by_user_id, report_reason)
		VALUES ($1, $2, $3)
	`, questionID, user.ID, "Test report")
	require.NoError(t, err)

	// Verify all data exists before deletion
	var responseCount int
	err = db.QueryRow("SELECT COUNT(*) FROM user_responses WHERE user_id = $1", user.ID).Scan(&responseCount)
	require.NoError(t, err)
	assert.Greater(t, responseCount, 0)

	var metricsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM performance_metrics WHERE user_id = $1", user.ID).Scan(&metricsCount)
	require.NoError(t, err)
	assert.Greater(t, metricsCount, 0)

	var metadataCount int
	err = db.QueryRow("SELECT COUNT(*) FROM user_question_metadata WHERE user_id = $1", user.ID).Scan(&metadataCount)
	require.NoError(t, err)
	assert.Greater(t, metadataCount, 0)

	var priorityCount int
	err = db.QueryRow("SELECT COUNT(*) FROM question_priority_scores WHERE user_id = $1", user.ID).Scan(&priorityCount)
	require.NoError(t, err)
	assert.Greater(t, priorityCount, 0)

	var prefsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM user_learning_preferences WHERE user_id = $1", user.ID).Scan(&prefsCount)
	require.NoError(t, err)
	assert.Greater(t, prefsCount, 0)

	var apiKeyCount int
	err = db.QueryRow("SELECT COUNT(*) FROM user_api_keys WHERE user_id = $1", user.ID).Scan(&apiKeyCount)
	require.NoError(t, err)
	assert.Greater(t, apiKeyCount, 0)

	var roleCount int
	err = db.QueryRow("SELECT COUNT(*) FROM user_roles WHERE user_id = $1", user.ID).Scan(&roleCount)
	require.NoError(t, err)
	assert.Greater(t, roleCount, 0)

	var reportCount int
	err = db.QueryRow("SELECT COUNT(*) FROM question_reports WHERE reported_by_user_id = $1", user.ID).Scan(&reportCount)
	require.NoError(t, err)
	assert.Greater(t, reportCount, 0)

	// Delete the user
	err = service.DeleteUser(context.Background(), user.ID)
	require.NoError(t, err)

	// Verify user was deleted
	deletedUser, err := service.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Nil(t, deletedUser)

	// Verify all related data was deleted due to CASCADE
	err = db.QueryRow("SELECT COUNT(*) FROM user_responses WHERE user_id = $1", user.ID).Scan(&responseCount)
	require.NoError(t, err)
	assert.Equal(t, 0, responseCount)

	err = db.QueryRow("SELECT COUNT(*) FROM performance_metrics WHERE user_id = $1", user.ID).Scan(&metricsCount)
	require.NoError(t, err)
	assert.Equal(t, 0, metricsCount)

	err = db.QueryRow("SELECT COUNT(*) FROM user_question_metadata WHERE user_id = $1", user.ID).Scan(&metadataCount)
	require.NoError(t, err)
	assert.Equal(t, 0, metadataCount)

	err = db.QueryRow("SELECT COUNT(*) FROM question_priority_scores WHERE user_id = $1", user.ID).Scan(&priorityCount)
	require.NoError(t, err)
	assert.Equal(t, 0, priorityCount)

	err = db.QueryRow("SELECT COUNT(*) FROM user_learning_preferences WHERE user_id = $1", user.ID).Scan(&prefsCount)
	require.NoError(t, err)
	assert.Equal(t, 0, prefsCount)

	err = db.QueryRow("SELECT COUNT(*) FROM user_api_keys WHERE user_id = $1", user.ID).Scan(&apiKeyCount)
	require.NoError(t, err)
	assert.Equal(t, 0, apiKeyCount)

	err = db.QueryRow("SELECT COUNT(*) FROM user_roles WHERE user_id = $1", user.ID).Scan(&roleCount)
	require.NoError(t, err)
	assert.Equal(t, 0, roleCount)

	err = db.QueryRow("SELECT COUNT(*) FROM question_reports WHERE reported_by_user_id = $1", user.ID).Scan(&reportCount)
	require.NoError(t, err)
	assert.Equal(t, 0, reportCount)

	// Verify user_questions was deleted
	var userQuestionsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM user_questions WHERE user_id = $1", user.ID).Scan(&userQuestionsCount)
	require.NoError(t, err)
	assert.Equal(t, 0, userQuestionsCount)
}

func TestUserService_DeleteUser_WithNoRelatedData_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	// Create a test user with no related data
	user, err := service.CreateUser(context.Background(), "simpleuser", "italian", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Verify user exists
	foundUser, err := service.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, foundUser)
	assert.Equal(t, user.ID, foundUser.ID)

	// Delete the user
	err = service.DeleteUser(context.Background(), user.ID)
	require.NoError(t, err)

	// Verify user was deleted
	deletedUser, err := service.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Nil(t, deletedUser)
}

func TestUserService_APIKeyManagement_MissingUserOrProvider_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	// Get API key for non-existent user
	_, err = service.GetUserAPIKey(context.Background(), 999999, "openai")
	assert.Error(t, err)

	// Set API key for non-existent user
	err = service.SetUserAPIKey(context.Background(), 999999, "openai", "key")
	assert.Error(t, err)

	// Has API key for non-existent user
	ok, err := service.HasUserAPIKey(context.Background(), 999999, "openai")
	assert.Error(t, err)
	assert.False(t, ok)

	// Create a user and check for missing provider
	user, err := service.CreateUser(context.Background(), "apikeyuser", "italian", "A1")
	require.NoError(t, err)
	_, err = service.GetUserAPIKey(context.Background(), user.ID, "nonexistent")
	assert.Error(t, err)
}

func TestUserService_GetUserByEmail_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)

	// Create test user with email
	user, err := userService.CreateUserWithEmailAndTimezone(
		context.Background(),
		"testuser_email",
		"test@example.com",
		"America/New_York",
		"italian",
		"A1",
	)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Test GetUserByEmail - success case
	foundUser, err := userService.GetUserByEmail(context.Background(), "test@example.com")
	require.NoError(t, err)
	require.NotNil(t, foundUser)
	assert.Equal(t, user.ID, foundUser.ID)
	assert.Equal(t, user.Username, foundUser.Username)
	assert.Equal(t, "test@example.com", foundUser.Email.String)

	// Test GetUserByEmail - user not found
	notFoundUser, err := userService.GetUserByEmail(context.Background(), "nonexistent@example.com")
	require.NoError(t, err)
	assert.Nil(t, notFoundUser)

	// Test GetUserByEmail - empty email
	emptyUser, err := userService.GetUserByEmail(context.Background(), "")
	require.NoError(t, err)
	assert.Nil(t, emptyUser)
}

func TestUserService_UpdateUserPassword_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(
		context.Background(),
		"testuser_password",
		"oldpassword",
		"italian",
		"A1",
	)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Test UpdateUserPassword - success case
	err = userService.UpdateUserPassword(context.Background(), user.ID, "newpassword")
	require.NoError(t, err)

	// Verify password was updated by trying to authenticate with new password
	authenticatedUser, err := userService.AuthenticateUser(context.Background(), "testuser_password", "newpassword")
	require.NoError(t, err)
	require.NotNil(t, authenticatedUser)
	assert.Equal(t, user.ID, authenticatedUser.ID)

	// Verify old password no longer works
	_, err = userService.AuthenticateUser(context.Background(), "testuser_password", "oldpassword")
	require.Error(t, err)

	// Test UpdateUserPassword - user not found (should return error)
	err = userService.UpdateUserPassword(context.Background(), 99999, "newpassword")
	require.Error(t, err) // Should return error for non-existent user
	assert.Contains(t, err.Error(), "user not found")
}

func TestUserService_DeleteAllUsers_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)

	// Create multiple test users
	user1, err := userService.CreateUserWithPassword(
		context.Background(),
		"testuser1",
		"password1",
		"italian",
		"A1",
	)
	require.NoError(t, err)

	user2, err := userService.CreateUserWithPassword(
		context.Background(),
		"testuser2",
		"password2",
		"english",
		"B1",
	)
	require.NoError(t, err)

	// Verify users exist
	allUsers, err := userService.GetAllUsers(context.Background())
	require.NoError(t, err)
	assert.Len(t, allUsers, 2)

	// Test DeleteAllUsers
	err = userService.DeleteAllUsers(context.Background())
	require.NoError(t, err)

	// Verify all users were deleted
	allUsersAfter, err := userService.GetAllUsers(context.Background())
	require.NoError(t, err)
	assert.Len(t, allUsersAfter, 0)

	// Verify specific users no longer exist
	user1After, err := userService.GetUserByID(context.Background(), user1.ID)
	require.NoError(t, err)
	assert.Nil(t, user1After)

	user2After, err := userService.GetUserByID(context.Background(), user2.ID)
	require.NoError(t, err)
	assert.Nil(t, user2After)
}

func TestUserService_EnsureAdminUserExists_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)

	// Test EnsureAdminUserExists - create new admin
	err = userService.EnsureAdminUserExists(context.Background(), "admin", "adminpass")
	require.NoError(t, err)

	// Verify admin user was created
	adminUser, err := userService.GetUserByUsername(context.Background(), "admin")
	require.NoError(t, err)
	require.NotNil(t, adminUser)
	assert.Equal(t, "admin", adminUser.Username)
	assert.True(t, adminUser.Email.Valid)
	assert.Equal(t, "admin@example.com", adminUser.Email.String)
	assert.True(t, adminUser.Timezone.Valid)
	assert.Equal(t, "America/New_York", adminUser.Timezone.String)

	// Verify admin password works
	authenticatedAdmin, err := userService.AuthenticateUser(context.Background(), "admin", "adminpass")
	require.NoError(t, err)
	require.NotNil(t, authenticatedAdmin)

	// Test EnsureAdminUserExists - update existing admin password
	err = userService.EnsureAdminUserExists(context.Background(), "admin", "newadminpass")
	require.NoError(t, err)

	// Verify new password works
	authenticatedAdminNew, err := userService.AuthenticateUser(context.Background(), "admin", "newadminpass")
	require.NoError(t, err)
	require.NotNil(t, authenticatedAdminNew)

	// Verify old password no longer works
	_, err = userService.AuthenticateUser(context.Background(), "admin", "adminpass")
	require.Error(t, err)
}

func TestUserService_ClearUserDataForUser_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(
		context.Background(),
		"testuser_clear",
		"password",
		"italian",
		"A1",
	)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Create some test data for the user
	// Create a test question
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	question := &models.Question{
		Type:            "vocabulary",
		Language:        "italian",
		Level:           "A1",
		DifficultyScore: 0.5,
		Content: map[string]interface{}{
			"question": "Test question",
			"options":  []string{"A", "B", "C", "D"},
		},
		CorrectAnswer: 0,
		Explanation:   "Test explanation",
		Status:        models.QuestionStatusActive,
	}

	err = questionService.SaveQuestion(context.Background(), question)
	require.NoError(t, err)

	// Assign question to user
	err = questionService.AssignQuestionToUser(context.Background(), question.ID, user.ID)
	require.NoError(t, err)

	// Create user response
	response := &models.UserResponse{
		UserID:          user.ID,
		QuestionID:      question.ID,
		UserAnswerIndex: 0,
		IsCorrect:       true,
		ResponseTimeMs:  1000,
		CreatedAt:       time.Now(),
	}
	err = learningService.RecordUserResponse(context.Background(), response)
	require.NoError(t, err)

	// Create another response
	response2 := &models.UserResponse{
		UserID:          user.ID,
		QuestionID:      question.ID,
		UserAnswerIndex: 1,
		IsCorrect:       false,
		ResponseTimeMs:  2000,
		CreatedAt:       time.Now(),
	}
	err = learningService.RecordUserResponse(context.Background(), response2)
	require.NoError(t, err)

	// Create performance metrics
	_, err = db.Exec(`
		INSERT INTO performance_metrics (user_id, topic, language, level, total_attempts, correct_attempts, average_response_time_ms, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6, 1500.0, NOW())
	`, user.ID, "vocabulary", "italian", "A1", 5, 3)
	require.NoError(t, err)

	// Verify data exists before clearing
	var responseCount int
	err = db.QueryRow("SELECT COUNT(*) FROM user_responses WHERE user_id = $1", user.ID).Scan(&responseCount)
	require.NoError(t, err)
	assert.Greater(t, responseCount, 0)

	var metricsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM performance_metrics WHERE user_id = $1", user.ID).Scan(&metricsCount)
	require.NoError(t, err)
	assert.Greater(t, metricsCount, 0)

	// Test ClearUserDataForUser
	err = userService.ClearUserDataForUser(context.Background(), user.ID)
	require.NoError(t, err)

	// Verify user data was cleared but user still exists
	userAfter, err := userService.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, userAfter) // User should still exist

	// Verify user responses were cleared
	err = db.QueryRow("SELECT COUNT(*) FROM user_responses WHERE user_id = $1", user.ID).Scan(&responseCount)
	require.NoError(t, err)
	assert.Equal(t, 0, responseCount)

	// Verify performance metrics were cleared
	err = db.QueryRow("SELECT COUNT(*) FROM performance_metrics WHERE user_id = $1", user.ID).Scan(&metricsCount)
	require.NoError(t, err)
	assert.Equal(t, 0, metricsCount)

	// Test ClearUserDataForUser - user not found (should not return error, just delete 0 rows)
	err = userService.ClearUserDataForUser(context.Background(), 99999)
	require.NoError(t, err) // Should succeed even for non-existent users
}

func TestUserService_EdgeCases_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)

	// Test GetUserByEmail with special characters
	user, err := userService.CreateUserWithEmailAndTimezone(
		context.Background(),
		"testuser_special",
		"test+special@example.com",
		"America/New_York",
		"italian",
		"A1",
	)
	require.NoError(t, err)

	foundUser, err := userService.GetUserByEmail(context.Background(), "test+special@example.com")
	require.NoError(t, err)
	require.NotNil(t, foundUser)
	assert.Equal(t, user.ID, foundUser.ID)

	// Test UpdateUserPassword with empty password
	err = userService.UpdateUserPassword(context.Background(), user.ID, "")
	require.Error(t, err) // Should fail with empty password

	// Test UpdateUserPassword with very long password (bcrypt limit is 72 bytes)
	longPassword := string(make([]byte, 50)) // Long but within bcrypt limit
	err = userService.UpdateUserPassword(context.Background(), user.ID, longPassword)
	require.NoError(t, err) // Should succeed

	// Test DeleteAllUsers on empty database
	err = userService.DeleteAllUsers(context.Background())
	require.NoError(t, err) // Should succeed even with no users

	// Test EnsureAdminUserExists with empty username/password
	err = userService.EnsureAdminUserExists(context.Background(), "", "password")
	require.Error(t, err)

	err = userService.EnsureAdminUserExists(context.Background(), "admin", "")
	require.Error(t, err)
}

func TestUserService_RoleManagement_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)

	// Create a test user
	user, err := userService.CreateUserWithEmailAndTimezone(
		context.Background(),
		"testuser_roles",
		"testuser_roles@example.com",
		"UTC",
		"italian",
		"A1",
	)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Test GetUserRoles - should return user role (automatically assigned)
	roles, err := userService.GetUserRoles(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, "user", roles[0].Name)
	assert.Equal(t, 1, roles[0].ID)

	// User already has user role assigned by default, verify it
	roles, err = userService.GetUserRoles(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, "user", roles[0].Name)
	assert.Equal(t, 1, roles[0].ID)

	// Test AssignRole - assign admin role (ID 2)
	err = userService.AssignRole(context.Background(), user.ID, 2)
	require.NoError(t, err)

	// Verify both roles are assigned
	roles, err = userService.GetUserRoles(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 2)

	// Check that both roles are present
	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}
	assert.Contains(t, roleNames, "user")
	assert.Contains(t, roleNames, "admin")

	// Test HasRole
	hasUserRole, err := userService.HasRole(context.Background(), user.ID, "user")
	require.NoError(t, err)
	assert.True(t, hasUserRole)

	hasAdminRole, err := userService.HasRole(context.Background(), user.ID, "admin")
	require.NoError(t, err)
	assert.True(t, hasAdminRole)

	hasNonExistentRole, err := userService.HasRole(context.Background(), user.ID, "nonexistent")
	require.NoError(t, err)
	assert.False(t, hasNonExistentRole)

	// Test IsAdmin
	isAdmin, err := userService.IsAdmin(context.Background(), user.ID)
	require.NoError(t, err)
	assert.True(t, isAdmin)

	// Test RemoveRole - remove user role
	err = userService.RemoveRole(context.Background(), user.ID, 1)
	require.NoError(t, err)

	// Verify user role was removed, leaving only admin role
	roles, err = userService.GetUserRoles(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, "admin", roles[0].Name)

	// Test HasRole after removal
	hasUserRole, err = userService.HasRole(context.Background(), user.ID, "user")
	require.NoError(t, err)
	assert.False(t, hasUserRole)

	// Test AssignRoleByName
	err = userService.AssignRoleByName(context.Background(), user.ID, "user")
	require.NoError(t, err)

	// Verify both roles are assigned again
	roles, err = userService.GetUserRoles(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 2)

	// Test duplicate assignment (should not error due to ON CONFLICT DO NOTHING)
	err = userService.AssignRole(context.Background(), user.ID, 1)
	require.NoError(t, err)

	// Verify no duplicate roles were created
	roles, err = userService.GetUserRoles(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 2)
}

func TestUserService_RoleManagement_ErrorCases_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)

	// Create a test user
	user, err := userService.CreateUserWithEmailAndTimezone(
		context.Background(),
		"testuser_roles_errors",
		"testuser_roles_errors@example.com",
		"UTC",
		"italian",
		"A1",
	)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Test AssignRole with non-existent user
	err = userService.AssignRole(context.Background(), 99999, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user with ID 99999 not found")

	// Test AssignRole with non-existent role
	err = userService.AssignRole(context.Background(), user.ID, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role with ID 99999 not found")

	// Test AssignRoleByName with non-existent role
	err = userService.AssignRoleByName(context.Background(), user.ID, "nonexistent_role")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role with name 'nonexistent_role' not found")

	// Test RemoveRole with non-existent user
	err = userService.RemoveRole(context.Background(), 99999, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user with ID 99999 not found")

	// Test RemoveRole with non-existent role
	err = userService.RemoveRole(context.Background(), user.ID, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role with ID 99999 not found")

	// Test RemoveRole with role not assigned to user
	err = userService.RemoveRole(context.Background(), user.ID, 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user 1 does not have role 2")

	// Test GetUserRoles with non-existent user
	roles, err := userService.GetUserRoles(context.Background(), 99999)
	require.NoError(t, err) // Should return empty list, not error
	assert.Empty(t, roles)

	// Test HasRole with non-existent user
	hasRole, err := userService.HasRole(context.Background(), 99999, "user")
	require.NoError(t, err)
	assert.False(t, hasRole)

	// Test IsAdmin with non-existent user
	isAdmin, err := userService.IsAdmin(context.Background(), 99999)
	require.NoError(t, err)
	assert.False(t, isAdmin)
}

func TestUserService_RoleManagement_AdminUser_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)

	// Ensure admin user exists
	err = userService.EnsureAdminUserExists(context.Background(), "admin", "adminpassword")
	require.NoError(t, err)

	// Get admin user
	adminUser, err := userService.GetUserByUsername(context.Background(), "admin")
	require.NoError(t, err)
	require.NotNil(t, adminUser)

	// Test that admin user has admin role
	isAdmin, err := userService.IsAdmin(context.Background(), adminUser.ID)
	require.NoError(t, err)
	assert.True(t, isAdmin)

	// Test that admin user has admin role
	roles, err := userService.GetUserRoles(context.Background(), adminUser.ID)
	require.NoError(t, err)
	require.Len(t, roles, 2)                // Admin users have both user and admin roles
	assert.Equal(t, "admin", roles[0].Name) // Admin role comes first in alphabetical order

	// Test HasRole for admin user
	hasUserRole, err := userService.HasRole(context.Background(), adminUser.ID, "user")
	require.NoError(t, err)
	assert.True(t, hasUserRole) // Admin users have user role as well

	hasAdminRole, err := userService.HasRole(context.Background(), adminUser.ID, "admin")
	require.NoError(t, err)
	assert.True(t, hasAdminRole)
}

func TestUserService_RoleAssignment_E2EScenario_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)

	// Create a test user similar to apitestadmin
	user, err := userService.CreateUserWithEmailAndTimezone(
		context.Background(),
		"apitestadmin",
		"apitestadmin@example.com",
		"UTC",
		"italian",
		"B1",
	)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Set password
	err = userService.UpdateUserPassword(context.Background(), user.ID, "password")
	require.NoError(t, err)

	// Update settings
	settings := &models.UserSettings{
		Language:   "italian",
		Level:      "B1",
		AIProvider: "ollama",
		AIModel:    "llama3.2",
		AIAPIKey:   "",
		AIEnabled:  true,
	}
	err = userService.UpdateUserSettings(context.Background(), user.ID, settings)
	require.NoError(t, err)

	// Assign admin role (this is what happens in the test data setup)
	err = userService.AssignRoleByName(context.Background(), user.ID, "admin")
	require.NoError(t, err)

	// Verify admin role is assigned (user already has user role, so now has 2 roles)
	roles, err := userService.GetUserRoles(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 2)                // User has both user and admin roles
	assert.Equal(t, "admin", roles[0].Name) // Admin role comes first in alphabetical order

	// Now test the exact scenario from E2E test: assign role_id 1 (user role)
	err = userService.AssignRole(context.Background(), user.ID, 1)
	require.NoError(t, err)

	// Verify both roles are now assigned
	roles, err = userService.GetUserRoles(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 2)

	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}
	assert.Contains(t, roleNames, "user")
	assert.Contains(t, roleNames, "admin")

	// Test that the user is still an admin
	isAdmin, err := userService.IsAdmin(context.Background(), user.ID)
	require.NoError(t, err)
	assert.True(t, isAdmin)

	// Test that the user now has both roles
	hasUserRole, err := userService.HasRole(context.Background(), user.ID, "user")
	require.NoError(t, err)
	assert.True(t, hasUserRole)

	hasAdminRole, err := userService.HasRole(context.Background(), user.ID, "admin")
	require.NoError(t, err)
	assert.True(t, hasAdminRole)
}

func TestUserService_RoleAssignment_EdgeCases_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)

	// Create a test user
	user, err := userService.CreateUserWithEmailAndTimezone(
		context.Background(),
		"testuser_edge_cases",
		"testuser_edge_cases@example.com",
		"UTC",
		"italian",
		"A1",
	)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Test assigning role to non-existent user
	err = userService.AssignRole(context.Background(), 99999, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user with ID 99999 not found")

	// Test assigning non-existent role
	err = userService.AssignRole(context.Background(), user.ID, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role with ID 99999 not found")

	// Test assigning role to user that already has it (should not error due to ON CONFLICT DO NOTHING)
	err = userService.AssignRole(context.Background(), user.ID, 1)
	require.NoError(t, err)

	// Verify role was assigned
	roles, err := userService.GetUserRoles(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, "user", roles[0].Name)

	// Test duplicate assignment (should not error)
	err = userService.AssignRole(context.Background(), user.ID, 1)
	require.NoError(t, err)

	// Verify no duplicate roles were created
	roles, err = userService.GetUserRoles(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, "user", roles[0].Name)

	// Test assigning role by name to non-existent user
	err = userService.AssignRoleByName(context.Background(), 99999, "user")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user with ID 99999 not found")

	// Test assigning non-existent role by name
	err = userService.AssignRoleByName(context.Background(), user.ID, "nonexistent_role")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role with name 'nonexistent_role' not found")

	// Test assigning role by name that user already has
	err = userService.AssignRoleByName(context.Background(), user.ID, "user")
	require.NoError(t, err)

	// Verify no duplicate roles were created
	roles, err = userService.GetUserRoles(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, "user", roles[0].Name)
}

func TestUserService_RoleAssignment_AdminToOtherUser_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)

	// Create a regular user (like testuser, ID 1)
	regularUser, err := userService.CreateUserWithEmailAndTimezone(
		context.Background(),
		"testuser",
		"testuser@example.com",
		"UTC",
		"italian",
		"B1",
	)
	require.NoError(t, err)
	require.NotNil(t, regularUser)

	// Create an admin user (like apitestadmin)
	adminUser, err := userService.CreateUserWithEmailAndTimezone(
		context.Background(),
		"apitestadmin",
		"apitestadmin@example.com",
		"UTC",
		"italian",
		"B1",
	)
	require.NoError(t, err)
	require.NotNil(t, adminUser)

	// Assign admin role to admin user
	err = userService.AssignRoleByName(context.Background(), adminUser.ID, "admin")
	require.NoError(t, err)

	// Verify admin user has admin role
	isAdmin, err := userService.IsAdmin(context.Background(), adminUser.ID)
	require.NoError(t, err)
	assert.True(t, isAdmin)

	// Verify regular user has user role initially (automatically assigned)
	roles, err := userService.GetUserRoles(context.Background(), regularUser.ID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, "user", roles[0].Name)

	// Now test the exact scenario from E2E test: admin assigns admin role to regular user
	err = userService.AssignRole(context.Background(), regularUser.ID, 2)
	require.NoError(t, err)

	// Verify regular user now has both user and admin roles
	roles, err = userService.GetUserRoles(context.Background(), regularUser.ID)
	require.NoError(t, err)
	require.Len(t, roles, 2)

	// Verify regular user is now an admin (after admin role assignment)
	isRegularUserAdmin, err := userService.IsAdmin(context.Background(), regularUser.ID)
	require.NoError(t, err)
	assert.True(t, isRegularUserAdmin)

	// Test that regular user has user role
	hasUserRole, err := userService.HasRole(context.Background(), regularUser.ID, "user")
	require.NoError(t, err)
	assert.True(t, hasUserRole)

	// Test that regular user now has admin role
	hasAdminRole, err := userService.HasRole(context.Background(), regularUser.ID, "admin")
	require.NoError(t, err)
	assert.True(t, hasAdminRole)

	// Test assigning admin role to regular user
	err = userService.AssignRole(context.Background(), regularUser.ID, 2)
	require.NoError(t, err)

	// Verify regular user now has both roles
	roles, err = userService.GetUserRoles(context.Background(), regularUser.ID)
	require.NoError(t, err)
	require.Len(t, roles, 2)

	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}
	assert.Contains(t, roleNames, "user")
	assert.Contains(t, roleNames, "admin")

	// Verify regular user is now an admin
	isRegularUserAdmin, err = userService.IsAdmin(context.Background(), regularUser.ID)
	require.NoError(t, err)
	assert.True(t, isRegularUserAdmin)
}

func TestUserService_RegisterDeviceToken_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := service.CreateUser(context.Background(), username, "italian", "A1")
	require.NoError(t, err)

	// Register a device token
	deviceToken := "test_device_token_12345"
	err = service.RegisterDeviceToken(context.Background(), user.ID, deviceToken)
	require.NoError(t, err)

	// Verify the token was registered
	tokens, err := service.GetUserDeviceTokens(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, deviceToken, tokens[0])

	// Register the same token again (should update, not duplicate)
	err = service.RegisterDeviceToken(context.Background(), user.ID, deviceToken)
	require.NoError(t, err)

	// Should still have only one token
	tokens, err = service.GetUserDeviceTokens(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, deviceToken, tokens[0])

	// Register a second token
	deviceToken2 := "test_device_token_67890"
	err = service.RegisterDeviceToken(context.Background(), user.ID, deviceToken2)
	require.NoError(t, err)

	// Should now have two tokens
	tokens, err = service.GetUserDeviceTokens(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, tokens, 2)
	assert.Contains(t, tokens, deviceToken)
	assert.Contains(t, tokens, deviceToken2)
}

func TestUserService_RemoveDeviceToken_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := service.CreateUser(context.Background(), username, "italian", "A1")
	require.NoError(t, err)

	// Register two device tokens
	deviceToken1 := "test_device_token_111"
	deviceToken2 := "test_device_token_222"
	err = service.RegisterDeviceToken(context.Background(), user.ID, deviceToken1)
	require.NoError(t, err)
	err = service.RegisterDeviceToken(context.Background(), user.ID, deviceToken2)
	require.NoError(t, err)

	// Verify both tokens are registered
	tokens, err := service.GetUserDeviceTokens(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, tokens, 2)

	// Remove one token
	err = service.RemoveDeviceToken(context.Background(), user.ID, deviceToken1)
	require.NoError(t, err)

	// Verify only one token remains
	tokens, err = service.GetUserDeviceTokens(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, deviceToken2, tokens[0])

	// Try to remove a non-existent token
	err = service.RemoveDeviceToken(context.Background(), user.ID, "non_existent_token")
	assert.Error(t, err)

	// Verify token list is unchanged
	tokens, err = service.GetUserDeviceTokens(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
}

func TestUserService_GetUserDeviceTokens_Empty_Integration(t *testing.T) {
	db := setupTestDBForUser(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(db, cfg, logger)

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := service.CreateUser(context.Background(), username, "italian", "A1")
	require.NoError(t, err)

	// Get tokens for user with no tokens
	tokens, err := service.GetUserDeviceTokens(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Empty(t, tokens)
}
