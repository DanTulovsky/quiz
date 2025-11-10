//go:build integration

package services

import (
	"context"
	"fmt"
	"testing"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuestionService_GetAdaptiveQuestionsForDaily_NoDuplicates(t *testing.T) {
	// Setup test database using the shared pattern
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	// Create services
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)

	ctx := context.Background()

	// Create a test user using the user service
	user, err := userService.CreateUser(ctx, "testuser", "italian", "B1")
	require.NoError(t, err)

	// Create test questions using the existing helper function
	createTestQuestionsForDaily(t, db, user.ID, 12)

	// Test 1: Check for duplicates within a single call
	t.Run("NoDuplicatesWithinSingleCall", func(t *testing.T) {
		questions, err := questionService.GetAdaptiveQuestionsForDaily(ctx, user.ID, "italian", "B1", 10)
		require.NoError(t, err)
		require.Len(t, questions, 10)

		// Check for duplicates
		questionIDs := make(map[int]bool)
		for _, q := range questions {
			assert.False(t, questionIDs[q.ID], "Duplicate question ID found: %d", q.ID)
			questionIDs[q.ID] = true
		}
		assert.Len(t, questionIDs, 10, "Expected 10 unique questions")
	})

	// Test 2: Check edge case with limited questions
	t.Run("NoDuplicatesWithLimitedQuestions", func(t *testing.T) {
		// Try to get more questions than available
		questions, err := questionService.GetAdaptiveQuestionsForDaily(ctx, user.ID, "italian", "B1", 20)
		require.NoError(t, err)
		// Should get all available questions (12 in this case)
		assert.Len(t, questions, 12)

		// Check for duplicates
		questionIDs := make(map[int]bool)
		for _, q := range questions {
			assert.False(t, questionIDs[q.ID], "Duplicate question ID found: %d", q.ID)
			questionIDs[q.ID] = true
		}
		assert.Len(t, questionIDs, 12, "Expected 12 unique questions")
	})

	// Test 3: Check that questions are properly distributed across types
	t.Run("QuestionTypeDistribution", func(t *testing.T) {
		questions, err := questionService.GetAdaptiveQuestionsForDaily(ctx, user.ID, "italian", "B1", 10)
		require.NoError(t, err)
		require.Len(t, questions, 10)

		// Count questions by type
		typeCounts := make(map[models.QuestionType]int)
		for _, q := range questions {
			typeCounts[q.Type]++
		}

		// Should have questions from all types (vocabulary, fill_in_blank, question_answer, reading_comprehension)
		assert.Greater(t, typeCounts[models.Vocabulary], 0, "Should have vocabulary questions")
		assert.Greater(t, typeCounts[models.FillInBlank], 0, "Should have fill-in-blank questions")
		assert.Greater(t, typeCounts[models.QuestionAnswer], 0, "Should have question-answer questions")
		assert.Greater(t, typeCounts[models.ReadingComprehension], 0, "Should have reading comprehension questions")

		// Check for duplicates
		questionIDs := make(map[int]bool)
		for _, q := range questions {
			assert.False(t, questionIDs[q.ID], "Duplicate question ID found: %d", q.ID)
			questionIDs[q.ID] = true
		}
		assert.Len(t, questionIDs, 10, "Expected 10 unique questions")
	})

	// Test 4: Check with different question counts to ensure no duplicates
	t.Run("NoDuplicatesWithDifferentCounts", func(t *testing.T) {
		testCases := []int{3, 5, 8, 10}

		for _, count := range testCases {
			t.Run(fmt.Sprintf("Count%d", count), func(t *testing.T) {
				questions, err := questionService.GetAdaptiveQuestionsForDaily(ctx, user.ID, "italian", "B1", count)
				require.NoError(t, err)

				// Should get the requested number of questions (or all available if less)
				expectedCount := count
				if count > 12 {
					expectedCount = 12
				}
				assert.Len(t, questions, expectedCount)

				// Check for duplicates
				questionIDs := make(map[int]bool)
				for _, q := range questions {
					assert.False(t, questionIDs[q.ID], "Duplicate question ID found for count %d: %d", count, q.ID)
					questionIDs[q.ID] = true
				}
				assert.Len(t, questionIDs, expectedCount, "Expected %d unique questions for count %d", expectedCount, count)
			})
		}
	})
}

// New test to verify interleaving of question types in adaptive selection
func TestQuestionService_GetAdaptiveQuestionsForDaily_Interleaving(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer CleanupTestDatabase(db, t)

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	learningService := NewLearningServiceWithLogger(db, cfg, logger)
	questionService := NewQuestionServiceWithLogger(db, learningService, cfg, logger)

	ctx := context.Background()

	// Create a test user
	user, err := userService.CreateUser(ctx, "interleave_user", "italian", "B1")
	require.NoError(t, err)

	// Create a pool of 20 questions, arranged so each consecutive group cycles types
	createTestQuestionsForDaily(t, db, user.ID, 20)

	// Request 10 questions
	questions, err := questionService.GetAdaptiveQuestionsForDaily(ctx, user.ID, "italian", "B1", 10)
	require.NoError(t, err)
	require.Len(t, questions, 10)

	// Extract types in order
	var observedTypes []models.QuestionType
	for _, q := range questions {
		observedTypes = append(observedTypes, q.Type)
	}

	// Ensure that at least two different types alternate in the first 8 items
	typeSeen := make(map[models.QuestionType]int)
	for i, tpe := range observedTypes {
		typeSeen[tpe]++
		// check that no single type dominates the prefix
		if i < 8 {
			if typeSeen[tpe] > 4 {
				t.Fatalf("Type %s appears too frequently in prefix: %d", tpe, typeSeen[tpe])
			}
		}
	}

	// Verify that reading comprehension appears at least once in the 10
	if typeSeen[models.ReadingComprehension] == 0 {
		t.Fatalf("Expected at least one reading comprehension question in selection, got none. Types: %v", observedTypes)
	}
}
