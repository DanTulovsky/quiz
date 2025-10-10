//go:build integration
// +build integration

package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/database"
	"quizapp/internal/handlers"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type DailyQuestionHandlerIntegrationTestSuite struct {
	suite.Suite
	Router               *gin.Engine
	db                   *sql.DB
	testUser             *models.User
	userService          *services.UserService
	dailyQuestionService *services.DailyQuestionService
	cfg                  *config.Config
	testQuestions        []models.Question
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) SetupSuite() {
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
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)
	suite.dailyQuestionService = dailyQuestionService
	aiService := services.NewAIService(cfg, logger)
	workerService := services.NewWorkerServiceWithLogger(db, logger)
	oauthService := services.NewOAuthServiceWithLogger(cfg, logger)

	// Create router with daily question service
	generationHintService := services.NewGenerationHintService(db, logger)
	storyService := services.NewStoryService(db, cfg, logger)
	router := handlers.NewRouter(cfg, userService, questionService, learningService, aiService, workerService, dailyQuestionService, storyService, services.NewConversationService(db), oauthService, generationHintService, logger)
	suite.Router = router
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) SetupTest() {
	// Clean up database before each test using the shared cleanup function
	services.CleanupTestDatabase(suite.db, suite.T())

	// Create test user with password
	user, err := suite.userService.CreateUserWithPassword(context.Background(), "daily_test_user", "password123", "italian", "B1")
	suite.Require().NoError(err)
	suite.Require().NotNil(user)

	// Update the user with all required fields that the validation middleware expects
	_, err = suite.db.Exec(`
		UPDATE users
		SET email = $1, timezone = $2, ai_provider = $3, ai_model = $4, last_active = $5
		WHERE id = $6
	`, "daily_test_user@example.com", "UTC", "ollama", "llama3", time.Now(), user.ID)
	suite.Require().NoError(err)

	// Reload the user to get the updated data
	user, err = suite.userService.GetUserByID(context.Background(), user.ID)
	suite.Require().NoError(err)
	suite.testUser = user

	// Create test questions
	suite.createTestQuestions()
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TearDownTest() {
	// Clean up test data
	if suite.testUser != nil {
		suite.userService.DeleteUser(context.Background(), suite.testUser.ID)
	}
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) createTestQuestions() {
	// Create 20 test questions to ensure we can assign 10 (5 of each type)
	suite.testQuestions = make([]models.Question, 0, 20)

	// Define the question types that the system expects
	questionTypes := []models.QuestionType{
		models.Vocabulary,
		models.FillInBlank,
		models.QuestionAnswer,
		models.ReadingComprehension,
	}

	for i := 0; i < 20; i++ {
		// Cycle through the question types
		questionType := questionTypes[i%len(questionTypes)]

		question := models.Question{
			Type:               questionType,
			Language:           "italian",
			Level:              "B1",
			DifficultyScore:    0.5,
			Content:            map[string]interface{}{"question": fmt.Sprintf("Test question %d", i+1), "options": []string{"A", "B", "C", "D"}},
			CorrectAnswer:      i % 4,
			Explanation:        fmt.Sprintf("Test explanation %d", i+1),
			Status:             models.QuestionStatusActive,
			TopicCategory:      "grammar",
			GrammarFocus:       "present_tense",
			VocabularyDomain:   "daily_life",
			Scenario:           "conversation",
			StyleModifier:      "formal",
			DifficultyModifier: "medium",
			TimeContext:        "present",
		}

		// Insert question into database
		insertQuery := `
			INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation, status,
								 topic_category, grammar_focus, vocabulary_domain, scenario, style_modifier,
								 difficulty_modifier, time_context, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW())
			RETURNING id, created_at`

		contentJSON, _ := json.Marshal(question.Content)
		err := suite.db.QueryRow(insertQuery,
			question.Type, question.Language, question.Level, question.DifficultyScore,
			contentJSON, question.CorrectAnswer, question.Explanation, question.Status,
			question.TopicCategory, question.GrammarFocus, question.VocabularyDomain,
			question.Scenario, question.StyleModifier, question.DifficultyModifier,
			question.TimeContext,
		).Scan(&question.ID, &question.CreatedAt)

		suite.Require().NoError(err)
		suite.testQuestions = append(suite.testQuestions, question)

		// Also assign the question to the test user in user_questions table
		assignQuery := `
			INSERT INTO user_questions (user_id, question_id, created_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (user_id, question_id) DO NOTHING`
		_, err = suite.db.Exec(assignQuery, suite.testUser.ID, question.ID)
		suite.Require().NoError(err)
	}
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) loginUser() *http.Cookie {
	// Login the test user
	loginReq := api.LoginRequest{
		Username: "daily_test_user",
		Password: "password123",
	}
	reqBody, _ := json.Marshal(loginReq)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	suite.Router.ServeHTTP(w, req)
	suite.T().Logf("Login response status: %d", w.Code)
	suite.T().Logf("Login response body: %s", w.Body.String())
	suite.T().Logf("Login response headers: %v", w.Header())
	suite.Require().Equal(http.StatusOK, w.Code)

	// Extract session cookie
	cookies := w.Result().Cookies()
	suite.T().Logf("Number of cookies returned: %d", len(cookies))
	for _, cookie := range cookies {
		suite.T().Logf("Cookie: %s = %s", cookie.Name, cookie.Value)
		if cookie.Name == config.SessionName {
			return cookie
		}
	}
	suite.Fail("Session cookie not found")
	return nil
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetDailyQuestions_Success() {
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// First assign daily questions
	err := suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	// Test GET /v1/daily/questions/{date}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/questions/%s", today.Format("2006-01-02")), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response struct {
		Questions []api.DailyQuestionWithDetails `json:"questions"`
		Date      string                         `json:"date"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	suite.Equal(today.Format("2006-01-02"), response.Date)
	suite.Equal(10, len(response.Questions)) // Should have 10 daily questions

	// Verify question structure
	for _, q := range response.Questions {
		suite.NotZero(q.Id)
		suite.Equal(int64(suite.testUser.ID), q.UserId)
		suite.NotZero(q.QuestionId)
		// assignment_date is an openapi_types.Date (date-only). Use its Time field.
		assignDate := q.AssignmentDate.Time
		suite.Equal(today.Format("2006-01-02"), assignDate.Format("2006-01-02"))
		suite.False(q.IsCompleted) // Should start as incomplete
		suite.NotNil(q.Question)
		suite.Equal(api.Language("italian"), *q.Question.Language)
		suite.Equal(api.Level("B1"), *q.Question.Level)
	}
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetDailyQuestions_InvalidDate() {
	cookie := suite.loginUser()

	// Test with invalid date format
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/daily/questions/invalid-date", nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Equal("INVALID_FORMAT", response["code"])
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetDailyQuestions_Unauthorized() {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Test without authentication
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/questions/%s", today.Format("2006-01-02")), nil)

	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestMarkQuestionCompleted_Success() {
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// First assign daily questions
	err := suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	// Get the first question ID
	questions, err := suite.dailyQuestionService.GetDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)
	suite.Require().Greater(len(questions), 0)

	questionID := questions[0].QuestionID

	// Test POST /v1/daily/questions/{date}/complete/{questionId}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", fmt.Sprintf("/v1/daily/questions/%s/complete/%d", today.Format("2006-01-02"), questionID), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response api.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Contains(*response.Message, "completed")

	// Verify the question is marked as completed
	updatedQuestions, err := suite.dailyQuestionService.GetDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	var found bool
	for _, q := range updatedQuestions {
		if q.QuestionID == questionID {
			suite.True(q.IsCompleted)
			suite.NotNil(q.CompletedAt)
			found = true
			break
		}
	}
	suite.True(found, "Question should be found and marked as completed")
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestMarkQuestionCompleted_InvalidQuestionID() {
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Test with invalid question ID
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", fmt.Sprintf("/v1/daily/questions/%s/complete/invalid", today.Format("2006-01-02")), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Equal("INVALID_FORMAT", response["code"])
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestResetQuestionCompleted_Success() {
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// First assign daily questions
	err := suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	// Get the first question ID and mark it as completed
	questions, err := suite.dailyQuestionService.GetDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)
	suite.Require().Greater(len(questions), 0)

	questionID := questions[0].QuestionID
	err = suite.dailyQuestionService.MarkQuestionCompleted(context.Background(), suite.testUser.ID, questionID, today)
	suite.Require().NoError(err)

	// Test DELETE /v1/daily/questions/{date}/complete/{questionId}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/v1/daily/questions/%s/complete/%d", today.Format("2006-01-02"), questionID), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response api.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Contains(*response.Message, "reset")

	// Verify the question is no longer marked as completed
	updatedQuestions, err := suite.dailyQuestionService.GetDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	var found bool
	for _, q := range updatedQuestions {
		if q.QuestionID == questionID {
			suite.False(q.IsCompleted)
			found = true
			break
		}
	}
	suite.True(found, "Question should be found and no longer marked as completed")
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetAvailableDates_Success() {
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)

	// Assign questions for two different dates
	err := suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)
	err = suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, yesterday)
	suite.Require().NoError(err)

	// Test GET /v1/daily/dates
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/daily/dates", nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response struct {
		Dates []string `json:"dates"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	suite.Equal(2, len(response.Dates))
	suite.Contains(response.Dates, today.Format("2006-01-02"))
	suite.Contains(response.Dates, yesterday.Format("2006-01-02"))
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetDailyProgress_Success() {
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// First assign daily questions
	err := suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	// Get questions and mark some as completed
	questions, err := suite.dailyQuestionService.GetDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)
	suite.Require().Equal(10, len(questions))

	// Mark first 3 questions as completed
	for i := 0; i < 3; i++ {
		err = suite.dailyQuestionService.MarkQuestionCompleted(context.Background(), suite.testUser.ID, questions[i].QuestionID, today)
		suite.Require().NoError(err)
	}

	// Test GET /v1/daily/progress/{date}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/progress/%s", today.Format("2006-01-02")), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var progress api.DailyProgress
	err = json.Unmarshal(w.Body.Bytes(), &progress)
	suite.Require().NoError(err)

	suite.Equal(today.Format("2006-01-02"), progress.Date.Format("2006-01-02"))
	suite.Equal(3, progress.Completed) // 3 completed questions
	suite.Equal(10, progress.Total)    // 10 total questions
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetDailyProgress_NoAssignments() {
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Test without any assignments
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/progress/%s", today.Format("2006-01-02")), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var progress api.DailyProgress
	err := json.Unmarshal(w.Body.Bytes(), &progress)
	suite.Require().NoError(err)

	suite.Equal(today.Format("2006-01-02"), progress.Date.Format("2006-01-02"))
	suite.Equal(0, progress.Completed) // 0 completed questions
	suite.Equal(0, progress.Total)     // 0 total questions
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestWorkflowIntegration() {
	// Test the complete workflow: assign -> get -> complete -> check progress -> reset
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// 1. Assign daily questions (simulating worker behavior)
	err := suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	// 2. Get daily questions
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/questions/%s", today.Format("2006-01-02")), nil)
	req.AddCookie(cookie)
	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	var questionsResponse struct {
		Questions []api.DailyQuestionWithDetails `json:"questions"`
		Date      string                         `json:"date"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &questionsResponse)
	suite.Require().NoError(err)
	suite.Equal(10, len(questionsResponse.Questions))

	// 3. Mark first question as completed
	questionID := questionsResponse.Questions[0].QuestionId
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", fmt.Sprintf("/v1/daily/questions/%s/complete/%d", today.Format("2006-01-02"), questionID), nil)
	req.AddCookie(cookie)
	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	// 4. Check progress
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/progress/%s", today.Format("2006-01-02")), nil)
	req.AddCookie(cookie)
	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	var progress api.DailyProgress
	err = json.Unmarshal(w.Body.Bytes(), &progress)
	suite.Require().NoError(err)
	suite.Equal(1, progress.Completed)
	suite.Equal(10, progress.Total)

	// 5. Reset the completed question
	w = httptest.NewRecorder()
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/v1/daily/questions/%s/complete/%d", today.Format("2006-01-02"), questionID), nil)
	req.AddCookie(cookie)
	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	// 6. Verify progress is back to 0 completed
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/progress/%s", today.Format("2006-01-02")), nil)
	req.AddCookie(cookie)
	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &progress)
	suite.Require().NoError(err)
	suite.Equal(0, progress.Completed)
	suite.Equal(10, progress.Total)
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestSubmitDailyQuestionAnswer_Success() {
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Assign daily questions
	err := suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	// Get the first question ID
	questions, err := suite.dailyQuestionService.GetDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)
	suite.Require().Greater(len(questions), 0)
	questionID := questions[0].QuestionID

	// Prepare request body
	requestBody := struct {
		UserAnswerIndex int `json:"user_answer_index"`
	}{
		UserAnswerIndex: 0,
	}
	body, _ := json.Marshal(requestBody)

	// POST /v1/daily/questions/{date}/answer/{questionId}
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/v1/daily/questions/%s/answer/%d", today.Format("2006-01-02"), questionID)
	req := httptest.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code, "Expected 200 OK for valid answer submission")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Contains(response, "user_answer_index")
	suite.Equal(float64(0), response["user_answer_index"])
	suite.Contains(response, "is_completed")
	suite.True(response["is_completed"].(bool))
}

func TestDailyQuestionHandlerIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DailyQuestionHandlerIntegrationTestSuite))
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetDailyQuestions_WithCompletedQuestions_IncludesUserAnswerData() {
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// First assign daily questions
	err := suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	// Get the first question ID
	questions, err := suite.dailyQuestionService.GetDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)
	suite.Require().Greater(len(questions), 0)
	questionID := questions[0].QuestionID

	// Submit an answer to the first question
	requestBody := struct {
		UserAnswerIndex int `json:"user_answer_index"`
	}{
		UserAnswerIndex: 2, // Choose option 2
	}
	body, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	url := fmt.Sprintf("/v1/daily/questions/%s/answer/%d", today.Format("2006-01-02"), questionID)
	req := httptest.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	// Now get the questions again and verify the completed question includes user answer data
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/questions/%s", today.Format("2006-01-02")), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	var response struct {
		Questions []api.DailyQuestionWithDetails `json:"questions"`
		Date      string                         `json:"date"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	// Find the completed question
	var completedQuestion *api.DailyQuestionWithDetails
	for i := range response.Questions {
		if response.Questions[i].QuestionId == int64(questionID) {
			completedQuestion = &response.Questions[i]
			break
		}
	}

	suite.Require().NotNil(completedQuestion, "Should find the completed question")
	suite.True(completedQuestion.IsCompleted, "Question should be marked as completed")
	suite.NotNil(completedQuestion.UserAnswerIndex, "UserAnswerIndex should not be null")
	suite.Equal(2, *completedQuestion.UserAnswerIndex, "UserAnswerIndex should match the submitted answer")
	suite.NotNil(completedQuestion.SubmittedAt, "SubmittedAt should not be null")
	subAt, perr := time.Parse(time.RFC3339, *completedQuestion.SubmittedAt)
	suite.Require().NoError(perr)
	suite.True(subAt.After(today), "SubmittedAt should be after the assignment date")
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetDailyQuestions_WithMultipleCompletedQuestions_AllIncludeUserAnswerData() {
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// First assign daily questions
	err := suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	// Get all questions
	questions, err := suite.dailyQuestionService.GetDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)
	suite.Require().Greater(len(questions), 0)

	// Submit answers to the first 3 questions
	for i := 0; i < 3 && i < len(questions); i++ {
		questionID := questions[i].QuestionID
		userAnswerIndex := i % 4 // Use different answers for variety

		requestBody := struct {
			UserAnswerIndex int `json:"user_answer_index"`
		}{
			UserAnswerIndex: userAnswerIndex,
		}
		body, _ := json.Marshal(requestBody)

		w := httptest.NewRecorder()
		url := fmt.Sprintf("/v1/daily/questions/%s/answer/%d", today.Format("2006-01-02"), questionID)
		req := httptest.NewRequest("POST", url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)

		suite.Router.ServeHTTP(w, req)
		suite.Equal(http.StatusOK, w.Code)
	}

	// Now get the questions again and verify all completed questions include user answer data
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/questions/%s", today.Format("2006-01-02")), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	var response struct {
		Questions []api.DailyQuestionWithDetails `json:"questions"`
		Date      string                         `json:"date"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	completedCount := 0
	for _, question := range response.Questions {
		if question.IsCompleted {
			completedCount++
			suite.NotNil(question.UserAnswerIndex, "Completed question should have UserAnswerIndex")
			suite.NotNil(question.SubmittedAt, "Completed question should have SubmittedAt")
			subAt, perr := time.Parse(time.RFC3339, *question.SubmittedAt)
			suite.Require().NoError(perr)
			suite.True(subAt.After(today), "SubmittedAt should be after the assignment date")
			suite.GreaterOrEqual(*question.UserAnswerIndex, 0, "UserAnswerIndex should be >= 0")
			suite.Less(*question.UserAnswerIndex, 4, "UserAnswerIndex should be < 4 (number of options)")
		} else {
			suite.Nil(question.UserAnswerIndex, "Incomplete question should have null UserAnswerIndex")
			suite.Nil(question.SubmittedAt, "Incomplete question should have null SubmittedAt")
		}
	}

	suite.Equal(3, completedCount, "Should have exactly 3 completed questions")
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetDailyQuestions_WithResetCompletedQuestion_UserAnswerDataIsCleared() {
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// First assign daily questions
	err := suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	// Get the first question ID
	questions, err := suite.dailyQuestionService.GetDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)
	suite.Require().Greater(len(questions), 0)
	questionID := questions[0].QuestionID

	// Submit an answer to the first question
	requestBody := struct {
		UserAnswerIndex int `json:"user_answer_index"`
	}{
		UserAnswerIndex: 1,
	}
	body, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	url := fmt.Sprintf("/v1/daily/questions/%s/answer/%d", today.Format("2006-01-02"), questionID)
	req := httptest.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	// Verify the question is completed with user answer data
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/questions/%s", today.Format("2006-01-02")), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	var response struct {
		Questions []api.DailyQuestionWithDetails `json:"questions"`
		Date      string                         `json:"date"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	// Find the completed question
	var completedQuestion *api.DailyQuestionWithDetails
	for i := range response.Questions {
		if response.Questions[i].QuestionId == int64(questionID) {
			completedQuestion = &response.Questions[i]
			break
		}
	}

	suite.Require().NotNil(completedQuestion, "Should find the completed question")
	suite.True(completedQuestion.IsCompleted, "Question should be marked as completed")
	suite.NotNil(completedQuestion.UserAnswerIndex, "UserAnswerIndex should not be null before reset")
	suite.NotNil(completedQuestion.SubmittedAt, "SubmittedAt should not be null before reset")

	// Now reset the question
	w = httptest.NewRecorder()
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/v1/daily/questions/%s/complete/%d", today.Format("2006-01-02"), questionID), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	// Verify the question is no longer completed and user answer data is cleared
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/questions/%s", today.Format("2006-01-02")), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	// Find the reset question
	var resetQuestion *api.DailyQuestionWithDetails
	for i := range response.Questions {
		if response.Questions[i].QuestionId == int64(questionID) {
			resetQuestion = &response.Questions[i]
			break
		}
	}

	suite.Require().NotNil(resetQuestion, "Should find the reset question")
	suite.False(resetQuestion.IsCompleted, "Question should no longer be marked as completed")
	suite.Nil(resetQuestion.UserAnswerIndex, "UserAnswerIndex should be null after reset")
	suite.Nil(resetQuestion.SubmittedAt, "SubmittedAt should be null after reset")
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetDailyQuestions_WithMixedCompletedAndIncompleteQuestions() {
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// First assign daily questions
	err := suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	// Get all questions
	questions, err := suite.dailyQuestionService.GetDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)
	suite.Require().Greater(len(questions), 0)

	// Submit answers to only the first 2 questions (leave others incomplete)
	for i := 0; i < 2 && i < len(questions); i++ {
		questionID := questions[i].QuestionID
		userAnswerIndex := i % 4

		requestBody := struct {
			UserAnswerIndex int `json:"user_answer_index"`
		}{
			UserAnswerIndex: userAnswerIndex,
		}
		body, _ := json.Marshal(requestBody)

		w := httptest.NewRecorder()
		url := fmt.Sprintf("/v1/daily/questions/%s/answer/%d", today.Format("2006-01-02"), questionID)
		req := httptest.NewRequest("POST", url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)

		suite.Router.ServeHTTP(w, req)
		suite.Equal(http.StatusOK, w.Code)
	}

	// Now get the questions again and verify the mixed state
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/questions/%s", today.Format("2006-01-02")), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	var response struct {
		Questions []api.DailyQuestionWithDetails `json:"questions"`
		Date      string                         `json:"date"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	completedCount := 0
	incompleteCount := 0

	for _, question := range response.Questions {
		if question.IsCompleted {
			completedCount++
			suite.NotNil(question.UserAnswerIndex, "Completed question should have UserAnswerIndex")
			suite.NotNil(question.SubmittedAt, "Completed question should have SubmittedAt")
			subAt, perr := time.Parse(time.RFC3339, *question.SubmittedAt)
			suite.Require().NoError(perr)
			suite.True(subAt.After(today), "SubmittedAt should be after the assignment date")
		} else {
			incompleteCount++
			suite.Nil(question.UserAnswerIndex, "Incomplete question should have null UserAnswerIndex")
			suite.Nil(question.SubmittedAt, "Incomplete question should have null SubmittedAt")
		}
	}

	suite.Equal(2, completedCount, "Should have exactly 2 completed questions")
	suite.Equal(8, incompleteCount, "Should have exactly 8 incomplete questions")
	suite.Equal(10, len(response.Questions), "Should have exactly 10 total questions")
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetDailyQuestions_IncludesCorrectAnswerField() {
	cookie := suite.loginUser()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// First assign daily questions
	err := suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	// Get the questions
	questions, err := suite.dailyQuestionService.GetDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(questions)

	// Make API request
	req := httptest.NewRequest("GET", "/v1/daily/questions/"+today.Format("2006-01-02"), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	suite.Router.ServeHTTP(w, req)

	// Verify response
	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	questionsData, ok := response["questions"].([]interface{})
	suite.Require().True(ok, "Response should contain questions array")
	suite.Require().NotEmpty(questionsData, "Questions array should not be empty")

	// Check that each question has the correct_answer field
	for i, questionData := range questionsData {
		question, ok := questionData.(map[string]interface{})
		suite.Require().True(ok, "Question should be an object")

		questionObj, ok := question["question"].(map[string]interface{})
		suite.Require().True(ok, "Question should have a question object")

		// Verify that correct_answer field is present and is a number
		correctAnswer, exists := questionObj["correct_answer"]
		suite.Require().True(exists, "Question should have correct_answer field")
		suite.Require().NotNil(correctAnswer, "correct_answer should not be null")

		// Verify it's a number and within valid range (0-3 for 4 options)
		correctAnswerFloat, ok := correctAnswer.(float64)
		suite.Require().True(ok, "correct_answer should be a number")
		suite.Require().GreaterOrEqual(correctAnswerFloat, 0.0, "correct_answer should be >= 0")
		suite.Require().Less(correctAnswerFloat, 4.0, "correct_answer should be < 4 (for 4 options)")

		// Verify it's an integer
		correctAnswerInt := int(correctAnswerFloat)
		suite.Equal(correctAnswerFloat, float64(correctAnswerInt), "correct_answer should be an integer")

		suite.T().Logf("Question %d has correct_answer: %d", i, correctAnswerInt)
	}
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetDailyQuestions_TimezoneAware() {
	// Test with different timezones to ensure date parsing works correctly
	timezones := []string{"UTC", "America/New_York", "Europe/London", "Asia/Tokyo"}

	for _, timezone := range timezones {
		suite.Run(fmt.Sprintf("Timezone_%s", timezone), func() {
			// Update user timezone
			_, err := suite.db.Exec(`
				UPDATE users SET timezone = $1 WHERE id = $2
			`, timezone, suite.testUser.ID)
			suite.Require().NoError(err)

			// Assign daily questions for today in the user's timezone
			loc, err := time.LoadLocation(timezone)
			suite.Require().NoError(err)

			today := time.Now().In(loc)
			todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)

			err = suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, todayDate)
			suite.Require().NoError(err)

			// Login user
			cookie := suite.loginUser()

			// Get daily questions using the date string (frontend format)
			dateStr := todayDate.Format("2006-01-02")
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/questions/%s", dateStr), nil)
			req.AddCookie(cookie)

			suite.Router.ServeHTTP(w, req)

			suite.Equal(http.StatusOK, w.Code)

			var response struct {
				Questions []api.DailyQuestionWithDetails `json:"questions"`
				Date      string                         `json:"date"`
			}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			suite.Require().NoError(err)

			// Should have questions for this date
			suite.Greater(len(response.Questions), 0, "Should have questions for timezone %s", timezone)
			suite.Equal(dateStr, response.Date)
		})
	}
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetDailyQuestions_InvalidTimezone() {
	// Test with invalid timezone - should fallback to UTC
	_, err := suite.db.Exec(`
		UPDATE users SET timezone = $1 WHERE id = $2
	`, "Invalid/Timezone", suite.testUser.ID)
	suite.Require().NoError(err)

	// Assign daily questions for today
	today := time.Now().UTC().Truncate(24 * time.Hour)
	err = suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, today)
	suite.Require().NoError(err)

	// Login user
	cookie := suite.loginUser()

	// Get daily questions
	dateStr := today.Format("2006-01-02")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/questions/%s", dateStr), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)

	// Should still work (fallback to UTC)
	suite.Equal(http.StatusOK, w.Code)

	var response struct {
		Questions []api.DailyQuestionWithDetails `json:"questions"`
		Date      string                         `json:"date"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	suite.Greater(len(response.Questions), 0, "Should have questions even with invalid timezone")
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetDailyQuestions_TimezoneDateMismatch() {
	// Test that dates are parsed correctly in user's timezone
	// Set user to a timezone ahead of UTC
	_, err := suite.db.Exec(`
		UPDATE users SET timezone = $1 WHERE id = $2
	`, "Asia/Tokyo", suite.testUser.ID)
	suite.Require().NoError(err)

	// Get Tokyo timezone
	tokyoLoc, err := time.LoadLocation("Asia/Tokyo")
	suite.Require().NoError(err)

	// Create a date that would be different in Tokyo vs UTC
	// For example, if it's 11 PM UTC on Aug 6, it's already Aug 7 in Tokyo
	tokyoTime := time.Now().In(tokyoLoc)
	tokyoDate := time.Date(tokyoTime.Year(), tokyoTime.Month(), tokyoTime.Day(), 0, 0, 0, 0, tokyoLoc)

	// Assign questions for Tokyo's "today"
	err = suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, tokyoDate)
	suite.Require().NoError(err)

	// Login user
	cookie := suite.loginUser()

	// Request the date in Tokyo's timezone
	dateStr := tokyoDate.Format("2006-01-02")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/questions/%s", dateStr), nil)
	req.AddCookie(cookie)

	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response struct {
		Questions []api.DailyQuestionWithDetails `json:"questions"`
		Date      string                         `json:"date"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	// Should have questions for the Tokyo date
	suite.Greater(len(response.Questions), 0, "Should have questions for Tokyo timezone date")
	suite.Equal(dateStr, response.Date)
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestMarkQuestionCompleted_TimezoneAware() {
	// Test marking questions as completed with timezone-aware date parsing
	timezones := []string{"UTC", "America/New_York", "Europe/London"}

	for _, timezone := range timezones {
		suite.Run(fmt.Sprintf("Timezone_%s", timezone), func() {
			// Update user timezone
			_, err := suite.db.Exec(`
				UPDATE users SET timezone = $1 WHERE id = $2
			`, timezone, suite.testUser.ID)
			suite.Require().NoError(err)

			// Assign daily questions for today in the user's timezone
			loc, err := time.LoadLocation(timezone)
			suite.Require().NoError(err)

			today := time.Now().In(loc)
			todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)

			err = suite.dailyQuestionService.AssignDailyQuestions(context.Background(), suite.testUser.ID, todayDate)
			suite.Require().NoError(err)

			// Get questions to find one to complete
			questions, err := suite.dailyQuestionService.GetDailyQuestions(context.Background(), suite.testUser.ID, todayDate)
			suite.Require().NoError(err)
			suite.Require().Greater(len(questions), 0)

			questionID := questions[0].QuestionID

			// Login user
			cookie := suite.loginUser()

			// Mark question as completed using the date string
			dateStr := todayDate.Format("2006-01-02")
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", fmt.Sprintf("/v1/daily/questions/%s/complete/%d", dateStr, questionID), nil)
			req.AddCookie(cookie)

			suite.Router.ServeHTTP(w, req)

			suite.Equal(http.StatusOK, w.Code)

			// Verify the question is now completed
			questions, err = suite.dailyQuestionService.GetDailyQuestions(context.Background(), suite.testUser.ID, todayDate)
			suite.Require().NoError(err)

			found := false
			for _, q := range questions {
				if q.QuestionID == questionID {
					suite.True(q.IsCompleted, "Question should be completed in timezone %s", timezone)
					found = true
					break
				}
			}
			suite.True(found, "Should find the completed question")
		})
	}
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestParseDateInUserTimezone() {
	// Test the helper function directly
	handler := handlers.NewDailyQuestionHandler(
		suite.userService,
		suite.dailyQuestionService,
		suite.cfg,
		observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}),
	)

	// Test with different timezones
	testCases := []struct {
		timezone string
		dateStr  string
		expected time.Time
	}{
		{
			timezone: "UTC",
			dateStr:  "2024-08-06",
			expected: time.Date(2024, 8, 6, 0, 0, 0, 0, time.UTC),
		},
		{
			timezone: "America/New_York",
			dateStr:  "2024-08-06",
			expected: time.Date(2024, 8, 6, 0, 0, 0, 0, time.UTC), // Should be UTC in database
		},
		{
			timezone: "Asia/Tokyo",
			dateStr:  "2024-08-06",
			expected: time.Date(2024, 8, 6, 0, 0, 0, 0, time.UTC), // Should be UTC in database
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Timezone_%s", tc.timezone), func() {
			// Update user timezone
			_, err := suite.db.Exec(`
				UPDATE users SET timezone = $1 WHERE id = $2
			`, tc.timezone, suite.testUser.ID)
			suite.Require().NoError(err)

			// Test the helper function
			date, timezone, err := handler.ParseDateInUserTimezone(context.Background(), suite.testUser.ID, tc.dateStr)
			suite.Require().NoError(err)
			suite.Equal(tc.timezone, timezone)

			// The date should be parsed correctly in the user's timezone
			// but stored as UTC in the database
			suite.Equal(tc.expected.Year(), date.Year())
			suite.Equal(tc.expected.Month(), date.Month())
			suite.Equal(tc.expected.Day(), date.Day())
		})
	}

	// Test invalid date format
	_, _, err := handler.ParseDateInUserTimezone(context.Background(), suite.testUser.ID, "invalid-date")
	suite.Require().Error(err)
	suite.Contains(err.Error(), "invalid date format")
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestParseDateInUserTimezone_InvalidTimezone() {
	handler := handlers.NewDailyQuestionHandler(
		suite.userService,
		suite.dailyQuestionService,
		suite.cfg,
		observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}),
	)

	// Set invalid timezone
	_, err := suite.db.Exec(`
		UPDATE users SET timezone = $1 WHERE id = $2
	`, "Invalid/Timezone", suite.testUser.ID)
	suite.Require().NoError(err)

	// Should still work (fallback to UTC)
	date, timezone, err := handler.ParseDateInUserTimezone(context.Background(), suite.testUser.ID, "2024-08-06")
	suite.Require().NoError(err)
	suite.Equal("UTC", timezone) // Should fallback to UTC
	suite.Equal(2024, date.Year())
	suite.Equal(time.August, date.Month())
	suite.Equal(6, date.Day())
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetQuestionHistory() {
	// Create a test question and assign it to the user for multiple dates
	// Use the first question from the existing test questions
	suite.Require().NotEmpty(suite.testQuestions, "Test questions should be created")
	question := suite.testQuestions[0]

	// Assign the question for the last 3 days
	today := time.Now().UTC().Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)
	twoDaysAgo := today.Add(-48 * time.Hour)

	dates := []time.Time{today, yesterday, twoDaysAgo}

	for i, date := range dates {
		// Create assignment
		_, err := suite.db.Exec(`
			INSERT INTO daily_question_assignments (user_id, question_id, assignment_date, is_completed, user_answer_index, submitted_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, suite.testUser.ID, question.ID, date, i == 0, i, time.Now()) // Only today's is completed
		suite.Require().NoError(err)
	}

	// Login and get session
	cookie := suite.loginUser()

	// Test getting question history
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/daily/history/%d", question.ID), nil)
	req.AddCookie(cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	// Verify response structure
	suite.Contains(response, "history")
	history := response["history"].([]interface{})
	suite.Len(history, 3, "Should return history for 3 days")

	// Verify the history entries
	for i, entry := range history {
		entryMap := entry.(map[string]interface{})
		suite.Contains(entryMap, "assignment_date")
		suite.Contains(entryMap, "is_completed")
		suite.Contains(entryMap, "is_correct")

		// Check that today's entry is completed, others are not
		// Note: Results are ordered by assignment_date ASC, so:
		// i=0: two days ago (not completed)
		// i=1: yesterday (not completed)
		// i=2: today (completed)
		if i == 2 {
			suite.True(entryMap["is_completed"].(bool), "Today's question should be completed")
		} else {
			suite.False(entryMap["is_completed"].(bool), "Previous days should not be completed")
		}
	}
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetQuestionHistory_ExcludesFutureDates() {
	// Ensure future-dated assignments are not returned in history
	suite.Require().NotEmpty(suite.testQuestions, "Test questions should be created")
	question := suite.testQuestions[0]

	// Create assignment for tomorrow and today
	today := time.Now().UTC().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	// Insert tomorrow assignment
	_, err := suite.db.Exec(`
        INSERT INTO daily_question_assignments (user_id, question_id, assignment_date, is_completed)
        VALUES ($1, $2, $3, false)
    `, suite.testUser.ID, question.ID, tomorrow)
	suite.Require().NoError(err)

	// Insert today assignment
	_, err = suite.db.Exec(`
        INSERT INTO daily_question_assignments (user_id, question_id, assignment_date, is_completed)
        VALUES ($1, $2, $3, false)
    `, suite.testUser.ID, question.ID, today)
	suite.Require().NoError(err)

	// Login and request history
	cookie := suite.loginUser()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/daily/history/%d", question.ID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	historyIface, ok := response["history"]
	suite.Require().True(ok, "history key present in response")
	historyArr := historyIface.([]interface{})

	tomorrowStr := tomorrow.Format("2006-01-02")
	for _, item := range historyArr {
		entry := item.(map[string]interface{})
		if ad, ok := entry["assignment_date"].(string); ok {
			// Assert none of the returned entries equal the future date
			suite.NotEqual(tomorrowStr, ad, "history should not include future date %s", tomorrowStr)
		}
	}
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetQuestionHistory_Unauthenticated() {
	// Use the first question from the existing test questions
	suite.Require().NotEmpty(suite.testQuestions, "Test questions should be created")
	question := suite.testQuestions[0]

	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/daily/history/%d", question.ID), nil)
	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *DailyQuestionHandlerIntegrationTestSuite) TestGetQuestionHistory_InvalidQuestionID() {
	// Login and get session
	cookie := suite.loginUser()

	req, _ := http.NewRequest("GET", "/v1/daily/history/invalid", nil)
	req.AddCookie(cookie)

	w := httptest.NewRecorder()
	suite.Router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)
}
