package services

import (
	"context"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCleanupService(t *testing.T) {
	// Use nil database for testing tracer functionality
	service := NewCleanupServiceWithLogger(nil, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))
	assert.NotNil(t, service)
	assert.Nil(t, service.db)
	assert.NotNil(t, service.logger, "CleanupService should have a logger")
}

func TestCleanupService_GlobalTracer(t *testing.T) {
	// Use nil database for testing tracer functionality
	service := NewCleanupServiceWithLogger(nil, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Verify that the service uses the global tracer
	assert.NotNil(t, service.logger, "CleanupService should have a logger")

	// Test that the global tracer is properly initialized
	ctx := context.Background()
	ctx, span := observability.TraceCleanupFunction(ctx, "test_function")
	assert.NotNil(t, span, "Global tracer should create valid spans")
	span.End()

	// Test error handling with the global tracer
	err := observability.TraceFunctionWithErrorHandling(ctx, "cleanup", "test_error_function", func() error {
		return assert.AnError
	})
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestCleanupOrphanedResponses_NoOrphans(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
		require.NoError(t, mock.ExpectationsWereMet())
	}()

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(db, logger)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)\\s+FROM user_responses ur").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	err = service.CleanupOrphanedResponses(context.Background())
	require.NoError(t, err)
}

func TestCleanupOrphanedResponses_WithOrphans(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
		require.NoError(t, mock.ExpectationsWereMet())
	}()

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(db, logger)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)\\s+FROM user_responses ur").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectExec("DELETE FROM user_responses").
		WillReturnResult(sqlmock.NewResult(0, 3))

	err = service.CleanupOrphanedResponses(context.Background())
	require.NoError(t, err)
}

func TestCleanupOrphanedResponses_NoDatabase(t *testing.T) {
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(nil, logger)

	err := service.CleanupOrphanedResponses(context.Background())
	require.EqualError(t, err, "database connection not available")
}

func TestCleanupService_RunFullCleanup(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
		require.NoError(t, mock.ExpectationsWereMet())
	}()

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(db, logger)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)\\s+FROM questions").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)\\s+FROM user_responses ur").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	err = service.RunFullCleanup(context.Background())
	require.NoError(t, err)
}

func TestCleanupService_RunFullCleanup_ErrorFromLegacyCleanup(t *testing.T) {
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(nil, logger)

	err := service.RunFullCleanup(context.Background())
	require.EqualError(t, err, "database connection not available")
}

func TestCleanupService_GetCleanupStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
		require.NoError(t, mock.ExpectationsWereMet())
	}()

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(db, logger)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)\\s+FROM questions").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)\\s+FROM user_responses ur").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	stats, err := service.GetCleanupStats(context.Background())
	require.NoError(t, err)
	require.Equal(t, map[string]int{
		"legacy_questions":   4,
		"orphaned_responses": 2,
	}, stats)
}

func TestCleanupService_GetCleanupStats_NoDatabase(t *testing.T) {
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(nil, logger)

	stats, err := service.GetCleanupStats(context.Background())
	require.Nil(t, stats)
	require.EqualError(t, err, "database connection not available")
}
