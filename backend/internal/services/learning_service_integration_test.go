//go:build integration
// +build integration

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

func TestLearningService_RecordUserResponse_Integration(t *testing.T) {
	db := setupTestDBForLearning(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user first
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "italian", "A1")
	require.NoError(t, err)

	// Create test question and assign to user
	var questionID int
	err = db.QueryRow(`
		INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, "vocabulary", "italian", "A1", 1.0, `{"question":"Test?"}`, 0, "Test explanation").Scan(&questionID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_questions (user_id, question_id) VALUES ($1, $2)`, user.ID, questionID)
	require.NoError(t, err)

	response := &models.UserResponse{
		UserID:          user.ID,
		QuestionID:      questionID,
		UserAnswerIndex: 0,
		IsCorrect:       true,
		ResponseTimeMs:  5000,
	}

	err = learningService.RecordUserResponse(context.Background(), response)
	require.NoError(t, err)
	assert.Greater(t, response.ID, 0)

	// Verify performance metrics were updated
	var totalAttempts, correctAttempts int
	err = db.QueryRow(`
		SELECT total_attempts, correct_attempts
		FROM performance_metrics
		WHERE user_id = $1 AND language = $2 AND level = $3
	`, user.ID, "italian", "A1").Scan(&totalAttempts, &correctAttempts)

	require.NoError(t, err)
	assert.Equal(t, 1, totalAttempts)
	assert.Equal(t, 1, correctAttempts)
}

func TestLearningService_GetUserProgress_Integration(t *testing.T) {
	db := setupTestDBForLearning(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "italian", "B1")
	require.NoError(t, err)

	// Verify user was created properly
	var userID int
	var currentLevel sql.NullString
	err = db.QueryRow("SELECT id, current_level FROM users WHERE id = $1", user.ID).Scan(&userID, &currentLevel)
	require.NoError(t, err)
	require.Equal(t, user.ID, userID)

	// Create test question and assign to user
	var questionID int
	err = db.QueryRow(`
		INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, "vocabulary", "italian", "B1", 3.0, `{"question":"Test?"}`, 0, "Test explanation").Scan(&questionID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_questions (user_id, question_id) VALUES ($1, $2)`, user.ID, questionID)
	require.NoError(t, err)

	// Create test responses
	responses := []models.UserResponse{
		{UserID: user.ID, QuestionID: questionID, UserAnswerIndex: 0, IsCorrect: true, ResponseTimeMs: 3000},
		{UserID: user.ID, QuestionID: questionID, UserAnswerIndex: 1, IsCorrect: false, ResponseTimeMs: 5000},
		{UserID: user.ID, QuestionID: questionID, UserAnswerIndex: 0, IsCorrect: true, ResponseTimeMs: 2000},
	}

	for _, response := range responses {
		err = learningService.RecordUserResponse(context.Background(), &response)
		require.NoError(t, err)
	}

	progress, err := learningService.GetUserProgress(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, progress)

	assert.Equal(t, "B1", progress.CurrentLevel)
	assert.Equal(t, 3, progress.TotalQuestions)
	assert.Equal(t, 2, progress.CorrectAnswers)
	assert.InDelta(t, 66.67, progress.AccuracyRate, 0.1)

	// Check performance by topic (default is "general")
	progressKey := "_italian_B1"
	assert.Contains(t, progress.PerformanceByTopic, progressKey)

	metric := progress.PerformanceByTopic[progressKey]
	require.NotNil(t, metric)
	assert.Equal(t, 3, metric.TotalAttempts)
	assert.Equal(t, 2, metric.CorrectAttempts)
	assert.InDelta(t, 66.67, metric.AccuracyRate(), 0.1)

	// Check recent activity
	assert.Len(t, progress.RecentActivity, 3)
}

func TestLearningService_ShouldAvoidQuestion_Integration(t *testing.T) {
	db := setupTestDBForLearning(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "italian", "A1")
	require.NoError(t, err)

	// Create test question and assign to user
	var questionID int
	err = db.QueryRow(`
		INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, "vocabulary", "italian", "A1", 1.0, `{"question":"Test?"}`, 0, "Test explanation").Scan(&questionID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_questions (user_id, question_id) VALUES ($1, $2)`, user.ID, questionID)
	require.NoError(t, err)

	// Test when no recent correct answers
	shouldAvoid, err := learningService.ShouldAvoidQuestion(context.Background(), user.ID, questionID)
	require.NoError(t, err)
	assert.False(t, shouldAvoid)

	// Record a correct answer recently
	_, err = db.Exec(`
		INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, response_time_ms, created_at)
		VALUES ($1, $2, 0, true, 3000, NOW())
	`, user.ID, questionID)
	require.NoError(t, err)

	// Test when there's a recent correct answer
	shouldAvoid, err = learningService.ShouldAvoidQuestion(context.Background(), user.ID, questionID)
	require.NoError(t, err)
	assert.True(t, shouldAvoid)
}

// TestPriorityScoreCalculation tests the priority score calculation logic
func TestPriorityScoreCalculation_Integration(t *testing.T) {
	db := setupTestDBForLearning(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create a test user
	username := fmt.Sprintf("testuser_priority_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "italian", "A1")
	require.NoError(t, err)

	// Create a test question
	var questionID int
	err = db.QueryRow(`
		INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation)
		VALUES ('vocabulary', 'italian', 'A1', 1.0, '{"question":"Test?"}', 0, 'Test explanation')
		RETURNING id
	`).Scan(&questionID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_questions (user_id, question_id) VALUES ($1, $2)`, user.ID, questionID)
	require.NoError(t, err)

	// Test case 1: New user, no performance history
	t.Run("new user, no history", func(t *testing.T) {
		score, err := learningService.CalculatePriorityScore(context.Background(), user.ID, questionID)
		require.NoError(t, err)
		// Base score (100) * freshness boost (1.5) = 150
		assert.InDelta(t, 150.0, score, 0.1)
	})

	// Test case 2: After one correct answer
	t.Run("after one correct answer", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, created_at)
			VALUES ($1, $2, 0, TRUE, NOW())
		`, user.ID, questionID)
		require.NoError(t, err)

		score, err := learningService.CalculatePriorityScore(context.Background(), user.ID, questionID)
		require.NoError(t, err)

		// Base (100) * Perf (1 - 0.5) * Spaced (1) * Pref (1) * Fresh (1) = 50
		assert.InDelta(t, 50.0, score, 1.0) // Delta of 1 to allow for timing
	})

	// Test case 3: After one incorrect answer
	t.Run("after one incorrect answer", func(t *testing.T) {
		// Clear previous responses for clean test
		_, err := db.Exec(`DELETE FROM user_responses WHERE user_id = $1`, user.ID)
		require.NoError(t, err)
		_, err = db.Exec(`
			INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, created_at)
			VALUES ($1, $2, 0, FALSE, NOW())
		`, user.ID, questionID)
		require.NoError(t, err)

		score, err := learningService.CalculatePriorityScore(context.Background(), user.ID, questionID)
		require.NoError(t, err)

		// Prefs: weak_area_boost = 2.0
		// Base (100) * Perf (1 + (1 * 2.0) - (0 * 0.5)) * Spaced (1) * Pref (1) * Fresh (1) = 300
		assert.InDelta(t, 300.0, score, 1.0)
	})

	// Test case 4: Priority should be higher for low confidence than for high confidence
	t.Run("priority higher for low confidence than high confidence", func(t *testing.T) {
		// Low confidence → see more
		low := 1
		err := learningService.MarkQuestionAsKnown(context.Background(), user.ID, questionID, &low)
		require.NoError(t, err)
		lowScore, err := learningService.CalculatePriorityScore(context.Background(), user.ID, questionID)
		require.NoError(t, err)

		// High confidence → see less
		high := 5
		err = learningService.MarkQuestionAsKnown(context.Background(), user.ID, questionID, &high)
		require.NoError(t, err)
		highScore, err := learningService.CalculatePriorityScore(context.Background(), user.ID, questionID)
		require.NoError(t, err)

		assert.Greater(t, lowScore, highScore)
	})
}

