//go:build integration
// +build integration

package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/observability"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSnippetsService_CreateSnippet_Integration tests snippet creation
func TestSnippetsService_CreateSnippet_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	require.NotNil(t, db, "Database connection should not be nil")

	// Test that we can connect to the database
	err := db.Ping()
	require.NoError(t, err, "Database ping should succeed")

	// Test a simple query to ensure the database is working
	var result int
	err = db.QueryRow("SELECT 1").Scan(&result)
	require.NoError(t, err, "Database should be able to execute simple queries")
	require.Equal(t, 1, result, "Simple query should return 1")

	// Ensure the snippets table exists (migrations should run automatically, but let's be safe)
	err = ensureSnippetsTableExists(db)
	require.NoError(t, err, "Should be able to ensure snippets table exists")

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Create a test user first
	userService := NewUserServiceWithLogger(db, cfg, logger)
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "fr", "A1")
	require.NoError(t, err, "Should be able to create test user")
	require.NotNil(t, user, "Created user should not be nil")

	service := NewSnippetsService(db, cfg, logger)

	// Test data
	req := api.CreateSnippetRequest{
		OriginalText:   "bonjour",
		TranslatedText: "hello",
		SourceLanguage: "fr",
		TargetLanguage: "en",
		Context:        stringPtr("Test context"),
	}

	// Create snippet
	snippet, err := service.CreateSnippet(context.Background(), int64(user.ID), req)
	require.NoError(t, err)
	require.NotNil(t, snippet)

	// Verify snippet was created correctly
	assert.Greater(t, snippet.ID, int64(0))
	assert.Equal(t, int64(user.ID), snippet.UserID)
	assert.Equal(t, "bonjour", snippet.OriginalText)
	assert.Equal(t, "hello", snippet.TranslatedText)
	assert.Equal(t, "fr", snippet.SourceLanguage)
	assert.Equal(t, "en", snippet.TargetLanguage)
	assert.NotNil(t, snippet.Context)
	assert.Equal(t, "Test context", *snippet.Context)
	require.NotNil(t, snippet.DifficultyLevel)
	assert.Equal(t, "Unknown", *snippet.DifficultyLevel)
	assert.NotEmpty(t, snippet.CreatedAt)
	assert.NotEmpty(t, snippet.UpdatedAt)
}

