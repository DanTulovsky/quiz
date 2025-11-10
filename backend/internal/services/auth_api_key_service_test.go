package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"quizapp/internal/config"
	"quizapp/internal/observability"
)

func newTestAuthAPIKeyService(t *testing.T) (*AuthAPIKeyService, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	cleanup := func() {
		require.NoError(t, mock.ExpectationsWereMet())
		require.NoError(t, db.Close())
	}

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	return &AuthAPIKeyService{db: db, logger: logger}, mock, cleanup
}

func TestAuthAPIKeyService_ListAPIKeysQueryError(t *testing.T) {
	service, mock, cleanup := newTestAuthAPIKeyService(t)
	defer cleanup()

	mock.ExpectQuery("SELECT id, user_id, key_name").
		WithArgs(42).
		WillReturnError(errors.New("query failed"))

	_, err := service.ListAPIKeys(context.Background(), 42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list API keys")
}

func TestAuthAPIKeyService_ListAPIKeysScanError(t *testing.T) {
	service, mock, cleanup := newTestAuthAPIKeyService(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id", "user_id", "key_name"}).AddRow(1, 42, "bad")
	mock.ExpectQuery("SELECT id, user_id, key_name").
		WithArgs(42).
		WillReturnRows(rows)

	_, err := service.ListAPIKeys(context.Background(), 42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to scan API key")
}

func TestAuthAPIKeyService_ListAPIKeysRowsError(t *testing.T) {
	service, mock, cleanup := newTestAuthAPIKeyService(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "key_name", "key_hash", "key_prefix", "permission_level", "last_used_at", "created_at", "updated_at",
	}).AddRow(1, 42, "name", "hash", "prefix", "readonly", nil, nil, nil).RowError(0, errors.New("iter error"))

	mock.ExpectQuery("SELECT id, user_id, key_name").
		WithArgs(42).
		WillReturnRows(rows)

	_, err := service.ListAPIKeys(context.Background(), 42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list API keys")
}

func TestAuthAPIKeyService_GetAPIKeyByIDNotFound(t *testing.T) {
	service, mock, cleanup := newTestAuthAPIKeyService(t)
	defer cleanup()

	mock.ExpectQuery("SELECT id, user_id, key_name").
		WithArgs(7, 42).
		WillReturnError(sql.ErrNoRows)

	apiKey, err := service.GetAPIKeyByID(context.Background(), 42, 7)
	require.NoError(t, err)
	assert.Nil(t, apiKey)
}

func TestAuthAPIKeyService_ValidateAPIKeyRowsError(t *testing.T) {
	service, mock, cleanup := newTestAuthAPIKeyService(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "key_name", "key_hash", "key_prefix", "permission_level", "last_used_at", "created_at", "updated_at",
	}).CloseError(errors.New("iteration failed"))

	mock.ExpectQuery("SELECT id, user_id, key_name").
		WillReturnRows(rows)

	_, err := service.ValidateAPIKey(context.Background(), KeyPrefix+"aaaaaaaaaaaaaaaa")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate API key")
}

func TestAuthAPIKeyService_UpdateLastUsedError(t *testing.T) {
	service, mock, cleanup := newTestAuthAPIKeyService(t)
	defer cleanup()

	mock.ExpectExec("UPDATE auth_api_keys").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 9).
		WillReturnError(errors.New("update failed"))

	err := service.UpdateLastUsed(context.Background(), 9)
	assert.NoError(t, err)
}