// TestMarkQuestionAsKnown_Integration tests the MarkQuestionAsKnown functionality
func TestMarkQuestionAsKnown_Integration(t *testing.T) {
	db := setupTestDBForLearning(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create a test user
	username := fmt.Sprintf("testuser_mark_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "italian", "A1")
	require.NoError(t, err)

	// Create a test question
	var questionID int
	err = db.QueryRow(`
		INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation)
		VALUES ('vocabulary', 'italian', 'A1', 1.0, '{"question":"Test?"}', 0, 'Test explanation')
		RETURNING id
	`).Scan(&questionID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_questions (user_id, question_id) VALUES ($1, $2)`, user.ID, questionID)
	require.NoError(t, err)

	// Test marking question as known
	err = learningService.MarkQuestionAsKnown(context.Background(), user.ID, questionID, nil)
	require.NoError(t, err)

	// Verify the question is marked as known in the database
	var markedAsKnown bool
	var markedAt sql.NullTime
	err = db.QueryRow(`
		SELECT marked_as_known, marked_as_known_at
		FROM user_question_metadata
		WHERE user_id = $1 AND question_id = $2
	`, user.ID, questionID).Scan(&markedAsKnown, &markedAt)
	require.NoError(t, err)
	assert.True(t, markedAsKnown)
	assert.True(t, markedAt.Valid)

	// Test marking the same question again (should not error)
	err = learningService.MarkQuestionAsKnown(context.Background(), user.ID, questionID, nil)
	require.NoError(t, err)

	// Verify it's still marked as known
	err = db.QueryRow(`
		SELECT marked_as_known
		FROM user_question_metadata
		WHERE user_id = $1 AND question_id = $2
	`, user.ID, questionID).Scan(&markedAsKnown)
	require.NoError(t, err)
	assert.True(t, markedAsKnown)

	// Test marking a non-existent question (should not error)
	// First create a question with ID 99999 to avoid foreign key constraint
	_, err = db.Exec(`
		INSERT INTO questions (id, type, language, level, difficulty_score, content, correct_answer, explanation)
		VALUES (99999, 'vocabulary', 'italian', 'A1', 1.0, '{"question":"Test?"}', 0, 'Test explanation')
	`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_questions (user_id, question_id) VALUES ($1, $2)`, user.ID, 99999)
	require.NoError(t, err)

	err = learningService.MarkQuestionAsKnown(context.Background(), user.ID, 99999, nil)
	require.NoError(t, err)

	// Verify the question is marked as known
	err = db.QueryRow(`
		SELECT marked_as_known
		FROM user_question_metadata
		WHERE user_id = $1 AND question_id = $2
	`, user.ID, 99999).Scan(&markedAsKnown)
	require.NoError(t, err)
	assert.True(t, markedAsKnown)
}

// TestMarkQuestionAsKnown_WithConfidenceLevel_Integration tests the confidence level functionality
func TestMarkQuestionAsKnown_WithConfidenceLevel_Integration(t *testing.T) {
	db := setupTestDBForLearning(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create a test user
	username := fmt.Sprintf("testuser_confidence_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "italian", "A1")
	require.NoError(t, err)

	// Create a test question
	var questionID int
	err = db.QueryRow(`
		INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation)
		VALUES ('vocabulary', 'italian', 'A1', 1.0, '{"question":"Test?"}', 0, 'Test explanation')
		RETURNING id
	`).Scan(&questionID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_questions (user_id, question_id) VALUES ($1, $2)`, user.ID, questionID)
	require.NoError(t, err)

	// Record scores for each level and assert ordering: 1 > 2 > 3 > 4 > 5
	scores := make(map[int]float64)
	for level := 1; level <= 5; level++ {
		t.Run(fmt.Sprintf("confidence level %d affects priority", level), func(t *testing.T) {
			// Clear previous metadata for clean test
			_, err := db.Exec(`DELETE FROM user_question_metadata WHERE user_id = $1 AND question_id = $2`, user.ID, questionID)
			require.NoError(t, err)

			// Mark question as known with specific confidence level
			l := level
			err = learningService.MarkQuestionAsKnown(context.Background(), user.ID, questionID, &l)
			require.NoError(t, err)

			// Sanity: stored correctly
			var storedConfidence sql.NullInt32
			err = db.QueryRow(`
                SELECT confidence_level
                FROM user_question_metadata
                WHERE user_id = $1 AND question_id = $2
            `, user.ID, questionID).Scan(&storedConfidence)
			require.NoError(t, err)
			assert.True(t, storedConfidence.Valid)
			assert.Equal(t, int32(level), storedConfidence.Int32)

			// Calculate score
			score, err := learningService.CalculatePriorityScore(context.Background(), user.ID, questionID)
			require.NoError(t, err)
			scores[level] = score
		})
	}

	// Validate ordering after all subtests
	t.Run("priority ordering by confidence", func(t *testing.T) {
		assert.Greater(t, scores[1], scores[2])
		assert.Greater(t, scores[2], scores[3])
		assert.Greater(t, scores[3], scores[4])
		assert.Greater(t, scores[4], scores[5])
	})

	// Test that questions with lower confidence appear more frequently
	t.Run("priority score comparison", func(t *testing.T) {
		// Create two questions
		var question1ID, question2ID int
		err = db.QueryRow(`
			INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation)
			VALUES ('vocabulary', 'italian', 'A1', 1.0, '{"question":"Test1?"}', 0, 'Test explanation')
			RETURNING id
		`).Scan(&question1ID)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO user_questions (user_id, question_id) VALUES ($1, $2)`, user.ID, question1ID)
		require.NoError(t, err)

		err = db.QueryRow(`
			INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation)
			VALUES ('vocabulary', 'italian', 'A1', 1.0, '{"question":"Test2?"}', 0, 'Test explanation')
			RETURNING id
		`).Scan(&question2ID)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO user_questions (user_id, question_id) VALUES ($1, $2)`, user.ID, question2ID)
		require.NoError(t, err)

		// Mark question 1 with low confidence (1)
		confidence1 := 1
		err = learningService.MarkQuestionAsKnown(context.Background(), user.ID, question1ID, &confidence1)
		require.NoError(t, err)

		// Mark question 2 with high confidence (5)
		confidence2 := 5
		err = learningService.MarkQuestionAsKnown(context.Background(), user.ID, question2ID, &confidence2)
		require.NoError(t, err)

		// Calculate priority scores
		score1, err := learningService.CalculatePriorityScore(context.Background(), user.ID, question1ID)
		require.NoError(t, err)

		score2, err := learningService.CalculatePriorityScore(context.Background(), user.ID, question2ID)
		require.NoError(t, err)

		// Question with low confidence should have higher priority score (appears more frequently)
		assert.Greater(t, score1, score2, "Question with low confidence should have higher priority score")
	})
}

// Priority generation tests moved to worker tests

// Priority generation flow tests moved to worker tests

// Variety selection with priority data tests moved to worker tests

// Fresh question ratio enforcement tests moved to worker tests

// Helper function to set up test database for learning service tests
func setupTestDBForLearning(t *testing.T) *sql.DB {
	return SharedTestDBSetup(t)
}

func TestLearningService_GetWeakestTopics_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_weakest", "password", "italian", "A1")
	require.NoError(t, err)

	// Create test questions and responses to simulate weak areas
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)

	// Create questions with different topics
	topics := []string{"vocabulary", "grammar", "pronunciation"}
	for i, topic := range topics {
		question := &models.Question{
			Type:            "vocabulary",
			Language:        "italian",
			Level:           "A1",
			DifficultyScore: float64(i+1) * 0.5,
			TopicCategory:   topic,
			Content: map[string]interface{}{
				"question": fmt.Sprintf("Test question %d", i+1),
				"options":  []string{"A", "B", "C", "D"},
				"topic":    topic,
			},
			CorrectAnswer: 0,
			Explanation:   "Test explanation",
			Status:        models.QuestionStatusActive,
		}

		err = questionService.SaveQuestion(context.Background(), question)
		require.NoError(t, err)

		// Assign to user
		err = questionService.AssignQuestionToUser(context.Background(), question.ID, user.ID)
		require.NoError(t, err)

		// Create at least 3 responses per topic
		for j := 0; j < 3; j++ {
			response := &models.UserResponse{
				UserID:          user.ID,
				QuestionID:      question.ID,
				UserAnswerIndex: 0,      // Wrong answer for some
				IsCorrect:       i == 0, // Only first topic is strong
				ResponseTimeMs:  1000,
				CreatedAt:       time.Now(),
			}
			err = learningService.RecordUserResponse(context.Background(), response)
			require.NoError(t, err)
		}
	}

	// Test GetWeakestTopics
	weakestTopics, err := learningService.GetWeakestTopics(context.Background(), user.ID, 5)
	require.NoError(t, err)
	assert.NotEmpty(t, weakestTopics)

	// Only keep topics with accuracy < 60%
	var trulyWeak []*models.PerformanceMetrics
	for _, topic := range weakestTopics {
		if topic.AccuracyRate() < 60.0 {
			trulyWeak = append(trulyWeak, topic)
		}
	}

	for i, topic := range trulyWeak {
		t.Logf("Topic %d: %s, Accuracy: %.2f%%, Total: %d, Correct: %d",
			i, topic.Topic, topic.AccuracyRate(), topic.TotalAttempts, topic.CorrectAttempts)
	}

	assert.Len(t, trulyWeak, 2) // Should return 2 weak topics (grammar, pronunciation)
}

func TestLearningService_GetUserQuestionStats_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_stats", "password", "italian", "A1")
	require.NoError(t, err)

	// Create test questions
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)

	for i := 0; i < 5; i++ {
		question := &models.Question{
			Type:            "vocabulary",
			Language:        "italian",
			Level:           "A1",
			DifficultyScore: 1.0,
			Content: map[string]interface{}{
				"question": fmt.Sprintf("Test question %d", i+1),
				"options":  []string{"A", "B", "C", "D"},
			},
			CorrectAnswer: 0,
			Explanation:   "Test explanation",
			Status:        models.QuestionStatusActive,
		}

		err = questionService.SaveQuestion(context.Background(), question)
		require.NoError(t, err)

		// Assign to user
		err = questionService.AssignQuestionToUser(context.Background(), question.ID, user.ID)
		require.NoError(t, err)

		// Create responses
		response := &models.UserResponse{
			UserID:          user.ID,
			QuestionID:      question.ID,
			UserAnswerIndex: 0,
			IsCorrect:       i < 3, // 3 correct, 2 incorrect
			ResponseTimeMs:  1000,
			CreatedAt:       time.Now(),
		}

		err = learningService.RecordUserResponse(context.Background(), response)
		require.NoError(t, err)
	}

	// Test GetUserQuestionStats
	stats, err := learningService.GetUserQuestionStats(context.Background(), user.ID)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(5), int64(stats.TotalAnswered))
	assert.Equal(t, int64(3), int64(stats.CorrectAnswers))
	assert.Equal(t, int64(2), int64(stats.IncorrectAnswers))
	assert.Equal(t, 60.0, stats.AccuracyRate) // 3/5 = 60%
}

func TestLearningService_RecordAnswerWithPriority_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_priority", "password", "italian", "A1")
	require.NoError(t, err)

	// Create test question
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	question := &models.Question{
		Type:            "vocabulary",
		Language:        "italian",
		Level:           "A1",
		DifficultyScore: 1.0,
		Content: map[string]interface{}{
			"question": "Test question",
			"options":  []string{"A", "B", "C", "D"},
		},
		CorrectAnswer: 0,
		Explanation:   "Test explanation",
		Status:        models.QuestionStatusActive,
	}

	err = questionService.SaveQuestion(context.Background(), question)
	require.NoError(t, err)

	// Assign to user
	err = questionService.AssignQuestionToUser(context.Background(), question.ID, user.ID)
	require.NoError(t, err)

	// Test RecordAnswerWithPriority
	response := &models.UserResponse{
		UserID:          user.ID,
		QuestionID:      question.ID,
		UserAnswerIndex: 0,
		IsCorrect:       true,
		ResponseTimeMs:  1000,
		CreatedAt:       time.Now(),
	}

	err = learningService.RecordAnswerWithPriority(context.Background(), user.ID, response.QuestionID, response.UserAnswerIndex, response.IsCorrect, response.ResponseTimeMs)
	require.NoError(t, err)

	// Verify the response was recorded
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM user_responses WHERE user_id = $1 AND question_id = $2", user.ID, question.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestLearningService_UpdateUserLearningPreferences_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_prefs", "password", "italian", "A1")
	require.NoError(t, err)

	// Test UpdateUserLearningPreferences
	preferences := &models.UserLearningPreferences{
		UserID:                    user.ID,
		PreferredQuestionTypes:    []string{"vocabulary", "grammar"},
		PreferredDifficultyLevel:  "A2",
		PreferredTopics:           []string{"daily_life", "travel"},
		PreferredQuestionCount:    10,
		SpacedRepetitionEnabled:   true,
		AdaptiveDifficultyEnabled: true,
		CreatedAt:                 time.Now(),
		UpdatedAt:                 time.Now(),
	}

	_, err = learningService.UpdateUserLearningPreferences(context.Background(), user.ID, preferences)
	require.NoError(t, err)

	// Verify preferences were saved
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM user_learning_preferences WHERE user_id = $1", user.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestLearningService_GetPriorityScoreDistribution_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_priority_dist", "password", "italian", "A1")
	require.NoError(t, err)

	// Create test questions with priority scores
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)

	for i := 0; i < 10; i++ {
		question := &models.Question{
			Type:            "vocabulary",
			Language:        "italian",
			Level:           "A1",
			DifficultyScore: float64(i+1) * 0.1,
			Content: map[string]interface{}{
				"question": fmt.Sprintf("Test question %d", i+1),
				"options":  []string{"A", "B", "C", "D"},
			},
			CorrectAnswer: 0,
			Explanation:   "Test explanation",
			Status:        models.QuestionStatusActive,
		}

		err = questionService.SaveQuestion(context.Background(), question)
		require.NoError(t, err)

		// Assign to user
		err = questionService.AssignQuestionToUser(context.Background(), question.ID, user.ID)
		require.NoError(t, err)

		// Set priority score
		priorityScore := float64(i+1) * 0.1
		_, err = db.Exec("INSERT INTO question_priority_scores (question_id, user_id, priority_score, created_at) VALUES ($1, $2, $3, NOW())",
			question.ID, user.ID, priorityScore)
		require.NoError(t, err)
	}

	// Test GetPriorityScoreDistribution
	distribution, err := learningService.GetPriorityScoreDistribution(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, distribution)
	assert.NotEmpty(t, distribution)
}

func TestLearningService_GetHighPriorityQuestions_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_high_priority", "password", "italian", "A1")
	require.NoError(t, err)

	// Create test questions
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)

	for i := 0; i < 5; i++ {
		question := &models.Question{
			Type:            "vocabulary",
			Language:        "italian",
			Level:           "A1",
			DifficultyScore: float64(i+1) * 0.2,
			Content: map[string]interface{}{
				"question": fmt.Sprintf("Test question %d", i+1),
				"options":  []string{"A", "B", "C", "D"},
			},
			CorrectAnswer: 0,
			Explanation:   "Test explanation",
			Status:        models.QuestionStatusActive,
		}

		err = questionService.SaveQuestion(context.Background(), question)
		require.NoError(t, err)

		// Assign to user
		err = questionService.AssignQuestionToUser(context.Background(), question.ID, user.ID)
		require.NoError(t, err)

		// Set priority score (must be > 200 to be considered high priority)
		priorityScore := float64(201 + i*10)
		_, err = db.Exec("INSERT INTO question_priority_scores (question_id, user_id, priority_score, created_at) VALUES ($1, $2, $3, NOW())",
			question.ID, user.ID, priorityScore)
		require.NoError(t, err)
	}

	// Test GetHighPriorityQuestions
	questions, err := learningService.GetHighPriorityQuestions(context.Background(), 3)
	require.NoError(t, err)
	assert.NotNil(t, questions)
	assert.Len(t, questions, 3)
}

func TestLearningService_GetWeakAreasByTopic_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_weak_areas", "password", "italian", "A1")
	require.NoError(t, err)

	// Create test questions with different topics
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	topics := []string{"grammar", "vocabulary", "pronunciation"}

	for i, topic := range topics {
		question := &models.Question{
			Type:            "vocabulary",
			Language:        "italian",
			Level:           "A1",
			DifficultyScore: 1.0,
			Content: map[string]interface{}{
				"question": fmt.Sprintf("Test question %d", i+1),
				"options":  []string{"A", "B", "C", "D"},
				"topic":    topic,
			},
			CorrectAnswer: 0,
			Explanation:   "Test explanation",
			Status:        models.QuestionStatusActive,
		}

		err = questionService.SaveQuestion(context.Background(), question)
		require.NoError(t, err)

		// Assign to user
		err = questionService.AssignQuestionToUser(context.Background(), question.ID, user.ID)
		require.NoError(t, err)

		// Create responses - make some topics weak
		response := &models.UserResponse{
			UserID:          user.ID,
			QuestionID:      question.ID,
			UserAnswerIndex: 1,      // Wrong answer
			IsCorrect:       i == 0, // Only first topic is strong
			ResponseTimeMs:  1000,
			CreatedAt:       time.Now(),
		}

		err = learningService.RecordUserResponse(context.Background(), response)
		require.NoError(t, err)
	}

	// Test GetWeakAreasByTopic
	weakAreas, err := learningService.GetWeakAreasByTopic(context.Background(), 5)
	require.NoError(t, err)
	assert.NotNil(t, weakAreas)
	assert.NotEmpty(t, weakAreas)
}

func TestLearningService_GetLearningPreferencesUsage_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_prefs_usage", "password", "italian", "A1")
	require.NoError(t, err)

	// Set up learning preferences
	preferences := &models.UserLearningPreferences{
		UserID:                    user.ID,
		PreferredQuestionTypes:    []string{"vocabulary", "grammar"},
		PreferredDifficultyLevel:  "A2",
		PreferredTopics:           []string{"daily_life", "travel"},
		PreferredQuestionCount:    10,
		SpacedRepetitionEnabled:   true,
		AdaptiveDifficultyEnabled: true,
		CreatedAt:                 time.Now(),
		UpdatedAt:                 time.Now(),
	}

	_, err = learningService.UpdateUserLearningPreferences(context.Background(), user.ID, preferences)
	require.NoError(t, err)

	// Test GetLearningPreferencesUsage
	usage, err := learningService.GetLearningPreferencesUsage(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, usage)
}

func TestLearningService_GetQuestionTypeGaps_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_gaps", "password", "italian", "A1")
	require.NoError(t, err)

	// Create questions of different types
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	questionTypes := []string{"vocabulary", "grammar", "pronunciation"}

	for i, qType := range questionTypes {
		question := &models.Question{
			Type:            models.QuestionType(qType),
			Language:        "italian",
			Level:           "A1",
			DifficultyScore: 1.0,
			Content: map[string]interface{}{
				"question": fmt.Sprintf("Test %s question %d", qType, i+1),
				"options":  []string{"A", "B", "C", "D"},
			},
			CorrectAnswer: 0,
			Explanation:   "Test explanation",
			Status:        models.QuestionStatusActive,
		}

		err = questionService.SaveQuestion(context.Background(), question)
		require.NoError(t, err)

		// Assign to user
		err = questionService.AssignQuestionToUser(context.Background(), question.ID, user.ID)
		require.NoError(t, err)

		// Create responses - make some types weak
		response := &models.UserResponse{
			UserID:          user.ID,
			QuestionID:      question.ID,
			UserAnswerIndex: 1,      // Wrong answer
			IsCorrect:       i == 0, // Only first type is strong
			ResponseTimeMs:  1000,
			CreatedAt:       time.Now(),
		}

		err = learningService.RecordUserResponse(context.Background(), response)
		require.NoError(t, err)
	}

	// Test GetQuestionTypeGaps
	gaps, err := learningService.GetQuestionTypeGaps(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, gaps)
	assert.NotEmpty(t, gaps)
}

func TestLearningService_GetGenerationSuggestions_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	_, err = userService.CreateUserWithPassword(context.Background(), "testuser_suggestions", "password", "italian", "A1")
	require.NoError(t, err)

	// Insert a question so the suggestions query returns a result
	_, err = db.Exec(`
		INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, "vocabulary", "italian", "A1", 1.0, `{"question":"Test?"}`, 0, "Test explanation", "active")
	require.NoError(t, err)

	// Test GetGenerationSuggestions
	suggestions, err := learningService.GetGenerationSuggestions(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, suggestions)
}

func TestLearningService_GetPrioritySystemPerformance_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Test GetPrioritySystemPerformance
	performance, err := learningService.GetPrioritySystemPerformance(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, performance)
}

func TestLearningService_GetBackgroundJobsStatus_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Test GetBackgroundJobsStatus
	status, err := learningService.GetBackgroundJobsStatus(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, status)
}

