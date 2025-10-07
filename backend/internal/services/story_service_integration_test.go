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
	canGenerate, err := storyService.CanGenerateSection(ctx, story.ID)
	require.NoError(t, err)
	assert.True(t, canGenerate, "Should be able to generate section initially")

	// Test 2: Create a section for today (user generation)
	sectionContent := "This is a test story section."
	section, err := storyService.CreateSection(ctx, story.ID, sectionContent, "A1", 50)
	require.NoError(t, err)
	require.NotNil(t, section)

	// Test 3: Debug - check what's actually in the database
	var sectionCount int
	today := time.Now().Truncate(24 * time.Hour)
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM story_sections WHERE story_id = $1 AND generation_date = $2", story.ID, today).Scan(&sectionCount)
	require.NoError(t, err)
	t.Logf("Found %d sections for story %d on date %v", sectionCount, story.ID, today)

	// Test 3b: Should be able to generate another section today (first section doesn't count against limit)
	canGenerate, err = storyService.CanGenerateSection(ctx, story.ID)
	require.NoError(t, err)
	t.Logf("CanGenerateSection returned: %v (expected true)", canGenerate)
	assert.True(t, canGenerate, "Should be able to generate another section today")

	// Test 3c: Create a second section (user generation) - should work and increment extra_generations_today
	section2, err := storyService.CreateSection(ctx, story.ID, "This is a second section.", "A1", 40)
	require.NoError(t, err)
	require.NotNil(t, section2)
	assert.Equal(t, 2, section2.SectionNumber)

	// Test 3d: Update the generation time (simulating what the handler does)
	err = storyService.UpdateLastGenerationTime(ctx, story.ID, true) // isUserGeneration = true
	require.NoError(t, err)

	// Test 3e: Check that extra_generations_today was incremented for user generation
	var extraGenerations int
	err = db.QueryRowContext(ctx, "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 1, extraGenerations, "extra_generations_today should be 1 after user generation")

	// Test 3f: Should not be able to generate a third section (limit reached)
	canGenerate, err = storyService.CanGenerateSection(ctx, story.ID)
	require.NoError(t, err)
	t.Logf("CanGenerateSection returned: %v (expected false)", canGenerate)
	assert.False(t, canGenerate, "Should not be able to generate third section today")

	// Test 4: Test with a different story (should be able to generate)
	otherStory, err := storyService.CreateStory(ctx, uint(user.ID), "italian", &models.CreateStoryRequest{
		Title:   "Other Story",
		Subject: stringPtr("Other Subject"),
	})
	require.NoError(t, err)

	canGenerate, err = storyService.CanGenerateSection(ctx, otherStory.ID)
	require.NoError(t, err)
	assert.True(t, canGenerate, "Should be able to generate section for different story")
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
	err = storyService.UpdateLastGenerationTime(ctx, story.ID, false) // isUserGeneration = false
	require.NoError(t, err)

	var extraGenerations int
	err = db.QueryRowContext(ctx, "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 0, extraGenerations, "Worker generation should not increment extra_generations_today")

	// Test 2: Second generation today (user generation) - should increment extra_generations_today
	err = storyService.UpdateLastGenerationTime(ctx, story.ID, true) // isUserGeneration = true
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 1, extraGenerations, "User generation should increment extra_generations_today")

	// Test 3: Third generation today (user generation) - should increment extra_generations_today again
	err = storyService.UpdateLastGenerationTime(ctx, story.ID, true) // isUserGeneration = true
	require.NoError(t, err)

	err = db.QueryRowContext(ctx, "SELECT extra_generations_today FROM stories WHERE id = $1", story.ID).Scan(&extraGenerations)
	require.NoError(t, err)
	assert.Equal(t, 2, extraGenerations, "Second user generation should increment extra_generations_today to 2")
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
	err = storyService.UpdateLastGenerationTime(ctx, story.ID, true) // isUserGeneration = true
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