// TestSnippetsService_CreateSnippet_Duplicate_Integration tests duplicate snippet prevention
func TestSnippetsService_CreateSnippet_Duplicate_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Create a test user first
	userService := NewUserServiceWithLogger(db, cfg, logger)
	username := fmt.Sprintf("testuser_dup_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "en", "B1")
	require.NoError(t, err, "Should be able to create test user")
	require.NotNil(t, user, "Created user should not be nil")

	service := NewSnippetsService(db, cfg, logger)

	// Test data
	req := api.CreateSnippetRequest{
		OriginalText:   "test_word",
		TranslatedText: "test_translation",
		SourceLanguage: "en",
		TargetLanguage: "es",
	}

	// Create first snippet
	_, err = service.CreateSnippet(context.Background(), int64(user.ID), req)
	require.NoError(t, err)

	// Try to create duplicate
	_, err = service.CreateSnippet(context.Background(), int64(user.ID), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snippet already exists")
}

// TestSnippetsService_GetSnippets_Integration tests snippet listing
func TestSnippetsService_GetSnippets_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Create a test user first
	userService := NewUserServiceWithLogger(db, cfg, logger)
	username := fmt.Sprintf("testuser_list_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "en", "B2")
	require.NoError(t, err, "Should be able to create test user")
	require.NotNil(t, user, "Created user should not be nil")

	service := NewSnippetsService(db, cfg, logger)

	// Create test snippets
	snippets := []api.CreateSnippetRequest{
		{
			OriginalText:   "word1",
			TranslatedText: "translation1",
			SourceLanguage: "fr",
			TargetLanguage: "en",
		},
		{
			OriginalText:   "word2",
			TranslatedText: "translation2",
			SourceLanguage: "de",
			TargetLanguage: "en",
		},
		{
			OriginalText:   "word3",
			TranslatedText: "translation3",
			SourceLanguage: "fr",
			TargetLanguage: "en",
		},
	}

	// Create snippets for the test user
	for _, req := range snippets {
		_, err := service.CreateSnippet(context.Background(), int64(user.ID), req)
		require.NoError(t, err)
	}

	// Test listing all snippets for the test user
	params := api.GetV1SnippetsParams{}
	snippetList, err := service.GetSnippets(context.Background(), int64(user.ID), params)
	require.NoError(t, err)
	require.NotNil(t, snippetList)
	assert.Equal(t, 3, *snippetList.Total)
	assert.Len(t, *snippetList.Snippets, 3)

	// Test listing snippets for non-existent user
	emptyList, err := service.GetSnippets(context.Background(), 999, params)
	require.NoError(t, err)
	require.NotNil(t, emptyList)
	assert.Equal(t, 0, *emptyList.Total)
	assert.Len(t, *emptyList.Snippets, 0)
}

// TestSnippetsService_GetSnippet_Integration tests single snippet retrieval
func TestSnippetsService_GetSnippet_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Create a test user first
	userService := NewUserServiceWithLogger(db, cfg, logger)
	username := fmt.Sprintf("testuser_get_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "it", "A2")
	require.NoError(t, err, "Should be able to create test user")
	require.NotNil(t, user, "Created user should not be nil")

	service := NewSnippetsService(db, cfg, logger)

	// Create test snippet
	req := api.CreateSnippetRequest{
		OriginalText:   "unique_word",
		TranslatedText: "unique_translation",
		SourceLanguage: "it",
		TargetLanguage: "en",
	}

	snippet, err := service.CreateSnippet(context.Background(), int64(user.ID), req)
	require.NoError(t, err)
	require.NotNil(t, snippet)

	// Test retrieving the snippet
	retrieved, err := service.GetSnippet(context.Background(), int64(user.ID), snippet.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, snippet.ID, retrieved.ID)
	assert.Equal(t, snippet.OriginalText, retrieved.OriginalText)
	assert.Equal(t, snippet.TranslatedText, retrieved.TranslatedText)

	// Test retrieving non-existent snippet
	_, err = service.GetSnippet(context.Background(), int64(user.ID), 999999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snippet not found")
}

// TestSnippetsService_UpdateSnippet_Integration tests snippet update
func TestSnippetsService_UpdateSnippet_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Create a test user first
	userService := NewUserServiceWithLogger(db, cfg, logger)
	username := fmt.Sprintf("testuser_update_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "pt", "B1")
	require.NoError(t, err, "Should be able to create test user")
	require.NotNil(t, user, "Created user should not be nil")

	service := NewSnippetsService(db, cfg, logger)

	// Create test snippet
	req := api.CreateSnippetRequest{
		OriginalText:   "update_test",
		TranslatedText: "update_translation",
		SourceLanguage: "pt",
		TargetLanguage: "en",
		Context:        stringPtr("Original context"),
	}

	snippet, err := service.CreateSnippet(context.Background(), int64(user.ID), req)
	require.NoError(t, err)
	require.NotNil(t, snippet)

	// Update the snippet context
	updateReq := api.UpdateSnippetRequest{
		Context:        stringPtr("Updated context"),
		OriginalText:   stringPtr("Updated original text"),
		TranslatedText: stringPtr("Updated translated text"),
		SourceLanguage: stringPtr("EN"),
		TargetLanguage: stringPtr("IT"),
	}

	updated, err := service.UpdateSnippet(context.Background(), int64(user.ID), snippet.ID, updateReq)
	require.NoError(t, err)
	require.NotNil(t, updated)

	assert.Equal(t, "Updated context", *updated.Context)
	assert.Equal(t, "Updated original text", updated.OriginalText)
	assert.Equal(t, "Updated translated text", updated.TranslatedText)
	assert.Equal(t, "EN", updated.SourceLanguage)
	assert.Equal(t, "IT", updated.TargetLanguage)
	assert.NotEqual(t, snippet.UpdatedAt, updated.UpdatedAt) // Should be updated

	// Test updating non-existent snippet
	_, err = service.UpdateSnippet(context.Background(), int64(user.ID), 999999, updateReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snippet not found")
}

// TestSnippetsService_DeleteSnippet_Integration tests snippet deletion
func TestSnippetsService_DeleteSnippet_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Create a test user first
	userService := NewUserServiceWithLogger(db, cfg, logger)
	username := fmt.Sprintf("testuser_delete_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "es", "A1")
	require.NoError(t, err, "Should be able to create test user")
	require.NotNil(t, user, "Created user should not be nil")

	service := NewSnippetsService(db, cfg, logger)

	// Create test snippet
	req := api.CreateSnippetRequest{
		OriginalText:   "delete_test",
		TranslatedText: "delete_translation",
		SourceLanguage: "es",
		TargetLanguage: "en",
	}

	snippet, err := service.CreateSnippet(context.Background(), int64(user.ID), req)
	require.NoError(t, err)
	require.NotNil(t, snippet)

	// Delete the snippet
	err = service.DeleteSnippet(context.Background(), int64(user.ID), snippet.ID)
	require.NoError(t, err)

	// Verify snippet is deleted
	_, err = service.GetSnippet(context.Background(), int64(user.ID), snippet.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snippet not found")

	// Test deleting non-existent snippet
	err = service.DeleteSnippet(context.Background(), int64(user.ID), 999999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snippet not found")
}

// TestSnippetsService_GetSnippets_WithFilters_Integration tests snippet filtering
func TestSnippetsService_GetSnippets_WithFilters_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Create a test user first
	userService := NewUserServiceWithLogger(db, cfg, logger)
	username := fmt.Sprintf("testuser_filter_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "en", "B2")
	require.NoError(t, err, "Should be able to create test user")
	require.NotNil(t, user, "Created user should not be nil")

	service := NewSnippetsService(db, cfg, logger)

	// Create test snippets with different languages
	snippets := []api.CreateSnippetRequest{
		{
			OriginalText:   "french_word",
			TranslatedText: "english_translation",
			SourceLanguage: "fr",
			TargetLanguage: "en",
		},
		{
			OriginalText:   "german_word",
			TranslatedText: "english_translation",
			SourceLanguage: "de",
			TargetLanguage: "en",
		},
	}

	// Create snippets for the test user
	for _, req := range snippets {
		_, err := service.CreateSnippet(context.Background(), int64(user.ID), req)
		require.NoError(t, err)
	}

	// Test filtering by source language
	params := api.GetV1SnippetsParams{
		SourceLang: stringPtr("fr"),
	}
	filteredList, err := service.GetSnippets(context.Background(), int64(user.ID), params)
	require.NoError(t, err)
	require.NotNil(t, filteredList)
	assert.Equal(t, 1, *filteredList.Total)
	assert.Len(t, *filteredList.Snippets, 1)
	assert.Equal(t, "fr", *(*filteredList.Snippets)[0].SourceLanguage)

	// Test search query
	searchParams := api.GetV1SnippetsParams{
		Q: stringPtr("french"),
	}
	searchList, err := service.GetSnippets(context.Background(), int64(user.ID), searchParams)
	require.NoError(t, err)
	require.NotNil(t, searchList)
	assert.Equal(t, 1, *searchList.Total)
}

// ensureSnippetsTableExists checks if the snippets table exists and creates it if it doesn't
func ensureSnippetsTableExists(db *sql.DB) error {
	// Check if table exists
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'snippets'
		);`

	var exists bool
	err := db.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if snippets table exists: %w", err)
	}

	if exists {
		return nil // Table already exists
	}

	// Create the snippets table
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS snippets (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			original_text TEXT NOT NULL,
			translated_text TEXT NOT NULL,
			source_language VARCHAR(10) NOT NULL,
			target_language VARCHAR(10) NOT NULL,
			question_id INTEGER REFERENCES questions(id) ON DELETE SET NULL,
			context TEXT,
			difficulty_level VARCHAR(20), -- CEFR level (A1, A2, B1, B2, C1, C2) from question or default
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

			-- Ensure one snippet per user per original text (case-insensitive)
			UNIQUE(user_id, original_text, source_language, target_language),

			FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
			FOREIGN KEY (question_id) REFERENCES questions (id)
		);

		-- Create indexes for efficient queries on snippets table
		CREATE INDEX IF NOT EXISTS idx_snippets_user_id ON snippets(user_id);
		CREATE INDEX IF NOT EXISTS idx_snippets_user_created ON snippets(user_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_snippets_source_language ON snippets(source_language);
		CREATE INDEX IF NOT EXISTS idx_snippets_target_language ON snippets(target_language);
		CREATE INDEX IF NOT EXISTS idx_snippets_question_id ON snippets(question_id);
		CREATE INDEX IF NOT EXISTS idx_snippets_search_text ON snippets USING gin(to_tsvector('english', original_text || ' ' || translated_text));`

	_, err = db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create snippets table: %w", err)
	}

	return nil
}