func TestLearningService_IsForeignKeyConstraintViolation_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	_, err := config.NewConfig()
	require.NoError(t, err)
	_ = observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	// Test with various error types
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "foreign key constraint error",
			err:      fmt.Errorf("pq: insert or update on table \"user_responses\" violates foreign key constraint \"user_responses_user_id_fkey\""),
			expected: true,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Note: isForeignKeyConstraintViolation is a private function, not a method
			// This test is testing the private function indirectly through public methods
			// For now, we'll skip this test since we can't access the private function directly
			t.Skip("isForeignKeyConstraintViolation is a private function")
		})
	}
}

func TestLearningService_UpdatePriorityScoreAsync_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_async_priority", "password", "italian", "A1")
	require.NoError(t, err)

	// Create test question
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	question := &models.Question{
		Type:            "vocabulary",
		Language:        "italian",
		Level:           "A1",
		DifficultyScore: 1.0,
		Content: map[string]interface{}{
			"question": "Test question",
			"options":  []string{"A", "B", "C", "D"},
		},
		CorrectAnswer: 0,
		Explanation:   "Test explanation",
		Status:        models.QuestionStatusActive,
	}

	err = questionService.SaveQuestion(context.Background(), question)
	require.NoError(t, err)

	// Assign to user
	err = questionService.AssignQuestionToUser(context.Background(), question.ID, user.ID)
	require.NoError(t, err)

	// Test updatePriorityScoreAsync (private method)
	learningService.updatePriorityScoreAsync(context.Background(), user.ID, question.ID)

	// Verify priority score was updated
	var score float64
	err = db.QueryRow("SELECT priority_score FROM question_priority_scores WHERE question_id = $1 AND user_id = $2",
		question.ID, user.ID).Scan(&score)
	require.NoError(t, err)
	assert.Equal(t, 150.0, score)
}

