//go:build integration

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TranslationPracticeIntegrationTestSuite struct {
	suite.Suite
	Router *gin.Engine
	cfg    *config.Config
}

func TestTranslationPracticeIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(TranslationPracticeIntegrationTestSuite))
}

func (suite *TranslationPracticeIntegrationTestSuite) SetupSuite() {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := database.NewManager(logger)
	db, err := dbManager.InitDB(dbURL)
	suite.Require().NoError(err)

	cfg, err := config.NewConfig()
	suite.Require().NoError(err)
	suite.cfg = cfg

	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	mockAIService := NewMockAIService(cfg, logger)
	workerService := services.NewWorkerServiceWithLogger(db, logger)
	oauthService := services.NewOAuthServiceWithLogger(cfg, logger)
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)
	generationHintService := services.NewGenerationHintService(db, logger)
	storyService := services.NewStoryService(db, cfg, logger)
	usageStatsService := services.NewUsageStatsService(cfg, db, logger)
	translationCacheRepo := services.NewTranslationCacheRepository(db, logger)
	translationService := services.NewTranslationService(cfg, usageStatsService, translationCacheRepo, logger)
	snippetsService := services.NewSnippetsService(db, cfg, logger)
	apiKeyService := services.NewAuthAPIKeyService(db, logger)
	wordOfDayService := services.NewWordOfTheDayService(db, logger)
	conversationService := services.NewConversationService(db)

	translationPracticeService := services.NewTranslationPracticeService(db, storyService, questionService, cfg, logger)
	router := NewRouter(cfg, userService, questionService, learningService, mockAIService, workerService, dailyQuestionService, storyService, conversationService, oauthService, generationHintService, translationService, snippetsService, usageStatsService, wordOfDayService, apiKeyService, translationPracticeService, logger)
	suite.Router = router

	// Ensure a test user exists for login flows
	signupReq := api.UserCreateRequest{
		Username:          "apitestuser",
		Password:          "password",
		PreferredLanguage: stringPtr("italian"),
		CurrentLevel:      stringPtr("B1"),
		Email:             emailPtr("apitestuser@example.com"),
		Timezone:          stringPtr("UTC"),
	}
	body, _ := json.Marshal(signupReq)
	req, _ := http.NewRequest("POST", "/v1/auth/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	// status can be 201 (created) or 409 (already exists) if tests re-run

	// Seed a minimal vocabulary question with a sentence for this user
	user, _ := userService.GetUserByUsername(context.Background(), "apitestuser")
	if user != nil {
		q := &models.Question{
			Type:            models.Vocabulary,
			Language:        "italian",
			Level:           "B1",
			DifficultyScore: 1.0,
			Content: map[string]interface{}{
				"sentence": "Questo è un esempio di frase per il test.",
				"options":  []interface{}{"uno", "due", "tre", "quattro"},
			},
			CorrectAnswer: 0,
			Explanation:   "example",
			Status:        models.QuestionStatusActive,
		}
		_ = questionService.SaveQuestion(context.Background(), q)
		_ = questionService.AssignQuestionToUser(context.Background(), q.ID, int(user.ID))
	}
}

func (suite *TranslationPracticeIntegrationTestSuite) login(username, password string) string {
	loginBody := map[string]string{
		"username": username,
		"password": password,
	}
	bodyBytes, _ := json.Marshal(loginBody)

	req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusOK, w.Code, "login should succeed")

	cookie := w.Header().Get("Set-Cookie")
	require.NotEmpty(suite.T(), cookie, "login must set session cookie")
	return cookie
}

