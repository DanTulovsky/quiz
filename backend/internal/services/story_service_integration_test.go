//go:build integration
// +build integration

package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"quizapp/internal/config"
	"quizapp/internal/models"
	"quizapp/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStoryService_CanGenerateSection_Integration tests the CanGenerateSection functionality
func TestStoryService_CanGenerateSection_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	cfg.Story.MaxExtraGenerationsPerDay = 1 // Set explicit value for test
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	storyService := NewStoryService(db, cfg, logger)

	ctx := context.Background()

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(ctx, username, "italian", "A1")
	require.NoError(t, err)

	// Create a test story
	story, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:       "Test Story",
		Subject:     stringPtr("Test Subject"),
		AuthorStyle: stringPtr("Simple"),
	})
	require.NoError(t, err)
	require.NotNil(t, story)

	// Test 1: Should be able to generate section initially
	eligibility, err := storyService.canGenerateSection(ctx, story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate section initially")

	// Test 2: Create a section for today (user generation)
	sectionContent := "This is a test story section."
	section, err := storyService.CreateSection(ctx, story.ID, sectionContent, "A1", 50, models.GeneratorTypeUser)
	require.NoError(t, err)
	require.NotNil(t, section)

	// Test 2b: Update the generation time after first section creation (simulating what the handler does)
	err = storyService.UpdateLastGenerationTime(ctx, story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)

	// Test 3: Debug - check what's actually in the database
	var sectionCount int
	today := time.Now().Truncate(24 * time.Hour)
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM story_sections WHERE story_id = $1 AND generation_date = $2", story.ID, today).Scan(&sectionCount)
	require.NoError(t, err)
	t.Logf("Found %d sections for story %d on date %v", sectionCount, story.ID, today)

	// Test 3b: Should still be able to generate another section today (first user section doesn't reach limit with MaxExtraGenerationsPerDay=1)
	eligibility, err = storyService.canGenerateSection(ctx, story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)
	t.Logf("CanGenerateSection returned: %v (expected true)", eligibility.CanGenerate)
	assert.True(t, eligibility.CanGenerate, "Should still be able to generate another user section today")

	// Test 3c: Create a second section (should work with MaxExtraGenerationsPerDay=1)
	section2, err := storyService.CreateSection(ctx, story.ID, "Second test story section.", "A1", 50, models.GeneratorTypeUser)
	require.NoError(t, err)
	require.NotNil(t, section2)

	// Update generation time for second section
	err = storyService.UpdateLastGenerationTime(ctx, story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)

	// Test 3d: Should not be able to create a third section (limit reached after 2 user sections with MaxExtraGenerationsPerDay=1)
	eligibility2, err := storyService.canGenerateSection(ctx, story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)
	t.Logf("CanGenerateSection returned: %v (expected false)", eligibility2.CanGenerate)
	assert.False(t, eligibility2.CanGenerate, "Should not be able to generate third user section today")

	// Test 4: Test with a different story (should be able to generate)
	otherStory, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:   "Other Story",
		Subject: stringPtr("Other Subject"),
	})
	require.NoError(t, err)

	eligibility, err = storyService.canGenerateSection(ctx, otherStory.ID, models.GeneratorTypeUser)
	require.NoError(t, err)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate section for different story")
}

// TestStoryService_UpdateLastGenerationTime_Integration tests the UpdateLastGenerationTime functionality
func TestStoryService_UpdateLastGenerationTime_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	storyService := NewStoryService(db, cfg, logger)

	ctx := context.Background()

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(ctx, username, "italian", "A1")
	require.NoError(t, err)

	// Create a test story
	story, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:       "Test Story",
		Subject:     stringPtr("Test Subject"),
		AuthorStyle: stringPtr("Simple"),
	})
	require.NoError(t, err)
	require.NotNil(t, story)

	// Test 1: First generation today (worker generation) - should not increment extra_generations_today
	err = storyService.UpdateLastGenerationTime(ctx, story.ID, models.GeneratorTypeWorker)
	require.NoError(t, err)

	var extraGenerations int
	err = db.QueryRowContext(ctx, "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 0, extraGenerations, "Worker generation should not increment extra_generations_today")

	// Test 2: Second generation today (user generation) - should increment extra_generations_today to 1
	err = storyService.UpdateLastGenerationTime(ctx, story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 1, extraGenerations, "User generation should increment extra_generations_today to 1")

	// Test 3: Third generation today (user generation) - should increment extra_generations_today to 2
	err = storyService.UpdateLastGenerationTime(ctx, story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 2, extraGenerations, "Third generation should increment extra_generations_today to 2")
}