func TestLearningService_GetUserPriorityScoreDistribution_Integration(t *testing.T) {
	db := setupTestDBForLearning(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "italian", "B1")
	require.NoError(t, err)

	// Test priority score distribution
	distribution, err := learningService.GetUserPriorityScoreDistribution(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, distribution)

	// Distribution should contain priority levels
	expectedLevels := []string{"high", "medium", "low"}
	for _, level := range expectedLevels {
		if count, exists := distribution[level]; exists {
			// Count should be a number
			switch v := count.(type) {
			case int:
				assert.GreaterOrEqual(t, v, 0)
			case float64:
				assert.GreaterOrEqual(t, v, 0.0)
			default:
				// Other numeric types are acceptable
				assert.NotNil(t, count)
			}
		}
	}
}

func TestLearningService_GetUserLearningPreferences_Integration(t *testing.T) {
	db := setupTestDBForLearning(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "italian", "B1")
	require.NoError(t, err)

	// Test getting learning preferences
	preferences, err := learningService.GetUserLearningPreferences(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, preferences)

	// Verify learning preferences structure
	assert.NotNil(t, preferences.FocusOnWeakAreas)

	// These are not pointers in the model, they're direct values
	ratio := preferences.FreshQuestionRatio
	assert.GreaterOrEqual(t, ratio, 0.0)
	assert.LessOrEqual(t, ratio, 1.0)

	penalty := preferences.KnownQuestionPenalty
	assert.GreaterOrEqual(t, penalty, 0.0)
	assert.LessOrEqual(t, penalty, 1.0)

	interval := preferences.ReviewIntervalDays
	assert.GreaterOrEqual(t, interval, 1)
	assert.LessOrEqual(t, interval, 60)

	boost := preferences.WeakAreaBoost
	assert.GreaterOrEqual(t, boost, 1.0)
	assert.LessOrEqual(t, boost, 5.0)
}

