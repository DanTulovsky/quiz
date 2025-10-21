//go:build integration
// +build integration

package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/handlers"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// closeNotifierRecorder is a custom response recorder that implements http.CloseNotifier
// to fix the panic in streaming tests with newer Go versions
type closeNotifierRecorder struct {
	*httptest.ResponseRecorder
	closeNotify chan bool
}

func newCloseNotifierRecorder() *closeNotifierRecorder {
	return &closeNotifierRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		closeNotify:      make(chan bool, 1),
	}
}

func (r *closeNotifierRecorder) CloseNotify() <-chan bool {
	return r.closeNotify
}

type QuizIntegrationTestSuite struct {
	suite.Suite
	Router      *gin.Engine
	db          *sql.DB
	testUser    *models.User
	userService *services.UserService
	cfg         *config.Config
}

func (suite *QuizIntegrationTestSuite) SetupSuite() {
	// Initialize database
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://quiz_user:quiz_password@localhost:5433/quiz_test_db?sslmode=disable"
	}

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	dbManager := database.NewManager(logger)
	db, err := dbManager.InitDB(dbURL)
	suite.Require().NoError(err)
	suite.db = db

	// Load config
	cfg, err := config.NewConfig()
	suite.Require().NoError(err)
	suite.cfg = cfg

	// Initialize services
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	suite.userService = userService
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	aiService := services.NewAIService(cfg, logger, services.NewNoopUsageStatsService())
	workerService := services.NewWorkerServiceWithLogger(db, logger)
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)
	oauthService := services.NewOAuthServiceWithLogger(cfg, logger)
	generationHintService := services.NewGenerationHintService(db, logger)

	// Use the real application router
	storyService := services.NewStoryService(db, cfg, logger)
	usageStatsService := services.NewUsageStatsService(cfg, db, logger)
	translationService := services.NewTranslationService(cfg, usageStatsService, logger)
	snippetsService := services.NewSnippetsService(db, cfg, logger)
	router := handlers.NewRouter(cfg, userService, questionService, learningService, aiService, workerService, dailyQuestionService, storyService, services.NewConversationService(db), oauthService, generationHintService, translationService, snippetsService, usageStatsService, logger)
	suite.Router = router
}

func (suite *QuizIntegrationTestSuite) SetupTest() {
	// Clean up database before each test using the shared cleanup function
	services.CleanupTestDatabase(suite.db, suite.T())

	// Create a test user
	testUser, err := suite.userService.CreateUserWithPassword(context.Background(), "testuser_quiz", "testpass", "english", "A1")
	suite.Require().NoError(err)
	suite.testUser = testUser

	// Verify that the user has the correct preferences set
	suite.Require().True(testUser.PreferredLanguage.Valid, "PreferredLanguage should be valid")
	suite.Require().Equal("english", testUser.PreferredLanguage.String, "PreferredLanguage should be 'english'")
	suite.Require().True(testUser.CurrentLevel.Valid, "CurrentLevel should be valid")
	suite.Require().Equal("A1", testUser.CurrentLevel.String, "CurrentLevel should be 'A1'")

	// Reload the user to make sure the preferences are properly set in the database
	reloadedUser, err := suite.userService.GetUserByID(context.Background(), testUser.ID)
	suite.Require().NoError(err)
	suite.Require().NotNil(reloadedUser)
	suite.Require().True(reloadedUser.PreferredLanguage.Valid, "Reloaded user PreferredLanguage should be valid")
	suite.Require().Equal("english", reloadedUser.PreferredLanguage.String, "Reloaded user PreferredLanguage should be 'english'")
	suite.Require().True(reloadedUser.CurrentLevel.Valid, "Reloaded user CurrentLevel should be valid")
	suite.Require().Equal("A1", reloadedUser.CurrentLevel.String, "Reloaded user CurrentLevel should be 'A1'")
}

func (suite *QuizIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *QuizIntegrationTestSuite) login() string {
	loginReq := api.LoginRequest{
		Username: "testuser_quiz",
		Password: "testpass",
	}
	reqBody, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Extract cookie from response
	cookies := w.Result().Cookies()
	var sessionCookie string
	for _, cookie := range cookies {
		if cookie.Name == config.SessionName {
			sessionCookie = cookie.String()
			break
		}
	}
	return sessionCookie
}

