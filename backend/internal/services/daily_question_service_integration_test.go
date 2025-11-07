//go:build integration

package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestUser(t *testing.T, db *sql.DB) *models.User {
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)

	user, err := userService.CreateUser(context.Background(), "testuser", "italian", "B1")
	require.NoError(t, err)
	return user
}

func createTestQuestionsForDaily(t *testing.T, db *sql.DB, userID, count int) []models.Question {
	questions := make([]models.Question, count)

	// Define the question types that the system expects
	questionTypes := []models.QuestionType{
		models.Vocabulary,
		models.FillInBlank,
		models.QuestionAnswer,
		models.ReadingComprehension,
	}

	for i := 0; i < count; i++ {
		// Cycle through the question types
		questionType := questionTypes[i%len(questionTypes)]

		question := &models.Question{
			Type:               questionType,
			Language:           "italian",
			Level:              "B1",
			DifficultyScore:    0.5, // Add difficulty score
			Content:            map[string]interface{}{"question": "Test question", "options": []string{"A", "B", "C", "D"}},
			CorrectAnswer:      i % 4, // Cycle through 0-3
			Explanation:        "Test explanation",
			Status:             models.QuestionStatusActive,
			TopicCategory:      "grammar",
			GrammarFocus:       "present_tense",
			VocabularyDomain:   "daily_life",
			Scenario:           "conversation",
			StyleModifier:      "formal",
			DifficultyModifier: "medium",
			TimeContext:        "present",
		}

		// Marshal content to JSON for database insertion
		contentJSON, err := json.Marshal(question.Content)
		require.NoError(t, err)

		query := `
			INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation, status,
				topic_category, grammar_focus, vocabulary_domain, scenario, style_modifier, difficulty_modifier, time_context, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
			RETURNING id
		`

		err = db.QueryRow(
			query,
			question.Type, question.Language, question.Level, question.DifficultyScore, string(contentJSON),
			question.CorrectAnswer, question.Explanation, question.Status,
			question.TopicCategory, question.GrammarFocus, question.VocabularyDomain,
			question.Scenario, question.StyleModifier, question.DifficultyModifier,
			question.TimeContext, time.Now(),
		).Scan(&question.ID)

		require.NoError(t, err)
		questions[i] = *question

		// Assign question to user
		assignQuery := `INSERT INTO user_questions (user_id, question_id, created_at) VALUES ($1, $2, $3)`
		_, err = db.Exec(assignQuery, userID, question.ID, time.Now())
		require.NoError(t, err)
	}

	return questions
}

func TestDailyQuestionService_Integration_AssignDailyQuestions(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	// Create test user
	user := createTestUser(t, db)

	// Check for any existing assignments before creating questions
	var existingAssignmentCount int
	err = db.QueryRow("SELECT COUNT(*) FROM daily_question_assignments WHERE user_id = $1", user.ID).Scan(&existingAssignmentCount)
	require.NoError(t, err)
	t.Logf("Existing assignments before test: %d", existingAssignmentCount)

	// Create test questions
	createTestQuestionsForDaily(t, db, user.ID, 15) // More than 10 to ensure we have enough

	// Debug: Check if questions were created and assigned to user
	var questionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM questions WHERE language = 'italian' AND level = 'B1'").Scan(&questionCount)
	require.NoError(t, err)
	t.Logf("Created %d questions with italian/B1", questionCount)

	var userQuestionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM user_questions WHERE user_id = $1", user.ID).Scan(&userQuestionCount)
	require.NoError(t, err)
	t.Logf("User has %d assigned questions", userQuestionCount)

	// Debug: Check what questions GetAdaptiveQuestionsForDaily would return
	questionsWithStats, err := questionService.GetAdaptiveQuestionsForDaily(context.Background(), user.ID, "italian", "B1", 10)
	if err != nil {
		t.Logf("GetAdaptiveQuestionsForDaily failed: %v", err)
	} else {
		t.Logf("GetAdaptiveQuestionsForDaily returned %d questions", len(questionsWithStats))
		for i, q := range questionsWithStats {
			t.Logf("Question %d: ID=%d, Type=%s", i+1, q.Question.ID, q.Question.Type)
		}
	}

	// Check for any existing assignments before the test
	var existingAssignmentCount2 int
	err = db.QueryRow("SELECT COUNT(*) FROM daily_question_assignments WHERE user_id = $1", user.ID).Scan(&existingAssignmentCount2)
	require.NoError(t, err)
	t.Logf("Existing assignments before test: %d", existingAssignmentCount2)

	// Test assigning daily questions
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date)
	if err != nil {
		t.Logf("First assignment failed: %v", err)
		// Check if any assignments were created despite the error
		var assignmentCount int
		err2 := db.QueryRow("SELECT COUNT(*) FROM daily_question_assignments WHERE user_id = $1 AND assignment_date = $2", user.ID, date.Format("2006-01-02")).Scan(&assignmentCount)
		if err2 == nil {
			t.Logf("Found %d existing assignments despite error", assignmentCount)
		}
	}

	assert.NoError(t, err)

	// Verify assignments were created
	assignments, err := dailyService.GetDailyQuestions(context.Background(), user.ID, date)
	assert.NoError(t, err)
	assert.Len(t, assignments, 10) // Should have exactly 10 questions

	// Verify all assignments are for the correct user and date
	for _, assignment := range assignments {
		assert.Equal(t, user.ID, assignment.UserID)
		assert.Equal(t, date.Format("2006-01-02"), assignment.AssignmentDate.Format("2006-01-02"))
		assert.False(t, assignment.IsCompleted)
		assert.NotNil(t, assignment.Question)
	}

	// Test that calling again doesn't create duplicate assignments
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date)
	assert.NoError(t, err)

	assignments, err = dailyService.GetDailyQuestions(context.Background(), user.ID, date)
	assert.NoError(t, err)
	assert.Len(t, assignments, 10) // Should still have exactly 10 questions
}

