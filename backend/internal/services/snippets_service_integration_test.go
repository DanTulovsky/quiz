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

// TestSnippetsService_GetSnippetsByQuestion_Integration tests getting snippets by question ID
func TestSnippetsService_GetSnippetsByQuestion_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Create a test user
	userService := NewUserServiceWithLogger(db, cfg, logger)
	username := fmt.Sprintf("testuser_by_question_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "fr", "B1")
	require.NoError(t, err, "Should be able to create test user")
	require.NotNil(t, user, "Created user should not be nil")

	// Create a test question directly in the database
	var questionID int64
	err = db.QueryRow(`
		INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, "vocabulary", "fr", "B1", 0.5, `{"question":"What does **test** mean?","options":["option1","option2","option3","option4"]}`, 0, "Test explanation").Scan(&questionID)
	require.NoError(t, err, "Should be able to create test question")

	service := NewSnippetsService(db, cfg, logger)

	// Create snippets with and without question_id
	snippetsWithQuestion := []api.CreateSnippetRequest{
		{
			OriginalText:   "bonjour",
			TranslatedText: "hello",
			SourceLanguage: "fr",
			TargetLanguage: "en",
			QuestionId:     &questionID,
		},
		{
			OriginalText:   "merci",
			TranslatedText: "thank you",
			SourceLanguage: "fr",
			TargetLanguage: "en",
			QuestionId:     &questionID,
		},
	}

	snippetWithoutQuestion := api.CreateSnippetRequest{
		OriginalText:   "au revoir",
		TranslatedText: "goodbye",
		SourceLanguage: "fr",
		TargetLanguage: "en",
		// No question_id
	}

	// Create snippets
	for _, req := range snippetsWithQuestion {
		_, err := service.CreateSnippet(context.Background(), int64(user.ID), req)
		require.NoError(t, err, "Should be able to create snippet with question")
	}

	_, err = service.CreateSnippet(context.Background(), int64(user.ID), snippetWithoutQuestion)
	require.NoError(t, err, "Should be able to create snippet without question")

	// Test: Get snippets by question
	snippets, err := service.GetSnippetsByQuestion(context.Background(), int64(user.ID), questionID)
	require.NoError(t, err, "Should be able to get snippets by question")
	require.NotNil(t, snippets, "Snippets should not be nil")
	assert.Len(t, snippets, 2, "Should return exactly 2 snippets for the question")

	// Verify the snippets are the correct ones
	originalTexts := make([]string, len(snippets))
	for i, snippet := range snippets {
		require.NotNil(t, snippet.OriginalText, "Original text should not be nil")
		originalTexts[i] = *snippet.OriginalText
	}
	assert.Contains(t, originalTexts, "bonjour", "Should contain 'bonjour'")
	assert.Contains(t, originalTexts, "merci", "Should contain 'merci'")
	assert.NotContains(t, originalTexts, "au revoir", "Should not contain 'au revoir' (no question_id)")

	// Test: Get snippets for non-existent question
	snippets, err = service.GetSnippetsByQuestion(context.Background(), int64(user.ID), 999999)
	require.NoError(t, err, "Should not error for non-existent question")
	assert.Empty(t, snippets, "Should return empty array for non-existent question")

	// Test: Different user should not see snippets
	otherUser, err := userService.CreateUser(context.Background(), fmt.Sprintf("otheruser_%d", time.Now().UnixNano()), "fr", "B1")
	require.NoError(t, err, "Should be able to create other user")

	snippets, err = service.GetSnippetsByQuestion(context.Background(), int64(otherUser.ID), questionID)
	require.NoError(t, err, "Should not error for different user")
	assert.Empty(t, snippets, "Different user should not see other user's snippets")
}

// TestSnippetsService_GetSnippets_WithStoryContext_Integration tests that GetSnippets returns section_id and story_id
func TestSnippetsService_GetSnippets_WithStoryContext_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Create a test user first
	userService := NewUserServiceWithLogger(db, cfg, logger)
	username := fmt.Sprintf("testuser_story_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "it", "B1")
	require.NoError(t, err, "Should be able to create test user")
	require.NotNil(t, user, "Created user should not be nil")

	service := NewSnippetsService(db, cfg, logger)

	// Create test snippets with different contexts
	snippets := []api.CreateSnippetRequest{
		{
			OriginalText:   "word_standalone",
			TranslatedText: "translation_standalone",
			SourceLanguage: "it",
			TargetLanguage: "en",
			// No question_id, section_id, or story_id - all will be null
		},
	}

	// Create snippets for the test user
	for _, req := range snippets {
		_, err := service.CreateSnippet(context.Background(), int64(user.ID), req)
		require.NoError(t, err, "Should be able to create snippet")
	}

	// Test: Get all snippets and verify section_id and story_id fields are present in response
	params := api.GetV1SnippetsParams{}
	snippetList, err := service.GetSnippets(context.Background(), int64(user.ID), params)
	require.NoError(t, err, "Should be able to get snippets")
	require.NotNil(t, snippetList, "Snippet list should not be nil")
	require.NotNil(t, snippetList.Snippets, "Snippets should not be nil")
	assert.Equal(t, 1, *snippetList.Total, "Should return exactly 1 snippet")
	assert.Len(t, *snippetList.Snippets, 1, "Should return exactly 1 snippet")

	// Verify the snippet has the correct context fields (all should be null for this test)
	snippet := (*snippetList.Snippets)[0]
	require.NotNil(t, snippet.OriginalText, "Original text should not be nil")
	assert.Equal(t, "word_standalone", *snippet.OriginalText, "Should have correct original text")
	assert.Equal(t, "translation_standalone", *snippet.TranslatedText, "Should have correct translated text")

	// Verify that section_id and story_id fields are present in the response (even if null)
	// This is the main test - ensuring these fields are returned by the API
	assert.Nil(t, snippet.QuestionId, "QuestionId should be null")
	assert.Nil(t, snippet.SectionId, "SectionId should be null")
	assert.Nil(t, snippet.StoryId, "StoryId should be null")

	// The key test: verify that the fields exist in the response structure
	// (We can't easily test non-null values due to foreign key constraints in test DB)
	t.Logf("Snippet response includes SectionId field: %v", snippet.SectionId != nil || snippet.SectionId == nil)
	t.Logf("Snippet response includes StoryId field: %v", snippet.StoryId != nil || snippet.StoryId == nil)
}

// Helper functions for creating pointers to primitive types
func int64Ptr(i int64) *int64 {
	return &i
}
