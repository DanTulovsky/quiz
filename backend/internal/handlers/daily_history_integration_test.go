//go:build integration
// +build integration

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"quizapp/internal/api"
	"quizapp/internal/config"
	"quizapp/internal/handlers"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/stretchr/testify/require"
)

// This integration test verifies that submitting a daily answer results in a recorded
// user_responses entry which the history endpoint reads so is_correct is present.
func TestSubmitDailyAnswer_RecordsUserResponse(t *testing.T) {
	// Use shared test DB setup helper to get a clean DB per test
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)

	// Initialize logger and services
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)
	generationHintService := services.NewGenerationHintService(db, logger)

	storyService := services.NewStoryService(db, cfg, logger)
	translationService := services.NewTranslationService(cfg)
	router := handlers.NewRouter(cfg, userService, questionService, learningService, services.NewAIService(cfg, logger), services.NewWorkerServiceWithLogger(db, logger), dailyQuestionService, storyService, services.NewConversationService(db), services.NewOAuthServiceWithLogger(cfg, logger), generationHintService, translationService, logger)

	// Create a user
	user, err := userService.CreateUserWithPassword(context.Background(), fmt.Sprintf("daily_integ_%d", time.Now().UnixNano()), "password123", "italian", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Update required fields
	_, err = db.Exec(`UPDATE users SET email = $1, timezone = $2, ai_provider = $3, ai_model = $4, last_active = $5 WHERE id = $6`, "daily_integ@example.com", "UTC", "ollama", "llama3", time.Now(), user.ID)
	require.NoError(t, err)

	// Create a question and assign
	q := &models.Question{
		Type:            "vocabulary",
		Language:        "italian",
		Level:           "A1",
		DifficultyScore: 1.0,
		Content:         map[string]interface{}{"question": "Test?", "options": []string{"A", "B", "C", "D"}},
		CorrectAnswer:   0,
		Explanation:     "",
		Status:          models.QuestionStatusActive,
	}
	require.NoError(t, questionService.SaveQuestion(context.Background(), q))
	require.NoError(t, questionService.AssignQuestionToUser(context.Background(), q.ID, user.ID))

	// Assign daily questions for today at midnight UTC so route parsing of
	// YYYY-MM-DD (which yields midnight UTC) will match the stored value.
	today := time.Now().UTC().Truncate(24 * time.Hour)
	require.NoError(t, dailyQuestionService.AssignDailyQuestions(context.Background(), user.ID, today))

	// Login to get session cookie
	loginReq := api.LoginRequest{Username: user.Username, Password: "password123"}
	body, _ := json.Marshal(loginReq)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == config.SessionName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)

	// Find an assignment question id for today
	assignments, err := dailyQuestionService.GetDailyQuestions(context.Background(), user.ID, today)
	require.NoError(t, err)
	require.NotEmpty(t, assignments)
	questionID := assignments[0].QuestionID

	// Submit an answer
	ansReq := map[string]int{"user_answer_index": 1}
	bodyAns, _ := json.Marshal(ansReq)

	// Dump route listing for debugging
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/?json=true", nil)
	router.ServeHTTP(w2, req2)
	fmt.Printf("[debug test] route listing (status=%d): %s\n", w2.Code, string(w2.Body.Bytes()))

	w = httptest.NewRecorder()
	url := fmt.Sprintf("/v1/daily/questions/%s/answer/%d", today.Format("2006-01-02"), questionID)
	fmt.Printf("[debug test] submitting POST %s cookie=%v\n", url, sessionCookie)
	req = httptest.NewRequest("POST", url, bytes.NewBuffer(bodyAns))
	req.AddCookie(sessionCookie)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Assert mapping row exists linking today's assignment to a user_responses row
	var mapCount int
	err = db.QueryRow("SELECT COUNT(*) FROM daily_assignment_responses dar JOIN daily_question_assignments dqa ON dar.assignment_id = dqa.id WHERE dqa.user_id = $1 AND dqa.question_id = $2", user.ID, q.ID).Scan(&mapCount)
	require.NoError(t, err)
	require.Greater(t, mapCount, 0)

	// Fetch history and assert today's entry has non-null is_correct
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/history/%d", questionID), nil)
	req.AddCookie(sessionCookie)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string][]map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	history := resp["history"]
	require.NotEmpty(t, history)

	found := false
	for _, entry := range history {
		ad, ok := entry["assignment_date"].(string)
		if !ok {
			continue
		}
		var at time.Time
		if t1, err1 := time.Parse(time.RFC3339, ad); err1 == nil {
			at = t1
		} else if t2, err2 := time.Parse("2006-01-02", ad); err2 == nil {
			at = t2
		} else {
			continue
		}
		if at.Format("2006-01-02") == today.Format("2006-01-02") {
			found = true
			// is_correct must be present and not null
			v, exists := entry["is_correct"]
			require.True(t, exists)
			require.NotNil(t, v)
			break
		}
	}
	require.True(t, found, "expected to find today's history entry")
}