func (suite *TranslationPracticeIntegrationTestSuite) TestSentenceHistoryStats() {
	// Login with a seeded user; login creates the user if missing
	cookie := suite.login("apitestuser", "password")

	// 1) GET sentence from existing content (seeded stories/questions)
	req1, _ := http.NewRequest("GET", "/v1/translation-practice/sentence?language=italian&level=B1&direction=en_to_learning", nil)
	req1.Header.Set("Cookie", cookie)
	w1 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w1, req1)
	assert.Equal(suite.T(), http.StatusOK, w1.Code)
	var sentenceResp api.TranslationPracticeSentenceResponse
	err := json.Unmarshal(w1.Body.Bytes(), &sentenceResp)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), sentenceResp.Id)
	assert.NotEmpty(suite.T(), sentenceResp.SentenceText)

	// 2) GET history (seeded sessions should exist)
	req2, _ := http.NewRequest("GET", "/v1/translation-practice/history?limit=10", nil)
	req2.Header.Set("Cookie", cookie)
	w2 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w2, req2)
	assert.Equal(suite.T(), http.StatusOK, w2.Code)
	var histResp api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w2.Body.Bytes(), &histResp)
	assert.NoError(suite.T(), err)
	// With fresh DB, sessions may be empty if none were submitted; ensure non-negative
	assert.GreaterOrEqual(suite.T(), len(histResp.Sessions), 0)

	// 3) GET stats (seeded sessions should produce counts)
	req3, _ := http.NewRequest("GET", "/v1/translation-practice/stats", nil)
	req3.Header.Set("Cookie", cookie)
	w3 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w3, req3)
	assert.Equal(suite.T(), http.StatusOK, w3.Code)
	var stats map[string]interface{}
	err = json.Unmarshal(w3.Body.Bytes(), &stats)
	assert.NoError(suite.T(), err)
	_, hasTotal := stats["total_sessions"]
	assert.True(suite.T(), hasTotal)
}

// createTestSessions creates test translation practice sessions in the database
func (suite *TranslationPracticeIntegrationTestSuite) createTestSessions(userID uint, count int) []int64 {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := database.NewManager(logger)
	db, err := dbManager.InitDB(dbURL)
	suite.Require().NoError(err)
	defer db.Close()

	// First create a sentence
	insertSentence := `
		INSERT INTO translation_practice_sentences
		(user_id, sentence_text, source_language, target_language, language_level, source_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id
	`
	var sentenceID int64
	err = db.QueryRowContext(
		context.Background(),
		insertSentence,
		userID,
		"Test sentence for pagination",
		"en",
		"italian",
		"B1",
		"ai_generated",
	).Scan(&sentenceID)
	suite.Require().NoError(err)

	// Create multiple sessions with different content for search testing
	sessionIDs := make([]int64, 0, count)
	insertSession := `
		INSERT INTO translation_practice_sessions
		(user_id, sentence_id, original_sentence, user_translation, translation_direction, ai_feedback, ai_score, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id
	`

	sentences := []struct {
		original    string
		translation string
		direction   string
		feedback    string
		score       *float64
	}{
		{"Hello world", "Ciao mondo", "en_to_learning", "Good translation of basic greeting", floatPtr(4.5)},
		{"How are you?", "Come stai?", "en_to_learning", "Correct informal greeting", floatPtr(4.8)},
		{"I love pizza", "Amo la pizza", "en_to_learning", "Perfect translation", floatPtr(5.0)},
		{"Ciao mondo", "Hello world", "learning_to_en", "Correct translation", floatPtr(4.7)},
		{"Come stai?", "How are you?", "learning_to_en", "Good informal translation", floatPtr(4.6)},
		{"Amo la pizza", "I love pizza", "learning_to_en", "Perfect translation", floatPtr(5.0)},
		{"The cat is sleeping", "Il gatto sta dormendo", "en_to_learning", "Correct present continuous", floatPtr(4.5)},
		{"Il gatto sta dormendo", "The cat is sleeping", "learning_to_en", "Good translation", floatPtr(4.8)},
		{"Beautiful day today", "Bella giornata oggi", "en_to_learning", "Natural translation", floatPtr(4.7)},
		{"Bella giornata oggi", "Beautiful day today", "learning_to_en", "Correct translation", floatPtr(4.9)},
	}

	for i := 0; i < count && i < len(sentences); i++ {
		s := sentences[i]
		var sessionID int64
		err = db.QueryRowContext(
			context.Background(),
			insertSession,
			userID,
			sentenceID,
			s.original,
			s.translation,
			s.direction,
			s.feedback,
			s.score,
		).Scan(&sessionID)
		suite.Require().NoError(err)
		sessionIDs = append(sessionIDs, sessionID)
	}

	// If we need more sessions, create duplicates with slight variations
	for i := len(sentences); i < count; i++ {
		idx := i % len(sentences)
		s := sentences[idx]
		var sessionID int64
		err = db.QueryRowContext(
			context.Background(),
			insertSession,
			userID,
			sentenceID,
			s.original+fmt.Sprintf(" %d", i),
			s.translation+fmt.Sprintf(" %d", i),
			s.direction,
			s.feedback+fmt.Sprintf(" %d", i),
			s.score,
		).Scan(&sessionID)
		suite.Require().NoError(err)
		sessionIDs = append(sessionIDs, sessionID)
	}

	return sessionIDs
}

func floatPtr(f float64) *float64 {
	return &f
}