func (suite *QuizIntegrationTestSuite) createTestQuestion() int {
	// Create a test question
	testQuestion := &models.Question{
		Type:            models.Vocabulary,
		Language:        "english",
		Level:           "A1",
		TopicCategory:   "test",
		DifficultyScore: 1.0,
		Content:         map[string]interface{}{"question": "What is hello?", "options": []string{"Hello", "Goodbye", "Thanks", "Please"}},
		CorrectAnswer:   0,
		Explanation:     "Hello is a greeting",
		Status:          models.QuestionStatusActive,
	}

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, logger)

	err := questionService.SaveQuestion(context.Background(), testQuestion)
	suite.Require().NoError(err)

	// Assign question to user
	err = questionService.AssignQuestionToUser(context.Background(), testQuestion.ID, suite.testUser.ID)
	suite.Require().NoError(err)

	return testQuestion.ID
}

func (suite *QuizIntegrationTestSuite) TestGetQuestion_Success() {
	// Create a test question
	questionID := suite.createTestQuestion()

	// Login
	cookie := suite.login()

	// Get question
	req, _ := http.NewRequest("GET", "/v1/quiz/question", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Check if we got a question or a generating response
	if w.Code == http.StatusOK {
		var questionResp api.Question
		err := json.Unmarshal(w.Body.Bytes(), &questionResp)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), int64(questionID), *questionResp.Id)
		assert.Equal(suite.T(), "What is hello?", questionResp.Content.Question)
	} else if w.Code == http.StatusAccepted {
		var generatingResp api.GeneratingResponse
		err := json.Unmarshal(w.Body.Bytes(), &generatingResp)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "generating", *generatingResp.Status)
	} else {
		suite.T().Fatalf("Unexpected status code: %d", w.Code)
	}
}

func (suite *QuizIntegrationTestSuite) TestGetQuestion_NoQuestionsAvailable() {
	// Login without creating any questions
	cookie := suite.login()

	// Try to get a question, requesting a specific type to trigger hint
	req, _ := http.NewRequest("GET", "/v1/quiz/question?type=reading_comprehension", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Should return 202 Accepted with generating status
	assert.Equal(suite.T(), http.StatusAccepted, w.Code)

	var generatingResp api.GeneratingResponse
	err := json.Unmarshal(w.Body.Bytes(), &generatingResp)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "generating", *generatingResp.Status)

	// If a type is requested and none available, server records a generation hint (best effort); validate message updated
	// Note: We can't directly observe DB here; just verify message mentions prioritizing when present
	if generatingResp.Message != nil {
		assert.Contains(suite.T(), *generatingResp.Message, "Prioritizing")
	}
}

func (suite *QuizIntegrationTestSuite) TestGetQuestion_Unauthorized() {
	// Try to get a question without authentication
	req, _ := http.NewRequest("GET", "/v1/quiz/question", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *QuizIntegrationTestSuite) TestGetQuestion_ByID() {
	// Create a test question
	questionID := suite.createTestQuestion()

	// Login
	cookie := suite.login()

	// Get specific question by ID
	req, _ := http.NewRequest("GET", "/v1/quiz/question/"+strconv.Itoa(questionID), nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var questionResp api.Question
	err := json.Unmarshal(w.Body.Bytes(), &questionResp)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(questionID), *questionResp.Id)
}

func (suite *QuizIntegrationTestSuite) TestGetQuestion_InvalidID() {
	// Login
	cookie := suite.login()

	// Try to get question with invalid ID
	req, _ := http.NewRequest("GET", "/v1/quiz/question/invalid", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *QuizIntegrationTestSuite) TestSubmitAnswer_Success() {
	// Create a test question
	questionID := suite.createTestQuestion()

	// Login
	cookie := suite.login()

	// Submit correct answer
	answerReq := api.AnswerRequest{
		QuestionId:      int64(questionID),
		UserAnswerIndex: 0,
		ResponseTimeMs:  nil,
	}
	reqBody, _ := json.Marshal(answerReq)

	req, _ := http.NewRequest("POST", "/v1/quiz/answer", bytes.NewBuffer(reqBody))
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var answerResp api.AnswerResponse
	err := json.Unmarshal(w.Body.Bytes(), &answerResp)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), *answerResp.IsCorrect)
}