// TestStoryService_StoryGenerationLimits_Integration tests that story generation is properly limited
func TestStoryService_StoryGenerationLimits_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	today := time.Now().Truncate(24 * time.Hour)
	var debugSectionCount int

	// Create a config with story limits
	cfg, err := config.NewConfig()
	require.NoError(t, err)
	cfg.Story.MaxArchivedPerUser = 20
	cfg.Story.GenerationEnabled = true
	cfg.Story.MaxExtraGenerationsPerDay = 1
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	storyService := NewStoryService(db, cfg, logger)

	// Create a test user
	user := createTestUser(t, db)

	// Clean up any existing stories and sections for the test user
	_, err = db.ExecContext(context.Background(), "DELETE FROM story_sections WHERE story_id IN (SELECT id FROM stories WHERE user_id = $1)", user.ID)
	require.NoError(t, err)
	_, err = db.ExecContext(context.Background(), "DELETE FROM stories WHERE user_id = $1", user.ID)
	require.NoError(t, err)

	// Create a story with a unique title to ensure it's fresh
	story, err := storyService.CreateStory(context.Background(), uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:   "Test Story Generation Limits " + fmt.Sprintf("%d", time.Now().Unix()),
		Subject: stringPtr("Test Subject for Generation Limits"),
	})
	require.NoError(t, err)
	require.NotNil(t, story)

	// Debug: Check that the story is clean
	err = db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM story_sections WHERE story_id = $1 AND generation_date = $2", story.ID, today).Scan(&debugSectionCount)
	require.NoError(t, err)
	t.Logf("DEBUG: After story creation, story.ID=%d, sections today=%d", story.ID, debugSectionCount)

	// Test 1: Initially, should be able to generate (no sections exist today)
	eligibility, err := storyService.canGenerateSection(context.Background(), story.ID, models.GeneratorTypeWorker)
	require.NoError(t, err)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate section initially")

	// Test 2: Create first section (worker generation)
	section, err := storyService.CreateSection(context.Background(), story.ID, "Worker section", "A1", 50, models.GeneratorTypeWorker)
	require.NoError(t, err)
	require.NotNil(t, section)

	// Update generation time for worker generation
	err = storyService.UpdateLastGenerationTime(context.Background(), story.ID, models.GeneratorTypeWorker)
	require.NoError(t, err)

	var extraGenerations int
	err = db.QueryRowContext(context.Background(), "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 0, extraGenerations, "Worker generation should not increment extra_generations_today")

	// Debug: Check what values we have
	var sectionCount int
	err = db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM story_sections WHERE story_id = $1 AND generation_date = $2", story.ID, today).Scan(&sectionCount)
	require.NoError(t, err)

	err = db.QueryRowContext(context.Background(), "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)

	t.Logf("After worker generation: sectionCount=%d, extraGenerationsToday=%d", sectionCount, extraGenerations)
	assert.Equal(t, 0, extraGenerations, "Worker generation should not increment extra_generations_today")

	// Test 3: Should be able to generate user section after worker generation (worker limit not reached)
	eligibility, err = storyService.canGenerateSection(context.Background(), story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)
	t.Logf("CanGenerateSection returned: %v (expected true)", eligibility.CanGenerate)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate user section after worker generation")

	// Test 4: But users should be able to generate extra sections up to the configured limit
	// Reset the story state to simulate a new day or reset the counter
	_, err = db.ExecContext(context.Background(), "UPDATE stories SET extra_generations_today = 0 WHERE id = $1", story.ID)
	require.NoError(t, err)

	// Create a section to simulate worker generation
	_, err = storyService.CreateSection(context.Background(), story.ID, "Worker section", "A1", 50, models.GeneratorTypeWorker)
	require.NoError(t, err)

	// Update generation time for worker generation
	err = storyService.UpdateLastGenerationTime(context.Background(), story.ID, models.GeneratorTypeWorker)
	require.NoError(t, err)

	// Test 5: Should be able to generate user section (extra_generations_today is 1)
	eligibility, err = storyService.canGenerateSection(context.Background(), story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate user section after worker generation")

	// Test 6: Create user section and update generation time
	_, err = storyService.CreateSection(context.Background(), story.ID, "User section", "A1", 40, models.GeneratorTypeUser)
	require.NoError(t, err)

	err = storyService.UpdateLastGenerationTime(context.Background(), story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)

	err = db.QueryRowContext(context.Background(), "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	expectedTotal := 1 // User generation increments to 1 (only user generations count)
	assert.Equal(t, expectedTotal, extraGenerations, "User generation should increment extra_generations_today to 1")

	// Test 7: Should still be able to generate second section (limit not reached after 1 user section with MaxExtraGenerationsPerDay=1)
	eligibility, err = storyService.canGenerateSection(context.Background(), story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)
	assert.True(t, eligibility.CanGenerate, "Should still be able to generate second section after user generation")

	// Create second user section
	_, err = storyService.CreateSection(context.Background(), story.ID, "Second user section", "A1", 40, models.GeneratorTypeUser)
	require.NoError(t, err)

	err = storyService.UpdateLastGenerationTime(context.Background(), story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)

	// Test 7b: Should not be able to generate third section (limit reached after 2 user sections)
	eligibility, err = storyService.canGenerateSection(context.Background(), story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)
	assert.False(t, eligibility.CanGenerate, "Should not be able to generate third section after two user generations")

	// Test 8: Test with MaxExtraGenerationsPerDay = 1 (allow 1 extra user generation)
	cfg2, err := config.NewConfig()
	require.NoError(t, err)
	cfg2.Story.MaxArchivedPerUser = 20
	cfg2.Story.GenerationEnabled = true
	cfg2.Story.MaxExtraGenerationsPerDay = 1
	logger2 := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	storyService2 := NewStoryService(db, cfg2, logger2)

	// Clean up the story for the new test
	_, err = db.ExecContext(context.Background(), "DELETE FROM story_sections WHERE story_id = $1", story.ID)
	require.NoError(t, err)
	_, err = db.ExecContext(context.Background(), "UPDATE stories SET extra_generations_today = 0 WHERE id = $1", story.ID)
	require.NoError(t, err)

	// Should be able to generate worker section (no sections exist after cleanup)
	eligibility, err = storyService2.canGenerateSection(context.Background(), story.ID, models.GeneratorTypeWorker)
	require.NoError(t, err)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate worker section after cleanup")

	// Debug: Check what sections exist for this story
	err = db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM story_sections WHERE story_id = $1 AND generation_date = $2", story.ID, today).Scan(&debugSectionCount)
	require.NoError(t, err)
	var debugUserSections int
	err = db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM story_sections WHERE story_id = $1 AND generation_date = $2 AND generated_by = 'user'", story.ID, today).Scan(&debugUserSections)
	require.NoError(t, err)
	t.Logf("DEBUG: story.ID=%d, total sections today=%d, user sections today=%d", story.ID, debugSectionCount, debugUserSections)

	// Should be able to generate user section (user limit not reached yet)
	eligibility, err = storyService2.canGenerateSection(context.Background(), story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)
	t.Logf("CanGenerateSection for user returned: %v, reason: %s", eligibility.CanGenerate, eligibility.Reason)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate user section after worker generation")

	// Create user section
	_, err = storyService2.CreateSection(context.Background(), story.ID, "User section 2", "A1", 40, models.GeneratorTypeUser)
	require.NoError(t, err)

	// Update generation time for user generation
	err = storyService2.UpdateLastGenerationTime(context.Background(), story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)

	// Check that extra_generations_today was incremented to 1
	var extraGenerations2 int
	err = db.QueryRowContext(context.Background(), "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations2)
	require.NoError(t, err)
	assert.Equal(t, 1, extraGenerations2, "User generation should increment extra_generations_today to 1")

	// Should be able to generate second user section (limit not reached after 1 user section with MaxExtraGenerationsPerDay=1)
	eligibility, err = storyService2.canGenerateSection(context.Background(), story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate second user section after first user section")

	// Create second user section
	_, err = storyService2.CreateSection(context.Background(), story.ID, "Second user section", "A1", 40, models.GeneratorTypeUser)
	require.NoError(t, err)

	// Update generation time for user generation
	err = storyService2.UpdateLastGenerationTime(context.Background(), story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)

	// Should not be able to generate third user section (limit reached after 2 user sections with MaxExtraGenerationsPerDay=1)
	eligibility, err = storyService2.canGenerateSection(context.Background(), story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)
	assert.False(t, eligibility.CanGenerate, "Should not be able to generate third user section (limit reached)")
}

// TestStoryService_CreateSection_Integration tests section creation
func TestStoryService_CreateSection_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	storyService := NewStoryService(db, cfg, logger)

	ctx := context.Background()

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(ctx, username, "italian", "A1")
	require.NoError(t, err)

	// Create a test story
	story, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:   "Test Story",
		Subject: stringPtr("Test Subject"),
	})
	require.NoError(t, err)

	// Test creating a section
	sectionContent := "This is a test story section with enough content to count words properly."
	section, err := storyService.CreateSection(ctx, story.ID, sectionContent, "A1", 50, models.GeneratorTypeUser)
	require.NoError(t, err)
	require.NotNil(t, section)

	// Verify section properties
	assert.Equal(t, story.ID, section.StoryID)
	assert.Equal(t, 1, section.SectionNumber)
	assert.Equal(t, sectionContent, section.Content)
	assert.Equal(t, "A1", section.LanguageLevel)
	assert.Greater(t, section.WordCount, 0)
	assert.NotEmpty(t, section.GeneratedAt)
	assert.NotEmpty(t, section.GenerationDate)

	// Test that creating a second section on the same day works (increments extra_generations_today)
	section2, err := storyService.CreateSection(ctx, story.ID, "Second section content.", "A1", 30, models.GeneratorTypeUser)
	require.NoError(t, err)
	require.NotNil(t, section2)
	assert.Equal(t, 2, section2.SectionNumber)

	// Update the generation time (simulating what the handler does)
	err = storyService.UpdateLastGenerationTime(ctx, story.ID, models.GeneratorTypeUser)
	require.NoError(t, err)

	// Verify that extra_generations_today was incremented
	var extraGenerations int
	err = db.QueryRowContext(ctx, "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 1, extraGenerations, "extra_generations_today should be 1 after second section")

	// Test creating a section for a different story (should work)
	otherStory, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:   "Other Story",
		Subject: stringPtr("Other Subject"),
	})
	require.NoError(t, err)

	// Creating a section for a different story should work
	otherSection, err := storyService.CreateSection(ctx, otherStory.ID, "Other story section.", "A1", 25, models.GeneratorTypeUser)
	require.NoError(t, err)
	require.NotNil(t, otherSection)

	assert.Equal(t, otherStory.ID, otherSection.StoryID)
	assert.Equal(t, 1, otherSection.SectionNumber)
	assert.Equal(t, "Other story section.", otherSection.Content)
}

// TestStoryService_GetLatestSection_Integration tests retrieving the latest section
func TestStoryService_GetLatestSection_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	storyService := NewStoryService(db, cfg, logger)

	ctx := context.Background()

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(ctx, username, "italian", "A1")
	require.NoError(t, err)

	// Create a test story
	story, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:   "Test Story",
		Subject: stringPtr("Test Subject"),
	})
	require.NoError(t, err)

	// Initially no sections
	latestSection, err := storyService.GetLatestSection(ctx, story.ID)
	require.NoError(t, err)
	assert.Nil(t, latestSection)

	// Create first section
	section1, err := storyService.CreateSection(ctx, story.ID, "First section.", "A1", 20, models.GeneratorTypeUser)
	require.NoError(t, err)

	// Should return the first section
	latestSection, err = storyService.GetLatestSection(ctx, story.ID)
	require.NoError(t, err)
	require.NotNil(t, latestSection)
	assert.Equal(t, section1.ID, latestSection.ID)
	assert.Equal(t, 1, latestSection.SectionNumber)

	// Test that creating a second section on the same day works (no database constraint)
	section2, err := storyService.CreateSection(ctx, story.ID, "Second section.", "A1", 25, models.GeneratorTypeUser)
	require.NoError(t, err)

	// Latest section should now be the second section
	latestSection, err = storyService.GetLatestSection(ctx, story.ID)
	require.NoError(t, err)
	require.NotNil(t, latestSection)
	assert.Equal(t, section2.ID, latestSection.ID)
	assert.Equal(t, 2, latestSection.SectionNumber)
}