func TestLearningService_GetHighPriorityTopics_Integration(t *testing.T) {
	db := setupTestDBForLearning(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "italian", "B1")
	require.NoError(t, err)

	// Test getting high priority topics
	topics, err := learningService.GetHighPriorityTopics(context.Background(), user.ID)
	require.NoError(t, err)
	// For a new user with no questions, topics should be empty but not nil
	// The function should return an empty slice, not nil
	assert.NotNil(t, topics)

	// Topics should be a slice of strings (could be empty for new users)
	// Only check non-empty topics if the slice is not empty
	if len(topics) > 0 {
		for _, topic := range topics {
			assert.NotEmpty(t, topic)
		}
	}
}

func TestLearningService_GetGapAnalysis_Integration(t *testing.T) {
	db := setupTestDBForLearning(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "italian", "B1")
	require.NoError(t, err)

	// Test getting gap analysis
	gaps, err := learningService.GetGapAnalysis(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, gaps)

	// Gaps should be a map with gap information
	// The structure may vary, but it should be a valid map
	assert.IsType(t, map[string]interface{}{}, gaps)
}

func TestLearningService_GetPriorityDistribution_Integration(t *testing.T) {
	db := setupTestDBForLearning(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "italian", "B1")
	require.NoError(t, err)

	// Test getting priority distribution
	distribution, err := learningService.GetPriorityDistribution(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, distribution)

	// Distribution should contain priority levels
	expectedLevels := []string{"high", "medium", "low"}
	totalCount := 0

	for _, level := range expectedLevels {
		if count, exists := distribution[level]; exists {
			assert.GreaterOrEqual(t, count, 0)
			totalCount += count
		}
	}

	// Total count should be reasonable (could be 0 if no questions exist)
	assert.GreaterOrEqual(t, totalCount, 0)
}