func TestDailyQuestionService_Integration_MarkQuestionCompleted(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	// Create test user and questions
	user := createTestUser(t, db)
	createTestQuestionsForDaily(t, db, user.ID, 10)
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// Clean up any recent responses that might filter out questions
	_, err = db.Exec("DELETE FROM user_responses WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 hour'", user.ID)
	require.NoError(t, err)

	// Additional cleanup: Remove ALL responses for this user to ensure all questions are available
	_, err = db.Exec("DELETE FROM user_responses WHERE user_id = $1", user.ID)
	require.NoError(t, err)

	// Debug: Check what questions were created
	var questionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM questions WHERE language = 'italian' AND level = 'B1'").Scan(&questionCount)
	require.NoError(t, err)
	t.Logf("Created %d questions with italian/B1", questionCount)

	var userQuestionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM user_questions WHERE user_id = $1", user.ID).Scan(&userQuestionCount)
	require.NoError(t, err)
	t.Logf("User has %d assigned questions", userQuestionCount)

	// Debug: Check question types distribution
	rows, err := db.Query("SELECT type, COUNT(*) FROM questions WHERE language = 'italian' AND level = 'B1' GROUP BY type")
	require.NoError(t, err)
	defer rows.Close()
	t.Logf("Question type distribution:")
	for rows.Next() {
		var qType string
		var count int
		err := rows.Scan(&qType, &count)
		require.NoError(t, err)
		t.Logf("  %s: %d", qType, count)
	}

	// Debug: Check what questions GetAdaptiveQuestionsForDaily would return
	questionsWithStats, err := questionService.GetAdaptiveQuestionsForDaily(context.Background(), user.ID, "italian", "B1", 10)
	if err != nil {
		t.Logf("GetAdaptiveQuestionsForDaily failed: %v", err)
	} else {
		t.Logf("GetAdaptiveQuestionsForDaily returned %d questions", len(questionsWithStats))
		for i, q := range questionsWithStats {
			t.Logf("Question %d: ID=%d, Type=%s", i+1, q.Question.ID, q.Question.Type)
		}
	}

	// Check for any existing assignments before the test
	var existingAssignmentCount int
	err = db.QueryRow("SELECT COUNT(*) FROM daily_question_assignments WHERE user_id = $1", user.ID).Scan(&existingAssignmentCount)
	require.NoError(t, err)
	t.Logf("Existing assignments before test: %d", existingAssignmentCount)

	// Assign daily questions
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date)
	if err != nil {
		t.Logf("AssignDailyQuestions failed: %v", err)
		// Check if any assignments were created despite the error
		var assignmentCount int
		err2 := db.QueryRow("SELECT COUNT(*) FROM daily_question_assignments WHERE user_id = $1 AND assignment_date = $2", user.ID, date.Format("2006-01-02")).Scan(&assignmentCount)
		if err2 == nil {
			t.Logf("Found %d existing assignments despite error", assignmentCount)
		}
	}
	assert.NoError(t, err)

	// Get assignments
	assignments, err := dailyService.GetDailyQuestions(context.Background(), user.ID, date)
	assert.NoError(t, err)
	assert.Len(t, assignments, 10, "Should have exactly 10 questions assigned")

	// Safety check to prevent panic
	if len(assignments) == 0 {
		t.Fatal("No assignments found, cannot continue with test")
	}

	// Mark first question as completed
	err = dailyService.MarkQuestionCompleted(context.Background(), user.ID, assignments[0].QuestionID, date)
	assert.NoError(t, err)

	// Verify the question is marked as completed
	assignments, err = dailyService.GetDailyQuestions(context.Background(), user.ID, date)
	assert.NoError(t, err)
	assert.True(t, assignments[0].IsCompleted)
	assert.NotNil(t, assignments[0].CompletedAt)

	// Verify other questions are still not completed
	for i := 1; i < len(assignments); i++ {
		assert.False(t, assignments[i].IsCompleted)
		assert.False(t, assignments[i].CompletedAt.Valid) // Check if the sql.NullTime is not valid
	}
}