func (suite *QuizIntegrationTestSuite) TestSubmitAnswer_Incorrect() {
	// Create a test question
	questionID := suite.createTestQuestion()

	// Login
	cookie := suite.login()

	// Submit incorrect answer (index 1 instead of 0)
	answerReq := api.AnswerRequest{
		QuestionId:      int64(questionID),
		UserAnswerIndex: 1,
		ResponseTimeMs:  nil,
	}
	reqBody, _ := json.Marshal(answerReq)

	req, _ := http.NewRequest("POST", "/v1/quiz/answer", bytes.NewBuffer(reqBody))
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var answerResp api.AnswerResponse
	err := json.Unmarshal(w.Body.Bytes(), &answerResp)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), *answerResp.IsCorrect)
}

func (suite *QuizIntegrationTestSuite) TestSubmitAnswer_InvalidRequest() {
	// Login
	cookie := suite.login()

	// Submit invalid request
	reqBody := []byte(`{"invalid": "json"`)

	req, _ := http.NewRequest("POST", "/v1/quiz/answer", bytes.NewBuffer(reqBody))
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *QuizIntegrationTestSuite) TestSubmitAnswer_Unauthorized() {
	// Submit answer without authentication
	answerReq := api.AnswerRequest{
		QuestionId:      1,
		UserAnswerIndex: 0,
		ResponseTimeMs:  nil,
	}
	reqBody, _ := json.Marshal(answerReq)

	req, _ := http.NewRequest("POST", "/v1/quiz/answer", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *QuizIntegrationTestSuite) TestGetProgress_Success() {
	// Create a test question and submit an answer
	questionID := suite.createTestQuestion()
	cookie := suite.login()

	// Submit an answer first
	answerReq := api.AnswerRequest{
		QuestionId:      int64(questionID),
		UserAnswerIndex: 0,
		ResponseTimeMs:  nil,
	}
	reqBody, _ := json.Marshal(answerReq)

	req, _ := http.NewRequest("POST", "/v1/quiz/answer", bytes.NewBuffer(reqBody))
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Now get progress
	req, _ = http.NewRequest("GET", "/v1/quiz/progress", nil)
	req.Header.Set("Cookie", cookie)

	w = httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var progressResp models.UserProgress
	err := json.Unmarshal(w.Body.Bytes(), &progressResp)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), progressResp)
}

func (suite *QuizIntegrationTestSuite) TestGetProgress_Unauthorized() {
	// Try to get progress without authentication
	req, _ := http.NewRequest("GET", "/v1/quiz/progress", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *QuizIntegrationTestSuite) TestMarkQuestionAsKnown_Success() {
	// Create a test question
	questionID := suite.createTestQuestion()

	// Login
	cookie := suite.login()

	// Mark question as known
	req, _ := http.NewRequest("POST", "/v1/quiz/question/"+strconv.Itoa(questionID)+"/mark-known", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var resp api.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), *resp.Message, "marked as known")
}

func (suite *QuizIntegrationTestSuite) TestMarkQuestionAsKnown_InvalidID() {
	// Login
	cookie := suite.login()

	// Try to mark question with invalid ID
	req, _ := http.NewRequest("POST", "/v1/quiz/question/invalid/mark-known", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *QuizIntegrationTestSuite) TestMarkQuestionAsKnown_Unauthorized() {
	// Try to mark question as known without authentication
	req, _ := http.NewRequest("POST", "/v1/quiz/question/1/mark-known", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *QuizIntegrationTestSuite) TestReportQuestion_Success() {
	// Create a test question
	questionID := suite.createTestQuestion()

	// Login
	cookie := suite.login()

	// Report question - no request body needed
	req, _ := http.NewRequest("POST", "/v1/quiz/question/"+strconv.Itoa(questionID)+"/report", nil)
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var resp api.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), *resp.Message, "reported")
}

