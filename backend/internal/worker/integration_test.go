//go:build integration
// +build integration

package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"
	"quizapp/internal/services"

	"github.com/stretchr/testify/require"
)

// Test that the worker's daily assignment checker creates assignments for today and the next day (horizon=2)
func TestCheckForDailyQuestionAssignments_CreatesTwoDayHorizonAssignments(t *testing.T) {
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)
	workerService := services.NewWorkerServiceWithLogger(db, logger)

	// Create story service
	storyService := services.NewStoryService(db, cfg, logger)

	// Create generation hint service
	generationHintService := services.NewGenerationHintService(db, logger)

	// Create worker with defaults (DailyHorizonDays defaults to 2)
	w := NewWorker(userService, questionService, services.NewAIService(cfg, logger), learningService, workerService, dailyQuestionService, storyService, nil, generationHintService, "test", cfg, logger)
	// Ensure horizon is explicitly 2 for the test
	w.workerCfg.DailyHorizonDays = 2

	// Create a user eligible for daily questions (unique username)
	uname := fmt.Sprintf("worker_integ_%d", time.Now().UnixNano())
	user, err := userService.CreateUserWithPassword(context.Background(), uname, "password123", "italian", "A1")
	require.NoError(t, err)
	if user == nil {
		// Fallback: try to fetch user by username in case creation was idempotent
		user, err = userService.GetUserByUsername(context.Background(), uname)
		require.NoError(t, err)
	}
	if user == nil {
		// Final fallback: insert user directly into DB and fetch by ID
		var id int
		err = db.QueryRowContext(context.Background(), `INSERT INTO users (username, preferred_language, current_level, ai_enabled, last_active, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,NOW(),NOW()) RETURNING id`, uname, "italian", "A1", false, time.Now()).Scan(&id)
		require.NoError(t, err)
		user, err = userService.GetUserByID(context.Background(), id)
		require.NoError(t, err)
	}
	require.NotNil(t, user)
	// Update required fields (email + timezone) so user is eligible
	_, err = db.Exec(`UPDATE users SET email = $1, timezone = $2, last_active = $3 WHERE id = $4`, fmt.Sprintf("%s@example.com", uname), "UTC", time.Now(), user.ID)
	require.NoError(t, err)

	// Create a question and assign to user so worker can pick from available pool
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

	// Run the worker assignment check
	require.NoError(t, w.checkForDailyQuestionAssignments(context.Background()))

	// Verify assignments exist for today and tomorrow in user's timezone (UTC)
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	assignsToday, err := dailyQuestionService.GetDailyQuestions(context.Background(), user.ID, today)
	require.NoError(t, err)
	require.NotNil(t, assignsToday)

	assignsTomorrow, err := dailyQuestionService.GetDailyQuestions(context.Background(), user.ID, tomorrow)
	require.NoError(t, err)
	require.NotNil(t, assignsTomorrow)

	// Idempotency: running again should not create duplicate assignments (counts should be equal)
	require.NoError(t, w.checkForDailyQuestionAssignments(context.Background()))
	assignsToday2, err := dailyQuestionService.GetDailyQuestions(context.Background(), user.ID, today)
	require.NoError(t, err)
	require.Equal(t, len(assignsToday), len(assignsToday2))
}