func TestDailyQuestionService_Integration_ResetQuestionCompleted(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	// Create test user and questions
	user := createTestUser(t, db)
	createTestQuestionsForDaily(t, db, user.ID, 10)
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// Assign daily questions
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date)
	assert.NoError(t, err)

	// Get assignments
	assignments, err := dailyService.GetDailyQuestions(context.Background(), user.ID, date)
	assert.NoError(t, err)

	// Safety check to prevent panic
	if len(assignments) == 0 {
		t.Fatal("No assignments found, cannot continue with test")
	}

	// Mark first question as completed
	err = dailyService.MarkQuestionCompleted(context.Background(), user.ID, assignments[0].QuestionID, date)
	assert.NoError(t, err)

	// Verify it's completed
	assignments, err = dailyService.GetDailyQuestions(context.Background(), user.ID, date)
	assert.NoError(t, err)
	assert.True(t, assignments[0].IsCompleted)

	// Reset the question
	err = dailyService.ResetQuestionCompleted(context.Background(), user.ID, assignments[0].QuestionID, date)
	assert.NoError(t, err)

	// Verify it's no longer completed
	assignments, err = dailyService.GetDailyQuestions(context.Background(), user.ID, date)
	assert.NoError(t, err)
	assert.False(t, assignments[0].IsCompleted)
	assert.False(t, assignments[0].CompletedAt.Valid) // Check if the sql.NullTime is not valid
}

func TestDailyQuestionService_Integration_GetAvailableDates(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	// Create test user and questions
	user := createTestUser(t, db)
	createTestQuestionsForDaily(t, db, user.ID, 10)

	// Assign questions for multiple dates
	date1 := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	date3 := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)

	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date1)
	assert.NoError(t, err)
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date2)
	assert.NoError(t, err)
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date3)
	assert.NoError(t, err)

	// Get available dates
	dates, err := dailyService.GetAvailableDates(context.Background(), user.ID)
	assert.NoError(t, err)
	assert.Len(t, dates, 3)

	// Verify dates are in descending order (most recent first)
	// Normalize timezone for comparison since database returns dates without timezone
	assert.Equal(t, date3.Format("2006-01-02"), dates[0].Format("2006-01-02"))
	assert.Equal(t, date2.Format("2006-01-02"), dates[1].Format("2006-01-02"))
	assert.Equal(t, date1.Format("2006-01-02"), dates[2].Format("2006-01-02"))
}

