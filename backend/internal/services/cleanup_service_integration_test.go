//go:build integration
// +build integration

package services

import (
	"context"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupService_NewCleanupServiceWithLogger(t *testing.T) {
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(nil, logger)

	assert.NotNil(t, service)
	assert.Nil(t, service.db)
	assert.NotNil(t, service.logger)
}

func TestCleanupService_CleanupLegacyQuestionTypes_NoLegacyQuestions(t *testing.T) {
	// Setup test database
	db := SharedTestDBSetup(t)
	defer db.Close()

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(db, logger)

	// Test cleanup when no legacy questions exist
	err := service.CleanupLegacyQuestionTypes(context.Background())
	assert.NoError(t, err)
}

func TestCleanupService_CleanupLegacyQuestionTypes_WithLegacyQuestions(t *testing.T) {
	// Setup test database
	db := SharedTestDBSetup(t)
	defer db.Close()

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(db, logger)

	// Create some legacy questions with unsupported types
	_, err := db.Exec(`
		INSERT INTO questions (type, language, level, topic_category, difficulty_score, content, correct_answer, explanation, status)
		VALUES
		('legacy_type_1', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active'),
		('legacy_type_2', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active'),
		('vocabulary', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active')
	`)
	require.NoError(t, err)

	// Verify legacy questions exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM questions WHERE type NOT IN ('vocabulary', 'fill_blank', 'qa', 'reading_comprehension')").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Run cleanup
	err = service.CleanupLegacyQuestionTypes(context.Background())
	assert.NoError(t, err)

	// Verify legacy questions were removed
	err = db.QueryRow("SELECT COUNT(*) FROM questions WHERE type NOT IN ('vocabulary', 'fill_blank', 'qa', 'reading_comprehension')").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Verify valid questions remain
	err = db.QueryRow("SELECT COUNT(*) FROM questions WHERE type IN ('vocabulary', 'fill_blank', 'qa', 'reading_comprehension')").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCleanupService_CleanupLegacyQuestionTypes_DatabaseError(t *testing.T) {
	// Create service with nil database to simulate database error
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(nil, logger)

	// Test cleanup with database error
	err := service.CleanupLegacyQuestionTypes(context.Background())
	assert.Error(t, err)
}

func TestCleanupService_CleanupLegacyQuestionTypes_ContextCancellation(t *testing.T) {
	// Setup test database
	db := SharedTestDBSetup(t)
	defer db.Close()

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(db, logger)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test cleanup with cancelled context
	err := service.CleanupLegacyQuestionTypes(ctx)
	assert.Error(t, err)
}

func TestCleanupService_CleanupLegacyQuestionTypes_MixedQuestionTypes(t *testing.T) {
	// Setup test database
	db := SharedTestDBSetup(t)
	defer db.Close()

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(db, logger)

	// Create a mix of valid and legacy questions
	_, err := db.Exec(`
		INSERT INTO questions (type, language, level, topic_category, difficulty_score, content, correct_answer, explanation, status)
		VALUES
		('vocabulary', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active'),
		('fill_blank', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active'),
		('qa', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active'),
		('reading_comprehension', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active'),
		('legacy_type_1', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active'),
		('legacy_type_2', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active')
	`)
	require.NoError(t, err)

	// Verify initial state
	var totalCount, legacyCount, validCount int
	err = db.QueryRow("SELECT COUNT(*) FROM questions").Scan(&totalCount)
	require.NoError(t, err)
	assert.Equal(t, 6, totalCount)

	err = db.QueryRow("SELECT COUNT(*) FROM questions WHERE type NOT IN ('vocabulary', 'fill_blank', 'qa', 'reading_comprehension')").Scan(&legacyCount)
	require.NoError(t, err)
	assert.Equal(t, 2, legacyCount)

	err = db.QueryRow("SELECT COUNT(*) FROM questions WHERE type IN ('vocabulary', 'fill_blank', 'qa', 'reading_comprehension')").Scan(&validCount)
	require.NoError(t, err)
	assert.Equal(t, 4, validCount)

	// Run cleanup
	err = service.CleanupLegacyQuestionTypes(context.Background())
	assert.NoError(t, err)

	// Verify final state
	err = db.QueryRow("SELECT COUNT(*) FROM questions").Scan(&totalCount)
	require.NoError(t, err)
	assert.Equal(t, 4, totalCount)

	err = db.QueryRow("SELECT COUNT(*) FROM questions WHERE type NOT IN ('vocabulary', 'fill_blank', 'qa', 'reading_comprehension')").Scan(&legacyCount)
	require.NoError(t, err)
	assert.Equal(t, 0, legacyCount)

	err = db.QueryRow("SELECT COUNT(*) FROM questions WHERE type IN ('vocabulary', 'fill_blank', 'qa', 'reading_comprehension')").Scan(&validCount)
	require.NoError(t, err)
	assert.Equal(t, 4, validCount)
}

func TestCleanupService_CleanupLegacyQuestionTypes_EmptyDatabase(t *testing.T) {
	// Setup test database
	db := SharedTestDBSetup(t)
	defer db.Close()

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(db, logger)

	// Verify database is empty
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM questions").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Run cleanup on empty database
	err = service.CleanupLegacyQuestionTypes(context.Background())
	assert.NoError(t, err)

	// Verify database is still empty
	err = db.QueryRow("SELECT COUNT(*) FROM questions").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCleanupService_CleanupLegacyQuestionTypes_AllValidQuestions(t *testing.T) {
	// Setup test database
	db := SharedTestDBSetup(t)
	defer db.Close()

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(db, logger)

	// Create only valid questions
	_, err := db.Exec(`
		INSERT INTO questions (type, language, level, topic_category, difficulty_score, content, correct_answer, explanation, status)
		VALUES
		('vocabulary', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active'),
		('fill_blank', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active'),
		('qa', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active'),
		('reading_comprehension', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active')
	`)
	require.NoError(t, err)

	// Verify initial state
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM questions").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 4, count)

	// Run cleanup
	err = service.CleanupLegacyQuestionTypes(context.Background())
	assert.NoError(t, err)

	// Verify no questions were removed
	err = db.QueryRow("SELECT COUNT(*) FROM questions").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

func TestCleanupService_CleanupLegacyQuestionTypes_AllLegacyQuestions(t *testing.T) {
	// Setup test database
	db := SharedTestDBSetup(t)
	defer db.Close()

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	service := NewCleanupServiceWithLogger(db, logger)

	// Create only legacy questions
	_, err := db.Exec(`
		INSERT INTO questions (type, language, level, topic_category, difficulty_score, content, correct_answer, explanation, status)
		VALUES
		('legacy_type_1', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active'),
		('legacy_type_2', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active'),
		('legacy_type_3', 'english', 'A1', 'test', 1.0, '{"question": "test"}', 0, 'test', 'active')
	`)
	require.NoError(t, err)

	// Verify initial state
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM questions").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Run cleanup
	err = service.CleanupLegacyQuestionTypes(context.Background())
	assert.NoError(t, err)

	// Verify all legacy questions were removed
	err = db.QueryRow("SELECT COUNT(*) FROM questions").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