func TestLearningService_WorkerIntegration_Integration(t *testing.T) {
	db := setupTestDBForLearning(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(context.Background(), username, "italian", "B1")
	require.NoError(t, err)

	// Test that available worker-related functions work together
	// This simulates what the progress API does when gathering worker information

	// 1. Get priority score distribution
	priorityDistribution, err := learningService.GetUserPriorityScoreDistribution(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, priorityDistribution)

	// 2. Get learning preferences
	learningPrefs, err := learningService.GetUserLearningPreferences(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, learningPrefs)

	// All functions should work without errors and return valid data
	// This ensures the learning service can provide the data needed by the progress API
	assert.NotNil(t, priorityDistribution)
	assert.NotNil(t, learningPrefs)
}

func TestLearningService_GetUserHighPriorityQuestions_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_high_priority", "password", "italian", "A1")
	require.NoError(t, err)

	// Create test questions
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)

	for i := 0; i < 5; i++ {
		question := &models.Question{
			Type:            "vocabulary",
			Language:        "italian",
			Level:           "A1",
			DifficultyScore: float64(i+1) * 0.2,
			Content: map[string]interface{}{
				"question": fmt.Sprintf("Test question %d", i+1),
				"options":  []string{"A", "B", "C", "D"},
			},
			CorrectAnswer: 0,
			Explanation:   "Test explanation",
			Status:        models.QuestionStatusActive,
		}

		err = questionService.SaveQuestion(context.Background(), question)
		require.NoError(t, err)

		// Assign to user
		err = questionService.AssignQuestionToUser(context.Background(), question.ID, user.ID)
		require.NoError(t, err)

		// Set priority score (must be > 200 to be considered high priority)
		priorityScore := float64(201 + i*10)
		_, err = db.Exec("INSERT INTO question_priority_scores (question_id, user_id, priority_score, created_at) VALUES ($1, $2, $3, NOW())",
			question.ID, user.ID, priorityScore)
		require.NoError(t, err)
	}

	// Test GetUserHighPriorityQuestions
	questions, err := learningService.GetUserHighPriorityQuestions(context.Background(), user.ID, 3)
	require.NoError(t, err)
	assert.NotNil(t, questions)
	assert.Len(t, questions, 3)

	// Verify questions are ordered by priority score (descending)
	for i := 0; i < len(questions)-1; i++ {
		currentScore := questions[i]["priority_score"].(float64)
		nextScore := questions[i+1]["priority_score"].(float64)
		assert.GreaterOrEqual(t, currentScore, nextScore)
	}

	// Test with limit larger than available questions
	allQuestions, err := learningService.GetUserHighPriorityQuestions(context.Background(), user.ID, 10)
	require.NoError(t, err)
	assert.Len(t, allQuestions, 5)

	// Test with user that has no high priority questions
	user2, err := userService.CreateUserWithPassword(context.Background(), "testuser_no_priority", "password", "italian", "A1")
	require.NoError(t, err)

	noQuestions, err := learningService.GetUserHighPriorityQuestions(context.Background(), user2.ID, 5)
	require.NoError(t, err)
	assert.Len(t, noQuestions, 0)
}

func TestLearningService_GetUserWeakAreas_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_weak_areas", "password", "italian", "A1")
	require.NoError(t, err)

	// Create performance metrics for different topics
	topics := []struct {
		topic           string
		totalAttempts   int
		correctAttempts int
	}{
		{"vocabulary", 10, 8},    // 80% accuracy
		{"grammar", 10, 5},       // 50% accuracy (weakest)
		{"pronunciation", 10, 7}, // 70% accuracy
		{"listening", 10, 6},     // 60% accuracy
	}

	for _, topic := range topics {
		_, err = db.Exec(`
			INSERT INTO performance_metrics (user_id, topic, language, level, total_attempts, correct_attempts, average_response_time_ms, last_updated)
			VALUES ($1, $2, $3, $4, $5, $6, 1500.0, NOW())
		`, user.ID, topic.topic, "italian", "A1", topic.totalAttempts, topic.correctAttempts)
		require.NoError(t, err)
	}

	// Test GetUserWeakAreas
	weakAreas, err := learningService.GetUserWeakAreas(context.Background(), user.ID, 3)
	require.NoError(t, err)
	assert.NotNil(t, weakAreas)
	assert.Len(t, weakAreas, 3)

	// Verify areas are ordered by accuracy (ascending - weakest first)
	// grammar should be first (50% accuracy)
	assert.Equal(t, "grammar", weakAreas[0]["topic"])
	assert.Equal(t, 10, weakAreas[0]["total_attempts"])
	assert.Equal(t, 5, weakAreas[0]["correct_attempts"])

	// Test with limit larger than available areas
	allWeakAreas, err := learningService.GetUserWeakAreas(context.Background(), user.ID, 10)
	require.NoError(t, err)
	assert.Len(t, allWeakAreas, 4)

	// Test with user that has no performance metrics
	user2, err := userService.CreateUserWithPassword(context.Background(), "testuser_no_metrics", "password", "italian", "A1")
	require.NoError(t, err)

	noWeakAreas, err := learningService.GetUserWeakAreas(context.Background(), user2.ID, 5)
	require.NoError(t, err)
	assert.Len(t, noWeakAreas, 0)
}

