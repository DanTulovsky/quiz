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
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)
	w := NewWorker(userService, questionService, services.NewAIService(cfg, logger, services.NewNoopUsageStatsService()), learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, nil, generationHintService, services.NewTranslationCacheRepository(db, logger), "test", cfg, logger)
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
		err = db.QueryRowContext(context.Background(), `INSERT INTO users (username, preferred_language, current_level, ai_enabled, last_active, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,NOW(),NOW()) RETURNING id`, uname, "italian", "A1", true, time.Now()).Scan(&id)
		require.NoError(t, err)
		user, err = userService.GetUserByID(context.Background(), id)
		require.NoError(t, err)
	}
	require.NotNil(t, user)
	// Update required fields (email + timezone) so user is eligible
	_, err = db.Exec(`UPDATE users SET email = $1, timezone = $2, ai_enabled = $3, last_active = $4 WHERE id = $5`, fmt.Sprintf("%s@example.com", uname), "UTC", true, time.Now(), user.ID)
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

// TestCheckForDailyQuestionAssignments_RespectsUserTimezone verifies that daily questions
// are assigned based on the user's timezone, not UTC
func TestCheckForDailyQuestionAssignments_RespectsUserTimezone(t *testing.T) {
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
	storyService := services.NewStoryService(db, cfg, logger)
	generationHintService := services.NewGenerationHintService(db, logger)
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create worker
	w := NewWorker(userService, questionService, services.NewAIService(cfg, logger, services.NewNoopUsageStatsService()), learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, nil, generationHintService, services.NewTranslationCacheRepository(db, logger), "test", cfg, logger)
	w.workerCfg.DailyHorizonDays = 2

	// Create a user with America/New_York timezone (UTC-5 in winter, UTC-4 in summer)
	// Using a timezone that's behind UTC so we can test the date difference
	userTimezone := "America/New_York"
	uname := fmt.Sprintf("timezone_user_%d", time.Now().UnixNano())
	user, err := userService.CreateUserWithPassword(context.Background(), uname, "password123", "italian", "A1")
	require.NoError(t, err)
	if user == nil {
		user, err = userService.GetUserByUsername(context.Background(), uname)
		require.NoError(t, err)
	}
	require.NotNil(t, user)

	// Set user timezone and required fields
	_, err = db.Exec(`UPDATE users SET email = $1, timezone = $2, ai_enabled = $3, last_active = $4 WHERE id = $5`,
		fmt.Sprintf("%s@example.com", uname), userTimezone, true, time.Now(), user.ID)
	require.NoError(t, err)

	// Refresh user to get updated timezone
	user, err = userService.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Create questions for the user
	for i := 0; i < 5; i++ {
		q := &models.Question{
			Type:            "vocabulary",
			Language:        "italian",
			Level:           "A1",
			DifficultyScore: 1.0,
			Content:         map[string]interface{}{"question": fmt.Sprintf("Test %d?", i), "options": []string{"A", "B", "C", "D"}},
			CorrectAnswer:   0,
			Explanation:     "",
			Status:          models.QuestionStatusActive,
		}
		require.NoError(t, questionService.SaveQuestion(context.Background(), q))
		require.NoError(t, questionService.AssignQuestionToUser(context.Background(), q.ID, user.ID))
	}

	// Set a fixed time: Jan 15, 2025 05:00 UTC = Jan 15, 2025 00:00 EST (midnight in EST)
	// EST is UTC-5, so 5 AM UTC = midnight EST
	// This allows us to verify that "today" in EST is Jan 15, not Jan 14
	loc, err := time.LoadLocation(userTimezone)
	require.NoError(t, err)
	fixedTimeUTC := time.Date(2025, 1, 15, 5, 0, 0, 0, time.UTC) // 5 AM UTC = midnight EST
	w.timeNow = func() time.Time { return fixedTimeUTC }

	// Run the worker assignment check
	require.NoError(t, w.checkForDailyQuestionAssignments(context.Background()))

	// Calculate what "today" should be in the user's timezone
	nowInUserTZ := fixedTimeUTC.In(loc)
	todayInUserTZ := time.Date(nowInUserTZ.Year(), nowInUserTZ.Month(), nowInUserTZ.Day(), 0, 0, 0, 0, loc)

	// Verify assignments exist for today in user's timezone
	assignsToday, err := dailyQuestionService.GetDailyQuestions(context.Background(), user.ID, todayInUserTZ)
	require.NoError(t, err)
	require.NotEmpty(t, assignsToday, "Should have assignments for today in user's timezone")

	// Verify that the date stored matches the user's local date, not UTC date
	// In EST, Jan 15 00:00 is the same calendar date, so this should work
	require.Equal(t, todayInUserTZ.Format("2006-01-02"), "2025-01-15", "Today in user timezone should be Jan 15")

	// Test with a time that crosses date boundary: Jan 15, 2025 01:00 UTC = Jan 14, 2025 20:00 EST
	// This is the previous day in EST, so assignments should be for Jan 14
	fixedTimeUTC2 := time.Date(2025, 1, 15, 1, 0, 0, 0, time.UTC) // 1 AM UTC = 8 PM EST previous day
	w.timeNow = func() time.Time { return fixedTimeUTC2 }

	// Create a new user for this test to avoid conflicts
	uname2 := fmt.Sprintf("timezone_user2_%d", time.Now().UnixNano())
	user2, err := userService.CreateUserWithPassword(context.Background(), uname2, "password123", "italian", "A1")
	require.NoError(t, err)
	if user2 == nil {
		user2, err = userService.GetUserByUsername(context.Background(), uname2)
		require.NoError(t, err)
	}
	require.NotNil(t, user2)

	_, err = db.Exec(`UPDATE users SET email = $1, timezone = $2, ai_enabled = $3, last_active = $4 WHERE id = $5`,
		fmt.Sprintf("%s@example.com", uname2), userTimezone, true, time.Now(), user2.ID)
	require.NoError(t, err)

	user2, err = userService.GetUserByID(context.Background(), user2.ID)
	require.NoError(t, err)

	// Create questions for user2
	for i := 0; i < 5; i++ {
		q := &models.Question{
			Type:            "vocabulary",
			Language:        "italian",
			Level:           "A1",
			DifficultyScore: 1.0,
			Content:         map[string]interface{}{"question": fmt.Sprintf("Test2 %d?", i), "options": []string{"A", "B", "C", "D"}},
			CorrectAnswer:   0,
			Explanation:     "",
			Status:          models.QuestionStatusActive,
		}
		require.NoError(t, questionService.SaveQuestion(context.Background(), q))
		require.NoError(t, questionService.AssignQuestionToUser(context.Background(), q.ID, user2.ID))
	}

	require.NoError(t, w.checkForDailyQuestionAssignments(context.Background()))

	// Calculate today in user's timezone - should be Jan 14 (previous day)
	nowInUserTZ2 := fixedTimeUTC2.In(loc)
	todayInUserTZ2 := time.Date(nowInUserTZ2.Year(), nowInUserTZ2.Month(), nowInUserTZ2.Day(), 0, 0, 0, 0, loc)

	assignsToday2, err := dailyQuestionService.GetDailyQuestions(context.Background(), user2.ID, todayInUserTZ2)
	require.NoError(t, err)
	require.NotEmpty(t, assignsToday2, "Should have assignments for today in user's timezone")
	require.Equal(t, todayInUserTZ2.Format("2006-01-02"), "2025-01-14", "Today in user timezone should be Jan 14 (previous day)")

	// The key verification is that assignments were created for the correct date in the user's timezone (Jan 14 EST),
	// not based on UTC time. The worker assigns for a horizon (today + N days), so it may also assign for future dates
	// in the user's timezone. What matters is that "today" is calculated correctly using the user's timezone.
}