// TestStoryService_CreateSectionQuestions_Integration tests the CreateSectionQuestions functionality
func TestStoryService_CreateSectionQuestions_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	storyService := NewStoryService(db, cfg, logger)

	ctx := context.Background()

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(ctx, username, "italian", "A1")
	require.NoError(t, err)

	// Create a test story
	story, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:   "Test Story",
		Subject: stringPtr("Test Subject"),
	})
	require.NoError(t, err)

	// Create a test section
	section, err := storyService.CreateSection(ctx, story.ID, "This is a test section with some content to test questions.", "A1", 50, models.GeneratorTypeUser)
	require.NoError(t, err)
	require.NotNil(t, section)

	// Create test questions data (this is what would come from the AI service)
	questions := []models.StorySectionQuestionData{
		{
			QuestionText:       "What is the main topic of this section?",
			Options:            []string{"Adventure", "Romance", "Mystery", "History"},
			CorrectAnswerIndex: 2,
			Explanation:        stringPtr("The section is about a mysterious adventure."),
		},
		{
			QuestionText:       "Who is the main character?",
			Options:            []string{"Alice", "Bob", "Charlie", "Diana"},
			CorrectAnswerIndex: 0,
			Explanation:        stringPtr("Alice is clearly the main character in this section."),
		},
		{
			QuestionText:       "Where does the story take place?",
			Options:            []string{"Forest", "City", "Mountain", "Ocean"},
			CorrectAnswerIndex: 1,
			Explanation:        stringPtr("The story takes place in a bustling city."),
		},
	}

	// Test creating questions - this should not fail with the original error
	err = storyService.CreateSectionQuestions(ctx, section.ID, questions)
	require.NoError(t, err, "CreateSectionQuestions should succeed with properly formatted options")

	// Verify that questions were saved correctly by retrieving them
	savedQuestions, err := storyService.GetSectionQuestions(ctx, section.ID)
	require.NoError(t, err)
	require.Len(t, savedQuestions, 3, "Should have saved 3 questions")

	// Verify the first question details
	firstQuestion := savedQuestions[0]
	assert.Equal(t, section.ID, firstQuestion.SectionID)
	assert.Equal(t, "What is the main topic of this section?", firstQuestion.QuestionText)
	assert.Equal(t, []string{"Adventure", "Romance", "Mystery", "History"}, firstQuestion.Options)
	assert.Equal(t, 2, firstQuestion.CorrectAnswerIndex)
	assert.Equal(t, "The section is about a mysterious adventure.", *firstQuestion.Explanation)
	assert.NotEmpty(t, firstQuestion.CreatedAt)

	// Verify the second question details
	secondQuestion := savedQuestions[1]
	assert.Equal(t, section.ID, secondQuestion.SectionID)
	assert.Equal(t, "Who is the main character?", secondQuestion.QuestionText)
	assert.Equal(t, []string{"Alice", "Bob", "Charlie", "Diana"}, secondQuestion.Options)
	assert.Equal(t, 0, secondQuestion.CorrectAnswerIndex)
	assert.Equal(t, "Alice is clearly the main character in this section.", *secondQuestion.Explanation)

	// Verify the third question details
	thirdQuestion := savedQuestions[2]
	assert.Equal(t, section.ID, thirdQuestion.SectionID)
	assert.Equal(t, "Where does the story take place?", thirdQuestion.QuestionText)
	assert.Equal(t, []string{"Forest", "City", "Mountain", "Ocean"}, thirdQuestion.Options)
	assert.Equal(t, 1, thirdQuestion.CorrectAnswerIndex)
	assert.Equal(t, "The story takes place in a bustling city.", *thirdQuestion.Explanation)

	// Test that GetRandomQuestions also works with the saved questions
	randomQuestions, err := storyService.GetRandomQuestions(ctx, section.ID, 2)
	require.NoError(t, err)
	require.Len(t, randomQuestions, 2, "Should return 2 random questions")
	assert.NotEmpty(t, randomQuestions[0].Options, "Random questions should have options")
	assert.NotEmpty(t, randomQuestions[1].Options, "Random questions should have options")
}

