//go:build integration

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

func TestGetEligibleQuestionCount_Integration(t *testing.T) {
	db := services.SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})

	userService := services.NewUserServiceWithLogger(db, cfg, logger)
	learningService := services.NewLearningServiceWithLogger(db, cfg, logger)
	questionService := services.NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	workerService := services.NewWorkerServiceWithLogger(db, logger)
	dailyQuestionService := services.NewDailyQuestionService(db, logger, questionService, learningService)

	storyService := services.NewStoryService(db, cfg, logger)
	generationHintService := services.NewGenerationHintService(db, logger)

	// Build worker using existing constructor
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)
	apnsService, _ := services.NewAPNSService(cfg, logger)
	w := NewWorker(userService, questionService, nil, learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, nil, apnsService, generationHintService, services.NewTranslationCacheRepository(db, logger), "test-instance", cfg, logger)

	// Create a user
	user, err := userService.CreateUserWithPassword(context.Background(), "integ_worker_user", "password123", "italian", "A1")
	require.NoError(t, err)

	// Ensure timezone UTC
	_, err = db.Exec("UPDATE users SET timezone = $1 WHERE id = $2", "UTC", user.ID)
	require.NoError(t, err)

	// Create 3 questions and assign to user
	qs := []*models.Question{}
	for i := 0; i < 3; i++ {
		q := &models.Question{
			Type:            models.Vocabulary,
			Language:        "italian",
			Level:           "A1",
			DifficultyScore: 1.0,
			Content: map[string]interface{}{
				"question": fmt.Sprintf("Test question %d", i),
				"options":  []string{"Option A", "Option B", "Option C", "Option D"},
			},
			CorrectAnswer: 0,
			Status:        models.QuestionStatusActive,
		}
		require.NoError(t, questionService.SaveQuestion(context.Background(), q))
		qs = append(qs, q)
		require.NoError(t, questionService.AssignQuestionToUser(context.Background(), q.ID, user.ID))
	}

	// Mark one question as recently correctly answered (within local day)
	_, err = learningService.RecordAnswerWithPriorityReturningID(context.Background(), user.ID, qs[0].ID, 0, true, 100)
	require.NoError(t, err)

	// Mark another question as old response (3 days ago)
	old := time.Now().UTC().AddDate(0, 0, -3)
	_, err = db.Exec(`INSERT INTO user_responses (user_id, question_id, user_answer_index, is_correct, response_time_ms, created_at) VALUES ($1,$2,$3,$4,$5,$6)`, user.ID, qs[1].ID, 0, true, 100, old)
	require.NoError(t, err)

	// Now get eligible count: should exclude qs[0] (recent correct), include qs[1] (old), include qs[2] (never answered) => expect 2
	count, err := w.getEligibleQuestionCount(context.Background(), user.ID, "italian", "A1", models.Vocabulary)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}
