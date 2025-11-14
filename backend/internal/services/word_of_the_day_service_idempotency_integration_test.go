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

func TestWordOfTheDay_GetWordOfTheDay_Idempotent_Integration(t *testing.T) {
	// Arrange DB and services
	db := SharedTestDBSetup(t)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	cfg, err := config.NewConfig()
	require.NoError(t, err)

	userSvc := NewUserServiceWithLogger(db, cfg, logger)
	learningSvc := NewLearningServiceWithLogger(db, cfg, logger)
	questionSvc := NewQuestionServiceWithLogger(db, learningSvc, cfg, logger)
	wotd := NewWordOfTheDayService(db, logger)

	// Create a user with language/level so WOTD selection can proceed
	user, err := userSvc.CreateUser(context.Background(), "wotd-idem-user", "italian", "A2")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Seed at least one active vocabulary question in user's language/level so selection can succeed
	question := &models.Question{
		Type:            models.Vocabulary,
		Language:        "italian",
		Level:           "A2",
		DifficultyScore: 0.3,
		Content: map[string]interface{}{
			"question": "Ciao",
			"options":  []string{"Hello", "Bye", "Thanks", "Please"},
			"sentence": "Lui dice: Ciao!",
		},
		CorrectAnswer: 0,
		Explanation:   "Ciao means Hello.",
		Status:        models.QuestionStatusActive,
	}
	err = questionSvc.SaveQuestion(context.Background(), question)
	require.NoError(t, err)

	// Use a deterministic date (today in UTC date-only)
	now := time.Now().UTC()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Act: first call creates/returns word
	first, err := wotd.GetWordOfTheDay(context.Background(), user.ID, day)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NotEmpty(t, first.Word)

	// Act: second call same user/date should return the same assignment (idempotent)
	second, err := wotd.GetWordOfTheDay(context.Background(), user.ID, day)
	require.NoError(t, err)
	require.NotNil(t, second)

	// Assert: stable fields remain identical (word/translation/source)
	require.Equal(t, first.Word, second.Word)
	require.Equal(t, first.Translation, second.Translation)
	require.Equal(t, first.SourceType, second.SourceType)
	require.Equal(t, first.SourceID, second.SourceID)
}