// TestStoryService_GetStorySections_Integration tests the GetStorySections functionality
func TestStoryService_GetStorySections_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	storyService := NewStoryService(db, cfg, logger)

	ctx := context.Background()

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(ctx, username, "italian", "A1")
	require.NoError(t, err)

	// Create a test story
	story, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:       "Test Story for Sections",
		Subject:     stringPtr("Test Subject"),
		AuthorStyle: stringPtr("Simple"),
	})
	require.NoError(t, err)
	require.NotNil(t, story)

	// Initially, story should have no sections
	sections, err := storyService.GetStorySections(ctx, story.ID)
	require.NoError(t, err)
	assert.Len(t, sections, 0, "New story should have no sections")

	// Create a test section
	section, err := storyService.CreateSection(ctx, story.ID, "This is the first section of the story.", "A1", 50, models.GeneratorTypeUser)
	require.NoError(t, err)
	require.NotNil(t, section)
	assert.Equal(t, 1, section.SectionNumber)
	assert.Equal(t, "This is the first section of the story.", section.Content)
	assert.Equal(t, "A1", section.LanguageLevel)
	assert.Equal(t, 50, section.WordCount)
	assert.Equal(t, models.GeneratorTypeUser, section.GeneratedBy)

	// Now get sections and verify we have one
	sections, err = storyService.GetStorySections(ctx, story.ID)
	require.NoError(t, err)
	require.Len(t, sections, 1, "Should have one section after creation")

	retrievedSection := sections[0]
	assert.Equal(t, section.ID, retrievedSection.ID)
	assert.Equal(t, story.ID, retrievedSection.StoryID)
	assert.Equal(t, 1, retrievedSection.SectionNumber)
	assert.Equal(t, "This is the first section of the story.", retrievedSection.Content)
	assert.Equal(t, "A1", retrievedSection.LanguageLevel)
	assert.Equal(t, 50, retrievedSection.WordCount)
	assert.Equal(t, models.GeneratorTypeUser, retrievedSection.GeneratedBy)
	assert.NotZero(t, retrievedSection.GeneratedAt)
	assert.NotZero(t, retrievedSection.GenerationDate)

	// Create a second section
	section2, err := storyService.CreateSection(ctx, story.ID, "This is the second section of the story.", "A1", 45, models.GeneratorTypeWorker)
	require.NoError(t, err)
	require.NotNil(t, section2)
	assert.Equal(t, 2, section2.SectionNumber)

	// Get sections again and verify we have two, in correct order
	sections, err = storyService.GetStorySections(ctx, story.ID)
	require.NoError(t, err)
	require.Len(t, sections, 2, "Should have two sections after creating second")

	// Verify order (should be by section_number ASC)
	assert.Equal(t, 1, sections[0].SectionNumber)
	assert.Equal(t, 2, sections[1].SectionNumber)

	// Test with non-existent story ID
	nonExistentSections, err := storyService.GetStorySections(ctx, 999999)
	require.NoError(t, err)
	assert.Len(t, nonExistentSections, 0, "Non-existent story should return empty slice")
}

