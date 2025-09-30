//go:build integration

package services

import (
	"context"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserService_AIEnabledFunctionality_Integration(t *testing.T) {
	// Setup test database
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg := &config.Config{}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)

	t.Run("New user has AI disabled by default", func(t *testing.T) {
		// Create a new user
		testUser, err := userService.CreateUserWithEmailAndTimezone(context.Background(), "testuser", "test@example.com", "UTC", "italian", "A1")
		require.NoError(t, err)
		require.NotNil(t, testUser)

		// Fetch the user and check AI enabled status
		user, err := userService.GetUserByID(context.Background(), testUser.ID)
		require.NoError(t, err)
		require.NotNil(t, user)

		// Verify AI is disabled by default
		assert.True(t, user.AIEnabled.Valid, "AIEnabled should be valid")
		assert.False(t, user.AIEnabled.Bool, "AIEnabled should be false by default for new users")
	})

	t.Run("Can enable AI and set provider/model", func(t *testing.T) {
		// Create a new user
		testUser, err := userService.CreateUserWithEmailAndTimezone(context.Background(), "testuser2", "test2@example.com", "UTC", "italian", "A1")
		require.NoError(t, err)

		// Enable AI with provider and model
		err = userService.UpdateUserSettings(context.Background(), testUser.ID, &models.UserSettings{
			Language:   "italian",
			Level:      "A1",
			AIProvider: "ollama",
			AIModel:    "llama4:latest",
			AIEnabled:  true,
		})
		require.NoError(t, err)

		// Fetch user and verify AI is enabled
		user, err := userService.GetUserByID(context.Background(), testUser.ID)
		require.NoError(t, err)
		require.NotNil(t, user)

		assert.True(t, user.AIEnabled.Valid, "AIEnabled should be valid")
		assert.True(t, user.AIEnabled.Bool, "AIEnabled should be true after enabling")
		assert.True(t, user.AIProvider.Valid, "AIProvider should be valid")
		assert.Equal(t, "ollama", user.AIProvider.String, "AIProvider should be set")
		assert.True(t, user.AIModel.Valid, "AIModel should be valid")
		assert.Equal(t, "llama4:latest", user.AIModel.String, "AIModel should be set")
	})

	t.Run("Disabling AI clears provider and model", func(t *testing.T) {
		// Create a new user and enable AI
		testUser, err := userService.CreateUserWithEmailAndTimezone(context.Background(), "testuser3", "test3@example.com", "UTC", "italian", "A1")
		require.NoError(t, err)

		// First enable AI
		err = userService.UpdateUserSettings(context.Background(), testUser.ID, &models.UserSettings{
			Language:   "italian",
			Level:      "A1",
			AIProvider: "ollama",
			AIModel:    "llama4:latest",
			AIEnabled:  true,
		})
		require.NoError(t, err)

		// Now disable AI
		err = userService.UpdateUserSettings(context.Background(), testUser.ID, &models.UserSettings{
			Language:  "italian",
			Level:     "A1",
			AIEnabled: false,
		})
		require.NoError(t, err)

		// Fetch user and verify AI is disabled and provider/model cleared
		user, err := userService.GetUserByID(context.Background(), testUser.ID)
		require.NoError(t, err)
		require.NotNil(t, user)

		assert.True(t, user.AIEnabled.Valid, "AIEnabled should be valid")
		assert.False(t, user.AIEnabled.Bool, "AIEnabled should be false after disabling")

		// Check if provider and model were cleared
		if user.AIProvider.Valid {
			assert.Empty(t, user.AIProvider.String, "AIProvider should be empty when AI disabled")
		}
		if user.AIModel.Valid {
			assert.Empty(t, user.AIModel.String, "AIModel should be empty when AI disabled")
		}
	})

	t.Run("CreateUserWithPassword defaults to AI disabled", func(t *testing.T) {
		// Create user with password
		testUser, err := userService.CreateUserWithPassword(context.Background(), "testuser4", "password123", "italian", "A1")
		require.NoError(t, err)
		require.NotNil(t, testUser)

		// Fetch the user and check AI enabled status
		user, err := userService.GetUserByID(context.Background(), testUser.ID)
		require.NoError(t, err)
		require.NotNil(t, user)

		// Verify AI is disabled by default
		assert.True(t, user.AIEnabled.Valid, "AIEnabled should be valid")
		assert.False(t, user.AIEnabled.Bool, "AIEnabled should be false by default for password users")
	})

	t.Run("GetAllUsers includes ai_enabled field", func(t *testing.T) {
		// Create a test user with AI enabled
		testUser, err := userService.CreateUserWithEmailAndTimezone(context.Background(), "testuser5", "test5@example.com", "UTC", "italian", "A1")
		require.NoError(t, err)

		err = userService.UpdateUserSettings(context.Background(), testUser.ID, &models.UserSettings{
			Language:   "italian",
			Level:      "A1",
			AIProvider: "ollama",
			AIModel:    "llama4:latest",
			AIEnabled:  true,
		})
		require.NoError(t, err)

		// Get all users
		users, err := userService.GetAllUsers(context.Background())
		require.NoError(t, err)

		// Find our test user
		var foundUser *models.User
		for _, user := range users {
			if user.ID == testUser.ID {
				foundUser = &user
				break
			}
		}
		require.NotNil(t, foundUser, "Test user should be found in GetAllUsers")

		// Verify AI enabled field is properly populated
		assert.True(t, foundUser.AIEnabled.Valid, "AIEnabled should be valid in GetAllUsers")
		assert.True(t, foundUser.AIEnabled.Bool, "AIEnabled should be true for this test user")
	})

	t.Run("Worker simulation - filters users by ai_enabled", func(t *testing.T) {
		// This simulates the worker's user filtering logic
		// Create multiple test users with different AI settings
		testUsers := []struct {
			username  string
			email     string
			aiEnabled bool
		}{
			{"worker_test1", "wt1@example.com", true},
			{"worker_test2", "wt2@example.com", false},
			{"worker_test3", "wt3@example.com", true},
			{"worker_test4", "wt4@example.com", false},
		}

		var createdUsers []models.User
		for _, userData := range testUsers {
			user, err := userService.CreateUserWithEmailAndTimezone(context.Background(), userData.username, userData.email, "UTC", "italian", "A1")
			require.NoError(t, err)

			if userData.aiEnabled {
				err = userService.UpdateUserSettings(context.Background(), user.ID, &models.UserSettings{
					Language:   "italian",
					Level:      "A1",
					AIProvider: "ollama",
					AIModel:    "llama4:latest",
					AIEnabled:  true,
				})
				require.NoError(t, err)
			}
			createdUsers = append(createdUsers, *user)
		}

		// Get all users (simulating worker's GetAllUsers call)
		allUsers, err := userService.GetAllUsers(context.Background())
		require.NoError(t, err)

		// Filter users with AI enabled (simulating worker's filtering logic)
		var aiEnabledUsers []models.User
		for _, user := range allUsers {
			// Check if this is one of our test users
			isTestUser := false
			for _, createdUser := range createdUsers {
				if user.ID == createdUser.ID {
					isTestUser = true
					break
				}
			}
			if !isTestUser {
				continue // Skip users created by other tests
			}

			// Worker's filtering logic: skip users with AI disabled
			if !user.AIEnabled.Valid || !user.AIEnabled.Bool {
				t.Logf("Worker would skip user %s (ID: %d) - AI disabled", user.Username, user.ID)
				continue
			}

			aiEnabledUsers = append(aiEnabledUsers, user)
		}

		// Verify that only AI-enabled users would be processed by the worker
		assert.Len(t, aiEnabledUsers, 2, "Worker should only process 2 AI-enabled users")

		for _, user := range aiEnabledUsers {
			assert.True(t, user.AIEnabled.Valid, "AI-enabled user should have valid AIEnabled field")
			assert.True(t, user.AIEnabled.Bool, "AI-enabled user should have AI enabled")
			assert.True(t, user.Username == "worker_test1" || user.Username == "worker_test3",
				"Only worker_test1 and worker_test3 should be AI-enabled")
		}
	})
}