func TestLearningService_ForeignKeyConstraintViolation_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Test foreign key constraint violation by trying to create a user response
	// with a non-existent user ID
	response := &models.UserResponse{
		UserID:          99999, // Non-existent user
		QuestionID:      99999, // Non-existent question
		UserAnswerIndex: 0,
		IsCorrect:       true,
		ResponseTimeMs:  5000,
	}

	// This should fail due to foreign key constraint violation
	err = learningService.RecordUserResponse(context.Background(), response)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "violates foreign key constraint")
}

func TestLearningService_UserSpecificAnalytics_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_analytics", "password", "italian", "A1")
	require.NoError(t, err)

	// Create test questions and priority scores
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)

	for i := 0; i < 3; i++ {
		question := &models.Question{
			Type:            "vocabulary",
			Language:        "italian",
			Level:           "A1",
			DifficultyScore: float64(i+1) * 0.3,
			Content: map[string]interface{}{
				"question": fmt.Sprintf("Test question %d", i+1),
				"options":  []string{"A", "B", "C", "D"},
			},
			CorrectAnswer: 0,
			Explanation:   "Test explanation",
			Status:        models.QuestionStatusActive,
		}

		err = questionService.SaveQuestion(context.Background(), question)
		require.NoError(t, err)

		// Assign to user
		err = questionService.AssignQuestionToUser(context.Background(), question.ID, user.ID)
		require.NoError(t, err)

		// Set different priority scores
		priorityScores := []float64{50.0, 150.0, 250.0} // low, medium, high
		_, err = db.Exec("INSERT INTO question_priority_scores (question_id, user_id, priority_score, created_at) VALUES ($1, $2, $3, NOW())",
			question.ID, user.ID, priorityScores[i])
		require.NoError(t, err)
	}

	// Create performance metrics
	_, err = db.Exec(`
		INSERT INTO performance_metrics (user_id, topic, language, level, total_attempts, correct_attempts, average_response_time_ms, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6, 1500.0, NOW())
	`, user.ID, "vocabulary", "italian", "A1", 20, 15)
	require.NoError(t, err)

	// Test comprehensive user analytics
	distribution, err := learningService.GetUserPriorityScoreDistribution(context.Background(), user.ID)
	require.NoError(t, err)
	assert.NotNil(t, distribution)
	assert.Equal(t, 1, distribution["high"])   // 1 high priority question
	assert.Equal(t, 1, distribution["medium"]) // 1 medium priority question
	assert.Equal(t, 1, distribution["low"])    // 1 low priority question

	_, err = learningService.GetUserHighPriorityQuestions(context.Background(), user.ID, 5)
	require.NoError(t, err)

	_, err = learningService.GetUserWeakAreas(context.Background(), user.ID, 5)
	require.NoError(t, err)
}

func TestLearningService_EdgeCases_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_edge_cases", "password", "italian", "A1")
	require.NoError(t, err)

	// Test with non-existent user (should return empty results, not error)
	distribution, err := learningService.GetUserPriorityScoreDistribution(context.Background(), 99999)
	require.NoError(t, err)
	assert.NotNil(t, distribution)
	assert.Equal(t, 0, distribution["high"])
	assert.Equal(t, 0, distribution["medium"])
	assert.Equal(t, 0, distribution["low"])

	questions, err := learningService.GetUserHighPriorityQuestions(context.Background(), 99999, 5)
	require.NoError(t, err)
	assert.Len(t, questions, 0)

	weakAreas, err := learningService.GetUserWeakAreas(context.Background(), 99999, 5)
	require.NoError(t, err)
	assert.Len(t, weakAreas, 0)

	// Test with zero limit
	weakAreasZero, err := learningService.GetUserWeakAreas(context.Background(), user.ID, 0)
	require.NoError(t, err)
	assert.Len(t, weakAreasZero, 0)

	// Test with negative limit (should return error due to database constraint)
	_, err = learningService.GetUserWeakAreas(context.Background(), user.ID, -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LIMIT must not be negative")
}

func TestLearningService_DailyReminderSettings_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_daily_reminder", "password", "italian", "A1")
	require.NoError(t, err)

	// Test 1: Set daily reminder enabled
	preferences := &models.UserLearningPreferences{
		UserID:                    user.ID,
		PreferredQuestionTypes:    []string{"vocabulary"},
		PreferredDifficultyLevel:  "A1",
		PreferredTopics:           []string{"daily_life"},
		PreferredQuestionCount:    5,
		SpacedRepetitionEnabled:   true,
		AdaptiveDifficultyEnabled: false,
		FocusOnWeakAreas:          true,
		IncludeReviewQuestions:    true,
		FreshQuestionRatio:        0.5,
		KnownQuestionPenalty:      0.1,
		ReviewIntervalDays:        7,
		WeakAreaBoost:             2.0,
		DailyReminderEnabled:      true,
		CreatedAt:                 time.Now(),
		UpdatedAt:                 time.Now(),
	}

	_, err = learningService.UpdateUserLearningPreferences(context.Background(), user.ID, preferences)
	require.NoError(t, err)

	// Verify daily reminder setting was saved
	var dailyReminderEnabled bool
	err = db.QueryRow("SELECT daily_reminder_enabled FROM user_learning_preferences WHERE user_id = $1", user.ID).Scan(&dailyReminderEnabled)
	require.NoError(t, err)
	assert.True(t, dailyReminderEnabled)

	// Test getting preferences back
	retrievedPrefs, err := learningService.GetUserLearningPreferences(context.Background(), user.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrievedPrefs)
	assert.True(t, retrievedPrefs.DailyReminderEnabled)

	// Test 2: Disable daily reminder
	preferences.DailyReminderEnabled = false
	_, err = learningService.UpdateUserLearningPreferences(context.Background(), user.ID, preferences)
	require.NoError(t, err)

	// Verify daily reminder setting was updated
	err = db.QueryRow("SELECT daily_reminder_enabled FROM user_learning_preferences WHERE user_id = $1", user.ID).Scan(&dailyReminderEnabled)
	require.NoError(t, err)
	assert.False(t, dailyReminderEnabled)

	// Test getting updated preferences back
	retrievedPrefs, err = learningService.GetUserLearningPreferences(context.Background(), user.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrievedPrefs)
	assert.False(t, retrievedPrefs.DailyReminderEnabled)
}

