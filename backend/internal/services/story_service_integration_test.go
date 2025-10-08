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
	eligibility, err := storyService.CanGenerateSection(ctx, story.ID)
	require.NoError(t, err)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate section initially")

	// Test 2: Create a section for today (user generation)
	sectionContent := "This is a test story section."
	section, err := storyService.CreateSection(ctx, story.ID, sectionContent, "A1", 50)
	require.NoError(t, err)
	require.NotNil(t, section)

	// Test 2b: Update the generation time after first section creation (simulating what the handler does)
	err = storyService.UpdateLastGenerationTime(ctx, story.ID)
	require.NoError(t, err)

	// Test 3: Debug - check what's actually in the database
	var sectionCount int
	today := time.Now().Truncate(24 * time.Hour)
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM story_sections WHERE story_id = $1 AND generation_date = $2", story.ID, today).Scan(&sectionCount)
	require.NoError(t, err)
	t.Logf("Found %d sections for story %d on date %v", sectionCount, story.ID, today)

	// Test 3b: Should be able to generate another section today (first section doesn't count against limit)
	eligibility, err = storyService.CanGenerateSection(ctx, story.ID)
	require.NoError(t, err)
	t.Logf("CanGenerateSection returned: %v (expected true)", eligibility.CanGenerate)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate another section today")

	// Test 3c: Create a second section (user generation) - should work and increment extra_generations_today
	section2, err := storyService.CreateSection(ctx, story.ID, "This is a second section.", "A1", 40)
	require.NoError(t, err)
	require.NotNil(t, section2)
	assert.Equal(t, 2, section2.SectionNumber)

	// Test 3d: Update the generation time (simulating what the handler does)
	err = storyService.UpdateLastGenerationTime(ctx, story.ID)
	require.NoError(t, err)

	// Test 3e: Check that extra_generations_today was incremented for user generation
	var extraGenerations int
	err = db.QueryRowContext(ctx, "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 2, extraGenerations, "extra_generations_today should be 2 after second user generation")

	// Test 3f: Should not be able to generate a third section (limit reached after 2 sections)
	eligibility, err = storyService.CanGenerateSection(ctx, story.ID)
	require.NoError(t, err)
	t.Logf("CanGenerateSection returned: %v (expected false)", eligibility.CanGenerate)
	assert.False(t, eligibility.CanGenerate, "Should not be able to generate third section today")

	// Test 4: Test with a different story (should be able to generate)
	otherStory, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:   "Other Story",
		Subject: stringPtr("Other Subject"),
	})
	require.NoError(t, err)

	eligibility, err = storyService.CanGenerateSection(ctx, otherStory.ID)
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

	// Test 1: First generation today (worker generation) - should increment extra_generations_today to 1
	err = storyService.UpdateLastGenerationTime(ctx, story.ID)
	require.NoError(t, err)

	var extraGenerations int
	err = db.QueryRowContext(ctx, "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 1, extraGenerations, "Worker generation should increment extra_generations_today to 1")

	// Test 2: Second generation today (user generation) - should increment extra_generations_today to 2
	err = storyService.UpdateLastGenerationTime(ctx, story.ID)
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 2, extraGenerations, "User generation should increment extra_generations_today to 2")

	// Test 3: Third generation today (user generation) - should not increment extra_generations_today (limit reached)
	err = storyService.UpdateLastGenerationTime(ctx, story.ID)
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 2, extraGenerations, "Third generation should not increment extra_generations_today beyond 2")
}

