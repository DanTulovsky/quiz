//go:build integration

package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	contextutils "quizapp/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthAPIKeyServiceTest(t *testing.T) (*sql.DB, *AuthAPIKeyService, *models.User) {
	t.Helper()

	db := SharedTestDBSetup(t)
	t.Cleanup(func() {
		CleanupTestDatabase(db, t)
		_ = db.Close()
	})

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	cfg, err := config.NewConfig()
	require.NoError(t, err)

	userService := NewUserServiceWithLogger(db, cfg, logger)

	username := fmt.Sprintf("apikey_user_%d", time.Now().UnixNano())
	userEmail := fmt.Sprintf("%s@example.com", username)

	user, err := userService.CreateUserWithEmailAndTimezone(context.Background(), username, userEmail, "UTC", "italian", "A1")
	require.NoError(t, err)

	service := NewAuthAPIKeyService(db, logger)

	return db, service, user
}

func TestAuthAPIKeyServiceIntegration_FullLifecycle(t *testing.T) {
	db, service, user := setupAuthAPIKeyServiceTest(t)
	ctx := context.Background()

	apiKey, rawKey, err := service.CreateAPIKey(ctx, user.ID, "Primary Key", models.PermissionLevelFull)
	require.NoError(t, err)
	require.NotEmpty(t, rawKey)
	assert.True(t, strings.HasPrefix(rawKey, KeyPrefix))
	assert.Equal(t, models.PermissionLevelFull, apiKey.PermissionLevel)

	keys, err := service.ListAPIKeys(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, apiKey.ID, keys[0].ID)
	assert.Equal(t, "Primary Key", keys[0].KeyName)

	fetched, err := service.GetAPIKeyByID(ctx, user.ID, apiKey.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, apiKey.ID, fetched.ID)
	assert.Equal(t, apiKey.KeyPrefix, fetched.KeyPrefix)

	validated, err := service.ValidateAPIKey(ctx, rawKey)
	require.NoError(t, err)
	require.NotNil(t, validated)
	assert.Equal(t, apiKey.ID, validated.ID)

	err = service.UpdateLastUsed(ctx, apiKey.ID)
	require.NoError(t, err)

	var lastUsed sql.NullTime
	err = db.QueryRowContext(ctx, "SELECT last_used_at FROM auth_api_keys WHERE id = $1", apiKey.ID).Scan(&lastUsed)
	require.NoError(t, err)
	assert.True(t, lastUsed.Valid, "expected last_used_at to be set")

	err = service.DeleteAPIKey(ctx, user.ID, apiKey.ID)
	require.NoError(t, err)

	keysAfter, err := service.ListAPIKeys(ctx, user.ID)
	require.NoError(t, err)
	assert.Empty(t, keysAfter)
}

func TestAuthAPIKeyServiceIntegration_CreateAPIKeyValidationErrors(t *testing.T) {
	_, service, user := setupAuthAPIKeyServiceTest(t)
	ctx := context.Background()

	_, _, err := service.CreateAPIKey(ctx, user.ID, "Invalid Permission", "nope")
	require.Error(t, err)
	appErr, ok := err.(*contextutils.AppError)
	require.True(t, ok)
	assert.Equal(t, contextutils.ErrorCodeInvalidInput, appErr.Code)

	_, _, err = service.CreateAPIKey(ctx, user.ID, "", models.PermissionLevelReadonly)
	require.Error(t, err)
	appErr, ok = err.(*contextutils.AppError)
	require.True(t, ok)
	assert.Equal(t, contextutils.ErrorCodeInvalidInput, appErr.Code)
}

func TestAuthAPIKeyServiceIntegration_ValidateAndDeleteErrors(t *testing.T) {
	_, service, user := setupAuthAPIKeyServiceTest(t)
	ctx := context.Background()

	_, err := service.ValidateAPIKey(ctx, "")
	require.Error(t, err)
	assert.EqualError(t, err, "API key is empty")

	_, err = service.ValidateAPIKey(ctx, "wrongprefix_key")
	require.Error(t, err)
	assert.EqualError(t, err, "invalid API key format")

	err = service.DeleteAPIKey(ctx, user.ID, 999999)
	require.Error(t, err)
	appErr, ok := err.(*contextutils.AppError)
	require.True(t, ok)
	assert.Equal(t, contextutils.ErrorCodeRecordNotFound, appErr.Code)
}
