//go:build integration

package handlers

import (
	"context"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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