// TestStoryService_StoryGenerationLimits_Integration tests that story generation is properly limited
func TestStoryService_StoryGenerationLimits_Integration(t *testing.T) {
	db := SharedTestDBSetup(t)
	defer db.Close()

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

	// Clean up any existing stories for the test user
	_, err = db.ExecContext(context.Background(), "DELETE FROM stories WHERE user_id = $1", user.ID)
	require.NoError(t, err)

	// Create a story
	story, err := storyService.CreateStory(context.Background(), uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:   "Test Story",
		Subject: stringPtr("Test Subject"),
	})
	require.NoError(t, err)
	require.NotNil(t, story)

	// Test 1: Initially, should be able to generate (no sections exist today)
	eligibility, err := storyService.CanGenerateSection(context.Background(), story.ID)
	require.NoError(t, err)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate section initially")

	// Test 2: Create first section (worker generation)
	section, err := storyService.CreateSection(context.Background(), story.ID, "Worker section", "A1", 50)
	require.NoError(t, err)
	require.NotNil(t, section)

	// Update generation time for worker generation
	err = storyService.UpdateLastGenerationTime(context.Background(), story.ID)
	require.NoError(t, err)

	var extraGenerations int
	err = db.QueryRowContext(context.Background(), "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 1, extraGenerations, "Worker generation should increment extra_generations_today to 1")

	// Debug: Check what values we have
	var sectionCount int
	today := time.Now().Truncate(24 * time.Hour)
	err = db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM story_sections WHERE story_id = $1 AND generation_date = $2", story.ID, today).Scan(&sectionCount)
	require.NoError(t, err)

	err = db.QueryRowContext(context.Background(), "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)

	t.Logf("After worker generation: sectionCount=%d, extraGenerationsToday=%d", sectionCount, extraGenerations)

	// Test 3: Should be able to generate user section after worker generation (worker limit not reached)
	eligibility, err = storyService.CanGenerateSection(context.Background(), story.ID)
	require.NoError(t, err)
	t.Logf("CanGenerateSection returned: %v (expected true)", eligibility.CanGenerate)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate user section after worker generation")

	// Test 4: But users should be able to generate extra sections up to the configured limit
	// Reset the story state to simulate a new day or reset the counter
	_, err = db.ExecContext(context.Background(), "UPDATE stories SET extra_generations_today = 0 WHERE id = $1", story.ID)
	require.NoError(t, err)

	// Create a section to simulate worker generation
	_, err = storyService.CreateSection(context.Background(), story.ID, "Worker section", "A1", 50)
	require.NoError(t, err)

	// Update generation time for worker generation
	err = storyService.UpdateLastGenerationTime(context.Background(), story.ID)
	require.NoError(t, err)

	// Test 5: Should be able to generate user section (extra_generations_today is 1)
	eligibility, err = storyService.CanGenerateSection(context.Background(), story.ID)
	require.NoError(t, err)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate user section after worker generation")

	// Test 6: Create user section and update generation time
	_, err = storyService.CreateSection(context.Background(), story.ID, "User section", "A1", 40)
	require.NoError(t, err)

	err = storyService.UpdateLastGenerationTime(context.Background(), story.ID)
	require.NoError(t, err)

	err = db.QueryRowContext(context.Background(), "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	expectedTotal := 2 // User generation increments to 2 (1 worker + 1 user)
	assert.Equal(t, expectedTotal, extraGenerations, "User generation should increment extra_generations_today to 2")

	// Test 7: Should not be able to generate third section (limit reached)
	eligibility, err = storyService.CanGenerateSection(context.Background(), story.ID)
	require.NoError(t, err)
	assert.False(t, eligibility.CanGenerate, "Should not be able to generate third section after user generation")

	// Test 8: Test with MaxExtraGenerationsPerDay = 1 (allow 1 extra user generation)
	cfg2, err := config.NewConfig()
	require.NoError(t, err)
	cfg2.Story.MaxArchivedPerUser = 20
	cfg2.Story.GenerationEnabled = true
	cfg2.Story.MaxExtraGenerationsPerDay = 1
	logger2 := observability.NewLogger(&config.OpenTelemetryConfig{EnableLogging: false})
	storyService2 := NewStoryService(db, cfg2, logger2)

	// Clean up the story for the new test
	_, err = db.ExecContext(context.Background(), "UPDATE stories SET extra_generations_today = 0 WHERE id = $1", story.ID)
	require.NoError(t, err)

	// Should be able to generate worker section
	eligibility, err = storyService2.CanGenerateSection(context.Background(), story.ID)
	require.NoError(t, err)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate worker section")

	// Create worker section
	_, err = storyService2.CreateSection(context.Background(), story.ID, "Worker section 2", "A1", 50)
	require.NoError(t, err)

	// Update generation time for worker generation
	err = storyService2.UpdateLastGenerationTime(context.Background(), story.ID)
	require.NoError(t, err)

	// Should be able to generate user section (extra_generations_today = 1)
	eligibility, err = storyService2.CanGenerateSection(context.Background(), story.ID)
	require.NoError(t, err)
	assert.True(t, eligibility.CanGenerate, "Should be able to generate user section after worker generation")

	// Create user section
	_, err = storyService2.CreateSection(context.Background(), story.ID, "User section 2", "A1", 40)
	require.NoError(t, err)

	// Update generation time for user generation
	err = storyService2.UpdateLastGenerationTime(context.Background(), story.ID)
	require.NoError(t, err)

	// Check that extra_generations_today was incremented to 2
	var extraGenerations2 int
	err = db.QueryRowContext(context.Background(), "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations2)
	require.NoError(t, err)
	assert.Equal(t, 2, extraGenerations2, "User generation should increment extra_generations_today to 2")

	// Should not be able to generate third section (limit reached)
	eligibility, err = storyService2.CanGenerateSection(context.Background(), story.ID)
	require.NoError(t, err)
	assert.False(t, eligibility.CanGenerate, "Should not be able to generate third section after user generation")
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
	section, err := storyService.CreateSection(ctx, story.ID, sectionContent, "A1", 50)
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
	section2, err := storyService.CreateSection(ctx, story.ID, "Second section content.", "A1", 30)
	require.NoError(t, err)
	require.NotNil(t, section2)
	assert.Equal(t, 2, section2.SectionNumber)

	// Update the generation time (simulating what the handler does)
	err = storyService.UpdateLastGenerationTime(ctx, story.ID)
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
	otherSection, err := storyService.CreateSection(ctx, otherStory.ID, "Other story section.", "A1", 25)
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
	section1, err := storyService.CreateSection(ctx, story.ID, "First section.", "A1", 20)
	require.NoError(t, err)

	// Should return the first section
	latestSection, err = storyService.GetLatestSection(ctx, story.ID)
	require.NoError(t, err)
	require.NotNil(t, latestSection)
	assert.Equal(t, section1.ID, latestSection.ID)
	assert.Equal(t, 1, latestSection.SectionNumber)

	// Test that creating a second section on the same day works (no database constraint)
	section2, err := storyService.CreateSection(ctx, story.ID, "Second section.", "A1", 25)
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
	section, err := storyService.CreateSection(ctx, story.ID, "This is a test section with some content to test questions.", "A1", 50)
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
