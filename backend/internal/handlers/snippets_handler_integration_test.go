//go:build integration
// +build integration

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnippetsHandler_GetSnippets_WithFilters_Integration(t *testing.T) {
	// Setup test database
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Create services
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	snippetsService := services.NewSnippetsService(db, cfg, logger)
	handler := NewSnippetsHandler(snippetsService, cfg, logger)

	// Create test user
	username := fmt.Sprintf("testuser_handler_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "it", "B1")
	require.NoError(t, err, "Should be able to create test user")
	require.NotNil(t, user, "Created user should not be nil")

	// Create test snippets with different levels
	snippets := []api.CreateSnippetRequest{
		{
			OriginalText:   "word_a1",
			TranslatedText: "translation_a1",
			SourceLanguage: "it",
			TargetLanguage: "en",
		},
		{
			OriginalText:   "word_b1",
			TranslatedText: "translation_b1",
			SourceLanguage: "it",
			TargetLanguage: "en",
		},
		{
			OriginalText:   "word_c1",
			TranslatedText: "translation_c1",
			SourceLanguage: "it",
			TargetLanguage: "en",
		},
	}

	// Create snippets for the test user
	for _, req := range snippets {
		_, err := snippetsService.CreateSnippet(context.Background(), int64(user.ID), req)
		require.NoError(t, err, "Should be able to create snippet")
	}

	// Update snippets with difficulty levels using direct database update
	// since we don't have difficulty_level in CreateSnippetRequest
	levels := []string{"A1", "B1", "C1"}
	updateQuery := `UPDATE snippets SET difficulty_level = $1 WHERE user_id = $2 AND original_text = $3`
	for i, snippet := range snippets {
		_, err := db.Exec(updateQuery, levels[i], user.ID, snippet.OriginalText)
		require.NoError(t, err, "Should be able to update snippet with difficulty level")
	}

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Mock authentication middleware
		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Next()
	})
	router.GET("/v1/snippets", handler.GetSnippets)

	// Test: Get all snippets without filters
	req, _ := http.NewRequest("GET", "/v1/snippets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK")

	var response api.SnippetList
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Should be able to unmarshal response")
	require.NotNil(t, response.Snippets, "Snippets should not be nil")
	assert.Equal(t, 3, *response.Total, "Should return exactly 3 snippets")
	assert.Len(t, *response.Snippets, 3, "Should return exactly 3 snippets")

	// Test: Filter by level A1
	req, _ = http.NewRequest("GET", "/v1/snippets?level=A1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK for level filter")

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Should be able to unmarshal response")
	require.NotNil(t, response.Snippets, "Snippets should not be nil")
	assert.Equal(t, 1, *response.Total, "Should return exactly 1 snippet for A1")
	assert.Len(t, *response.Snippets, 1, "Should return exactly 1 snippet for A1")

	snippet := (*response.Snippets)[0]
	require.NotNil(t, snippet.OriginalText, "Original text should not be nil")
	assert.Equal(t, "word_a1", *snippet.OriginalText, "Should have correct original text for A1")
	require.NotNil(t, snippet.DifficultyLevel, "Difficulty level should not be nil")
	assert.Equal(t, "A1", *snippet.DifficultyLevel, "Should have correct difficulty level")

	// Test: Filter by level B1
	req, _ = http.NewRequest("GET", "/v1/snippets?level=B1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK for B1 filter")

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Should be able to unmarshal response")
	require.NotNil(t, response.Snippets, "Snippets should not be nil")
	assert.Equal(t, 1, *response.Total, "Should return exactly 1 snippet for B1")
	assert.Len(t, *response.Snippets, 1, "Should return exactly 1 snippet for B1")

	snippet = (*response.Snippets)[0]
	require.NotNil(t, snippet.OriginalText, "Original text should not be nil")
	assert.Equal(t, "word_b1", *snippet.OriginalText, "Should have correct original text for B1")
	require.NotNil(t, snippet.DifficultyLevel, "Difficulty level should not be nil")
	assert.Equal(t, "B1", *snippet.DifficultyLevel, "Should have correct difficulty level")

	// Test: Filter by story_id (should return 0 since we didn't create snippets with story_id)
	req, _ = http.NewRequest("GET", "/v1/snippets?story_id=123", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK for story_id filter")

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Should be able to unmarshal response")
	require.NotNil(t, response.Snippets, "Snippets should not be nil")
	assert.Equal(t, 0, *response.Total, "Should return 0 snippets for non-existent story_id")
	assert.Len(t, *response.Snippets, 0, "Should return 0 snippets for non-existent story_id")

	// Test: Combine level filter with search
	req, _ = http.NewRequest("GET", "/v1/snippets?q=word&level=A1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK for combined filters")

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Should be able to unmarshal response")
	require.NotNil(t, response.Snippets, "Snippets should not be nil")
	assert.Equal(t, 1, *response.Total, "Should return exactly 1 snippet for search + A1 filter")
	assert.Len(t, *response.Snippets, 1, "Should return exactly 1 snippet for search + A1 filter")

	snippet = (*response.Snippets)[0]
	require.NotNil(t, snippet.OriginalText, "Original text should not be nil")
	assert.Equal(t, "word_a1", *snippet.OriginalText, "Should have correct original text for search + A1")
	require.NotNil(t, snippet.DifficultyLevel, "Difficulty level should not be nil")
	assert.Equal(t, "A1", *snippet.DifficultyLevel, "Should have correct difficulty level for search + A1")

	// Test: Invalid level should still work (no error, just no results)
	req, _ = http.NewRequest("GET", "/v1/snippets?level=D1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK for invalid level")

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Should be able to unmarshal response")
	require.NotNil(t, response.Snippets, "Snippets should not be nil")
	assert.Equal(t, 0, *response.Total, "Should return 0 snippets for invalid level")
	assert.Len(t, *response.Snippets, 0, "Should return 0 snippets for invalid level")

	// Test: Invalid story_id should still work (no error, just no results)
	req, _ = http.NewRequest("GET", "/v1/snippets?story_id=invalid", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK for invalid story_id")

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Should be able to unmarshal response")
	require.NotNil(t, response.Snippets, "Snippets should not be nil")
	assert.Equal(t, 3, *response.Total, "Should return all snippets for invalid story_id (ignored)")
	assert.Len(t, *response.Snippets, 3, "Should return all snippets for invalid story_id (ignored)")
}