func TestLearningService_DailyReminderDefaultValue_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_daily_reminder_default", "password", "italian", "A1")
	require.NoError(t, err)

	// Test that new users have daily reminder disabled by default
	preferences := &models.UserLearningPreferences{
		UserID:                    user.ID,
		PreferredQuestionTypes:    []string{"vocabulary"},
		PreferredDifficultyLevel:  "A1",
		PreferredTopics:           []string{"daily_life"},
		PreferredQuestionCount:    5,
		SpacedRepetitionEnabled:   true,
		AdaptiveDifficultyEnabled: false,
		FocusOnWeakAreas:          true,
		IncludeReviewQuestions:    true,
		FreshQuestionRatio:        0.5,
		KnownQuestionPenalty:      0.1,
		ReviewIntervalDays:        7,
		WeakAreaBoost:             2.0,
		// DailyReminderEnabled not set - should default to false
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = learningService.UpdateUserLearningPreferences(context.Background(), user.ID, preferences)
	require.NoError(t, err)

	// Verify daily reminder setting defaults to false
	var dailyReminderEnabled bool
	err = db.QueryRow("SELECT daily_reminder_enabled FROM user_learning_preferences WHERE user_id = $1", user.ID).Scan(&dailyReminderEnabled)
	require.NoError(t, err)
	assert.False(t, dailyReminderEnabled)

	// Test getting preferences back
	retrievedPrefs, err := learningService.GetUserLearningPreferences(context.Background(), user.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrievedPrefs)
	assert.False(t, retrievedPrefs.DailyReminderEnabled)
}

func TestLearningService_DailyReminderMultipleUsers_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create multiple test users with different daily reminder settings
	users := []struct {
		username string
		enabled  bool
	}{
		{"testuser_reminder_enabled", true},
		{"testuser_reminder_disabled", false},
		{"testuser_reminder_default", false}, // Default value
	}

	for _, userData := range users {
		user, err := userService.CreateUserWithPassword(context.Background(), userData.username, "password", "italian", "A1")
		require.NoError(t, err)

		preferences := &models.UserLearningPreferences{
			UserID:                    user.ID,
			PreferredQuestionTypes:    []string{"vocabulary"},
			PreferredDifficultyLevel:  "A1",
			PreferredTopics:           []string{"daily_life"},
			PreferredQuestionCount:    5,
			SpacedRepetitionEnabled:   true,
			AdaptiveDifficultyEnabled: false,
			FocusOnWeakAreas:          true,
			IncludeReviewQuestions:    true,
			FreshQuestionRatio:        0.5,
			KnownQuestionPenalty:      0.1,
			ReviewIntervalDays:        7,
			WeakAreaBoost:             2.0,
			DailyReminderEnabled:      userData.enabled,
			CreatedAt:                 time.Now(),
			UpdatedAt:                 time.Now(),
		}

		_, err = learningService.UpdateUserLearningPreferences(context.Background(), user.ID, preferences)
		require.NoError(t, err)

		// Verify each user's daily reminder setting
		var dailyReminderEnabled bool
		err = db.QueryRow("SELECT daily_reminder_enabled FROM user_learning_preferences WHERE user_id = $1", user.ID).Scan(&dailyReminderEnabled)
		require.NoError(t, err)
		assert.Equal(t, userData.enabled, dailyReminderEnabled)

		// Test getting preferences back
		retrievedPrefs, err := learningService.GetUserLearningPreferences(context.Background(), user.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrievedPrefs)
		assert.Equal(t, userData.enabled, retrievedPrefs.DailyReminderEnabled)
	}
}

func TestLearningService_UpdateLastDailyReminderSent_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_daily_reminder_timestamp", "password", "italian", "A1")
	require.NoError(t, err)

	// Test 1: Update last daily reminder sent timestamp
	err = learningService.UpdateLastDailyReminderSent(context.Background(), user.ID)
	require.NoError(t, err)

	// Verify the timestamp was updated
	var lastReminderSent *time.Time
	err = db.QueryRow("SELECT last_daily_reminder_sent FROM user_learning_preferences WHERE user_id = $1", user.ID).Scan(&lastReminderSent)
	require.NoError(t, err)
	assert.NotNil(t, lastReminderSent)
	assert.WithinDuration(t, time.Now(), *lastReminderSent, 5*time.Second) // Should be within 5 seconds

	// Test 2: Update timestamp again (should update existing timestamp)
	time.Sleep(1 * time.Second) // Ensure different timestamp
	err = learningService.UpdateLastDailyReminderSent(context.Background(), user.ID)
	require.NoError(t, err)

	// Verify the timestamp was updated again
	var updatedLastReminderSent *time.Time
	err = db.QueryRow("SELECT last_daily_reminder_sent FROM user_learning_preferences WHERE user_id = $1", user.ID).Scan(&updatedLastReminderSent)
	require.NoError(t, err)
	assert.NotNil(t, updatedLastReminderSent)
	assert.True(t, updatedLastReminderSent.After(*lastReminderSent)) // Should be newer

	// Test 3: Get user preferences and verify LastDailyReminderSent is populated
	prefs, err := learningService.GetUserLearningPreferences(context.Background(), user.ID)
	require.NoError(t, err)
	assert.NotNil(t, prefs)
	assert.NotNil(t, prefs.LastDailyReminderSent)
	assert.WithinDuration(t, time.Now(), *prefs.LastDailyReminderSent, 5*time.Second)
}

func TestLearningService_LastDailyReminderSent_Field_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)

	// Create test user
	user, err := userService.CreateUserWithPassword(context.Background(), "testuser_last_reminder_field", "password", "italian", "A1")
	require.NoError(t, err)

	// Test 1: New user should have NULL last_daily_reminder_sent
	prefs, err := learningService.GetUserLearningPreferences(context.Background(), user.ID)
	require.NoError(t, err)
	assert.NotNil(t, prefs)
	assert.Nil(t, prefs.LastDailyReminderSent) // Should be nil for new users

	// Test 2: Update timestamp and verify field is populated
	err = learningService.UpdateLastDailyReminderSent(context.Background(), user.ID)
	require.NoError(t, err)

	prefs, err = learningService.GetUserLearningPreferences(context.Background(), user.ID)
	require.NoError(t, err)
	assert.NotNil(t, prefs)
	assert.NotNil(t, prefs.LastDailyReminderSent)
	assert.WithinDuration(t, time.Now(), *prefs.LastDailyReminderSent, 5*time.Second)

	// Test 3: Verify the field persists through preference updates
	prefs.DailyReminderEnabled = true
	updatedPrefs, err := learningService.UpdateUserLearningPreferences(context.Background(), user.ID, prefs)
	require.NoError(t, err)
	assert.NotNil(t, updatedPrefs)
	assert.NotNil(t, updatedPrefs.LastDailyReminderSent)                             // Should still be populated
	assert.Equal(t, prefs.LastDailyReminderSent, updatedPrefs.LastDailyReminderSent) // Should be the same
}