// Test that GetQuestionHistory returns 500 if assignment_date in DB appears date-only (missing timezone)
func TestGetQuestionHistory_DateOnlyTimestamp_Returns500(t *testing.T) {
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)
	generationHintService := services.NewGenerationHintService(db, logger)

	storyService := services.NewStoryService(db, cfg, logger)
	translationService := services.NewTranslationService(cfg)
	router := handlers.NewRouter(cfg, userService, questionService, learningService, services.NewAIService(cfg, logger), services.NewWorkerServiceWithLogger(db, logger), dailyQuestionService, storyService, services.NewConversationService(db), services.NewOAuthServiceWithLogger(cfg, logger), generationHintService, translationService, logger)

	// Create a user
	user, err := userService.CreateUserWithPassword(context.Background(), fmt.Sprintf("daily_integ_%d", time.Now().UnixNano()), "password123", "italian", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Update required fields
	_, err = db.Exec(`UPDATE users SET email = $1, timezone = $2, ai_provider = $3, ai_model = $4, last_active = $5 WHERE id = $6`, "daily_integ@example.com", "America/Los_Angeles", "ollama", "llama3", time.Now(), user.ID)
	require.NoError(t, err)

	// Create a question and assign
	q := &models.Question{
		Type:            "vocabulary",
		Language:        "italian",
		Level:           "A1",
		DifficultyScore: 1.0,
		Content:         map[string]interface{}{"question": "Test?", "options": []string{"A", "B", "C", "D"}},
		CorrectAnswer:   0,
		Explanation:     "",
		Status:          models.QuestionStatusActive,
	}
	require.NoError(t, questionService.SaveQuestion(context.Background(), q))
	require.NoError(t, questionService.AssignQuestionToUser(context.Background(), q.ID, user.ID))

	// Insert an assignment row with date-only timestamp (midnight UTC) to simulate missing timezone
	today := time.Now().UTC().Truncate(24 * time.Hour)
	_, err = db.Exec(`INSERT INTO daily_question_assignments (user_id, question_id, assignment_date, created_at) VALUES ($1, $2, $3, $4)`, user.ID, q.ID, today, time.Now())
	require.NoError(t, err)

	// Login to get session cookie
	loginReq := api.LoginRequest{Username: user.Username, Password: "password123"}
	body, _ := json.Marshal(loginReq)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == config.SessionName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie)

	// Call history endpoint
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/daily/history/%d", q.ID), nil)
	req.AddCookie(sessionCookie)
	router.ServeHTTP(w, req)

	// With date-only assignment_date stored as DATE, the handler should return
	// the history successfully and expose the date-only value.
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string][]map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	history := resp["history"]
	require.NotEmpty(t, history)
	foundDate := false
	for _, entry := range history {
		if ad, ok := entry["assignment_date"].(string); ok && ad == today.Format("2006-01-02") {
			foundDate = true
			break
		}
	}
	require.True(t, foundDate, "expected to find date-only assignment_date in history")
}
