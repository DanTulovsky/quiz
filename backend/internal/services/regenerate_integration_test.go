//go:build integration
// +build integration

package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/require"
)

func createUserForRegenerateTest(t *testing.T, db *sql.DB) *models.User {
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)

	user, err := userService.CreateUser(context.Background(), "regen_user", "italian", "B1")
	require.NoError(t, err)
	return user
}

func createQuestions(t *testing.T, db *sql.DB, userID, count int) {
	for i := 0; i < count; i++ {
		q := &models.Question{
			Type:            models.Vocabulary,
			Language:        "italian",
			Level:           "B1",
			DifficultyScore: 0.5,
			Content:         map[string]interface{}{"question": "q", "options": []string{"A", "B", "C", "D"}},
			CorrectAnswer:   0,
			Explanation:     "exp",
			Status:          models.QuestionStatusActive,
		}
		contentJSON, err := json.Marshal(q.Content)
		require.NoError(t, err)
		err = db.QueryRow(`INSERT INTO questions (type, language, level, difficulty_score, content, correct_answer, explanation, status, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, q.Type, q.Language, q.Level, q.DifficultyScore, string(contentJSON), q.CorrectAnswer, q.Explanation, q.Status, time.Now()).Scan(&q.ID)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO user_questions (user_id, question_id, created_at) VALUES ($1,$2,$3)`, userID, q.ID, time.Now())
		require.NoError(t, err)
	}
}

func TestRegenerateDailyQuestions_RespectsUserGoal(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)
	dailyService := NewDailyQuestionService(db, logger, questionService, learningService)

	user := createUserForRegenerateTest(t, db)

	// set daily goal to 12
	_, err = db.Exec(`INSERT INTO user_learning_preferences (user_id, daily_goal, created_at, updated_at) VALUES ($1,$2,NOW(),NOW()) ON CONFLICT (user_id) DO UPDATE SET daily_goal = EXCLUDED.daily_goal`, user.ID, 12)
	require.NoError(t, err)

	createQuestions(t, db, user.ID, 20)

	date := time.Now().UTC().Truncate(24 * time.Hour)
	err = dailyService.RegenerateDailyQuestions(context.Background(), user.ID, date)
	require.NoError(t, err)

	var assigned int
	err = db.QueryRow(`SELECT COUNT(*) FROM daily_question_assignments WHERE user_id = $1 AND assignment_date = $2`, user.ID, date).Scan(&assigned)
	require.NoError(t, err)
	require.Equal(t, 12, assigned)
}
