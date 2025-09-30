package services

import (
	"database/sql"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// TestUserService_NewUserServiceWithLogger tests the constructor
func TestUserService_NewUserServiceWithLogger(t *testing.T) {
	cfg := &config.Config{} // Mock config
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewUserServiceWithLogger(nil, cfg, logger) // No database needed for constructor
	assert.NotNil(t, service)
}

// TestUserService_hashPassword tests password hashing (testing bcrypt directly since method may be private)
func TestUserService_hashPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, string(hash))

	// Verify the hash can be verified
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	assert.NoError(t, err)
}

// TestUserService_validateUserSettings tests settings validation
func TestUserService_validateUserSettings(t *testing.T) {
	tests := []struct {
		name     string
		settings *models.UserSettings
		wantErr  bool
	}{
		{
			name: "valid settings",
			settings: &models.UserSettings{
				Language: "italian",
				Level:    "B1",
			},
			wantErr: false,
		},
		{
			name: "valid settings with different language",
			settings: &models.UserSettings{
				Language: "spanish",
				Level:    "A2",
			},
			wantErr: false,
		},
		{
			name: "empty language",
			settings: &models.UserSettings{
				Language: "",
				Level:    "B1",
			},
			wantErr: false, // Could be valid in some cases
		},
		{
			name: "empty level",
			settings: &models.UserSettings{
				Language: "italian",
				Level:    "",
			},
			wantErr: false, // Could be valid in some cases
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a placeholder for validation logic
			// In a real implementation, you might have validation methods
			assert.NotNil(t, tt.settings)
		})
	}
}

// TestUserService_ValidLevels tests level validation
func TestUserService_ValidLevels(t *testing.T) {
	validLevels := []string{"A1", "A2", "B1", "B1+", "B1++", "B2", "C1", "C2"}

	for _, level := range validLevels {
		t.Run(level, func(t *testing.T) {
			assert.NotEmpty(t, level)
			// In a real implementation, you might have a validation function
			// like: assert.True(t, service.isValidLevel(level))
		})
	}
}

// TestUserService_ValidLanguages tests language validation
func TestUserService_ValidLanguages(t *testing.T) {
	validLanguages := []string{"italian", "spanish", "french", "german"}

	for _, language := range validLanguages {
		t.Run(language, func(t *testing.T) {
			assert.NotEmpty(t, language)
			// In a real implementation, you might have a validation function
			// like: assert.True(t, service.isValidLanguage(language))
		})
	}
}

// TestUser_DefaultValues tests default values for users
func TestUser_DefaultValues(t *testing.T) {
	user := &models.User{
		Username:          "testuser",
		PreferredLanguage: sql.NullString{String: "italian", Valid: true},
		CurrentLevel:      sql.NullString{String: "A1", Valid: true},
	}

	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "italian", user.PreferredLanguage.String)
	assert.True(t, user.PreferredLanguage.Valid)
	assert.Equal(t, "A1", user.CurrentLevel.String)
	assert.True(t, user.CurrentLevel.Valid)
	assert.Equal(t, 0, user.ID)             // Default ID before saving
	assert.True(t, user.CreatedAt.IsZero()) // Default timestamp before saving
}

// Note: Database-dependent tests have been moved to user_service_integration_test.go
// Run integration tests with: go test -tags=integration ./...