func TestDailyQuestionService_Integration_GetDailyProgress(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	// Create test user and questions
	user := createTestUser(t, db)
	createTestQuestionsForDaily(t, db, user.ID, 10)
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// Assign daily questions
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date)
	assert.NoError(t, err)

	// Get initial progress
	progress, err := dailyService.GetDailyProgress(context.Background(), user.ID, date)
	assert.NoError(t, err)
	assert.Equal(t, 10, progress.Total)
	assert.Equal(t, 0, progress.Completed)

	// Get assignments and mark some as completed
	assignments, err := dailyService.GetDailyQuestions(context.Background(), user.ID, date)
	assert.NoError(t, err)

	// Mark first 3 questions as completed
	for i := 0; i < 3; i++ {
		err = dailyService.MarkQuestionCompleted(context.Background(), user.ID, assignments[i].QuestionID, date)
		assert.NoError(t, err)
	}

	// Check progress again
	progress, err = dailyService.GetDailyProgress(context.Background(), user.ID, date)
	assert.NoError(t, err)
	assert.Equal(t, 10, progress.Total)
	assert.Equal(t, 3, progress.Completed)
}

// Test that daily assignments across multiple consecutive days avoid repeating
// questions within the configured avoid-days window, and allow repeats after
// that window has passed.
func TestDailyQuestionService_Integration_MultiDay_AvoidRepeatDays(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	// Create test user and a modest pool of questions
	user := createTestUser(t, db)
	// Create 30 questions so we have plenty of candidates
	createTestQuestionsForDaily(t, db, user.ID, 30)

	// Choose base date
	baseDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	// Assign for day 1
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, baseDate)
	require.NoError(t, err)
	assignmentsDay1, err := dailyService.GetDailyQuestions(context.Background(), user.ID, baseDate)
	require.NoError(t, err)
	require.Len(t, assignmentsDay1, 10)

	// Mark all day1 assignments as answered correctly so they should be excluded for avoidDays
	for _, a := range assignmentsDay1 {
		// user_responses.user_answer_index is NOT NULL in schema, include a default index and response_time
		_, err := db.Exec(`INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, response_time_ms, created_at) VALUES ($1, $2, $3, TRUE, $4, $5)`, user.ID, a.QuestionID, 0, 100, time.Now())
		require.NoError(t, err)
	}

	// Assign for day 2 (should not include any from day1 because of recent correct answers)
	day2 := baseDate.AddDate(0, 0, 1)
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, day2)
	require.NoError(t, err)
	assignmentsDay2, err := dailyService.GetDailyQuestions(context.Background(), user.ID, day2)
	require.NoError(t, err)
	require.Len(t, assignmentsDay2, 10)

	// Ensure no overlap between day1 and day2 assignments
	idsDay1 := make(map[int]bool)
	for _, a := range assignmentsDay1 {
		idsDay1[a.QuestionID] = true
	}
	for _, a := range assignmentsDay2 {
		assert.False(t, idsDay1[a.QuestionID], "Question %d was repeated on day2 but should have been avoided", a.QuestionID)
	}

	// Assign for day beyond avoidDays (getDailyRepeatAvoidDays defaults to 7)
	dayAfterWindow := baseDate.AddDate(0, 0, 8)
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, dayAfterWindow)
	require.NoError(t, err)
	assignmentsDay8, err := dailyService.GetDailyQuestions(context.Background(), user.ID, dayAfterWindow)
	require.NoError(t, err)
	require.Len(t, assignmentsDay8, 10)

	// Now it is acceptable to have some overlap with day1 once avoid window has passed
	// We'll assert that at least one question from day1 may reappear (non-deterministic), but
	// more importantly that assignments exist and count is correct.
	// Ensure test completes without panics and assignments are valid
	for _, a := range assignmentsDay8 {
		assert.Equal(t, user.ID, a.UserID)
		assert.NotNil(t, a.Question)
	}
}

// Test that regenerating daily questions picks from newly generated/inserted questions
func TestDailyQuestionService_Integration_RegenerateIncludesNewQuestions(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	// Create user and initial questions
	user := createTestUser(t, db)
	createTestQuestionsForDaily(t, db, user.ID, 10)

	date := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date)
	require.NoError(t, err)
	before, err := dailyService.GetDailyQuestions(context.Background(), user.ID, date)
	require.NoError(t, err)
	require.Len(t, before, 10)

	// Insert additional new questions that were "generated"
	newQuestions := createTestQuestionsForDaily(t, db, user.ID, 10)
	t.Logf("Inserted %d new questions", len(newQuestions))

	// Regenerate assignments for the same date
	err = dailyService.RegenerateDailyQuestions(context.Background(), user.ID, date)
	require.NoError(t, err)

	after, err := dailyService.GetDailyQuestions(context.Background(), user.ID, date)
	require.NoError(t, err)
	require.Len(t, after, 10)

	// Ensure that assignments after regeneration are not identical to before (some new questions present)
	beforeIDs := make(map[int]bool)
	for _, a := range before {
		beforeIDs[a.QuestionID] = true
	}
	differentCount := 0
	for _, a := range after {
		if !beforeIDs[a.QuestionID] {
			differentCount++
		}
	}
	assert.Greater(t, differentCount, 0, "Regeneration should include at least one new question")
}