func (suite *QuizIntegrationTestSuite) TestReportQuestion_InvalidRequest() {
	// Create a test question
	questionID := suite.createTestQuestion()

	// Login
	cookie := suite.login()

	// Submit invalid report request - the handler doesn't validate request body
	// so this should succeed even with invalid JSON
	reqBody := []byte(`{"invalid": "json"`)

	req, _ := http.NewRequest("POST", "/v1/quiz/question/"+strconv.Itoa(questionID)+"/report", bytes.NewBuffer(reqBody))
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// The handler ignores the request body, so this should succeed
	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *QuizIntegrationTestSuite) TestReportQuestion_Unauthorized() {
	// Try to report question without authentication
	req, _ := http.NewRequest("POST", "/v1/quiz/question/1/report", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *QuizIntegrationTestSuite) TestReportQuestion_NotFound() {
	// Login
	cookie := suite.login()

	// Try to report a non-existent question
	nonExistentQuestionID := 999999
	req, _ := http.NewRequest("POST", "/v1/quiz/question/"+strconv.Itoa(nonExistentQuestionID)+"/report", nil)
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Should return 404 Not Found
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	// Check the response format - it could be either ErrorResponse or a different structure
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(suite.T(), err)

	// The response should contain a message field with the error information
	assert.Contains(suite.T(), resp, "message")
	assert.Contains(suite.T(), resp["message"].(string), "Question not found")
}

func (suite *QuizIntegrationTestSuite) TestGetWorkerStatus_Success() {
	// Login
	cookie := suite.login()

	// Get worker status
	req, _ := http.NewRequest("GET", "/v1/quiz/worker-status", nil)
	req.Header.Set("Cookie", cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	// Should return 200 OK (even if no workers are running)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var statusResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &statusResp)
	assert.NoError(suite.T(), err)
}

func (suite *QuizIntegrationTestSuite) TestGetWorkerStatus_Unauthorized() {
	// Try to get worker status without authentication
	req, _ := http.NewRequest("GET", "/v1/quiz/worker-status", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *QuizIntegrationTestSuite) TestChatStream_Unauthorized() {
	// Try to access chat stream without authentication
	req, _ := http.NewRequest("POST", "/v1/quiz/chat/stream", nil)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *QuizIntegrationTestSuite) TestChatStream_VocabularyQuestionWithSentence() {
	// Create a vocabulary question with a sentence field
	vocabularyQuestion := &models.Question{
		Type:            models.Vocabulary,
		Language:        "italian",
		Level:           "B1",
		TopicCategory:   "test",
		DifficultyScore: 1.0,
		Content: map[string]interface{}{
			"question": "What does stazione mean in this context?",
			"options":  []string{"bank", "park", "shop", "station"},
			"sentence": "Ci troviamo alla stazione alle sette per andare a cena nel nuovo ristorante.",
		},
		CorrectAnswer: 3, // "station" is correct
		Explanation:   "Stazione means station in Italian.",
		Status:        models.QuestionStatusActive,
	}

	// Use the service to save the question
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, logger)

	err := questionService.SaveQuestion(context.Background(), vocabularyQuestion)
	suite.Require().NoError(err)

	// Login
	cookie := suite.login()

	// Create chat request with vocabulary question
	chatReq := api.QuizChatRequest{
		UserMessage: "Translate this question, text and options to English",
		Question: api.Question{
			Language: languagePtr(api.Language("italian")),
			Level:    levelPtr(api.Level("B1")),
			Type:     questionTypePtr(api.Vocabulary),
			Content: &api.QuestionContent{
				Question: "What does stazione mean in this context?",
				Options:  []string{"bank", "park", "shop", "station"},
				Sentence: stringPtr("Ci troviamo alla stazione alle sette per andare a cena nel nuovo ristorante."),
			},
		},
	}

	reqBody, _ := json.Marshal(chatReq)
	req, _ := http.NewRequest("POST", "/v1/quiz/chat/stream", bytes.NewBuffer(reqBody))
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")

	w := newCloseNotifierRecorder()
	suite.Router.ServeHTTP(w, req)

	// For streaming endpoints, we expect 200 OK
	assert.Equal(suite.T(), http.StatusOK, w.Code, "Expected successful processing, got status: %d", w.Code)
}

func (suite *QuizIntegrationTestSuite) TestChatStream_ReadingComprehensionWithPassage() {
	// Create a reading comprehension question with a passage field
	readingQuestion := &models.Question{
		Type:            models.ReadingComprehension,
		Language:        "italian",
		Level:           "B1",
		TopicCategory:   "test",
		DifficultyScore: 1.0,
		Content: map[string]interface{}{
			"question": "What is the main topic of this passage?",
			"options":  []string{"Travel", "Food", "Work", "Education"},
			"passage":  "Il viaggio in Italia è sempre un'esperienza meravigliosa. La cultura, il cibo e la gente rendono ogni visita speciale.",
		},
		CorrectAnswer: 0, // "Travel" is correct
		Explanation:   "The passage discusses travel in Italy.",
		Status:        models.QuestionStatusActive,
	}

	// Use the service to save the question
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, logger)

	err := questionService.SaveQuestion(context.Background(), readingQuestion)
	suite.Require().NoError(err)

	// Login
	cookie := suite.login()

	// Create chat request with reading comprehension question
	chatReq := api.QuizChatRequest{
		UserMessage: "What is this passage about?",
		Question: api.Question{
			Language: languagePtr(api.Language("italian")),
			Level:    levelPtr(api.Level("B1")),
			Type:     questionTypePtr(api.ReadingComprehension),
			Content: &api.QuestionContent{
				Question: "What is the main topic of this passage?",
				Options:  []string{"Travel", "Food", "Work", "Education"},
				Passage:  stringPtr("Il viaggio in Italia è sempre un'esperienza meravigliosa. La cultura, il cibo e la gente rendono ogni visita speciale."),
			},
		},
	}

	reqBody, _ := json.Marshal(chatReq)
	req, _ := http.NewRequest("POST", "/v1/quiz/chat/stream", bytes.NewBuffer(reqBody))
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")

	w := newCloseNotifierRecorder()
	suite.Router.ServeHTTP(w, req)

	// For streaming endpoints, we expect 200 OK
	assert.Equal(suite.T(), http.StatusOK, w.Code, "Expected successful processing, got status: %d", w.Code)
}

func (suite *QuizIntegrationTestSuite) TestChatStream_QuestionWithoutPassage() {
	// Create a question without passage/sentence (like fill-in-blank)
	question := &models.Question{
		Type:            models.FillInBlank,
		Language:        "italian",
		Level:           "B1",
		TopicCategory:   "test",
		DifficultyScore: 1.0,
		Content: map[string]interface{}{
			"question": "Complete the sentence: Io ___ studente.",
			"options":  []string{"sono", "sei", "è", "siamo"},
		},
		CorrectAnswer: 0, // "sono" is correct
		Explanation:   "Io sono means 'I am' in Italian.",
		Status:        models.QuestionStatusActive,
	}

	// Use the service to save the question
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	learningService := services.NewLearningServiceWithLogger(suite.db, suite.cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(suite.db, learningService, suite.cfg, logger)

	err := questionService.SaveQuestion(context.Background(), question)
	suite.Require().NoError(err)

	// Login
	cookie := suite.login()

	// Create chat request without passage/sentence
	chatReq := api.QuizChatRequest{
		UserMessage: "Help me understand this grammar question",
		Question: api.Question{
			Language: languagePtr(api.Language("italian")),
			Level:    levelPtr(api.Level("B1")),
			Type:     questionTypePtr(api.FillBlank),
			Content: &api.QuestionContent{
				Question: "Complete the sentence: Io ___ studente.",
				Options:  []string{"sono", "sei", "è", "siamo"},
			},
		},
	}

	reqBody, _ := json.Marshal(chatReq)
	req, _ := http.NewRequest("POST", "/v1/quiz/chat/stream", bytes.NewBuffer(reqBody))
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")

	w := newCloseNotifierRecorder()
	suite.Router.ServeHTTP(w, req)

	// For streaming endpoints, we expect 200 OK
	assert.Equal(suite.T(), http.StatusOK, w.Code, "Expected successful processing, got status: %d", w.Code)
}

func TestQuizIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(QuizIntegrationTestSuite))
}
