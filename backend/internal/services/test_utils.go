//go:build integration

package services

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/require"
)

// SharedTestDBSetup provides a clean, isolated database for each integration test
// Uses the optimized CleanupTestDatabase function for consistent cleanup
func SharedTestDBSetup(t *testing.T) *sql.DB {
	observabilityLogger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := database.NewManager(observabilityLogger)

	// Require TEST_DATABASE_URL environment variable to be set
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL environment variable must be set for integration tests")
	}

	db, err := dbManager.InitDB(databaseURL)
	require.NoError(t, err)

	// Use the optimized cleanup function
	CleanupTestDatabase(db, t)

	return db
}

// cleanupDatabase performs the core database cleanup operations
// This is the shared implementation used by both CleanupTestDatabase and SharedTestSuite.Cleanup
func cleanupDatabase(db *sql.DB, logger *observability.Logger) {
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "Failed to begin cleanup transaction", err)
		}
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Fast cleanup with batched operations
	cleanupQueries := []string{
		"TRUNCATE TABLE user_responses CASCADE",
		"TRUNCATE TABLE performance_metrics CASCADE",
		"TRUNCATE TABLE user_question_metadata CASCADE",
		"TRUNCATE TABLE question_priority_scores CASCADE",
		"TRUNCATE TABLE user_learning_preferences CASCADE",
		"TRUNCATE TABLE user_questions CASCADE",
		"TRUNCATE TABLE questions CASCADE",
		"TRUNCATE TABLE worker_status CASCADE",
		"TRUNCATE TABLE worker_settings CASCADE",
		"TRUNCATE TABLE user_api_keys CASCADE",
		"TRUNCATE TABLE user_roles CASCADE",
		"TRUNCATE TABLE question_reports CASCADE",
		"TRUNCATE TABLE notification_errors CASCADE",
		"TRUNCATE TABLE upcoming_notifications CASCADE",
		"TRUNCATE TABLE sent_notifications CASCADE",
		"TRUNCATE TABLE auth_api_keys CASCADE",
		"TRUNCATE TABLE daily_question_assignments CASCADE",
		"TRUNCATE TABLE story_sections CASCADE",
		"TRUNCATE TABLE story_section_questions CASCADE",
		"TRUNCATE TABLE stories CASCADE",
		"TRUNCATE TABLE snippets CASCADE",
		"TRUNCATE TABLE usage_stats CASCADE",
		"TRUNCATE TABLE users CASCADE",
	}

	for _, query := range cleanupQueries {
		_, err := tx.ExecContext(ctx, query)
		if err != nil {
			if logger != nil {
				logger.Warn(ctx, "Could not execute cleanup query", map[string]interface{}{
					"query": query,
				})
			}
		}
	}

	// Reset sequences
	sequenceQueries := []string{
		"ALTER SEQUENCE users_id_seq RESTART WITH 1",
		"ALTER SEQUENCE questions_id_seq RESTART WITH 1",
		"ALTER SEQUENCE user_responses_id_seq RESTART WITH 1",
		"ALTER SEQUENCE performance_metrics_id_seq RESTART WITH 1",
		"ALTER SEQUENCE snippets_id_seq RESTART WITH 1",
		"ALTER SEQUENCE auth_api_keys_id_seq RESTART WITH 1",
	}

	for _, query := range sequenceQueries {
		_, err := tx.ExecContext(ctx, query)
		if err != nil {
			if logger != nil {
				logger.Warn(ctx, "Could not reset sequence", map[string]interface{}{
					"query": query,
				})
			}
		}
	}

	// Re-insert default worker settings
	_, err = tx.ExecContext(ctx, `
		INSERT INTO worker_settings (setting_key, setting_value, created_at, updated_at)
		VALUES ('global_pause', 'false', NOW(), NOW())
		ON CONFLICT (setting_key) DO NOTHING;
	`)
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "Failed to insert worker settings", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "Failed to commit cleanup transaction", err)
		}
	}
}

// CleanupTestDatabase cleans up the database for integration tests
// This function can be used by any integration test that needs to clean up the database
// Optimized to use batched transactions for better performance
func CleanupTestDatabase(db *sql.DB, t *testing.T) {
	cleanupDatabase(db, nil)
}