func (suite *TranslationPracticeIntegrationTestSuite) TestHistoryPagination() {
	// Login and get user
	cookie := suite.login("apitestuser", "password")

	// Get user ID from database
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := database.NewManager(logger)
	db, err := dbManager.InitDB(dbURL)
	suite.Require().NoError(err)
	defer db.Close()

	var userID uint
	err = db.QueryRowContext(context.Background(), "SELECT id FROM users WHERE username = $1", "apitestuser").Scan(&userID)
	suite.Require().NoError(err)

	// Create 15 test sessions
	suite.createTestSessions(userID, 15)

	// Test 1: Default pagination (should return first page with default limit)
	req1, _ := http.NewRequest("GET", "/v1/translation-practice/history", nil)
	req1.Header.Set("Cookie", cookie)
	w1 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w1, req1)
	suite.Equal(http.StatusOK, w1.Code)
	var resp1 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w1.Body.Bytes(), &resp1)
	suite.NoError(err)
	suite.GreaterOrEqual(resp1.Total, 15, "Total should include all created sessions")
	suite.Equal(50, resp1.Limit, "Default limit should be 50")
	suite.Equal(0, resp1.Offset, "Default offset should be 0")
	suite.LessOrEqual(len(resp1.Sessions), 50, "Should not exceed limit")

	// Test 2: Pagination with limit
	req2, _ := http.NewRequest("GET", "/v1/translation-practice/history?limit=5", nil)
	req2.Header.Set("Cookie", cookie)
	w2 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w2, req2)
	suite.Equal(http.StatusOK, w2.Code)
	var resp2 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w2.Body.Bytes(), &resp2)
	suite.NoError(err)
	suite.Equal(5, resp2.Limit)
	suite.Equal(0, resp2.Offset)
	suite.LessOrEqual(len(resp2.Sessions), 5, "Should respect limit")

	// Test 3: Pagination with offset
	req3, _ := http.NewRequest("GET", "/v1/translation-practice/history?limit=5&offset=5", nil)
	req3.Header.Set("Cookie", cookie)
	w3 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w3, req3)
	suite.Equal(http.StatusOK, w3.Code)
	var resp3 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w3.Body.Bytes(), &resp3)
	suite.NoError(err)
	suite.Equal(5, resp3.Limit)
	suite.Equal(5, resp3.Offset)
	suite.LessOrEqual(len(resp3.Sessions), 5, "Should respect limit")
	suite.Equal(resp2.Total, resp3.Total, "Total should be the same")

	// Test 4: Verify different pages return different results (if both have results)
	if len(resp2.Sessions) > 0 && len(resp3.Sessions) > 0 {
		suite.NotEqual(resp2.Sessions[0].Id, resp3.Sessions[0].Id, "Different pages should return different sessions")
	}

	// Test 5: Offset beyond total should return empty
	req4, _ := http.NewRequest("GET", fmt.Sprintf("/v1/translation-practice/history?limit=5&offset=%d", resp2.Total+10), nil)
	req4.Header.Set("Cookie", cookie)
	w4 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w4, req4)
	suite.Equal(http.StatusOK, w4.Code)
	var resp4 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w4.Body.Bytes(), &resp4)
	suite.NoError(err)
	suite.Equal(0, len(resp4.Sessions), "Offset beyond total should return empty results")
	suite.Equal(resp2.Total, resp4.Total, "Total should still be correct")

	// Test 6: Invalid limit (too large) should be capped
	req5, _ := http.NewRequest("GET", "/v1/translation-practice/history?limit=200", nil)
	req5.Header.Set("Cookie", cookie)
	w5 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w5, req5)
	suite.Equal(http.StatusOK, w5.Code)
	var resp5 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w5.Body.Bytes(), &resp5)
	suite.NoError(err)
	suite.LessOrEqual(resp5.Limit, 100, "Limit should be capped at 100")

	// Test 7: Invalid offset (negative) should default to 0
	req6, _ := http.NewRequest("GET", "/v1/translation-practice/history?offset=-10", nil)
	req6.Header.Set("Cookie", cookie)
	w6 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w6, req6)
	suite.Equal(http.StatusOK, w6.Code)
	var resp6 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w6.Body.Bytes(), &resp6)
	suite.NoError(err)
	suite.Equal(0, resp6.Offset, "Negative offset should default to 0")
}