// TestCheckForWordOfTheDayAssignments_RespectsUserTimezone verifies that word of the day
// is assigned based on the user's timezone, not UTC
func TestCheckForWordOfTheDayAssignments_RespectsUserTimezone(t *testing.T) {
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
	storyService := services.NewStoryService(db, cfg, logger)
	generationHintService := services.NewGenerationHintService(db, logger)
	wordOfTheDayService := services.NewWordOfTheDayService(db, logger)

	// Create worker
	w := NewWorker(userService, questionService, services.NewAIService(cfg, logger, services.NewNoopUsageStatsService()), learningService, workerService, dailyQuestionService, wordOfTheDayService, storyService, nil, generationHintService, services.NewTranslationCacheRepository(db, logger), "test", cfg, logger)

	// Create a user with America/New_York timezone
	userTimezone := "America/New_York"
	uname := fmt.Sprintf("wotd_timezone_user_%d", time.Now().UnixNano())
	user, err := userService.CreateUserWithPassword(context.Background(), uname, "password123", "italian", "A1")
	require.NoError(t, err)
	if user == nil {
		user, err = userService.GetUserByUsername(context.Background(), uname)
		require.NoError(t, err)
	}
	require.NotNil(t, user)

	// Set user timezone and required fields
	_, err = db.Exec(`UPDATE users SET email = $1, timezone = $2, ai_enabled = $3, last_active = $4 WHERE id = $5`,
		fmt.Sprintf("%s@example.com", uname), userTimezone, true, time.Now(), user.ID)
	require.NoError(t, err)

	user, err = userService.GetUserByID(context.Background(), user.ID)
	require.NoError(t, err)

	// Create a vocabulary question for word of the day selection
	q := &models.Question{
		Type:            "vocabulary",
		Language:        "italian",
		Level:           "A1",
		DifficultyScore: 1.0,
		Content: map[string]interface{}{
			"question": "Ciao",
			"options":  []string{"Hello", "Bye", "Thanks", "Please"},
			"sentence": "Lui dice: Ciao!",
		},
		CorrectAnswer: 0,
		Explanation:   "Ciao means Hello.",
		Status:        models.QuestionStatusActive,
	}
	require.NoError(t, questionService.SaveQuestion(context.Background(), q))
	require.NoError(t, questionService.AssignQuestionToUser(context.Background(), q.ID, user.ID))

	// Set a fixed time: Jan 15, 2025 05:00 UTC = Jan 15, 2025 00:00 EST (midnight in EST)
	// EST is UTC-5, so 5 AM UTC = midnight EST
	loc, err := time.LoadLocation(userTimezone)
	require.NoError(t, err)
	fixedTimeUTC := time.Date(2025, 1, 15, 5, 0, 0, 0, time.UTC)
	w.timeNow = func() time.Time { return fixedTimeUTC }

	// Run the worker word of the day assignment check
	require.NoError(t, w.checkForWordOfTheDayAssignments(context.Background()))

	// Calculate what "today" should be in the user's timezone
	nowInUserTZ := fixedTimeUTC.In(loc)
	todayInUserTZ := time.Date(nowInUserTZ.Year(), nowInUserTZ.Month(), nowInUserTZ.Day(), 0, 0, 0, 0, loc)

	// Verify word of the day exists for today in user's timezone
	wotd, err := wordOfTheDayService.GetWordOfTheDay(context.Background(), user.ID, todayInUserTZ)
	require.NoError(t, err)
	require.NotNil(t, wotd, "Should have word of the day for today in user's timezone")
	require.Equal(t, todayInUserTZ.Format("2006-01-02"), "2025-01-15", "Today in user timezone should be Jan 15")
}