// TestStoryService_GetSection_Integration tests the GetSection functionality
func TestStoryService_GetSection_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	storyService := NewStoryService(db, cfg, logger)

	ctx := context.Background()

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(ctx, username, "italian", "A1")
	require.NoError(t, err)

	// Create a test story
	story, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:       "Test Story for GetSection",
		Subject:     stringPtr("Test Subject"),
		AuthorStyle: stringPtr("Simple"),
	})
	require.NoError(t, err)
	require.NotNil(t, story)

	// Create a test section
	section, err := storyService.CreateSection(ctx, story.ID, "This is the first section of the story.", "A1", 50, models.GeneratorTypeUser)
	require.NoError(t, err)
	require.NotNil(t, section)

	// Create some questions for the section
	questions := []models.StorySectionQuestionData{
		{
			QuestionText:       "What is the main topic?",
			Options:            []string{"Topic A", "Topic B", "Topic C", "Topic D"},
			CorrectAnswerIndex: 0,
			Explanation:        stringPtr("The main topic is clearly stated"),
		},
		{
			QuestionText:       "Where does this take place?",
			Options:            []string{"Place A", "Place B", "Place C", "Place D"},
			CorrectAnswerIndex: 1,
			Explanation:        stringPtr("The location is mentioned early"),
		},
	}
	err = storyService.CreateSectionQuestions(ctx, section.ID, questions)
	require.NoError(t, err)

	// Test GetSection with valid section and user
	sectionWithQuestions, err := storyService.GetSection(ctx, section.ID, uint(user.ID))
	require.NoError(t, err)
	require.NotNil(t, sectionWithQuestions)

	// Verify section data
	assert.Equal(t, section.ID, sectionWithQuestions.StorySection.ID)
	assert.Equal(t, story.ID, sectionWithQuestions.StorySection.StoryID)
	assert.Equal(t, 1, sectionWithQuestions.StorySection.SectionNumber)
	assert.Equal(t, "This is the first section of the story.", sectionWithQuestions.StorySection.Content)
	assert.Equal(t, "A1", sectionWithQuestions.StorySection.LanguageLevel)
	assert.Equal(t, 50, sectionWithQuestions.StorySection.WordCount)
	assert.Equal(t, models.GeneratorTypeUser, sectionWithQuestions.StorySection.GeneratedBy)

	// Verify questions
	require.Len(t, sectionWithQuestions.Questions, 2, "Should have 2 questions")
	firstQuestion := sectionWithQuestions.Questions[0]
	assert.Equal(t, "What is the main topic?", firstQuestion.QuestionText)
	assert.Equal(t, []string{"Topic A", "Topic B", "Topic C", "Topic D"}, firstQuestion.Options)
	assert.Equal(t, 0, firstQuestion.CorrectAnswerIndex)
	assert.Equal(t, "The main topic is clearly stated", *firstQuestion.Explanation)

	// Test GetSection with non-existent section ID
	_, err = storyService.GetSection(ctx, 999999, uint(user.ID))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "section not found or access denied")

	// Test GetSection with valid section but wrong user ID (access denied)
	anotherUser, err := userService.CreateUser(ctx, fmt.Sprintf("anotheruser_%d", time.Now().UnixNano()), "italian", "A1")
	require.NoError(t, err)

	_, err = storyService.GetSection(ctx, section.ID, uint(anotherUser.ID))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "section not found or access denied")
}