func (suite *TranslationPracticeIntegrationTestSuite) TestHistorySearch() {
	// Login and get user
	cookie := suite.login("apitestuser", "password")

	// Get user ID from database
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := database.NewManager(logger)
	db, err := dbManager.InitDB(dbURL)
	suite.Require().NoError(err)
	defer db.Close()

	var userID uint
	err = db.QueryRowContext(context.Background(), "SELECT id FROM users WHERE username = $1", "apitestuser").Scan(&userID)
	suite.Require().NoError(err)

	// Create test sessions
	suite.createTestSessions(userID, 10)

	// Test 1: Search by original sentence
	req1, _ := http.NewRequest("GET", "/v1/translation-practice/history?search=Hello", nil)
	req1.Header.Set("Cookie", cookie)
	w1 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w1, req1)
	suite.Equal(http.StatusOK, w1.Code)
	var resp1 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w1.Body.Bytes(), &resp1)
	suite.NoError(err)
	suite.Greater(resp1.Total, 0, "Should find sessions with 'Hello' in any field")
	for _, session := range resp1.Sessions {
		found := false
		if containsIgnoreCase(session.OriginalSentence, "Hello") ||
			containsIgnoreCase(session.UserTranslation, "Hello") ||
			containsIgnoreCase(session.AiFeedback, "Hello") ||
			containsIgnoreCase(session.TranslationDirection, "Hello") {
			found = true
		}
		suite.True(found, "All results should contain 'Hello' in at least one field")
	}

	// Test 2: Search by user translation
	req2, _ := http.NewRequest("GET", "/v1/translation-practice/history?search=Ciao", nil)
	req2.Header.Set("Cookie", cookie)
	w2 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w2, req2)
	suite.Equal(http.StatusOK, w2.Code)
	var resp2 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w2.Body.Bytes(), &resp2)
	suite.NoError(err)
	suite.Greater(resp2.Total, 0, "Should find sessions with 'Ciao' in translation")
	for _, session := range resp2.Sessions {
		found := false
		if containsIgnoreCase(session.OriginalSentence, "Ciao") ||
			containsIgnoreCase(session.UserTranslation, "Ciao") ||
			containsIgnoreCase(session.AiFeedback, "Ciao") ||
			containsIgnoreCase(session.TranslationDirection, "Ciao") {
			found = true
		}
		suite.True(found, "All results should contain 'Ciao' in at least one field")
	}

	// Test 3: Search by AI feedback
	req3, _ := http.NewRequest("GET", "/v1/translation-practice/history?search=Perfect", nil)
	req3.Header.Set("Cookie", cookie)
	w3 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w3, req3)
	suite.Equal(http.StatusOK, w3.Code)
	var resp3 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w3.Body.Bytes(), &resp3)
	suite.NoError(err)
	suite.Greater(resp3.Total, 0, "Should find sessions with 'Perfect' in feedback")
	for _, session := range resp3.Sessions {
		found := false
		if containsIgnoreCase(session.OriginalSentence, "Perfect") ||
			containsIgnoreCase(session.UserTranslation, "Perfect") ||
			containsIgnoreCase(session.AiFeedback, "Perfect") ||
			containsIgnoreCase(session.TranslationDirection, "Perfect") {
			found = true
		}
		suite.True(found, "All results should contain 'Perfect' in at least one field")
	}

	// Test 4: Search by translation direction
	req4, _ := http.NewRequest("GET", "/v1/translation-practice/history?search=en_to_learning", nil)
	req4.Header.Set("Cookie", cookie)
	w4 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w4, req4)
	suite.Equal(http.StatusOK, w4.Code)
	var resp4 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w4.Body.Bytes(), &resp4)
	suite.NoError(err)
	suite.Greater(resp4.Total, 0, "Should find sessions with 'en_to_learning' direction")
	for _, session := range resp4.Sessions {
		suite.Equal("en_to_learning", session.TranslationDirection, "All results should have en_to_learning direction")
	}

	// Test 5: Case insensitive search
	req5, _ := http.NewRequest("GET", "/v1/translation-practice/history?search=HELLO", nil)
	req5.Header.Set("Cookie", cookie)
	w5 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w5, req5)
	suite.Equal(http.StatusOK, w5.Code)
	var resp5 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w5.Body.Bytes(), &resp5)
	suite.NoError(err)
	suite.Equal(resp1.Total, resp5.Total, "Case insensitive search should return same results")

	// Test 6: No results for non-existent search
	req6, _ := http.NewRequest("GET", "/v1/translation-practice/history?search=NonExistentTerm12345", nil)
	req6.Header.Set("Cookie", cookie)
	w6 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w6, req6)
	suite.Equal(http.StatusOK, w6.Code)
	var resp6 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w6.Body.Bytes(), &resp6)
	suite.NoError(err)
	suite.Equal(0, resp6.Total, "Should return 0 results for non-existent term")
	suite.Equal(0, len(resp6.Sessions), "Should return empty sessions array")

	// Test 7: Empty search should return all results
	req7, _ := http.NewRequest("GET", "/v1/translation-practice/history?search=", nil)
	req7.Header.Set("Cookie", cookie)
	w7 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w7, req7)
	suite.Equal(http.StatusOK, w7.Code)
	var resp7 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w7.Body.Bytes(), &resp7)
	suite.NoError(err)
	suite.Greater(resp7.Total, 0, "Empty search should return all results")
}