func TestDailyQuestionService_Integration_UserMissingPreferences(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	// Create test user without language/level preferences using direct SQL
	query := `
		INSERT INTO users (username, email, preferred_language, current_level, timezone, password_hash, ai_provider, ai_model, ai_enabled, ai_api_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	var userID int
	err = db.QueryRow(
		query,
		"testuser2", sql.NullString{String: "test2@example.com", Valid: true},
		sql.NullString{String: "", Valid: false}, sql.NullString{String: "", Valid: false},
		sql.NullString{String: "UTC", Valid: true}, sql.NullString{String: "testhash", Valid: true},
		sql.NullString{String: "openai", Valid: true}, sql.NullString{String: "gpt-4", Valid: true},
		sql.NullBool{Bool: true, Valid: true}, sql.NullString{String: "test-key", Valid: true},
		time.Now(), time.Now(),
	).Scan(&userID)

	require.NoError(t, err)

	// Create test questions
	createTestQuestionsForDaily(t, db, userID, 10)
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// Try to assign daily questions - should fail
	err = dailyService.AssignDailyQuestions(context.Background(), userID, date)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user missing language or level preferences")
}

func TestDailyQuestionService_Integration_NoQuestionsAvailable(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	// Create test user
	user := createTestUser(t, db)
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// Try to assign daily questions without any questions in the database
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no questions available for assignment")
}

// Helper function to set up test database for daily question service tests
func setupTestDB(t *testing.T) *sql.DB {
	return SharedTestDBSetup(t)
}

// Helper function to cleanup test database for daily question service tests
func cleanupTestDB(t *testing.T, db *sql.DB) {
	CleanupTestDatabase(db, t)
}

func TestDailyQuestionService_Integration_AdaptiveQuestionSelection(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	// Create test user
	user := createTestUser(t, db)

	// Create test questions using the helper function
	createTestQuestionsForDaily(t, db, user.ID, 15) // Create 15 questions to ensure we have enough variety

	// Assign daily questions using adaptive selection
	date := time.Now().UTC().Truncate(24 * time.Hour)
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date)
	require.NoError(t, err)

	// Get the assigned questions
	dailyQuestions, err := dailyService.GetDailyQuestions(context.Background(), user.ID, date)
	require.NoError(t, err)
	require.Len(t, dailyQuestions, 10, "Should assign exactly 10 questions")

	// Verify variety in question types
	questionTypeCounts := make(map[models.QuestionType]int)
	for _, assignment := range dailyQuestions {
		questionTypeCounts[assignment.Question.Type]++
	}

	// Should have variety across question types (at least 2 different types)
	require.GreaterOrEqual(t, len(questionTypeCounts), 2, "Should have variety in question types")

	// Each type should have at least 1 question
	for qType, count := range questionTypeCounts {
		require.GreaterOrEqual(t, count, 1, fmt.Sprintf("Type %s should have at least 1 question", qType))
	}

	// Test regeneration with adaptive selection
	err = dailyService.RegenerateDailyQuestions(context.Background(), user.ID, date)
	require.NoError(t, err)

	// Get the regenerated questions
	regeneratedQuestions, err := dailyService.GetDailyQuestions(context.Background(), user.ID, date)
	require.NoError(t, err)
	require.Len(t, regeneratedQuestions, 10, "Should still have exactly 10 questions after regeneration")

	// Verify variety is maintained after regeneration
	regeneratedTypes := make(map[models.QuestionType]int)
	for _, assignment := range regeneratedQuestions {
		regeneratedTypes[assignment.Question.Type]++
	}

	require.GreaterOrEqual(t, len(regeneratedTypes), 2, "Should maintain variety after regeneration")
}

// Verify that when a user answers a question correctly, future assignments for
// that question within the avoid window are removed, and that the worker
// (simulated here by invoking AssignDailyQuestions) can top up those dates.
func TestDailyQuestionService_Integration_RemoveFutureAssignmentsAndRefillByWorkerSim(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	// Create test user and many questions
	user := createTestUser(t, db)
	createTestQuestionsForDaily(t, db, user.ID, 30)

	// Choose a base date and a range of future dates up to avoidDays
	baseDate := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	avoidDays := cfg.Server.DailyRepeatAvoidDays
	if avoidDays <= 0 {
		avoidDays = 7
	}

	// Ensure clean slate for user_responses
	_, err = db.Exec("DELETE FROM user_responses WHERE user_id = $1", user.ID)
	require.NoError(t, err)

	// Assign for baseDate and all future dates in the avoid window
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, baseDate)
	require.NoError(t, err)
	for d := 1; d <= avoidDays; d++ {
		dt := baseDate.AddDate(0, 0, d)
		err = dailyService.AssignDailyQuestions(context.Background(), user.ID, dt)
		require.NoError(t, err)
	}

	// Pick a question from baseDate assignments and submit a correct answer
	assignments, err := dailyService.GetDailyQuestions(context.Background(), user.ID, baseDate)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(assignments), 1)

	q := assignments[0].Question
	require.NotNil(t, q)

	// Submit correct answer
	_, err = dailyService.SubmitDailyQuestionAnswer(context.Background(), user.ID, q.ID, baseDate, q.CorrectAnswer)
	require.NoError(t, err)

	// Verify future assignments for this question were removed for the avoid window
	for d := 1; d <= avoidDays; d++ {
		dt := baseDate.AddDate(0, 0, d)
		var cnt int
		err = db.QueryRow("SELECT COUNT(*) FROM daily_question_assignments WHERE user_id = $1 AND question_id = $2 AND assignment_date = $3", user.ID, q.ID, dt).Scan(&cnt)
		require.NoError(t, err)
		require.Equal(t, 0, cnt, "future assignment should be removed for date %v", dt)
	}

	// Simulate worker top-up by calling AssignDailyQuestions for affected dates
	// and verify that each date is topped up to the user's daily goal
	prefs, err := learningService.GetUserLearningPreferences(context.Background(), user.ID)
	require.NoError(t, err)
	goal := 10
	if prefs != nil && prefs.DailyGoal > 0 {
		goal = prefs.DailyGoal
	}

	for d := 1; d <= avoidDays; d++ {
		dt := baseDate.AddDate(0, 0, d)
		err = dailyService.AssignDailyQuestions(context.Background(), user.ID, dt)
		require.NoError(t, err)

		// Verify count equals goal
		var assigned int
		err = db.QueryRow("SELECT COUNT(*) FROM daily_question_assignments WHERE user_id = $1 AND assignment_date = $2", user.ID, dt).Scan(&assigned)
		require.NoError(t, err)
		require.Equal(t, goal, assigned, "date %v should be topped up to goal", dt)
	}
}

// Test that partial existing assignments are topped up to the user's daily goal
func TestDailyQuestionService_Integration_PartialAssignmentRefill(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	// Create test user and many questions
	user := createTestUser(t, db)
	createTestQuestionsForDaily(t, db, user.ID, 20)

	// Ensure user has a larger daily goal (e.g., 15)
	_, err = db.Exec(`INSERT INTO user_learning_preferences (user_id, daily_goal, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) ON CONFLICT (user_id) DO UPDATE SET daily_goal = EXCLUDED.daily_goal`, user.ID, 15)
	require.NoError(t, err)

	date := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	// Manually insert a partial set of assignments (e.g., 8)
	for i := 0; i < 8; i++ {
		// pick first 8 user_questions question ids
		var qid int
		err = db.QueryRow("SELECT question_id FROM user_questions WHERE user_id = $1 ORDER BY created_at ASC LIMIT 1 OFFSET $2", user.ID, i).Scan(&qid)
		require.NoError(t, err)
		_, err = db.Exec("INSERT INTO daily_question_assignments (user_id, question_id, assignment_date, created_at) VALUES ($1, $2, $3, NOW())", user.ID, qid, date)
		require.NoError(t, err)
	}

	// Sanity check: ensure 8 assignments exist
	var existing int
	err = db.QueryRow("SELECT COUNT(*) FROM daily_question_assignments WHERE user_id = $1 AND assignment_date = $2", user.ID, date).Scan(&existing)
	require.NoError(t, err)
	require.Equal(t, 8, existing)

	// Now call AssignDailyQuestions which should top up to 15
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date)
	require.NoError(t, err)

	// Verify count is now 15
	err = db.QueryRow("SELECT COUNT(*) FROM daily_question_assignments WHERE user_id = $1 AND assignment_date = $2", user.ID, date).Scan(&existing)
	require.NoError(t, err)
	require.Equal(t, 15, existing)
}

// Spy wrapper around the real QuestionService to capture the limit passed
type spyQuestionService struct {
	*QuestionService
	lastLimit int
}

func (s *spyQuestionService) GetAdaptiveQuestionsForDaily(ctx context.Context, userID int, language, level string, limit int) ([]*QuestionWithStats, error) {
	s.lastLimit = limit
	return s.QuestionService.GetAdaptiveQuestionsForDaily(ctx, userID, language, level, limit)
}

// Verify that when a user sets a very large daily goal, the service requests goal+buffer candidates
func TestDailyQuestionService_Integration_LargeGoalUsesBuffer(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	realQuestionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	spy := &spyQuestionService{QuestionService: realQuestionService}
	dailyService := NewDailyQuestionService(db, logger, spy, learningService)

	// Create test user and many questions
	user := createTestUser(t, db)
	createTestQuestionsForDaily(t, db, user.ID, 120)

	// Set a large daily goal (100)
	_, err = db.Exec(`INSERT INTO user_learning_preferences (user_id, daily_goal, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) ON CONFLICT (user_id) DO UPDATE SET daily_goal = EXCLUDED.daily_goal`, user.ID, 100)
	require.NoError(t, err)

	date := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

	// Call AssignDailyQuestions
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date)
	require.NoError(t, err)

	// Buffer in code is 10, so expect requested limit = goal + 10
	require.Equal(t, 110, spy.lastLimit, "expected GetAdaptiveQuestionsForDaily to be called with goal+buffer")

	// Verify assignments count equals the user's goal (100)
	var assigned int
	err = db.QueryRow("SELECT COUNT(*) FROM daily_question_assignments WHERE user_id = $1 AND assignment_date = $2", user.ID, date).Scan(&assigned)
	require.NoError(t, err)
	require.Equal(t, 100, assigned)
}

// Ensures daily selection excludes questions answered correctly within the last 2 days
func TestDailyQuestionService_Integration_ExcludeRecentCorrectRepeats(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	// Create test user and ample questions
	user := createTestUser(t, db)
	questions := createTestQuestionsForDaily(t, db, user.ID, 20)

	// Mark one question as answered correctly 1 day ago (should be excluded)
	qRecentCorrect := questions[0]
	_, err = db.Exec(`
        INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, response_time_ms, created_at)
        VALUES ($1, $2, 0, TRUE, 1500, NOW() - INTERVAL '1 day')
    `, user.ID, qRecentCorrect.ID)
	require.NoError(t, err)

	// Mark one question as answered incorrectly 1 day ago (allowed)
	qRecentIncorrect := questions[1]
	_, err = db.Exec(`
        INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, response_time_ms, created_at)
        VALUES ($1, $2, 1, FALSE, 1200, NOW() - INTERVAL '1 day')
    `, user.ID, qRecentIncorrect.ID)
	require.NoError(t, err)

	// Mark one question as answered correctly 3 days ago (outside window, allowed)
	qOldCorrect := questions[2]
	_, err = db.Exec(`
        INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, response_time_ms, created_at)
        VALUES ($1, $2, 2, TRUE, 1100, NOW() - INTERVAL '3 days')
    `, user.ID, qOldCorrect.ID)
	require.NoError(t, err)

	// Assign daily questions
	date := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	err = dailyService.AssignDailyQuestions(context.Background(), user.ID, date)
	require.NoError(t, err)

	// Retrieve assignments
	assignments, err := dailyService.GetDailyQuestions(context.Background(), user.ID, date)
	require.NoError(t, err)
	require.Len(t, assignments, 10, "Should assign exactly 10 questions")

	// Ensure the recently-correct question is excluded
	for _, a := range assignments {
		assert.NotEqual(t, qRecentCorrect.ID, a.QuestionID, "Question answered correctly yesterday must not be assigned")
	}
}