// TestStoryService_GetCurrentStory_Integration tests the GetCurrentStory functionality
func TestStoryService_GetCurrentStory_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	storyService := NewStoryService(db, cfg, logger)

	ctx := context.Background()

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(ctx, username, "italian", "A1")
	require.NoError(t, err)

	// Initially, user should have no current story
	currentStory, err := storyService.GetCurrentStory(ctx, uint(user.ID))
	require.NoError(t, err)
	assert.Nil(t, currentStory, "User should have no current story initially")

	// Create a test story (should be set as current automatically)
	story, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:       "Test Current Story",
		Subject:     stringPtr("Test Subject"),
		AuthorStyle: stringPtr("Simple"),
	})
	require.NoError(t, err)
	require.NotNil(t, story)
	assert.True(t, story.Status == models.StoryStatusActive, "New story should be set as current")

	// Create a section for the story
	section, err := storyService.CreateSection(ctx, story.ID, "This is the first section of the current story.", "A1", 50, models.GeneratorTypeUser)
	require.NoError(t, err)
	require.NotNil(t, section)

	// Now get current story and verify it includes the section
	currentStory, err = storyService.GetCurrentStory(ctx, uint(user.ID))
	require.NoError(t, err)
	require.NotNil(t, currentStory, "User should now have a current story")

	// Verify story data
	assert.Equal(t, story.ID, currentStory.Story.ID)
	assert.Equal(t, "Test Current Story", currentStory.Story.Title)
	assert.Equal(t, "italian", currentStory.Story.Language)
	assert.True(t, currentStory.Story.Status == models.StoryStatusActive)

	// Verify section data
	require.Len(t, currentStory.Sections, 1, "Current story should have one section")
	assert.Equal(t, section.ID, currentStory.Sections[0].ID)
	assert.Equal(t, 1, currentStory.Sections[0].SectionNumber)
	assert.Equal(t, "This is the first section of the current story.", currentStory.Sections[0].Content)
	assert.Equal(t, models.GeneratorTypeUser, currentStory.Sections[0].GeneratedBy)

	// Create another story in the same language (should become current, replacing the first)
	anotherStory, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:       "Another Story",
		Subject:     stringPtr("Another Subject"),
		AuthorStyle: stringPtr("Simple"),
	})
	require.NoError(t, err)
	require.NotNil(t, anotherStory)
	assert.True(t, anotherStory.Status == models.StoryStatusActive, "Second story should become current")
	// Note: story.Status == models.StoryStatusActive is not updated in memory, so we don't check it

	// Current story should now be the second one
	currentStory, err = storyService.GetCurrentStory(ctx, uint(user.ID))
	require.NoError(t, err)
	require.NotNil(t, currentStory)
	assert.Equal(t, anotherStory.ID, currentStory.Story.ID, "Second story should now be current")

	// Archive the current story (second story) and create a third story to test setting current
	err = storyService.ArchiveStory(ctx, anotherStory.ID, uint(user.ID))
	require.NoError(t, err)

	// Now create a third story which should become current
	thirdStory, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:       "Third Story",
		Subject:     stringPtr("Third Subject"),
		AuthorStyle: stringPtr("Simple"),
	})
	require.NoError(t, err)
	require.NotNil(t, thirdStory)
	assert.True(t, thirdStory.Status == models.StoryStatusActive, "Third story should become current")

	// Current story should now be the third one
	currentStory, err = storyService.GetCurrentStory(ctx, uint(user.ID))
	require.NoError(t, err)
	require.NotNil(t, currentStory)
	assert.Equal(t, thirdStory.ID, currentStory.Story.ID, "Third story should now be current")
	// Note: anotherStory.Status == models.StoryStatusActive is not updated in memory, so we don't check it

	// Test with user who has no stories in their preferred language
	user2, err := userService.CreateUser(ctx, fmt.Sprintf("user2_%d", time.Now().UnixNano()), "french", "A1")
	require.NoError(t, err)

	currentStory, err = storyService.GetCurrentStory(ctx, uint(user2.ID))
	require.NoError(t, err)
	assert.Nil(t, currentStory, "User with different language should have no current story")
}