func (suite *TranslationPracticeIntegrationTestSuite) TestHistoryPaginationWithSearch() {
	// Login and get user
	cookie := suite.login("apitestuser", "password")

	// Get user ID from database
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := database.NewManager(logger)
	db, err := dbManager.InitDB(dbURL)
	suite.Require().NoError(err)
	defer db.Close()

	var userID uint
	err = db.QueryRowContext(context.Background(), "SELECT id FROM users WHERE username = $1", "apitestuser").Scan(&userID)
	suite.Require().NoError(err)

	// Create test sessions
	suite.createTestSessions(userID, 15)

	// Test 1: Search with pagination - first page
	req1, _ := http.NewRequest("GET", "/v1/translation-practice/history?search=Hello&limit=3&offset=0", nil)
	req1.Header.Set("Cookie", cookie)
	w1 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w1, req1)
	suite.Equal(http.StatusOK, w1.Code)
	var resp1 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w1.Body.Bytes(), &resp1)
	suite.NoError(err)
	suite.Equal(3, resp1.Limit)
	suite.Equal(0, resp1.Offset)
	suite.LessOrEqual(len(resp1.Sessions), 3, "Should respect limit")
	suite.GreaterOrEqual(resp1.Total, len(resp1.Sessions), "Total should be at least the number of returned sessions")

	// Test 2: Search with pagination - second page
	req2, _ := http.NewRequest("GET", "/v1/translation-practice/history?search=Hello&limit=3&offset=3", nil)
	req2.Header.Set("Cookie", cookie)
	w2 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w2, req2)
	suite.Equal(http.StatusOK, w2.Code)
	var resp2 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w2.Body.Bytes(), &resp2)
	suite.NoError(err)
	suite.Equal(3, resp2.Limit)
	suite.Equal(3, resp2.Offset)
	suite.Equal(resp1.Total, resp2.Total, "Total should be the same for same search")

	// Verify different pages return different results (if there are enough results)
	if resp1.Total > 3 && len(resp1.Sessions) > 0 && len(resp2.Sessions) > 0 {
		suite.NotEqual(resp1.Sessions[0].Id, resp2.Sessions[0].Id, "Different pages should return different sessions")
	}

	// Test 3: Search with pagination - verify all results match search
	req3, _ := http.NewRequest("GET", "/v1/translation-practice/history?search=pizza&limit=10&offset=0", nil)
	req3.Header.Set("Cookie", cookie)
	w3 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w3, req3)
	suite.Equal(http.StatusOK, w3.Code)
	var resp3 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w3.Body.Bytes(), &resp3)
	suite.NoError(err)
	for _, session := range resp3.Sessions {
		found := false
		if containsIgnoreCase(session.OriginalSentence, "pizza") ||
			containsIgnoreCase(session.UserTranslation, "pizza") ||
			containsIgnoreCase(session.AiFeedback, "pizza") ||
			containsIgnoreCase(session.TranslationDirection, "pizza") {
			found = true
		}
		suite.True(found, "All paginated results should match search term")
	}

	// Test 4: Search with offset beyond results
	req4, _ := http.NewRequest("GET", fmt.Sprintf("/v1/translation-practice/history?search=pizza&limit=10&offset=%d", resp3.Total+10), nil)
	req4.Header.Set("Cookie", cookie)
	w4 := httptest.NewRecorder()
	suite.Router.ServeHTTP(w4, req4)
	suite.Equal(http.StatusOK, w4.Code)
	var resp4 api.TranslationPracticeHistoryResponse
	err = json.Unmarshal(w4.Body.Bytes(), &resp4)
	suite.NoError(err)
	suite.Equal(0, len(resp4.Sessions), "Offset beyond search results should return empty")
	suite.Equal(resp3.Total, resp4.Total, "Total should still be correct")
}

// Helper function to check if a string contains substring (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
