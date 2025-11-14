//go:build integration

package services

import (
	"context"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/require"
)

func TestShouldAvoidQuestion_WithRecentCorrectResponse(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)

	// Create user
	user, err := userService.CreateUserWithPassword(context.Background(), "integ_user_shouldavoid", "password123", "italian", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Ensure timezone is UTC for predictable behavior
	_, err = db.Exec("UPDATE users SET timezone = $1 WHERE id = $2", "UTC", user.ID)
	require.NoError(t, err)

	// Create question and assign
	q := &models.Question{
		Type:            models.Vocabulary,
		Language:        "italian",
		Level:           "A1",
		DifficultyScore: 1.0,
		Content:         map[string]interface{}{"question": "Test?", "options": []string{"A", "B", "C", "D"}},
		CorrectAnswer:   0,
		Status:          models.QuestionStatusActive,
	}
	require.NoError(t, questionService.SaveQuestion(context.Background(), q))
	require.NoError(t, questionService.AssignQuestionToUser(context.Background(), q.ID, user.ID))

	// Record a correct response just now
	_, err = learningService.RecordAnswerWithPriorityReturningID(context.Background(), user.ID, q.ID, 0, true, 120)
	require.NoError(t, err)

	avoid, err := learningService.ShouldAvoidQuestion(context.Background(), user.ID, q.ID)
	require.NoError(t, err)
	require.True(t, avoid)
}

func TestShouldAvoidQuestion_WithOldCorrectResponse(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)

	// Create user
	user, err := userService.CreateUserWithPassword(context.Background(), "integ_user_shouldavoid_old", "password123", "italian", "A1")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Ensure timezone is UTC
	_, err = db.Exec("UPDATE users SET timezone = $1 WHERE id = $2", "UTC", user.ID)
	require.NoError(t, err)

	// Create question and assign
	q := &models.Question{
		Type:            models.Vocabulary,
		Language:        "italian",
		Level:           "A1",
		DifficultyScore: 1.0,
		Content:         map[string]interface{}{"question": "Test?", "options": []string{"A", "B", "C", "D"}},
		CorrectAnswer:   0,
		Status:          models.QuestionStatusActive,
	}
	require.NoError(t, questionService.SaveQuestion(context.Background(), q))
	require.NoError(t, questionService.AssignQuestionToUser(context.Background(), q.ID, user.ID))

	// Insert an old correct response (3 days ago)
	old := time.Now().UTC().AddDate(0, 0, -3)
	_, err = db.Exec(`INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, response_time_ms, created_at) VALUES ($1,$2,$3,$4,$5,$6)`, user.ID, q.ID, 0, true, 100, old)
	require.NoError(t, err)

	avoid, err := learningService.ShouldAvoidQuestion(context.Background(), user.ID, q.ID)
	require.NoError(t, err)
	require.False(t, avoid)
}