// TestStoryService_GetUserStories_Integration tests the GetUserStories functionality
func TestStoryService_GetUserStories_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

	cfg, err := config.NewConfig()
	require.NoError(t, err)
	logger := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	userService := NewUserServiceWithLogger(db, cfg, logger)
	storyService := NewStoryService(db, cfg, logger)

	ctx := context.Background()

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	user, err := userService.CreateUser(ctx, username, "italian", "A1")
	require.NoError(t, err)

	// Initially, user should have no stories
	stories, err := storyService.GetUserStories(ctx, uint(user.ID), false)
	require.NoError(t, err)
	assert.Len(t, stories, 0, "User should have no stories initially")

	// Create first story
	story1, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:       "First Story",
		Subject:     stringPtr("First Subject"),
		AuthorStyle: stringPtr("Simple"),
	})
	require.NoError(t, err)
	require.NotNil(t, story1)

	// Create second story
	story2, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:       "Second Story",
		Subject:     stringPtr("Second Subject"),
		AuthorStyle: stringPtr("Simple"),
	})
	require.NoError(t, err)
	require.NotNil(t, story2)

	// Get all stories (should return both, including archived)
	allStories, err := storyService.GetUserStories(ctx, uint(user.ID), true)
	require.NoError(t, err)
	assert.Len(t, allStories, 2, "User should have 2 stories")

	// Stories should be ordered by is_current DESC, created_at DESC
	assert.Equal(t, story2.ID, allStories[0].ID, "Second story (current) should be first")
	assert.Equal(t, story1.ID, allStories[1].ID, "First story should be second")

	// Archive the first story
	err = storyService.ArchiveStory(ctx, story1.ID, uint(user.ID))
	require.NoError(t, err)

	// Get stories excluding archived (should return only second story)
	activeStories, err := storyService.GetUserStories(ctx, uint(user.ID), false)
	require.NoError(t, err)
	assert.Len(t, activeStories, 1, "Should have 1 active story after archiving first")
	assert.Equal(t, story2.ID, activeStories[0].ID, "Should return the current active story")

	// Get stories including archived (should return both)
	allStoriesWithArchived, err := storyService.GetUserStories(ctx, uint(user.ID), true)
	require.NoError(t, err)
	assert.Len(t, allStoriesWithArchived, 2, "Should have 2 stories including archived")

	// Test with non-existent user
	emptyStories, err := storyService.GetUserStories(ctx, 999999, false)
	require.NoError(t, err)
	assert.Len(t, emptyStories, 0, "Non-existent user should return empty slice")
}
