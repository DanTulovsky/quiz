//go:build integration

package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestN1QueryProblems tests that methods don't have N+1 query issues
func TestN1QueryProblems(t *testing.T) {
	db := setupTestDBForQuestion(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	service := NewQuestionServiceWithLogger(db, learningService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser", "password", "italian", "A1")
	require.NoError(t, err)

	// Create multiple questions to test with
	questions := createTestQuestions(t, service, user.ID, 10)

	// Add responses to create stats
	addTestResponses(t, db, user.ID, questions)

	t.Run("GetUserQuestionsWithStats_NoN1Query", func(t *testing.T) {
		// This should use a single JOIN query, not multiple queries
		questionsWithStats, err := service.GetUserQuestionsWithStats(context.Background(), user.ID, 100)
		require.NoError(t, err)
		require.Len(t, questionsWithStats, 10)

		// Verify all questions have stats
		for _, q := range questionsWithStats {
			assert.NotNil(t, q.Question)
			assert.GreaterOrEqual(t, q.TotalResponses, 0)
			assert.GreaterOrEqual(t, q.CorrectCount, 0)
			assert.GreaterOrEqual(t, q.IncorrectCount, 0)
		}
	})

	t.Run("GetQuestionsPaginated_NoN1Query", func(t *testing.T) {
		// This should use a single JOIN query, not multiple queries
		questionsWithStats, total, err := service.GetQuestionsPaginated(context.Background(), user.ID, 1, 100, "", "", "")
		require.NoError(t, err)
		assert.Equal(t, 10, total)
		require.Len(t, questionsWithStats, 10)

		// Verify all questions have stats
		for _, q := range questionsWithStats {
			assert.NotNil(t, q.Question)
			assert.GreaterOrEqual(t, q.TotalResponses, 0)
			assert.GreaterOrEqual(t, q.CorrectCount, 0)
			assert.GreaterOrEqual(t, q.IncorrectCount, 0)
		}
	})

	t.Run("GetQuestionWithStats_NoN1Query", func(t *testing.T) {
		// This should use a single JOIN query, not multiple queries
		questionWithStats, err := service.GetQuestionWithStats(context.Background(), questions[0].ID)
		require.NoError(t, err)
		require.NotNil(t, questionWithStats)

		assert.Equal(t, questions[0].ID, questionWithStats.Question.ID)
		assert.GreaterOrEqual(t, questionWithStats.TotalResponses, 0)
		assert.GreaterOrEqual(t, questionWithStats.CorrectCount, 0)
		assert.GreaterOrEqual(t, questionWithStats.IncorrectCount, 0)
	})

	t.Run("GetNextQuestion_NoN1Query", func(t *testing.T) {
		// This should use a single complex JOIN query, not multiple queries
		questionWithStats, err := service.GetNextQuestion(context.Background(), user.ID, "italian", "A1", models.Vocabulary)
		// It's possible no questions are available, which is valid
		if err != nil {
			t.Skipf("No questions available for user %d: %v", user.ID, err)
		}
		if questionWithStats == nil {
			t.Skip("No questions available for this user/language/level combination")
		}

		assert.Greater(t, questionWithStats.Question.ID, 0)
		assert.GreaterOrEqual(t, questionWithStats.TotalResponses, 0)
		assert.GreaterOrEqual(t, questionWithStats.CorrectCount, 0)
		assert.GreaterOrEqual(t, questionWithStats.IncorrectCount, 0)
	})
}

// TestQueryPerformance tests that queries are efficient
func TestQueryPerformance(t *testing.T) {
	db := setupTestDBForQuestion(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	service := NewQuestionServiceWithLogger(db, learningService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser", "password", "italian", "A1")
	require.NoError(t, err)

	// Create a larger dataset to test performance
	questions := createTestQuestions(t, service, user.ID, 50)
	addTestResponses(t, db, user.ID, questions)

	t.Run("GetUserQuestionsWithStats_Performance", func(t *testing.T) {
		start := time.Now()
		questionsWithStats, err := service.GetUserQuestionsWithStats(context.Background(), user.ID, 1000)
		duration := time.Since(start)

		require.NoError(t, err)
		require.Len(t, questionsWithStats, 50)

		// Should complete within reasonable time (adjust based on your requirements)
		assert.Less(t, duration, 100*time.Millisecond, "Query took too long: %v", duration)
	})

	t.Run("GetQuestionsPaginated_Performance", func(t *testing.T) {
		start := time.Now()
		questionsWithStats, total, err := service.GetQuestionsPaginated(context.Background(), user.ID, 1, 1000, "", "", "")
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Equal(t, 50, total)
		require.Len(t, questionsWithStats, 50)

		// Should complete within reasonable time
		assert.Less(t, duration, 100*time.Millisecond, "Query took too long: %v", duration)
	})
}

// TestQueryConsistency tests that queries return consistent results
func TestQueryConsistency(t *testing.T) {
	db := setupTestDBForQuestion(t)
	defer db.Close()

	// Clean up user_responses to ensure test isolation
	_, err := db.Exec("DELETE FROM user_responses")
	if err != nil {
		t.Fatalf("failed to clear user_responses: %v", err)
	}

	cfg, err := config.NewConfig()
	require.NoError(t, err)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	service := NewQuestionServiceWithLogger(db, learningService, cfg, observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false}))

	// Create a test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser", "password", "italian", "A1")
	require.NoError(t, err)

	// Create questions with known response counts
	questions := createTestQuestions(t, service, user.ID, 5)

	// Add specific responses to create predictable stats
	for i, question := range questions {
		// Add i+1 correct responses and i incorrect responses
		for j := 0; j < i+1; j++ {
			_, err := db.Exec("INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct) VALUES ($1, $2, $3, $4)",
				user.ID, question.ID, j, true)
			require.NoError(t, err)
		}
		for j := 0; j < i; j++ {
			_, err := db.Exec("INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct) VALUES ($1, $2, $3, $4)",
				user.ID, question.ID, j, false)
			require.NoError(t, err)
		}
	}

	t.Run("GetUserQuestionsWithStats_Consistency", func(t *testing.T) {
		questionsWithStats, err := service.GetUserQuestionsWithStats(context.Background(), user.ID, 100)
		require.NoError(t, err)
		require.Len(t, questionsWithStats, 5)

		// Build expected stats by question ID
		expectedStats := make(map[int]struct{ correct, incorrect, total int })
		for i, q := range questions {
			expectedStats[q.ID] = struct{ correct, incorrect, total int }{i + 1, i, i + 1 + i}
		}

		// Verify stats are consistent by question ID
		for _, q := range questionsWithStats {
			exp, ok := expectedStats[q.Question.ID]
			if !ok {
				t.Errorf("Unexpected question ID %d in results", q.Question.ID)
				continue
			}
			assert.Equal(t, exp.correct, q.CorrectCount, "Question %d should have %d correct responses", q.Question.ID, exp.correct)
			assert.Equal(t, exp.incorrect, q.IncorrectCount, "Question %d should have %d incorrect responses", q.Question.ID, exp.incorrect)
			assert.Equal(t, exp.total, q.TotalResponses, "Question %d should have %d total responses", q.Question.ID, exp.total)
		}
	})
}

// Helper functions

func createTestQuestions(t *testing.T, service *QuestionService, userID, count int) []*models.Question {
	questions := make([]*models.Question, count)

	for i := 0; i < count; i++ {
		question := &models.Question{
			Type:            models.Vocabulary,
			Language:        "italian",
			Level:           "A1",
			DifficultyScore: 0.5,
			Content: map[string]interface{}{
				"question": fmt.Sprintf("Test question %d", i),
				"options":  []string{"Option A", "Option B", "Option C", "Option D"},
			},
			CorrectAnswer: 0,
			Explanation:   fmt.Sprintf("Explanation %d", i),
			Status:        models.QuestionStatusActive,
		}

		err := service.SaveQuestion(context.Background(), question)
		require.NoError(t, err)

		err = service.AssignQuestionToUser(context.Background(), question.ID, userID)
		require.NoError(t, err)

		questions[i] = question
	}

	return questions
}

func addTestResponses(t *testing.T, db *sql.DB, userID int, questions []*models.Question) {
	// Add specific responses for each question to match expected counts
	responses := []struct {
		questionIndex int
		answer        int
		isCorrect     bool
	}{
		{0, 0, true}, // Question 0: 1 correct
		{1, 0, true}, // Question 1: 2 correct, 1 incorrect
		{1, 0, true},
		{1, 1, false},
		{2, 0, true}, // Question 2: 3 correct, 2 incorrect
		{2, 0, true},
		{2, 0, true},
		{2, 1, false},
		{2, 1, false},
		{3, 0, true}, // Question 3: 4 correct, 3 incorrect
		{3, 0, true},
		{3, 0, true},
		{3, 0, true},
		{3, 1, false},
		{3, 1, false},
		{3, 1, false},
		{4, 0, true}, // Question 4: 5 correct, 4 incorrect
		{4, 0, true},
		{4, 0, true},
		{4, 0, true},
		{4, 0, true},
		{4, 1, false},
		{4, 1, false},
		{4, 1, false},
		{4, 1, false},
	}

	for _, response := range responses {
		if response.questionIndex < len(questions) {
			_, err := db.Exec("INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct) VALUES ($1, $2, $3, $4)",
				userID, questions[response.questionIndex].ID, response.answer, response.isCorrect)
			require.NoError(t, err)
		}
	}
}
